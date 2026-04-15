package termtest

import (
	"strconv"
	"strings"
)

// processANSI parses ANSI escape sequences and updates the screen state.
func (s *Screen) processANSI(data string) {
	i := 0
	for i < len(data) {
		if data[i] == '\x1b' {
			// Start of escape sequence
			if i+1 < len(data) {
				switch data[i+1] {
				case '[':
					// CSI sequence
					end := s.parseCSI(data[i+2:])
					if end >= 0 {
						i += 2 + end
						continue
					}
				case ']':
					// OSC sequence (Operating System Command)
					end := s.parseOSC(data[i+2:])
					if end >= 0 {
						i += 2 + end
						continue
					}
				case '(':
					// Character set designation, skip
					if i+2 < len(data) {
						i += 3
						continue
					}
				case '7':
					// Save cursor
					s.savedX = s.cursorX
					s.savedY = s.cursorY
					i += 2
					continue
				case '8':
					// Restore cursor
					s.cursorX = s.savedX
					s.cursorY = s.savedY
					i += 2
					continue
				case 'c':
					// Reset terminal
					s.Clear()
					s.style = Style{}
					i += 2
					continue
				case 'M':
					// Reverse index (scroll down)
					if s.cursorY == 0 {
						s.scrollDown()
					} else {
						s.cursorY--
					}
					i += 2
					continue
				case 'D':
					// Index (scroll up)
					if s.cursorY == s.height-1 {
						s.scrollUp()
					} else {
						s.cursorY++
					}
					i += 2
					continue
				case 'E':
					// Next line
					s.cursorX = 0
					if s.cursorY < s.height-1 {
						s.cursorY++
					}
					i += 2
					continue
				}
			}
			// Unknown escape, skip the ESC
			i++
		} else {
			// Parse a contiguous run of plain text so grapheme clusters are
			// segmented as whole display cells rather than rune-by-rune.
			nextEscape := strings.IndexByte(data[i:], '\x1b')
			if nextEscape < 0 {
				s.WriteString(data[i:])
				break
			}
			s.WriteString(data[i : i+nextEscape])
			i += nextEscape
		}
	}
}

// parseCSI parses a CSI (Control Sequence Introducer) sequence.
// Returns the number of bytes consumed (not including the CSI prefix).
func (s *Screen) parseCSI(data string) int {
	if len(data) == 0 {
		return -1
	}

	// Find the end of the sequence (final byte is 0x40-0x7E)
	end := -1
	for i := 0; i < len(data); i++ {
		c := data[i]
		if c >= 0x40 && c <= 0x7E {
			end = i
			break
		}
	}
	if end < 0 {
		return -1 // Incomplete sequence
	}

	params := data[:end]
	cmd := data[end]

	// Parse parameters
	args := parseParams(params)

	switch cmd {
	case 'A': // Cursor Up
		n := getParam(args, 0, 1)
		s.cursorY = max(0, s.cursorY-n)
	case 'B': // Cursor Down
		n := getParam(args, 0, 1)
		s.cursorY = min(s.height-1, s.cursorY+n)
	case 'C': // Cursor Forward
		n := getParam(args, 0, 1)
		s.cursorX = min(s.width-1, s.cursorX+n)
	case 'D': // Cursor Back
		n := getParam(args, 0, 1)
		s.cursorX = max(0, s.cursorX-n)
	case 'E': // Cursor Next Line
		n := getParam(args, 0, 1)
		s.cursorX = 0
		s.cursorY = min(s.height-1, s.cursorY+n)
	case 'F': // Cursor Previous Line
		n := getParam(args, 0, 1)
		s.cursorX = 0
		s.cursorY = max(0, s.cursorY-n)
	case 'G': // Cursor Horizontal Absolute
		n := getParam(args, 0, 1)
		s.cursorX = clamp(n-1, 0, s.width-1)
	case 'H', 'f': // Cursor Position
		row := getParam(args, 0, 1)
		col := getParam(args, 1, 1)
		s.cursorY = clamp(row-1, 0, s.height-1)
		s.cursorX = clamp(col-1, 0, s.width-1)
	case 'J': // Erase in Display
		n := getParam(args, 0, 0)
		switch n {
		case 0:
			s.ClearToEndOfScreen()
		case 1:
			s.ClearToStartOfScreen()
		case 2, 3:
			s.Clear()
		}
	case 'K': // Erase in Line
		n := getParam(args, 0, 0)
		switch n {
		case 0:
			s.ClearToEndOfLine()
		case 1:
			s.ClearToStartOfLine()
		case 2:
			s.ClearLine()
		}
	case 'L': // Insert Lines
		n := getParam(args, 0, 1)
		s.insertLines(n)
	case 'M': // Delete Lines
		n := getParam(args, 0, 1)
		s.deleteLines(n)
	case 'P': // Delete Characters
		n := getParam(args, 0, 1)
		s.deleteChars(n)
	case '@': // Insert Characters
		n := getParam(args, 0, 1)
		s.insertChars(n)
	case 'X': // Erase Characters
		n := getParam(args, 0, 1)
		s.eraseChars(n)
	case 'd': // Line Position Absolute
		n := getParam(args, 0, 1)
		s.cursorY = clamp(n-1, 0, s.height-1)
	case 'm': // SGR - Select Graphic Rendition
		s.processSGR(args)
	case 's': // Save Cursor Position
		s.savedX = s.cursorX
		s.savedY = s.cursorY
	case 'u': // Restore Cursor Position
		s.cursorX = s.savedX
		s.cursorY = s.savedY
	case 'r': // Set Scrolling Region (ignored for now)
		// Could implement scroll regions if needed
	case 'h', 'l': // Set/Reset Mode
		// Private modes like ?25h (show cursor) are display-only
		// We don't track cursor visibility in tests
	case 'n': // Device Status Report (ignore)
	case 'c': // Device Attributes (ignore)
	case 't': // Window manipulation (ignore)
	}

	return end + 1
}

