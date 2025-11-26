package gooey

import (
	"image"
	"sync"
)

// InteractiveRegistry tracks clickable regions for mouse event routing.
// It is cleared before each render and populated as views are drawn.
var interactiveRegistry = &interactiveRegistryImpl{
	buttons: make([]buttonRegion, 0),
}

type interactiveRegistryImpl struct {
	mu      sync.Mutex
	buttons []buttonRegion
}

type buttonRegion struct {
	bounds   image.Rectangle
	callback func()
}

// Clear clears all registered interactive regions.
// Called by the runtime before each render.
func (r *interactiveRegistryImpl) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.buttons = r.buttons[:0]
}

// RegisterButton adds a clickable region.
func (r *interactiveRegistryImpl) RegisterButton(bounds image.Rectangle, callback func()) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.buttons = append(r.buttons, buttonRegion{bounds: bounds, callback: callback})
}

// HandleClick checks if a click hit any registered button and invokes its callback.
// Returns true if a button was clicked.
func (r *interactiveRegistryImpl) HandleClick(x, y int) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	pt := image.Pt(x, y)
	for _, btn := range r.buttons {
		if pt.In(btn.bounds) {
			// Call the callback outside the lock
			callback := btn.callback
			r.mu.Unlock()
			callback()
			r.mu.Lock()
			return true
		}
	}
	return false
}

// clickableView displays an interactive clickable element
type clickableView struct {
	label    string
	callback func()
	style    Style
	width    int
}

// Clickable creates an interactive clickable element.
// The callback is invoked when the element is clicked.
//
// Example:
//
//	Clickable("Submit", func() { app.submit() })
func Clickable(label string, callback func()) *clickableView {
	return &clickableView{
		label:    label,
		callback: callback,
		style:    NewStyle(),
		width:    0,
	}
}

// Fg sets the foreground color.
func (c *clickableView) Fg(col Color) *clickableView {
	c.style = c.style.WithForeground(col)
	return c
}

// Bg sets the background color.
func (c *clickableView) Bg(col Color) *clickableView {
	c.style = c.style.WithBackground(col)
	return c
}

// Bold enables bold text.
func (c *clickableView) Bold() *clickableView {
	c.style = c.style.WithBold()
	return c
}

// Reverse enables reverse video (useful for selected state).
func (c *clickableView) Reverse() *clickableView {
	c.style = c.style.WithReverse()
	return c
}

// Style sets the complete style.
func (c *clickableView) Style(s Style) *clickableView {
	c.style = s
	return c
}

// Width sets a fixed width for the clickable.
func (c *clickableView) Width(w int) *clickableView {
	c.width = w
	return c
}

func (c *clickableView) size(maxWidth, maxHeight int) (int, int) {
	w, h := MeasureText(c.label)
	if c.width > 0 {
		w = c.width
	}
	if maxWidth > 0 && w > maxWidth {
		w = maxWidth
	}
	if maxHeight > 0 && h > maxHeight {
		h = maxHeight
	}
	return w, h
}

func (c *clickableView) render(frame RenderFrame, bounds image.Rectangle) {
	if bounds.Empty() {
		return
	}

	// Register this clickable for click handling
	// bounds are already in global coordinates
	if c.callback != nil {
		interactiveRegistry.RegisterButton(bounds, c.callback)
	}

	// Render the label
	subFrame := frame.SubFrame(bounds)
	subFrame.PrintTruncated(0, 0, c.label, c.style)
}
