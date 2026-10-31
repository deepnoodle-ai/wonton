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
// # Raw Mode
//
// By default a Screen models a tty in cooked mode, where a bare "\n" also
// returns to column 0. To test output that goes to a tty in raw mode, such as
// output from tui.InlineApp, use WithRawMode. Then a bare "\n" keeps the
// cursor column, so output that forgets a "\r" fails the test:
//
//	screen := termtest.NewScreen(80, 24, termtest.WithRawMode(true))
//	screen.Write([]byte("ab\ncd"))  // "cd" starts at column 2 of row 1
//
// # ANSI Support
//
// The package supports common ANSI escape sequences:
//   - Cursor movement (up, down, left, right, position)
//   - Text styling (bold, italic, underline, colors)
//   - Screen clearing and line editing, with background color erase: erased
//     and inserted cells take the current background color, as in xterm
//   - Scrolling and line insertion/deletion, inside a scroll region
//     (CSI top ; bottom r) and with origin mode (CSI ? 6 h)
//   - Scrollback: rows that scroll off the top (see Screen.Scrollback)
//   - The alternate screen (?1049, ?1047, ?47); reads show the visible screen
//   - C0 controls (CR, LF, BS, ...) inside a CSI sequence, which run as
//     they do in xterm
//   - 256-color and 24-bit RGB colors
//
// Screen.Resize changes the size between two writes, as a user resizes the
// terminal window. It reflows soft-wrapped text the way ghostty does, the
// scrollback included. The alternate screen is truncated at the new edge
// instead, as in ghostty.
//
// Unicode and wide characters (CJK, emoji) are handled correctly with proper
// width calculation.
package termtest

