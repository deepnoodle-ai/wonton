package tui

import (
	"image"
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
