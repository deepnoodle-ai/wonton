package gooey

import (
	"image"
)

// FlexDirection specifies the direction in which flex items are placed
type FlexDirection int

const (
	FlexRow    FlexDirection = iota // Horizontal, left to right
	FlexColumn                      // Vertical, top to bottom
)

// FlexWrap specifies whether flex items wrap to new lines
type FlexWrap int

const (
	FlexNoWrap FlexWrap = iota // All items on one line
	FlexWrapOn                 // Items wrap to new lines as needed
)

// FlexJustify specifies how items are distributed along main axis
type FlexJustify int

const (
	FlexJustifyStart        FlexJustify = iota // Items packed to start
	FlexJustifyEnd                             // Items packed to end
	FlexJustifyCenter                          // Items centered
	FlexJustifySpaceBetween                    // Even spacing between items
	FlexJustifySpaceAround                     // Even spacing around items
	FlexJustifySpaceEvenly                     // Equal spacing between and around items
)

// FlexAlignItems specifies how items are aligned along cross axis
type FlexAlignItems int

const (
	FlexAlignItemsStart   FlexAlignItems = iota // Items aligned to start of cross axis
	FlexAlignItemsEnd                           // Items aligned to end of cross axis
	FlexAlignItemsCenter                        // Items centered on cross axis
	FlexAlignItemsStretch                       // Items stretched to fill cross axis
)

// FlexLayout implements a flexbox-style layout manager with support for
// flexible sizing, wrapping, and sophisticated alignment options.
//
// This provides capabilities similar to CSS Flexbox:
// - Flexible item sizing with grow/shrink factors
// - Multiple justification options (start, end, center, space-between, etc.)
// - Cross-axis alignment
// - Support for row and column directions
// - Optional wrapping
type FlexLayout struct {
	direction   FlexDirection  // Main axis direction
	wrap        FlexWrap       // Whether to wrap items
	justify     FlexJustify    // Main axis justification
	alignItems  FlexAlignItems // Cross axis alignment
	spacing     int            // Spacing between items
	lineSpacing int            // Spacing between wrapped lines
}

// NewFlexLayout creates a new flex layout with default settings
func NewFlexLayout() *FlexLayout {
	return &FlexLayout{
		direction:   FlexRow,
		wrap:        FlexNoWrap,
		justify:     FlexJustifyStart,
		alignItems:  FlexAlignItemsStretch,
		spacing:     0,
		lineSpacing: 0,
	}
}

// WithDirection sets the flex direction
func (fl *FlexLayout) WithDirection(direction FlexDirection) *FlexLayout {
	fl.direction = direction
	return fl
}

// WithWrap sets the flex wrap behavior
func (fl *FlexLayout) WithWrap(wrap FlexWrap) *FlexLayout {
	fl.wrap = wrap
	return fl
}

// WithJustify sets the main axis justification
func (fl *FlexLayout) WithJustify(justify FlexJustify) *FlexLayout {
	fl.justify = justify
	return fl
}

// WithAlignItems sets the cross axis alignment
func (fl *FlexLayout) WithAlignItems(alignItems FlexAlignItems) *FlexLayout {
	fl.alignItems = alignItems
	return fl
}

// WithSpacing sets the spacing between items
func (fl *FlexLayout) WithSpacing(spacing int) *FlexLayout {
	fl.spacing = spacing
	return fl
}

// WithLineSpacing sets the spacing between wrapped lines
func (fl *FlexLayout) WithLineSpacing(lineSpacing int) *FlexLayout {
	fl.lineSpacing = lineSpacing
	return fl
}

// flexItem represents an item being laid out with its sizing information
type flexItem struct {
	widget     ComposableWidget
	params     LayoutParams
	baseSize   image.Point // Preferred/minimum size
	flexGrow   int
	flexShrink int
	mainSize   int // Size along main axis after flex calculation
	crossSize  int // Size along cross axis
}

