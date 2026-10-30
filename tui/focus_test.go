package tui

import (
	"fmt"
	"image"
	"io"
	"testing"
	"time"

	"github.com/deepnoodle-ai/wonton/assert"
)

// Mock focusable for testing
type mockFocusable struct {
	id      string
	focused bool
}

func (m *mockFocusable) FocusID() string {
	return m.id
}

func (m *mockFocusable) IsFocused() bool {
	return m.focused
}

func (m *mockFocusable) SetFocused(focused bool) {
	m.focused = focused
}

func (m *mockFocusable) HandleKeyEvent(event KeyEvent) bool {
	return false
}

func (m *mockFocusable) FocusBounds() image.Rectangle {
	return image.Rect(0, 0, 10, 10)
}

func TestFocusManager_Register(t *testing.T) {
	fm := &FocusManager{
		focusables: make(map[string]Focusable),
	}

	btn1 := &mockFocusable{id: "btn1"}
	btn2 := &mockFocusable{id: "btn2"}

	fm.Register(btn1)
	assert.True(t, btn1.focused, "First registered element should be auto-focused")
	assert.Equal(t, "btn1", fm.focusedID)

	fm.Register(btn2)
	assert.False(t, btn2.focused, "Subsequent element should not be auto-focused")
	assert.Equal(t, "btn1", fm.focusedID)
}

func TestFocusManager_Clear(t *testing.T) {
	fm := &FocusManager{
		focusables: make(map[string]Focusable),
	}

	btn1 := &mockFocusable{id: "btn1"}
	fm.Register(btn1)

	assert.Equal(t, 1, len(fm.focusables))
	assert.Equal(t, 1, len(fm.order))

	fm.Clear()

	// Order should be cleared
	assert.Equal(t, 0, len(fm.order))
	// Map should be cleared (new behavior)
	assert.Equal(t, 0, len(fm.focusables))
	// Focused ID should be preserved
	assert.Equal(t, "btn1", fm.focusedID)
}

func TestFocusManager_Persistence(t *testing.T) {
	// Verify focus is restored after clear/re-register
	fm := &FocusManager{
		focusables: make(map[string]Focusable),
	}

	btn1 := &mockFocusable{id: "btn1"}
	btn2 := &mockFocusable{id: "btn2"}

	// Frame 1
	fm.Register(btn1)
	fm.Register(btn2)
	fm.SetFocus("btn2")
	assert.True(t, btn2.focused)
	assert.Equal(t, "btn2", fm.focusedID)

	// Clear (Simulate end of frame)
	fm.Clear()
	assert.Equal(t, 0, len(fm.focusables))

	// Frame 2
	// Re-register btn1 (not focused)
	btn1New := &mockFocusable{id: "btn1"}
	fm.Register(btn1New)
	assert.False(t, btn1New.focused)

	// Re-register btn2 (should be restored to focused)
	btn2New := &mockFocusable{id: "btn2"}
	fm.Register(btn2New)
	assert.True(t, btn2New.focused, "Focus should be restored to btn2")
}

func TestFocusManager_FocusNext(t *testing.T) {
	fm := &FocusManager{
		focusables: make(map[string]Focusable),
	}

	btn1 := &mockFocusable{id: "btn1"}
	btn2 := &mockFocusable{id: "btn2"}
	btn3 := &mockFocusable{id: "btn3"}

	fm.Register(btn1) // auto-focus btn1
	fm.Register(btn2)
	fm.Register(btn3)

	assert.Equal(t, "btn1", fm.focusedID)

	fm.FocusNext()
	assert.Equal(t, "btn2", fm.focusedID)
	assert.True(t, btn2.focused)
	assert.False(t, btn1.focused)

	fm.FocusNext()
	assert.Equal(t, "btn3", fm.focusedID)

	fm.FocusNext()
	assert.Equal(t, "btn1", fm.focusedID) // Wrap around
}

