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

// CursorConfig configures cursor control functions.
// The zero value is acceptable and will use sensible defaults (os.Stdout for output).
type CursorConfig struct {
	Output io.Writer // nil = os.Stdout. Specify where to write cursor commands.
}

// cursorOutput returns the output writer from the config, defaulting to os.Stdout.
func cursorOutput(cfg []CursorConfig) io.Writer {
	if len(cfg) > 0 && cfg[0].Output != nil {
		return cfg[0].Output
	}
	return os.Stdout
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
func SaveCursor(cfg ...CursorConfig) {
	fmt.Fprint(cursorOutput(cfg), "\033[s")
}

// RestoreCursor restores the cursor to the last saved position.
func RestoreCursor(cfg ...CursorConfig) {
	fmt.Fprint(cursorOutput(cfg), "\033[u")
}

// MoveCursorUp moves the cursor up n lines.
func MoveCursorUp(n int, cfg ...CursorConfig) {
	if n <= 0 {
		return
	}
	fmt.Fprintf(cursorOutput(cfg), "\033[%dA", n)
}

// MoveCursorDown moves the cursor down n lines.
func MoveCursorDown(n int, cfg ...CursorConfig) {
	if n <= 0 {
		return
	}
	fmt.Fprintf(cursorOutput(cfg), "\033[%dB", n)
}

// MoveCursorRight moves the cursor right n columns.
func MoveCursorRight(n int, cfg ...CursorConfig) {
	if n <= 0 {
		return
	}
	fmt.Fprintf(cursorOutput(cfg), "\033[%dC", n)
}

// MoveCursorLeft moves the cursor left n columns.
func MoveCursorLeft(n int, cfg ...CursorConfig) {
	if n <= 0 {
		return
	}
	fmt.Fprintf(cursorOutput(cfg), "\033[%dD", n)
}

// MoveToColumn moves the cursor to the specified column (1-based).
func MoveToColumn(col int, cfg ...CursorConfig) {
	if col < 1 {
		col = 1
	}
	fmt.Fprintf(cursorOutput(cfg), "\033[%dG", col)
}

// MoveToLineStart moves the cursor to the beginning of the current line.
func MoveToLineStart(cfg ...CursorConfig) {
	fmt.Fprint(cursorOutput(cfg), "\r")
}

// ClearLine clears the entire current line.
func ClearLine(cfg ...CursorConfig) {
	fmt.Fprint(cursorOutput(cfg), "\033[2K")
}

// ClearToEndOfLine clears from cursor to end of current line.
func ClearToEndOfLine(cfg ...CursorConfig) {
	fmt.Fprint(cursorOutput(cfg), "\033[K")
}

// ClearToStartOfLine clears from cursor to start of current line.
func ClearToStartOfLine(cfg ...CursorConfig) {
	fmt.Fprint(cursorOutput(cfg), "\033[1K")
}

// ClearLines clears n lines starting from the current cursor position,
// moving upward. Useful for clearing multi-line input areas.
// After clearing, the cursor is positioned at the start of the top-most cleared line.
func ClearLines(n int, cfg ...CursorConfig) {
	if n <= 0 {
		return
	}
	out := cursorOutput(cfg)
	for i := 0; i < n; i++ {
		fmt.Fprint(out, "\033[2K") // Clear current line
		if i < n-1 {
			fmt.Fprint(out, "\033[1A") // Move up (except on last iteration)
		}
	}
	fmt.Fprint(out, "\r") // Move to start of line
}

// ClearLinesDown clears n lines starting from current position, moving downward.
// After clearing, cursor is at the start of the last cleared line.
func ClearLinesDown(n int, cfg ...CursorConfig) {
	if n <= 0 {
		return
	}
	out := cursorOutput(cfg)
	for i := 0; i < n; i++ {
		fmt.Fprint(out, "\033[2K") // Clear current line
		if i < n-1 {
			fmt.Fprint(out, "\033[1B") // Move down (except on last iteration)
		}
	}
	fmt.Fprint(out, "\r") // Move to start of line
}

// ClearScreen clears the entire screen and moves cursor to top-left.
func ClearScreen(cfg ...CursorConfig) {
	fmt.Fprint(cursorOutput(cfg), "\033[2J\033[H")
}

// ClearToEndOfScreen clears from cursor to end of screen.
func ClearToEndOfScreen(cfg ...CursorConfig) {
	fmt.Fprint(cursorOutput(cfg), "\033[0J")
}

// ClearToStartOfScreen clears from cursor to start of screen.
func ClearToStartOfScreen(cfg ...CursorConfig) {
	fmt.Fprint(cursorOutput(cfg), "\033[1J")
}

// HideCursor hides the cursor.
func HideCursor(cfg ...CursorConfig) {
	fmt.Fprint(cursorOutput(cfg), "\033[?25l")
}

// ShowCursor shows the cursor.
func ShowCursor(cfg ...CursorConfig) {
	fmt.Fprint(cursorOutput(cfg), "\033[?25h")
}

// Newline prints a newline character.
func Newline(cfg ...CursorConfig) {
	fmt.Fprintln(cursorOutput(cfg))
}
