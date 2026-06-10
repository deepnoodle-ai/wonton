package tui

import "image"

// animatedBorderedView wraps a view with an animated border.
type animatedBorderedView struct {
	inner             View
	border            *BorderStyle
	borderAnimation   BorderAnimation
	title             string
	titleStyle        Style
	staticBorderStyle Style
}

// AnimatedBordered wraps a view with an animated border.
func AnimatedBordered(inner View, animation BorderAnimation) *animatedBorderedView {
	return &animatedBorderedView{
		inner: inner,
		border: &BorderStyle{
			TopLeft: "┌", TopRight: "┐", BottomLeft: "└", BottomRight: "┘",
			Horizontal: "─", Vertical: "│",
		},
		borderAnimation:   animation,
		staticBorderStyle: NewStyle(),
		titleStyle:        NewStyle(),
	}
}

// Border sets the border characters.
func (f *animatedBorderedView) Border(style *BorderStyle) *animatedBorderedView {
	f.border = style
	return f
}

// Title sets the title shown in the top border.
func (f *animatedBorderedView) Title(title string) *animatedBorderedView {
	f.title = title
	return f
}

// TitleStyle sets the style for the title text.
func (f *animatedBorderedView) TitleStyle(s Style) *animatedBorderedView {
	f.titleStyle = s
	return f
}

// StaticBorderStyle sets a fallback style when no animation is active.
func (f *animatedBorderedView) StaticBorderStyle(s Style) *animatedBorderedView {
	f.staticBorderStyle = s
	return f
}

func (f *animatedBorderedView) size(maxWidth, maxHeight int) (int, int) {
	borderSize := 0
	if f.border != nil {
		borderSize = 2
	}

	innerMaxW := maxWidth
	if maxWidth > 0 {
		innerMaxW = maxWidth - borderSize
		if innerMaxW < 0 {
			innerMaxW = 0
		}
	}
	innerMaxH := maxHeight
	if maxHeight > 0 {
		innerMaxH = maxHeight - borderSize
		if innerMaxH < 0 {
			innerMaxH = 0
		}
	}

	innerW, innerH := f.inner.size(innerMaxW, innerMaxH)
	return innerW + borderSize, innerH + borderSize
}

func (f *animatedBorderedView) render(ctx *RenderContext) {
	w, h := ctx.Size()
	if w == 0 || h == 0 {
		return
	}

	if f.border == nil {
		f.inner.render(ctx)
		return
	}

	frame := ctx.Frame()

	// Positions run clockwise around the full perimeter, corners included,
	// so position-based animations flow smoothly through the corners:
	// TL=0, top edge 1..w-2, TR=w-1, right edge w..w+h-3, BR=w+h-2,
	// bottom edge w+h-1..2w+h-4, BL=2w+h-3, left edge 2w+h-2..perimeter-1.
	perimeter := (w-2)*2 + (h-2)*2 + 4

	// Draw top border
	position := 1 // position 0 is the top-left corner
	for x := 1; x < w-1; x++ {
		style := f.staticBorderStyle
		if f.borderAnimation != nil {
			style = f.borderAnimation.GetBorderStyle(frame, BorderPartTop, position, perimeter)
		}
		ctx.PrintTruncated(x, 0, f.border.Horizontal, style)
		position++
	}
	position++ // top-right corner

	// Draw right border
	for y := 1; y < h-1; y++ {
		style := f.staticBorderStyle
		if f.borderAnimation != nil {
			style = f.borderAnimation.GetBorderStyle(frame, BorderPartRight, position, perimeter)
		}
		if w > 1 {
			ctx.PrintTruncated(w-1, y, f.border.Vertical, style)
		}
		position++
	}
	position++ // bottom-right corner

	// Draw bottom border
	for x := w - 2; x > 0; x-- {
		style := f.staticBorderStyle
		if f.borderAnimation != nil {
			style = f.borderAnimation.GetBorderStyle(frame, BorderPartBottom, position, perimeter)
		}
		if h > 1 {
			ctx.PrintTruncated(x, h-1, f.border.Horizontal, style)
		}
		position++
	}
	position++ // bottom-left corner

	// Draw left border
	for y := h - 2; y > 0; y-- {
		style := f.staticBorderStyle
		if f.borderAnimation != nil {
			style = f.borderAnimation.GetBorderStyle(frame, BorderPartLeft, position, perimeter)
		}
		ctx.PrintTruncated(0, y, f.border.Vertical, style)
		position++
	}

	// Draw corners at their perimeter positions
	tlStyle := f.staticBorderStyle
	trStyle := f.staticBorderStyle
	blStyle := f.staticBorderStyle
	brStyle := f.staticBorderStyle

	if f.borderAnimation != nil {
		tlStyle = f.borderAnimation.GetBorderStyle(frame, BorderPartTopLeft, 0, perimeter)
		trStyle = f.borderAnimation.GetBorderStyle(frame, BorderPartTopRight, w-1, perimeter)
		if h > 1 {
			blStyle = f.borderAnimation.GetBorderStyle(frame, BorderPartBottomLeft, 2*w+h-3, perimeter)
		}
		if w > 1 && h > 1 {
			brStyle = f.borderAnimation.GetBorderStyle(frame, BorderPartBottomRight, w+h-2, perimeter)
		}
	}

	ctx.PrintTruncated(0, 0, f.border.TopLeft, tlStyle)
	if w > 1 {
		ctx.PrintTruncated(w-1, 0, f.border.TopRight, trStyle)
	}
	if h > 1 {
		ctx.PrintTruncated(0, h-1, f.border.BottomLeft, blStyle)
		if w > 1 {
			ctx.PrintTruncated(w-1, h-1, f.border.BottomRight, brStyle)
		}
	}

	// Title
	if f.title != "" && w > 4 {
		titleW, _ := MeasureText(f.title)
		maxTitleW := w - 4
		if titleW > maxTitleW {
			titleW = maxTitleW
		}
		titleX := 2
		ctx.PrintTruncated(titleX, 0, f.title[:min(len(f.title), maxTitleW)], f.titleStyle)
	}

	// Inner content
	innerBounds := image.Rect(1, 1, w-1, h-1)
	if innerBounds.Dx() > 0 && innerBounds.Dy() > 0 {
		innerCtx := ctx.SubContext(innerBounds)
		f.inner.render(innerCtx)
	}
}