func TestFocusManager_FocusPrev(t *testing.T) {
	fm := NewFocusManager()

	btn1 := &mockFocusable{id: "btn1"}
	btn2 := &mockFocusable{id: "btn2"}
	btn3 := &mockFocusable{id: "btn3"}

	fm.Register(btn1)
	fm.Register(btn2)
	fm.Register(btn3)

	assert.Equal(t, "btn1", fm.GetFocusedID())

	fm.FocusPrev()
	assert.Equal(t, "btn3", fm.GetFocusedID()) // Wrap around to end
	assert.True(t, btn3.focused)
	assert.False(t, btn1.focused)

	fm.FocusPrev()
	assert.Equal(t, "btn2", fm.GetFocusedID())

	fm.FocusPrev()
	assert.Equal(t, "btn1", fm.GetFocusedID())
}

func TestFocusManager_SetFocus(t *testing.T) {
	fm := NewFocusManager()

	btn1 := &mockFocusable{id: "btn1"}
	btn2 := &mockFocusable{id: "btn2"}
	btn3 := &mockFocusable{id: "btn3"}

	fm.Register(btn1)
	fm.Register(btn2)
	fm.Register(btn3)

	// btn1 is auto-focused
	assert.True(t, btn1.focused)
	assert.False(t, btn2.focused)
	assert.False(t, btn3.focused)

	// Set focus to btn3
	fm.SetFocus("btn3")
	assert.False(t, btn1.focused)
	assert.False(t, btn2.focused)
	assert.True(t, btn3.focused)
	assert.Equal(t, "btn3", fm.GetFocusedID())

	// Set focus to non-existent element (should just update focusedID)
	fm.SetFocus("nonexistent")
	assert.False(t, btn3.focused) // btn3 should be unfocused
	assert.Equal(t, "nonexistent", fm.GetFocusedID())
}

func TestFocusManager_GetFocused(t *testing.T) {
	fm := NewFocusManager()

	// Empty manager returns nil
	assert.Nil(t, fm.GetFocused())

	btn1 := &mockFocusable{id: "btn1"}
	fm.Register(btn1)

	focused := fm.GetFocused()
	assert.NotNil(t, focused)
	assert.Equal(t, "btn1", focused.FocusID())
}

func TestFocusManager_HandleKey_Tab(t *testing.T) {
	fm := NewFocusManager()

	btn1 := &mockFocusable{id: "btn1"}
	btn2 := &mockFocusable{id: "btn2"}

	fm.Register(btn1)
	fm.Register(btn2)

	// Tab should move to next
	handled := fm.HandleKey(KeyEvent{Key: KeyTab})
	assert.True(t, handled)
	assert.Equal(t, "btn2", fm.GetFocusedID())

	// Shift+Tab should move to previous
	handled = fm.HandleKey(KeyEvent{Key: KeyTab, Shift: true})
	assert.True(t, handled)
	assert.Equal(t, "btn1", fm.GetFocusedID())
}

func TestFocusManager_HandleKey_TabWithoutFocusables(t *testing.T) {
	fm := NewFocusManager()

	// With nothing registered there is nothing to cycle through; Tab must
	// not be consumed so the application can observe it.
	assert.False(t, fm.HandleKey(KeyEvent{Key: KeyTab}))
	assert.False(t, fm.HandleKey(KeyEvent{Key: KeyTab, Shift: true}))
}

func TestFocusManager_HandleKey_TabSingleFocusable(t *testing.T) {
	fm := NewFocusManager()
	fm.Register(&mockFocusable{id: "only"})

	// A lone focusable that doesn't claim Tab: there is nowhere to cycle
	// to, so Tab must propagate to the application.
	assert.False(t, fm.HandleKey(KeyEvent{Key: KeyTab}))
	assert.Equal(t, "only", fm.GetFocusedID(), "focus should not change")
}

