package gif

import (
	"image/color"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Emulator interprets ANSI escape sequences and maintains terminal screen state.
// It processes raw terminal output (including control characters and escape
// sequences) and updates a TerminalScreen buffer that can be rendered to GIF
// frames.
//
// The emulator supports:
//   - Standard ANSI color codes (30-37, 40-47, 90-97, 100-107)
//   - 256-color mode (ESC[38;5;Nm and ESC[48;5;Nm)
//   - True color / 24-bit RGB (ESC[38;2;R;G;Bm and ESC[48;2;R;G;Bm)
//   - Cursor movement (CSI sequences: H, A, B, C, D, E, F, G, d)
//   - Screen clearing (CSI J and K sequences)
//   - Common control characters (newline, carriage return, backspace, tab)
//
// Example:
//
//	emulator := gif.NewEmulator(80, 24)
//	emulator.ProcessOutput("Hello \x1b[31mRed\x1b[0m World\n")
//	emulator.ProcessOutput("Line 2")
//
//	renderer := gif.NewTerminalRenderer(emulator.Screen(), 8)
//	renderer.RenderFrame(10)
//	renderer.Save("output.gif")
type skipMode uint8

const (
	skipNone skipMode = iota
	skipST            // skipping DCS/SOS/PM/APC payload until ST (ESC \)
	skipOSC           // skipping OSC payload until BEL or ST (ESC \)
)

type Emulator struct {
	screen  *TerminalScreen
	fg      color.Color
	bg      color.Color
	pending string
	skip    skipMode
	skipEsc bool
}

// maxPendingBytes bounds how many bytes of an incomplete escape sequence are
// buffered between ProcessOutput calls before being dropped.
const maxPendingBytes = 1 << 20

func (e *Emulator) savePending(s string) {
	if len(s) <= maxPendingBytes {
		e.pending = s
		return
	}
	e.pending = ""
	if len(s) >= 2 && s[0] == '\x1b' {
		switch s[1] {
		case ']':
			e.skip = skipOSC
			e.skipEsc = len(s) > 2 && s[len(s)-1] == '\x1b'
		case 'P', 'X', '^', '_':
			e.skip = skipST
			e.skipEsc = len(s) > 2 && s[len(s)-1] == '\x1b'
		}
	}
}

// NewEmulator creates a new terminal emulator with the specified dimensions.
// The cols and rows parameters define the terminal size in character cells.
// Standard terminal sizes include 80x24 (classic), 80x25 (DOS), and 132x24.
func NewEmulator(cols, rows int) *Emulator {
	screen := NewTerminalScreen(cols, rows)
	return &Emulator{
		screen: screen,
		fg:     White,
		bg:     Black,
	}
}

// Screen returns the current terminal screen state, which contains the
// character cells, colors, and cursor position. The returned screen can
// be rendered to GIF frames using a TerminalRenderer.
func (e *Emulator) Screen() *TerminalScreen {
	return e.screen
}

// Reset clears the screen and resets all state to initial values.
// This includes clearing all text, resetting colors to defaults (white on
// black), and moving the cursor to the home position (0, 0).
func (e *Emulator) Reset() {
	e.screen.Clear()
	e.fg = White
	e.bg = Black
	e.pending = ""
	e.skip = skipNone
	e.skipEsc = false
}

// Resize changes the terminal dimensions, preserving as much existing content
// as possible. If the new size is smaller, content outside the bounds is lost.
// If larger, new areas are filled with spaces using default colors.
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
		// A wide cluster cut in half by the new right edge becomes a space.
		if maxCols > 0 && newScreen.Cells[y][maxCols-1].Width == 2 {
			newScreen.SetCell(maxCols-1, y, ' ', nil, newScreen.Cells[y][maxCols-1].BG)
		}
	}

	// Adjust cursor position
	newScreen.CursorX = e.screen.CursorX
	newScreen.CursorY = e.screen.CursorY
	newScreen.CursorX = max(0, min(newScreen.CursorX, cols-1))
	newScreen.CursorY = max(0, min(newScreen.CursorY, rows-1))

	e.screen = newScreen
}

// Write implements io.Writer, allowing the emulator to be used as a writer
// for terminal output. It processes the data through ProcessOutput and always
// returns the full length written with no error.
//
// This makes Emulator compatible with any code that writes to an io.Writer,
// such as terminal applications or command execution output.
func (e *Emulator) Write(p []byte) (n int, err error) {
	e.ProcessOutput(string(p))
	return len(p), nil
}

