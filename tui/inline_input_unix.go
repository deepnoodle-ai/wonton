//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris || aix

package tui

import (
	"os"
	"syscall"
)

// promptResizeSignals returns the signals that tell Prompt the terminal was
// resized (Unix: SIGWINCH).
func promptResizeSignals() []os.Signal {
	return []os.Signal{syscall.SIGWINCH}
}
