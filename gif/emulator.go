package gif

import (
	"image/color"
	"regexp"
	"strconv"
	"strings"
)

// Emulator interprets ANSI escape sequences and maintains terminal screen state.
// It processes raw terminal output (as captured in .cast files) and updates
// a TerminalScreen buffer that can be rendered to GIF frames.
type Emulator struct {
	screen *TerminalScreen
	fg     color.Color
	bg     color.Color
}

// NewEmulator creates a new terminal emulator with the given dimensions.
func NewEmulator(cols, rows int) *Emulator {
	screen := NewTerminalScreen(cols, rows)
	return &Emulator{
		screen: screen,
		fg:     White,
		bg:     Black,
	}
}

// Screen returns the current terminal screen state.
func (e *Emulator) Screen() *TerminalScreen {
	return e.screen
}

// Reset clears the screen and resets all state.
func (e *Emulator) Reset() {
	e.screen.Clear()
	e.fg = White
	e.bg = Black
}

// Resize changes the terminal dimensions.
func (e *Emulator) Resize(cols, rows int) {
	// Create new screen with new dimensions
	newScreen := NewTerminalScreen(cols, rows)

	// Copy existing content that fits
	maxRows := rows
	if e.screen.Height < maxRows {
		maxRows = e.screen.Height
	}
	maxCols := cols
	if e.screen.Width < maxCols {
		maxCols = e.screen.Width
	}

	for y := 0; y < maxRows; y++ {
		for x := 0; x < maxCols; x++ {
			newScreen.Cells[y][x] = e.screen.Cells[y][x]
		}
	}

	// Adjust cursor position
	newScreen.CursorX = e.screen.CursorX
	newScreen.CursorY = e.screen.CursorY
	if newScreen.CursorX >= cols {
		newScreen.CursorX = cols - 1
	}
	if newScreen.CursorY >= rows {
		newScreen.CursorY = rows - 1
	}

	e.screen = newScreen
}

// Write implements io.Writer. It processes terminal output data
// and updates the screen state accordingly.
func (e *Emulator) Write(p []byte) (n int, err error) {
	e.ProcessOutput(string(p))
	return len(p), nil
}

// ProcessOutput processes raw terminal output with ANSI escape sequences.
func (e *Emulator) ProcessOutput(data string) {
	i := 0
	for i < len(data) {
		// Check for escape sequences
		if data[i] == '\x1b' {
			// Try to match CSI sequence
			if i+1 < len(data) && data[i+1] == '[' {
				match := ansiCSI.FindStringSubmatchIndex(data[i:])
				if match != nil {
					params := data[i+match[2] : i+match[3]]
					cmd := data[i+match[4] : i+match[5]]
					e.processCSI(params, cmd)
					i += match[1]
					continue
				}
			}

			// Try to match OSC sequence
			if i+1 < len(data) && data[i+1] == ']' {
				match := ansiOSC.FindStringIndex(data[i:])
				if match != nil {
					i += match[1]
					continue
				}
			}

			// Try to match simple escape
			match := ansiSimple.FindStringIndex(data[i:])
			if match != nil {
				i += match[1]
				continue
			}

			// Skip unknown escape
			i++
			continue
		}

		// Handle control characters
		switch data[i] {
		case '\n':
			e.screen.CursorX = 0
			e.screen.CursorY++
			if e.screen.CursorY >= e.screen.Height {
				e.screen.scrollUp()
				e.screen.CursorY = e.screen.Height - 1
			}
		case '\r':
			e.screen.CursorX = 0
		case '\b':
			if e.screen.CursorX > 0 {
				e.screen.CursorX--
			}
		case '\t':
			// Tab to next 8-column boundary
			e.screen.CursorX = ((e.screen.CursorX / 8) + 1) * 8
			if e.screen.CursorX >= e.screen.Width {
				e.screen.CursorX = e.screen.Width - 1
			}
		case '\x07': // Bell - ignore
		default:
			if data[i] >= 32 { // Printable character
				if e.screen.CursorX >= e.screen.Width {
					e.screen.CursorX = 0
					e.screen.CursorY++
					if e.screen.CursorY >= e.screen.Height {
						e.screen.scrollUp()
						e.screen.CursorY = e.screen.Height - 1
					}
				}
				e.screen.SetCell(e.screen.CursorX, e.screen.CursorY, rune(data[i]), e.fg, e.bg)
				e.screen.CursorX++
			}
		}
		i++
	}
}

