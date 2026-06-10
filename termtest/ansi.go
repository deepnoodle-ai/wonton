package termtest

import (
	"strconv"
	"strings"
	"unicode/utf8"
)

// processANSI parses ANSI escape sequences and updates the screen state.
// It returns the number of bytes consumed; a trailing incomplete escape
// sequence or UTF-8 rune is left unconsumed so Write can buffer it until
// the remaining bytes arrive.
func (s *Screen) processANSI(data string) int {
	i := 0
	for i < len(data) {
		if data[i] == '\x1b' {
			// Start of escape sequence
			if i+1 >= len(data) {
				return i // Lone ESC at end of data: wait for more bytes
			}
			switch data[i+1] {
			case '\x1b':
				// ESC ESC (e.g. Alt-prefixed sequence): consume the first
				// ESC and reprocess from the second.
				i++
				continue
			case '[':
				// CSI sequence
				end := s.parseCSI(data[i+2:])
				if end >= 0 {
					i += 2 + end
					continue
				}
				return i // Incomplete sequence
			case ']':
				// OSC sequence (Operating System Command)
				end := s.parseOSC(data[i+2:])
				if end >= 0 {
					i += 2 + end
					continue
				}
				return i // Incomplete sequence
			case 'P', 'X', '^', '_':
				// DCS/SOS/PM/APC: string sequence, consume payload through ST
				end := parseStringTerminator(data[i+2:])
				if end >= 0 {
					i += 2 + end
					continue
				}
				return i // Incomplete sequence
			case '(', ')', '*', '+':
				// Character set designation, skip
				if i+2 < len(data) {
					i += 3
					continue
				}
				return i // Incomplete sequence
			case '7':
				// Save cursor and attributes (DECSC)
				s.savedX = s.cursorX
				s.savedY = s.cursorY
				s.savedStyle = s.style
				i += 2
				continue
			case '8':
				// Restore cursor and attributes (DECRC)
				s.wrapPending = false
				s.cursorX = s.savedX
				s.cursorY = s.savedY
				s.style = s.savedStyle
				i += 2
				continue
			case 'c':
				// Reset terminal (RIS)
				s.Reset()
				i += 2
				continue
			case 'M':
				// Reverse index (scroll down)
				s.wrapPending = false
				if s.cursorY == 0 {
					s.scrollDown()
				} else {
					s.cursorY--
				}
				i += 2
				continue
			case 'D':
				// Index (scroll up)
				s.wrapPending = false
				if s.cursorY == s.height-1 {
					s.scrollUp()
				} else {
					s.cursorY++
				}
				i += 2
				continue
			case 'E':
				// Next line (NEL): carriage return + index, scrolls at bottom
				s.wrapPending = false
				s.cursorX = 0
				s.lineFeed()
				i += 2
				continue
			}
			// Unknown escape (ESC =, ESC >, ...): skip the ESC and its final byte
			i += 2
		} else {
			// Parse a contiguous run of plain text so grapheme clusters are
			// segmented as whole display cells rather than rune-by-rune.
			nextEscape := strings.IndexByte(data[i:], '\x1b')
			if nextEscape < 0 {
				run := data[i:]
				tail := incompleteRuneTail(run)
				s.WriteString(run[:len(run)-tail])
				return len(data) - tail
			}
			s.WriteString(data[i : i+nextEscape])
			i += nextEscape
		}
	}
	return len(data)
}

// incompleteRuneTail returns the number of trailing bytes in str that form
// the prefix of an incomplete multi-byte UTF-8 rune, or 0 if the string ends
// on a rune boundary (or with bytes that can never become a valid rune).
func incompleteRuneTail(str string) int {
	for back := 1; back <= utf8.UTFMax && back <= len(str); back++ {
		b := str[len(str)-back]
		if utf8.RuneStart(b) {
			if b >= 0xC0 && !utf8.FullRuneInString(str[len(str)-back:]) {
				return back
			}
			return 0
		}
	}
	return 0
}

