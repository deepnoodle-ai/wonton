package tui

import (
	"image"
)

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
