package tui

import "image"

// Note: We use the existing Alignment type from frame.go:
// - AlignLeft (equivalent to AlignStart)
// - AlignCenter
// - AlignRight (equivalent to AlignEnd)

// StackView arranges children vertically. Construct it with Stack.
type StackView struct {
	children   []View
	gap        int
	alignment  Alignment
	flexFactor int
	childSizes []image.Point // cached during size() for use in render()
}

// Stack creates a vertical stack that arranges children top-to-bottom.
// This is one of the primary layout containers in TUI applications.
//
// Children are laid out vertically with optional spacing and alignment.
// Flexible children (like Spacer) will expand to fill available space.
//
// Example:
//
//	Stack(
//	    Text("Header").Bold(),
//	    Spacer(),
//	    Text("Footer"),
//	).Gap(1).Align(AlignCenter)
func Stack(children ...View) *StackView {
	return &StackView{
		children:   children,
		gap:        0,
		alignment:  AlignLeft,
		flexFactor: 0,
	}
}

// Flex sets the flex factor for this stack.
// Used when this stack is a child of another flex container.
func (s *StackView) Flex(factor int) *StackView {
	s.flexFactor = factor
	return s
}

// flex implements the Flexible interface.
// If no explicit flex factor is set, the stack inherits flexibility from
// its flexible children (like Canvas or Fill), but NOT from Spacers.
// Spacers are layout utilities that shouldn't make their container flexible.
func (s *StackView) flex() int {
	if s.flexFactor != 0 {
		return s.flexFactor
	}
	// Auto-derive: inherit flex from non-Spacer flexible children
	// This makes Stack(Canvas()) flexible, but Stack(Text, Spacer, Text)
	// won't become flexible just because it contains a Spacer
	for _, child := range s.children {
		// Skip spacers - they're layout utilities, not content
		if _, isSpacer := child.(*SpacerView); isSpacer {
			continue
		}
		if flex, ok := child.(Flexible); ok && flex.flex() > 0 {
			return flex.flex()
		}
	}
	return 0
}

// Gap sets the spacing between children in number of rows.
// Only visible children (non-zero size) contribute to spacing.
func (s *StackView) Gap(n int) *StackView {
	s.gap = n
	return s
}

// Align sets the horizontal alignment of children within the stack.
// Options: AlignLeft (default), AlignCenter, AlignRight.
func (s *StackView) Align(a Alignment) *StackView {
	s.alignment = a
	return s
}

func (s *StackView) size(maxWidth, maxHeight int) (int, int) {
	if len(s.children) == 0 {
		return 0, 0
	}

	// Separate flexible vs fixed children
	var flexChildren []int
	var fixedChildren []int
	totalFlex := 0

	for i, child := range s.children {
		if flex, ok := child.(Flexible); ok && flex.flex() > 0 {
			flexChildren = append(flexChildren, i)
			totalFlex += flex.flex()
		} else {
			fixedChildren = append(fixedChildren, i)
		}
	}

	// Initialize child sizes
	s.childSizes = make([]image.Point, len(s.children))

	// Measure fixed children first (unconstrained height)
	totalFixedHeight := 0
	maxChildWidth := 0
	visibleCount := 0

	for _, i := range fixedChildren {
		w, h := s.children[i].size(maxWidth, 0)
		s.childSizes[i] = image.Point{X: w, Y: h}
		totalFixedHeight += h
		if w > maxChildWidth {
			maxChildWidth = w
		}
		if w > 0 || h > 0 {
			visibleCount++
		}
	}

	// Calculate remaining space for flexible children
	if maxHeight > 0 && len(flexChildren) > 0 && totalFlex > 0 {
		// Estimate spacing for now (will be recalculated after measuring flex children)
		estimatedSpacing := 0
		if visibleCount+len(flexChildren) > 1 {
			estimatedSpacing = s.gap * (visibleCount + len(flexChildren) - 1)
		}

		remainingHeight := maxHeight - totalFixedHeight - estimatedSpacing
		if remainingHeight < 0 {
			remainingHeight = 0
		}

		// First pass: get minimum sizes for all flex children
		minHeights := make([]int, len(flexChildren))
		totalMinHeight := 0
		for i, idx := range flexChildren {
			_, minH := s.children[idx].size(maxWidth, 0)
			minHeights[i] = minH
			totalMinHeight += minH
		}

		// A flexible child's "minimum" is its natural content height, and that
		// can be larger than the space left for it: a ScrollView reports the
		// full height of everything it scrolls. Left unclamped it overruns the
		// stack and pushes the children after it off the bottom, so scale the
		// minimums down to what is actually available.
		if totalMinHeight > remainingHeight {
			assigned := 0
			for i := range minHeights {
				scaled := 0
				if totalMinHeight > 0 {
					scaled = minHeights[i] * remainingHeight / totalMinHeight
				}
				minHeights[i] = scaled
				assigned += scaled
			}
			// Integer division can leave a row or two unassigned.
			minHeights[0] += remainingHeight - assigned
			totalMinHeight = remainingHeight
		}

		// Calculate extra space beyond minimums
		extraHeight := remainingHeight - totalMinHeight
		if extraHeight < 0 {
			extraHeight = 0
		}

		// Distribute extra space proportionally, ensuring minimums are respected
		distributedExtra := 0
		for i, idx := range flexChildren {
			flex := s.children[idx].(Flexible).flex()
			extra := (extraHeight * flex) / totalFlex
			// Give remainder to last flex child to avoid rounding issues
			if i == len(flexChildren)-1 {
				extra = extraHeight - distributedExtra
			}
			distributedExtra += extra

			// Total height = minimum + proportional extra
			height := minHeights[i] + extra

			// Get the width for this flexible child
			w, _ := s.children[idx].size(maxWidth, height)
			s.childSizes[idx] = image.Point{X: w, Y: height}
			if w > maxChildWidth {
				maxChildWidth = w
			}
			if w > 0 || height > 0 {
				visibleCount++
			}
		}
	} else {
		// No space constraint or no flex children
		for _, idx := range flexChildren {
			w, h := s.children[idx].size(maxWidth, 0)
			s.childSizes[idx] = image.Point{X: w, Y: h}
			if w > maxChildWidth {
				maxChildWidth = w
			}
			if w > 0 || h > 0 {
				visibleCount++
			}
		}
	}

	// Calculate total height (only count gaps between visible children)
	totalHeight := 0
	for _, size := range s.childSizes {
		totalHeight += size.Y
	}
	if visibleCount > 1 {
		totalHeight += s.gap * (visibleCount - 1)
	}

	return maxChildWidth, totalHeight
}

func (s *StackView) render(ctx *RenderContext) {
	width, height := ctx.Size()
	if width == 0 || height == 0 || len(s.children) == 0 {
		return
	}

	// Re-measure with actual bounds to get correct sizes
	s.size(width, height)

	currentY := 0
	renderedVisible := false

	for i, child := range s.children {
		size := s.childSizes[i]
		// Skip empty children (both dimensions zero)
		if size.X == 0 && size.Y == 0 {
			continue
		}

		// Add gap before this child if we've already rendered a visible child
		if renderedVisible && s.gap > 0 {
			currentY += s.gap
		}

		// Calculate X position based on alignment
		x := 0
		switch s.alignment {
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

		currentY += size.Y
		renderedVisible = true
	}
}
