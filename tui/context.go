package tui

import "image"

// RenderContext provides drawing capabilities and contextual information to views.
// It flows through the view tree during rendering, giving views access to:
//   - Drawing operations (via the embedded RenderFrame)
//   - Animation frame counter
//   - Focus management
//
// Views should use SubContext() when rendering children to properly scope
// the drawing area while preserving context information.
type RenderContext struct {
	frame      RenderFrame
	frameCount uint64
	bounds     image.Rectangle
	focusMgr   *FocusManager
}

// NewRenderContext creates a new render context.
// This is typically called by the Runtime at the start of each render cycle.
func NewRenderContext(frame RenderFrame, frameCount uint64) *RenderContext {
	w, h := frame.Size()
	return &RenderContext{
		frame:      frame,
		frameCount: frameCount,
		bounds:     image.Rect(0, 0, w, h),
	}
}

// WithFocusManager returns a new context with the given focus manager.
func (c *RenderContext) WithFocusManager(fm *FocusManager) *RenderContext {
	return &RenderContext{
		frame:      c.frame,
		frameCount: c.frameCount,
		bounds:     c.bounds,
		focusMgr:   fm,
	}
}

// FocusManager returns the focus manager for this context, or nil if none.
func (c *RenderContext) FocusManager() *FocusManager {
	return c.focusMgr
}

// Frame returns the current animation frame counter.
// Use this for time-based animations - it increments each tick (typically 30-60 FPS).
func (c *RenderContext) Frame() uint64 {
	return c.frameCount
}

// RenderFrame returns the underlying RenderFrame.
// This is useful for views that need to create custom frame wrappers.
func (c *RenderContext) RenderFrame() RenderFrame {
	return c.frame
}

// Bounds returns the current drawing bounds.
func (c *RenderContext) Bounds() image.Rectangle {
	return c.bounds
}

// Size returns the width and height of the current drawing area.
func (c *RenderContext) Size() (width, height int) {
	return c.bounds.Dx(), c.bounds.Dy()
}

// SubContext creates a child context with a sub-region of the drawing area.
// The bounds are relative to the current context's origin.
// All context information (frame counter, focus manager, etc.) is preserved.
func (c *RenderContext) SubContext(bounds image.Rectangle) *RenderContext {
	// Translate bounds to be relative to current context's bounds
	absoluteBounds := bounds.Add(c.bounds.Min)
	// Clip to current bounds
	clippedBounds := absoluteBounds.Intersect(c.bounds)

	return &RenderContext{
		frame:      c.frame.SubFrame(clippedBounds),
		frameCount: c.frameCount,
		bounds:     image.Rect(0, 0, clippedBounds.Dx(), clippedBounds.Dy()),
		focusMgr:   c.focusMgr,
	}
}

// Drawing methods - delegate to the underlying RenderFrame

// SetCell sets a character at the given position with a style.
func (c *RenderContext) SetCell(x, y int, char rune, style Style) {
	c.frame.SetCell(x, y, char, style)
}

// PrintStyled prints text at the given position with a style.
// Text wraps at the frame edge.
func (c *RenderContext) PrintStyled(x, y int, text string, style Style) {
	c.frame.PrintStyled(x, y, text, style)
}

// PrintTruncated prints text at the given position, truncating at the frame edge.
func (c *RenderContext) PrintTruncated(x, y int, text string, style Style) {
	c.frame.PrintTruncated(x, y, text, style)
}

// FillStyled fills a rectangular area with a character and style.
func (c *RenderContext) FillStyled(x, y, width, height int, char rune, style Style) {
	c.frame.FillStyled(x, y, width, height, char, style)
}

// Fill fills the entire context area with a character and style.
func (c *RenderContext) Fill(char rune, style Style) {
	c.frame.Fill(char, style)
}

// PrintHyperlink prints a clickable hyperlink.
func (c *RenderContext) PrintHyperlink(x, y int, link Hyperlink) {
	c.frame.PrintHyperlink(x, y, link)
}

// PrintHyperlinkFallback prints a hyperlink in fallback format: "Text (URL)".
func (c *RenderContext) PrintHyperlinkFallback(x, y int, link Hyperlink) {
	c.frame.PrintHyperlinkFallback(x, y, link)
}

// AbsoluteBounds returns the absolute screen bounds of this context.
// Use this for registering interactive regions (click handlers, etc.).
func (c *RenderContext) AbsoluteBounds() image.Rectangle {
	return c.frame.GetBounds()
}

// WithFrame creates a new context using a different RenderFrame but preserving
// the frame counter and focus manager. This is useful for views that need
// custom frame wrappers (like scroll views).
func (c *RenderContext) WithFrame(frame RenderFrame) *RenderContext {
	w, h := frame.Size()
	return &RenderContext{
		frame:      frame,
		frameCount: c.frameCount,
		bounds:     image.Rect(0, 0, w, h),
		focusMgr:   c.focusMgr,
	}
}
