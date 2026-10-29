package tui

import (
	"image"
	"strings"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
	"github.com/deepnoodle-ai/wonton/termtest"
)

func renderMenu(m *MenuState, width, height int) *termtest.Screen {
	return SprintScreen(Menu(m), WithWidth(width), WithHeight(height))
}

func testMenu(chosen *string) []MenuItem {
	pick := func(s string) func() { return func() { *chosen = s } }
	return []MenuItem{
		{Label: "Split right", Hint: "|", Action: pick("right")},
		{Label: "Split down", Hint: "-", Action: pick("down")},
		{Label: "Zoom", Disabled: true, Action: pick("zoom")},
		MenuSeparator(),
		{Label: "Close", Danger: true, Action: pick("close")},
	}
}

func TestMenuRendersAtAnchor(t *testing.T) {
	var chosen string
	m := &MenuState{}
	m.Open(2, 1, "", testMenu(&chosen))
	screen := renderMenu(m, 30, 10)
	assert.Equal(t, screen.Row(1), "  ╭─────────────────╮")
	assert.Equal(t, screen.Row(2), "  │ Split right   | │")
	assert.Equal(t, screen.Row(3), "  │ Split down    - │")
	assert.Equal(t, screen.Row(4), "  │ Zoom            │")
	assert.Equal(t, screen.Row(5), "  ├─────────────────┤")
	assert.Equal(t, screen.Row(6), "  │ Close           │")
	assert.Equal(t, screen.Row(7), "  ╰─────────────────╯")
	assert.Equal(t, m.Bounds(), image.Rect(2, 1, 21, 8))
	assert.Equal(t, m.Selected(), 0)
}

func TestMenuFlipsToFitScreen(t *testing.T) {
	var chosen string
	m := &MenuState{}
	m.Open(28, 9, "", testMenu(&chosen))
	renderMenu(m, 30, 10)
	// Opened near the bottom-right corner, it grows up and to the left.
	assert.Equal(t, m.Bounds(), image.Rect(10, 3, 29, 10))
}

func TestMenuTitle(t *testing.T) {
	m := &MenuState{}
	m.Open(0, 0, "pane: zsh", []MenuItem{{Label: "Close"}})
	screen := renderMenu(m, 20, 6)
	assert.Equal(t, screen.Row(1), "│ pane: zsh │")
	assert.Equal(t, screen.Row(2), "├───────────┤")
	assert.Equal(t, screen.Row(3), "│ Close     │")
}

func TestMenuKeyboard(t *testing.T) {
	var chosen string
	m := &MenuState{}
	m.Open(0, 0, "", testMenu(&chosen))

	assert.True(t, m.HandleKey(KeyEvent{Key: KeyArrowDown}))
	assert.Equal(t, m.Selected(), 1)
	// Down skips the disabled entry and the separator.
	m.HandleKey(KeyEvent{Key: KeyArrowDown})
	assert.Equal(t, m.Selected(), 4)
	// And wraps.
	m.HandleKey(KeyEvent{Key: KeyArrowDown})
	assert.Equal(t, m.Selected(), 0)
	m.HandleKey(KeyEvent{Key: KeyArrowUp})
	assert.Equal(t, m.Selected(), 4)

	// Any key is consumed while open, even ones the menu ignores.
	assert.True(t, m.HandleKey(KeyEvent{Key: KeyF5}))

	m.HandleKey(KeyEvent{Key: KeyEnter})
	assert.Equal(t, chosen, "close")
	assert.False(t, m.IsOpen())
	assert.False(t, m.HandleKey(KeyEvent{Key: KeyEnter}))
}

func TestMenuLetterJump(t *testing.T) {
	var chosen string
	m := &MenuState{}
	m.Open(0, 0, "", testMenu(&chosen))
	// Two entries start with 's': the letter selects the next one.
	m.HandleKey(KeyEvent{Rune: 's'})
	assert.Equal(t, m.Selected(), 1)
	assert.True(t, m.IsOpen())
	// One starts with 'c': the letter chooses it.
	m.HandleKey(KeyEvent{Rune: 'C'})
	assert.Equal(t, chosen, "close")
	assert.False(t, m.IsOpen())
}

func TestMenuEscapeCloses(t *testing.T) {
	var chosen string
	m := &MenuState{}
	m.Open(0, 0, "", testMenu(&chosen))
	m.HandleKey(KeyEvent{Key: KeyEscape})
	assert.False(t, m.IsOpen())
	assert.Equal(t, chosen, "")
}

func TestMenuMouse(t *testing.T) {
	var chosen string
	m := &MenuState{}
	m.Open(2, 1, "", testMenu(&chosen))
	renderMenu(m, 30, 10)

	// Hovering selects; hovering a disabled entry does not.
	assert.True(t, m.HandleMouse(MouseEvent{Type: MouseMove, X: 5, Y: 3}))
	assert.Equal(t, m.Selected(), 1)
	m.HandleMouse(MouseEvent{Type: MouseMove, X: 5, Y: 4})
	assert.Equal(t, m.Selected(), 1)

	// Clicking a disabled entry or a separator does nothing.
	m.HandleMouse(MouseEvent{Type: MouseClick, Button: MouseButtonLeft, X: 5, Y: 5})
	assert.True(t, m.IsOpen())

	m.HandleMouse(MouseEvent{Type: MouseClick, Button: MouseButtonLeft, X: 5, Y: 2})
	assert.Equal(t, chosen, "right")
	assert.False(t, m.IsOpen())
	assert.False(t, m.HandleMouse(MouseEvent{Type: MouseClick, Button: MouseButtonLeft, X: 5, Y: 2}))
}

func TestMenuPressOutsideCloses(t *testing.T) {
	var chosen string
	m := &MenuState{}
	m.Open(2, 1, "", testMenu(&chosen))
	renderMenu(m, 30, 10)
	// The press is consumed so it does not reach what is under it.
	assert.True(t, m.HandleMouse(MouseEvent{Type: MousePress, Button: MouseButtonLeft, X: 25, Y: 8}))
	assert.False(t, m.IsOpen())
	assert.Equal(t, chosen, "")
	// So are its release and the click that follows.
	assert.True(t, m.HandleMouse(MouseEvent{Type: MouseRelease, Button: MouseButtonLeft, X: 25, Y: 8}))
	assert.True(t, m.HandleMouse(MouseEvent{Type: MouseClick, Button: MouseButtonLeft, X: 25, Y: 8}))
	// The next gesture is the app's again.
	assert.False(t, m.HandleMouse(MouseEvent{Type: MouseMove, X: 20, Y: 8}))
	assert.False(t, m.HandleMouse(MouseEvent{Type: MousePress, Button: MouseButtonLeft, X: 20, Y: 8}))
	assert.False(t, m.HandleMouse(MouseEvent{Type: MouseClick, Button: MouseButtonLeft, X: 20, Y: 8}))
}

func TestMenuClosedDrawsNothing(t *testing.T) {
	m := &MenuState{}
	screen := renderMenu(m, 10, 3)
	assert.Equal(t, strings.TrimSpace(screen.Text()), "")
}
