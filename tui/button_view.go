package tui

import (
	"fmt"
	"image"
	"sync"
)

// buttonRegistryImpl manages button state for focus management.
// Owned per-runtime via the registries struct.
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
	// Clear map to prevent memory leaks from transient IDs (buttons with closure callbacks)
	for k := range r.buttons {
		delete(r.buttons, k)
	}
}

// Register adds or updates a button.
func (r *buttonRegistryImpl) Register(id string, bounds image.Rectangle, callback func(), focusStyle Style, fm *FocusManager) *buttonState {
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

	// Register with the focus manager (if available)
	if fm != nil {
		fm.Register(state)
	}

	return state
}

// interactiveRegistryImpl tracks clickable regions for mouse event routing.
// It is cleared before each render and populated as views are drawn.
// This is separate from focus management - it handles mouse-only interactions.
// Owned per-runtime via the registries struct.
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

// ButtonView displays an interactive button that can be focused and activated
type ButtonView struct {
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
func Button(label string, callback func()) *ButtonView {
	// Generate unique ID from callback pointer
	id := fmt.Sprintf("button_%p", callback)
	return &ButtonView{
		id:         id,
		label:      label,
		callback:   callback,
		style:      NewStyle(),
		focusStyle: NewStyle().WithReverse(), // Default focus indicator
		width:      0,
	}
}

// ID sets a specific ID for this button (useful for focus management).
func (b *ButtonView) ID(id string) *ButtonView {
	b.id = id
	return b
}

// Fg sets the foreground color.
func (b *ButtonView) Fg(col Color) *ButtonView {
	b.style = b.style.WithForeground(col)
	return b
}

// Bg sets the background color.
func (b *ButtonView) Bg(col Color) *ButtonView {
	b.style = b.style.WithBackground(col)
	return b
}

// Bold enables bold text.
func (b *ButtonView) Bold() *ButtonView {
	b.style = b.style.WithBold()
	return b
}

// Reverse enables reverse video.
func (b *ButtonView) Reverse() *ButtonView {
	b.style = b.style.WithReverse()
	return b
}

// Style sets the complete style.
func (b *ButtonView) Style(s Style) *ButtonView {
	b.style = s
	return b
}

// FocusStyle sets the style applied when this button is focused.
func (b *ButtonView) FocusStyle(s Style) *ButtonView {
	b.focusStyle = s
	return b
}

// Width sets a fixed width for the button.
func (b *ButtonView) Width(w int) *ButtonView {
	b.width = w
	return b
}

func (b *ButtonView) size(maxWidth, maxHeight int) (int, int) {
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

func (b *ButtonView) render(ctx *RenderContext) {
	w, h := ctx.Size()
	if w == 0 || h == 0 {
		return
	}

	// Register this button for focus management
	bounds := ctx.AbsoluteBounds()
	state := ctx.registries().buttons.Register(b.id, bounds, b.callback, b.focusStyle, ctx.FocusManager())

	// Choose style based on focus state
	style := b.style
	if state.focused {
		style = b.focusStyle
	}

	// Render the label
	ctx.PrintTruncated(0, 0, b.label, style)
}

// ClickableView displays an interactive clickable element (mouse-only, not focusable)
type ClickableView struct {
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
func Clickable(label string, callback func()) *ClickableView {
	return &ClickableView{
		label:    label,
		callback: callback,
		style:    NewStyle(),
		width:    0,
	}
}

// Fg sets the foreground color.
func (c *ClickableView) Fg(col Color) *ClickableView {
	c.style = c.style.WithForeground(col)
	return c
}

// Bg sets the background color.
func (c *ClickableView) Bg(col Color) *ClickableView {
	c.style = c.style.WithBackground(col)
	return c
}

// Bold enables bold text.
func (c *ClickableView) Bold() *ClickableView {
	c.style = c.style.WithBold()
	return c
}

// Reverse enables reverse video (useful for selected state).
func (c *ClickableView) Reverse() *ClickableView {
	c.style = c.style.WithReverse()
	return c
}

// Style sets the complete style.
func (c *ClickableView) Style(s Style) *ClickableView {
	c.style = s
	return c
}

// Width sets a fixed width for the clickable.
func (c *ClickableView) Width(w int) *ClickableView {
	c.width = w
	return c
}

func (c *ClickableView) size(maxWidth, maxHeight int) (int, int) {
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

func (c *ClickableView) render(ctx *RenderContext) {
	w, h := ctx.Size()
	if w == 0 || h == 0 {
		return
	}

	// Register this clickable for click handling (mouse only)
	if c.callback != nil {
		ctx.registries().interactive.RegisterRegion(ctx.AbsoluteBounds(), c.callback)
	}

	// Render the label
	ctx.PrintTruncated(0, 0, c.label, c.style)
}
