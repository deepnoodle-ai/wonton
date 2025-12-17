package tui

import (
	"image"
	"testing"

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
