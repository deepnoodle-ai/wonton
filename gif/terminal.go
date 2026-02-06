package gif

import (
	"image/color"
)

// TerminalCell represents a single character cell in a terminal screen,
// including the character itself and its foreground and background colors.
type TerminalCell struct {
	Char rune        // The Unicode character displayed in this cell
	FG   color.Color // Foreground (text) color
	BG   color.Color // Background color
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
func NewTerminalScreen(cols, rows int) *TerminalScreen {
	ts := &TerminalScreen{
		Width:     cols,
		Height:    rows,
		DefaultFG: White,
		DefaultBG: Black,
	}
	ts.Cells = make([][]TerminalCell, rows)
	for y := 0; y < rows; y++ {
		ts.Cells[y] = make([]TerminalCell, cols)
		for x := 0; x < cols; x++ {
			ts.Cells[y][x] = TerminalCell{
				Char: ' ',
				FG:   ts.DefaultFG,
				BG:   ts.DefaultBG,
			}
		}
	}
	return ts
}

// Clear clears the entire screen, filling all cells with spaces using the
// default foreground and background colors. The cursor is reset to position (0,0).
func (ts *TerminalScreen) Clear() {
	for y := 0; y < ts.Height; y++ {
		for x := 0; x < ts.Width; x++ {
			ts.Cells[y][x] = TerminalCell{
				Char: ' ',
				FG:   ts.DefaultFG,
				BG:   ts.DefaultBG,
			}
		}
	}
	ts.CursorX = 0
	ts.CursorY = 0
}

// SetCell sets the character and colors for a cell at the specified position.
// If the position is out of bounds, the call is silently ignored. Nil colors
// are replaced with the default foreground or background colors.
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
	ts.Cells[y][x] = TerminalCell{Char: char, FG: fg, BG: bg}
}

// WriteString writes a string to the screen starting at the current cursor
// position. The cursor advances as characters are written. Newlines (\n) move
// the cursor to the start of the next line. Carriage returns (\r) move to the
// start of the current line. The screen scrolls up if text extends beyond the
// bottom row.
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
	for _, r := range s {
		if r == '\n' {
			ts.CursorX = 0
			ts.CursorY++
			if ts.CursorY >= ts.Height {
				ts.scrollUp()
				ts.CursorY = ts.Height - 1
			}
			continue
		}
		if r == '\r' {
			ts.CursorX = 0
			continue
		}
		if ts.CursorX >= ts.Width {
			ts.CursorX = 0
			ts.CursorY++
			if ts.CursorY >= ts.Height {
				ts.scrollUp()
				ts.CursorY = ts.Height - 1
			}
		}
		ts.Cells[ts.CursorY][ts.CursorX] = TerminalCell{Char: r, FG: fg, BG: bg}
		ts.CursorX++
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
	ts.Cells[ts.Height-1] = make([]TerminalCell, ts.Width)
	for x := 0; x < ts.Width; x++ {
		ts.Cells[ts.Height-1][x] = TerminalCell{
			Char: ' ',
			FG:   ts.DefaultFG,
			BG:   ts.DefaultBG,
		}
	}
}

// MoveCursor moves the cursor to the specified position, clamping coordinates
// to valid screen bounds. Negative values are clamped to 0, and values beyond
// the screen dimensions are clamped to the maximum valid position.
func (ts *TerminalScreen) MoveCursor(x, y int) {
	if x < 0 {
		x = 0
	}
	if x >= ts.Width {
		x = ts.Width - 1
	}
	if y < 0 {
		y = 0
	}
	if y >= ts.Height {
		y = ts.Height - 1
	}
	ts.CursorX = x
	ts.CursorY = y
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

		// Render each cell
		for y := 0; y < tr.screen.Height; y++ {
			for x := 0; x < tr.screen.Width; x++ {
				cell := tr.screen.Cells[y][x]
				pixelX := tr.padding + x*tr.cellWidth
				pixelY := tr.padding + y*tr.cellHeight

				// Fill cell background
				f.FillRect(pixelX, pixelY, tr.cellWidth, tr.cellHeight, cell.BG)

				// Draw character
				if cell.Char != ' ' && cell.Char != 0 {
					tr.drawChar(f, pixelX, pixelY, cell.Char, cell.FG)
				}
			}
		}
	}, delay)
}

// drawChar draws a character at the given pixel position.
func (tr *TerminalRenderer) drawChar(f *Frame, px, py int, char rune, fg color.Color) {
	if tr.font != nil {
		// Use TTF font
		tr.font.DrawChar(f.img, px, py, char, fg)
	} else {
		// Use bitmap font
		DrawBitmapChar(f, px, py, char, fg, tr.bitmapFont)
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