func TestFocusManager_HandleKey_TabDelegatesToFocusedFirst(t *testing.T) {
	fm := NewFocusManager()

	claimer := &mockFocusableWithHandler{
		id:    "claimer",
		onKey: func(e KeyEvent) bool { return e.Key == KeyTab },
	}
	fm.Register(claimer)
	fm.Register(&mockFocusable{id: "other"})

	// The focused element claims Tab, so focus must not move.
	assert.True(t, fm.HandleKey(KeyEvent{Key: KeyTab}))
	assert.Equal(t, "claimer", fm.GetFocusedID())

	// Once focus moves to an element that doesn't claim Tab, cycling
	// resumes normally.
	fm.SetFocus("other")
	assert.True(t, fm.HandleKey(KeyEvent{Key: KeyTab}))
	assert.Equal(t, "claimer", fm.GetFocusedID())
}

func TestFocusManager_HandleKey_DelegatesToFocused(t *testing.T) {
	fm := NewFocusManager()

	keyHandled := false
	mock := &mockFocusableWithHandler{
		id: "handler",
		onKey: func(e KeyEvent) bool {
			keyHandled = true
			return e.Rune == 'x'
		},
	}

	fm.Register(mock)

	// Non-tab key should delegate to focused element
	handled := fm.HandleKey(KeyEvent{Rune: 'x'})
	assert.True(t, keyHandled)
	assert.True(t, handled, "Handler returned true for 'x'")

	keyHandled = false
	handled = fm.HandleKey(KeyEvent{Rune: 'y'})
	assert.True(t, keyHandled)
	assert.False(t, handled, "Handler returned false for 'y'")
}

func TestFocusManager_HandleClick(t *testing.T) {
	fm := NewFocusManager()

	btn1 := &mockFocusableWithBounds{id: "btn1", bounds: image.Rect(0, 0, 10, 10)}
	btn2 := &mockFocusableWithBounds{id: "btn2", bounds: image.Rect(20, 0, 30, 10)}

	fm.Register(btn1)
	fm.Register(btn2)

	// btn1 is auto-focused
	assert.True(t, btn1.focused)
	assert.False(t, btn2.focused)

	// Click on btn2
	handled := fm.HandleClick(25, 5)
	assert.True(t, handled)
	assert.False(t, btn1.focused)
	assert.True(t, btn2.focused)
	assert.Equal(t, "btn2", fm.GetFocusedID())

	// Click outside all elements
	handled = fm.HandleClick(100, 100)
	assert.False(t, handled)
	// Focus should remain unchanged
	assert.Equal(t, "btn2", fm.GetFocusedID())
}

func TestFocusManager_EmptyManager(t *testing.T) {
	fm := NewFocusManager()

	// Operations on empty manager should not panic
	fm.FocusNext()                      // No-op with no elements
	fm.FocusPrev()                      // No-op with no elements
	fm.HandleKey(KeyEvent{Key: KeyTab}) // Tab cycles through nothing
	fm.HandleClick(0, 0)                // Click hits nothing

	// Verify empty state
	assert.Equal(t, "", fm.GetFocusedID())
	assert.Nil(t, fm.GetFocused())

	// SetFocus still updates the focusedID even with no registered elements
	fm.SetFocus("pending")
	assert.Equal(t, "pending", fm.GetFocusedID())
	assert.Nil(t, fm.GetFocused()) // Still nil since "pending" isn't registered

	// Clear resets the map/order but preserves focusedID for persistence
	fm.Clear()
	assert.Equal(t, "pending", fm.GetFocusedID())
}

func TestNewFocusManager(t *testing.T) {
	fm := NewFocusManager()

	assert.NotNil(t, fm)
	assert.NotNil(t, fm.focusables)
	assert.Equal(t, 0, len(fm.focusables))
	assert.Equal(t, "", fm.focusedID)
}

// mockFocusableWithHandler extends mockFocusable with custom key handling
type mockFocusableWithHandler struct {
	id      string
	focused bool
	onKey   func(KeyEvent) bool
}

func (m *mockFocusableWithHandler) FocusID() string              { return m.id }
func (m *mockFocusableWithHandler) IsFocused() bool              { return m.focused }
func (m *mockFocusableWithHandler) SetFocused(focused bool)      { m.focused = focused }
func (m *mockFocusableWithHandler) FocusBounds() image.Rectangle { return image.Rect(0, 0, 10, 10) }
func (m *mockFocusableWithHandler) HandleKeyEvent(e KeyEvent) bool {
	if m.onKey != nil {
		return m.onKey(e)
	}
	return false
}

