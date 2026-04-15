package pty

import (
	"fmt"
	"os"
	"unsafe"

	"golang.org/x/sys/unix"
)

func open() (*os.File, *os.File, error) {
	// Open /dev/ptmx to get a master fd.
	master, err := os.OpenFile("/dev/ptmx", os.O_RDWR|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, nil, fmt.Errorf("pty: open /dev/ptmx: %w", err)
	}

	// Get the slave device name via TIOCPTYGNAME.
	var buf [128]byte
	if err := ioctlPtr(master.Fd(), unix.TIOCPTYGNAME, unsafe.Pointer(&buf[0])); err != nil {
		master.Close()
		return nil, nil, fmt.Errorf("pty: TIOCPTYGNAME: %w", err)
	}

	// Grant and unlock the slave.
	if err := ioctl(master.Fd(), unix.TIOCPTYGRANT, 0); err != nil {
		master.Close()
		return nil, nil, fmt.Errorf("pty: grantpt: %w", err)
	}
	if err := ioctl(master.Fd(), unix.TIOCPTYUNLK, 0); err != nil {
		master.Close()
		return nil, nil, fmt.Errorf("pty: unlockpt: %w", err)
	}

	// Determine the slave name from the null-terminated buffer.
	slaveName := ""
	for i, b := range buf {
		if b == 0 {
			slaveName = string(buf[:i])
			break
		}
	}

	slave, err := os.OpenFile(slaveName, os.O_RDWR|unix.O_NOCTTY, 0)
	if err != nil {
		master.Close()
		return nil, nil, fmt.Errorf("pty: open slave %s: %w", slaveName, err)
	}

	return master, slave, nil
}
