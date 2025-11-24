package gooey

import (
	"image"
	"strings"

	"github.com/mattn/go-runewidth"
)

// ListItem represents an item in a list
type ListItem struct {
	Label string
	Value interface{}
	Icon  string // Optional icon/prefix
}

// List is a scrollable list widget
type List struct {
	BaseWidget
	Items         []ListItem
	SelectedIndex int
	ScrollOffset  int

	// Styles
	NormalStyle   Style
	SelectedStyle Style

	// Callbacks
	OnSelect func(item ListItem)

	// Internal
	focused bool
}

// NewList creates a new list widget
func NewList(items []ListItem) *List {
	l := &List{
		BaseWidget:    NewBaseWidget(),
		Items:         items,
		SelectedIndex: 0,
		NormalStyle:   NewStyle(),
		SelectedStyle: NewStyle().WithReverse(),
	}
	l.SetMinSize(image.Point{X: 10, Y: 5})
	return l
}

// SetItems updates the list items
func (l *List) SetItems(items []ListItem) {
	l.Items = items
	if l.SelectedIndex >= len(items) {
		l.SelectedIndex = len(items) - 1
	}
	if l.SelectedIndex < 0 && len(items) > 0 {
		l.SelectedIndex = 0
	}
	l.MarkDirty()
}

// Draw renders the list
func (l *List) Draw(frame RenderFrame) {
	bounds := l.GetBounds()
	width, height := bounds.Dx(), bounds.Dy()

	if height <= 0 {
		return
	}

	// Determine if we are in a SubFrame (width/height match frame size)
	frameW, frameH := frame.Size()
	drawX, drawY := bounds.Min.X, bounds.Min.Y
	if frameW == width && frameH == height {
		drawX, drawY = 0, 0
	}

	// Auto-scroll to keep selected item visible
	if l.SelectedIndex < l.ScrollOffset {
		l.ScrollOffset = l.SelectedIndex
	}
	if l.SelectedIndex >= l.ScrollOffset+height {
		l.ScrollOffset = l.SelectedIndex - height + 1
	}
	if l.ScrollOffset < 0 {
		l.ScrollOffset = 0
	}

	maxScroll := len(l.Items) - height
	if maxScroll < 0 {
		maxScroll = 0
	}
	if l.ScrollOffset > maxScroll {
		l.ScrollOffset = maxScroll
	}

	for i := 0; i < height; i++ {
		itemIndex := l.ScrollOffset + i
		if itemIndex >= len(l.Items) {
			// Draw empty lines
			frame.FillStyled(drawX, drawY+i, width, 1, ' ', l.NormalStyle)
			continue
		}

		item := l.Items[itemIndex]
		style := l.NormalStyle
		if itemIndex == l.SelectedIndex {
			style = l.SelectedStyle
		}

		// Draw item
		text := item.Label
		if item.Icon != "" {
			text = item.Icon + " " + text
		}

		// Truncate
		if runewidth.StringWidth(text) > width {
			text = runewidth.Truncate(text, width, "…")
		}

		// Pad to full width
		padding := width - runewidth.StringWidth(text)
		if padding > 0 {
			text += strings.Repeat(" ", padding)
		}

		frame.PrintStyled(drawX, drawY+i, text, style)
	}
}

// HandleKey handles key events
func (l *List) HandleKey(event KeyEvent) bool {
	switch event.Key {
	case KeyArrowUp:
		l.SelectPrev()
		return true
	case KeyArrowDown:
		l.SelectNext()
		return true
	case KeyHome:
		l.SelectedIndex = 0
		l.MarkDirty()
		return true
	case KeyEnd:
		l.SelectedIndex = len(l.Items) - 1
		l.MarkDirty()
		return true
	case KeyPageUp:
		l.SelectedIndex -= l.GetBounds().Dy()
		if l.SelectedIndex < 0 {
			l.SelectedIndex = 0
		}
		l.MarkDirty()
		return true
	case KeyPageDown:
		l.SelectedIndex += l.GetBounds().Dy()
		if l.SelectedIndex >= len(l.Items) {
			l.SelectedIndex = len(l.Items) - 1
		}
		l.MarkDirty()
		return true
	case KeyEnter:
		if l.OnSelect != nil && l.SelectedIndex >= 0 && l.SelectedIndex < len(l.Items) {
			l.OnSelect(l.Items[l.SelectedIndex])
		}
		return true
	}
	return false
}

// SelectNext selects the next item
func (l *List) SelectNext() {
	if l.SelectedIndex < len(l.Items)-1 {
		l.SelectedIndex++
		l.MarkDirty()
	}
}

// SelectPrev selects the previous item
func (l *List) SelectPrev() {
	if l.SelectedIndex > 0 {
		l.SelectedIndex--
		l.MarkDirty()
	}
}

// GetSelected returns the currently selected item
func (l *List) GetSelected() *ListItem {
	if l.SelectedIndex >= 0 && l.SelectedIndex < len(l.Items) {
		return &l.Items[l.SelectedIndex]
	}
	return nil
}

// SetStyles sets the styles for the list
func (l *List) SetStyles(normal, selected Style) {
	l.NormalStyle = normal
	l.SelectedStyle = selected
	l.MarkDirty()
}

// HandleMouse handles mouse events
func (l *List) HandleMouse(event MouseEvent) bool {
	bounds := l.GetBounds()
	if event.X < bounds.Min.X || event.X >= bounds.Max.X ||
		event.Y < bounds.Min.Y || event.Y >= bounds.Max.Y {
		return false
	}

	if event.Type == MousePress || event.Type == MouseClick {
		// Calculate index
		relY := event.Y - bounds.Min.Y
		index := l.ScrollOffset + relY

		if index >= 0 && index < len(l.Items) {
			l.SelectedIndex = index
			l.MarkDirty()
			// Only trigger select on click (release) or double click?
			// For now trigger on Click to be safe.
			if event.Type == MouseClick {
				if l.OnSelect != nil {
					l.OnSelect(l.Items[index])
				}
			}
			return true
		}
	}

	// Wheel support
	if event.Type == MouseScroll {
		if event.Button == MouseButtonWheelDown {
			l.ScrollOffset++
		} else if event.Button == MouseButtonWheelUp {
			l.ScrollOffset--
		}
		l.MarkDirty()
		return true
	}

	return false
}
