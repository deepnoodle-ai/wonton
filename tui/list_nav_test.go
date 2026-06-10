package tui

import (
	"fmt"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
	"github.com/deepnoodle-ai/wonton/termtest"
)

func TestListNavForKey(t *testing.T) {
	tests := []struct {
		name    string
		event   KeyEvent
		allowVi bool
		want    listNav
	}{
		{"arrow up", KeyEvent{Key: KeyArrowUp}, false, listNavUp},
		{"arrow down", KeyEvent{Key: KeyArrowDown}, false, listNavDown},
		{"page up", KeyEvent{Key: KeyPageUp}, false, listNavPageUp},
		{"page down", KeyEvent{Key: KeyPageDown}, false, listNavPageDown},
		{"home", KeyEvent{Key: KeyHome}, false, listNavHome},
		{"end", KeyEvent{Key: KeyEnd}, false, listNavEnd},
		{"k with vi", KeyEvent{Rune: 'k'}, true, listNavUp},
		{"j with vi", KeyEvent{Rune: 'j'}, true, listNavDown},
		{"g with vi", KeyEvent{Rune: 'g'}, true, listNavHome},
		{"G with vi", KeyEvent{Rune: 'G'}, true, listNavEnd},
		{"k without vi", KeyEvent{Rune: 'k'}, false, listNavNone},
		{"j without vi", KeyEvent{Rune: 'j'}, false, listNavNone},
		{"unrelated rune", KeyEvent{Rune: 'x'}, true, listNavNone},
		{"enter", KeyEvent{Key: KeyEnter}, true, listNavNone},
		{"alt+j is not nav", KeyEvent{Rune: 'j', Alt: true}, true, listNavNone},
		{"ctrl+k is not nav", KeyEvent{Rune: 'k', Ctrl: true}, true, listNavNone},
		{"alt+G is not nav", KeyEvent{Rune: 'G', Alt: true}, true, listNavNone},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, listNavForKey(tt.event, tt.allowVi))
		})
	}
}

func TestMoveListCursor(t *testing.T) {
	tests := []struct {
		name      string
		nav       listNav
		cursor    int
		count     int
		page      int
		want      int
		wantMoved bool
	}{
		{"up", listNavUp, 5, 10, 3, 4, true},
		{"up at top", listNavUp, 0, 10, 3, 0, false},
		{"down", listNavDown, 5, 10, 3, 6, true},
		{"down at bottom", listNavDown, 9, 10, 3, 9, false},
		{"page up", listNavPageUp, 5, 10, 3, 2, true},
		{"page up clamps", listNavPageUp, 1, 10, 5, 0, true},
		{"page down", listNavPageDown, 2, 10, 3, 5, true},
		{"page down clamps", listNavPageDown, 8, 10, 5, 9, true},
		{"home", listNavHome, 7, 10, 3, 0, true},
		{"home at top", listNavHome, 0, 10, 3, 0, false},
		{"end", listNavEnd, 2, 10, 3, 9, true},
		{"end at bottom", listNavEnd, 9, 10, 3, 9, false},
		{"empty list", listNavDown, 0, 0, 3, 0, false},
		{"none action", listNavNone, 4, 10, 3, 4, false},
		{"zero page treated as one", listNavPageDown, 0, 10, 0, 1, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, moved := moveListCursor(tt.nav, tt.cursor, tt.count, tt.page)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantMoved, moved)
		})
	}
}

func TestScrollIntoView(t *testing.T) {
	t.Run("cursor above window scrolls up", func(t *testing.T) {
		scroll := 5
		scrollIntoView(&scroll, 2, 4)
		assert.Equal(t, 2, scroll)
	})
	t.Run("cursor below window scrolls down", func(t *testing.T) {
		scroll := 0
		scrollIntoView(&scroll, 7, 4)
		assert.Equal(t, 4, scroll)
	})
	t.Run("cursor inside window unchanged", func(t *testing.T) {
		scroll := 2
		scrollIntoView(&scroll, 3, 4)
		assert.Equal(t, 2, scroll)
	})
	t.Run("nil scroll is no-op", func(t *testing.T) {
		scrollIntoView(nil, 3, 4)
	})
	t.Run("non-positive visible is no-op", func(t *testing.T) {
		scroll := 2
		scrollIntoView(&scroll, 9, 0)
		assert.Equal(t, 2, scroll)
	})
}