// parseOSC parses an OSC (Operating System Command) sequence.
// Returns the number of bytes consumed (not including the OSC prefix).
func (s *Screen) parseOSC(data string) int {
	// OSC sequences end with ST (\x1b\\) or BEL (\x07)
	for i := 0; i < len(data); i++ {
		if data[i] == '\x07' {
			return i + 1
		}
		if data[i] == '\x1b' && i+1 < len(data) && data[i+1] == '\\' {
			return i + 2
		}
	}
	return -1 // Incomplete
}

// processSGR processes SGR (Select Graphic Rendition) parameters.
func (s *Screen) processSGR(args []int) {
	if len(args) == 0 {
		args = []int{0}
	}

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case 0: // Reset
			s.style = Style{}
		case 1: // Bold
			s.style.Bold = true
		case 2: // Dim
			s.style.Dim = true
		case 3: // Italic
			s.style.Italic = true
		case 4: // Underline
			s.style.Underline = true
		case 5: // Blink
			s.style.Blink = true
		case 7: // Reverse
			s.style.Reverse = true
		case 8: // Hidden
			s.style.Hidden = true
		case 9: // Strike
			s.style.Strike = true
		case 21: // Double underline (treat as underline)
			s.style.Underline = true
		case 22: // Normal intensity
			s.style.Bold = false
			s.style.Dim = false
		case 23: // Not italic
			s.style.Italic = false
		case 24: // Not underlined
			s.style.Underline = false
		case 25: // Not blinking
			s.style.Blink = false
		case 27: // Not reversed
			s.style.Reverse = false
		case 28: // Not hidden
			s.style.Hidden = false
		case 29: // Not struck
			s.style.Strike = false
		case 30, 31, 32, 33, 34, 35, 36, 37: // Standard foreground colors
			s.style.Foreground = Color{Type: ColorBasic, Value: uint8(args[i] - 30)}
		case 38: // Extended foreground color
			if i+1 < len(args) {
				switch args[i+1] {
				case 5: // 256-color
					if i+2 < len(args) {
						s.style.Foreground = Color{Type: Color256, Value: uint8(args[i+2])}
						i += 2
					}
				case 2: // RGB
					if i+4 < len(args) {
						s.style.Foreground = Color{
							Type: ColorRGB,
							R:    uint8(args[i+2]),
							G:    uint8(args[i+3]),
							B:    uint8(args[i+4]),
						}
						i += 4
					}
				}
			}
		case 39: // Default foreground
			s.style.Foreground = Color{}
		case 40, 41, 42, 43, 44, 45, 46, 47: // Standard background colors
			s.style.Background = Color{Type: ColorBasic, Value: uint8(args[i] - 40)}
		case 48: // Extended background color
			if i+1 < len(args) {
				switch args[i+1] {
				case 5: // 256-color
					if i+2 < len(args) {
						s.style.Background = Color{Type: Color256, Value: uint8(args[i+2])}
						i += 2
					}
				case 2: // RGB
					if i+4 < len(args) {
						s.style.Background = Color{
							Type: ColorRGB,
							R:    uint8(args[i+2]),
							G:    uint8(args[i+3]),
							B:    uint8(args[i+4]),
						}
						i += 4
					}
				}
			}
		case 49: // Default background
			s.style.Background = Color{}
		case 90, 91, 92, 93, 94, 95, 96, 97: // Bright foreground colors
			s.style.Foreground = Color{Type: ColorBasic, Value: uint8(args[i] - 90 + 8)}
		case 100, 101, 102, 103, 104, 105, 106, 107: // Bright background colors
			s.style.Background = Color{Type: ColorBasic, Value: uint8(args[i] - 100 + 8)}
		}
	}
}

