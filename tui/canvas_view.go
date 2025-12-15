package tui

import "image"

// canvasView allows imperative drawing within the declarative tree
type canvasView struct {
	draw        func(frame RenderFrame, bounds image.Rectangle)
	drawContext func(ctx *RenderContext)
	width       int
	height      int
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

// CanvasContext creates a view with access to the full RenderContext.
// Use this when you need access to context information like the animation
// frame counter (ctx.Frame()) for custom animations.
//
// Example:
//
//	CanvasContext(func(ctx *RenderContext) {
//	    w, h := ctx.Size()
//	    frame := ctx.Frame()  // Animation frame counter
//	    // Custom drawing using frame for animation timing
//	    x := int(frame) % w
//	    ctx.SetCell(x, 0, '█', style)
//	})
func CanvasContext(draw func(ctx *RenderContext)) *canvasView {
	return &canvasView{
		drawContext: draw,
		width:       0,
		height:      0,
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

func (c *canvasView) render(ctx *RenderContext) {
	w, h := ctx.Size()
	if w == 0 || h == 0 {
		return
	}

	// Use context-aware draw function if provided
	if c.drawContext != nil {
		c.drawContext(ctx)
		return
	}

	// Fall back to legacy draw function
	if c.draw != nil {
		// The bounds passed to the draw function are relative to the context
		c.draw(ctx.frame, image.Rect(0, 0, w, h))
	}
}

// Canvas is flexible - it can expand to fill available space
func (c *canvasView) flex() int {
	// Only flexible if no explicit size is set
	if c.width == 0 && c.height == 0 {
		return 1
	}
	return 0
}
