package gif

import (
	"image"
	"image/color"
	"image/draw"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/deepnoodle-ai/wonton/runewidth"
	"golang.org/x/image/font/sfnt"
)

// TerminalCell represents a single character cell in a terminal screen,
// including the character itself and its foreground and background colors.
//
// A cell holds one grapheme cluster: Char is its first rune and Trailing
// holds the rest, such as a combining mark, a variation selector, or the
// other runes of an emoji ZWJ sequence. A wide cluster (CJK, most emoji)
// occupies two columns: the cell in the first has Width 2, and the cell in
// the second is a Continuation cell with no character of its own.
type TerminalCell struct {
	Char         rune        // First rune of the grapheme cluster displayed in this cell
	Trailing     string      // Remaining runes of a multi-rune cluster; usually empty
	Width        int         // Columns the cluster occupies: 1 or 2, or 0 for a continuation cell
	Continuation bool        // True if this cell is the second column of a wide cluster
	FG           color.Color // Foreground (text) color
	BG           color.Color // Background color
}

// Text returns the grapheme cluster displayed in the cell, or "" for a
// continuation cell.
func (c TerminalCell) Text() string {
	if c.Continuation || c.Char == 0 {
		return ""
	}
	return string(c.Char) + c.Trailing
}

// TerminalScreen represents a virtual terminal screen buffer with character
// cells arranged in rows and columns. It maintains the complete state needed
// to render terminal output, including all characters, their colors, and the
// cursor position.
//
// The screen uses (0,0) as the top-left position, with X increasing to the
// right and Y increasing downward.
type TerminalScreen struct {
	Width     int              // Screen width in columns (character cells)
	Height    int              // Screen height in rows (character cells)
	Cells     [][]TerminalCell // 2D array of cells [row][column]
	CursorX   int              // Current cursor column (0-indexed)
	CursorY   int              // Current cursor row (0-indexed)
	DefaultFG color.Color      // Default foreground color (typically white)
	DefaultBG color.Color      // Default background color (typically black)
}

// NewTerminalScreen creates a new terminal screen buffer with the specified
// dimensions. All cells are initialized with space characters and default colors.
// Negative dimensions are treated as 0.
func NewTerminalScreen(cols, rows int) *TerminalScreen {
	cols, rows = max(cols, 0), max(rows, 0)
	ts := &TerminalScreen{
		Width:     cols,
		Height:    rows,
		DefaultFG: White,
		DefaultBG: Black,
	}
	ts.Cells = make([][]TerminalCell, rows)
	for y := 0; y < rows; y++ {
		ts.Cells[y] = ts.blankRow()
	}
	return ts
}

// blankRow returns a row of spaces in the default colors.
func (ts *TerminalScreen) blankRow() []TerminalCell {
	row := make([]TerminalCell, ts.Width)
	for x := range row {
		row[x] = TerminalCell{Char: ' ', Width: 1, FG: ts.DefaultFG, BG: ts.DefaultBG}
	}
	return row
}

// Clear clears the entire screen, filling all cells with spaces using the
// default foreground and background colors. The cursor is reset to position (0,0).
func (ts *TerminalScreen) Clear() {
	for y := 0; y < ts.Height; y++ {
		ts.Cells[y] = ts.blankRow()
	}
	ts.CursorX = 0
	ts.CursorY = 0
}

// SetCell sets the character and colors for a cell at the specified position.
// If the position is out of bounds, the call is silently ignored. Nil colors
// are replaced with the default foreground or background colors.
//
// The character takes one column. Writing over either half of a wide
// cluster blanks its other half, as a terminal does.
func (ts *TerminalScreen) SetCell(x, y int, char rune, fg, bg color.Color) {
	if x < 0 || x >= ts.Width || y < 0 || y >= ts.Height {
		return
	}
	if fg == nil {
		fg = ts.DefaultFG
	}
	if bg == nil {
		bg = ts.DefaultBG
	}
	ts.breakWide(x, y)
	ts.Cells[y][x] = TerminalCell{Char: char, Width: 1, FG: fg, BG: bg}
}

