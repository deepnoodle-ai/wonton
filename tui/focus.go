package tui

import (
	"image"
	"sync"
)

// Focusable is implemented by views that can receive keyboard focus.
// Views implementing this interface participate in Tab navigation and
// can handle keyboard events when focused.
type Focusable interface {
	// FocusID returns a unique identifier for this focusable element.
	// This ID is used for programmatic focus control via Focus(id).
	FocusID() string

	// IsFocused returns whether this element currently has focus.
	IsFocused() bool

	// SetFocused is called when focus changes to or from this element.
	SetFocused(focused bool)

	// HandleKeyEvent processes a key event while this element has focus.
	// Returns true if the event was consumed and should not propagate.
	HandleKeyEvent(event KeyEvent) bool

	// FocusBounds returns the screen bounds of this focusable element.
	// Used for mouse click focus detection.
	FocusBounds() image.Rectangle
}

// FocusManager manages focus state for all focusable elements.
// It handles Tab/Shift+Tab navigation and routes keyboard events.
type FocusManager struct {
	mu         sync.Mutex
	focusables map[string]Focusable
	focusedID  string
	order      []string // registration order for Tab navigation
}

// Global focus manager instance
var focusManager = &FocusManager{
	focusables: make(map[string]Focusable),
}

// Clear clears all registered focusables (called before each render).
func (fm *FocusManager) Clear() {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	fm.order = fm.order[:0]
}

// Register adds a focusable element to the manager.
// Elements are registered in render order, which determines Tab navigation order.
func (fm *FocusManager) Register(f Focusable) {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	id := f.FocusID()
	fm.focusables[id] = f
	fm.order = append(fm.order, id)

	// Auto-focus first element if none focused
	if fm.focusedID == "" {
		fm.focusedID = id
		f.SetFocused(true)
	} else {
		f.SetFocused(fm.focusedID == id)
	}
}

// GetFocused returns the currently focused element, or nil if none.
func (fm *FocusManager) GetFocused() Focusable {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	if fm.focusedID == "" {
		return nil
	}
	return fm.focusables[fm.focusedID]
}

// GetFocusedID returns the ID of the currently focused element.
func (fm *FocusManager) GetFocusedID() string {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	return fm.focusedID
}

// SetFocus sets focus to a specific element by ID.
func (fm *FocusManager) SetFocus(id string) {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	// Unfocus previous
	if prev, exists := fm.focusables[fm.focusedID]; exists {
		prev.SetFocused(false)
	}

	fm.focusedID = id

	// Focus new
	if next, exists := fm.focusables[id]; exists {
		next.SetFocused(true)
	}
}

// FocusNext moves focus to the next element in registration order.
func (fm *FocusManager) FocusNext() {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	fm.focusNextLocked()
}

func (fm *FocusManager) focusNextLocked() {
	if len(fm.order) == 0 {
		return
	}

	currentIdx := fm.findCurrentIndex()

	// Unfocus current
	if prev, exists := fm.focusables[fm.focusedID]; exists {
		prev.SetFocused(false)
	}

	// Focus next (wrap around)
	nextIdx := (currentIdx + 1) % len(fm.order)
	fm.focusedID = fm.order[nextIdx]

	if next, exists := fm.focusables[fm.focusedID]; exists {
		next.SetFocused(true)
	}
}

// FocusPrev moves focus to the previous element in registration order.
func (fm *FocusManager) FocusPrev() {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	fm.focusPrevLocked()
}

func (fm *FocusManager) focusPrevLocked() {
	if len(fm.order) == 0 {
		return
	}

	currentIdx := fm.findCurrentIndex()

	// Unfocus current
	if prev, exists := fm.focusables[fm.focusedID]; exists {
		prev.SetFocused(false)
	}

	// Focus previous (wrap around)
	prevIdx := currentIdx - 1
	if prevIdx < 0 {
		prevIdx = len(fm.order) - 1
	}
	fm.focusedID = fm.order[prevIdx]

	if next, exists := fm.focusables[fm.focusedID]; exists {
		next.SetFocused(true)
	}
}

func (fm *FocusManager) findCurrentIndex() int {
	for i, id := range fm.order {
		if id == fm.focusedID {
			return i
		}
	}
	return -1
}

