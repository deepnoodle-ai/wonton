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

	return insetSize(p.inner, maxWidth, maxHeight, paddingW, paddingH)
}

// insetSize measures inner inside a frame that takes insetW columns and insetH
// rows, such as padding or a border. When the frame uses up all of a limit,
// inner has no room on that axis. A limit of 0 means unconstrained, so inner
// is measured with a limit of 1 and the result is held at the limit.
func insetSize(inner View, maxWidth, maxHeight, insetW, insetH int) (int, int) {
	innerMaxW, fullW := insetLimit(maxWidth, insetW)
	innerMaxH, fullH := insetLimit(maxHeight, insetH)

	innerW, innerH := inner.size(innerMaxW, innerMaxH)
	w, h := innerW+insetW, innerH+insetH
	if fullW {
		w = maxWidth
	}
	if fullH {
		h = maxHeight
	}
	return w, h
}

// insetLimit reduces a size limit by inset. full reports a limit the inset
// uses up.
func insetLimit(limit, inset int) (inner int, full bool) {
	if limit <= 0 {
		return limit, false
	}
	if limit <= inset {
		return 1, true
	}
	return limit - inset, false
}

// insetBounds returns the area inside a width by height space that an inset
// of left, top, right and bottom cells leaves, such as padding or a border.
// ok is false when the inset leaves no room. image.Rect would swap the
// corners instead and return an area that overlaps the inset.
func insetBounds(width, height, left, top, right, bottom int) (r image.Rectangle, ok bool) {
	if width-right <= left || height-bottom <= top {
		return image.Rectangle{}, false
	}
	return image.Rect(left, top, width-right, height-bottom), true
}

func (p *paddingView) render(ctx *RenderContext) {
	width, height := ctx.Size()
	if width == 0 || height == 0 {
		return
	}

	innerBounds, ok := insetBounds(width, height, p.left, p.top, p.right, p.bottom)
	if ok {
		innerCtx := ctx.SubContext(innerBounds)
		p.inner.render(innerCtx)
	}
}
