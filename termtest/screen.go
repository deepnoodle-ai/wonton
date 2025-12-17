// Package termtest provides snapshot testing utilities for terminal applications.
// It works with any terminal output by capturing ANSI sequences and comparing
// screen states against golden files.
//
// Basic usage:
//
//	func TestMyApp(t *testing.T) {
//	    screen := termtest.NewScreen(80, 24)
//	    screen.Write([]byte("Hello, World!"))
//	    termtest.AssertScreen(t, screen)
//	}
//
// To update snapshots, run tests with -update flag:
//
//	go test -update ./...
package termtest

import (
	"strings"

	"github.com/mattn/go-runewidth"
)

// Style represents terminal text styling.
type Style struct {
	Foreground Color
	Background Color
	Bold       bool
	Dim        bool
	Italic     bool
	Underline  bool
	Blink      bool
	Reverse    bool
	Hidden     bool
	Strike     bool
}

// Color represents a terminal color.
type Color struct {
	Type  ColorType
	Value uint8 // For 256-color mode
	R, G, B uint8 // For true color
}

// ColorType indicates the type of color.
type ColorType uint8

const (
	ColorDefault ColorType = iota
	ColorBasic           // 0-7 standard, 8-15 bright
	Color256             // 256-color palette
	ColorRGB             // 24-bit true color
)

// Cell represents a single character cell on the screen.
type Cell struct {
	Char  rune
	Width int // Display width (1 or 2 for wide chars, 0 for continuation)
	Style Style
}

// Screen represents a virtual terminal screen buffer.
type Screen struct {
	width    int
	height   int
	cells    [][]Cell
	cursorX  int
	cursorY  int
	style    Style
	savedX   int
	savedY   int
}

// NewScreen creates a new screen with the given dimensions.
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

// Cell returns the cell at the given position.
func (s *Screen) Cell(x, y int) Cell {
	if x < 0 || x >= s.width || y < 0 || y >= s.height {
		return Cell{Char: ' ', Width: 1}
	}
	return s.cells[y][x]
}

// SetCell sets the cell at the given position.
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

// WriteRune writes a single rune at the cursor position and advances the cursor.
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

// Write implements io.Writer by parsing ANSI sequences and updating the screen.
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

// Text returns the screen content as plain text.
// Trailing spaces on each line are trimmed.
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

// Row returns the text content of a single row.
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

// Contains checks if the screen contains the given text anywhere.
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
