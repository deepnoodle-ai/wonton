//go:build darwin || linux

package tui

import (
	"bytes"
	"testing"

	"github.com/creack/pty"

	"github.com/deepnoodle-ai/wonton/assert"
)

// InlineApp reads its width and height from the terminal it writes to,
// not from standard output, and reads them again when Run starts.
func TestInlineAppSizeComesFromItsOutput(t *testing.T) {
	master, tty, err := pty.Open()
	if err != nil {
		t.Skipf("pty unavailable: %v", err)
	}
	defer master.Close()
	defer tty.Close()
	assert.NoError(t, pty.Setsize(tty, &pty.Winsize{Cols: 77, Rows: 21}))

	runner := NewInlineApp(WithInlineOutput(tty))
	assert.Equal(t, runner.config.Width, 77)
	assert.Equal(t, runner.live.maxHeight, 21)

	assert.NoError(t, pty.Setsize(tty, &pty.Winsize{Cols: 64, Rows: 18}))
	runner.refreshScreenSize()
	assert.Equal(t, runner.config.Width, 64)
	assert.Equal(t, runner.live.config.Width, 64)
	assert.Equal(t, runner.live.maxHeight, 18)

	// An explicit width stays; the height still follows the terminal.
	fixed := NewInlineApp(WithInlineOutput(tty), WithInlineWidth(50))
	assert.NoError(t, pty.Setsize(tty, &pty.Winsize{Cols: 70, Rows: 15}))
	fixed.refreshScreenSize()
	assert.Equal(t, fixed.config.Width, 50)
	assert.Equal(t, fixed.live.maxHeight, 15)

	// Output that is not a terminal keeps the defaults.
	var buf bytes.Buffer
	plain := NewInlineApp(WithInlineOutput(&buf))
	assert.Equal(t, plain.config.Width, 80)
	assert.Equal(t, plain.live.maxHeight, 0)
}
