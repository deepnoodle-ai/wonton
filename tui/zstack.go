package tui

import "image"

// zStack layers children on top of each other
type zStack struct {
	children   []View
	alignment  Alignment
	childSizes []image.Point
}

// ZStack creates a stack that layers children on top of each other (z-axis).
// The first child is at the bottom layer, the last child is on top.
//
// This is useful for overlays, backgrounds, and layered effects.
// The stack sizes to the largest child.
//
// Example:
//
//	ZStack(
//	    // Background layer
//	    Bordered(Empty()).Border(&SingleBorder),
//	    // Foreground content
//	    Padding(2, Text("Overlay")),
//	).Align(AlignCenter)
func ZStack(children ...View) *zStack {
	return &zStack{
		children:  children,
		alignment: AlignCenter,
	}
}

// Align sets the alignment of children within the stack.
// Options: AlignLeft, AlignCenter (default), AlignRight.
// This affects how smaller children are positioned relative to larger ones.
func (z *zStack) Align(a Alignment) *zStack {
	z.alignment = a
	return z
}

func (z *zStack) size(maxWidth, maxHeight int) (int, int) {
	if len(z.children) == 0 {
		return 0, 0
	}

	z.childSizes = make([]image.Point, len(z.children))

	var maxW, maxH int
	for i, child := range z.children {
		w, h := child.size(maxWidth, maxHeight)
		z.childSizes[i] = image.Point{X: w, Y: h}
		if w > maxW {
			maxW = w
		}
		if h > maxH {
			maxH = h
		}
	}

	return maxW, maxH
}

func (z *zStack) render(ctx *RenderContext) {
	width, height := ctx.Size()
	if width == 0 || height == 0 || len(z.children) == 0 {
		return
	}

	// Re-measure with actual bounds
	z.size(width, height)

	// Render children back-to-front (first child at bottom)
	for i, child := range z.children {
		size := z.childSizes[i]

		// Calculate position based on alignment
		var x, y int
		switch z.alignment {
		case AlignLeft:
			x = 0
			y = 0
		case AlignCenter:
			x = (width - size.X) / 2
			y = (height - size.Y) / 2
		case AlignRight:
			x = width - size.X
			y = height - size.Y
		}

		childCtx := ctx.SubContext(image.Rect(x, y, x+size.X, y+size.Y))
		child.render(childCtx)
	}
}
