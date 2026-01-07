//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris || aix

package tui

import (
	"os"
	"os/signal"
	"syscall"
)

// setupResizeWatcher initializes terminal resize signal handling (Unix).
func (r *InlineApp) setupResizeWatcher() {
	r.resizeChan = make(chan os.Signal, 1)
	signal.Notify(r.resizeChan, syscall.SIGWINCH)
}

// cleanupResizeWatcher stops resize signal handling (Unix).
func (r *InlineApp) cleanupResizeWatcher() {
	signal.Stop(r.resizeChan)
	close(r.resizeChan)
}
