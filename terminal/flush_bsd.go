//go:build dragonfly || freebsd || netbsd || openbsd

package terminal

import "golang.org/x/sys/unix"

// flushInput discards input the terminal has received but no one has read.
func flushInput(fd int) error { return unix.IoctlSetPointerInt(fd, unix.TIOCFLUSH, 1) } // FREAD
