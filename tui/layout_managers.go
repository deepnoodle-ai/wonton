package tui

import (
	"image"
)

// VBoxLayout arranges children vertically in a column, top to bottom.
// Each child gets its preferred/minimum height, and width fills the container.
type VBoxLayout struct {
	spacing    int             // Vertical spacing between children in pixels
	alignment  LayoutAlignment // How to align children horizontally
	distribute bool            // Whether to distribute extra space among children
}

// NewVBoxLayout creates a new vertical box layout with the specified spacing
func NewVBoxLayout(spacing int) *VBoxLayout {
	return &VBoxLayout{
		spacing:    spacing,
		alignment:  LayoutAlignStart,
		distribute: false,
	}
}

// WithAlignment sets the horizontal alignment for children
func (vb *VBoxLayout) WithAlignment(align LayoutAlignment) *VBoxLayout {
	vb.alignment = align
	return vb
}

// WithDistribute enables distributing extra vertical space among children
func (vb *VBoxLayout) WithDistribute(distribute bool) *VBoxLayout {
	vb.distribute = distribute
	return vb
}

// Layout positions children vertically within the container bounds
func (vb *VBoxLayout) Layout(containerBounds image.Rectangle, children []ComposableWidget) {
	if len(children) == 0 {
		return
	}

	containerWidth := containerBounds.Dx()
	containerHeight := containerBounds.Dy()

	// Calculate total height needed and collect child sizes
	totalMinHeight := 0
	childSizes := make([]image.Point, len(children))
	totalGrow := 0

	for i, child := range children {
		params := child.(interface{ GetLayoutParams() LayoutParams }).GetLayoutParams()

		// Get preferred size, but respect container width
		preferredSize := child.GetPreferredSize()
		childSizes[i] = preferredSize

		// Account for margins
		height := preferredSize.Y + params.MarginTop + params.MarginBottom
		totalMinHeight += height
		totalGrow += params.Grow

		// Add spacing (except for last child)
		if i < len(children)-1 {
			totalMinHeight += vb.spacing
		}
	}

	// Calculate extra space to distribute
	extraSpace := containerHeight - totalMinHeight
	if extraSpace < 0 {
		extraSpace = 0
	}

	// Position children
	currentY := containerBounds.Min.Y

	for i, child := range children {
		params := child.(interface{ GetLayoutParams() LayoutParams }).GetLayoutParams()

		// Calculate child height
		childHeight := childSizes[i].Y

		// Distribute extra space if enabled
		if vb.distribute && extraSpace > 0 && totalGrow > 0 && params.Grow > 0 {
			extraHeight := (extraSpace * params.Grow) / totalGrow
			childHeight += extraHeight
		}

		// Apply margins
		currentY += params.MarginTop

		// Calculate child width based on alignment
		var childWidth int
		if vb.alignment == LayoutAlignStretch {
			// Stretch to fill available width
			childWidth = containerWidth - params.MarginLeft - params.MarginRight
		} else {
			// Use preferred size, but clamp to available width
			maxWidth := containerWidth - params.MarginLeft - params.MarginRight
			childWidth = childSizes[i].X
			if childWidth > maxWidth {
				childWidth = maxWidth
			}
		}

		// Apply size constraints
		childSize := ApplyConstraints(
			image.Point{X: childWidth, Y: childHeight},
			params.Constraints,
		)

		// Calculate X position based on alignment
		childX := containerBounds.Min.X + params.MarginLeft

		switch vb.alignment {
		case LayoutAlignCenter:
			// Center horizontally
			childX = containerBounds.Min.X + (containerWidth-childSize.X)/2
		case LayoutAlignEnd:
			// Align to right
			childX = containerBounds.Max.X - childSize.X - params.MarginRight
		case LayoutAlignStretch:
			// Already positioned at left with full width
		default: // LayoutAlignStart
			// Already set to left alignment
		}

		// Set child bounds
		child.SetBounds(image.Rect(
			childX,
			currentY,
			childX+childSize.X,
			currentY+childSize.Y,
		))

		// Move to next position
		currentY += childSize.Y + params.MarginBottom + vb.spacing
	}
}

