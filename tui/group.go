package tui

import "image"

// group arranges children horizontally
type group struct {
	children   []View
	gap        int
	alignment  Alignment
	flexFactor int
	childSizes []image.Point
}

// Group creates a group that arranges children left-to-right (horizontal layout).
//
// Children are laid out horizontally with optional spacing and alignment.
// Flexible children (like Spacer) will expand to fill available space.
//
// Example:
//
//	Group(
//	    Text("Left"),
//	    Spacer(),
//	    Text("Right"),
//	).Gap(2).Align(AlignCenter)
func Group(children ...View) *group {
	return &group{
		children:   children,
		gap:        0,
		alignment:  AlignLeft,
		flexFactor: 0,
	}
}

// Flex sets the flex factor for this group.
// Used when this group is a child of another flex container.
func (g *group) Flex(factor int) *group {
	g.flexFactor = factor
	return g
}

// flex implements the Flexible interface.
// If no explicit flex factor is set, the group inherits flexibility from
// its children. This allows flexible views (like Canvas) to expand even
// when nested inside containers, matching CSS flexbox behavior.
func (g *group) flex() int {
	if g.flexFactor != 0 {
		return g.flexFactor
	}
	// Auto-derive: inherit max flex from children
	// This makes Group(Canvas()) flexible because Canvas is flexible
	for _, child := range g.children {
		if flex, ok := child.(Flexible); ok && flex.flex() > 0 {
			return flex.flex()
		}
	}
	return 0
}

// Gap sets the spacing between children in number of columns.
// Only visible children (non-zero size) contribute to spacing.
func (g *group) Gap(n int) *group {
	g.gap = n
	return g
}

// Align sets the vertical alignment of children within the group.
// Options: AlignLeft (top), AlignCenter (middle), AlignRight (bottom).
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
		if flex, ok := child.(Flexible); ok && flex.flex() > 0 {
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
	visibleCount := 0

	for _, i := range fixedChildren {
		w, ht := g.children[i].size(0, maxHeight)
		g.childSizes[i] = image.Point{X: w, Y: ht}
		totalFixedWidth += w
		if ht > maxChildHeight {
			maxChildHeight = ht
		}
		if w > 0 || ht > 0 {
			visibleCount++
		}
	}

	// Calculate remaining space for flexible children
	if maxWidth > 0 && len(flexChildren) > 0 && totalFlex > 0 {
		// Estimate spacing for now (will be recalculated after measuring flex children)
		estimatedSpacing := 0
		if visibleCount+len(flexChildren) > 1 {
			estimatedSpacing = g.gap * (visibleCount + len(flexChildren) - 1)
		}

		remainingWidth := maxWidth - totalFixedWidth - estimatedSpacing
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
			if width > 0 || ht > 0 {
				visibleCount++
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
			if w > 0 || ht > 0 {
				visibleCount++
			}
		}
	}

	// Calculate total width (only count gaps between visible children)
	totalWidth := 0
	for _, size := range g.childSizes {
		totalWidth += size.X
	}
	if visibleCount > 1 {
		totalWidth += g.gap * (visibleCount - 1)
	}

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
	renderedVisible := false

	for i, child := range g.children {
		size := g.childSizes[i]
		// Skip empty children (both dimensions zero)
		if size.X == 0 && size.Y == 0 {
			continue
		}

		// Add gap before this child if we've already rendered a visible child
		if renderedVisible && g.gap > 0 {
			currentX += g.gap
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

		currentX += size.X
		renderedVisible = true
	}
}

// Example:
//
//	Group(
//	    Text("Left"),
//	    Spacer(),
//	    Text("Right"),
//	)
