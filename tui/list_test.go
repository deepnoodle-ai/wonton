package tui

import (
	"image"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestNewList(t *testing.T) {
	items := []ListItem{
		{Label: "Item 1", Value: 1},
		{Label: "Item 2", Value: 2},
	}

	list := NewList(items)

	assert.Equal(t, 2, len(list.Items))
	assert.Equal(t, 0, list.SelectedIndex)
	assert.Equal(t, 0, list.ScrollOffset)
}

func TestList_SetItems(t *testing.T) {
	list := NewList([]ListItem{
		{Label: "Item 1"},
		{Label: "Item 2"},
		{Label: "Item 3"},
	})

	// Select the last item
	list.SelectedIndex = 2

	// Set fewer items
	list.SetItems([]ListItem{
		{Label: "New Item 1"},
	})

	// Selected index should be clamped
	assert.Equal(t, 0, list.SelectedIndex)
}

func TestList_SetItems_Empty(t *testing.T) {
	list := NewList([]ListItem{
		{Label: "Item 1"},
	})

	list.SetItems([]ListItem{})
	assert.Equal(t, -1, list.SelectedIndex)
}

func TestList_SelectNext(t *testing.T) {
	list := NewList([]ListItem{
		{Label: "Item 1"},
		{Label: "Item 2"},
		{Label: "Item 3"},
	})

	assert.Equal(t, 0, list.SelectedIndex)

	list.SelectNext()
	assert.Equal(t, 1, list.SelectedIndex)

	list.SelectNext()
	assert.Equal(t, 2, list.SelectedIndex)

	// Should not go past the end
	list.SelectNext()
	assert.Equal(t, 2, list.SelectedIndex)
}

func TestList_SelectPrev(t *testing.T) {
	list := NewList([]ListItem{
		{Label: "Item 1"},
		{Label: "Item 2"},
		{Label: "Item 3"},
	})

	list.SelectedIndex = 2

	list.SelectPrev()
	assert.Equal(t, 1, list.SelectedIndex)

	list.SelectPrev()
	assert.Equal(t, 0, list.SelectedIndex)

	// Should not go below 0
	list.SelectPrev()
	assert.Equal(t, 0, list.SelectedIndex)
}

func TestList_GetSelected(t *testing.T) {
	items := []ListItem{
		{Label: "Item 1", Value: 1},
		{Label: "Item 2", Value: 2},
	}
	list := NewList(items)

	selected := list.GetSelected()
	assert.NotNil(t, selected)
	assert.Equal(t, "Item 1", selected.Label)

	list.SelectedIndex = 1
	selected = list.GetSelected()
	assert.Equal(t, "Item 2", selected.Label)
}

func TestList_GetSelected_Empty(t *testing.T) {
	list := NewList([]ListItem{})
	assert.Nil(t, list.GetSelected())
}

func TestList_GetSelected_InvalidIndex(t *testing.T) {
	list := NewList([]ListItem{{Label: "Item"}})
	list.SelectedIndex = -1
	assert.Nil(t, list.GetSelected())

	list.SelectedIndex = 10
	assert.Nil(t, list.GetSelected())
}

func TestList_SetStyles(t *testing.T) {
	list := NewList([]ListItem{})
	normalStyle := NewStyle().WithForeground(ColorRed)
	selectedStyle := NewStyle().WithForeground(ColorGreen)

	list.SetStyles(normalStyle, selectedStyle)

	assert.Equal(t, normalStyle, list.NormalStyle)
	assert.Equal(t, selectedStyle, list.SelectedStyle)
}

func TestList_HandleKey(t *testing.T) {
	items := []ListItem{
		{Label: "Item 1"},
		{Label: "Item 2"},
		{Label: "Item 3"},
	}
	list := NewList(items)
	list.SetBounds(image.Rect(0, 0, 20, 10))

	t.Run("arrow down", func(t *testing.T) {
		list.SelectedIndex = 0
		handled := list.HandleKey(KeyEvent{Key: KeyArrowDown})
		assert.True(t, handled)
		assert.Equal(t, 1, list.SelectedIndex)
	})

	t.Run("arrow up", func(t *testing.T) {
		list.SelectedIndex = 1
		handled := list.HandleKey(KeyEvent{Key: KeyArrowUp})
		assert.True(t, handled)
		assert.Equal(t, 0, list.SelectedIndex)
	})

	t.Run("home", func(t *testing.T) {
		list.SelectedIndex = 2
		handled := list.HandleKey(KeyEvent{Key: KeyHome})
		assert.True(t, handled)
		assert.Equal(t, 0, list.SelectedIndex)
	})

	t.Run("end", func(t *testing.T) {
		list.SelectedIndex = 0
		handled := list.HandleKey(KeyEvent{Key: KeyEnd})
		assert.True(t, handled)
		assert.Equal(t, 2, list.SelectedIndex)
	})

	t.Run("enter triggers callback", func(t *testing.T) {
		var selectedItem ListItem
		list.OnSelect = func(item ListItem) {
			selectedItem = item
		}
		list.SelectedIndex = 1
		handled := list.HandleKey(KeyEvent{Key: KeyEnter})
		assert.True(t, handled)
		assert.Equal(t, "Item 2", selectedItem.Label)
	})

	t.Run("unhandled key", func(t *testing.T) {
		handled := list.HandleKey(KeyEvent{Key: KeyTab})
		assert.False(t, handled)
	})
}

func TestList_HandleMouse(t *testing.T) {
	items := []ListItem{
		{Label: "Item 1"},
		{Label: "Item 2"},
		{Label: "Item 3"},
	}
	list := NewList(items)
	list.SetBounds(image.Rect(0, 0, 20, 10))

	t.Run("click selects item", func(t *testing.T) {
		var clickedItem ListItem
		list.OnSelect = func(item ListItem) {
			clickedItem = item
		}
		handled := list.HandleMouse(MouseEvent{
			Type: MouseClick,
			X:    5,
			Y:    1,
		})
		assert.True(t, handled)
		assert.Equal(t, 1, list.SelectedIndex)
		assert.Equal(t, "Item 2", clickedItem.Label)
	})

	t.Run("click outside bounds", func(t *testing.T) {
		handled := list.HandleMouse(MouseEvent{
			Type: MouseClick,
			X:    25,
			Y:    5,
		})
		assert.False(t, handled)
	})

	t.Run("scroll wheel down", func(t *testing.T) {
		list.ScrollOffset = 0
		handled := list.HandleMouse(MouseEvent{
			Type:   MouseScroll,
			Button: MouseButtonWheelDown,
			X:      5,
			Y:      1,
		})
		assert.True(t, handled)
		assert.Equal(t, 1, list.ScrollOffset)
	})

	t.Run("scroll wheel up", func(t *testing.T) {
		list.ScrollOffset = 1
		handled := list.HandleMouse(MouseEvent{
			Type:   MouseScroll,
			Button: MouseButtonWheelUp,
			X:      5,
			Y:      1,
		})
		assert.True(t, handled)
		assert.Equal(t, 0, list.ScrollOffset)
	})
}
