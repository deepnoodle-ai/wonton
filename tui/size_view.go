package tui

import "image"

// sizeView wraps a view with fixed, minimum, or maximum size constraints
type sizeView struct {
	inner     View
	width     int // 0 = use inner's width
	height    int // 0 = use inner's height
	minWidth  int // 0 = no min constraint
	minHeight int // 0 = no min constraint
	maxWidth  int // 0 = no max constraint
	maxHeight int // 0 = no max constraint
}

// Width wraps a view with a fixed width in character cells.
// The view will be exactly this width, clipping or padding as needed.
//
// Example:
//
//	Width(40, Text("This text will be exactly 40 cells wide"))
func Width(w int, inner View) View {
	return &sizeView{inner: inner, width: w}
}

// Height wraps a view with a fixed height in rows.
// The view will be exactly this height, clipping or padding as needed.
//
// Example:
//
//	Height(10, content)  // Exactly 10 rows tall
func Height(h int, inner View) View {
	return &sizeView{inner: inner, height: h}
}

// Size wraps a view with fixed width and height.
// Combines Width and Height into a single modifier.
//
// Example:
//
//	Size(80, 24, content)  // Exactly 80x24 cells
func Size(w, h int, inner View) View {
	return &sizeView{inner: inner, width: w, height: h}
}

// MaxWidth wraps a view with a maximum width constraint.
// The view can be smaller but will not exceed this width.
//
// Example:
//
//	MaxWidth(80, Text("Long text..."))  // Won't exceed 80 cells
func MaxWidth(w int, inner View) View {
	return &sizeView{inner: inner, maxWidth: w}
}

// MaxHeight wraps a view with a maximum height constraint.
// The view can be smaller but will not exceed this height.
//
// Example:
//
//	MaxHeight(20, content)  // Won't exceed 20 rows
func MaxHeight(h int, inner View) View {
	return &sizeView{inner: inner, maxHeight: h}
}

// MinWidth wraps a view with a minimum width constraint.
// The view can be larger but will not be smaller than this width.
//
// Example:
//
//	MinWidth(40, content)  // At least 40 cells wide
func MinWidth(w int, inner View) View {
	return &sizeView{inner: inner, minWidth: w}
}

// MinHeight wraps a view with a minimum height constraint.
// The view can be larger but will not be smaller than this height.
//
// Example:
//
//	MinHeight(10, content)  // At least 10 rows tall
func MinHeight(h int, inner View) View {
	return &sizeView{inner: inner, minHeight: h}
}

// MinSize wraps a view with minimum width and height constraints.
//
// Example:
//
//	MinSize(40, 10, content)  // At least 40x10 cells
func MinSize(w, h int, inner View) View {
	return &sizeView{inner: inner, minWidth: w, minHeight: h}
}

func (s *sizeView) size(maxWidth, maxHeight int) (int, int) {
	// Apply max constraints
	constrainedMaxW := maxWidth
	if s.maxWidth > 0 && (constrainedMaxW == 0 || s.maxWidth < constrainedMaxW) {
		constrainedMaxW = s.maxWidth
	}
	constrainedMaxH := maxHeight
	if s.maxHeight > 0 && (constrainedMaxH == 0 || s.maxHeight < constrainedMaxH) {
		constrainedMaxH = s.maxHeight
	}

	// Get inner size
	innerW, innerH := s.inner.size(constrainedMaxW, constrainedMaxH)

	// Apply fixed sizes
	w := innerW
	if s.width > 0 {
		w = s.width
	}
	h := innerH
	if s.height > 0 {
		h = s.height
	}

	// Apply min constraints
	if s.minWidth > 0 && w < s.minWidth {
		w = s.minWidth
	}
	if s.minHeight > 0 && h < s.minHeight {
		h = s.minHeight
	}

	// Clamp to max constraints
	if maxWidth > 0 && w > maxWidth {
		w = maxWidth
	}
	if maxHeight > 0 && h > maxHeight {
		h = maxHeight
	}

	return w, h
}

func (s *sizeView) render(ctx *RenderContext) {
	ctxWidth, ctxHeight := ctx.Size()
	if ctxWidth == 0 || ctxHeight == 0 {
		return
	}

	// Calculate the constrained size
	constrainedW, constrainedH := s.size(ctxWidth, ctxHeight)

	// Clamp to context bounds
	if constrainedW > ctxWidth {
		constrainedW = ctxWidth
	}
	if constrainedH > ctxHeight {
		constrainedH = ctxHeight
	}

	// Create subcontext with the constrained size
	innerCtx := ctx.SubContext(image.Rect(0, 0, constrainedW, constrainedH))
	s.inner.render(innerCtx)
}
