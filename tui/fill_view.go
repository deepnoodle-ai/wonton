package tui

// FillView fills available space with a character
type FillView struct {
	char  rune
	style Style
}

// Fill creates a view that fills its available space with the given character.
func Fill(char rune) *FillView {
	return &FillView{
		char:  char,
		style: NewStyle(),
	}
}

// Fg sets the foreground color.
func (f *FillView) Fg(c Color) *FillView {
	f.style = f.style.WithForeground(c)
	return f
}

// FgRGB sets the foreground color using RGB values.
func (f *FillView) FgRGB(r, g, b uint8) *FillView {
	f.style = f.style.WithFgRGB(RGB{R: r, G: g, B: b})
	return f
}

// Bg sets the background color.
func (f *FillView) Bg(c Color) *FillView {
	f.style = f.style.WithBackground(c)
	return f
}

// BgRGB sets the background color using RGB values.
func (f *FillView) BgRGB(r, g, b uint8) *FillView {
	f.style = f.style.WithBgRGB(RGB{R: r, G: g, B: b})
	return f
}

// Style sets the complete style.
func (f *FillView) Style(s Style) *FillView {
	f.style = s
	return f
}

func (f *FillView) render(ctx *RenderContext) {
	width, height := ctx.Size()
	if width == 0 || height == 0 {
		return
	}
	ctx.FillStyled(0, 0, width, height, f.char, f.style)
}

func (f *FillView) size(maxWidth, maxHeight int) (int, int) {
	// Fill expands to fill available space
	return maxWidth, maxHeight
}

// Fill is flexible - it expands to fill available space
func (f *FillView) flex() int {
	return 1
}