// ANSI escape sequence patterns
var (
	ansiCSI    = regexp.MustCompile(`\x1b\[([0-9;]*)([A-Za-z])`)
	ansiOSC    = regexp.MustCompile(`\x1b\]([^\x07\x1b]*)(?:\x07|\x1b\\)`)
	ansiSimple = regexp.MustCompile(`\x1b[()][AB012]`)
)

// processCSI handles CSI escape sequences
func (e *Emulator) processCSI(params, cmd string) {
	switch cmd {
	case "m": // SGR - Select Graphic Rendition
		e.processSGR(params)
	case "H", "f": // Cursor Position
		row, col := 1, 1
		if params != "" {
			parts := strings.Split(params, ";")
			if len(parts) >= 1 && parts[0] != "" {
				row, _ = strconv.Atoi(parts[0])
			}
			if len(parts) >= 2 && parts[1] != "" {
				col, _ = strconv.Atoi(parts[1])
			}
		}
		e.screen.MoveCursor(col-1, row-1)
	case "A": // Cursor Up
		n := parseParam(params, 1)
		e.screen.CursorY -= n
		if e.screen.CursorY < 0 {
			e.screen.CursorY = 0
		}
	case "B": // Cursor Down
		n := parseParam(params, 1)
		e.screen.CursorY += n
		if e.screen.CursorY >= e.screen.Height {
			e.screen.CursorY = e.screen.Height - 1
		}
	case "C": // Cursor Forward
		n := parseParam(params, 1)
		e.screen.CursorX += n
		if e.screen.CursorX >= e.screen.Width {
			e.screen.CursorX = e.screen.Width - 1
		}
	case "D": // Cursor Back
		n := parseParam(params, 1)
		e.screen.CursorX -= n
		if e.screen.CursorX < 0 {
			e.screen.CursorX = 0
		}
	case "J": // Erase in Display
		n := parseParam(params, 0)
		switch n {
		case 0: // Clear from cursor to end
			e.clearToEnd()
		case 1: // Clear from start to cursor
			e.clearToStart()
		case 2, 3: // Clear entire screen
			e.screen.Clear()
		}
	case "K": // Erase in Line
		n := parseParam(params, 0)
		switch n {
		case 0: // Clear from cursor to end of line
			for x := e.screen.CursorX; x < e.screen.Width; x++ {
				e.screen.SetCell(x, e.screen.CursorY, ' ', e.fg, e.bg)
			}
		case 1: // Clear from start to cursor
			for x := 0; x <= e.screen.CursorX; x++ {
				e.screen.SetCell(x, e.screen.CursorY, ' ', e.fg, e.bg)
			}
		case 2: // Clear entire line
			for x := 0; x < e.screen.Width; x++ {
				e.screen.SetCell(x, e.screen.CursorY, ' ', e.fg, e.bg)
			}
		}
	case "G": // Cursor Horizontal Absolute
		col := parseParam(params, 1)
		e.screen.CursorX = col - 1
		if e.screen.CursorX < 0 {
			e.screen.CursorX = 0
		}
		if e.screen.CursorX >= e.screen.Width {
			e.screen.CursorX = e.screen.Width - 1
		}
	case "d": // Cursor Vertical Absolute
		row := parseParam(params, 1)
		e.screen.CursorY = row - 1
		if e.screen.CursorY < 0 {
			e.screen.CursorY = 0
		}
		if e.screen.CursorY >= e.screen.Height {
			e.screen.CursorY = e.screen.Height - 1
		}
	case "E": // Cursor Next Line
		n := parseParam(params, 1)
		e.screen.CursorX = 0
		e.screen.CursorY += n
		if e.screen.CursorY >= e.screen.Height {
			e.screen.CursorY = e.screen.Height - 1
		}
	case "F": // Cursor Previous Line
		n := parseParam(params, 1)
		e.screen.CursorX = 0
		e.screen.CursorY -= n
		if e.screen.CursorY < 0 {
			e.screen.CursorY = 0
		}
	}
}

func parseParam(params string, defaultVal int) int {
	if params == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(params)
	if err != nil {
		return defaultVal
	}
	return n
}

func (e *Emulator) clearToEnd() {
	// Clear from cursor to end of line
	for x := e.screen.CursorX; x < e.screen.Width; x++ {
		e.screen.SetCell(x, e.screen.CursorY, ' ', White, e.bg)
	}
	// Clear remaining lines
	for y := e.screen.CursorY + 1; y < e.screen.Height; y++ {
		for x := 0; x < e.screen.Width; x++ {
			e.screen.SetCell(x, y, ' ', White, e.bg)
		}
	}
}

