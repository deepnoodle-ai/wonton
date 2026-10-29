package tmux

import (
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/deepnoodle-ai/wonton/color"
	"github.com/deepnoodle-ai/wonton/runewidth"
	"github.com/deepnoodle-ai/wonton/terminal"
)

// Screen is a snapshot of a pane: what it shows and the modes its program
// has set. A Screen is never modified after a Terminal hands it out, so it is
// safe to read from any goroutine.
type Screen struct {
	// Width and Height are the pane's size in cells.
	Width, Height int

	// Cells holds the pane's rows, each exactly Width cells long. A wide
	// character's second column is a continuation cell, as in
	// terminal.Cell. While the pane is in copy mode, Cells shows the
	// history lines the mode has scrolled to.
	Cells [][]terminal.Cell

	// CursorX and CursorY are the cursor's cell, and CursorVisible whether
	// the program wants it shown.
	CursorX, CursorY int
	CursorVisible    bool

	// AltScreen is set while a full-screen program is using the alternate
	// screen.
	AltScreen bool

	// MouseReporting is set while the program has asked for mouse events,
	// and MouseSGR while it wants them in SGR (1006) encoding. A program
	// that asks for presses and releases (1000) gets no motion; MouseDrags
	// is set when it also wants motion while a button is held (1002), and
	// MouseMotion when it wants all motion (1003).
	MouseReporting bool
	MouseSGR       bool
	MouseDrags     bool
	MouseMotion    bool

	// InMode is set while the pane is in a mode such as copy mode, and
	// ScrollPosition is how many lines up from the bottom copy mode is
	// showing.
	InMode         bool
	ScrollPosition int
	HistorySize    int

	// PaneID is tmux's ID for the pane ("%3").
	PaneID string

	// Title is the title the program set with an escape sequence, Command
	// is the name of the foreground process, and Path its working
	// directory.
	Title   string
	Command string
	Path    string

	// Captured is when this snapshot was taken.
	Captured time.Time
}

// Cell returns the cell at (x, y), or a blank cell outside the screen.
func (s *Screen) Cell(x, y int) terminal.Cell {
	if s == nil || y < 0 || y >= len(s.Cells) || x < 0 || x >= len(s.Cells[y]) {
		return blankCell()
	}
	return s.Cells[y][x]
}

// Text returns the screen's text, one line per row, with trailing spaces
// removed.
func (s *Screen) Text() string {
	if s == nil {
		return ""
	}
	var b strings.Builder
	for y, row := range s.Cells {
		if y > 0 {
			b.WriteByte('\n')
		}
		var line strings.Builder
		for _, c := range row {
			if c.Continuation {
				continue
			}
			line.WriteRune(c.Char)
			line.WriteString(c.Trailing)
		}
		b.WriteString(strings.TrimRight(line.String(), " "))
	}
	return b.String()
}

func blankCell() terminal.Cell {
	return terminal.Cell{Char: ' ', Width: 1, Style: terminal.NewStyle()}
}

// ParseCapture turns the output of "capture-pane -p -e" into rows of cells,
// width by height. Text and SGR attributes are interpreted the way tmux
// writes them: style carries over from one line to the next, a tab runs to
// the next multiple of eight columns, and the cells past the end of a line
// are blank. OSC 8 hyperlinks set the cells' URL.
func ParseCapture(lines []string, width, height int) [][]terminal.Cell {
	rows := make([][]terminal.Cell, height)
	style := terminal.NewStyle()
	for y := range rows {
		row := make([]terminal.Cell, width)
		for x := range row {
			row[x] = blankCell()
		}
		rows[y] = row
		if y < len(lines) {
			style = parseLine(lines[y], row, style)
		}
	}
	return rows
}

// parseLine fills row from one captured line and returns the style in
// effect at its end.
func parseLine(line string, row []terminal.Cell, style terminal.Style) terminal.Style {
	x := 0
	text := func(s string) {
		for cluster, w := range runewidth.Graphemes(s) {
			if cluster == "\t" {
				// tmux keeps tabs in its grid and captures them as tabs:
				// blank cells up to the next tab stop.
				stop := min(len(row), (x/8+1)*8)
				for ; x < stop; x++ {
					row[x] = terminal.Cell{Char: ' ', Width: 1, Style: style}
				}
				continue
			}
			if w <= 0 {
				// A zero-width cluster on its own (a stray combining mark)
				// belongs to the cell before it.
				if x > 0 {
					prev := &row[x-1]
					if prev.Continuation && x > 1 {
						prev = &row[x-2]
					}
					prev.Trailing += cluster
				}
				continue
			}
			if x+w > len(row) {
				return
			}
			r, size := utf8.DecodeRuneInString(cluster)
			row[x] = terminal.Cell{Char: r, Trailing: cluster[size:], Style: style, Width: w}
			for i := 1; i < w; i++ {
				row[x+i] = terminal.Cell{Style: style, Continuation: true}
			}
			x += w
		}
	}

	for len(line) > 0 {
		esc := strings.IndexByte(line, 0x1b)
		if esc < 0 {
			text(line)
			break
		}
		if esc > 0 {
			text(line[:esc])
		}
		line = line[esc:]
		if len(line) < 2 {
			break
		}
		switch line[1] {
		case '[': // CSI: only SGR appears in a capture
			end := 2
			for end < len(line) && (line[end] < 0x40 || line[end] > 0x7e) {
				end++
			}
			if end >= len(line) {
				return style
			}
			if line[end] == 'm' {
				style = applySGR(style, line[2:end])
			}
			line = line[end+1:]
		case ']': // OSC, terminated by ST (ESC \) or BEL
			body, rest := cutOSC(line[2:])
			if params, ok := strings.CutPrefix(body, "8;"); ok {
				_, url, _ := strings.Cut(params, ";")
				style.URL = url
			}
			line = rest
		default:
			line = line[2:]
		}
	}
	return style
}