// parseStringTerminator scans for the ST (ESC \) terminator of a DCS, SOS,
// PM, or APC string sequence. Returns the number of bytes consumed including
// the terminator, or -1 if the sequence is incomplete.
func parseStringTerminator(data string) int {
	for i := 0; i < len(data); i++ {
		if data[i] == '\x1b' && i+1 < len(data) && data[i+1] == '\\' {
			return i + 2
		}
	}
	return -1
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

	// Private-parameter sequences (CSI ?, CSI >, CSI =) such as DEC private
	// modes or xterm modifyOtherKeys don't affect screen content; consume
	// and ignore them so e.g. "CSI >4;2m" isn't misread as SGR.
	if len(params) > 0 && (params[0] == '?' || params[0] == '>' || params[0] == '=') {
		return end + 1
	}

	// Parse parameters
	args := parseParams(params)

	// Commands that move the cursor or edit the line cancel a deferred wrap.
	switch cmd {
	case 'm', 'n', 'c', 't', 'h', 'l', 'r':
		// No cursor effect; keep deferred-wrap state.
	default:
		s.wrapPending = false
	}

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
			// ED never moves the cursor (apps send ESC[2J ESC[H as a pair)
			s.clearCells()
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
		s.processSGR(parseSGRParams(params))
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

// sgrParam is a single SGR parameter with optional colon-separated
// sub-parameters (e.g. "38:2::r:g:b" or "4:3"). Sub-parameters are scoped to
// their parameter per ECMA-48 / ITU T.416, unlike semicolon-separated
// parameters which form independent top-level entries.
type sgrParam struct {
	value int
	sub   []int
}

// processSGR processes SGR (Select Graphic Rendition) parameters.
func (s *Screen) processSGR(params []sgrParam) {
	if len(params) == 0 {
		params = []sgrParam{{value: 0}}
	}

	for i := 0; i < len(params); i++ {
		p := params[i]
		switch p.value {
		case 0: // Reset
			s.style = Style{}
		case 1: // Bold
			s.style.Bold = true
		case 2: // Dim
			s.style.Dim = true
		case 3: // Italic
			s.style.Italic = true
		case 4: // Underline; "4:0" disables, "4:n" selects an underline style
			if len(p.sub) > 0 && p.sub[0] == 0 {
				s.style.Underline = false
			} else {
				s.style.Underline = true
			}
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
			s.style.Foreground = Color{Type: ColorBasic, Value: uint8(p.value - 30)}
		case 38: // Extended foreground color
			if color, consumed, ok := parseExtendedColor(params, i); ok {
				s.style.Foreground = color
				i += consumed
			}
		case 39: // Default foreground
			s.style.Foreground = Color{}
		case 40, 41, 42, 43, 44, 45, 46, 47: // Standard background colors
			s.style.Background = Color{Type: ColorBasic, Value: uint8(p.value - 40)}
		case 48: // Extended background color
			if color, consumed, ok := parseExtendedColor(params, i); ok {
				s.style.Background = color
				i += consumed
			}
		case 49: // Default background
			s.style.Background = Color{}
		case 90, 91, 92, 93, 94, 95, 96, 97: // Bright foreground colors
			s.style.Foreground = Color{Type: ColorBasic, Value: uint8(p.value - 90 + 8)}
		case 100, 101, 102, 103, 104, 105, 106, 107: // Bright background colors
			s.style.Background = Color{Type: ColorBasic, Value: uint8(p.value - 100 + 8)}
		}
	}
}

// parseExtendedColor parses an extended color (SGR 38/48) starting at
// params[i]. It supports the semicolon form (38;5;n and 38;2;r;g;b), which
// consumes following top-level parameters, and the colon sub-parameter forms
// (38:5:n, 38:2:r:g:b, and ITU T.416 38:2:colorspace:r:g:b), which are
// self-contained. Returns the color, the number of extra top-level
// parameters consumed, and whether parsing succeeded.
func parseExtendedColor(params []sgrParam, i int) (Color, int, bool) {
	if sub := params[i].sub; len(sub) > 0 {
		switch sub[0] {
		case 5: // 256-color
			if len(sub) >= 2 {
				return Color{Type: Color256, Value: uint8(sub[1])}, 0, true
			}
		case 2: // RGB
			if len(sub) >= 5 {
				// 2:colorspace:r:g:b (colorspace ID ignored)
				return Color{Type: ColorRGB, R: uint8(sub[2]), G: uint8(sub[3]), B: uint8(sub[4])}, 0, true
			}
			if len(sub) == 4 {
				// 2:r:g:b
				return Color{Type: ColorRGB, R: uint8(sub[1]), G: uint8(sub[2]), B: uint8(sub[3])}, 0, true
			}
		}
		return Color{}, 0, false
	}

	// Semicolon form: color components are independent parameters
	if i+1 < len(params) {
		switch params[i+1].value {
		case 5: // 256-color
			if i+2 < len(params) {
				return Color{Type: Color256, Value: uint8(params[i+2].value)}, 2, true
			}
		case 2: // RGB
			if i+4 < len(params) {
				return Color{
					Type: ColorRGB,
					R:    uint8(params[i+2].value),
					G:    uint8(params[i+3].value),
					B:    uint8(params[i+4].value),
				}, 4, true
			}
		}
	}
	return Color{}, 0, false
}

// parseParams parses semicolon-separated parameters.
// Colon-separated sub-parameters are only meaningful for SGR, which uses
// parseSGRParams; here only each parameter's primary value is kept.
func parseParams(s string) []int {
	if s == "" {
		return nil
	}

	// Strip a private-parameter prefix for robustness; parseCSI normally
	// filters private sequences out before they get here.
	if s[0] == '?' || s[0] == '>' || s[0] == '=' {
		s = s[1:]
		if s == "" {
			return nil
		}
	}

	parts := strings.Split(s, ";")
	result := make([]int, len(parts))
	for i, p := range parts {
		if idx := strings.IndexByte(p, ':'); idx >= 0 {
			p = p[:idx]
		}
		result[i], _ = strconv.Atoi(p)
	}
	return result
}

// parseSGRParams parses SGR parameters, preserving the grouping of
// colon-separated sub-parameters within each semicolon-separated parameter.
func parseSGRParams(s string) []sgrParam {
	if s == "" {
		return nil
	}

	parts := strings.Split(s, ";")
	result := make([]sgrParam, len(parts))
	for i, p := range parts {
		subparts := strings.Split(p, ":")
		result[i].value, _ = strconv.Atoi(subparts[0])
		if len(subparts) > 1 {
			result[i].sub = make([]int, len(subparts)-1)
			for j, sp := range subparts[1:] {
				result[i].sub[j], _ = strconv.Atoi(sp)
			}
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

// deleteChars deletes n characters at the cursor position (DCH).
// Characters left of the cursor are unaffected; characters right of the
// deleted range shift left and the line is blank-filled at the end.
func (s *Screen) deleteChars(n int) {
	if n <= 0 {
		return
	}
	y := s.cursorY
	start := clamp(s.cursorX, 0, s.width)
	end := min(s.width, start+n)
	shift := end - start

	var rewritten []rowGlyph
	for _, g := range s.rowGlyphs(y) {
		glyphEnd := g.start + g.width
		switch {
		case glyphEnd <= start:
			rewritten = append(rewritten, g)
		case g.start >= end:
			g.start -= shift
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
