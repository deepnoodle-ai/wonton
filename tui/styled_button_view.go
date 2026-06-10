package tui

// StyledButtonView displays a button with dimensions and styling
type StyledButtonView struct {
	label       string
	callback    func()
	width       int
	height      int
	style       Style
	hoverStyle  Style
	borderStyle borderStyleType
	centered    bool
}

// StyledButton creates a styled button with dimensions.
// Unlike Clickable, this draws a filled box with centered text.
//
// Example:
//
//	StyledButton("Submit", func() { app.submit() }).Width(20).Height(3).Bg(ColorBlue)
func StyledButton(label string, callback func()) *StyledButtonView {
	return &StyledButtonView{
		label:       label,
		callback:    callback,
		width:       0,
		height:      1,
		style:       NewStyle().WithBackground(ColorBlue).WithForeground(ColorWhite),
		hoverStyle:  NewStyle().WithBackground(ColorCyan).WithForeground(ColorBlack),
		borderStyle: BorderNone,
		centered:    true,
	}
}

// Width sets the button width.
func (s *StyledButtonView) Width(w int) *StyledButtonView {
	s.width = w
	return s
}

// Height sets the button height.
func (s *StyledButtonView) Height(h int) *StyledButtonView {
	s.height = h
	return s
}

// Size sets both width and height at once.
func (s *StyledButtonView) Size(w, h int) *StyledButtonView {
	s.width = w
	s.height = h
	return s
}

// Bg sets the background color.
func (s *StyledButtonView) Bg(c Color) *StyledButtonView {
	s.style = s.style.WithBackground(c)
	return s
}

// Fg sets the foreground color.
func (s *StyledButtonView) Fg(c Color) *StyledButtonView {
	s.style = s.style.WithForeground(c)
	return s
}

// Style sets the complete style.
func (s *StyledButtonView) Style(st Style) *StyledButtonView {
	s.style = st
	return s
}

// HoverStyle sets the style when hovered (if hover tracking is enabled).
func (s *StyledButtonView) HoverStyle(st Style) *StyledButtonView {
	s.hoverStyle = st
	return s
}

// Bold makes the label bold.
func (s *StyledButtonView) Bold() *StyledButtonView {
	s.style = s.style.WithBold()
	return s
}

// Border sets the border style.
func (s *StyledButtonView) Border(style borderStyleType) *StyledButtonView {
	s.borderStyle = style
	return s
}

// Centered sets whether the label is centered (default true).
func (s *StyledButtonView) Centered(centered bool) *StyledButtonView {
	s.centered = centered
	return s
}

func (s *StyledButtonView) size(maxWidth, maxHeight int) (int, int) {
	labelW, _ := MeasureText(s.label)
	w := s.width
	if w == 0 {
		w = labelW + 2 // padding
	}
	h := s.height
	if h == 0 {
		h = 1
	}

	if maxWidth > 0 && w > maxWidth {
		w = maxWidth
	}
	if maxHeight > 0 && h > maxHeight {
		h = maxHeight
	}
	return w, h
}

func (s *StyledButtonView) render(ctx *RenderContext) {
	width, height := ctx.Size()
	if width == 0 || height == 0 {
		return
	}

	// Fill background
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			ctx.SetCell(x, y, ' ', s.style)
		}
	}

	// Calculate text position
	labelW, _ := MeasureText(s.label)
	textX := 0
	textY := height / 2
	if s.centered {
		textX = (width - labelW) / 2
		if textX < 0 {
			textX = 0
		}
	}

	// Draw label
	ctx.PrintTruncated(textX, textY, s.label, s.style)

	// Register click region
	if s.callback != nil {
		ctx.registries().interactive.RegisterButton(ctx.AbsoluteBounds(), s.callback)
	}
}