// ProcessOutput processes raw terminal output containing printable characters,
// control characters, and ANSI escape sequences. It updates the screen state,
// cursor position, and text attributes accordingly.
//
// Supported sequences include:
//   - CSI sequences (ESC[...): cursor movement, colors, clearing
//   - OSC sequences (ESC]...): ignored (window title, etc.)
//   - Control characters: \n, \r, \b, \t, and printable ASCII
func (e *Emulator) ProcessOutput(data string) {
	if data == "" {
		return
	}
	if e.skip != skipNone {
		if e.skipEsc {
			data = "\x1b" + data
			e.skipEsc = false
		}
		end, status := scanStringSequence(data, e.skip == skipOSC)
		if status == seqIncomplete {
			e.skipEsc = data[len(data)-1] == '\x1b'
			return
		}
		e.skip = skipNone
		data = data[end:]
	}
	if e.pending != "" {
		data = e.pending + data
		e.pending = ""
	}
	i := 0
	for i < len(data) {
		// Check for escape sequences
		if data[i] == '\x1b' {
			if i+1 >= len(data) {
				e.savePending(data[i:])
				return
			}

			switch data[i+1] {
			case '\x1b':
				// ESC ESC: consume the first ESC and reprocess from the second.
				i++
				continue

			case '[':
				// CSI sequence: ESC [ <params 0x30-0x3F> <intermediates 0x20-0x2F> <final 0x40-0x7E>
				end, status := scanCSI(data[i+2:])
				switch status {
				case seqIncomplete:
					e.savePending(data[i:])
					return
				case seqAborted:
					i += 2 + end
					continue
				case seqComplete:
					match := ansiCSI.FindStringSubmatchIndex(data[i : i+2+end])
					if match != nil {
						params := data[i+match[2] : i+match[3]]
						intermediates := data[i+match[4] : i+match[5]]
						cmd := data[i+match[6] : i+match[7]]
						// Sequences with intermediate bytes (such as DECSCUSR
						// "ESC [ 0 SP q") do not affect the screen model.
						if intermediates == "" {
							e.processCSI(params, cmd)
						}
					}
					i += 2 + end
					continue
				}

			case ']':
				// OSC sequence: ESC ] ... (BEL | ESC \)
				end, status := scanStringSequence(data[i+2:], true)
				if status == seqIncomplete {
					e.savePending(data[i:])
					return
				}
				i += 2 + end
				continue

			case 'P', 'X', '^', '_':
				// DCS, SOS, PM, APC string sequence: ESC <type> ... ESC \
				end, status := scanStringSequence(data[i+2:], false)
				if status == seqIncomplete {
					e.savePending(data[i:])
					return
				}
				i += 2 + end
				continue

			default:
				// Simple or intermediate-prefixed escape: ESC <0x20-0x2F>* <0x30-0x7E>
				j := i + 1
				for j < len(data) && data[j] >= 0x20 && data[j] <= 0x2F {
					j++
				}
				if j >= len(data) {
					e.savePending(data[i:])
					return
				}
				if data[j] >= 0x30 && data[j] <= 0x7E {
					i = j + 1
					continue
				}
				i++
				continue
			}
		}

		// Decode the next rune (handles multi-byte UTF-8)
		r, size := utf8.DecodeRuneInString(data[i:])

		// Handle control characters
		switch r {
		case '\n':
			e.screen.CursorX = 0
			e.screen.lineFeed()
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
			if r >= 32 { // Printable character
				// Print the run up to the next control character or escape
				// at once, so the screen can split it into grapheme
				// clusters. ESC and the other controls are below 0x20, and
				// multi-byte UTF-8 sequences contain no such bytes.
				end := i + size
				for end < len(data) && data[end] >= 0x20 {
					end++
				}
				run := data[i:end]
				if end == len(data) {
					if tail := incompleteRuneTail(run); tail > 0 {
						if len(run) > tail {
							e.screen.print(run[:len(run)-tail], e.fg, e.bg)
						}
						e.savePending(run[len(run)-tail:])
						return
					}
				}
				e.screen.print(run, e.fg, e.bg)
				i = end
				continue
			}
		}
		i += size
	}
}

type seqStatus int

const (
	seqComplete seqStatus = iota
	seqIncomplete
	seqAborted
)