// CalculateMinSize returns the minimum size needed for all children
func (vb *VBoxLayout) CalculateMinSize(children []ComposableWidget) image.Point {
	if len(children) == 0 {
		return image.Point{X: 0, Y: 0}
	}

	maxWidth := 0
	totalHeight := 0

	for i, child := range children {
		params := child.(interface{ GetLayoutParams() LayoutParams }).GetLayoutParams()
		minSize := child.GetMinSize()

		// Track maximum width needed
		width := minSize.X + params.MarginLeft + params.MarginRight
		if width > maxWidth {
			maxWidth = width
		}

		// Sum heights
		height := minSize.Y + params.MarginTop + params.MarginBottom
		totalHeight += height

		// Add spacing (except for last child)
		if i < len(children)-1 {
			totalHeight += vb.spacing
		}
	}

	return image.Point{X: maxWidth, Y: totalHeight}
}

// CalculatePreferredSize returns the preferred size for all children
func (vb *VBoxLayout) CalculatePreferredSize(children []ComposableWidget) image.Point {
	if len(children) == 0 {
		return image.Point{X: 0, Y: 0}
	}

	maxWidth := 0
	totalHeight := 0

	for i, child := range children {
		params := child.(interface{ GetLayoutParams() LayoutParams }).GetLayoutParams()
		preferredSize := child.GetPreferredSize()

		// Track maximum width needed
		width := preferredSize.X + params.MarginLeft + params.MarginRight
		if width > maxWidth {
			maxWidth = width
		}

		// Sum heights
		height := preferredSize.Y + params.MarginTop + params.MarginBottom
		totalHeight += height

		// Add spacing (except for last child)
		if i < len(children)-1 {
			totalHeight += vb.spacing
		}
	}

	return image.Point{X: maxWidth, Y: totalHeight}
}

// Measure implements ConstraintLayoutManager.Measure
func (vb *VBoxLayout) Measure(children []ComposableWidget, constraints SizeConstraints) image.Point {
	if len(children) == 0 {
		return image.Point{X: 0, Y: 0}
	}

	maxWidth := 0
	totalHeight := 0

	// For VBox, we typically constrain width but let height grow
	childConstraints := constraints
	childConstraints.MinHeight = 0
	childConstraints.MaxHeight = 0 // Unconstrained height

	for i, child := range children {
		// Get layout params (safe cast as per existing pattern)
		var params LayoutParams
		if lp, ok := child.(interface{ GetLayoutParams() LayoutParams }); ok {
			params = lp.GetLayoutParams()
		} else {
			params = DefaultLayoutParams()
		}

		// Adjust constraints for margins
		cConstraints := childConstraints
		marginW := params.MarginLeft + params.MarginRight
		if cConstraints.HasMaxWidth() {
			cConstraints.MaxWidth -= marginW
			if cConstraints.MaxWidth < 0 {
				cConstraints.MaxWidth = 0
			}
		}
		// If stretching, we might enforce min width?
		// For now, let's just respect the parent constraints.

		size := MeasureWidget(child, cConstraints)

		// Track maximum width needed
		width := size.X + marginW
		if width > maxWidth {
			maxWidth = width
		}

		// Sum heights
		height := size.Y + params.MarginTop + params.MarginBottom
		totalHeight += height

		// Add spacing
		if i < len(children)-1 {
			totalHeight += vb.spacing
		}
	}

	return image.Point{X: maxWidth, Y: totalHeight}
}

