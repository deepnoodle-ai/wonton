package tui

import "image"

// fillView fills available space with a character
type fillView struct {
	char  rune
	style Style
}

// Fill creates a view that fills its available space with the given character.
func Fill(char rune) *fillView {
	return &fillView{
		char:  char,
		style: NewStyle(),
	}
}

// Fg sets the foreground color.
func (f *fillView) Fg(c Color) *fillView {
	f.style = f.style.WithForeground(c)
	return f
}

// FgRGB sets the foreground color using RGB values.
func (f *fillView) FgRGB(r, g, b uint8) *fillView {
	f.style = f.style.WithFgRGB(RGB{R: r, G: g, B: b})
	return f
}

// Bg sets the background color.
func (f *fillView) Bg(c Color) *fillView {
	f.style = f.style.WithBackground(c)
	return f
}

// BgRGB sets the background color using RGB values.
func (f *fillView) BgRGB(r, g, b uint8) *fillView {
	f.style = f.style.WithBgRGB(RGB{R: r, G: g, B: b})
	return f
}

// Style sets the complete style.
func (f *fillView) Style(s Style) *fillView {
	f.style = s
	return f
}

func (f *fillView) render(frame RenderFrame, bounds image.Rectangle) {
	if bounds.Empty() {
		return
	}
	subFrame := frame.SubFrame(bounds)
	width, height := subFrame.Size()
	subFrame.FillStyled(0, 0, width, height, f.char, f.style)
}

func (f *fillView) size(maxWidth, maxHeight int) (int, int) {
	// Fill expands to fill available space
	return maxWidth, maxHeight
}

// Fill is flexible - it expands to fill available space
func (f *fillView) flex() int {
	return 1
}
