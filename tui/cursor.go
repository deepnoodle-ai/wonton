package tui

import (
	"fmt"
	"io"
	"os"
)

// Cursor and terminal control functions for inline terminal mode.
// These utilities complement Print() and LivePrinter by providing
// low-level cursor manipulation for building interactive CLI experiences
// that don't take over the entire screen.

// CursorOption configures cursor control functions.
type CursorOption func(*cursorConfig)

type cursorConfig struct {
	output io.Writer
}

func defaultCursorConfig() cursorConfig {
	return cursorConfig{
		output: os.Stdout,
	}
}

// WithCursorOutput sets the output writer for cursor control functions.
func WithCursorOutput(w io.Writer) CursorOption {
	return func(c *cursorConfig) {
		c.output = w
	}
}

// SaveCursor saves the current cursor position.
// Use RestoreCursor to return to this position later.
//
// Example:
//
//	tui.SaveCursor()
//	fmt.Println("This text appears below")
//	tui.RestoreCursor()
//	fmt.Print("Cursor is back!")
func SaveCursor(opts ...CursorOption) {
	cfg := defaultCursorConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	fmt.Fprint(cfg.output, "\033[s")
}

// RestoreCursor restores the cursor to the last saved position.
func RestoreCursor(opts ...CursorOption) {
	cfg := defaultCursorConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	fmt.Fprint(cfg.output, "\033[u")
}

// MoveCursorUp moves the cursor up n lines.
func MoveCursorUp(n int, opts ...CursorOption) {
	if n <= 0 {
		return
	}
	cfg := defaultCursorConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	fmt.Fprintf(cfg.output, "\033[%dA", n)
}

// MoveCursorDown moves the cursor down n lines.
func MoveCursorDown(n int, opts ...CursorOption) {
	if n <= 0 {
		return
	}
	cfg := defaultCursorConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	fmt.Fprintf(cfg.output, "\033[%dB", n)
}

// MoveCursorRight moves the cursor right n columns.
func MoveCursorRight(n int, opts ...CursorOption) {
	if n <= 0 {
		return
	}
	cfg := defaultCursorConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	fmt.Fprintf(cfg.output, "\033[%dC", n)
}

// MoveCursorLeft moves the cursor left n columns.
func MoveCursorLeft(n int, opts ...CursorOption) {
	if n <= 0 {
		return
	}
	cfg := defaultCursorConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	fmt.Fprintf(cfg.output, "\033[%dD", n)
}

// MoveToColumn moves the cursor to the specified column (1-based).
func MoveToColumn(col int, opts ...CursorOption) {
	if col < 1 {
		col = 1
	}
	cfg := defaultCursorConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	fmt.Fprintf(cfg.output, "\033[%dG", col)
}

// MoveToLineStart moves the cursor to the beginning of the current line.
func MoveToLineStart(opts ...CursorOption) {
	cfg := defaultCursorConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	fmt.Fprint(cfg.output, "\r")
}

// ClearLine clears the entire current line.
func ClearLine(opts ...CursorOption) {
	cfg := defaultCursorConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	fmt.Fprint(cfg.output, "\033[2K")
}

// ClearToEndOfLine clears from cursor to end of current line.
func ClearToEndOfLine(opts ...CursorOption) {
	cfg := defaultCursorConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	fmt.Fprint(cfg.output, "\033[K")
}

// ClearToStartOfLine clears from cursor to start of current line.
func ClearToStartOfLine(opts ...CursorOption) {
	cfg := defaultCursorConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	fmt.Fprint(cfg.output, "\033[1K")
}

// ClearLines clears n lines starting from the current cursor position,
// moving upward. Useful for clearing multi-line input areas.
// After clearing, the cursor is positioned at the start of the top-most cleared line.
func ClearLines(n int, opts ...CursorOption) {
	if n <= 0 {
		return
	}
	cfg := defaultCursorConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	for i := 0; i < n; i++ {
		fmt.Fprint(cfg.output, "\033[2K") // Clear current line
		if i < n-1 {
			fmt.Fprint(cfg.output, "\033[1A") // Move up (except on last iteration)
		}
	}
	fmt.Fprint(cfg.output, "\r") // Move to start of line
}

// ClearLinesDown clears n lines starting from current position, moving downward.
// After clearing, cursor is at the start of the last cleared line.
func ClearLinesDown(n int, opts ...CursorOption) {
	if n <= 0 {
		return
	}
	cfg := defaultCursorConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	for i := 0; i < n; i++ {
		fmt.Fprint(cfg.output, "\033[2K") // Clear current line
		if i < n-1 {
			fmt.Fprint(cfg.output, "\033[1B") // Move down (except on last iteration)
		}
	}
	fmt.Fprint(cfg.output, "\r") // Move to start of line
}

// ClearScreen clears the entire screen and moves cursor to top-left.
func ClearScreen(opts ...CursorOption) {
	cfg := defaultCursorConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	fmt.Fprint(cfg.output, "\033[2J\033[H")
}

// ClearToEndOfScreen clears from cursor to end of screen.
func ClearToEndOfScreen(opts ...CursorOption) {
	cfg := defaultCursorConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	fmt.Fprint(cfg.output, "\033[0J")
}

// ClearToStartOfScreen clears from cursor to start of screen.
func ClearToStartOfScreen(opts ...CursorOption) {
	cfg := defaultCursorConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	fmt.Fprint(cfg.output, "\033[1J")
}

// HideCursor hides the cursor.
func HideCursor(opts ...CursorOption) {
	cfg := defaultCursorConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	fmt.Fprint(cfg.output, "\033[?25l")
}

// ShowCursor shows the cursor.
func ShowCursor(opts ...CursorOption) {
	cfg := defaultCursorConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	fmt.Fprint(cfg.output, "\033[?25h")
}

// Newline prints a newline character.
func Newline(opts ...CursorOption) {
	cfg := defaultCursorConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	fmt.Fprintln(cfg.output)
}
