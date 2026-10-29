package terminalinput

import "golang.org/x/sys/unix"

// FlushInput discards input the terminal has received but no one has read.
func FlushInput(fd int) error { return unix.IoctlSetPointerInt(fd, unix.TIOCFLUSH, 1) } // FREAD