// HandleKey routes a key event to the focused element.
// Returns true if the event was handled.
func (fm *FocusManager) HandleKey(event KeyEvent) bool {
	// Handle Tab/Shift+Tab for navigation (before delegating to focused element)
	if event.Key == KeyTab {
		if event.Shift {
			fm.FocusPrev()
		} else {
			fm.FocusNext()
		}
		return true
	}

	fm.mu.Lock()
	focused := fm.focusables[fm.focusedID]
	fm.mu.Unlock()

	if focused == nil {
		return false
	}

	return focused.HandleKeyEvent(event)
}

// HandleClick checks if a click hit any focusable element and focuses it.
// Returns true if a focusable was clicked.
func (fm *FocusManager) HandleClick(x, y int) bool {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	pt := image.Pt(x, y)
	for id, f := range fm.focusables {
		if pt.In(f.FocusBounds()) {
			// Unfocus previous
			if prev, exists := fm.focusables[fm.focusedID]; exists {
				prev.SetFocused(false)
			}
			fm.focusedID = id
			f.SetFocused(true)
			return true
		}
	}
	return false
}

// Focus returns a command that sets focus to the specified element ID.
// Use this in HandleEvent to programmatically focus an element.
//
// Example:
//
//	func (app *App) HandleEvent(e Event) []Cmd {
//	    if key, ok := e.(KeyEvent); ok && key.Rune == 'n' {
//	        return []Cmd{Focus("name-input")}
//	    }
//	    return nil
//	}
func Focus(id string) Cmd {
	return func() Event {
		focusManager.SetFocus(id)
		return nil
	}
}

// FocusNext returns a command that moves focus to the next element.
func FocusNext() Cmd {
	return func() Event {
		focusManager.FocusNext()
		return nil
	}
}

// FocusPrev returns a command that moves focus to the previous element.
func FocusPrev() Cmd {
	return func() Event {
		focusManager.FocusPrev()
		return nil
	}
}

// GetFocusedID returns the ID of the currently focused element.
// Useful for conditionally styling views based on focus state.
func GetFocusedID() string {
	return focusManager.GetFocusedID()
}

// RegisterFocusable registers a custom focusable component with the focus manager.
// This is called automatically by FocusableView during render.
func RegisterFocusable(f Focusable) {
	focusManager.Register(f)
}

// FocusHandler defines callbacks for custom focusable behavior.
type FocusHandler struct {
	// ID is the unique identifier for this focusable element.
	ID string
	// OnKey is called when a key event occurs while this element is focused.
	// Return true to consume the event.
	OnKey func(event KeyEvent) bool
}

// focusableWrapper wraps a view with focusable behavior
type focusableWrapper struct {
	inner   View
	handler *FocusHandler
	bounds  image.Rectangle
	focused bool
}

// FocusableView wraps any view to make it participate in focus navigation.
// The handler defines the focus ID and optional key event handling.
//
// Example:
//
//	FocusableView(
//	    myScrollView,
//	    &tui.FocusHandler{
//	        ID: "content-viewer",
//	        OnKey: func(e tui.KeyEvent) bool {
//	            if e.Key == tui.KeyArrowDown { scrollY++; return true }
//	            return false
//	        },
//	    },
//	)
func FocusableView(inner View, handler *FocusHandler) View {
	return &focusableWrapper{
		inner:   inner,
		handler: handler,
	}
}

// Focusable interface implementation

func (f *focusableWrapper) FocusID() string {
	return f.handler.ID
}

func (f *focusableWrapper) IsFocused() bool {
	return f.focused
}

func (f *focusableWrapper) SetFocused(focused bool) {
	f.focused = focused
}

func (f *focusableWrapper) FocusBounds() image.Rectangle {
	return f.bounds
}

func (f *focusableWrapper) HandleKeyEvent(event KeyEvent) bool {
	if f.handler.OnKey != nil {
		return f.handler.OnKey(event)
	}
	return false
}

// View interface implementation

func (f *focusableWrapper) size(maxWidth, maxHeight int) (int, int) {
	return f.inner.size(maxWidth, maxHeight)
}

func (f *focusableWrapper) render(ctx *RenderContext) {
	f.bounds = ctx.AbsoluteBounds()
	focusManager.Register(f)
	f.inner.render(ctx)
}
