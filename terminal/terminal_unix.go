//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris || aix

package terminal

import (
	"os/signal"
	"syscall"
)

// setupResizeSignal sets up SIGWINCH signal handling for terminal resize (Unix).
func (t *Terminal) setupResizeSignal() {
	signal.Notify(t.resizeChan, syscall.SIGWINCH)
}
