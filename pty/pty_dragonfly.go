package pty

import (
	"fmt"
	"os"
	"strconv"
	"unsafe"

	"golang.org/x/sys/unix"
)

func open() (*os.File, *os.File, error) {
	// DragonFly uses /dev/ptmx like Linux.
	master, err := os.OpenFile("/dev/ptmx", os.O_RDWR|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, nil, fmt.Errorf("pty: open /dev/ptmx: %w", err)
	}

	// Get the PTS number.
	var n uint32
	if err := ioctl(master.Fd(), unix.TIOCGPTN, uintptr(unsafe.Pointer(&n))); err != nil {
		master.Close()
		return nil, nil, fmt.Errorf("pty: TIOCGPTN: %w", err)
	}

	// Unlock the slave.
	var unlock uint32
	if err := ioctl(master.Fd(), unix.TIOCSPTLCK, uintptr(unsafe.Pointer(&unlock))); err != nil {
		master.Close()
		return nil, nil, fmt.Errorf("pty: TIOCSPTLCK: %w", err)
	}

	slaveName := "/dev/pts/" + strconv.FormatUint(uint64(n), 10)
	slave, err := os.OpenFile(slaveName, os.O_RDWR|unix.O_NOCTTY, 0)
	if err != nil {
		master.Close()
		return nil, nil, fmt.Errorf("pty: open slave %s: %w", slaveName, err)
	}

	return master, slave, nil
}