// Layout positions children using flexbox algorithm
func (fl *FlexLayout) Layout(containerBounds image.Rectangle, children []ComposableWidget) {
	if len(children) == 0 {
		return
	}

	// Determine main and cross axis dimensions
	var containerMain, containerCross int
	if fl.direction == FlexRow {
		containerMain = containerBounds.Dx()
		containerCross = containerBounds.Dy()
	} else {
		containerMain = containerBounds.Dy()
		containerCross = containerBounds.Dx()
	}

	// Prepare flex items
	items := make([]flexItem, len(children))
	for i, child := range children {
		params := child.(interface{ GetLayoutParams() LayoutParams }).GetLayoutParams()
		preferredSize := child.GetPreferredSize()

		items[i] = flexItem{
			widget:     child,
			params:     params,
			baseSize:   preferredSize,
			flexGrow:   params.Grow,
			flexShrink: params.Shrink,
		}
	}

	// Layout items (with or without wrapping)
	if fl.wrap == FlexWrapOn {
		fl.layoutWithWrap(containerBounds, items, containerMain, containerCross)
	} else {
		fl.layoutSingleLine(containerBounds, items, containerMain, containerCross)
	}
}

// layoutSingleLine lays out all items on a single line
func (fl *FlexLayout) layoutSingleLine(containerBounds image.Rectangle, items []flexItem, containerMain, containerCross int) {
	// Calculate total base size and flex factors
	totalBaseSize := 0
	totalGrow := 0
	totalShrink := 0

	for i := range items {
		item := &items[i]

		// Get size along main axis
		if fl.direction == FlexRow {
			item.mainSize = item.baseSize.X + item.params.MarginLeft + item.params.MarginRight
			item.crossSize = item.baseSize.Y + item.params.MarginTop + item.params.MarginBottom
		} else {
			item.mainSize = item.baseSize.Y + item.params.MarginTop + item.params.MarginBottom
			item.crossSize = item.baseSize.X + item.params.MarginLeft + item.params.MarginRight
		}

		totalBaseSize += item.mainSize
		totalGrow += item.flexGrow
		totalShrink += item.flexShrink
	}

	// Add spacing
	totalBaseSize += fl.spacing * (len(items) - 1)

	// Calculate remaining space
	remainingSpace := containerMain - totalBaseSize

	// Distribute remaining space based on flex grow/shrink
	if remainingSpace > 0 && totalGrow > 0 {
		// Grow items
		for i := range items {
			item := &items[i]
			if item.flexGrow > 0 {
				extraSize := (remainingSpace * item.flexGrow) / totalGrow
				item.mainSize += extraSize
			}
		}
	} else if remainingSpace < 0 && totalShrink > 0 {
		// Shrink items
		shrinkAmount := -remainingSpace
		for i := range items {
			item := &items[i]
			if item.flexShrink > 0 {
				shrinkSize := (shrinkAmount * item.flexShrink) / totalShrink
				item.mainSize -= shrinkSize
				if item.mainSize < 0 {
					item.mainSize = 0
				}
			}
		}
	}

	// Position items based on justify content
	mainPos := fl.calculateStartPosition(items, containerMain)
	spacing := fl.calculateSpacing(items, containerMain)

	for i := range items {
		item := &items[i]

		// Calculate cross axis position and size
		crossPos, crossSize := fl.calculateCrossPosition(item, containerCross)

		// Apply margins
		var x, y, width, height int
		if fl.direction == FlexRow {
			x = containerBounds.Min.X + mainPos + item.params.MarginLeft
			y = containerBounds.Min.Y + crossPos + item.params.MarginTop
			width = item.mainSize - item.params.MarginLeft - item.params.MarginRight
			height = crossSize - item.params.MarginTop - item.params.MarginBottom
		} else {
			x = containerBounds.Min.X + crossPos + item.params.MarginLeft
			y = containerBounds.Min.Y + mainPos + item.params.MarginTop
			width = crossSize - item.params.MarginLeft - item.params.MarginRight
			height = item.mainSize - item.params.MarginTop - item.params.MarginBottom
		}

		// Apply constraints
		finalSize := ApplyConstraints(
			image.Point{X: width, Y: height},
			item.params.Constraints,
		)

		item.widget.SetBounds(image.Rect(x, y, x+finalSize.X, y+finalSize.Y))

		// Move to next position
		mainPos += item.mainSize + spacing
	}
}

