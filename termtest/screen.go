// Package termtest provides terminal output testing with ANSI parsing and snapshot comparison.
//
// The package simulates a virtual terminal screen, interpreting ANSI escape sequences
// to track cursor position, text styling, and screen content. It's designed for testing
// terminal UI applications, CLI tools, and anything that produces terminal output.
//
// # Core Components
//
// Screen: Virtual terminal buffer that processes ANSI sequences and tracks content.
// Capture/Recorder: Writers that capture output and convert it to Screen instances.
// Assertions: Test helpers for verifying screen content and cursor state.
// Snapshots: Golden file comparison for regression testing.
//
// # Basic Usage
//
// Create a screen and write terminal output to it:
//
//	screen := termtest.NewScreen(80, 24)
//	screen.Write([]byte("\x1b[1mBold\x1b[0m text"))
//	fmt.Println(screen.Text())  // "Bold text"
//
// # Testing with Assertions
//
// Use the provided assertion helpers in tests:
//
//	func TestTerminalOutput(t *testing.T) {
//	    screen := termtest.NewScreen(80, 24)
//	    app.RenderTo(screen)  // Your app writes to the screen
//
//	    termtest.AssertContains(t, screen, "Expected text")
//	    termtest.AssertRow(t, screen, 0, "First line")
//	    termtest.AssertCursor(t, screen, 10, 5)
//	}
//
// # Snapshot Testing
//
// Compare screen output to golden files for regression testing:
//
//	func TestUISnapshot(t *testing.T) {
//	    screen := termtest.NewScreen(80, 24)
//	    app.RenderTo(screen)
//	    termtest.AssertScreen(t, screen)  // Compares to testdata/snapshots/
//	}
//
// Update snapshots when output changes intentionally:
//
//	go test -update ./...
//
// # Capturing Live Output
//
// Capture output from code that writes to io.Writer:
//
//	capture := termtest.NewCapture(nil)
//	app.Run(capture)  // App writes ANSI output
//	screen := capture.Screen(80, 24)
//	termtest.AssertContains(t, screen, "Success")
//
// # ANSI Support
//
// The package supports common ANSI escape sequences:
//   - Cursor movement (up, down, left, right, position)
//   - Text styling (bold, italic, underline, colors)
//   - Screen clearing and line editing
//   - Scrolling and line insertion/deletion
//   - 256-color and 24-bit RGB colors
//
// Unicode and wide characters (CJK, emoji) are handled correctly with proper
// width calculation.
package termtest

import (
	"strings"

	"github.com/mattn/go-runewidth"
)

// Style represents terminal text styling attributes.
// It captures all SGR (Select Graphic Rendition) attributes that can be
// applied to terminal text via ANSI escape sequences.
type Style struct {
	Foreground Color // Text color
	Background Color // Background color
	Bold       bool  // Bold/bright text (SGR 1)
	Dim        bool  // Dimmed/faint text (SGR 2)
	Italic     bool  // Italic text (SGR 3)
	Underline  bool  // Underlined text (SGR 4)
	Blink      bool  // Blinking text (SGR 5)
	Reverse    bool  // Reversed foreground/background (SGR 7)
	Hidden     bool  // Hidden/invisible text (SGR 8)
	Strike     bool  // Strikethrough text (SGR 9)
}

// Color represents a terminal color in one of several formats.
// The Type field indicates which color format is used, and the appropriate
// fields contain the color value.
type Color struct {
	Type    ColorType
	Value   uint8         // For ColorBasic (0-15) and Color256 (0-255)
	R, G, B uint8         // For ColorRGB (24-bit true color)
}

// ColorType indicates the format of a Color value.
type ColorType uint8

const (
	ColorDefault ColorType = iota // Terminal default color
	ColorBasic                    // 16-color palette (0-7 standard, 8-15 bright)
	Color256                      // 256-color palette (SGR 38;5;n / 48;5;n)
	ColorRGB                      // 24-bit true color (SGR 38;2;r;g;b / 48;2;r;g;b)
)

// Cell represents a single character cell on the virtual terminal screen.
// Each cell contains a character, its display width, and styling information.
type Cell struct {
	Char  rune  // The Unicode character (or 0 for continuation cells)
	Width int   // Display width: 1 for normal, 2 for wide characters, 0 for continuation
	Style Style // Text styling applied to this cell
}

