package tui

import (
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

// At a tight height the selected item must stay visible; scroll indicators
// must not overwrite it.
func TestFilterableListTightHeightKeepsItem(t *testing.T) {
	selected := 2
	scroll := 0
	view := FilterableListStrings([]string{"A", "B", "Cherry-long-label", "D", "E"}, &selected).
		ScrollOffset(&scroll).Height(2).Width(20)
	screen := SprintScreen(view, WithWidth(20), WithHeight(2))
	if !screen.Contains("Cherry-long-label") {
		t.Fatalf("selected item hidden:\n%s", screen.Text())
	}
}

// Backspace removes a whole rune, and non-ASCII runes can be typed.
func TestFilterableListFilterUTF8(t *testing.T) {
	selected := 0
	filter := "é"
	view := FilterableListStrings([]string{"café"}, &selected).Filter(&filter)
	view.HandleKeyEvent(KeyEvent{Key: KeyBackspace})
	assert.Equal(t, "", filter)

	assert.True(t, view.HandleKeyEvent(KeyEvent{Rune: 'é'}))
	assert.Equal(t, "é", filter)
}

// Enter and a mouse click report the same index and both mark the item chosen.
func TestFilterableListSelectIndexConsistent(t *testing.T) {
	items := []string{"Apple", "Banana", "Cherry"}
	filter := "an" // filtered list: Banana only
	var enterIdx, clickIdx int

	selected := 0
	view := FilterableListStrings(items, &selected).Filter(&filter).Height(5).
		OnSelect(func(_ ListItem, i int) { enterIdx = i })
	SprintScreen(view, WithWidth(20), WithHeight(5))
	view.HandleKeyEvent(KeyEvent{Key: KeyEnter})

	selected = 0
	var chosen []int
	reg := newRegistries()
	view = FilterableListStrings(items, &selected).Filter(&filter).Height(5).Chosen(&chosen).
		OnSelect(func(_ ListItem, i int) { clickIdx = i })
	renderWithRegistries(t, view, reg)
	// Row 0 is the filter input, row 1 the divider, row 2 the first item.
	reg.interactive.HandleClick(1, 2)

	assert.Equal(t, 1, enterIdx)
	assert.Equal(t, 1, clickIdx)
	assert.Equal(t, []int{1}, chosen)
}