// breakWide prepares the cell at (x, y) to be overwritten. If it is half of
// a wide cluster, the other half becomes a space in the same colors, so no
// orphaned half is left behind.
func (ts *TerminalScreen) breakWide(x, y int) {
	row := ts.Cells[y]
	if row[x].Continuation && x > 0 {
		row[x-1] = TerminalCell{Char: ' ', Width: 1, FG: row[x-1].FG, BG: row[x-1].BG}
	} else if row[x].Width == 2 && x+1 < ts.Width && row[x+1].Continuation {
		row[x+1] = TerminalCell{Char: ' ', Width: 1, FG: row[x+1].FG, BG: row[x+1].BG}
	}
}

// WriteString writes a string to the screen starting at the current cursor
// position. The cursor advances as characters are written. Newlines (\n) move
// the cursor to the start of the next line. Carriage returns (\r) move to the
// start of the current line. The screen scrolls up if text extends beyond the
// bottom row.
//
// Each grapheme cluster takes one cell, or two for a wide cluster, so the
// cursor advances by display width rather than by rune. Zero-width text with
// no cluster to join, such as a stray combining mark, takes no cell.
func (ts *TerminalScreen) WriteString(s string, fg, bg color.Color) {
	if ts.Height <= 0 || ts.Width <= 0 {
		return
	}
	if fg == nil {
		fg = ts.DefaultFG
	}
	if bg == nil {
		bg = ts.DefaultBG
	}
	for len(s) > 0 {
		switch s[0] {
		case '\n':
			ts.CursorX = 0
			ts.lineFeed()
			s = s[1:]
			continue
		case '\r':
			ts.CursorX = 0
			s = s[1:]
			continue
		}
		end := strings.IndexAny(s, "\r\n")
		if end < 0 {
			end = len(s)
		}
		ts.print(s[:end], fg, bg)
		s = s[end:]
	}
}

// print writes text containing no line breaks at the cursor, one grapheme
// cluster per cell, or two cells for a wide cluster. Text that continues the
// cluster before the cursor, such as a combining mark or variation selector
// written separately from its base, joins that cluster.
func (ts *TerminalScreen) print(s string, fg, bg color.Color) {
	if ts.Height <= 0 || ts.Width <= 0 {
		return
	}
	s = ts.extendPrevious(s)
	for cluster, width := range runewidth.Graphemes(s) {
		ts.putCluster(cluster, width, fg, bg)
	}
}

// extendPrevious joins the start of s to the cluster in the cell before the
// cursor when Unicode segmentation makes them one cluster, and returns the
// rest of s.
func (ts *TerminalScreen) extendPrevious(s string) string {
	if s == "" || ts.CursorX <= 0 {
		return s
	}
	x, y := min(ts.CursorX, ts.Width)-1, ts.CursorY
	if ts.Cells[y][x].Continuation && x > 0 {
		x--
	}
	prev := ts.Cells[y][x]
	prevText := prev.Text()
	if prevText == "" {
		return s
	}
	// ASCII printable text extends a cluster only after a Prepend rune
	// (GB9b), which is rare. Test one byte before joining all of s.
	if s[0] < utf8.RuneSelf && !joinsCluster(prevText, s[:1]) {
		return s
	}
	for cluster, width := range runewidth.Graphemes(prevText + s) {
		if len(cluster) <= len(prevText) {
			return s
		}
		// Rewrite the joined cluster in its base's place and colors. Its
		// width can change, as when VS16 gives a symbol emoji presentation.
		// If the wider cluster no longer fits before the right edge, it
		// keeps its old width, so it stays where it was printed instead of
		// wrapping to the next line.
		if x+width > ts.Width {
			width = max(prev.Width, 1)
		}
		ts.SetCell(x, y, ' ', prev.FG, prev.BG)
		ts.CursorX = x
		ts.putCluster(cluster, width, prev.FG, prev.BG)
		return s[len(cluster)-len(prevText):]
	}
	return s
}

