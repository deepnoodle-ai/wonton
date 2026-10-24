package tui

import "golang.org/x/sys/unix"

// flushInput discards input the terminal has received but no one has read.
func flushInput(fd int) error { return unix.IoctlSetInt(fd, unix.TCFLSH, unix.TCIFLUSH) }