// mockFocusableWithBounds extends mockFocusable with configurable bounds
type mockFocusableWithBounds struct {
	id      string
	focused bool
	bounds  image.Rectangle
}

func (m *mockFocusableWithBounds) FocusID() string                { return m.id }
func (m *mockFocusableWithBounds) IsFocused() bool                { return m.focused }
func (m *mockFocusableWithBounds) SetFocused(focused bool)        { m.focused = focused }
func (m *mockFocusableWithBounds) FocusBounds() image.Rectangle   { return m.bounds }
func (m *mockFocusableWithBounds) HandleKeyEvent(e KeyEvent) bool { return false }

// Focus command and event tests

func TestFocus_ReturnsCorrectEvent(t *testing.T) {
	cmd := Focus("test-input")
	event := cmd()

	focusEvent, ok := event.(FocusSetEvent)
	assert.True(t, ok, "Focus should return FocusSetEvent")
	assert.Equal(t, "test-input", focusEvent.ID)
	assert.False(t, focusEvent.Time.IsZero(), "Time should be set")
}

func TestFocusNext_ReturnsCorrectEvent(t *testing.T) {
	cmd := FocusNext()
	event := cmd()

	_, ok := event.(FocusNextEvent)
	assert.True(t, ok, "FocusNext should return FocusNextEvent")
}

func TestFocusPrev_ReturnsCorrectEvent(t *testing.T) {
	cmd := FocusPrev()
	event := cmd()

	_, ok := event.(FocusPrevEvent)
	assert.True(t, ok, "FocusPrev should return FocusPrevEvent")
}

func TestFocusSetEvent_ImplementsEvent(t *testing.T) {
	now := time.Now()
	event := FocusSetEvent{ID: "test", Time: now}

	// Verify it implements Event interface
	var e Event = event
	assert.Equal(t, now, e.Timestamp())
}

func TestFocusNextEvent_ImplementsEvent(t *testing.T) {
	now := time.Now()
	event := FocusNextEvent{Time: now}

	var e Event = event
	assert.Equal(t, now, e.Timestamp())
}

func TestFocusPrevEvent_ImplementsEvent(t *testing.T) {
	now := time.Now()
	event := FocusPrevEvent{Time: now}

	var e Event = event
	assert.Equal(t, now, e.Timestamp())
}

// renderFocusables simulates one frame: Clear, then Register each element.
func renderFocusables(fm *FocusManager, fs ...*mockFocusable) {
	fm.Clear()
	for _, f := range fs {
		fm.Register(f)
	}
}

func TestFocusManager_FocusedViewDisappearsRefocusesFirst(t *testing.T) {
	fm := NewFocusManager()
	main := &mockFocusable{id: "main-input"}
	modal := &mockFocusable{id: "modal-input"}

	renderFocusables(fm, main, modal)
	fm.SetFocus("modal-input")
	renderFocusables(fm, main, modal)
	assert.True(t, modal.focused)

	// The modal closes. The frame that drops it still shows the stale focus,
	// and the next frame focuses the first remaining focusable.
	renderFocusables(fm, main)
	renderFocusables(fm, main)
	assert.True(t, main.focused, "the remaining focusable should take focus")
	assert.Equal(t, "main-input", fm.GetFocusedID())
}

func TestFocusManager_KeyAfterFocusedViewDisappears(t *testing.T) {
	fm := NewFocusManager()
	main := &mockFocusable{id: "main-input"}
	modal := &mockFocusable{id: "modal-input"}

	renderFocusables(fm, main, modal)
	fm.SetFocus("modal-input")
	renderFocusables(fm, main) // modal closed, no render since

	fm.HandleKey(KeyEvent{Key: KeyTab})
	assert.True(t, main.focused, "Tab should reach the only remaining focusable")
	assert.Equal(t, "main-input", fm.GetFocusedID())
}

