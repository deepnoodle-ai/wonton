package terminalstate

import (
	"errors"
	"golang.org/x/sys/unix"
	"golang.org/x/term"
	"os"
)

func drain(file *os.File) error {
	fd := int(file.Fd())
	if !term.IsTerminal(fd) {
		return nil
	}
	for {
		err := unix.IoctlSetInt(fd, unix.TCSBRK, 1)
		if !errors.Is(err, unix.EINTR) {
			return err
		}
	}
}
