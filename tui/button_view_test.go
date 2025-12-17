package tui

import (
	"image"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
	"github.com/deepnoodle-ai/wonton/termtest"
)

func TestButtonState_Focusable(t *testing.T) {
	state := &buttonState{
		id:      "test-btn",
		bounds:  image.Rect(0, 0, 10, 1),
		focused: false,
	}

	assert.Equal(t, "test-btn", state.FocusID())
	assert.False(t, state.IsFocused())

	state.SetFocused(true)
	assert.True(t, state.IsFocused())

	state.SetFocused(false)
	assert.False(t, state.IsFocused())
}

func TestButtonState_FocusBounds(t *testing.T) {
	bounds := image.Rect(5, 10, 20, 15)
	state := &buttonState{
		id:     "test-btn",
		bounds: bounds,
	}

	assert.Equal(t, bounds, state.FocusBounds())
}

func TestButtonState_HandleKeyEvent_Enter(t *testing.T) {
	called := false
	state := &buttonState{
		id: "test-btn",
		callback: func() {
			called = true
		},
	}

	// Enter key should activate button
	handled := state.HandleKeyEvent(KeyEvent{Key: KeyEnter})
	assert.True(t, handled)
	assert.True(t, called)
}

func TestButtonState_HandleKeyEvent_Space(t *testing.T) {
	called := false
	state := &buttonState{
		id: "test-btn",
		callback: func() {
			called = true
		},
	}

	// Space should activate button
	handled := state.HandleKeyEvent(KeyEvent{Rune: ' '})
	assert.True(t, handled)
	assert.True(t, called)
}

func TestButtonState_HandleKeyEvent_OtherKeys(t *testing.T) {
	called := false
	state := &buttonState{
		id: "test-btn",
		callback: func() {
			called = true
		},
	}

	// Other keys should not activate button
	handled := state.HandleKeyEvent(KeyEvent{Rune: 'a'})
	assert.False(t, handled)
	assert.False(t, called)

	handled = state.HandleKeyEvent(KeyEvent{Key: KeyArrowDown})
	assert.False(t, handled)
	assert.False(t, called)
}

func TestButtonState_HandleKeyEvent_NilCallback(t *testing.T) {
	state := &buttonState{
		id:       "test-btn",
		callback: nil,
	}

	// Should not panic with nil callback
	handled := state.HandleKeyEvent(KeyEvent{Key: KeyEnter})
	assert.True(t, handled)
}

func TestButtonRegistry_Register(t *testing.T) {
	// Reset registry state for test
	registry := &buttonRegistryImpl{
		buttons: make(map[string]*buttonState),
	}

	called := false
	bounds := image.Rect(0, 0, 10, 1)
	callback := func() { called = true }
	focusStyle := NewStyle().WithReverse()

	state := registry.Register("btn1", bounds, callback, focusStyle)

	assert.NotNil(t, state)
	assert.Equal(t, "btn1", state.id)
	assert.Equal(t, bounds, state.bounds)
	assert.Equal(t, focusStyle, state.focusStyle)

	// Invoke callback to verify it was set
	state.callback()
	assert.True(t, called)
}

func TestButtonRegistry_Register_UpdateExisting(t *testing.T) {
	registry := &buttonRegistryImpl{
		buttons: make(map[string]*buttonState),
	}

	// First registration
	bounds1 := image.Rect(0, 0, 10, 1)
	state1 := registry.Register("btn1", bounds1, nil, NewStyle())
	originalID := state1.id

	// Second registration with same ID should update the same state object
	bounds2 := image.Rect(5, 5, 15, 6)
	callbackCalled := false
	newCallback := func() { callbackCalled = true }
	newStyle := NewStyle().WithBold()
	state2 := registry.Register("btn1", bounds2, newCallback, newStyle)

	// Should be same state object (same pointer)
	assert.True(t, state1 == state2, "Expected same state object")
	assert.Equal(t, originalID, state2.id)
	assert.Equal(t, bounds2, state2.bounds)
	assert.Equal(t, newStyle, state2.focusStyle)

	// Verify callback was updated
	state2.callback()
	assert.True(t, callbackCalled)
}

func TestButtonRegistry_Clear(t *testing.T) {
	registry := &buttonRegistryImpl{
		buttons: make(map[string]*buttonState),
	}

	// Register a button
	registry.Register("btn1", image.Rect(0, 0, 10, 1), nil, NewStyle())

	// Clear should not panic
	registry.Clear()

	// Buttons map should be cleared to prevent memory leaks
	assert.Equal(t, 0, len(registry.buttons))
}

func TestInteractiveRegistry_RegisterRegion(t *testing.T) {
	registry := &interactiveRegistryImpl{
		regions: make([]interactiveRegion, 0),
	}

	bounds := image.Rect(0, 0, 10, 1)
	called := false
	callback := func() { called = true }

	registry.RegisterRegion(bounds, callback)

	assert.Equal(t, 1, len(registry.regions))
	assert.Equal(t, bounds, registry.regions[0].bounds)

	// Invoke callback to verify it was set
	registry.regions[0].callback()
	assert.True(t, called)
}

