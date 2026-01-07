//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris || aix

package termsession

import (
	"os"
	"os/signal"
	"syscall"
)

// setupResizeSignal sets up SIGWINCH signal handling (Unix).
func setupResizeSignal(ch chan os.Signal) {
	signal.Notify(ch, syscall.SIGWINCH)
}
