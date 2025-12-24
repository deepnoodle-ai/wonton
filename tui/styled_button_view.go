package tui

// styledButtonView displays a button with dimensions and styling
type styledButtonView struct {
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
func StyledButton(label string, callback func()) *styledButtonView {
	return &styledButtonView{
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
func (s *styledButtonView) Width(w int) *styledButtonView {
	s.width = w
	return s
}

// Height sets the button height.
func (s *styledButtonView) Height(h int) *styledButtonView {
	s.height = h
	return s
}

// Size sets both width and height at once.
func (s *styledButtonView) Size(w, h int) *styledButtonView {
	s.width = w
	s.height = h
	return s
}

// Bg sets the background color.
func (s *styledButtonView) Bg(c Color) *styledButtonView {
	s.style = s.style.WithBackground(c)
	return s
}

// Fg sets the foreground color.
func (s *styledButtonView) Fg(c Color) *styledButtonView {
	s.style = s.style.WithForeground(c)
	return s
}

// Style sets the complete style.
func (s *styledButtonView) Style(st Style) *styledButtonView {
	s.style = st
	return s
}

// HoverStyle sets the style when hovered (if hover tracking is enabled).
func (s *styledButtonView) HoverStyle(st Style) *styledButtonView {
	s.hoverStyle = st
	return s
}

// Bold makes the label bold.
func (s *styledButtonView) Bold() *styledButtonView {
	s.style = s.style.WithBold()
	return s
}

// Border sets the border style.
func (s *styledButtonView) Border(style borderStyleType) *styledButtonView {
	s.borderStyle = style
	return s
}

// Centered sets whether the label is centered (default true).
func (s *styledButtonView) Centered(centered bool) *styledButtonView {
	s.centered = centered
	return s
}

func (s *styledButtonView) size(maxWidth, maxHeight int) (int, int) {
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

func (s *styledButtonView) render(ctx *RenderContext) {
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
		interactiveRegistry.RegisterButton(ctx.AbsoluteBounds(), s.callback)
	}
}
