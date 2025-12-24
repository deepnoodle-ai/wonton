package tui

import "image"

// paddingView wraps a view with padding
type paddingView struct {
	inner                    View
	top, right, bottom, left int
}

// Padding wraps a view with equal padding on all sides.
// Padding adds empty space around content, measured in character cells.
//
// Example:
//
//	Padding(2, Text("Content"))  // 2 cells of padding on all sides
func Padding(n int, inner View) View {
	return &paddingView{
		inner:  inner,
		top:    n,
		right:  n,
		bottom: n,
		left:   n,
	}
}

// PaddingHV wraps a view with horizontal and vertical padding.
// The first parameter is horizontal (left and right), the second is vertical (top and bottom).
//
// Example:
//
//	PaddingHV(4, 1, Text("Content"))  // 4 cells horizontal, 1 cell vertical
func PaddingHV(h, v int, inner View) View {
	return &paddingView{
		inner:  inner,
		top:    v,
		right:  h,
		bottom: v,
		left:   h,
	}
}

// PaddingLTRB wraps a view with specific padding on each side.
// Parameters are in CSS order: left, top, right, bottom.
//
// Example:
//
//	PaddingLTRB(1, 2, 3, 4, Text("Content"))  // Different padding on each side
func PaddingLTRB(left, top, right, bottom int, inner View) View {
	return &paddingView{
		inner:  inner,
		top:    top,
		right:  right,
		bottom: bottom,
		left:   left,
	}
}

// flex implements the Flexible interface by delegating to the inner view.
// This allows padded views containing flexible content to participate in
// flex layout distribution.
func (p *paddingView) flex() int {
	if flex, ok := p.inner.(Flexible); ok {
		return flex.flex()
	}
	return 0
}

func (p *paddingView) size(maxWidth, maxHeight int) (int, int) {
	paddingW := p.left + p.right
	paddingH := p.top + p.bottom

	// Reduce constraints by padding
	innerMaxW := maxWidth
	if maxWidth > 0 {
		innerMaxW = maxWidth - paddingW
		if innerMaxW < 0 {
			innerMaxW = 0
		}
	}
	innerMaxH := maxHeight
	if maxHeight > 0 {
		innerMaxH = maxHeight - paddingH
		if innerMaxH < 0 {
			innerMaxH = 0
		}
	}

	innerW, innerH := p.inner.size(innerMaxW, innerMaxH)
	return innerW + paddingW, innerH + paddingH
}

func (p *paddingView) render(ctx *RenderContext) {
	width, height := ctx.Size()
	if width == 0 || height == 0 {
		return
	}

	// Calculate inner bounds
	innerBounds := image.Rect(
		p.left,
		p.top,
		width-p.right,
		height-p.bottom,
	)

	if innerBounds.Dx() > 0 && innerBounds.Dy() > 0 {
		innerCtx := ctx.SubContext(innerBounds)
		p.inner.render(innerCtx)
	}
}
