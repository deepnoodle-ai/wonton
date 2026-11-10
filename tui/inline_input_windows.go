//go:build windows

package tui

import "os"

// promptResizeSignals returns the signals that tell Prompt the terminal was
// resized. Windows has no resize signal, so Prompt keeps the width it
// measured when it started, as InlineApp does.
func promptResizeSignals() []os.Signal {
	return nil
}