import (
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/deepnoodle-ai/wonton/runewidth"
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
	Value   uint8 // For ColorBasic (0-15) and Color256 (0-255)
	R, G, B uint8 // For ColorRGB (24-bit true color)
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
//
// For single-rune glyphs, Char holds the entire cell content and Trailing is
// empty. For multi-rune grapheme clusters, Char holds the first rune and
// Trailing holds the remaining bytes so Row/Text can reconstruct the full
// cluster from a single lead cell.
type Cell struct {
	Char         rune   // The Unicode character (or 0 for continuation cells)
	Trailing     string // Remaining runes of a multi-rune grapheme cluster; usually empty
	Width        int    // Display width: 1 for normal, 2 for wide characters, 0 for zero-width/continuation
	Style        Style  // Text styling applied to this cell
	Continuation bool   // True if this cell is the continuation half of a wide glyph
}

// Screen represents a virtual terminal screen buffer that processes ANSI sequences.
// It maintains a 2D grid of cells, cursor position, current text style, and handles
// common terminal operations like scrolling, clearing, and cursor movement.
//
// Screen implements io.Writer, so it can be used anywhere an io.Writer is expected.
// Written bytes are interpreted as ANSI-formatted terminal output.
//
// A Screen is safe for concurrent use. A program under test can write to it
// from its own goroutines while the test reads it. Each method call sees the
// screen between two writes, never in the middle of one.
type Screen struct {
	mu sync.Mutex // Guards every field; each exported method holds it

	width  int // Screen width in columns
	height int // Screen height in rows
	// Rows move by swapping whole row slices, and new rows are freshly
	// allocated. Never copy cells between rows in place; see rowInfo.
	cells       [][]Cell // 2D grid of character cells [y][x]
	cursorX     int      // Current cursor column (0-based)
	cursorY     int      // Current cursor row (0-based)
	style       Style    // Current text style for new writes
	savedX      int      // Saved cursor X position (ESC 7/8, CSI s/u)
	savedY      int      // Saved cursor Y position (ESC 7/8, CSI s/u)
	savedStyle  Style    // Saved style (ESC 7/8)
	savedWrap   bool     // Saved pending wrap (ESC 7/8, ?1049)
	alt         altState // Hidden grid and state for the alternate screen
	wrapPending bool     // Deferred wrap: last column written, wrap on next glyph
	pending     []byte   // Incomplete escape sequence or UTF-8 rune from a previous Write
	skip        skipMode // Non-zero while inside an OSC/DCS/SOS/PM/APC payload, which is discarded
	rawMode     bool     // A bare LF keeps the column, as with the tty in raw mode; see WithRawMode

	// Soft wraps and printed cells by row, for Resize (resize.go). It is
	// keyed by each row's first cell, so it follows a row only if every row
	// move swaps whole row slices in cells and every new row is freshly
	// allocated. Never copy cells between rows in place (for example
	// copy(cells[y], cells[y+1])): the row's wrap flag and printed cells
	// would stay behind on the wrong row.
	rowInfo map[*Cell]*rowMeta

	// Scroll region (DECSTBM) and origin mode (DECOM); see region.go.
	top, bottom int       // Scroll region rows, 0-based and inclusive; the whole screen when unset
	originMode  bool      // Cursor rows count from the region top (CSI ? 6 h)
	savedOrigin bool      // Origin mode saved by ESC 7
	scrollback  []gridRow // Primary screen rows that scrolled off the top, oldest first; see Scrollback
}

type skipMode uint8

const (
	skipNone skipMode = iota
	skipST            // skipping DCS/SOS/PM/APC payload until ST (ESC \)
	skipOSC           // skipping OSC payload until BEL or ST (ESC \)
)

// NewScreen creates a new virtual terminal screen with the specified dimensions.
// The screen is initialized with empty cells (spaces) and the cursor at position (0, 0).
// Options such as WithRawMode change how the screen interprets output.
//
// Example:
//
//	screen := termtest.NewScreen(80, 24)  // Standard terminal size
//	screen.Write([]byte("Hello, World!"))
func NewScreen(width, height int, opts ...ScreenOption) *Screen {
	// A screen needs at least one cell for the cursor to stand on.
	width, height = max(width, 1), max(height, 1)
	s := &Screen{
		width:  width,
		height: height,
		cells:  make([][]Cell, height),
		bottom: height - 1,
	}
	for y := 0; y < height; y++ {
		s.cells[y] = make([]Cell, width)
		for x := 0; x < width; x++ {
			s.cells[y][x] = blankCell()
		}
	}
	s.applyOptions(opts)
	return s
}

func blankCell() Cell {
	return Cell{Char: ' ', Width: 1}
}

func continuationCell(style Style) Cell {
	return Cell{Char: 0, Width: 0, Style: style, Continuation: true}
}

func cellGlyph(cell Cell) string {
	if cell.Continuation {
		return ""
	}
	if cell.Char == 0 {
		return " "
	}
	if cell.Trailing == "" {
		return string(cell.Char)
	}
	return string(cell.Char) + cell.Trailing
}

func writeCellGlyph(w *strings.Builder, cell Cell) {
	w.WriteString(cellGlyph(cell))
}

// Size returns the screen dimensions.
func (s *Screen) Size() (width, height int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.width, s.height
}

// Cursor returns the current cursor position.
func (s *Screen) Cursor() (x, y int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cursorX, s.cursorY
}

// SetCursor moves the cursor to the specified position.
func (s *Screen) SetCursor(x, y int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.wrapPending = false
	s.cursorX = clamp(x, 0, s.width-1)
	s.cursorY = clamp(y, 0, s.height-1)
}

// Cell returns the cell at position (x, y).
// Returns a blank cell (space with width 1) if the position is out of bounds.
// Coordinates are 0-based, with (0, 0) at the top-left corner.
func (s *Screen) Cell(x, y int) Cell {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cell(x, y)
}

func (s *Screen) cell(x, y int) Cell {
	if x < 0 || x >= s.width || y < 0 || y >= s.height {
		return blankCell()
	}
	return s.cells[y][x]
}

// CellGlyph returns the full glyph at position (x, y).
// Lead cells return the complete grapheme cluster, continuation cells return
// an empty string, and blank cells return a single space.
func (s *Screen) CellGlyph(x, y int) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return cellGlyph(s.cell(x, y))
}

// SetCell sets the character and style at position (x, y).
// For wide characters (width 2), this also marks the next cell as a continuation.
// Does nothing if the position is out of bounds.
func (s *Screen) SetCell(x, y int, char rune, style Style) {
	s.mu.Lock()
	defer s.mu.Unlock()
	charWidth := runewidth.RuneWidth(char)
	s.setCellGlyph(x, y, char, "", charWidth, style)
}

