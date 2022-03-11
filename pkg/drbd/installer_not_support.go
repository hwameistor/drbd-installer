// +build !linux

package drbd

import (
	"fmt"
)

type DRBDKernelModInstaller struct {
}

func NewDRBDKernelModInstaller() (*DRBDKernelModInstaller, error) {
	return nil, nil
}

func (i *DRBDKernelModInstaller) HasSuitableDRBDKernelModBuilds() bool {
	return false
}

func (i *DRBDKernelModInstaller) CopyKernelModToHost() error {
	return fmt.Errorf("NOT SUPPORT")
}

func (i *DRBDKernelModInstaller) Insmod() error {
	return fmt.Errorf("NOT SUPPORT")
}

func (i *DRBDKernelModInstaller) EnsureAutoLoadWhenHostRestarted() error {
	return nil
}

func (i *DRBDKernelModInstaller) parseKernelVersionAndRelease() error {
	return fmt.Errorf("NOT SUPPORT")
}