// LayoutWithConstraints implements ConstraintLayoutManager.LayoutWithConstraints
func (vb *VBoxLayout) LayoutWithConstraints(children []ComposableWidget, constraints SizeConstraints, containerBounds image.Rectangle) {
	if len(children) == 0 {
		return
	}

	containerWidth := containerBounds.Dx()
	containerHeight := containerBounds.Dy()

	// Calculate total height needed and collect child sizes
	totalMinHeight := 0
	childSizes := make([]image.Point, len(children))
	totalGrow := 0

	// Child constraints: constrained width, unconstrained height
	childConstraints := constraints
	childConstraints.MinHeight = 0
	childConstraints.MaxHeight = 0

	for i, child := range children {
		var params LayoutParams
		if lp, ok := child.(interface{ GetLayoutParams() LayoutParams }); ok {
			params = lp.GetLayoutParams()
		} else {
			params = DefaultLayoutParams()
		}

		// Adjust constraints for margins
		cConstraints := childConstraints
		marginW := params.MarginLeft + params.MarginRight
		if cConstraints.HasMaxWidth() {
			cConstraints.MaxWidth -= marginW
			if cConstraints.MaxWidth < 0 {
				cConstraints.MaxWidth = 0
			}
		}

		// Measure child
		size := MeasureWidget(child, cConstraints)
		childSizes[i] = size

		// Account for margins
		height := size.Y + params.MarginTop + params.MarginBottom
		totalMinHeight += height
		totalGrow += params.Grow

		// Add spacing
		if i < len(children)-1 {
			totalMinHeight += vb.spacing
		}
	}

	// Calculate extra space to distribute
	extraSpace := containerHeight - totalMinHeight
	if extraSpace < 0 {
		extraSpace = 0
	}

	// Position children
	currentY := containerBounds.Min.Y

	for i, child := range children {
		var params LayoutParams
		if lp, ok := child.(interface{ GetLayoutParams() LayoutParams }); ok {
			params = lp.GetLayoutParams()
		} else {
			params = DefaultLayoutParams()
		}

		// Calculate child height
		childHeight := childSizes[i].Y

		// Distribute extra space if enabled
		if vb.distribute && extraSpace > 0 && totalGrow > 0 && params.Grow > 0 {
			extraHeight := (extraSpace * params.Grow) / totalGrow
			childHeight += extraHeight
		}

		// Apply margins
		currentY += params.MarginTop

		// Calculate child width based on alignment
		var childWidth int
		if vb.alignment == LayoutAlignStretch {
			// Stretch to fill available width
			childWidth = containerWidth - params.MarginLeft - params.MarginRight
		} else {
			// Use measured size, but clamp to available width
			maxWidth := containerWidth - params.MarginLeft - params.MarginRight
			childWidth = childSizes[i].X
			if childWidth > maxWidth {
				childWidth = maxWidth
			}
		}

		// Apply size constraints (from params)
		childSize := ApplyConstraints(
			image.Point{X: childWidth, Y: childHeight},
			params.Constraints,
		)

		// Calculate X position based on alignment
		childX := containerBounds.Min.X + params.MarginLeft

		switch vb.alignment {
		case LayoutAlignCenter:
			// Center horizontally
			childX = containerBounds.Min.X + (containerWidth-childSize.X)/2
		case LayoutAlignEnd:
			// Align to right
			childX = containerBounds.Max.X - childSize.X - params.MarginRight
		case LayoutAlignStretch:
			// Already positioned at left with full width
		default: // LayoutAlignStart
			// Already set to left alignment
		}

		// Set child bounds
		child.SetBounds(image.Rect(
			childX,
			currentY,
			childX+childSize.X,
			currentY+childSize.Y,
		))

		// Move to next position
		currentY += childSize.Y + params.MarginBottom + vb.spacing
	}
}

// HBoxLayout arranges children horizontally in a row, left to right.
// Each child gets its preferred/minimum width, and height fills the container.
type HBoxLayout struct {
	spacing    int             // Horizontal spacing between children in pixels
	alignment  LayoutAlignment // How to align children vertically
	distribute bool            // Whether to distribute extra space among children
}

// NewHBoxLayout creates a new horizontal box layout with the specified spacing
func NewHBoxLayout(spacing int) *HBoxLayout {
	return &HBoxLayout{
		spacing:    spacing,
		alignment:  LayoutAlignStart,
		distribute: false,
	}
}

// WithAlignment sets the vertical alignment for children
func (hb *HBoxLayout) WithAlignment(align LayoutAlignment) *HBoxLayout {
	hb.alignment = align
	return hb
}

// WithDistribute enables distributing extra horizontal space among children
func (hb *HBoxLayout) WithDistribute(distribute bool) *HBoxLayout {
	hb.distribute = distribute
	return hb
}

