package tui

// DividerView displays a horizontal line separator
type DividerView struct {
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
func Divider() *DividerView {
	return &DividerView{
		char:  '─',
		style: NewStyle().WithForeground(ColorBrightBlack),
	}
}

// Char sets the character used for the divider line.
func (d *DividerView) Char(c rune) *DividerView {
	d.char = c
	return d
}

// Fg sets the foreground color.
func (d *DividerView) Fg(c Color) *DividerView {
	d.style = d.style.WithForeground(c)
	return d
}

// Bg sets the background color.
func (d *DividerView) Bg(c Color) *DividerView {
	d.style = d.style.WithBackground(c)
	return d
}

// Style sets the complete style.
func (d *DividerView) Style(s Style) *DividerView {
	d.style = s
	return d
}

// Title adds centered text to the divider.
func (d *DividerView) Title(title string) *DividerView {
	d.title = title
	return d
}

// Bold makes the divider bold.
func (d *DividerView) Bold() *DividerView {
	d.style = d.style.WithBold()
	return d
}

// Dim makes the divider dim.
func (d *DividerView) Dim() *DividerView {
	d.style = d.style.WithDim()
	return d
}

func (d *DividerView) size(maxWidth, maxHeight int) (int, int) {
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

func (d *DividerView) render(ctx *RenderContext) {
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

// HeaderBarView displays a full-width header bar with centered text
type HeaderBarView struct {
	text  string
	style Style
}

// HeaderBar creates a full-width header bar with centered text.
//
// Example:
//
//	HeaderBar("My App").Bg(ColorBlue).Fg(ColorWhite)
func HeaderBar(text string) *HeaderBarView {
	return &HeaderBarView{
		text:  text,
		style: NewStyle().WithBackground(ColorBlue).WithForeground(ColorWhite).WithBold(),
	}
}

// Fg sets the foreground color.
func (h *HeaderBarView) Fg(c Color) *HeaderBarView {
	h.style = h.style.WithForeground(c)
	return h
}

// Bg sets the background color.
func (h *HeaderBarView) Bg(c Color) *HeaderBarView {
	h.style = h.style.WithBackground(c)
	return h
}

// Bold makes the text bold.
func (h *HeaderBarView) Bold() *HeaderBarView {
	h.style = h.style.WithBold()
	return h
}

// Style sets the complete style.
func (h *HeaderBarView) Style(s Style) *HeaderBarView {
	h.style = s
	return h
}

func (h *HeaderBarView) size(maxWidth, maxHeight int) (int, int) {
	w := maxWidth
	if w == 0 {
		textW, _ := MeasureText(h.text)
		w = textW + 2 // padding
	}
	return w, 1
}

func (h *HeaderBarView) render(ctx *RenderContext) {
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
func StatusBar(text string) *HeaderBarView {
	return &HeaderBarView{
		text:  text,
		style: NewStyle().WithBackground(ColorBrightBlack).WithForeground(ColorWhite),
	}
}