// parseParams parses semicolon-separated parameters.
func parseParams(s string) []int {
	if s == "" {
		return nil
	}

	// Handle private mode prefix
	if len(s) > 0 && s[0] == '?' {
		s = s[1:]
	}

	parts := strings.Split(s, ";")
	result := make([]int, len(parts))
	for i, p := range parts {
		// Handle colon-separated subparameters (e.g., "38:2:r:g:b")
		if strings.Contains(p, ":") {
			subparts := strings.Split(p, ":")
			for j, sp := range subparts {
				if j == 0 {
					result[i], _ = strconv.Atoi(sp)
				} else {
					n, _ := strconv.Atoi(sp)
					result = append(result, n)
				}
			}
		} else {
			result[i], _ = strconv.Atoi(p)
		}
	}
	return result
}

// getParam returns the parameter at index, or defaultVal if not present.
func getParam(args []int, index, defaultVal int) int {
	if index < len(args) && args[index] > 0 {
		return args[index]
	}
	return defaultVal
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

type rowGlyph struct {
	start int
	width int
	cell  Cell
}

func (s *Screen) rowGlyphs(y int) []rowGlyph {
	if y < 0 || y >= s.height {
		return nil
	}

	var glyphs []rowGlyph
	for x := 0; x < s.width; x++ {
		cell := s.cells[y][x]
		if cell.Continuation {
			continue
		}
		if cell == blankCell() {
			continue
		}
		glyphs = append(glyphs, rowGlyph{
			start: x,
			width: max(cell.Width, 0),
			cell:  cell,
		})
	}
	return glyphs
}

func (s *Screen) rewriteRow(y int, glyphs []rowGlyph) {
	if y < 0 || y >= s.height {
		return
	}

	for x := 0; x < s.width; x++ {
		s.cells[y][x] = blankCell()
	}

	for _, g := range glyphs {
		if g.start < 0 || g.start >= s.width {
			continue
		}
		if g.width > 1 && g.start+g.width > s.width {
			continue
		}
		s.setCellGlyph(g.start, y, g.cell.Char, g.cell.Trailing, g.cell.Width, g.cell.Style)
	}
}

// scrollDown scrolls the screen content down by one line.
func (s *Screen) scrollDown() {
	for y := s.height - 1; y > 0; y-- {
		s.cells[y] = s.cells[y-1]
	}
	s.cells[0] = make([]Cell, s.width)
	for x := 0; x < s.width; x++ {
		s.cells[0][x] = blankCell()
	}
}

// insertLines inserts n blank lines at the cursor position.
func (s *Screen) insertLines(n int) {
	for i := 0; i < n; i++ {
		for y := s.height - 1; y > s.cursorY; y-- {
			s.cells[y] = s.cells[y-1]
		}
		s.cells[s.cursorY] = make([]Cell, s.width)
		for x := 0; x < s.width; x++ {
			s.cells[s.cursorY][x] = blankCell()
		}
	}
}

// deleteLines deletes n lines at the cursor position.
func (s *Screen) deleteLines(n int) {
	for i := 0; i < n; i++ {
		for y := s.cursorY; y < s.height-1; y++ {
			s.cells[y] = s.cells[y+1]
		}
		s.cells[s.height-1] = make([]Cell, s.width)
		for x := 0; x < s.width; x++ {
			s.cells[s.height-1][x] = blankCell()
		}
	}
}

// deleteChars deletes n characters at the cursor position.
func (s *Screen) deleteChars(n int) {
	if n <= 0 {
		return
	}
	y := s.cursorY
	start := clamp(s.cursorX, 0, s.width)
	end := min(s.width, start+n)
	prefixLimit := min(start, max(s.width-n, 0))

	var rewritten []rowGlyph
	for _, g := range s.rowGlyphs(y) {
		glyphEnd := g.start + g.width
		switch {
		case glyphEnd <= prefixLimit:
			rewritten = append(rewritten, g)
		case g.start >= end:
			g.start -= n
			rewritten = append(rewritten, g)
		default:
			// Drop glyphs that overlap the deleted column range so we never
			// leave behind a dangling continuation or split grapheme.
		}
	}
	s.rewriteRow(y, rewritten)
}

// insertChars inserts n blank characters at the cursor position.
func (s *Screen) insertChars(n int) {
	if n <= 0 {
		return
	}
	y := s.cursorY
	start := clamp(s.cursorX, 0, s.width)

	var rewritten []rowGlyph
	for _, g := range s.rowGlyphs(y) {
		glyphEnd := g.start + g.width
		switch {
		case glyphEnd <= start:
			rewritten = append(rewritten, g)
		case g.start >= start:
			g.start += n
			rewritten = append(rewritten, g)
		default:
			// Inserting into the middle of a wide glyph invalidates that glyph
			// at the target columns; drop it rather than preserve a broken half.
		}
	}
	s.rewriteRow(y, rewritten)
}

func (s *Screen) eraseChars(n int) {
	if n <= 0 {
		return
	}
	y := s.cursorY
	start := clamp(s.cursorX, 0, s.width)
	end := min(s.width, start+n)

	var rewritten []rowGlyph
	for _, g := range s.rowGlyphs(y) {
		glyphEnd := g.start + g.width
		if glyphEnd <= start || g.start >= end {
			rewritten = append(rewritten, g)
		}
	}
	s.rewriteRow(y, rewritten)
}