func (s *Screen) setCellGlyph(x, y int, char rune, trailing string, charWidth int, style Style) {
	if x < 0 || x >= s.width || y < 0 || y >= s.height {
		return
	}

	// Clear artifacts if we're overwriting the lead or continuation half of
	// an existing wide glyph.
	oldCell := s.cells[y][x]
	if oldCell.Width == 2 && x+1 < s.width {
		s.cells[y][x+1] = blankCell()
	} else if oldCell.Continuation && x > 0 {
		s.cells[y][x-1] = blankCell()
	}

	s.cells[y][x] = Cell{
		Char:         char,
		Trailing:     trailing,
		Width:        charWidth,
		Style:        style,
		Continuation: false,
	}

	// For wide characters, mark the next cell as continuation
	if charWidth == 2 && x+1 < s.width {
		nextCell := s.cells[y][x+1]
		if nextCell.Width == 2 && x+2 < s.width {
			s.cells[y][x+2] = blankCell()
		}
		s.cells[y][x+1] = continuationCell(style)
	}
}

func (s *Screen) writeCluster(cluster string, clusterWidth int) {
	switch cluster {
	case "":
		return
	case "\r\n", "\n":
		s.wrapPending = false
		if cluster == "\r\n" || !s.rawMode {
			s.cursorX = 0
		}
		s.lineFeed()
		return
	case "\r":
		s.wrapPending = false
		s.cursorX = 0
		return
	case "\t":
		// A deferred wrap means the cursor is at the last column, so the
		// tab does nothing and the wrap stays pending, as in ghostty.
		if !s.wrapPending {
			nextTab := ((s.cursorX / 8) + 1) * 8
			s.cursorX = min(nextTab, s.width-1)
		}
		return
	}

	// Remaining C0/C1 control characters occupy no cells and must never be
	// written into the grid.
	if first, _ := utf8.DecodeRuneInString(cluster); first < 0x20 || (first >= 0x7f && first < 0xa0) {
		switch first {
		case '\b': // Backspace moves the cursor left, stopping at the margin
			s.wrapPending = false
			if s.cursorX > 0 {
				s.cursorX--
			}
		case '\v', '\f': // Vertical tab and form feed move down like line feed
			s.wrapPending = false
			s.lineFeed()
		}
		// All other controls (BEL, NUL, ...) are ignored
		return
	}

	switch {
	case clusterWidth <= 0:
		// Zero-width text with no cluster to join, such as a combining
		// mark at the start of a line or a zero-width space, takes no cell.
		return
	case clusterWidth > 2:
		clusterWidth = 1
	}
	if clusterWidth > s.width {
		// A wide cluster never fits a one-column screen.
		return
	}

	// Wrap if needed (deferred from filling the last column, or because the
	// cluster doesn't fit on the current line)
	if s.wrapPending || s.cursorX+clusterWidth > s.width {
		s.markWrapped(s.cursorY, !s.wrapPending && s.cursorX < s.width)
		s.wrapPending = false
		s.cursorX = 0
		s.lineFeed()
		s.markContinued(s.cursorY)
	}

	first, firstSize := utf8.DecodeRuneInString(cluster)
	s.setCellGlyph(s.cursorX, s.cursorY, first, cluster[firstSize:], clusterWidth, s.style)
	s.markPrinted(s.cursorY, s.cursorX, clusterWidth)
	s.cursorX += clusterWidth

	// Deferred wrap: after filling the last column the cursor stays on it
	// until the next printable cluster arrives (standard terminal behavior).
	if s.cursorX >= s.width {
		s.cursorX = s.width - 1
		s.wrapPending = true
	}
}

// extendPrevious joins the start of str to the cluster in the cell before the
// cursor when Unicode segmentation makes them one cluster, and returns the
// rest of str.
func (s *Screen) extendPrevious(str string) string {
	// ASCII never extends a cluster: only LF does, after CR.
	if str == "" || str[0] < utf8.RuneSelf || s.width <= 0 || s.height <= 0 {
		return str
	}
	var x int
	if s.wrapPending {
		x = s.cursorX
	} else {
		if s.cursorX <= 0 {
			return str
		}
		x = s.cursorX - 1
	}
	y := s.cursorY
	if s.cells[y][x].Continuation && x > 0 {
		x--
	}
	prev := s.cells[y][x]
	if prev.Continuation || prev.Char == 0 {
		return str
	}
	prevText := cellGlyph(prev)
	for cluster, width := range runewidth.Graphemes(prevText + str) {
		if len(cluster) <= len(prevText) {
			return str
		}
		switch {
		case width <= 0:
			width = max(prev.Width, 1)
		case width > 2:
			width = 1
		}
		if x+width > s.width {
			width = max(prev.Width, 1)
		}
		first, firstSize := utf8.DecodeRuneInString(cluster)
		s.setCellGlyph(x, y, first, cluster[firstSize:], width, prev.Style)
		s.markPrinted(y, x, width)
		s.cursorX = x + width
		if s.cursorX >= s.width {
			s.cursorX = s.width - 1
			s.wrapPending = true
		} else {
			s.wrapPending = false
		}
		return str[len(cluster)-len(prevText):]
	}
	return str
}