// Layout positions children horizontally within the container bounds
func (hb *HBoxLayout) Layout(containerBounds image.Rectangle, children []ComposableWidget) {
	if len(children) == 0 {
		return
	}

	containerWidth := containerBounds.Dx()
	containerHeight := containerBounds.Dy()

	// Calculate total width needed and collect child sizes
	totalMinWidth := 0
	childSizes := make([]image.Point, len(children))
	totalGrow := 0

	for i, child := range children {
		params := child.(interface{ GetLayoutParams() LayoutParams }).GetLayoutParams()

		// Get preferred size
		preferredSize := child.GetPreferredSize()
		childSizes[i] = preferredSize

		// Account for margins
		width := preferredSize.X + params.MarginLeft + params.MarginRight
		totalMinWidth += width
		totalGrow += params.Grow

		// Add spacing (except for last child)
		if i < len(children)-1 {
			totalMinWidth += hb.spacing
		}
	}

	// Calculate extra space to distribute
	extraSpace := containerWidth - totalMinWidth
	if extraSpace < 0 {
		extraSpace = 0
	}

	// Position children
	currentX := containerBounds.Min.X

	for i, child := range children {
		params := child.(interface{ GetLayoutParams() LayoutParams }).GetLayoutParams()

		// Calculate child width
		childWidth := childSizes[i].X

		// Distribute extra space if enabled
		if hb.distribute && extraSpace > 0 && totalGrow > 0 && params.Grow > 0 {
			extraWidth := (extraSpace * params.Grow) / totalGrow
			childWidth += extraWidth
		}

		// Apply margins
		currentX += params.MarginLeft

		// Calculate child height based on alignment
		var childHeight int
		if hb.alignment == LayoutAlignStretch {
			// Stretch to fill available height
			childHeight = containerHeight - params.MarginTop - params.MarginBottom
		} else {
			// Use preferred size, but clamp to available height
			maxHeight := containerHeight - params.MarginTop - params.MarginBottom
			childHeight = childSizes[i].Y
			if childHeight > maxHeight {
				childHeight = maxHeight
			}
		}

		// Apply size constraints
		childSize := ApplyConstraints(
			image.Point{X: childWidth, Y: childHeight},
			params.Constraints,
		)

		// Calculate Y position based on alignment
		childY := containerBounds.Min.Y + params.MarginTop

		switch hb.alignment {
		case LayoutAlignCenter:
			// Center vertically
			childY = containerBounds.Min.Y + (containerHeight-childSize.Y)/2
		case LayoutAlignEnd:
			// Align to bottom
			childY = containerBounds.Max.Y - childSize.Y - params.MarginBottom
		case LayoutAlignStretch:
			// Already positioned at top with full height
		default: // LayoutAlignStart
			// Already set to top alignment
		}

		// Set child bounds
		child.SetBounds(image.Rect(
			currentX,
			childY,
			currentX+childSize.X,
			childY+childSize.Y,
		))

		// Move to next position
		currentX += childSize.X + params.MarginRight + hb.spacing
	}
}

// CalculateMinSize returns the minimum size needed for all children
func (hb *HBoxLayout) CalculateMinSize(children []ComposableWidget) image.Point {
	if len(children) == 0 {
		return image.Point{X: 0, Y: 0}
	}

	totalWidth := 0
	maxHeight := 0

	for i, child := range children {
		params := child.(interface{ GetLayoutParams() LayoutParams }).GetLayoutParams()
		minSize := child.GetMinSize()

		// Sum widths
		width := minSize.X + params.MarginLeft + params.MarginRight
		totalWidth += width

		// Track maximum height needed
		height := minSize.Y + params.MarginTop + params.MarginBottom
		if height > maxHeight {
			maxHeight = height
		}

		// Add spacing (except for last child)
		if i < len(children)-1 {
			totalWidth += hb.spacing
		}
	}

	return image.Point{X: totalWidth, Y: maxHeight}
}

// CalculatePreferredSize returns the preferred size for all children
func (hb *HBoxLayout) CalculatePreferredSize(children []ComposableWidget) image.Point {
	if len(children) == 0 {
		return image.Point{X: 0, Y: 0}
	}

	totalWidth := 0
	maxHeight := 0

	for i, child := range children {
		params := child.(interface{ GetLayoutParams() LayoutParams }).GetLayoutParams()
		preferredSize := child.GetPreferredSize()

		// Sum widths
		width := preferredSize.X + params.MarginLeft + params.MarginRight
		totalWidth += width

		// Track maximum height needed
		height := preferredSize.Y + params.MarginTop + params.MarginBottom
		if height > maxHeight {
			maxHeight = height
		}

		// Add spacing (except for last child)
		if i < len(children)-1 {
			totalWidth += hb.spacing
		}
	}

	return image.Point{X: totalWidth, Y: maxHeight}
}

// Measure implements ConstraintLayoutManager.Measure
func (hb *HBoxLayout) Measure(children []ComposableWidget, constraints SizeConstraints) image.Point {
	if len(children) == 0 {
		return image.Point{X: 0, Y: 0}
	}

	totalWidth := 0
	maxHeight := 0

	// For HBox, we typically constrain height but let width grow
	childConstraints := constraints
	childConstraints.MinWidth = 0
	childConstraints.MaxWidth = 0 // Unconstrained width

	for i, child := range children {
		var params LayoutParams
		if lp, ok := child.(interface{ GetLayoutParams() LayoutParams }); ok {
			params = lp.GetLayoutParams()
		} else {
			params = DefaultLayoutParams()
		}

		// Adjust constraints for margins
		cConstraints := childConstraints
		marginH := params.MarginTop + params.MarginBottom
		if cConstraints.HasMaxHeight() {
			cConstraints.MaxHeight -= marginH
			if cConstraints.MaxHeight < 0 {
				cConstraints.MaxHeight = 0
			}
		}

		size := MeasureWidget(child, cConstraints)

		// Sum widths
		width := size.X + params.MarginLeft + params.MarginRight
		totalWidth += width

		// Track maximum height needed
		height := size.Y + marginH
		if height > maxHeight {
			maxHeight = height
		}

		// Add spacing
		if i < len(children)-1 {
			totalWidth += hb.spacing
		}
	}

	return image.Point{X: totalWidth, Y: maxHeight}
}