func TestInteractiveRegistry_RegisterButton(t *testing.T) {
	registry := &interactiveRegistryImpl{
		regions: make([]interactiveRegion, 0),
	}

	bounds := image.Rect(0, 0, 10, 1)
	callback := func() {}

	// RegisterButton is an alias for RegisterRegion
	registry.RegisterButton(bounds, callback)

	assert.Equal(t, 1, len(registry.regions))
	assert.Equal(t, bounds, registry.regions[0].bounds)
}

func TestInteractiveRegistry_Clear(t *testing.T) {
	registry := &interactiveRegistryImpl{
		regions: make([]interactiveRegion, 0),
	}

	// Register some regions
	registry.RegisterRegion(image.Rect(0, 0, 10, 1), func() {})
	registry.RegisterRegion(image.Rect(10, 0, 20, 1), func() {})

	assert.Equal(t, 2, len(registry.regions))

	// Clear should remove all regions
	registry.Clear()
	assert.Equal(t, 0, len(registry.regions))
}

func TestInteractiveRegistry_HandleClick_Hit(t *testing.T) {
	registry := &interactiveRegistryImpl{
		regions: make([]interactiveRegion, 0),
	}

	clicked := ""
	registry.RegisterRegion(image.Rect(0, 0, 10, 5), func() { clicked = "first" })
	registry.RegisterRegion(image.Rect(15, 0, 25, 5), func() { clicked = "second" })

	// Click inside first region
	handled := registry.HandleClick(5, 2)
	assert.True(t, handled)
	assert.Equal(t, "first", clicked)

	// Click inside second region
	handled = registry.HandleClick(20, 3)
	assert.True(t, handled)
	assert.Equal(t, "second", clicked)
}

func TestInteractiveRegistry_HandleClick_Miss(t *testing.T) {
	registry := &interactiveRegistryImpl{
		regions: make([]interactiveRegion, 0),
	}

	called := false
	registry.RegisterRegion(image.Rect(0, 0, 10, 5), func() { called = true })

	// Click outside region
	handled := registry.HandleClick(15, 2)
	assert.False(t, handled)
	assert.False(t, called)
}

func TestInteractiveRegistry_HandleClick_Empty(t *testing.T) {
	registry := &interactiveRegistryImpl{
		regions: make([]interactiveRegion, 0),
	}

	// Click with no regions registered
	handled := registry.HandleClick(5, 2)
	assert.False(t, handled)
}

func TestButton_Creation(t *testing.T) {
	callback := func() {}
	btn := Button("Click Me", callback)

	assert.NotNil(t, btn)
	assert.Equal(t, "Click Me", btn.label)
	assert.NotNil(t, btn.callback)
	assert.Equal(t, 0, btn.width) // Default width
}

func TestButton_ID(t *testing.T) {
	btn := Button("Test", func() {}).ID("custom-id")
	assert.Equal(t, "custom-id", btn.id)
}

func TestButton_Fg(t *testing.T) {
	btn := Button("Test", func() {}).Fg(ColorRed)
	assert.Equal(t, ColorRed, btn.style.Foreground)
}

func TestButton_Bg(t *testing.T) {
	btn := Button("Test", func() {}).Bg(ColorBlue)
	assert.Equal(t, ColorBlue, btn.style.Background)
}

func TestButton_Bold(t *testing.T) {
	btn := Button("Test", func() {}).Bold()
	assert.True(t, btn.style.Bold)
}

func TestButton_Reverse(t *testing.T) {
	btn := Button("Test", func() {}).Reverse()
	assert.True(t, btn.style.Reverse)
}

func TestButton_Style(t *testing.T) {
	style := NewStyle().WithForeground(ColorGreen).WithBold()
	btn := Button("Test", func() {}).Style(style)
	assert.Equal(t, style, btn.style)
}

func TestButton_FocusStyle(t *testing.T) {
	focusStyle := NewStyle().WithForeground(ColorYellow).WithReverse()
	btn := Button("Test", func() {}).FocusStyle(focusStyle)
	assert.Equal(t, focusStyle, btn.focusStyle)
}

func TestButton_Width(t *testing.T) {
	btn := Button("Test", func() {}).Width(20)
	assert.Equal(t, 20, btn.width)
}

func TestButton_Size(t *testing.T) {
	btn := Button("Hello", func() {})

	// Size without fixed width
	w, h := btn.size(100, 100)
	assert.Equal(t, 5, w) // "Hello" is 5 chars
	assert.Equal(t, 1, h)

	// Size with fixed width
	btn.Width(20)
	w, h = btn.size(100, 100)
	assert.Equal(t, 20, w)
	assert.Equal(t, 1, h)

	// Size constrained by maxWidth
	w, h = btn.size(10, 100)
	assert.Equal(t, 10, w)
	assert.Equal(t, 1, h)
}

