package tui

import "image"

// Note: We use the existing Alignment type from frame.go:
// - AlignLeft (equivalent to AlignStart)
// - AlignCenter
// - AlignRight (equivalent to AlignEnd)

// vStack arranges children vertically
type vStack struct {
	children   []View
	gap        int
	alignment  Alignment
	childSizes []image.Point // cached during size() for use in render()
}

// VStack creates a vertical stack that arranges children top-to-bottom.
func VStack(children ...View) *vStack {
	return &vStack{
		children:  children,
		gap:       0,
		alignment: AlignLeft,
	}
}

// Gap sets the spacing between children.
func (v *vStack) Gap(n int) *vStack {
	v.gap = n
	return v
}

// Align sets the horizontal alignment of children.
func (v *vStack) Align(a Alignment) *vStack {
	v.alignment = a
	return v
}

func (v *vStack) size(maxWidth, maxHeight int) (int, int) {
	if len(v.children) == 0 {
		return 0, 0
	}

	// Separate flexible vs fixed children
	var flexChildren []int
	var fixedChildren []int
	totalFlex := 0

	for i, child := range v.children {
		if flex, ok := child.(Flexible); ok {
			flexChildren = append(flexChildren, i)
			totalFlex += flex.flex()
		} else {
			fixedChildren = append(fixedChildren, i)
		}
	}

	// Initialize child sizes
	v.childSizes = make([]image.Point, len(v.children))

	// Measure fixed children first (unconstrained height)
	totalFixedHeight := 0
	maxChildWidth := 0

	for _, i := range fixedChildren {
		w, h := v.children[i].size(maxWidth, 0)
		v.childSizes[i] = image.Point{X: w, Y: h}
		totalFixedHeight += h
		if w > maxChildWidth {
			maxChildWidth = w
		}
	}

	// Add spacing
	totalSpacing := 0
	if len(v.children) > 1 {
		totalSpacing = v.gap * (len(v.children) - 1)
	}
	totalFixedHeight += totalSpacing

	// Calculate remaining space for flexible children
	if maxHeight > 0 && len(flexChildren) > 0 && totalFlex > 0 {
		remainingHeight := maxHeight - totalFixedHeight
		if remainingHeight < 0 {
			remainingHeight = 0
		}

		// Distribute remaining space among flexible children
		distributedHeight := 0
		for i, idx := range flexChildren {
			flex := v.children[idx].(Flexible).flex()
			height := (remainingHeight * flex) / totalFlex
			// Give remainder to last flex child to avoid rounding issues
			if i == len(flexChildren)-1 {
				height = remainingHeight - distributedHeight
			}
			distributedHeight += height

			// Get the width for this flexible child
			w, _ := v.children[idx].size(maxWidth, height)
			v.childSizes[idx] = image.Point{X: w, Y: height}
			if w > maxChildWidth {
				maxChildWidth = w
			}
		}
	} else {
		// No space constraint or no flex children
		for _, idx := range flexChildren {
			w, h := v.children[idx].size(maxWidth, 0)
			v.childSizes[idx] = image.Point{X: w, Y: h}
			if w > maxChildWidth {
				maxChildWidth = w
			}
		}
	}

	// Calculate total height
	totalHeight := 0
	for _, size := range v.childSizes {
		totalHeight += size.Y
	}
	totalHeight += totalSpacing

	return maxChildWidth, totalHeight
}

func (v *vStack) render(ctx *RenderContext) {
	width, height := ctx.Size()
	if width == 0 || height == 0 || len(v.children) == 0 {
		return
	}

	// Re-measure with actual bounds to get correct sizes
	v.size(width, height)

	currentY := 0

	for i, child := range v.children {
		size := v.childSizes[i]
		if size.Y == 0 {
			continue
		}

		// Calculate X position based on alignment
		x := 0
		switch v.alignment {
		case AlignCenter:
			x = (width - size.X) / 2
		case AlignRight:
			x = width - size.X
		}

		// Clip to bounds
		if currentY >= height {
			break
		}
		childHeight := size.Y
		if currentY+childHeight > height {
			childHeight = height - currentY
		}

		childCtx := ctx.SubContext(image.Rect(x, currentY, x+size.X, currentY+childHeight))
		child.render(childCtx)

		currentY += size.Y + v.gap
	}
}

// hStack arranges children horizontally
type hStack struct {
	children   []View
	gap        int
	alignment  Alignment
	childSizes []image.Point
}