// LayoutWithConstraints implements ConstraintLayoutManager.LayoutWithConstraints
func (hb *HBoxLayout) LayoutWithConstraints(children []ComposableWidget, constraints SizeConstraints, containerBounds image.Rectangle) {
	if len(children) == 0 {
		return
	}

	containerWidth := containerBounds.Dx()
	containerHeight := containerBounds.Dy()

	// Calculate total width needed and collect child sizes
	totalMinWidth := 0
	childSizes := make([]image.Point, len(children))
	totalGrow := 0

	// Child constraints: constrained height, unconstrained width
	childConstraints := constraints
	childConstraints.MinWidth = 0
	childConstraints.MaxWidth = 0

	for i, child := range children {
		var params LayoutParams
		if lp, ok := child.(interface{ GetLayoutParams() LayoutParams }); ok {
			params = lp.GetLayoutParams()
		} else {
			params = DefaultLayoutParams()
		}

		// Adjust constraints for margins
		cConstraints := childConstraints
		marginH := params.MarginTop + params.MarginBottom
		if cConstraints.HasMaxHeight() {
			cConstraints.MaxHeight -= marginH
			if cConstraints.MaxHeight < 0 {
				cConstraints.MaxHeight = 0
			}
		}

		// Measure child
		size := MeasureWidget(child, cConstraints)
		childSizes[i] = size

		// Account for margins
		width := size.X + params.MarginLeft + params.MarginRight
		totalMinWidth += width
		totalGrow += params.Grow

		// Add spacing
		if i < len(children)-1 {
			totalMinWidth += hb.spacing
		}
	}

	// Calculate extra space to distribute
	extraSpace := containerWidth - totalMinWidth
	if extraSpace < 0 {
		extraSpace = 0
	}

	// Position children
	currentX := containerBounds.Min.X

	for i, child := range children {
		var params LayoutParams
		if lp, ok := child.(interface{ GetLayoutParams() LayoutParams }); ok {
			params = lp.GetLayoutParams()
		} else {
			params = DefaultLayoutParams()
		}

		// Calculate child width
		childWidth := childSizes[i].X

		// Distribute extra space if enabled
		if hb.distribute && extraSpace > 0 && totalGrow > 0 && params.Grow > 0 {
			extraWidth := (extraSpace * params.Grow) / totalGrow
			childWidth += extraWidth
		}

		// Apply margins
		currentX += params.MarginLeft

		// Calculate child height based on alignment
		var childHeight int
		if hb.alignment == LayoutAlignStretch {
			// Stretch to fill available height
			childHeight = containerHeight - params.MarginTop - params.MarginBottom
		} else {
			// Use measured size, but clamp to available height
			maxHeight := containerHeight - params.MarginTop - params.MarginBottom
			childHeight = childSizes[i].Y
			if childHeight > maxHeight {
				childHeight = maxHeight
			}
		}

		// Apply size constraints (from params)
		childSize := ApplyConstraints(
			image.Point{X: childWidth, Y: childHeight},
			params.Constraints,
		)

		// Calculate Y position based on alignment
		childY := containerBounds.Min.Y + params.MarginTop

		switch hb.alignment {
		case LayoutAlignCenter:
			// Center vertically
			childY = containerBounds.Min.Y + (containerHeight-childSize.Y)/2
		case LayoutAlignEnd:
			// Align to bottom
			childY = containerBounds.Max.Y - childSize.Y - params.MarginBottom
		case LayoutAlignStretch:
			// Already positioned at top with full height
		default: // LayoutAlignStart
			// Already set to top alignment
		}

		// Set child bounds
		child.SetBounds(image.Rect(
			currentX,
			childY,
			currentX+childSize.X,
			childY+childSize.Y,
		))

		// Move to next position
		currentX += childSize.X + params.MarginRight + hb.spacing
	}
}
