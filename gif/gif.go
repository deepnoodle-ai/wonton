// Package gif provides utilities for creating animated GIF images with a
// clean, builder-style API. It includes comprehensive support for:
//   - Frame-by-frame GIF creation with drawing primitives
//   - Terminal emulation and ANSI escape sequence processing
//   - Converting terminal recordings (.cast files) to animated GIFs
//   - TTF and bitmap fonts for text rendering
//
// The package is designed to work seamlessly with the termsession package for
// converting terminal recordings into animated GIFs, making it ideal for
// creating documentation and tutorials.
//
// # Basic GIF Creation
//
// Create an animated GIF by building frames with drawing operations:
//
//	g := gif.New(100, 100)
//	for i := 0; i < 10; i++ {
//	    g.AddFrame(func(f *gif.Frame) {
//	        f.Fill(gif.White)
//	        f.FillCircle(50, 50+i*3, 10, gif.Red)
//	    })
//	}
//	g.Save("animation.gif")
//
// # Custom Palettes
//
// GIFs support up to 256 colors per frame. Create custom palettes:
//
//	palette := gif.Palette{gif.White, gif.Black, gif.RGB(255, 0, 0)}
//	g := gif.NewWithPalette(100, 100, palette)
//
// # Terminal Recording to GIF
//
// Convert asciinema recordings to animated GIFs:
//
//	opts := gif.DefaultCastOptions()
//	opts.FontSize = 14
//	opts.FPS = 10
//	g, err := gif.RenderCast("recording.cast", opts)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	g.Save("demo.gif")
//
// # Terminal Emulation
//
// Process raw terminal output with ANSI escape sequences:
//
//	emulator := gif.NewEmulator(80, 24)
//	emulator.ProcessOutput("\x1b[31mHello\x1b[0m World")
//
//	renderer := gif.NewTerminalRenderer(emulator.Screen(), 8)
//	renderer.RenderFrame(10)
//	renderer.Save("terminal.gif")
//
// # Drawing Primitives
//
// The Frame type provides various drawing operations:
//   - Fill, FillRect, FillCircle: Fill areas with color
//   - DrawLine, DrawRect, DrawCircle: Draw outlines
//   - SetPixel, SetPixelIndex: Set individual pixels
//
// All drawing operations handle bounds checking automatically.
package gif

import (
	"image"
	"image/color"
	"image/gif"
	"io"
	"os"
)

// Common colors for convenience.
var (
	Black       = color.RGBA{0, 0, 0, 255}
	White       = color.RGBA{255, 255, 255, 255}
	Red         = color.RGBA{255, 0, 0, 255}
	Green       = color.RGBA{0, 255, 0, 255}
	Blue        = color.RGBA{0, 0, 255, 255}
	Yellow      = color.RGBA{255, 255, 0, 255}
	Cyan        = color.RGBA{0, 255, 255, 255}
	Magenta     = color.RGBA{255, 0, 255, 255}
	Transparent = color.RGBA{0, 0, 0, 0}
)

// Palette is a slice of colors used for GIF frames.
// GIFs support up to 256 colors per frame.
type Palette []color.Color

// DefaultPalette provides a basic palette with common colors.
var DefaultPalette = Palette{
	White, Black, Red, Green, Blue, Yellow, Cyan, Magenta,
}

// GIF represents an animated GIF being constructed.
type GIF struct {
	width     int
	height    int
	palette   color.Palette
	images    []*image.Paletted
	delays    []int
	loopCount int
	disposal  []byte
}

// New creates a new GIF with the specified dimensions and the default palette.
// Dimensions must be positive; values less than 1 are clamped to 1.
func New(width, height int) *GIF {
	return NewWithPalette(width, height, DefaultPalette)
}

// NewWithPalette creates a new GIF with a custom palette.
// Dimensions must be positive; values less than 1 are clamped to 1.
// Palette must have 1-256 colors; empty palettes use DefaultPalette,
// and palettes exceeding 256 colors are truncated.
func NewWithPalette(width, height int, palette Palette) *GIF {
	// Clamp dimensions to minimum of 1
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}
	// Handle invalid palettes
	if len(palette) == 0 {
		palette = DefaultPalette
	} else if len(palette) > 256 {
		palette = palette[:256]
	}
	return &GIF{
		width:     width,
		height:    height,
		palette:   color.Palette(palette),
		loopCount: 0, // Loop forever by default
	}
}

