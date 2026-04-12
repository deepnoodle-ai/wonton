package pty

import (
	"fmt"
	"os"
	"unsafe"

	"golang.org/x/sys/unix"
)

// ptmget is the result struct from the PTMGET ioctl on OpenBSD.
type ptmget struct {
	Cfd int32
	Sfd int32
	Cn  [16]byte
	Sn  [16]byte
}

const ioctlPTMGET = 0x40287401 // PTMGET

func open() (*os.File, *os.File, error) {
	ptm, err := os.OpenFile("/dev/ptm", os.O_RDWR|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, nil, fmt.Errorf("pty: open /dev/ptm: %w", err)
	}
	defer ptm.Close()

	var pm ptmget
	if err := ioctl(ptm.Fd(), ioctlPTMGET, uintptr(unsafe.Pointer(&pm))); err != nil {
		return nil, nil, fmt.Errorf("pty: PTMGET: %w", err)
	}

	master := os.NewFile(uintptr(pm.Cfd), "ptmx")
	slave := os.NewFile(uintptr(pm.Sfd), "pts")

	return master, slave, nil
}