// Screen represents a virtual terminal screen buffer that processes ANSI sequences.
// It maintains a 2D grid of cells, cursor position, current text style, and handles
// common terminal operations like scrolling, clearing, and cursor movement.
//
// Screen implements io.Writer, so it can be used anywhere an io.Writer is expected.
// Written bytes are interpreted as ANSI-formatted terminal output.
type Screen struct {
	width    int      // Screen width in columns
	height   int      // Screen height in rows
	cells    [][]Cell // 2D grid of character cells [y][x]
	cursorX  int      // Current cursor column (0-based)
	cursorY  int      // Current cursor row (0-based)
	style    Style    // Current text style for new writes
	savedX   int      // Saved cursor X position (ESC 7/8, CSI s/u)
	savedY   int      // Saved cursor Y position (ESC 7/8, CSI s/u)
}

// NewScreen creates a new virtual terminal screen with the specified dimensions.
// The screen is initialized with empty cells (spaces) and the cursor at position (0, 0).
//
// Example:
//
//	screen := termtest.NewScreen(80, 24)  // Standard terminal size
//	screen.Write([]byte("Hello, World!"))
func NewScreen(width, height int) *Screen {
	s := &Screen{
		width:  width,
		height: height,
		cells:  make([][]Cell, height),
	}
	for y := 0; y < height; y++ {
		s.cells[y] = make([]Cell, width)
		for x := 0; x < width; x++ {
			s.cells[y][x] = Cell{Char: ' ', Width: 1}
		}
	}
	return s
}

// Size returns the screen dimensions.
func (s *Screen) Size() (width, height int) {
	return s.width, s.height
}

// Cursor returns the current cursor position.
func (s *Screen) Cursor() (x, y int) {
	return s.cursorX, s.cursorY
}

// SetCursor moves the cursor to the specified position.
func (s *Screen) SetCursor(x, y int) {
	s.cursorX = clamp(x, 0, s.width-1)
	s.cursorY = clamp(y, 0, s.height-1)
}

// Cell returns the cell at position (x, y).
// Returns a blank cell (space with width 1) if the position is out of bounds.
// Coordinates are 0-based, with (0, 0) at the top-left corner.
func (s *Screen) Cell(x, y int) Cell {
	if x < 0 || x >= s.width || y < 0 || y >= s.height {
		return Cell{Char: ' ', Width: 1}
	}
	return s.cells[y][x]
}

// SetCell sets the character and style at position (x, y).
// For wide characters (width 2), this also marks the next cell as a continuation.
// Does nothing if the position is out of bounds.
func (s *Screen) SetCell(x, y int, char rune, style Style) {
	if x < 0 || x >= s.width || y < 0 || y >= s.height {
		return
	}

	charWidth := runewidth.RuneWidth(char)
	s.cells[y][x] = Cell{Char: char, Width: charWidth, Style: style}

	// For wide characters, mark the next cell as continuation
	if charWidth == 2 && x+1 < s.width {
		s.cells[y][x+1] = Cell{Char: 0, Width: 0, Style: style}
	}
}

// WriteRune writes a single rune at the cursor position with the current style.
// The cursor advances by the character's display width (1 for normal, 2 for wide).
// Special characters are handled:
//   - '\n' moves to the next line
//   - '\r' returns to column 0
//   - '\t' advances to the next tab stop (every 8 columns)
//
// The screen scrolls up automatically when writing past the bottom.
func (s *Screen) WriteRune(r rune) {
	if r == '\n' {
		s.cursorX = 0
		s.cursorY++
		if s.cursorY >= s.height {
			s.scrollUp()
			s.cursorY = s.height - 1
		}
		return
	}
	if r == '\r' {
		s.cursorX = 0
		return
	}
	if r == '\t' {
		// Move to next tab stop (every 8 columns)
		nextTab := ((s.cursorX / 8) + 1) * 8
		s.cursorX = min(nextTab, s.width-1)
		return
	}

	charWidth := runewidth.RuneWidth(r)

	// Wrap if needed
	if s.cursorX+charWidth > s.width {
		s.cursorX = 0
		s.cursorY++
		if s.cursorY >= s.height {
			s.scrollUp()
			s.cursorY = s.height - 1
		}
	}

	s.SetCell(s.cursorX, s.cursorY, r, s.style)
	s.cursorX += charWidth
}

// WriteString writes a string at the cursor position.
func (s *Screen) WriteString(str string) {
	for _, r := range str {
		s.WriteRune(r)
	}
}