// SetLoopCount sets the number of times the animation should loop.
// 0 means loop forever, -1 means no loop (play once).
func (g *GIF) SetLoopCount(count int) *GIF {
	g.loopCount = count
	return g
}

// Width returns the GIF width.
func (g *GIF) Width() int {
	return g.width
}

// Height returns the GIF height.
func (g *GIF) Height() int {
	return g.height
}

// FrameCount returns the number of frames added so far.
func (g *GIF) FrameCount() int {
	return len(g.images)
}

// Frame represents a single frame being drawn.
type Frame struct {
	img     *image.Paletted
	palette color.Palette
	width   int
	height  int
}

// Width returns the frame width.
func (f *Frame) Width() int {
	return f.width
}

// Height returns the frame height.
func (f *Frame) Height() int {
	return f.height
}

// SetPixel sets a pixel to the given color.
// The color must be in the palette or it will be matched to the nearest color.
func (f *Frame) SetPixel(x, y int, c color.Color) {
	if x < 0 || x >= f.width || y < 0 || y >= f.height {
		return
	}
	f.img.Set(x, y, c)
}

// SetPixelIndex sets a pixel using a palette index directly.
// This is faster than SetPixel when you know the index.
func (f *Frame) SetPixelIndex(x, y int, index uint8) {
	if x < 0 || x >= f.width || y < 0 || y >= f.height {
		return
	}
	if int(index) >= len(f.palette) {
		return
	}
	f.img.SetColorIndex(x, y, index)
}

// Fill fills the entire frame with a color.
func (f *Frame) Fill(c color.Color) {
	idx := uint8(f.palette.Index(c))
	for y := 0; y < f.height; y++ {
		for x := 0; x < f.width; x++ {
			f.img.SetColorIndex(x, y, idx)
		}
	}
}

// FillRect fills a rectangle with a color.
func (f *Frame) FillRect(x, y, w, h int, c color.Color) {
	idx := uint8(f.palette.Index(c))
	for dy := 0; dy < h; dy++ {
		for dx := 0; dx < w; dx++ {
			px, py := x+dx, y+dy
			if px >= 0 && px < f.width && py >= 0 && py < f.height {
				f.img.SetColorIndex(px, py, idx)
			}
		}
	}
}

