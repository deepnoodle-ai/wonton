package tui

import "golang.org/x/sys/unix"

func handoffTestAttributes(fd int) (*unix.Termios, error) {
	attrs, err := unix.IoctlGetTermios(fd, unix.TIOCGETA)
	if err == nil {
		// The kernel sets PENDIN on raw-to-canonical transitions until its next read.
		// It describes queued-input reprocessing, not a changed terminal mode.
		attrs.Lflag &^= unix.PENDIN
	}
	return attrs, err
}