// joinsCluster reports whether next continues the last cluster of prev.
func joinsCluster(prev, next string) bool {
	for cluster := range runewidth.Graphemes(prev + next) {
		return len(cluster) > len(prev)
	}
	return false
}

// putCluster writes one grapheme cluster at the cursor and advances the
// cursor by its width, wrapping to the next line first if it does not fit.
func (ts *TerminalScreen) putCluster(cluster string, width int, fg, bg color.Color) {
	switch {
	case width == 0:
		// Zero-width text with no cluster to join, such as a combining
		// mark at the start of a line, takes no cell.
		return
	case width > 2:
		// runewidth gives U+2E3A and U+2E3B widths 3 and 4, but terminals
		// draw them in one cell.
		width = 1
	}
	if width > ts.Width {
		// A wide cluster never fits a one-column screen.
		return
	}
	if ts.CursorX+width > ts.Width {
		ts.CursorX = 0
		ts.lineFeed()
	}
	x, y := ts.CursorX, ts.CursorY
	first, size := utf8.DecodeRuneInString(cluster)
	ts.breakWide(x, y)
	if width == 2 {
		ts.breakWide(x+1, y)
		ts.Cells[y][x+1] = TerminalCell{Continuation: true, FG: fg, BG: bg}
	}
	ts.Cells[y][x] = TerminalCell{Char: first, Trailing: cluster[size:], Width: width, FG: fg, BG: bg}
	ts.CursorX += width
}

// lineFeed moves the cursor down one line, scrolling at the bottom row.
func (ts *TerminalScreen) lineFeed() {
	ts.CursorY++
	if ts.CursorY >= ts.Height {
		ts.scrollUp()
		ts.CursorY = max(ts.Height-1, 0)
	}
}

// scrollUp scrolls the screen up by one line.
func (ts *TerminalScreen) scrollUp() {
	if ts.Height <= 0 {
		return
	}
	// Move all lines up
	for y := 1; y < ts.Height; y++ {
		ts.Cells[y-1] = ts.Cells[y]
	}
	// Clear the bottom line
	ts.Cells[ts.Height-1] = ts.blankRow()
}

// MoveCursor moves the cursor to the specified position, clamping coordinates
// to valid screen bounds. Negative values are clamped to 0, and values beyond
// the screen dimensions are clamped to the maximum valid position.
func (ts *TerminalScreen) MoveCursor(x, y int) {
	ts.CursorX = max(0, min(x, ts.Width-1))
	ts.CursorY = max(0, min(y, ts.Height-1))
}

// TerminalRenderer converts a TerminalScreen buffer into animated GIF frames.
// It handles text rendering using either TTF fonts (for high quality) or bitmap
// fonts (for speed and simplicity), and manages the GIF creation process.
//
// The renderer calculates pixel dimensions based on the terminal size and
// selected font, then renders each frame by drawing character cells with their
// foreground and background colors.
//
// Example:
//
//	screen := gif.NewTerminalScreen(80, 24)
//	screen.WriteString("Hello World", gif.White, gif.Black)
//
//	renderer := gif.NewTerminalRenderer(screen, 8)
//	renderer.RenderFrame(10) // Add frame with 100ms delay
//	renderer.Save("output.gif")
type TerminalRenderer struct {
	screen     *TerminalScreen
	gif        *GIF
	cellWidth  int
	cellHeight int
	padding    int
	font       *FontFace  // TTF font (nil = use bitmap)
	bitmapFont BitmapFont // Bitmap font to use if font is nil

	// Fallback fonts and their caches, used with a TTF font only.
	fallbacks []*FallbackFont
	glyphBuf  sfnt.Buffer
	choices   map[string]glyphChoice
	masks     map[maskKey]*image.Alpha
}