// Write implements io.Writer by parsing ANSI escape sequences and updating the screen.
// It processes cursor movement, text styling, screen clearing, and other terminal
// control sequences. Plain text is written to the screen at the cursor position.
//
// Supported ANSI sequences include:
//   - CSI sequences (ESC[...): cursor movement, SGR, erase, insert/delete
//   - OSC sequences (ESC]...): parsed and ignored
//   - Simple escapes: save/restore cursor, reset, scroll
//
// Returns the number of bytes consumed (always len(p)) and no error.
func (s *Screen) Write(p []byte) (n int, err error) {
	s.processANSI(string(p))
	return len(p), nil
}

// Clear clears the entire screen.
func (s *Screen) Clear() {
	for y := 0; y < s.height; y++ {
		for x := 0; x < s.width; x++ {
			s.cells[y][x] = Cell{Char: ' ', Width: 1}
		}
	}
	s.cursorX = 0
	s.cursorY = 0
}

// ClearLine clears the current line.
func (s *Screen) ClearLine() {
	for x := 0; x < s.width; x++ {
		s.cells[s.cursorY][x] = Cell{Char: ' ', Width: 1}
	}
}

// ClearToEndOfLine clears from cursor to end of line.
func (s *Screen) ClearToEndOfLine() {
	for x := s.cursorX; x < s.width; x++ {
		s.cells[s.cursorY][x] = Cell{Char: ' ', Width: 1}
	}
}

// ClearToStartOfLine clears from start of line to cursor.
func (s *Screen) ClearToStartOfLine() {
	for x := 0; x <= s.cursorX && x < s.width; x++ {
		s.cells[s.cursorY][x] = Cell{Char: ' ', Width: 1}
	}
}

// ClearToEndOfScreen clears from cursor to end of screen.
func (s *Screen) ClearToEndOfScreen() {
	s.ClearToEndOfLine()
	for y := s.cursorY + 1; y < s.height; y++ {
		for x := 0; x < s.width; x++ {
			s.cells[y][x] = Cell{Char: ' ', Width: 1}
		}
	}
}

// ClearToStartOfScreen clears from start of screen to cursor.
func (s *Screen) ClearToStartOfScreen() {
	for y := 0; y < s.cursorY; y++ {
		for x := 0; x < s.width; x++ {
			s.cells[y][x] = Cell{Char: ' ', Width: 1}
		}
	}
	s.ClearToStartOfLine()
}

// Text returns the screen content as plain text with ANSI sequences removed.
// Each line is a row of the screen, with trailing spaces trimmed.
// Lines are separated by newlines, and a final newline is appended.
//
// This is useful for comparing screen output in tests:
//
//	expected := "Line 1\nLine 2\nLine 3\n"
//	if screen.Text() != expected {
//	    t.Errorf("unexpected output")
//	}
func (s *Screen) Text() string {
	var lines []string
	for y := 0; y < s.height; y++ {
		var line strings.Builder
		lastNonSpace := -1
		for x := 0; x < s.width; x++ {
			cell := s.cells[y][x]
			if cell.Width == 0 {
				continue // Skip continuation cells
			}
			r := cell.Char
			if r == 0 {
				r = ' '
			}
			line.WriteRune(r)
			if r != ' ' {
				lastNonSpace = line.Len()
			}
		}
		str := line.String()
		if lastNonSpace >= 0 {
			str = str[:lastNonSpace]
		} else {
			str = ""
		}
		lines = append(lines, str)
	}
	return strings.Join(lines, "\n") + "\n"
}

// Row returns the text content of a single row with trailing spaces trimmed.
// Returns an empty string if y is out of bounds.
// Row indices are 0-based, with 0 being the top of the screen.
func (s *Screen) Row(y int) string {
	if y < 0 || y >= s.height {
		return ""
	}
	var line strings.Builder
	for x := 0; x < s.width; x++ {
		cell := s.cells[y][x]
		if cell.Width == 0 {
			continue
		}
		r := cell.Char
		if r == 0 {
			r = ' '
		}
		line.WriteRune(r)
	}
	return strings.TrimRight(line.String(), " ")
}

// Contains checks if the screen contains the given text anywhere in its content.
// The search includes the full Text() output, so it can match across line boundaries.
func (s *Screen) Contains(text string) bool {
	return strings.Contains(s.Text(), text)
}

// scrollUp scrolls the screen content up by one line.
func (s *Screen) scrollUp() {
	// Move all lines up
	for y := 0; y < s.height-1; y++ {
		s.cells[y] = s.cells[y+1]
	}
	// Clear the last line
	s.cells[s.height-1] = make([]Cell, s.width)
	for x := 0; x < s.width; x++ {
		s.cells[s.height-1][x] = Cell{Char: ' ', Width: 1}
	}
}

func clamp(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
