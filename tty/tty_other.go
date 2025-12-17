//go:build !linux && !darwin && !dragonfly && !freebsd && !netbsd && !openbsd && !windows && !solaris && !illumos && !aix

package tty

import "os"

// IsTerminal reports whether f appears to be a terminal.
// On unsupported platforms, this uses a heuristic based on file mode.
func IsTerminal(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}
