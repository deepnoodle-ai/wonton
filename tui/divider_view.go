package tui

// dividerView displays a horizontal line separator
type dividerView struct {
	char  rune
	style Style
	title string
}

// Divider creates a horizontal line separator that fills available width.
//
// Example:
//
//	Divider()
//	Divider().Char('═')
//	Divider().Title("Section")
func Divider() *dividerView {
	return &dividerView{
		char:  '─',
		style: NewStyle().WithForeground(ColorBrightBlack),
	}
}

// Char sets the character used for the divider line.
func (d *dividerView) Char(c rune) *dividerView {
	d.char = c
	return d
}

// Fg sets the foreground color.
func (d *dividerView) Fg(c Color) *dividerView {
	d.style = d.style.WithForeground(c)
	return d
}

// Bg sets the background color.
func (d *dividerView) Bg(c Color) *dividerView {
	d.style = d.style.WithBackground(c)
	return d
}

// Style sets the complete style.
func (d *dividerView) Style(s Style) *dividerView {
	d.style = s
	return d
}

// Title adds centered text to the divider.
func (d *dividerView) Title(title string) *dividerView {
	d.title = title
	return d
}

// Bold makes the divider bold.
func (d *dividerView) Bold() *dividerView {
	d.style = d.style.WithBold()
	return d
}

// Dim makes the divider dim.
func (d *dividerView) Dim() *dividerView {
	d.style = d.style.WithDim()
	return d
}

func (d *dividerView) size(maxWidth, maxHeight int) (int, int) {
	// Request full width if available
	w := maxWidth
	if w == 0 {
		if d.title != "" {
			titleW, _ := MeasureText(d.title)
			w = titleW + 4 // padding around title
		} else {
			w = 1
		}
	}
	return w, 1
}

func (d *dividerView) render(ctx *RenderContext) {
	width, height := ctx.Size()
	if width == 0 || height == 0 {
		return
	}

	if d.title == "" {
		// Simple line
		for x := 0; x < width; x++ {
			ctx.SetCell(x, 0, d.char, d.style)
		}
		return
	}

	// Line with centered title
	titleW, _ := MeasureText(d.title)
	paddedTitle := " " + d.title + " "
	paddedTitleW := titleW + 2

	if paddedTitleW >= width {
		// Title too wide, just show what fits
		ctx.PrintTruncated(0, 0, d.title, d.style)
		return
	}

	// Calculate where to put the title
	titleStart := (width - paddedTitleW) / 2

	// Draw left side of line
	for x := 0; x < titleStart; x++ {
		ctx.SetCell(x, 0, d.char, d.style)
	}

	// Draw title
	ctx.PrintStyled(titleStart, 0, paddedTitle, d.style)

	// Draw right side of line
	for x := titleStart + paddedTitleW; x < width; x++ {
		ctx.SetCell(x, 0, d.char, d.style)
	}
}

// headerBarView displays a full-width header bar with centered text
type headerBarView struct {
	text  string
	style Style
}

// HeaderBar creates a full-width header bar with centered text.
//
// Example:
//
//	HeaderBar("My App").Bg(ColorBlue).Fg(ColorWhite)
func HeaderBar(text string) *headerBarView {
	return &headerBarView{
		text:  text,
		style: NewStyle().WithBackground(ColorBlue).WithForeground(ColorWhite).WithBold(),
	}
}

// Fg sets the foreground color.
func (h *headerBarView) Fg(c Color) *headerBarView {
	h.style = h.style.WithForeground(c)
	return h
}

// Bg sets the background color.
func (h *headerBarView) Bg(c Color) *headerBarView {
	h.style = h.style.WithBackground(c)
	return h
}

// Bold makes the text bold.
func (h *headerBarView) Bold() *headerBarView {
	h.style = h.style.WithBold()
	return h
}

// Style sets the complete style.
func (h *headerBarView) Style(s Style) *headerBarView {
	h.style = s
	return h
}

func (h *headerBarView) size(maxWidth, maxHeight int) (int, int) {
	w := maxWidth
	if w == 0 {
		textW, _ := MeasureText(h.text)
		w = textW + 2 // padding
	}
	return w, 1
}

func (h *headerBarView) render(ctx *RenderContext) {
	width, height := ctx.Size()
	if width == 0 || height == 0 {
		return
	}

	textW, _ := MeasureText(h.text)

	// Center the text
	startX := (width - textW) / 2
	if startX < 0 {
		startX = 0
	}

	// Fill entire width with background
	for x := 0; x < width; x++ {
		ctx.SetCell(x, 0, ' ', h.style)
	}

	// Draw centered text
	ctx.PrintStyled(startX, 0, h.text, h.style)
}

// StatusBar creates a full-width status bar (same as HeaderBar but defaults to bottom style).
func StatusBar(text string) *headerBarView {
	return &headerBarView{
		text:  text,
		style: NewStyle().WithBackground(ColorBrightBlack).WithForeground(ColorWhite),
	}
}