// HStack creates a horizontal stack that arranges children left-to-right.
func HStack(children ...View) *hStack {
	return &hStack{
		children:  children,
		gap:       0,
		alignment: AlignLeft,
	}
}

// Gap sets the spacing between children.
func (h *hStack) Gap(n int) *hStack {
	h.gap = n
	return h
}

// Align sets the vertical alignment of children.
func (h *hStack) Align(a Alignment) *hStack {
	h.alignment = a
	return h
}

func (h *hStack) size(maxWidth, maxHeight int) (int, int) {
	if len(h.children) == 0 {
		return 0, 0
	}

	// Separate flexible vs fixed children
	var flexChildren []int
	var fixedChildren []int
	totalFlex := 0

	for i, child := range h.children {
		if flex, ok := child.(Flexible); ok {
			flexChildren = append(flexChildren, i)
			totalFlex += flex.flex()
		} else {
			fixedChildren = append(fixedChildren, i)
		}
	}

	// Initialize child sizes
	h.childSizes = make([]image.Point, len(h.children))

	// Measure fixed children first (unconstrained width)
	totalFixedWidth := 0
	maxChildHeight := 0

	for _, i := range fixedChildren {
		w, ht := h.children[i].size(0, maxHeight)
		h.childSizes[i] = image.Point{X: w, Y: ht}
		totalFixedWidth += w
		if ht > maxChildHeight {
			maxChildHeight = ht
		}
	}

	// Add spacing
	totalSpacing := 0
	if len(h.children) > 1 {
		totalSpacing = h.gap * (len(h.children) - 1)
	}
	totalFixedWidth += totalSpacing

	// Calculate remaining space for flexible children
	if maxWidth > 0 && len(flexChildren) > 0 && totalFlex > 0 {
		remainingWidth := maxWidth - totalFixedWidth
		if remainingWidth < 0 {
			remainingWidth = 0
		}

		// Distribute remaining space among flexible children
		distributedWidth := 0
		for i, idx := range flexChildren {
			flex := h.children[idx].(Flexible).flex()
			width := (remainingWidth * flex) / totalFlex
			// Give remainder to last flex child
			if i == len(flexChildren)-1 {
				width = remainingWidth - distributedWidth
			}
			distributedWidth += width

			// Get the height for this flexible child
			_, ht := h.children[idx].size(width, maxHeight)
			h.childSizes[idx] = image.Point{X: width, Y: ht}
			if ht > maxChildHeight {
				maxChildHeight = ht
			}
		}
	} else {
		// No space constraint or no flex children
		for _, idx := range flexChildren {
			w, ht := h.children[idx].size(0, maxHeight)
			h.childSizes[idx] = image.Point{X: w, Y: ht}
			if ht > maxChildHeight {
				maxChildHeight = ht
			}
		}
	}

	// Calculate total width
	totalWidth := 0
	for _, size := range h.childSizes {
		totalWidth += size.X
	}
	totalWidth += totalSpacing

	return totalWidth, maxChildHeight
}

func (h *hStack) render(ctx *RenderContext) {
	width, height := ctx.Size()
	if width == 0 || height == 0 || len(h.children) == 0 {
		return
	}

	// Re-measure with actual bounds
	h.size(width, height)

	currentX := 0

	for i, child := range h.children {
		size := h.childSizes[i]
		if size.X == 0 {
			continue
		}

		// Calculate Y position based on alignment
		y := 0
		switch h.alignment {
		case AlignCenter:
			y = (height - size.Y) / 2
		case AlignRight:
			y = height - size.Y
		}

		// Clip to bounds
		if currentX >= width {
			break
		}
		childWidth := size.X
		if currentX+childWidth > width {
			childWidth = width - currentX
		}

		childCtx := ctx.SubContext(image.Rect(currentX, y, currentX+childWidth, y+size.Y))
		child.render(childCtx)

		currentX += size.X + h.gap
	}
}

// zStack layers children on top of each other
type zStack struct {
	children   []View
	alignment  Alignment
	childSizes []image.Point
}

// ZStack creates a stack that layers children on top of each other.
// The first child is at the bottom, the last child is on top.
func ZStack(children ...View) *zStack {
	return &zStack{
		children:  children,
		alignment: AlignCenter,
	}
}

// Align sets the alignment of children within the stack.
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
