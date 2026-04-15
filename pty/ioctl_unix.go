//go:build darwin || linux

package pty

import (
	"syscall"
	"unsafe"
)

func ioctl(fd uintptr, req uint, arg uintptr) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, uintptr(req), arg)
	if errno != 0 {
		return errno
	}
	return nil
}

// ioctlPtr is a variant for ioctls that take a pointer argument. The
// unsafe.Pointer to uintptr conversion must happen in the same expression as
// the syscall (see unsafe.Pointer rule 4), so we call syscall.Syscall directly
// rather than delegating to ioctl.
func ioctlPtr(fd uintptr, req uint, arg unsafe.Pointer) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, uintptr(req), uintptr(arg))
	if errno != 0 {
		return errno
	}
	return nil
}
