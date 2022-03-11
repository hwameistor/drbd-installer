// +build linux

package main

import (
	"flag"
	"fmt"
	"os"
	"path"
	"runtime"
	"strings"

	"github.com/hwameistor/drbd-installer/pkg/drbd"
	log "github.com/sirupsen/logrus"
)

var (
	debug                              = flag.Bool("debug", true, "debug mode, true by default")
	block                              = flag.Bool("block-the-pod", false, "block when succeccfully installed drbd ko")
	BUILDVERSION, BUILDTIME, GOVERSION string
)

func printVersion() {
	log.Info(fmt.Sprintf("GitCommit:%q, BuildDate:%q, GoVersion:%q", BUILDVERSION, BUILDTIME, GOVERSION))
}

func setupLogging(enableDebug bool) {
	if enableDebug {
		log.SetLevel(log.DebugLevel)
	}

	log.SetFormatter(&log.TextFormatter{
		DisableColors: true,
		FullTimestamp: true,
		// log with funcname, file fileds. eg: func=processNode file="node_task_worker.go:43"
		CallerPrettyfier: func(f *runtime.Frame) (string, string) {
			s := strings.Split(f.Function, ".")
			funcname := s[len(s)-1]
			filename := path.Base(f.File)
			return funcname, fmt.Sprintf("%s:%d", filename, f.Line)
		},
	})
	log.SetReportCaller(true)
}

func main() {
	flag.Parse()

	setupLogging(*debug)
	printVersion()

	DRBDKernelModInstaller, err := drbd.NewDRBDKernelModInstaller()
	if err != nil {
		os.Exit(1)
	}

	log.Info("Start find Suitable DRBD Kernel Mod")
	if !DRBDKernelModInstaller.HasSuitableDRBDKernelModBuilds() {
		log.Errorf("No Suitable DRBD Kernel Mods")
		return
	}

	log.Info("Start copy DRBD kernel mod to host")
	if err := DRBDKernelModInstaller.CopyKernelModToHost(); err != nil {
		log.WithError(err).Error("Failed to copy DRBD kernel mod to host")
		return
	}

	log.Info("Start install DRBD kernel mod on host")
	if err := DRBDKernelModInstaller.Insmod(); err != nil {
		log.WithError(err).Error("Failed to install DRBD kernel mod on host")
		return
	}

	log.Info("Start nsure DRBD kernel mod auto load when host restarted")
	if err := DRBDKernelModInstaller.EnsureAutoLoadWhenHostRestarted(); err != nil {
		log.WithError(err).Error("Failed to ensure DRBD kernel mod auto load when host restarted")
		return
	}

	if *block {
		log.Info("Blocking for debug reason")
		select {}
	}
}