// layoutWithWrap lays out items with wrapping to multiple lines
func (fl *FlexLayout) layoutWithWrap(containerBounds image.Rectangle, items []flexItem, containerMain, containerCross int) {
	// Split items into lines
	lines := fl.splitIntoLines(items, containerMain)

	// Calculate total cross size needed
	totalCrossSize := 0
	for i, line := range lines {
		maxCross := 0
		for _, item := range line {
			if item.crossSize > maxCross {
				maxCross = item.crossSize
			}
		}
		totalCrossSize += maxCross
		if i < len(lines)-1 {
			totalCrossSize += fl.lineSpacing
		}
	}

	// Position each line
	crossPos := containerBounds.Min.Y
	if fl.direction == FlexRow {
		crossPos = containerBounds.Min.Y
	} else {
		crossPos = containerBounds.Min.X
	}

	for _, line := range lines {
		// Calculate line height/width
		maxCross := 0
		for _, item := range line {
			if item.crossSize > maxCross {
				maxCross = item.crossSize
			}
		}

		// Layout this line
		lineItems := make([]flexItem, len(line))
		copy(lineItems, line)

		lineBounds := containerBounds
		if fl.direction == FlexRow {
			lineBounds = image.Rect(
				containerBounds.Min.X,
				crossPos,
				containerBounds.Max.X,
				crossPos+maxCross,
			)
		} else {
			lineBounds = image.Rect(
				crossPos,
				containerBounds.Min.Y,
				crossPos+maxCross,
				containerBounds.Max.Y,
			)
		}

		fl.layoutSingleLine(lineBounds, lineItems, containerMain, maxCross)

		crossPos += maxCross + fl.lineSpacing
	}
}

// splitIntoLines splits items into lines for wrapping
func (fl *FlexLayout) splitIntoLines(items []flexItem, containerMain int) [][]flexItem {
	lines := make([][]flexItem, 0)
	currentLine := make([]flexItem, 0)
	currentLineSize := 0

	for _, item := range items {
		itemSize := item.mainSize
		if len(currentLine) > 0 {
			itemSize += fl.spacing
		}

		if currentLineSize+itemSize > containerMain && len(currentLine) > 0 {
			// Start new line
			lines = append(lines, currentLine)
			currentLine = make([]flexItem, 0)
			currentLineSize = 0
		}

		currentLine = append(currentLine, item)
		currentLineSize += itemSize
	}

	if len(currentLine) > 0 {
		lines = append(lines, currentLine)
	}

	return lines
}

// calculateStartPosition calculates the starting position based on justify content
func (fl *FlexLayout) calculateStartPosition(items []flexItem, containerMain int) int {
	switch fl.justify {
	case FlexJustifyEnd:
		totalSize := 0
		for _, item := range items {
			totalSize += item.mainSize
		}
		totalSize += fl.spacing * (len(items) - 1)
		return containerMain - totalSize

	case FlexJustifyCenter:
		totalSize := 0
		for _, item := range items {
			totalSize += item.mainSize
		}
		totalSize += fl.spacing * (len(items) - 1)
		return (containerMain - totalSize) / 2

	default:
		return 0
	}
}

// calculateSpacing calculates spacing between items based on justify content
func (fl *FlexLayout) calculateSpacing(items []flexItem, containerMain int) int {
	if fl.justify == FlexJustifySpaceBetween && len(items) > 1 {
		totalSize := 0
		for _, item := range items {
			totalSize += item.mainSize
		}
		return (containerMain - totalSize) / (len(items) - 1)
	}

	if fl.justify == FlexJustifySpaceAround && len(items) > 0 {
		totalSize := 0
		for _, item := range items {
			totalSize += item.mainSize
		}
		return (containerMain - totalSize) / len(items)
	}

	if fl.justify == FlexJustifySpaceEvenly && len(items) > 0 {
		totalSize := 0
		for _, item := range items {
			totalSize += item.mainSize
		}
		return (containerMain - totalSize) / (len(items) + 1)
	}

	return fl.spacing
}

