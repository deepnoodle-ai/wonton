package tui

import "image"

// group arranges children horizontally
type group struct {
	children   []View
	gap        int
	alignment  Alignment
	childSizes []image.Point
}

// Group creates a group that arranges children left-to-right.
func Group(children ...View) *group {
	return &group{
		children:  children,
		gap:       0,
		alignment: AlignLeft,
	}
}

// Gap sets the spacing between children.
func (g *group) Gap(n int) *group {
	g.gap = n
	return g
}

// Align sets the vertical alignment of children.
func (g *group) Align(a Alignment) *group {
	g.alignment = a
	return g
}

func (g *group) size(maxWidth, maxHeight int) (int, int) {
	if len(g.children) == 0 {
		return 0, 0
	}

	// Separate flexible vs fixed children
	var flexChildren []int
	var fixedChildren []int
	totalFlex := 0

	for i, child := range g.children {
		if flex, ok := child.(Flexible); ok {
			flexChildren = append(flexChildren, i)
			totalFlex += flex.flex()
		} else {
			fixedChildren = append(fixedChildren, i)
		}
	}

	// Initialize child sizes
	g.childSizes = make([]image.Point, len(g.children))

	// Measure fixed children first (unconstrained width)
	totalFixedWidth := 0
	maxChildHeight := 0

	for _, i := range fixedChildren {
		w, ht := g.children[i].size(0, maxHeight)
		g.childSizes[i] = image.Point{X: w, Y: ht}
		totalFixedWidth += w
		if ht > maxChildHeight {
			maxChildHeight = ht
		}
	}

	// Add spacing
	totalSpacing := 0
	if len(g.children) > 1 {
		totalSpacing = g.gap * (len(g.children) - 1)
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
			flex := g.children[idx].(Flexible).flex()
			width := (remainingWidth * flex) / totalFlex
			// Give remainder to last flex child
			if i == len(flexChildren)-1 {
				width = remainingWidth - distributedWidth
			}
			distributedWidth += width

			// Get the height for this flexible child
			_, ht := g.children[idx].size(width, maxHeight)
			g.childSizes[idx] = image.Point{X: width, Y: ht}
			if ht > maxChildHeight {
				maxChildHeight = ht
			}
		}
	} else {
		// No space constraint or no flex children
		for _, idx := range flexChildren {
			w, ht := g.children[idx].size(0, maxHeight)
			g.childSizes[idx] = image.Point{X: w, Y: ht}
			if ht > maxChildHeight {
				maxChildHeight = ht
			}
		}
	}

	// Calculate total width
	totalWidth := 0
	for _, size := range g.childSizes {
		totalWidth += size.X
	}
	totalWidth += totalSpacing

	return totalWidth, maxChildHeight
}

func (g *group) render(ctx *RenderContext) {
	width, height := ctx.Size()
	if width == 0 || height == 0 || len(g.children) == 0 {
		return
	}

	// Re-measure with actual bounds
	g.size(width, height)

	currentX := 0

	for i, child := range g.children {
		size := g.childSizes[i]
		if size.X == 0 {
			continue
		}

		// Calculate Y position based on alignment
		y := 0
		switch g.alignment {
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

		currentX += size.X + g.gap
	}
}
