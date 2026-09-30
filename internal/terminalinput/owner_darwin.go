package terminalinput

import "golang.org/x/sys/unix"

// FIONREAD is _IOR('f', 127, int) in Darwin's sys/filio.h.
func availableBytes(fd int) (int, error) { return unix.IoctlGetInt(fd, 0x4004667f) }
func discardInput(fd int) error          { return unix.IoctlSetPointerInt(fd, unix.TIOCFLUSH, 1) } // FREAD
