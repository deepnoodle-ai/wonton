package tui

import "image"

// RadioListView displays a list with radio button items
type RadioListView struct {
	id             string
	items          []ListItem
	selected       *int
	onSelect       func(index int)
	style          Style
	cursorStyle    Style
	selectedChar   string
	unselectedChar string
	width          int
	height         int
	scroll         int
	lastHeight     int
	bounds         image.Rectangle
	focused        bool
}

// RadioList creates a list with radio button items (single selection).
// selected should be a pointer to the currently selected index.
//
// The component handles keyboard navigation (arrows, PageUp/PageDown, Home/End, j/k) and selection (Enter/Space)
// automatically when focused. Use Tab to focus the list.
//
// For focus management to work, you must set an ID using the ID() method.
// Without an ID, the list will not be focusable via Tab key navigation.
//
// Example:
//
//	RadioList(items, &app.selected).
//	    ID("my-radio-list").
//	    OnSelect(func(i int) { ... })
func RadioList(items []ListItem, selected *int) *RadioListView {
	return &RadioListView{
		items:          items,
		selected:       selected,
		style:          NewStyle(),
		cursorStyle:    NewStyle().WithBold(),
		selectedChar:   "●",
		unselectedChar: "○",
	}
}

// RadioListStrings creates a radio list from string labels.
func RadioListStrings(labels []string, selected *int) *RadioListView {
	items := make([]ListItem, len(labels))
	for i, label := range labels {
		items[i] = ListItem{Label: label, Value: label}
	}
	return RadioList(items, selected)
}

// OnSelect sets a callback when an item is selected.
func (r *RadioListView) OnSelect(fn func(index int)) *RadioListView {
	r.onSelect = fn
	return r
}

// ID sets a custom ID for this radio list (for focus management).
func (r *RadioListView) ID(id string) *RadioListView {
	r.id = id
	return r
}

// Focusable interface implementation
func (r *RadioListView) FocusID() string {
	return r.id
}

func (r *RadioListView) IsFocused() bool {
	return r.focused
}

func (r *RadioListView) SetFocused(focused bool) {
	r.focused = focused
}

func (r *RadioListView) FocusBounds() image.Rectangle {
	return r.bounds
}

func (r *RadioListView) HandleKeyEvent(event KeyEvent) bool {
	if nav := listNavForKey(event, true); nav != listNavNone && r.selected != nil {
		visible := listViewport(r.lastHeight, r.height, 10)
		next, moved := moveListCursor(nav, *r.selected, len(r.items), visible)
		if moved {
			*r.selected = next
			scrollIntoView(&r.scroll, next, visible)
		}
		return moved
	}

	switch event.Key {
	case KeyEnter:
		// Enter selects the current item
		if r.selected != nil && *r.selected >= 0 && *r.selected < len(r.items) {
			if r.onSelect != nil {
				r.onSelect(*r.selected)
			}
			return true
		}
	}

	// Handle space to select
	if event.Rune == ' ' {
		if r.selected != nil && *r.selected >= 0 && *r.selected < len(r.items) {
			if r.onSelect != nil {
				r.onSelect(*r.selected)
			}
			return true
		}
	}

	return false
}

// Fg sets the foreground color.
func (r *RadioListView) Fg(c Color) *RadioListView {
	r.style = r.style.WithForeground(c)
	return r
}

// CursorFg sets the foreground color for the focused item.
func (r *RadioListView) CursorFg(c Color) *RadioListView {
	r.cursorStyle = r.cursorStyle.WithForeground(c)
	return r
}

// Style sets the style for normal items.
func (r *RadioListView) Style(s Style) *RadioListView {
	r.style = s
	return r
}

// CursorStyle sets the style for the focused item.
func (r *RadioListView) CursorStyle(s Style) *RadioListView {
	r.cursorStyle = s
	return r
}

// SelectedChar sets the selected radio character.
func (r *RadioListView) SelectedChar(ch string) *RadioListView {
	r.selectedChar = ch
	return r
}

// UnselectedChar sets the unselected radio character.
func (r *RadioListView) UnselectedChar(ch string) *RadioListView {
	r.unselectedChar = ch
	return r
}

// Width sets a fixed width.
func (r *RadioListView) Width(w int) *RadioListView {
	r.width = w
	return r
}

// Height sets a fixed height.
func (r *RadioListView) Height(h int) *RadioListView {
	r.height = h
	return r
}

// Size sets both width and height at once.
func (r *RadioListView) Size(w, h int) *RadioListView {
	r.width = w
	r.height = h
	return r
}

func (r *RadioListView) size(maxWidth, maxHeight int) (int, int) {
	w := r.width
	if w == 0 {
		radioW, _ := MeasureText(r.selectedChar)
		for _, item := range r.items {
			itemW, _ := MeasureText(item.Label)
			if itemW+radioW+1 > w { // +1 for space
				w = itemW + radioW + 1
			}
		}
	}

	h := r.height
	if h == 0 {
		h = len(r.items)
	}

	if maxWidth > 0 && w > maxWidth {
		w = maxWidth
	}
	if maxHeight > 0 && h > maxHeight {
		h = maxHeight
	}
	return w, h
}

func (r *RadioListView) render(ctx *RenderContext) {
	width, height := ctx.Size()
	if width == 0 || height == 0 || len(r.items) == 0 {
		return
	}

	// Register with focus manager for keyboard input (if available)
	r.bounds = ctx.AbsoluteBounds()
	if fm := ctx.FocusManager(); fm != nil {
		fm.Register(r)
	}

	r.lastHeight = height

	selectedIdx := 0
	if r.selected != nil {
		selectedIdx = *r.selected
		if selectedIdx > len(r.items)-1 {
			selectedIdx = len(r.items) - 1
		}
		if selectedIdx < 0 {
			selectedIdx = 0
		}
	}

	// Clamp scroll and keep the selected item visible.
	if maxScroll := len(r.items) - height; r.scroll > maxScroll {
		r.scroll = maxScroll
	}
	if r.scroll < 0 {
		r.scroll = 0
	}
	scrollIntoView(&r.scroll, selectedIdx, height)

	radioW, _ := MeasureText(r.selectedChar)

	for y := 0; y < height && r.scroll+y < len(r.items); y++ {
		idx := r.scroll + y
		item := r.items[idx]
		isSelected := idx == selectedIdx
		style := r.style
		if isSelected {
			style = r.cursorStyle
		}

		// Draw radio button
		radioChar := r.unselectedChar
		if isSelected {
			radioChar = r.selectedChar
		}
		ctx.PrintStyled(0, y, radioChar, style)

		// Draw label
		ctx.PrintTruncated(radioW+1, y, item.Label, style)

		// Register clickable region
		if r.onSelect != nil || r.selected != nil {
			bounds := ctx.AbsoluteBounds()
			itemBounds := image.Rect(
				bounds.Min.X,
				bounds.Min.Y+y,
				bounds.Max.X,
				bounds.Min.Y+y+1,
			)
			ctx.registries().interactive.RegisterButton(itemBounds, func() {
				if r.selected != nil {
					*r.selected = idx
				}
				if r.onSelect != nil {
					r.onSelect(idx)
				}
			})
		}
	}
}