// lineFeed moves the cursor down one row, scrolling the scroll region at
// its bottom margin.
func (s *Screen) lineFeed() {
	s.index()
}

// WriteRune writes a single rune at the cursor position with the current style.
// The cursor advances by the character's display width (1 for normal, 2 for wide).
// Special characters are handled:
//   - '\n' moves to the next line (and keeps the column in raw mode; see WithRawMode)
//   - '\r' returns to column 0
//   - '\t' advances to the next tab stop (every 8 columns)
//
// The screen scrolls up automatically when writing past the bottom.
func (s *Screen) WriteRune(r rune) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.writeString(string(r))
}

// WriteString writes a string at the cursor position.
func (s *Screen) WriteString(str string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.writeString(str)
}

func (s *Screen) writeString(str string) {
	if s.width <= 0 || s.height <= 0 {
		return
	}
	str = s.extendPrevious(str)
	for cluster, clusterWidth := range runewidth.Graphemes(str) {
		s.writeCluster(cluster, clusterWidth)
	}
}

// maxPendingBytes bounds how many bytes of an incomplete escape sequence are
// buffered between Write calls before being dropped.
const maxPendingBytes = 1 << 20

// Write implements io.Writer by parsing ANSI escape sequences and updating the screen.
// It processes cursor movement, text styling, screen clearing, and other terminal
// control sequences. Plain text is written to the screen at the cursor position.
//
// Supported ANSI sequences include:
//   - CSI sequences (ESC[...): cursor movement, SGR, erase, insert/delete
//   - OSC/DCS/SOS/PM/APC sequences: parsed and ignored
//   - Simple escapes: save/restore cursor, reset, scroll
//
// An escape sequence or multi-byte UTF-8 rune split across Write calls is
// buffered and processed once the remaining bytes arrive, so chunk boundaries
// chosen by the writer never corrupt the screen.
//
// Returns the number of bytes consumed (always len(p)) and no error.
func (s *Screen) Write(p []byte) (n int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data := string(p)
	if len(s.pending) > 0 {
		data = string(s.pending) + data
		s.pending = s.pending[:0]
	}
	consumed := s.processANSI(data)
	if consumed < len(data) && len(data)-consumed <= maxPendingBytes {
		s.pending = append(s.pending, data[consumed:]...)
	}
	return len(p), nil
}

// Clear clears the entire screen and moves the cursor to the top-left.
// Cleared cells have the default style. (CSI 2 J, which a program sends,
// fills them with the current background color instead.)
func (s *Screen) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clear()
}

func (s *Screen) clear() {
	s.fillCells(blankCell())
	s.cursorX = 0
	s.cursorY = 0
	s.wrapPending = false
}

// clearCells erases every cell without moving the cursor (CSI 2J).
func (s *Screen) clearCells() {
	s.fillCells(s.eraseCell())
}

// fillCells sets every cell to cell and drops the row metadata.
func (s *Screen) fillCells(cell Cell) {
	s.rowInfo = nil
	for y := 0; y < s.height; y++ {
		for x := 0; x < s.width; x++ {
			s.cells[y][x] = cell
		}
	}
}

// Reset restores the screen to its initial state: all cells blank, cursor at
// the top-left, default style, no saved cursor state, no scroll region, and
// empty scrollback. This is equivalent to a terminal hard reset (ESC c / RIS).
func (s *Screen) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reset()
}

func (s *Screen) reset() {
	s.resetAltScreen()
	s.style = Style{}
	s.clear()
	s.savedX = 0
	s.savedY = 0
	s.savedStyle = Style{}
	s.savedWrap = false
	s.pending = nil
	s.skip = skipNone
	s.resetScrollRegion()
	s.originMode = false
	s.savedOrigin = false
	s.scrollback = nil
}

