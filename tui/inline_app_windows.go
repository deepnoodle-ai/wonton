//go:build windows

package tui

import (
	"os"
)

// setupResizeWatcher initializes terminal resize signal handling (Windows).
// Windows does not support SIGWINCH, so this is a no-op.
// Terminal resize detection on Windows would require polling or
// using Windows-specific console APIs.
func (r *InlineApp) setupResizeWatcher() {
	// Create the channel but don't set up any signal handling
	r.resizeChan = make(chan os.Signal, 1)
}

// cleanupResizeWatcher stops resize signal handling (Windows).
func (r *InlineApp) cleanupResizeWatcher() {
	close(r.resizeChan)
}
