package tui

// LayoutParams holds layout-specific parameters that widgets can use to influence
// how their parent container positions them.
type LayoutParams struct {
	// Flex/grow factor - how much extra space this widget should take relative to siblings
	// 0 means don't grow, 1 means grow proportionally with others, 2 means grow twice as much, etc.
	Grow int

	// Shrink factor - how much this widget should shrink when space is constrained
	// 0 means don't shrink, 1 means shrink proportionally
	Shrink int

	// Alignment within the container
	Align LayoutAlignment

	// Margin around the widget (space outside the widget)
	MarginTop    int
	MarginRight  int
	MarginBottom int
	MarginLeft   int

	// Padding inside the widget (space inside borders, outside content)
	PaddingTop    int
	PaddingRight  int
	PaddingBottom int
	PaddingLeft   int

	// Size constraints
	Constraints SizeConstraints
}

// DefaultLayoutParams returns sensible defaults for layout parameters
func DefaultLayoutParams() LayoutParams {
	return LayoutParams{
		Grow:   0,
		Shrink: 1,
		Align:  LayoutAlignStart,
	}
}
