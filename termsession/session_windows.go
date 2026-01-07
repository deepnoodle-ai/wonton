//go:build windows

package termsession

import "os"

// setupResizeSignal sets up terminal resize handling (Windows).
// Windows does not support SIGWINCH, so this is a no-op.
func setupResizeSignal(ch chan os.Signal) {
	// No signal handling on Windows
}
