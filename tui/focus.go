package tui

import (
	"image"
	"sync"
	"time"
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
//
// Each Runtime or InlineApp instance has its own FocusManager, avoiding
// global state and allowing multiple independent TUI applications.
type FocusManager struct {
	mu         sync.Mutex
	focusables map[string]Focusable
	focusedID  string
	order      []string // registration order for Tab navigation
	// keep tells the next Clear and key event not to treat an unregistered
	// focusedID as stale. SetFocus sets it for an element that has not
	// rendered yet, such as a dialog that opens in the same update. A
	// Viewport sets it when it leaves items unrendered, because the focused
	// element can be one that scrolled out of view.
	keep bool
}

// NewFocusManager creates a new focus manager instance.
func NewFocusManager() *FocusManager {
	return &FocusManager{
		focusables: make(map[string]Focusable),
	}
}

// Clear clears all registered focusables (called before each render).
func (fm *FocusManager) Clear() {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	if !fm.keep {
		fm.dropStaleLocked()
	}
	fm.keep = false
	fm.order = fm.order[:0]
	// Clear map to prevent memory leaks from transient IDs (e.g. Buttons with closures)
	for k := range fm.focusables {
		delete(fm.focusables, k)
	}
}

// Register adds a focusable element to the manager.
// Elements are registered in render order, which determines Tab navigation order.
func (fm *FocusManager) Register(f Focusable) {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	id := f.FocusID()
	// An element with an empty ID can hold focus while focusedID is "", so
	// a later element must not also auto-focus.
	_, emptyFocused := fm.focusables[""]
	fm.focusables[id] = f
	fm.order = append(fm.order, id)

	// Auto-focus first element if none focused
	if fm.focusedID == "" && !emptyFocused {
		fm.focusedID = id
		f.SetFocused(true)
	} else {
		// An element with an empty ID only holds focus when it auto-focused
		// above; a later one must not match the empty focusedID.
		f.SetFocused(id != "" && fm.focusedID == id)
	}
}

// wouldAutoFocus reports whether the next registered element would take
// focus, because nothing holds it yet.
func (fm *FocusManager) wouldAutoFocus() bool {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	_, emptyFocused := fm.focusables[""]
	return fm.focusedID == "" && !emptyFocused
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
	next, exists := fm.focusables[id]
	if exists {
		next.SetFocused(true)
	}
	if !exists {
		fm.keep = true
	}
}

// registeredCount returns how many elements have registered this frame. A
// view passes it to registeredSince to learn which IDs a child registered.
func (fm *FocusManager) registeredCount() int {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	return len(fm.order)
}

// registeredSince appends to dst the IDs registered after the first n.
func (fm *FocusManager) registeredSince(dst []string, n int) []string {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	if n < len(fm.order) {
		dst = append(dst, fm.order[n:]...)
	}
	return dst
}

// unregisteredFocusID returns the focused ID when its element has not
// registered this frame.
func (fm *FocusManager) unregisteredFocusID() (string, bool) {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	if fm.focusedID == "" {
		return "", false
	}
	_, exists := fm.focusables[fm.focusedID]
	return fm.focusedID, !exists
}

// keepFocus keeps the focused ID through the next frame even if its element
// does not render. A view that renders only some of its children calls it
// when the focused element is one of the children it skipped.
func (fm *FocusManager) keepFocus() {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	fm.keep = true
}

// dropStaleLocked clears focusedID when the focused element was not
// registered in the frame that just rendered, so the next frame auto-focuses
// its first focusable. Without this, focus is lost for good when the focused
// view disappears (for example, when a dialog closes).
func (fm *FocusManager) dropStaleLocked() {
	if _, exists := fm.focusables[fm.focusedID]; !exists {
		fm.focusedID = ""
	}
}

// settleLocked moves focus to the first registered element when the focused
// element is no longer registered. Key handling calls it so that a key that
// arrives before the next render reaches a focusable, as auto-focus would.
func (fm *FocusManager) settleLocked() {
	if fm.keep || len(fm.order) == 0 {
		return
	}
	if _, exists := fm.focusables[fm.focusedID]; exists {
		return
	}
	fm.focusedID = fm.order[0]
	fm.focusables[fm.focusedID].SetFocused(true)
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
//
// Tab/Shift+Tab are offered to the focused element first, so widgets that
// claim Tab (an input with OnComplete, or an OnKey hook that returns true)
// can consume it. An unclaimed Tab cycles focus when there are at least two
// focusables; with nowhere to go, it propagates to the application.
func (fm *FocusManager) HandleKey(event KeyEvent) bool {
	fm.mu.Lock()
	fm.settleLocked()
	focused := fm.focusables[fm.focusedID]
	fm.mu.Unlock()

	if event.Key == KeyTab {
		if focused != nil && focused.HandleKeyEvent(event) {
			return true
		}
		fm.mu.Lock()
		canCycle := len(fm.order) > 1
		fm.mu.Unlock()
		if !canCycle {
			// Nowhere to cycle to: let the application observe Tab.
			return false
		}
		if event.Shift {
			fm.FocusPrev()
		} else {
			fm.FocusNext()
		}
		return true
	}

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

// FocusSetEvent is produced by the Focus command and processed by the Runtime
// or InlineApp to set focus to a specific element. The runtime applies the
// Focus command directly; a custom Cmd that returns this event is applied
// when its result arrives, like any other command.
type FocusSetEvent struct {
	ID   string    // The ID of the element to focus
	Time time.Time // When the event was created
}

// Timestamp implements Event.
func (e FocusSetEvent) Timestamp() time.Time { return e.Time }

// FocusNextEvent is produced by the FocusNext command and processed by the
// Runtime or InlineApp to move focus to the next element in Tab order.
type FocusNextEvent struct {
	Time time.Time // When the event was created
}

// Timestamp implements Event.
func (e FocusNextEvent) Timestamp() time.Time { return e.Time }

// FocusPrevEvent is produced by the FocusPrev command and processed by the
// Runtime or InlineApp to move focus to the previous element in Tab order.
type FocusPrevEvent struct {
	Time time.Time // When the event was created
}

// Timestamp implements Event.
func (e FocusPrevEvent) Timestamp() time.Time { return e.Time }

// Focus returns a command that sets focus to the specified element ID.
// The Runtime or InlineApp applies it as soon as HandleEvent returns, before
// the next event, so a key that follows reaches the newly focused element.
//
// Use this in HandleEvent to programmatically focus an element:
//
//	func (app *App) HandleEvent(e Event) []Cmd {
//	    if key, ok := e.(KeyEvent); ok && key.Rune == 'n' {
//	        return []Cmd{Focus("name-input")}
//	    }
//	    return nil
//	}
//
// The element ID must match the ID used when creating the focusable component
// (e.g., Input("id"), Button("id", ...), etc.).
func Focus(id string) Cmd {
	return focusSetCmd{id: id}.event
}

// FocusNext returns a command that moves focus to the next element in Tab order.
// Like Focus, it is applied as soon as HandleEvent returns.
// This is equivalent to pressing Tab.
func FocusNext() Cmd {
	return focusNextCmd{}.event
}

// FocusPrev returns a command that moves focus to the previous element in Tab order.
// Like Focus, it is applied as soon as HandleEvent returns.
// This is equivalent to pressing Shift+Tab.
func FocusPrev() Cmd {
	return focusPrevCmd{}.event
}

// The focus commands are method values so that every command of one kind
// shares a single function, which isRuntimeCmd can recognize. A closure
// would not work: the compiler copies it into each caller that inlines Focus.
type focusSetCmd struct{ id string }

func (c focusSetCmd) event() Event { return FocusSetEvent{ID: c.id, Time: time.Now()} }

type focusNextCmd struct{}

func (focusNextCmd) event() Event { return FocusNextEvent{Time: time.Now()} }

type focusPrevCmd struct{}

func (focusPrevCmd) event() Event { return FocusPrevEvent{Time: time.Now()} }

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

// flex forwards the inner view's flex, so making a view focusable does not
// change its layout.
func (f *focusableWrapper) flex() int {
	if flex, ok := f.inner.(Flexible); ok {
		return flex.flex()
	}
	return 0
}

func (f *focusableWrapper) size(maxWidth, maxHeight int) (int, int) {
	return f.inner.size(maxWidth, maxHeight)
}

func (f *focusableWrapper) render(ctx *RenderContext) {
	// Only the visible part takes focus clicks; rows a Scroll hides must not.
	f.bounds = ctx.clickArea(ctx.AbsoluteBounds())
	if fm := ctx.FocusManager(); fm != nil {
		fm.Register(f)
	}
	f.inner.render(ctx)
}