func TestButton_Chaining(t *testing.T) {
	btn := Button("Submit", func() {}).
		ID("submit-btn").
		Fg(ColorWhite).
		Bg(ColorBlue).
		Bold().
		Width(15)

	assert.Equal(t, "submit-btn", btn.id)
	assert.Equal(t, "Submit", btn.label)
	assert.Equal(t, ColorWhite, btn.style.Foreground)
	assert.Equal(t, ColorBlue, btn.style.Background)
	assert.True(t, btn.style.Bold)
	assert.Equal(t, 15, btn.width)
}

func TestClickable_Creation(t *testing.T) {
	callback := func() {}
	c := Clickable("Link", callback)

	assert.NotNil(t, c)
	assert.Equal(t, "Link", c.label)
	assert.NotNil(t, c.callback)
	assert.Equal(t, 0, c.width) // Default width
}

func TestClickable_Fg(t *testing.T) {
	c := Clickable("Test", func() {}).Fg(ColorRed)
	assert.Equal(t, ColorRed, c.style.Foreground)
}

func TestClickable_Bg(t *testing.T) {
	c := Clickable("Test", func() {}).Bg(ColorBlue)
	assert.Equal(t, ColorBlue, c.style.Background)
}

func TestClickable_Bold(t *testing.T) {
	c := Clickable("Test", func() {}).Bold()
	assert.True(t, c.style.Bold)
}

func TestClickable_Reverse(t *testing.T) {
	c := Clickable("Test", func() {}).Reverse()
	assert.True(t, c.style.Reverse)
}

func TestClickable_Style(t *testing.T) {
	style := NewStyle().WithForeground(ColorGreen).WithBold()
	c := Clickable("Test", func() {}).Style(style)
	assert.Equal(t, style, c.style)
}

func TestClickable_Width(t *testing.T) {
	c := Clickable("Test", func() {}).Width(25)
	assert.Equal(t, 25, c.width)
}

func TestClickable_Size(t *testing.T) {
	c := Clickable("Click Here", func() {})

	// Size without fixed width
	w, h := c.size(100, 100)
	assert.Equal(t, 10, w) // "Click Here" is 10 chars
	assert.Equal(t, 1, h)

	// Size with fixed width
	c.Width(30)
	w, h = c.size(100, 100)
	assert.Equal(t, 30, w)
	assert.Equal(t, 1, h)

	// Size constrained by maxWidth
	w, h = c.size(15, 100)
	assert.Equal(t, 15, w)
	assert.Equal(t, 1, h)
}

func TestClickable_Chaining(t *testing.T) {
	c := Clickable("Open Link", func() {}).
		Fg(ColorCyan).
		Bg(ColorDefault).
		Bold().
		Width(20)

	assert.Equal(t, "Open Link", c.label)
	assert.Equal(t, ColorCyan, c.style.Foreground)
	assert.Equal(t, ColorDefault, c.style.Background)
	assert.True(t, c.style.Bold)
	assert.Equal(t, 20, c.width)
}

// Render tests using termtest with SprintScreen helper

func TestButton_Render(t *testing.T) {
	btn := Button("Submit", func() {})
	screen := SprintScreen(btn, WithWidth(20))
	termtest.AssertRowContains(t, screen, 0, "Submit")
}

func TestButton_Render_WithStyle(t *testing.T) {
	btn := Button("OK", func() {}).Fg(ColorGreen).Bold()
	screen := SprintScreen(btn, WithWidth(20))

	termtest.AssertRowContains(t, screen, 0, "OK")

	// Check that first character has bold style
	cell := screen.Cell(0, 0)
	assert.True(t, cell.Style.Bold)
}

func TestButton_Render_InStack(t *testing.T) {
	stack := Stack(
		Text("Form:"),
		Button("Save", func() {}),
		Button("Cancel", func() {}),
	)
	screen := SprintScreen(stack, WithWidth(20))

	termtest.AssertRowContains(t, screen, 0, "Form:")
	termtest.AssertRowContains(t, screen, 1, "Save")
	termtest.AssertRowContains(t, screen, 2, "Cancel")
}

func TestClickable_Render(t *testing.T) {
	c := Clickable("Click me", func() {})
	screen := SprintScreen(c, WithWidth(20))
	termtest.AssertRowContains(t, screen, 0, "Click me")
}

func TestClickable_Render_WithStyle(t *testing.T) {
	c := Clickable("Link", func() {}).Fg(ColorCyan).Bold()
	screen := SprintScreen(c, WithWidth(20))

	termtest.AssertRowContains(t, screen, 0, "Link")

	// Check that first character has bold style
	cell := screen.Cell(0, 0)
	assert.True(t, cell.Style.Bold)
}

func TestButton_Render_InGroup(t *testing.T) {
	group := Group(
		Button("Yes", func() {}),
		Button("No", func() {}),
	).Gap(2)
	screen := SprintScreen(group, WithWidth(20))

	termtest.AssertRowContains(t, screen, 0, "Yes")
	termtest.AssertRowContains(t, screen, 0, "No")
}