// ClearLine clears the current line. Like the other Clear methods, it
// leaves cells with the default style; the erase sequences a program sends
// (CSI K, CSI J) use the current background color instead.
func (s *Screen) ClearLine() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.eraseLine(blankCell())
}

// ClearToEndOfLine clears from cursor to end of line.
func (s *Screen) ClearToEndOfLine() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.eraseToEndOfLine(blankCell())
}

// ClearToStartOfLine clears from start of line to cursor.
func (s *Screen) ClearToStartOfLine() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.eraseToStartOfLine(blankCell())
}

// ClearToEndOfScreen clears from cursor to end of screen.
func (s *Screen) ClearToEndOfScreen() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.eraseToEndOfScreen(blankCell())
}

// ClearToStartOfScreen clears from start of screen to cursor.
func (s *Screen) ClearToStartOfScreen() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.eraseToStartOfScreen(blankCell())
}

// eraseLine sets every cell of the cursor row to cell (CSI 2 K).
func (s *Screen) eraseLine(cell Cell) {
	s.markErased(s.cursorY, 0, s.width, true)
	for x := 0; x < s.width; x++ {
		s.cells[s.cursorY][x] = cell
	}
}

// eraseToEndOfLine sets the cells from the cursor to the end of the row to
// cell (CSI K).
func (s *Screen) eraseToEndOfLine(cell Cell) {
	// If the cursor sits on the continuation half of a wide glyph, clear the
	// lead cell too so no dangling half remains (xterm behavior).
	if s.cursorX > 0 && s.cursorX < s.width && s.cells[s.cursorY][s.cursorX].Continuation {
		s.cells[s.cursorY][s.cursorX-1] = cell
		s.markErased(s.cursorY, s.cursorX-1, s.cursorX, false)
	}
	s.markErased(s.cursorY, s.cursorX, s.width, true)
	for x := s.cursorX; x < s.width; x++ {
		s.cells[s.cursorY][x] = cell
	}
}

// eraseToStartOfLine sets the cells from the start of the row through the
// cursor to cell (CSI 1 K).
func (s *Screen) eraseToStartOfLine(cell Cell) {
	// If the cursor sits on the lead half of a wide glyph, clear its
	// continuation cell too so no dangling half remains (xterm behavior).
	if s.cursorX >= 0 && s.cursorX < s.width-1 && s.cells[s.cursorY][s.cursorX].Width == 2 {
		s.cells[s.cursorY][s.cursorX+1] = cell
		s.markErased(s.cursorY, s.cursorX+1, s.cursorX+2, false)
	}
	s.markErased(s.cursorY, 0, s.cursorX+1, false)
	for x := 0; x <= s.cursorX && x < s.width; x++ {
		s.cells[s.cursorY][x] = cell
	}
}

// eraseToEndOfScreen sets the cells from the cursor to the end of the
// screen to cell (CSI J).
func (s *Screen) eraseToEndOfScreen(cell Cell) {
	s.eraseToEndOfLine(cell)
	for y := s.cursorY + 1; y < s.height; y++ {
		s.markErased(y, 0, s.width, true)
		for x := 0; x < s.width; x++ {
			s.cells[y][x] = cell
		}
	}
}

// eraseToStartOfScreen sets the cells from the start of the screen through
// the cursor to cell (CSI 1 J).
func (s *Screen) eraseToStartOfScreen(cell Cell) {
	for y := 0; y < s.cursorY; y++ {
		s.markErased(y, 0, s.width, true)
		for x := 0; x < s.width; x++ {
			s.cells[y][x] = cell
		}
	}
	s.eraseToStartOfLine(cell)
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
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.text()
}

func (s *Screen) text() string {
	var lines []string
	for y := 0; y < s.height; y++ {
		var line strings.Builder
		lastNonSpace := -1
		for x := 0; x < s.width; x++ {
			cell := s.cells[y][x]
			if cell.Continuation {
				continue // Skip continuation cells
			}
			writeCellGlyph(&line, cell)
			if cell.Char != ' ' && cell.Char != 0 || cell.Trailing != "" {
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
	s.mu.Lock()
	defer s.mu.Unlock()
	if y < 0 || y >= s.height {
		return ""
	}
	return rowText(s.cells[y])
}

// Contains checks if the screen contains the given text anywhere in its content.
// The search includes the full Text() output, so it can match across line boundaries.
func (s *Screen) Contains(text string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return strings.Contains(s.text(), text)
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
