//go:build windows

package terminal

// setupResizeSignal sets up terminal resize handling (Windows).
// Windows does not support SIGWINCH, so this is a no-op.
// Terminal resize detection on Windows would require polling or
// using Windows-specific console APIs.
func (t *Terminal) setupResizeSignal() {
	// No signal handling on Windows
}
