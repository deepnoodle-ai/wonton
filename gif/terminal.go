package gif

import (
	"image/color"
)

// TerminalCell represents a single cell in the terminal screen.
type TerminalCell struct {
	Char rune
	FG   color.Color
	BG   color.Color
}

// TerminalScreen represents a virtual terminal screen buffer.
type TerminalScreen struct {
	Width  int
	Height int
	Cells  [][]TerminalCell
	CursorX int
	CursorY int
	DefaultFG color.Color
	DefaultBG color.Color
}

// NewTerminalScreen creates a new terminal screen with the given dimensions.
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

// Clear clears the screen with the default background color.
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

// SetCell sets a cell at the given position.
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

// WriteString writes a string to the screen at the current cursor position.
func (ts *TerminalScreen) WriteString(s string, fg, bg color.Color) {
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

// MoveCursor moves the cursor to the given position.
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

// TerminalRenderer renders a TerminalScreen to GIF frames.
type TerminalRenderer struct {
	screen     *TerminalScreen
	gif        *GIF
	cellWidth  int
	cellHeight int
	padding    int
	font       *FontFace    // TTF font (nil = use bitmap)
	bitmapFont BitmapFont   // Bitmap font to use if font is nil
}

// RendererOptions configures terminal rendering.
type RendererOptions struct {
	Font       *FontFace  // TTF font (nil = use default TTF or bitmap fallback)
	FontSize   float64    // Font size for default TTF (default: 14)
	UseBitmap  bool       // Force bitmap font instead of TTF
	BitmapFont BitmapFont // Which bitmap font to use (default: 8x16)
	Padding    int        // Padding around terminal in pixels
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

// NewTerminalRenderer creates a renderer for the given terminal screen.
// The padding parameter adds extra pixels around the terminal content.
// This constructor uses the default TTF font for best quality.
func NewTerminalRenderer(screen *TerminalScreen, padding int) *TerminalRenderer {
	opts := DefaultRendererOptions()
	opts.Padding = padding
	return NewTerminalRendererWithOptions(screen, opts)
}

// NewTerminalRendererWithOptions creates a renderer with custom options.
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
func terminalPalette() Palette {
	return Palette{
		// Basic 16 ANSI colors
		RGB(0, 0, 0),       // 0: Black
		RGB(170, 0, 0),     // 1: Red
		RGB(0, 170, 0),     // 2: Green
		RGB(170, 85, 0),    // 3: Yellow/Brown
		RGB(0, 0, 170),     // 4: Blue
		RGB(170, 0, 170),   // 5: Magenta
		RGB(0, 170, 170),   // 6: Cyan
		RGB(170, 170, 170), // 7: White (gray)
		RGB(85, 85, 85),    // 8: Bright Black (dark gray)
		RGB(255, 85, 85),   // 9: Bright Red
		RGB(85, 255, 85),   // 10: Bright Green
		RGB(255, 255, 85),  // 11: Bright Yellow
		RGB(85, 85, 255),   // 12: Bright Blue
		RGB(255, 85, 255),  // 13: Bright Magenta
		RGB(85, 255, 255),  // 14: Bright Cyan
		RGB(255, 255, 255), // 15: Bright White
		// Extended grayscale for smoother rendering
		RGB(28, 28, 28),
		RGB(48, 48, 48),
		RGB(68, 68, 68),
		RGB(88, 88, 88),
		RGB(108, 108, 108),
		RGB(128, 128, 128),
		RGB(148, 148, 148),
		RGB(168, 168, 168),
		RGB(188, 188, 188),
		RGB(208, 208, 208),
		RGB(228, 228, 228),
	}
}

// SetLoopCount sets the GIF loop count.
func (tr *TerminalRenderer) SetLoopCount(count int) *TerminalRenderer {
	tr.gif.SetLoopCount(count)
	return tr
}

// RenderFrame renders the current terminal screen state as a GIF frame.
// Delay is in 100ths of a second (e.g., 10 = 100ms).
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

// GIF returns the underlying GIF being constructed.
func (tr *TerminalRenderer) GIF() *GIF {
	return tr.gif
}

// Save writes the GIF to a file.
func (tr *TerminalRenderer) Save(filename string) error {
	return tr.gif.Save(filename)
}

// Bytes returns the GIF as a byte slice.
func (tr *TerminalRenderer) Bytes() ([]byte, error) {
	return tr.gif.Bytes()
}
