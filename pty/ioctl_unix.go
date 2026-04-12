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

// ioctlPtr is a variant for ioctls that take a pointer argument, used on
// platforms where the argument must be passed indirectly.
func ioctlPtr(fd uintptr, req uint, arg unsafe.Pointer) error {
	return ioctl(fd, req, uintptr(arg))
}
