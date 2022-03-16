// +build linux

package drbd

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"github.com/hwameistor/drbd-installer/pkg/exechelper"
	"github.com/hwameistor/drbd-installer/pkg/exechelper/nsexecutor"
	log "github.com/sirupsen/logrus"
)

const (
	DRBDAutoloaderFile = "/etc/sysconfig/modules/drbd.modules"
	DRBDAutoloader     = `#!/bin/sh
/sbin/modinfo drbd > /dev/null 2>&1
if [ $? -eq 0 ]; then
    /sbin/modprobe drbd
fi

/sbin/modinfo drbd_transport_tcp > /dev/null 2>&1
if [ $? -eq 0 ]; then
    /sbin/modprobe drbd_transport_tcp
fi`
	LibModulesPathTemplate      = "/lib/modules/%s/extra/drbd90"
	DRBDPathInContainerTemplate = "/kernel-mods/drbd/%s/%s/%s/%s"
	DepmodCMD                   = "depmod"
	ModprobeCMD                 = "modprobe"
)

type DRBDKernelModInstaller struct {
	OS,
	Arch,
	KernelVersion,
	KernelRelease,
	KernelVersionReleaseOriginString,
	KernelModToHostPath,
	KernelModSourcePath string
}

func NewDRBDKernelModInstaller() (*DRBDKernelModInstaller, error) {
	installer := &DRBDKernelModInstaller{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
	}

	if err := installer.parseKernelVersionAndRelease(); err != nil {
		return nil, err
	}

	installer.KernelModToHostPath = strings.ToLower(fmt.Sprintf(LibModulesPathTemplate, installer.KernelVersionReleaseOriginString))
	installer.KernelModSourcePath = strings.ToLower(fmt.Sprintf(DRBDPathInContainerTemplate, installer.OS, installer.KernelVersion, installer.KernelRelease, installer.Arch))

	log.Infof("host OS: %s", installer.OS)
	log.Infof("host CPU arch: %s", installer.Arch)
	log.Infof("host kernel: %s", installer.KernelVersionReleaseOriginString)
	log.Infof("host kernel version: %s", installer.KernelVersion)
	log.Infof("host kernel release: %s", installer.KernelRelease)
	log.Infof("host kernel mods Host Path: %s", installer.KernelModToHostPath)

	return installer, nil
}

func (i *DRBDKernelModInstaller) HasSuitableDRBDKernelModBuilds() bool {
	if _, err := os.Stat(i.KernelModSourcePath); os.IsNotExist(err) {
		return false
	} else if err != nil {
		return false
	}

	return true
}

func (i *DRBDKernelModInstaller) CopyKernelModToHost() error {
	if err := os.MkdirAll(i.KernelModToHostPath, os.ModePerm); err != nil {
		return err
	}

	files, err := ioutil.ReadDir(i.KernelModSourcePath)
	if err != nil {
		return err
	}

	for _, file := range files {
		src := fmt.Sprintf("%s/%s", i.KernelModSourcePath, file.Name())
		dst := fmt.Sprintf("%s/%s", i.KernelModToHostPath, file.Name())

		source, err := os.Open(src)
		if err != nil {
			return err
		}
		defer source.Close()

		destination, err := os.Create(dst)
		if err != nil {
			return err
		}
		defer destination.Close()
		_, err = io.Copy(destination, source)
		if err != nil {
			return err
		}
	}
	return nil
}

func (i *DRBDKernelModInstaller) Depmod() error {
	cmd := exechelper.ExecParams{
		CmdName: DepmodCMD,
		Timeout: 300,
	}

	exec := nsexecutor.New()
	execRst := exec.RunCommand(cmd)
	if execRst.ExitCode != 0 {
		return fmt.Errorf("%w(%s)", execRst.Error, execRst.ErrBuf.Bytes())
	}
	return nil
}

func (i *DRBDKernelModInstaller) Modprobe() error {
	files, err := ioutil.ReadDir(i.KernelModToHostPath)
	if err != nil {
		return err
	}

	for _, file := range files {
		modName := strings.TrimSuffix(file.Name(), filepath.Ext(file.Name()))
		cmd := exechelper.ExecParams{
			CmdName: ModprobeCMD,
			CmdArgs: []string{modName},
		}

		exec := nsexecutor.New()
		execRst := exec.RunCommand(cmd)
		if execRst.ExitCode != 0 {
			return fmt.Errorf("%w(%s)", execRst.Error, execRst.ErrBuf.Bytes())
		}
		log.Infof("%s has being successfully installed on host", modName)
	}
	return nil
}

func (i *DRBDKernelModInstaller) parseKernelVersionAndRelease() error {
	var uname syscall.Utsname
	if err := syscall.Uname(&uname); err != nil {
		return err
	}

	versionReleaseStr := int8ToStr(uname.Release[:])
	splitedVersionReleaseStr := strings.Split(versionReleaseStr, "-")
	if len(splitedVersionReleaseStr) < 2 {
		return fmt.Errorf("failed to parse kernel version and release. origin string is %q", splitedVersionReleaseStr)
	}

	version := splitedVersionReleaseStr[0]
	splitedReleaseStr := strings.Split(splitedVersionReleaseStr[1], ".")

	i.KernelVersion = version
	i.KernelRelease = splitedReleaseStr[0]
	i.KernelVersionReleaseOriginString = versionReleaseStr

	return nil
}

func (i *DRBDKernelModInstaller) EnsureAutoLoadWhenHostRestarted() error {
	exists, err := isFileExists(DRBDAutoloaderFile)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	autoloader, err := os.Create(DRBDAutoloaderFile)
	if err != nil {
		return err
	}
	defer autoloader.Close()

	if _, err := autoloader.Write([]byte(DRBDAutoloader)); err != nil {
		return err
	}
	return nil
}

func isFileExists(path string) (bool, error) {
	file, err := os.Open(path)
	if err != nil && os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	defer file.Close()

	return true, nil
}

func int8ToStr(arr []int8) string {
	b := make([]byte, 0, len(arr))
	for _, v := range arr {
		if v == 0x00 {
			break
		}
		b = append(b, byte(v))
	}
	return string(b)
}
