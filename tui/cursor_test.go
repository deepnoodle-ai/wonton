package tui

import (
	"bytes"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestCursorFunctions(t *testing.T) {
	// Helper to capture output
	capture := func(f func(...CursorConfig)) string {
		var buf bytes.Buffer
		cfg := CursorConfig{Output: &buf}
		f(cfg)
		return buf.String()
	}

	captureInt := func(n int, f func(int, ...CursorConfig)) string {
		var buf bytes.Buffer
		cfg := CursorConfig{Output: &buf}
		f(n, cfg)
		return buf.String()
	}

	t.Run("SaveCursor", func(t *testing.T) {
		got := capture(SaveCursor)
		assert.Equal(t, "\033[s", got)
	})

	t.Run("RestoreCursor", func(t *testing.T) {
		got := capture(RestoreCursor)
		assert.Equal(t, "\033[u", got)
	})

	t.Run("MoveCursorUp", func(t *testing.T) {
		got := captureInt(5, MoveCursorUp)
		assert.Equal(t, "\033[5A", got)

		got = captureInt(0, MoveCursorUp)
		assert.Equal(t, "", got)
	})

	t.Run("MoveCursorDown", func(t *testing.T) {
		got := captureInt(3, MoveCursorDown)
		assert.Equal(t, "\033[3B", got)
	})

	t.Run("MoveCursorRight", func(t *testing.T) {
		got := captureInt(2, MoveCursorRight)
		assert.Equal(t, "\033[2C", got)
	})

	t.Run("MoveCursorLeft", func(t *testing.T) {
		got := captureInt(4, MoveCursorLeft)
		assert.Equal(t, "\033[4D", got)
	})

	t.Run("MoveToColumn", func(t *testing.T) {
		got := captureInt(10, MoveToColumn)
		assert.Equal(t, "\033[10G", got)

		got = captureInt(0, MoveToColumn) // Should default to 1
		assert.Equal(t, "\033[1G", got)
	})

	t.Run("MoveToLineStart", func(t *testing.T) {
		got := capture(MoveToLineStart)
		assert.Equal(t, "\r", got)
	})

	t.Run("ClearLine", func(t *testing.T) {
		got := capture(ClearLine)
		assert.Equal(t, "\033[2K", got)
	})

	t.Run("ClearToEndOfLine", func(t *testing.T) {
		got := capture(ClearToEndOfLine)
		assert.Equal(t, "\033[K", got)
	})

	t.Run("ClearToStartOfLine", func(t *testing.T) {
		got := capture(ClearToStartOfLine)
		assert.Equal(t, "\033[1K", got)
	})

	t.Run("ClearLines", func(t *testing.T) {
		got := captureInt(2, ClearLines)
		// Clear current (2K), Up (1A), Clear current (2K), CR (\r)
		assert.Equal(t, "\033[2K\033[1A\033[2K\r", got)

		got = captureInt(1, ClearLines)
		assert.Equal(t, "\033[2K\r", got)
	})

	t.Run("ClearLinesDown", func(t *testing.T) {
		got := captureInt(2, ClearLinesDown)
		// Clear current (2K), Down (1B), Clear current (2K), CR (\r)
		assert.Equal(t, "\033[2K\033[1B\033[2K\r", got)
	})

	t.Run("ClearScreen", func(t *testing.T) {
		got := capture(ClearScreen)
		assert.Equal(t, "\033[2J\033[H", got)
	})

	t.Run("ClearToEndOfScreen", func(t *testing.T) {
		got := capture(ClearToEndOfScreen)
		assert.Equal(t, "\033[0J", got)
	})

	t.Run("ClearToStartOfScreen", func(t *testing.T) {
		got := capture(ClearToStartOfScreen)
		assert.Equal(t, "\033[1J", got)
	})

	t.Run("HideShowCursor", func(t *testing.T) {
		got := capture(HideCursor)
		assert.Equal(t, "\033[?25l", got)

		got = capture(ShowCursor)
		assert.Equal(t, "\033[?25h", got)
	})

	t.Run("Newline", func(t *testing.T) {
		got := capture(Newline)
		assert.Equal(t, "\n", got)
	})
}
