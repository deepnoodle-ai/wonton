package tui

import (
	"fmt"
	"io"
	"os"
	"time"
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

// CursorPosition represents a cursor position on screen (0-based).
type CursorPosition struct {
	Row int
	Col int
}

// QueryCursorPosition queries the terminal for the current cursor position.
// It sends the Device Status Report (DSR) escape sequence and parses the response.
//
// This function requires the terminal to be in raw mode to read the response.
// The fd parameter should be the file descriptor for stdin (typically os.Stdin.Fd()).
//
// The returned position is 0-based (row 0 is the first row).
// Returns an error if the query times out or the response cannot be parsed.
func QueryCursorPosition(fd int) (CursorPosition, error) {
	return QueryCursorPositionWithTimeout(fd, 100*time.Millisecond)
}

// QueryCursorPositionWithTimeout is like QueryCursorPosition but with a custom timeout.
func QueryCursorPositionWithTimeout(fd int, timeout time.Duration) (CursorPosition, error) {
	// Send cursor position query: ESC[6n (Device Status Report)
	_, err := os.Stdout.WriteString("\033[6n")
	if err != nil {
		return CursorPosition{}, fmt.Errorf("failed to send cursor query: %w", err)
	}

	// Read response: ESC[<row>;<col>R
	// We need to read byte by byte until we get 'R'
	response := make([]byte, 0, 16)
	deadline := time.Now().Add(timeout)

	// Use os.NewFile to get a file handle from the fd
	stdin := os.NewFile(uintptr(fd), "/dev/stdin")

	// Set read deadline if possible
	stdin.SetReadDeadline(deadline)
	defer stdin.SetReadDeadline(time.Time{})

	buf := make([]byte, 1)
	inEscape := false
	for time.Now().Before(deadline) {
		n, err := stdin.Read(buf)
		if err != nil {
			if os.IsTimeout(err) {
				return CursorPosition{}, fmt.Errorf("cursor query timed out")
			}
			return CursorPosition{}, fmt.Errorf("failed to read cursor response: %w", err)
		}
		if n == 0 {
			continue
		}

		b := buf[0]
		if b == '\033' {
			inEscape = true
			response = response[:0]
			continue
		}
		if inEscape {
			response = append(response, b)
			if b == 'R' {
				break
			}
		}
	}

	// Parse response: [<row>;<col>R
	if len(response) < 4 || response[0] != '[' || response[len(response)-1] != 'R' {
		return CursorPosition{}, fmt.Errorf("invalid cursor response: %q", response)
	}

	// Extract row and col from [<row>;<col>R
	var row, col int
	_, err = fmt.Sscanf(string(response), "[%d;%d", &row, &col)
	if err != nil {
		return CursorPosition{}, fmt.Errorf("failed to parse cursor response %q: %w", response, err)
	}

	// Convert from 1-based to 0-based
	return CursorPosition{Row: row - 1, Col: col - 1}, nil
}

// MoveCursorTo moves the cursor to an absolute position (0-based row and column).
func MoveCursorTo(row, col int, cfg ...CursorConfig) {
	// Convert to 1-based for ANSI escape sequence
	fmt.Fprintf(cursorOutput(cfg), "\033[%d;%dH", row+1, col+1)
}

// SetScrollRegion sets the scrolling region to the specified rows (0-based, inclusive).
// Content outside this region will not scroll when new lines are added.
// Use ResetScrollRegion to restore normal scrolling behavior.
func SetScrollRegion(top, bottom int, cfg ...CursorConfig) {
	// Convert to 1-based for ANSI escape sequence
	fmt.Fprintf(cursorOutput(cfg), "\033[%d;%dr", top+1, bottom+1)
}

// ResetScrollRegion resets the scroll region to the full screen.
func ResetScrollRegion(cfg ...CursorConfig) {
	fmt.Fprint(cursorOutput(cfg), "\033[r")
}

// ScrollUp scrolls the content in the current scroll region up by n lines.
// New blank lines appear at the bottom of the region.
func ScrollUp(n int, cfg ...CursorConfig) {
	if n <= 0 {
		return
	}
	fmt.Fprintf(cursorOutput(cfg), "\033[%dS", n)
}

// ScrollDown scrolls the content in the current scroll region down by n lines.
// New blank lines appear at the top of the region.
func ScrollDown(n int, cfg ...CursorConfig) {
	if n <= 0 {
		return
	}
	fmt.Fprintf(cursorOutput(cfg), "\033[%dT", n)
}