// RendererOptions configures how a TerminalRenderer converts terminal screens
// to GIF frames. It provides control over font selection, sizing, and layout.
//
// Use DefaultRendererOptions() to get sensible defaults, then customize:
//
//	opts := gif.DefaultRendererOptions()
//	opts.FontSize = 16
//	opts.Padding = 10
//	renderer := gif.NewTerminalRendererWithOptions(screen, opts)
type RendererOptions struct {
	Font       *FontFace  // Custom TTF/OTF font (nil = use default Inconsolata or bitmap)
	FontSize   float64    // Font size in points for default TTF (default: 14)
	UseBitmap  bool       // Force bitmap font instead of TTF (faster, lower quality)
	BitmapFont BitmapFont // Which bitmap font to use when UseBitmap=true (default: 8x16)
	Padding    int        // Padding in pixels around terminal content (default: 8)

	// FallbackFonts are tried in order for characters the TTF font has no
	// glyph for, such as emoji and CJK text. Nil uses DefaultFallbackFonts;
	// an empty, non-nil slice turns fallback off. Bitmap fonts ignore it.
	FallbackFonts []*FallbackFont
}

// DefaultRendererOptions returns sensible defaults for terminal rendering.
func DefaultRendererOptions() RendererOptions {
	return RendererOptions{
		FontSize:   14,
		UseBitmap:  false,
		BitmapFont: BitmapFont8x16,
		Padding:    8,
	}
}

// NewTerminalRenderer creates a renderer for the given terminal screen using
// the default TTF font (Inconsolata) for best rendering quality. The padding
// parameter specifies extra pixels around the terminal content.
//
// This is a convenience constructor that uses DefaultRendererOptions with the
// specified padding. For more control over font selection and rendering options,
// use NewTerminalRendererWithOptions.
func NewTerminalRenderer(screen *TerminalScreen, padding int) *TerminalRenderer {
	opts := DefaultRendererOptions()
	opts.Padding = padding
	return NewTerminalRendererWithOptions(screen, opts)
}

// NewTerminalRendererWithOptions creates a renderer with full control over
// rendering options. This allows customizing font selection (TTF vs bitmap),
// font size, and padding.
//
// The renderer automatically calculates pixel dimensions based on the terminal
// size and selected font metrics.
func NewTerminalRendererWithOptions(screen *TerminalScreen, opts RendererOptions) *TerminalRenderer {
	tr := &TerminalRenderer{
		screen:     screen,
		padding:    opts.Padding,
		bitmapFont: opts.BitmapFont,
		fallbacks:  opts.FallbackFonts,
		choices:    map[string]glyphChoice{},
		masks:      map[maskKey]*image.Alpha{},
	}

	// Determine font and cell dimensions
	if opts.Font != nil {
		// Use provided TTF font
		tr.font = opts.Font
		tr.cellWidth = opts.Font.CellWidth()
		tr.cellHeight = opts.Font.CellHeight()
	} else if opts.UseBitmap {
		// Use bitmap font
		tr.cellWidth = opts.BitmapFont.Width
		tr.cellHeight = opts.BitmapFont.Height
	} else {
		// Try to load default TTF font
		fontSize := opts.FontSize
		if fontSize <= 0 {
			fontSize = 14
		}
		font, err := LoadDefaultFont(fontSize)
		if err == nil {
			tr.font = font
			tr.cellWidth = font.CellWidth()
			tr.cellHeight = font.CellHeight()
		} else {
			// Fall back to bitmap
			tr.cellWidth = opts.BitmapFont.Width
			tr.cellHeight = opts.BitmapFont.Height
		}
	}
	if tr.font != nil && tr.fallbacks == nil {
		tr.fallbacks = DefaultFallbackFonts()
	}

	// Calculate image dimensions
	width := screen.Width*tr.cellWidth + opts.Padding*2
	height := screen.Height*tr.cellHeight + opts.Padding*2

	// Create a palette with common terminal colors
	palette := terminalPalette()

	tr.gif = NewWithPalette(width, height, palette)

	return tr
}

// terminalPalette returns a color palette suitable for terminal rendering.
// It uses the standard xterm 256-color palette which provides excellent
// coverage for anti-aliased text rendering.
func terminalPalette() Palette {
	return Terminal256()
}