func TestFocusManager_SetFocusBeforeElementRenders(t *testing.T) {
	fm := NewFocusManager()
	main := &mockFocusable{id: "main-input"}
	modal := &mockFocusable{id: "modal-input"}

	renderFocusables(fm, main)
	// The app opens a modal and focuses it in the same update. The modal
	// is not registered until the next frame, so focus must wait for it.
	fm.SetFocus("modal-input")
	renderFocusables(fm, main, modal)
	assert.True(t, modal.focused)
	assert.False(t, main.focused)
	assert.Equal(t, "modal-input", fm.GetFocusedID())
}

// inputItems is a list of one-row input fields for a Viewport.
type inputItems struct{ values []string }

func (it *inputItems) Len() int { return len(it.values) }
func (it *inputItems) Item(i int) View {
	return InputField(&it.values[i]).ID(fmt.Sprintf("field-%d", i))
}

type focusViewportApp struct {
	vp    ViewportState
	items *inputItems
	width int // when set, the viewport's width
}

func (a *focusViewportApp) View() View {
	if a.width > 0 {
		return Width(a.width, Height(3, Viewport(&a.vp, a.items)))
	}
	return Height(3, Viewport(&a.vp, a.items))
}

func TestFocusManager_KeepsFocusOnItemScrolledOutOfViewport(t *testing.T) {
	app := &focusViewportApp{items: &inputItems{values: make([]string, 10)}}
	r := NewRuntime(NewTestTerminal(40, 10, io.Discard), app, 30)
	r.render()
	r.focusMgr.SetFocus("field-1")
	r.render()

	app.vp.ScrollToBottom()
	r.render()
	r.render()
	assert.Equal(t, "field-1", r.focusMgr.GetFocusedID(), "focus stays on an item scrolled out of view")

	app.vp.ScrollToTop()
	r.render()
	r.processEvent(KeyEvent{Rune: 'x'})
	assert.Equal(t, "x", app.items.values[1], "the field gets keys when it scrolls back")
	assert.Equal(t, "", app.items.values[0])
}

func TestFocusManager_EmptyIDDoesNotLetSecondAutoFocus(t *testing.T) {
	fm := NewFocusManager()
	anon := &mockFocusable{id: ""}
	named := &mockFocusable{id: "named"}

	renderFocusables(fm, anon, named)
	assert.True(t, anon.focused)
	assert.False(t, named.focused, "only the first focusable is auto-focused")
}

// overflowWithButtonApp shows an overflowing Viewport of fields and, while
// showButton is set, a Button outside it.
type overflowWithButtonApp struct {
	vp         ViewportState
	items      *inputItems
	showButton bool
}

func (a *overflowWithButtonApp) View() View {
	list := Height(3, Viewport(&a.vp, a.items))
	if a.showButton {
		return Stack(list, Button("OK", func() {}).ID("ok"))
	}
	return Stack(list)
}

func TestFocusManager_RemovedButtonOutsideOverflowingViewportMovesFocus(t *testing.T) {
	app := &overflowWithButtonApp{items: &inputItems{values: make([]string, 10)}, showButton: true}
	r := NewRuntime(NewTestTerminal(40, 10, io.Discard), app, 30)
	r.render()
	r.focusMgr.SetFocus("ok")
	r.render()
	assert.Equal(t, "ok", r.focusMgr.GetFocusedID())

	app.showButton = false
	r.render()
	r.render()
	assert.Equal(t, "field-0", r.focusMgr.GetFocusedID(), "focus moves to the first focusable")

	r.processEvent(KeyEvent{Rune: 'x'})
	assert.Equal(t, "x", app.items.values[0], "keys reach the newly focused field")
}

func TestFocusManager_KeepsScrolledOutFocusAcrossResize(t *testing.T) {
	app := &focusViewportApp{items: &inputItems{values: make([]string, 10)}}
	r := NewRuntime(NewTestTerminal(40, 10, io.Discard), app, 30)
	r.render()
	r.focusMgr.SetFocus("field-1")
	r.render()

	app.vp.ScrollToBottom()
	r.render()
	app.width = 30
	r.render()
	r.render()
	assert.Equal(t, "field-1", r.focusMgr.GetFocusedID(), "focus survives a width change while scrolled out")
}