func (e *Emulator) clearToStart() {
	// Clear from start to cursor
	for y := 0; y < e.screen.CursorY; y++ {
		for x := 0; x < e.screen.Width; x++ {
			e.screen.SetCell(x, y, ' ', White, e.bg)
		}
	}
	for x := 0; x <= e.screen.CursorX; x++ {
		e.screen.SetCell(x, e.screen.CursorY, ' ', White, e.bg)
	}
}

// processSGR handles SGR (Select Graphic Rendition) parameters
func (e *Emulator) processSGR(params string) {
	if params == "" || params == "0" {
		e.fg = White
		e.bg = Black
		return
	}

	parts := strings.Split(params, ";")
	for i := 0; i < len(parts); i++ {
		code, _ := strconv.Atoi(parts[i])
		switch {
		case code == 0:
			e.fg = White
			e.bg = Black
		case code >= 30 && code <= 37:
			e.fg = ansiColor(code - 30)
		case code == 38:
			// Extended foreground color
			if i+1 < len(parts) {
				mode, _ := strconv.Atoi(parts[i+1])
				if mode == 5 && i+2 < len(parts) {
					// 256 color mode
					idx, _ := strconv.Atoi(parts[i+2])
					e.fg = color256(idx)
					i += 2
				} else if mode == 2 && i+4 < len(parts) {
					// True color (RGB)
					r, _ := strconv.Atoi(parts[i+2])
					g, _ := strconv.Atoi(parts[i+3])
					b, _ := strconv.Atoi(parts[i+4])
					e.fg = RGB(uint8(r), uint8(g), uint8(b))
					i += 4
				}
			}
		case code == 39:
			e.fg = White
		case code >= 40 && code <= 47:
			e.bg = ansiColor(code - 40)
		case code == 48:
			// Extended background color
			if i+1 < len(parts) {
				mode, _ := strconv.Atoi(parts[i+1])
				if mode == 5 && i+2 < len(parts) {
					idx, _ := strconv.Atoi(parts[i+2])
					e.bg = color256(idx)
					i += 2
				} else if mode == 2 && i+4 < len(parts) {
					r, _ := strconv.Atoi(parts[i+2])
					g, _ := strconv.Atoi(parts[i+3])
					b, _ := strconv.Atoi(parts[i+4])
					e.bg = RGB(uint8(r), uint8(g), uint8(b))
					i += 4
				}
			}
		case code == 49:
			e.bg = Black
		case code >= 90 && code <= 97:
			e.fg = ansiBrightColor(code - 90)
		case code >= 100 && code <= 107:
			e.bg = ansiBrightColor(code - 100)
		}
	}
}

// ansiColor returns the RGBA color for basic ANSI color codes 0-7
func ansiColor(code int) color.RGBA {
	colors := []color.RGBA{
		{0, 0, 0, 255},       // Black
		{170, 0, 0, 255},     // Red
		{0, 170, 0, 255},     // Green
		{170, 85, 0, 255},    // Yellow
		{0, 0, 170, 255},     // Blue
		{170, 0, 170, 255},   // Magenta
		{0, 170, 170, 255},   // Cyan
		{170, 170, 170, 255}, // White
	}
	if code >= 0 && code < len(colors) {
		return colors[code]
	}
	return colors[7]
}

// ansiBrightColor returns the bright variant of ANSI colors
func ansiBrightColor(code int) color.RGBA {
	colors := []color.RGBA{
		{85, 85, 85, 255},    // Bright Black
		{255, 85, 85, 255},   // Bright Red
		{85, 255, 85, 255},   // Bright Green
		{255, 255, 85, 255},  // Bright Yellow
		{85, 85, 255, 255},   // Bright Blue
		{255, 85, 255, 255},  // Bright Magenta
		{85, 255, 255, 255},  // Bright Cyan
		{255, 255, 255, 255}, // Bright White
	}
	if code >= 0 && code < len(colors) {
		return colors[code]
	}
	return colors[7]
}

// color256 returns the color for 256-color mode
func color256(idx int) color.RGBA {
	if idx < 16 {
		// Standard colors
		if idx < 8 {
			return ansiColor(idx)
		}
		return ansiBrightColor(idx - 8)
	} else if idx < 232 {
		// 216 color cube (6x6x6)
		idx -= 16
		r := (idx / 36) * 51
		g := ((idx / 6) % 6) * 51
		b := (idx % 6) * 51
		return color.RGBA{uint8(r), uint8(g), uint8(b), 255}
	} else {
		// Grayscale (24 shades)
		gray := uint8((idx-232)*10 + 8)
		return color.RGBA{gray, gray, gray, 255}
	}
}