func navItems(n int) []ListItem {
	items := make([]ListItem, n)
	for i := range items {
		items[i] = ListItem{Label: "item", Value: i}
	}
	return items
}

func TestSelectListNavigationKeys(t *testing.T) {
	selected := 0
	list := SelectList(navItems(20), &selected).Height(5)

	assert.True(t, list.HandleKeyEvent(KeyEvent{Rune: 'j'}))
	assert.Equal(t, 1, selected)
	assert.True(t, list.HandleKeyEvent(KeyEvent{Rune: 'k'}))
	assert.Equal(t, 0, selected)
	assert.True(t, list.HandleKeyEvent(KeyEvent{Key: KeyPageDown}))
	assert.Equal(t, 5, selected)
	assert.True(t, list.HandleKeyEvent(KeyEvent{Key: KeyEnd}))
	assert.Equal(t, 19, selected)
	// Scroll follows the cursor so the selected item stays visible.
	assert.Equal(t, 15, list.scroll)
	assert.True(t, list.HandleKeyEvent(KeyEvent{Key: KeyHome}))
	assert.Equal(t, 0, selected)
	assert.Equal(t, 0, list.scroll)
	// At the top, up does not move and propagates.
	assert.False(t, list.HandleKeyEvent(KeyEvent{Key: KeyArrowUp}))
}

func labeledItems(n int) []ListItem {
	items := make([]ListItem, n)
	for i := range items {
		items[i] = ListItem{Label: fmt.Sprintf("item-%02d", i), Value: i}
	}
	return items
}

func TestSelectListRendersSelectedItemAfterEnd(t *testing.T) {
	selected := 0
	list := SelectList(labeledItems(20), &selected).Height(5)

	assert.True(t, list.HandleKeyEvent(KeyEvent{Key: KeyEnd}))
	assert.Equal(t, 19, selected)

	screen := SprintScreen(list, WithWidth(20), WithHeight(5))
	termtest.AssertRowContains(t, screen, 4, "item-19")
}

func TestSelectListRenderFollowsProgrammaticSelection(t *testing.T) {
	// The app moves the selection without a key event; render must still
	// bring it into view.
	selected := 19
	list := SelectList(labeledItems(20), &selected).Height(5)

	screen := SprintScreen(list, WithWidth(20), WithHeight(5))
	termtest.AssertRowContains(t, screen, 4, "item-19")
}

func TestCheckboxListRendersCursorAfterEnd(t *testing.T) {
	cursor := 0
	checked := make([]bool, 20)
	list := CheckboxList(labeledItems(20), checked, &cursor).Height(5)

	assert.True(t, list.HandleKeyEvent(KeyEvent{Key: KeyEnd}))
	assert.Equal(t, 19, cursor)

	screen := SprintScreen(list, WithWidth(20), WithHeight(5))
	termtest.AssertRowContains(t, screen, 4, "item-19")
}

func TestRadioListRendersSelectedAfterEnd(t *testing.T) {
	selected := 0
	list := RadioList(labeledItems(20), &selected).Height(5)

	assert.True(t, list.HandleKeyEvent(KeyEvent{Key: KeyEnd}))
	assert.Equal(t, 19, selected)

	screen := SprintScreen(list, WithWidth(20), WithHeight(5))
	termtest.AssertRowContains(t, screen, 4, "item-19")
}