// calculateCrossPosition calculates cross axis position and size for an item
func (fl *FlexLayout) calculateCrossPosition(item *flexItem, containerCross int) (int, int) {
	crossSize := item.crossSize

	switch fl.alignItems {
	case FlexAlignItemsEnd:
		return containerCross - crossSize, crossSize
	case FlexAlignItemsCenter:
		return (containerCross - crossSize) / 2, crossSize
	case FlexAlignItemsStretch:
		return 0, containerCross
	default: // FlexAlignItemsStart
		return 0, crossSize
	}
}

// CalculateMinSize returns minimum size needed for flex layout
func (fl *FlexLayout) CalculateMinSize(children []ComposableWidget) image.Point {
	if len(children) == 0 {
		return image.Point{X: 0, Y: 0}
	}

	if fl.direction == FlexRow {
		return fl.calculateMinSizeRow(children)
	}
	return fl.calculateMinSizeColumn(children)
}

// calculateMinSizeRow calculates minimum size for row direction
func (fl *FlexLayout) calculateMinSizeRow(children []ComposableWidget) image.Point {
	totalWidth := 0
	maxHeight := 0

	for i, child := range children {
		params := child.(interface{ GetLayoutParams() LayoutParams }).GetLayoutParams()
		minSize := child.GetMinSize()

		width := minSize.X + params.MarginLeft + params.MarginRight
		totalWidth += width

		height := minSize.Y + params.MarginTop + params.MarginBottom
		if height > maxHeight {
			maxHeight = height
		}

		if i < len(children)-1 {
			totalWidth += fl.spacing
		}
	}

	return image.Point{X: totalWidth, Y: maxHeight}
}

// calculateMinSizeColumn calculates minimum size for column direction
func (fl *FlexLayout) calculateMinSizeColumn(children []ComposableWidget) image.Point {
	maxWidth := 0
	totalHeight := 0

	for i, child := range children {
		params := child.(interface{ GetLayoutParams() LayoutParams }).GetLayoutParams()
		minSize := child.GetMinSize()

		width := minSize.X + params.MarginLeft + params.MarginRight
		if width > maxWidth {
			maxWidth = width
		}

		height := minSize.Y + params.MarginTop + params.MarginBottom
		totalHeight += height

		if i < len(children)-1 {
			totalHeight += fl.spacing
		}
	}

	return image.Point{X: maxWidth, Y: totalHeight}
}

// CalculatePreferredSize returns preferred size for flex layout
func (fl *FlexLayout) CalculatePreferredSize(children []ComposableWidget) image.Point {
	if len(children) == 0 {
		return image.Point{X: 0, Y: 0}
	}

	if fl.direction == FlexRow {
		return fl.calculatePreferredSizeRow(children)
	}
	return fl.calculatePreferredSizeColumn(children)
}

// calculatePreferredSizeRow calculates preferred size for row direction
func (fl *FlexLayout) calculatePreferredSizeRow(children []ComposableWidget) image.Point {
	totalWidth := 0
	maxHeight := 0

	for i, child := range children {
		params := child.(interface{ GetLayoutParams() LayoutParams }).GetLayoutParams()
		preferredSize := child.GetPreferredSize()

		width := preferredSize.X + params.MarginLeft + params.MarginRight
		totalWidth += width

		height := preferredSize.Y + params.MarginTop + params.MarginBottom
		if height > maxHeight {
			maxHeight = height
		}

		if i < len(children)-1 {
			totalWidth += fl.spacing
		}
	}

	return image.Point{X: totalWidth, Y: maxHeight}
}

// calculatePreferredSizeColumn calculates preferred size for column direction
func (fl *FlexLayout) calculatePreferredSizeColumn(children []ComposableWidget) image.Point {
	maxWidth := 0
	totalHeight := 0

	for i, child := range children {
		params := child.(interface{ GetLayoutParams() LayoutParams }).GetLayoutParams()
		preferredSize := child.GetPreferredSize()

		width := preferredSize.X + params.MarginLeft + params.MarginRight
		if width > maxWidth {
			maxWidth = width
		}

		height := preferredSize.Y + params.MarginTop + params.MarginBottom
		totalHeight += height

		if i < len(children)-1 {
			totalHeight += fl.spacing
		}
	}

	return image.Point{X: maxWidth, Y: totalHeight}
}
