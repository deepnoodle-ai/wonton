package tui

import (
	"image"
)

// selectListView displays a selectable list of items (declarative view)
type selectListView struct {
	items         []ListItem
	selected      *int
	onSelect      func(index int)
	style         Style
	selectedStyle Style
	cursorChar    string
	showCursor    bool
	width         int
	height        int
}

// SelectList creates a selectable list view using the existing ListItem type.
// selected should be a pointer to the currently selected index.
//
// Example:
//
//	SelectList(items, &app.selectedIndex).OnSelect(func(i int) { app.handleSelect(i) })
func SelectList(items []ListItem, selected *int) *selectListView {
	return &selectListView{
		items:         items,
		selected:      selected,
		style:         NewStyle(),
		selectedStyle: NewStyle().WithReverse(),
		cursorChar:    "▸",
		showCursor:    true,
	}
}

// SelectListStrings creates a list from string labels.
func SelectListStrings(labels []string, selected *int) *selectListView {
	items := make([]ListItem, len(labels))
	for i, label := range labels {
		items[i] = ListItem{Label: label, Value: label}
	}
	return SelectList(items, selected)
}

// OnSelect sets a callback when an item is clicked.
func (l *selectListView) OnSelect(fn func(index int)) *selectListView {
	l.onSelect = fn
	return l
}

// Fg sets the foreground color for normal items.
func (l *selectListView) Fg(c Color) *selectListView {
	l.style = l.style.WithForeground(c)
	return l
}

// Bg sets the background color for normal items.
func (l *selectListView) Bg(c Color) *selectListView {
	l.style = l.style.WithBackground(c)
	return l
}

// Style sets the style for normal items.
func (l *selectListView) Style(s Style) *selectListView {
	l.style = s
	return l
}

// SelectedStyle sets the style for the selected item.
func (l *selectListView) SelectedStyle(s Style) *selectListView {
	l.selectedStyle = s
	return l
}

// SelectedFg sets the foreground for selected items.
func (l *selectListView) SelectedFg(c Color) *selectListView {
	l.selectedStyle = l.selectedStyle.WithForeground(c)
	return l
}

// SelectedBg sets the background for selected items.
func (l *selectListView) SelectedBg(c Color) *selectListView {
	l.selectedStyle = l.selectedStyle.WithBackground(c)
	return l
}

// CursorChar sets the cursor indicator character.
func (l *selectListView) CursorChar(c string) *selectListView {
	l.cursorChar = c
	return l
}

// ShowCursor enables/disables the cursor indicator.
func (l *selectListView) ShowCursor(show bool) *selectListView {
	l.showCursor = show
	return l
}

// Width sets a fixed width for the list.
func (l *selectListView) Width(w int) *selectListView {
	l.width = w
	return l
}

// Height sets a fixed height for the list.
func (l *selectListView) Height(h int) *selectListView {
	l.height = h
	return l
}

// Size sets both width and height at once.
func (l *selectListView) Size(w, h int) *selectListView {
	l.width = w
	l.height = h
	return l
}

func (l *selectListView) size(maxWidth, maxHeight int) (int, int) {
	// Calculate width from items
	w := l.width
	if w == 0 {
		cursorW := 0
		if l.showCursor {
			cursorW, _ = MeasureText(l.cursorChar)
			cursorW += 1 // space after cursor
		}
		for _, item := range l.items {
			itemW, _ := MeasureText(item.Label)
			if itemW+cursorW > w {
				w = itemW + cursorW
			}
		}
	}

	h := l.height
	if h == 0 {
		h = len(l.items)
	}

	if maxWidth > 0 && w > maxWidth {
		w = maxWidth
	}
	if maxHeight > 0 && h > maxHeight {
		h = maxHeight
	}
	return w, h
}

func (l *selectListView) render(ctx *RenderContext) {
	width, height := ctx.Size()
	if width == 0 || height == 0 || len(l.items) == 0 {
		return
	}

	selectedIdx := 0
	if l.selected != nil {
		selectedIdx = *l.selected
	}

	cursorW := 0
	if l.showCursor {
		cursorW, _ = MeasureText(l.cursorChar)
		cursorW += 1 // space after cursor
	}

	for y := 0; y < height && y < len(l.items); y++ {
		item := l.items[y]
		isSelected := y == selectedIdx
		style := l.style
		if isSelected {
			style = l.selectedStyle
		}

		x := 0

		// Draw cursor
		if l.showCursor {
			if isSelected {
				ctx.PrintStyled(0, y, l.cursorChar, style)
			}
			x = cursorW
		}

		// Draw item label
		ctx.PrintTruncated(x, y, item.Label, style)

		// Register clickable region
		if l.onSelect != nil {
			bounds := ctx.AbsoluteBounds()
			itemBounds := image.Rect(
				bounds.Min.X,
				bounds.Min.Y+y,
				bounds.Max.X,
				bounds.Min.Y+y+1,
			)
			idx := y // capture for closure
			interactiveRegistry.RegisterButton(itemBounds, func() {
				if l.selected != nil {
					*l.selected = idx
				}
				l.onSelect(idx)
			})
		}
	}
}
