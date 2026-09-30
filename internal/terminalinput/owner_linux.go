package terminalinput

import "golang.org/x/sys/unix"

func availableBytes(fd int) (int, error) { return unix.IoctlGetInt(fd, unix.TIOCINQ) }
func discardInput(fd int) error          { return unix.IoctlSetInt(fd, unix.TCFLSH, unix.TCIFLUSH) }