// scanCSI scans the body of a CSI sequence (after "ESC [").
func scanCSI(data string) (int, seqStatus) {
	for i := 0; i < len(data); i++ {
		c := data[i]
		switch {
		case c == 0x18 || c == 0x1A: // CAN or SUB aborts the sequence
			return i + 1, seqAborted
		case c == 0x1B: // ESC aborts and starts a new sequence
			return i, seqAborted
		case c >= 0x40 && c <= 0x7E: // Final byte
			return i + 1, seqComplete
		}
	}
	return 0, seqIncomplete
}

// scanStringSequence scans the payload of an OSC (allowBEL=true) or
// DCS/SOS/PM/APC (allowBEL=false) sequence through its BEL or ST (ESC \)
// terminator.
func scanStringSequence(data string, allowBEL bool) (int, seqStatus) {
	for i := 0; i < len(data); i++ {
		c := data[i]
		if allowBEL && c == '\x07' {
			return i + 1, seqComplete
		}
		if c == 0x18 || c == 0x1A {
			return i + 1, seqAborted
		}
		if c == '\x1b' {
			if i+1 >= len(data) {
				return 0, seqIncomplete
			}
			if data[i+1] == '\\' {
				return i + 2, seqComplete
			}
			return i, seqAborted
		}
	}
	return 0, seqIncomplete
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

// ansiCSI matches CSI sequences per ECMA-48:
// ESC [ <parameter bytes 0x30-0x3F> <intermediate bytes 0x20-0x2F> <final byte 0x40-0x7E>
// Parameter bytes include digits, ';', ':', and private markers (<, =, >, ?).
var ansiCSI = regexp.MustCompile(`^\x1b\[([0-?]*)([ -/]*)([@-~])`)

// processCSI handles CSI escape sequences
func (e *Emulator) processCSI(params, cmd string) {
	if params != "" && strings.ContainsRune("<=>?", rune(params[0])) {
		// Private sequences (such as ESC[?25l or ESC[>4;2m) have their own
		// meanings. Only the selective erase forms of J and K (DECSED,
		// DECSEL) map onto the screen model; ignore the rest so they are not
		// misread as standard sequences.
		if params[0] != '?' || (cmd != "J" && cmd != "K") {
			return
		}
		params = params[1:]
	}
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
	if err != nil || (n == 0 && defaultVal > 0) {
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
		if strings.Contains(parts[i], ":") {
			e.processSGRSubparams(strings.Split(parts[i], ":"))
			continue
		}
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

// processSGRSubparams handles one colon-separated SGR parameter (ITU T.416
// form), such as "38:2::R:G:B", "38:5:N", or "4:3" (curly underline).
func (e *Emulator) processSGRSubparams(sub []string) {
	code, _ := strconv.Atoi(sub[0])
	if code != 38 && code != 48 {
		// Sub-parameters only refine the attribute (for example the
		// underline style); apply the base code alone.
		e.processSGR(strconv.Itoa(code))
		return
	}
	if len(sub) < 2 {
		return
	}
	mode, _ := strconv.Atoi(sub[1])
	var c color.Color
	switch {
	case mode == 5 && len(sub) >= 3:
		idx, _ := strconv.Atoi(sub[2])
		c = color256(idx)
	case mode == 2 && len(sub) >= 5:
		// "38:2:R:G:B" or "38:2:<colorspace>:R:G:B": RGB are the last three.
		n := len(sub)
		r, _ := strconv.Atoi(sub[n-3])
		g, _ := strconv.Atoi(sub[n-2])
		b, _ := strconv.Atoi(sub[n-1])
		c = RGB(uint8(r), uint8(g), uint8(b))
	default:
		return
	}
	if code == 38 {
		e.fg = c
	} else {
		e.bg = c
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

// cubeLevels are the xterm 6x6x6 color cube component values. They match
// Terminal256, so colors map to their exact palette entries.
var cubeLevels = [6]uint8{0, 95, 135, 175, 215, 255}

// color256 returns the color for 256-color mode. Out-of-range indexes are
// clamped to [0, 255].
func color256(idx int) color.RGBA {
	idx = max(0, min(idx, 255))
	if idx < 16 {
		// Standard colors
		if idx < 8 {
			return ansiColor(idx)
		}
		return ansiBrightColor(idx - 8)
	} else if idx < 232 {
		// 216 color cube (6x6x6)
		idx -= 16
		return color.RGBA{cubeLevels[idx/36], cubeLevels[(idx/6)%6], cubeLevels[idx%6], 255}
	} else {
		// Grayscale (24 shades)
		gray := uint8((idx-232)*10 + 8)
		return color.RGBA{gray, gray, gray, 255}
	}
}
