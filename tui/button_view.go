package tui

import (
	"fmt"
	"image"
	"sync"
)

// buttonRegistry manages button state for focus management.
var buttonRegistry = &buttonRegistryImpl{
	buttons: make(map[string]*buttonState),
}

type buttonRegistryImpl struct {
	mu      sync.Mutex
	buttons map[string]*buttonState
}

type buttonState struct {
	id         string
	bounds     image.Rectangle
	callback   func()
	focused    bool
	focusStyle Style
}

// Focusable interface implementation for buttonState

func (b *buttonState) FocusID() string {
	return b.id
}

func (b *buttonState) IsFocused() bool {
	return b.focused
}

func (b *buttonState) SetFocused(focused bool) {
	b.focused = focused
}

func (b *buttonState) FocusBounds() image.Rectangle {
	return b.bounds
}

func (b *buttonState) HandleKeyEvent(event KeyEvent) bool {
	// Activate button on Enter or Space
	if event.Key == KeyEnter || event.Rune == ' ' {
		if b.callback != nil {
			b.callback()
		}
		return true
	}
	return false
}

// Clear clears button tracking (called before each render).
func (r *buttonRegistryImpl) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	// Don't clear buttons map, just let focusManager handle order
}

// Register adds or updates a button.
func (r *buttonRegistryImpl) Register(id string, bounds image.Rectangle, callback func(), focusStyle Style) *buttonState {
	r.mu.Lock()
	defer r.mu.Unlock()

	state, exists := r.buttons[id]
	if !exists {
		state = &buttonState{
			id: id,
		}
		r.buttons[id] = state
	}

	state.bounds = bounds
	state.callback = callback
	state.focusStyle = focusStyle

	// Register with the global focus manager
	focusManager.Register(state)

	return state
}

// InteractiveRegistry tracks clickable regions for mouse event routing.
// It is cleared before each render and populated as views are drawn.
// This is separate from focus management - it handles mouse-only interactions.
var interactiveRegistry = &interactiveRegistryImpl{
	regions: make([]interactiveRegion, 0),
}

type interactiveRegistryImpl struct {
	mu      sync.Mutex
	regions []interactiveRegion
}

type interactiveRegion struct {
	bounds   image.Rectangle
	callback func()
}

// Clear clears all registered interactive regions.
// Called by the runtime before each render.
func (r *interactiveRegistryImpl) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.regions = r.regions[:0]
}

// RegisterRegion adds a clickable region (for non-focusable clickables).
func (r *interactiveRegistryImpl) RegisterRegion(bounds image.Rectangle, callback func()) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.regions = append(r.regions, interactiveRegion{bounds: bounds, callback: callback})
}

// RegisterButton is an alias for RegisterRegion for backward compatibility.
func (r *interactiveRegistryImpl) RegisterButton(bounds image.Rectangle, callback func()) {
	r.RegisterRegion(bounds, callback)
}

// HandleClick checks if a click hit any registered region and invokes its callback.
// Returns true if a region was clicked.
func (r *interactiveRegistryImpl) HandleClick(x, y int) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	pt := image.Pt(x, y)
	for _, region := range r.regions {
		if pt.In(region.bounds) {
			callback := region.callback
			r.mu.Unlock()
			callback()
			r.mu.Lock()
			return true
		}
	}
	return false
}

// buttonView displays an interactive button that can be focused and activated
type buttonView struct {
	id         string
	label      string
	callback   func()
	style      Style
	focusStyle Style
	width      int
}

// Button creates a focusable button element.
// The callback is invoked when the button is clicked or activated with Enter/Space.
//
// Example:
//
//	Button("Submit", func() { app.submit() })
func Button(label string, callback func()) *buttonView {
	// Generate unique ID from callback pointer
	id := fmt.Sprintf("button_%p", callback)
	return &buttonView{
		id:         id,
		label:      label,
		callback:   callback,
		style:      NewStyle(),
		focusStyle: NewStyle().WithReverse(), // Default focus indicator
		width:      0,
	}
}

// ID sets a specific ID for this button (useful for focus management).
func (b *buttonView) ID(id string) *buttonView {
	b.id = id
	return b
}

// Fg sets the foreground color.
func (b *buttonView) Fg(col Color) *buttonView {
	b.style = b.style.WithForeground(col)
	return b
}

// Bg sets the background color.
func (b *buttonView) Bg(col Color) *buttonView {
	b.style = b.style.WithBackground(col)
	return b
}

// Bold enables bold text.
func (b *buttonView) Bold() *buttonView {
	b.style = b.style.WithBold()
	return b
}

// Reverse enables reverse video.
func (b *buttonView) Reverse() *buttonView {
	b.style = b.style.WithReverse()
	return b
}

// Style sets the complete style.
func (b *buttonView) Style(s Style) *buttonView {
	b.style = s
	return b
}

// FocusStyle sets the style applied when this button is focused.
func (b *buttonView) FocusStyle(s Style) *buttonView {
	b.focusStyle = s
	return b
}

// Width sets a fixed width for the button.
func (b *buttonView) Width(w int) *buttonView {
	b.width = w
	return b
}

func (b *buttonView) size(maxWidth, maxHeight int) (int, int) {
	w, h := MeasureText(b.label)
	if b.width > 0 {
		w = b.width
	}
	if maxWidth > 0 && w > maxWidth {
		w = maxWidth
	}
	if maxHeight > 0 && h > maxHeight {
		h = maxHeight
	}
	return w, h
}

func (b *buttonView) render(ctx *RenderContext) {
	w, h := ctx.Size()
	if w == 0 || h == 0 {
		return
	}

	// Register this button for focus management
	bounds := ctx.AbsoluteBounds()
	state := buttonRegistry.Register(b.id, bounds, b.callback, b.focusStyle)

	// Choose style based on focus state
	style := b.style
	if state.focused {
		style = b.focusStyle
	}

	// Render the label
	ctx.PrintTruncated(0, 0, b.label, style)
}

// clickableView displays an interactive clickable element (mouse-only, not focusable)
type clickableView struct {
	label    string
	callback func()
	style    Style
	width    int
}

// Clickable creates a mouse-only clickable element (not keyboard focusable).
// For keyboard-accessible buttons, use Button() instead.
//
// Example:
//
//	Clickable("Link", func() { app.openLink() })
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

func (c *clickableView) render(ctx *RenderContext) {
	w, h := ctx.Size()
	if w == 0 || h == 0 {
		return
	}

	// Register this clickable for click handling (mouse only)
	if c.callback != nil {
		interactiveRegistry.RegisterRegion(ctx.AbsoluteBounds(), c.callback)
	}

	// Render the label
	ctx.PrintTruncated(0, 0, c.label, c.style)
}