// SetLoopCount sets how many times the GIF animation should loop.
// Use 0 for infinite looping (default), -1 to play once, or a positive
// number for a specific repeat count. Returns the renderer for method chaining.
func (tr *TerminalRenderer) SetLoopCount(count int) *TerminalRenderer {
	tr.gif.SetLoopCount(count)
	return tr
}

// RenderFrame captures the current terminal screen state and adds it as a frame
// to the GIF animation. The delay parameter specifies how long to display this
// frame, measured in hundredths of a second (e.g., 10 = 100ms, 50 = 500ms).
//
// Call this method multiple times as the terminal screen changes to create an
// animation showing the terminal session progression.
func (tr *TerminalRenderer) RenderFrame(delay int) {
	tr.gif.AddFrameWithDelay(func(f *Frame) {
		// Fill background
		f.Fill(tr.screen.DefaultBG)

		// Fill every cell background before drawing any glyph, so the
		// second column of a wide glyph is not painted over.
		for y := 0; y < tr.screen.Height; y++ {
			for x := 0; x < tr.screen.Width; x++ {
				pixelX := tr.padding + x*tr.cellWidth
				pixelY := tr.padding + y*tr.cellHeight
				f.FillRect(pixelX, pixelY, tr.cellWidth, tr.cellHeight, tr.screen.Cells[y][x].BG)
			}
		}

		// Draw characters. A continuation cell has no glyph of its own.
		for y := 0; y < tr.screen.Height; y++ {
			for x := 0; x < tr.screen.Width; x++ {
				cell := tr.screen.Cells[y][x]
				if cell.Continuation || cell.Char == 0 || (cell.Char == ' ' && cell.Trailing == "") {
					continue
				}
				pixelX := tr.padding + x*tr.cellWidth
				pixelY := tr.padding + y*tr.cellHeight
				tr.drawCell(f, pixelX, pixelY, cell)
			}
		}
	}, delay)
}

// drawCell draws a cell's grapheme cluster at the given pixel position.
//
// With a TTF font, a cluster the font cannot draw, or an emoji sequence,
// is drawn with the first fallback font that has a glyph for it (see
// chooseGlyph). Otherwise the font draws the cluster's first rune and its
// combining marks over it. The renderer does no mark positioning, so a mark
// drawn after its base would land in the next cell. Bitmap fonts draw only
// the first rune, since they draw a box for any glyph they lack.
func (tr *TerminalRenderer) drawCell(f *Frame, px, py int, cell TerminalCell) {
	if tr.font == nil {
		DrawBitmapChar(f, px, py, cell.Char, cell.FG, tr.bitmapFont)
		return
	}
	if c := tr.chooseGlyph(cell); c.font != nil {
		w := max(cell.Width, 1) * tr.cellWidth
		mask := tr.glyphMask(c, w)
		draw.DrawMask(f.img, image.Rect(px, py, px+w, py+tr.cellHeight),
			image.NewUniform(cell.FG), image.Point{}, mask, image.Point{}, draw.Over)
		return
	}
	tr.font.DrawChar(f.img, px, py, cell.Char, cell.FG)
	for _, r := range cell.Trailing {
		if unicode.In(r, unicode.Mn, unicode.Me) {
			tr.font.DrawChar(f.img, px, py, r, cell.FG)
		}
	}
}

// GIF returns the underlying GIF object being constructed. This allows access
// to the GIF for additional manipulation or encoding options.
func (tr *TerminalRenderer) GIF() *GIF {
	return tr.gif
}

// Save writes the rendered GIF animation to a file. This is a convenience
// method that calls Save on the underlying GIF object.
func (tr *TerminalRenderer) Save(filename string) error {
	return tr.gif.Save(filename)
}

// Bytes returns the GIF animation as a byte slice, suitable for writing to
// HTTP responses or other byte-oriented outputs. This is a convenience method
// that calls Bytes on the underlying GIF object.
func (tr *TerminalRenderer) Bytes() ([]byte, error) {
	return tr.gif.Bytes()
}