func TestTreeEndScrollsWithoutScrollYBinding(t *testing.T) {
	root := &TreeNode{Label: "root", Expanded: true}
	for i := 0; i < 10; i++ {
		root.Children = append(root.Children, &TreeNode{Label: fmt.Sprintf("child-%02d", i)})
	}
	tree := Tree(root).Height(5)

	assert.True(t, tree.HandleKeyEvent(KeyEvent{Key: KeyEnd}))
	assert.Equal(t, root.Children[9], tree.selected)

	// With no external ScrollY binding, the internal offset keeps the
	// selected node visible.
	screen := SprintScreen(tree, WithWidth(20), WithHeight(5))
	termtest.AssertRowContains(t, screen, 4, "child-09")
}

func TestTableNavigationKeysScroll(t *testing.T) {
	selected := 0
	rows := make([][]string, 30)
	for i := range rows {
		rows[i] = []string{"cell"}
	}
	table := Table([]TableColumn{{Title: "Col"}}, &selected).Rows(rows).Height(11)

	assert.True(t, table.HandleKeyEvent(KeyEvent{Key: KeyPageDown}))
	assert.True(t, selected > 0)
	assert.True(t, table.HandleKeyEvent(KeyEvent{Key: KeyEnd}))
	assert.Equal(t, 29, selected)
	// Scroll followed the cursor to keep it visible.
	assert.True(t, table.scrollY > 0)
	assert.True(t, table.HandleKeyEvent(KeyEvent{Key: KeyHome}))
	assert.Equal(t, 0, selected)
	assert.Equal(t, 0, table.scrollY)
}

func TestCheckboxListNavigationKeys(t *testing.T) {
	cursor := 0
	checked := make([]bool, 10)
	list := CheckboxList(navItems(10), checked, &cursor)

	assert.True(t, list.HandleKeyEvent(KeyEvent{Rune: 'j'}))
	assert.Equal(t, 1, cursor)
	assert.True(t, list.HandleKeyEvent(KeyEvent{Key: KeyEnd}))
	assert.Equal(t, 9, cursor)
	// Space still toggles after navigation.
	assert.True(t, list.HandleKeyEvent(KeyEvent{Rune: ' '}))
	assert.True(t, checked[9])
}

func TestRadioListNavigationKeys(t *testing.T) {
	selected := 0
	list := RadioList(navItems(10), &selected)

	assert.True(t, list.HandleKeyEvent(KeyEvent{Rune: 'j'}))
	assert.Equal(t, 1, selected)
	assert.True(t, list.HandleKeyEvent(KeyEvent{Rune: 'G'}))
	assert.Equal(t, 9, selected)
}

func TestTreeNavigationKeys(t *testing.T) {
	root := &TreeNode{Label: "root", Expanded: true}
	for i := 0; i < 10; i++ {
		root.Children = append(root.Children, &TreeNode{Label: "child"})
	}
	tree := Tree(root).Height(5)

	assert.True(t, tree.HandleKeyEvent(KeyEvent{Rune: 'j'}))
	assert.Equal(t, root.Children[0], tree.selected)
	assert.True(t, tree.HandleKeyEvent(KeyEvent{Key: KeyEnd}))
	assert.Equal(t, root.Children[9], tree.selected)
	assert.True(t, tree.HandleKeyEvent(KeyEvent{Key: KeyHome}))
	assert.Equal(t, root, tree.selected)
}

func TestFilterableListViKeysDisabledWithFilter(t *testing.T) {
	selected := 0
	filter := ""
	list := FilterableList(navItems(10), &selected).Filter(&filter)

	// With a filter bound, j must be typed into the filter, not navigate.
	assert.True(t, list.HandleKeyEvent(KeyEvent{Rune: 'j'}))
	assert.Equal(t, 0, selected)
	assert.Equal(t, "j", filter)

	// Without a filter, vi keys and paging navigate.
	list2sel := 0
	list2 := FilterableList(navItems(30), &list2sel)
	assert.True(t, list2.HandleKeyEvent(KeyEvent{Key: KeyPageDown}))
	assert.True(t, list2sel > 0)
	pos := list2sel
	assert.True(t, list2.HandleKeyEvent(KeyEvent{Rune: 'j'}))
	assert.Equal(t, pos+1, list2sel)
}
