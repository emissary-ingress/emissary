package pf

import (
	"os"
	"syscall"
	"unsafe"
)

// #include <sys/ioctl.h>
// #include <net/if.h>
// #include <net/pfvar.h>
import "C"

// ioctlDev to the pf kernel module using ioctl
type ioctlDev os.File

func newIoctlDev(path string) (*ioctlDev, error) {
	dev, err := os.OpenFile(path, os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}
	return (*ioctlDev)(dev), nil
}

// ioctl helper for pf dev
func (dev *ioctlDev) ioctl(cmd uintptr, ptr unsafe.Pointer) error {
	_, _, e := syscall.Syscall(syscall.SYS_IOCTL, (*os.File)(dev).Fd(), cmd, uintptr(ptr))
	if e != 0 {
		return e
	}
	return nil
}

// Close pf ioctl dev
func (dev *ioctlDev) Close() error {
	return (*os.File)(dev).Close()
}