func cutOSC(s string) (body, rest string) {
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case 0x07:
			return s[:i], s[i+1:]
		case 0x1b:
			if i+1 < len(s) && s[i+1] == '\\' {
				return s[:i], s[i+2:]
			}
		}
	}
	return s, ""
}

// applySGR applies one SGR parameter list ("1;38;5;200") to style.
func applySGR(style terminal.Style, params string) terminal.Style {
	if params == "" {
		return terminal.NewStyle().WithURL(style.URL)
	}
	parts := strings.Split(params, ";")
	for i := 0; i < len(parts); i++ {
		p := parts[i]
		// Colon sub-parameters: "4:3" (curly underline), "38:2::r:g:b".
		if strings.Contains(p, ":") {
			sub := strings.Split(p, ":")
			switch sub[0] {
			case "4":
				style.Underline = len(sub) < 2 || sub[1] != "0"
			case "38", "48":
				if rgb, idx, ok := extendedColor(sub[1:]); ok {
					style = setColor(style, sub[0] == "38", rgb, idx)
				}
			}
			continue
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			continue
		}
		switch {
		case n == 0:
			style = terminal.NewStyle().WithURL(style.URL)
		case n == 1:
			style.Bold = true
		case n == 2:
			style.Dim = true
		case n == 3:
			style.Italic = true
		case n == 4 || n == 21:
			style.Underline = true
		case n == 5 || n == 6:
			style.Blink = true
		case n == 7:
			style.Reverse = true
		case n == 8:
			style.Hidden = true
		case n == 9:
			style.Strikethrough = true
		case n == 22:
			style.Bold, style.Dim = false, false
		case n == 23:
			style.Italic = false
		case n == 24:
			style.Underline = false
		case n == 25:
			style.Blink = false
		case n == 27:
			style.Reverse = false
		case n == 28:
			style.Hidden = false
		case n == 29:
			style.Strikethrough = false
		case n >= 30 && n <= 37:
			style = style.WithForeground(color.Color(n - 30))
		case n == 39:
			style = style.WithForeground(color.Default)
		case n >= 40 && n <= 47:
			style = style.WithBackground(color.Color(n - 40))
		case n == 49:
			style = style.WithBackground(color.Default)
		case n >= 90 && n <= 97:
			style = style.WithForeground(color.Color(n - 90 + 8))
		case n >= 100 && n <= 107:
			style = style.WithBackground(color.Color(n - 100 + 8))
		case n == 38 || n == 48:
			rgb, idx, used := extendedColorSemicolon(parts[i+1:])
			if used > 0 {
				if rgb != nil || idx >= 0 {
					style = setColor(style, n == 38, rgb, idx)
				}
				i += used
			}
		case n == 58:
			// Underline color: not representable; skip its arguments.
			_, _, used := extendedColorSemicolon(parts[i+1:])
			i += used
		}
	}
	return style
}

func setColor(style terminal.Style, fg bool, rgb *color.RGB, idx int) terminal.Style {
	switch {
	case fg && rgb != nil:
		return style.WithForeground(color.Default).WithFgRGB(*rgb)
	case fg:
		return style.WithForeground(color.Palette(uint8(idx)))
	case rgb != nil:
		return style.WithBackground(color.Default).WithBgRGB(*rgb)
	default:
		return style.WithBackground(color.Palette(uint8(idx)))
	}
}

// extendedColorSemicolon reads "5;N" or "2;R;G;B" following a 38 or 48 and
// reports how many parameters it consumed.
func extendedColorSemicolon(rest []string) (rgb *color.RGB, idx int, used int) {
	if len(rest) == 0 {
		return nil, -1, 0
	}
	switch rest[0] {
	case "5":
		if len(rest) < 2 {
			return nil, -1, len(rest)
		}
		n, err := strconv.Atoi(rest[1])
		if err != nil || n < 0 || n > 255 {
			return nil, -1, 2
		}
		return nil, n, 2
	case "2":
		if len(rest) < 4 {
			return nil, -1, len(rest)
		}
		c, ok := rgbFrom(rest[1:4])
		if !ok {
			return nil, -1, 4
		}
		return &c, -1, 4
	}
	return nil, -1, 1
}

// extendedColor reads the colon form: "5:N", "2::R:G:B" or "2:R:G:B".
func extendedColor(sub []string) (*color.RGB, int, bool) {
	if len(sub) == 0 {
		return nil, -1, false
	}
	switch sub[0] {
	case "5":
		if len(sub) < 2 {
			return nil, -1, false
		}
		n, err := strconv.Atoi(sub[1])
		if err != nil || n < 0 || n > 255 {
			return nil, -1, false
		}
		return nil, n, true
	case "2":
		vals := sub[1:]
		if len(vals) == 4 { // a color-space ID precedes R:G:B
			vals = vals[1:]
		}
		if len(vals) != 3 {
			return nil, -1, false
		}
		c, ok := rgbFrom(vals)
		return &c, -1, ok
	}
	return nil, -1, false
}

func rgbFrom(vals []string) (color.RGB, bool) {
	var c [3]uint8
	for i, v := range vals {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 || n > 255 {
			return color.RGB{}, false
		}
		c[i] = uint8(n)
	}
	return color.NewRGB(c[0], c[1], c[2]), true
}
