// +build linux

package drbd

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"runtime"
	"strings"
	"syscall"

	"github.com/hwameistor/drbd-installer/pkg/exechelper"
	"github.com/hwameistor/drbd-installer/pkg/exechelper/nsexecutor"
	log "github.com/sirupsen/logrus"
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

	installer.KernelModToHostPath = strings.ToLower(fmt.Sprintf("/lib/modules/%s/extra/drbd90", installer.KernelVersionReleaseOriginString))
	installer.KernelModSourcePath = strings.ToLower(fmt.Sprintf("/kernel-mods/drbd/%s/%s/%s/%s", installer.OS, installer.KernelVersion, installer.KernelRelease, installer.Arch))

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

func (i *DRBDKernelModInstaller) Insmod() error {
	files, err := ioutil.ReadDir(i.KernelModToHostPath)
	if err != nil {
		return err
	}

	for _, file := range files {
		modPath := fmt.Sprintf("%s/%s", i.KernelModToHostPath, file.Name())
		cmd := exechelper.ExecParams{
			CmdName: "insmod",
			CmdArgs: []string{modPath},
		}

		exec := nsexecutor.New()
		execRst := exec.RunCommand(cmd)
		if execRst.ExitCode != 0 {
			errRespStr := string(execRst.ErrBuf.Bytes())
			if strings.Contains(errRespStr, "File exists") {
				log.Infof("%s has already being installed on host by manually, skiping...", file.Name())
				continue
			}
			return fmt.Errorf("%w(%s)", execRst.Error, execRst.ErrBuf.Bytes())
		}
		log.Infof("%s has being successfully installed on host", file.Name())
	}
	return nil
}

func (i *DRBDKernelModInstaller) EnsureAutoLoadWhenHostRestarted() error {
	return nil
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
