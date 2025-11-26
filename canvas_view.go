package gooey

import "image"

// canvasView allows imperative drawing within the declarative tree
type canvasView struct {
	draw   func(frame RenderFrame, bounds image.Rectangle)
	width  int
	height int
}

// Canvas creates a view that calls a custom draw function.
// This is the escape hatch for imperative drawing within declarative UI.
//
// Example:
//
//	Canvas(func(frame RenderFrame, bounds image.Rectangle) {
//	    // Custom imperative drawing
//	    frame.PrintStyled(0, 0, "Custom content", style)
//	})
func Canvas(draw func(frame RenderFrame, bounds image.Rectangle)) *canvasView {
	return &canvasView{
		draw:   draw,
		width:  0,
		height: 0,
	}
}

// Size sets the preferred size for the canvas.
// Without this, the canvas will use all available space.
func (c *canvasView) Size(w, h int) *canvasView {
	c.width = w
	c.height = h
	return c
}

// Width sets the preferred width for the canvas.
func (c *canvasView) Width(w int) *canvasView {
	c.width = w
	return c
}

// Height sets the preferred height for the canvas.
func (c *canvasView) Height(h int) *canvasView {
	c.height = h
	return c
}

func (c *canvasView) size(maxWidth, maxHeight int) (int, int) {
	w := c.width
	h := c.height

	// Use max available space if not specified
	if w == 0 {
		w = maxWidth
	}
	if h == 0 {
		h = maxHeight
	}

	// Clamp to constraints
	if maxWidth > 0 && w > maxWidth {
		w = maxWidth
	}
	if maxHeight > 0 && h > maxHeight {
		h = maxHeight
	}

	return w, h
}

func (c *canvasView) render(frame RenderFrame, bounds image.Rectangle) {
	if bounds.Empty() || c.draw == nil {
		return
	}

	// Create a subframe for the canvas bounds
	subFrame := frame.SubFrame(bounds)

	// Call the custom draw function with the subframe
	// The bounds passed to the draw function are relative to the subframe
	c.draw(subFrame, image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
}

// Canvas is flexible - it can expand to fill available space
func (c *canvasView) flex() int {
	// Only flexible if no explicit size is set
	if c.width == 0 && c.height == 0 {
		return 1
	}
	return 0
}