func TestFocusManager_RemovedViewportItemMovesFocus(t *testing.T) {
	app := &focusViewportApp{items: &inputItems{values: make([]string, 10)}}
	r := NewRuntime(NewTestTerminal(40, 10, io.Discard), app, 30)
	app.vp.ScrollToBottom()
	r.render()
	r.focusMgr.SetFocus("field-9")
	r.render()
	assert.Equal(t, "field-9", r.focusMgr.GetFocusedID())

	app.items.values = app.items.values[:9]
	r.render()
	r.render()
	assert.NotEqual(t, "field-9", r.focusMgr.GetFocusedID(), "a removed item does not keep focus")
	assert.NotNil(t, r.focusMgr.GetFocused(), "a remaining field has focus")
}

func TestFocusManager_UnnamedFocusableKeepsTabFocus(t *testing.T) {
	fm := NewFocusManager()
	a := &mockFocusable{id: "a"}
	anon := &mockFocusable{id: ""}
	b := &mockFocusable{id: "b"}

	renderFocusables(fm, a, anon, b)
	fm.HandleKey(KeyEvent{Key: KeyTab})
	renderFocusables(fm, a, anon, b)
	assert.True(t, anon.focused, "the unnamed focusable keeps focus after a re-render")
	assert.False(t, a.focused)
	assert.True(t, fm.GetFocused() == Focusable(anon))
	assert.Equal(t, fm.GetFocusedID(), "")

	fm.HandleKey(KeyEvent{Key: KeyTab})
	renderFocusables(fm, a, anon, b)
	assert.True(t, b.focused, "Tab moves past the unnamed focusable")
}

func TestFocusManager_TwoUnnamedFocusablesRouteKeysToFocused(t *testing.T) {
	fm := NewFocusManager()
	first := &mockFocusable{id: ""}
	second := &mockFocusable{id: ""}

	renderFocusables(fm, first, second)
	assert.True(t, first.focused)
	assert.False(t, second.focused)
	assert.True(t, fm.GetFocused() == Focusable(first))

	fm.HandleKey(KeyEvent{Key: KeyTab})
	renderFocusables(fm, first, second)
	assert.False(t, first.focused)
	assert.True(t, second.focused)
	assert.True(t, fm.GetFocused() == Focusable(second))
}

// unnamedListItems is Viewport content of one-row lists with no ID.
type unnamedListItems struct{ lists []*CheckboxListView }

func (it *unnamedListItems) Len() int        { return len(it.lists) }
func (it *unnamedListItems) Item(i int) View { return it.lists[i] }

type unnamedViewportApp struct {
	vp    ViewportState
	items *unnamedListItems
}

func (a *unnamedViewportApp) View() View { return Height(2, Viewport(&a.vp, a.items)) }

func TestFocusManager_UnnamedFocusableKeepsFocusWhenViewportScrolls(t *testing.T) {
	items := &unnamedListItems{}
	for range 4 {
		items.lists = append(items.lists, CheckboxListStrings([]string{"x"}, []bool{false}, new(int)))
	}
	app := &unnamedViewportApp{items: items}
	r := NewRuntime(NewTestTerminal(20, 5, io.Discard), app, 30)
	r.render()
	app.vp.ScrollBy(1) // items 1 and 2 show
	r.render()
	r.focusMgr.FocusNext() // focus was on item 0, now out of view
	r.render()
	assert.True(t, items.lists[1].IsFocused())
	r.processEvent(KeyEvent{Key: KeyTab})
	r.render()
	assert.True(t, items.lists[2].IsFocused())

	app.vp.ScrollBy(1) // items 2 and 3 show
	r.render()
	assert.True(t, items.lists[2].IsFocused(), "the focused item keeps focus after the scroll")
	assert.False(t, items.lists[3].IsFocused())
	assert.True(t, r.focusMgr.GetFocused() == Focusable(items.lists[2]))
}