// DrawLine draws a line from (x0, y0) to (x1, y1) using Bresenham's algorithm.
func (f *Frame) DrawLine(x0, y0, x1, y1 int, c color.Color) {
	idx := uint8(f.palette.Index(c))
	dx := abs(x1 - x0)
	dy := -abs(y1 - y0)
	sx := 1
	if x0 >= x1 {
		sx = -1
	}
	sy := 1
	if y0 >= y1 {
		sy = -1
	}
	err := dx + dy

	for {
		if x0 >= 0 && x0 < f.width && y0 >= 0 && y0 < f.height {
			f.img.SetColorIndex(x0, y0, idx)
		}
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

// DrawRect draws a rectangle outline.
func (f *Frame) DrawRect(x, y, w, h int, c color.Color) {
	f.DrawLine(x, y, x+w-1, y, c)         // Top
	f.DrawLine(x, y+h-1, x+w-1, y+h-1, c) // Bottom
	f.DrawLine(x, y, x, y+h-1, c)         // Left
	f.DrawLine(x+w-1, y, x+w-1, y+h-1, c) // Right
}

// DrawCircle draws a circle outline using the midpoint algorithm.
func (f *Frame) DrawCircle(cx, cy, r int, c color.Color) {
	idx := uint8(f.palette.Index(c))
	x := r
	y := 0
	err := 0

	setPixel := func(px, py int) {
		if px >= 0 && px < f.width && py >= 0 && py < f.height {
			f.img.SetColorIndex(px, py, idx)
		}
	}

	for x >= y {
		setPixel(cx+x, cy+y)
		setPixel(cx+y, cy+x)
		setPixel(cx-y, cy+x)
		setPixel(cx-x, cy+y)
		setPixel(cx-x, cy-y)
		setPixel(cx-y, cy-x)
		setPixel(cx+y, cy-x)
		setPixel(cx+x, cy-y)

		y++
		err += 1 + 2*y
		if 2*(err-x)+1 > 0 {
			x--
			err += 1 - 2*x
		}
	}
}

// FillCircle fills a circle.
func (f *Frame) FillCircle(cx, cy, r int, c color.Color) {
	idx := uint8(f.palette.Index(c))
	for y := -r; y <= r; y++ {
		for x := -r; x <= r; x++ {
			if x*x+y*y <= r*r {
				px, py := cx+x, cy+y
				if px >= 0 && px < f.width && py >= 0 && py < f.height {
					f.img.SetColorIndex(px, py, idx)
				}
			}
		}
	}
}

// Image returns the underlying paletted image for advanced manipulation.
func (f *Frame) Image() *image.Paletted {
	return f.img
}

// AddFrame adds a new frame with the default delay (100ms).
// The draw function is called to render the frame content.
func (g *GIF) AddFrame(draw func(*Frame)) *GIF {
	return g.AddFrameWithDelay(draw, 10) // 10 * 10ms = 100ms
}

// AddFrameWithDelay adds a new frame with a custom delay.
// Delay is in 100ths of a second (e.g., 10 = 100ms).
// Negative delays are clamped to 0.
func (g *GIF) AddFrameWithDelay(draw func(*Frame), delay int) *GIF {
	if delay < 0 {
		delay = 0
	}
	bounds := image.Rect(0, 0, g.width, g.height)
	img := image.NewPaletted(bounds, g.palette)

	frame := &Frame{
		img:     img,
		palette: g.palette,
		width:   g.width,
		height:  g.height,
	}

	if draw != nil {
		draw(frame)
	}

	g.images = append(g.images, img)
	g.delays = append(g.delays, delay)
	g.disposal = append(g.disposal, gif.DisposalBackground)
	return g
}

// AddImage adds an existing paletted image as a frame.
// The image is added directly without palette conversion; ensure the image
// uses a compatible palette or colors may not display correctly.
// Nil images are ignored. Negative delays are clamped to 0.
func (g *GIF) AddImage(img *image.Paletted, delay int) *GIF {
	if img == nil {
		return g
	}
	if delay < 0 {
		delay = 0
	}
	g.images = append(g.images, img)
	g.delays = append(g.delays, delay)
	g.disposal = append(g.disposal, gif.DisposalBackground)
	return g
}

// Encode writes the GIF to an io.Writer.
func (g *GIF) Encode(w io.Writer) error {
	anim := &gif.GIF{
		Image:     g.images,
		Delay:     g.delays,
		LoopCount: g.loopCount,
		Disposal:  g.disposal,
	}
	return gif.EncodeAll(w, anim)
}

// Save writes the GIF to a file.
func (g *GIF) Save(filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	return g.Encode(f)
}

// Bytes returns the GIF as a byte slice.
func (g *GIF) Bytes() ([]byte, error) {
	var buf bytesBuffer
	if err := g.Encode(&buf); err != nil {
		return nil, err
	}
	return buf.data, nil
}

// bytesBuffer is a simple buffer that implements io.Writer.
type bytesBuffer struct {
	data []byte
}

func (b *bytesBuffer) Write(p []byte) (int, error) {
	b.data = append(b.data, p...)
	return len(p), nil
}

// RGB creates an RGBA color from RGB values.
func RGB(r, g, b uint8) color.RGBA {
	return color.RGBA{r, g, b, 255}
}

// RGBA creates an RGBA color.
func RGBA(r, g, b, a uint8) color.RGBA {
	return color.RGBA{r, g, b, a}
}

// Grayscale creates a grayscale palette with n shades from black to white.
func Grayscale(n int) Palette {
	if n < 2 {
		n = 2
	}
	if n > 256 {
		n = 256
	}
	p := make(Palette, n)
	for i := 0; i < n; i++ {
		v := uint8(i * 255 / (n - 1))
		p[i] = color.RGBA{v, v, v, 255}
	}
	return p
}

// WebSafe creates the 216-color web-safe palette.
func WebSafe() Palette {
	p := make(Palette, 0, 216)
	for r := 0; r < 6; r++ {
		for g := 0; g < 6; g++ {
			for b := 0; b < 6; b++ {
				p = append(p, color.RGBA{
					uint8(r * 51),
					uint8(g * 51),
					uint8(b * 51),
					255,
				})
			}
		}
	}
	return p
}

// abs returns the absolute value of x.
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
