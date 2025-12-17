//go:build windows

package tty

import (
	"os"

	"golang.org/x/sys/windows"
)

// IsTerminal reports whether f is a terminal (console).
func IsTerminal(f *os.File) bool {
	var mode uint32
	err := windows.GetConsoleMode(windows.Handle(f.Fd()), &mode)
	return err == nil
}
