package pty

import (
	"fmt"
	"os"
	"unsafe"

	"golang.org/x/sys/unix"
)

// fiodgnameArg is the argument struct for the FIODGNAME ioctl on FreeBSD.
type fiodgnameArg struct {
	Len int32
	Buf *byte
}

func open() (*os.File, *os.File, error) {
	// Use posix_openpt to get a master fd.
	fd, err := unix.PosixOpenpt(os.O_RDWR | unix.O_CLOEXEC)
	if err != nil {
		return nil, nil, fmt.Errorf("pty: posix_openpt: %w", err)
	}
	master := os.NewFile(uintptr(fd), "ptmx")

	// Grant and unlock the slave.
	if err := unix.Grantpt(fd); err != nil {
		master.Close()
		return nil, nil, fmt.Errorf("pty: grantpt: %w", err)
	}
	if err := unix.Unlockpt(fd); err != nil {
		master.Close()
		return nil, nil, fmt.Errorf("pty: unlockpt: %w", err)
	}

	// Get the slave device name via FIODGNAME.
	var buf [128]byte
	arg := fiodgnameArg{
		Len: int32(len(buf)),
		Buf: &buf[0],
	}
	if err := ioctl(master.Fd(), unix.FIODGNAME, uintptr(unsafe.Pointer(&arg))); err != nil {
		master.Close()
		return nil, nil, fmt.Errorf("pty: FIODGNAME: %w", err)
	}

	slaveName := "/dev/"
	for i, b := range buf {
		if b == 0 {
			slaveName += string(buf[:i])
			break
		}
	}

	slave, err := os.OpenFile(slaveName, os.O_RDWR, 0)
	if err != nil {
		master.Close()
		return nil, nil, fmt.Errorf("pty: open slave %s: %w", slaveName, err)
	}

	return master, slave, nil
}
