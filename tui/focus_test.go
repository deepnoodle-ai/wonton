package tui

import (
	"image"
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
