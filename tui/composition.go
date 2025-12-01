package tui

import (
	"image"
)

// Widget represents a UI widget that can be rendered and handle input.
type Widget interface {
	Draw(frame RenderFrame)
	HandleKey(event KeyEvent) bool
}

// ComposableWidget extends the basic Widget interface with composition capabilities.
// This interface enables bounds-based positioning, parent-child relationships,
// and lifecycle management necessary for complex, nested layouts.
//
// Components implementing this interface can be nested inside containers and
// will have their bounds managed by layout managers instead of using absolute positioning.
type ComposableWidget interface {
	Widget // Embed basic Widget interface (Draw, HandleKey)

	// Lifecycle methods
	Init()    // Called when widget is first created or added to parent
	Destroy() // Called when widget is removed or screen is closing

	// Bounds management - enables relative positioning within containers
	SetBounds(bounds image.Rectangle) // Parent/container sets the widget's position and size
	GetBounds() image.Rectangle       // Returns current bounds
	GetMinSize() image.Point          // Returns minimum size needed to render properly
	GetPreferredSize() image.Point    // Returns preferred size (may be larger than minimum)

	// Parent-child tree structure for event propagation
	GetParent() ComposableWidget // Returns parent widget (nil for root)
	SetParent(ComposableWidget)  // Sets parent widget

	// Visibility and layout participation
	IsVisible() bool   // Whether widget should be rendered and included in layout
	SetVisible(bool)   // Show/hide the widget
	NeedsRedraw() bool // Whether widget needs to be redrawn
	MarkDirty()        // Mark widget as needing redraw
	ClearDirty()       // Clear dirty flag after redraw
}

// MouseAware is an optional interface for widgets that handle mouse events
type MouseAware interface {
	HandleMouse(event MouseEvent) bool
}

// LayoutManager defines strategies for positioning and sizing child widgets within a container.
// Different implementations provide different layout behaviors (vertical stack, horizontal stack,
// grid, flexbox-style, etc.).
type LayoutManager interface {
	// Layout positions and sizes all children within the container's bounds.
	// The container bounds represent the available space for children.
	// This method should call SetBounds() on each child widget.
	Layout(containerBounds image.Rectangle, children []ComposableWidget)

	// CalculateMinSize determines the minimum size needed to layout all children.
	// This enables containers to know their own minimum size based on their contents.
	CalculateMinSize(children []ComposableWidget) image.Point

	// CalculatePreferredSize determines the preferred size for the container.
	// This may be larger than minimum if children want more space.
	CalculatePreferredSize(children []ComposableWidget) image.Point
}

// LayoutAlignment specifies how content should be aligned within available space
type LayoutAlignment int

const (
	LayoutAlignStart   LayoutAlignment = iota // Align to top (vertical) or left (horizontal)
	LayoutAlignCenter                         // Center content
	LayoutAlignEnd                            // Align to bottom (vertical) or right (horizontal)
	LayoutAlignStretch                        // Stretch to fill available space
)

// SizeConstraints defines minimum and maximum size constraints for a widget.
// Zero values for MaxWidth/MaxHeight indicate no maximum constraint (unconstrained).
type SizeConstraints struct {
	MinWidth  int
	MinHeight int
	MaxWidth  int // 0 means no maximum
	MaxHeight int // 0 means no maximum
}

// HasMaxWidth returns true if there is a maximum width constraint
func (sc SizeConstraints) HasMaxWidth() bool {
	return sc.MaxWidth > 0
}

// HasMaxHeight returns true if there is a maximum height constraint
func (sc SizeConstraints) HasMaxHeight() bool {
	return sc.MaxHeight > 0
}

// IsTight returns true if min and max are equal (and max > 0)
func (sc SizeConstraints) IsTight() bool {
	return sc.MinWidth == sc.MaxWidth && sc.MinHeight == sc.MaxHeight && sc.MaxWidth > 0 && sc.MaxHeight > 0
}

// Constrain applies the constraints to a given size
func (sc SizeConstraints) Constrain(size image.Point) image.Point {
	return ApplyConstraints(size, sc)
}

// Measurable interface allows widgets to participate in a constraint-based layout pass.
// This is the "first pass" of a Flutter-style layout system, where parents pass
// constraints down to children, and children return their desired size.
type Measurable interface {
	ComposableWidget
	// Measure calculates the desired size of the widget given the constraints.
	// The returned size should respect the constraints (be within min/max).
	Measure(constraints SizeConstraints) image.Point
}

// ConstraintLayoutManager is an enhanced LayoutManager that supports constraint-based layout.
type ConstraintLayoutManager interface {
	LayoutManager // Embed legacy interface for compatibility

	// Measure calculates the size of the container given the constraints.
	// It should also measure children recursively.
	Measure(children []ComposableWidget, constraints SizeConstraints) image.Point

	// LayoutWithConstraints positions children.
	// It is similar to Layout but provides the constraints used during measurement.
	LayoutWithConstraints(children []ComposableWidget, constraints SizeConstraints, containerBounds image.Rectangle)
}

