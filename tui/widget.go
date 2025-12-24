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

// Measurable interface allows widgets to participate in a constraint-based layout pass.
// This is the "first pass" of a Flutter-style layout system, where parents pass
// constraints down to children, and children return their desired size.
type Measurable interface {
	ComposableWidget
	// Measure calculates the desired size of the widget given the constraints.
	// The returned size should respect the constraints (be within min/max).
	Measure(constraints SizeConstraints) image.Point
}