// MeasureWidget is a helper to measure a widget whether it implements Measurable or not.
// If the widget is Measurable, it calls Measure.
// Otherwise, it calls GetPreferredSize and applies the constraints.
func MeasureWidget(w ComposableWidget, c SizeConstraints) image.Point {
	if m, ok := w.(Measurable); ok {
		return m.Measure(c)
	}
	// Legacy fallback
	return ApplyConstraints(w.GetPreferredSize(), c)
}

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

// BaseWidget provides a default implementation of ComposableWidget that other
// widgets can embed to get composition support. This handles the boilerplate
// of bounds management, parent tracking, and visibility.
type BaseWidget struct {
	bounds        image.Rectangle
	parent        ComposableWidget
	visible       bool
	dirty         bool
	minSize       image.Point
	preferredSize image.Point
	layoutParams  LayoutParams
}

// NewBaseWidget creates a new BaseWidget with sensible defaults
func NewBaseWidget() BaseWidget {
	return BaseWidget{
		bounds:        image.Rectangle{},
		parent:        nil,
		visible:       true,
		dirty:         true,
		minSize:       image.Point{X: 0, Y: 0},
		preferredSize: image.Point{X: 0, Y: 0},
		layoutParams:  DefaultLayoutParams(),
	}
}

// Init initializes the widget (default: no-op)
func (bw *BaseWidget) Init() {}

// Destroy cleans up the widget (default: no-op)
func (bw *BaseWidget) Destroy() {}

// SetBounds sets the widget's position and size
func (bw *BaseWidget) SetBounds(bounds image.Rectangle) {
	if bw.bounds != bounds {
		bw.bounds = bounds
		bw.MarkDirty()
	}
}

// GetBounds returns the widget's current bounds
func (bw *BaseWidget) GetBounds() image.Rectangle {
	return bw.bounds
}

// GetMinSize returns the minimum size (can be overridden)
func (bw *BaseWidget) GetMinSize() image.Point {
	return bw.minSize
}

// SetMinSize sets the minimum size
func (bw *BaseWidget) SetMinSize(size image.Point) {
	bw.minSize = size
}

// GetPreferredSize returns the preferred size (can be overridden)
func (bw *BaseWidget) GetPreferredSize() image.Point {
	if bw.preferredSize.X > 0 || bw.preferredSize.Y > 0 {
		return bw.preferredSize
	}
	return bw.minSize
}

// SetPreferredSize sets the preferred size
func (bw *BaseWidget) SetPreferredSize(size image.Point) {
	bw.preferredSize = size
}

// GetParent returns the parent widget
func (bw *BaseWidget) GetParent() ComposableWidget {
	return bw.parent
}

// SetParent sets the parent widget
func (bw *BaseWidget) SetParent(parent ComposableWidget) {
	bw.parent = parent
}

// IsVisible returns whether the widget is visible
func (bw *BaseWidget) IsVisible() bool {
	return bw.visible
}

// SetVisible sets the widget's visibility
func (bw *BaseWidget) SetVisible(visible bool) {
	if bw.visible != visible {
		bw.visible = visible
		bw.MarkDirty()
		if bw.parent != nil {
			bw.parent.MarkDirty()
		}
	}
}

// NeedsRedraw returns whether the widget needs to be redrawn
func (bw *BaseWidget) NeedsRedraw() bool {
	return bw.dirty
}

// MarkDirty marks the widget as needing redraw
func (bw *BaseWidget) MarkDirty() {
	bw.dirty = true
	// Propagate dirty flag up the tree
	if bw.parent != nil {
		bw.parent.MarkDirty()
	}
}

// ClearDirty clears the dirty flag
func (bw *BaseWidget) ClearDirty() {
	bw.dirty = false
}

// GetLayoutParams returns the layout parameters
func (bw *BaseWidget) GetLayoutParams() LayoutParams {
	return bw.layoutParams
}

// SetLayoutParams sets the layout parameters
func (bw *BaseWidget) SetLayoutParams(params LayoutParams) {
	bw.layoutParams = params
	bw.MarkDirty()
	if bw.parent != nil {
		bw.parent.MarkDirty()
	}
}

// ApplyConstraints applies size constraints to a size, returning the constrained size
func ApplyConstraints(size image.Point, constraints SizeConstraints) image.Point {
	result := size

	// Apply minimum constraints
	if result.X < constraints.MinWidth {
		result.X = constraints.MinWidth
	}
	if result.Y < constraints.MinHeight {
		result.Y = constraints.MinHeight
	}

	// Apply maximum constraints
	if constraints.MaxWidth > 0 && result.X > constraints.MaxWidth {
		result.X = constraints.MaxWidth
	}
	if constraints.MaxHeight > 0 && result.Y > constraints.MaxHeight {
		result.Y = constraints.MaxHeight
	}

	return result
}
