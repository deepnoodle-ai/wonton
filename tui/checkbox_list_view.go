package tui

import "image"

// CheckboxListView displays a list with checkable items
type CheckboxListView struct {
	id             string
	items          []ListItem
	checked        []bool
	cursor         *int
	onToggle       func(index int, checked bool)
	style          Style
	cursorStyle    Style
	checkedStyle   Style
	highlightStyle *Style
	checkedChar    string
	uncheckedChar  string
	width          int
	height         int
	scroll         int
	lastHeight     int
	bounds         image.Rectangle
	focused        bool
}

// CheckboxList creates a list with checkable items.
// checked should be a slice tracking which items are checked.
// cursor should be a pointer to the current cursor position.
//
// The component handles keyboard navigation (arrows, PageUp/PageDown, Home/End, j/k) and toggling (space)
// automatically when focused. Use Tab to focus the list.
//
// For focus management to work, you must set an ID using the ID() method.
// Without an ID, the list will not be focusable via Tab key navigation.
//
// Example:
//
//	CheckboxList(items, app.checked, &app.cursor).
//	    ID("my-checkbox-list").
//	    OnToggle(func(i int, c bool) { ... })
func CheckboxList(items []ListItem, checked []bool, cursor *int) *CheckboxListView {
	return &CheckboxListView{
		items:         items,
		checked:       checked,
		cursor:        cursor,
		style:         NewStyle(),
		cursorStyle:   NewStyle().WithBold(),
		checkedStyle:  NewStyle().WithForeground(ColorGreen),
		checkedChar:   "[x]",
		uncheckedChar: "[ ]",
	}
}

// CheckboxListStrings creates a checkbox list from string labels.
func CheckboxListStrings(labels []string, checked []bool, cursor *int) *CheckboxListView {
	items := make([]ListItem, len(labels))
	for i, label := range labels {
		items[i] = ListItem{Label: label, Value: label}
	}
	return CheckboxList(items, checked, cursor)
}

// OnToggle sets a callback when an item is toggled.
func (c *CheckboxListView) OnToggle(fn func(index int, checked bool)) *CheckboxListView {
	c.onToggle = fn
	return c
}

// ID sets a custom ID for this checkbox list (for focus management).
func (c *CheckboxListView) ID(id string) *CheckboxListView {
	c.id = id
	return c
}

// Focusable interface implementation
func (c *CheckboxListView) FocusID() string {
	return c.id
}

func (c *CheckboxListView) IsFocused() bool {
	return c.focused
}

func (c *CheckboxListView) SetFocused(focused bool) {
	c.focused = focused
}

func (c *CheckboxListView) FocusBounds() image.Rectangle {
	return c.bounds
}

func (c *CheckboxListView) HandleKeyEvent(event KeyEvent) bool {
	if nav := listNavForKey(event, true); nav != listNavNone && c.cursor != nil {
		visible := listViewport(c.lastHeight, c.height, 10)
		next, moved := moveListCursor(nav, *c.cursor, len(c.items), visible)
		if moved {
			*c.cursor = next
			scrollIntoView(&c.scroll, next, visible)
		}
		return moved
	}

	// Handle space to toggle
	if event.Rune == ' ' {
		if c.cursor != nil && *c.cursor >= 0 && *c.cursor < len(c.checked) {
			c.checked[*c.cursor] = !c.checked[*c.cursor]
			if c.onToggle != nil {
				c.onToggle(*c.cursor, c.checked[*c.cursor])
			}
			return true
		}
	}

	return false
}

// Fg sets the foreground color for normal items.
func (c *CheckboxListView) Fg(col Color) *CheckboxListView {
	c.style = c.style.WithForeground(col)
	return c
}

// Bg sets the background color for normal items.
func (c *CheckboxListView) Bg(col Color) *CheckboxListView {
	c.style = c.style.WithBackground(col)
	return c
}

// CursorFg sets the foreground color for the cursor line.
func (c *CheckboxListView) CursorFg(col Color) *CheckboxListView {
	c.cursorStyle = c.cursorStyle.WithForeground(col)
	return c
}

// CursorBg sets the background color for the cursor line.
func (c *CheckboxListView) CursorBg(col Color) *CheckboxListView {
	c.cursorStyle = c.cursorStyle.WithBackground(col)
	return c
}

// CheckedFg sets the foreground color for checked items.
func (c *CheckboxListView) CheckedFg(col Color) *CheckboxListView {
	c.checkedStyle = c.checkedStyle.WithForeground(col)
	return c
}

// CheckedBg sets the background color for checked items.
func (c *CheckboxListView) CheckedBg(col Color) *CheckboxListView {
	c.checkedStyle = c.checkedStyle.WithBackground(col)
	return c
}

// HighlightFg sets the foreground color for highlighted items (when hovered).
func (c *CheckboxListView) HighlightFg(col Color) *CheckboxListView {
	if c.highlightStyle == nil {
		s := NewStyle()
		c.highlightStyle = &s
	}
	*c.highlightStyle = c.highlightStyle.WithForeground(col)
	return c
}

// HighlightBg sets the background color for highlighted items (when hovered).
func (c *CheckboxListView) HighlightBg(col Color) *CheckboxListView {
	if c.highlightStyle == nil {
		s := NewStyle()
		c.highlightStyle = &s
	}
	*c.highlightStyle = c.highlightStyle.WithBackground(col)
	return c
}

// Style sets the style for normal items.
func (c *CheckboxListView) Style(s Style) *CheckboxListView {
	c.style = s
	return c
}

// CursorStyle sets the style for the cursor line.
func (c *CheckboxListView) CursorStyle(s Style) *CheckboxListView {
	c.cursorStyle = s
	return c
}

// CheckedStyle sets the style for checked items.
func (c *CheckboxListView) CheckedStyle(s Style) *CheckboxListView {
	c.checkedStyle = s
	return c
}

// HighlightStyle sets the style for highlighted items (when hovered).
func (c *CheckboxListView) HighlightStyle(s Style) *CheckboxListView {
	c.highlightStyle = &s
	return c
}

// CheckedChar sets the checked checkbox character.
func (c *CheckboxListView) CheckedChar(ch string) *CheckboxListView {
	c.checkedChar = ch
	return c
}

// UncheckedChar sets the unchecked checkbox character.
func (c *CheckboxListView) UncheckedChar(ch string) *CheckboxListView {
	c.uncheckedChar = ch
	return c
}

// Width sets a fixed width.
func (c *CheckboxListView) Width(w int) *CheckboxListView {
	c.width = w
	return c
}

// Height sets a fixed height.
func (c *CheckboxListView) Height(h int) *CheckboxListView {
	c.height = h
	return c
}

// Size sets both width and height at once.
func (c *CheckboxListView) Size(w, h int) *CheckboxListView {
	c.width = w
	c.height = h
	return c
}

func (c *CheckboxListView) size(maxWidth, maxHeight int) (int, int) {
	w := c.width
	if w == 0 {
		checkW, _ := MeasureText(c.checkedChar)
		for _, item := range c.items {
			itemW, _ := MeasureText(item.Label)
			if itemW+checkW+1 > w { // +1 for space
				w = itemW + checkW + 1
			}
		}
	}

	h := c.height
	if h == 0 {
		h = len(c.items)
	}

	if maxWidth > 0 && w > maxWidth {
		w = maxWidth
	}
	if maxHeight > 0 && h > maxHeight {
		h = maxHeight
	}
	return w, h
}

func (c *CheckboxListView) render(ctx *RenderContext) {
	width, height := ctx.Size()
	if width == 0 || height == 0 || len(c.items) == 0 {
		return
	}

	// Register with focus manager for keyboard input (if available)
	c.bounds = ctx.AbsoluteBounds()
	if fm := ctx.FocusManager(); fm != nil {
		fm.Register(c)
	}

	c.lastHeight = height

	cursorIdx := 0
	if c.cursor != nil {
		cursorIdx = *c.cursor
		if cursorIdx > len(c.items)-1 {
			cursorIdx = len(c.items) - 1
		}
		if cursorIdx < 0 {
			cursorIdx = 0
		}
	}

	// Clamp scroll and keep the cursor line visible.
	if maxScroll := len(c.items) - height; c.scroll > maxScroll {
		c.scroll = maxScroll
	}
	if c.scroll < 0 {
		c.scroll = 0
	}
	scrollIntoView(&c.scroll, cursorIdx, height)

	checkW, _ := MeasureText(c.checkedChar)

	for y := 0; y < height && c.scroll+y < len(c.items); y++ {
		idx := c.scroll + y
		item := c.items[idx]
		isCursor := idx == cursorIdx
		isChecked := idx < len(c.checked) && c.checked[idx]

		// Determine style based on state priority:
		// 1. Cursor (focused) has highest priority
		// 2. Checked items have second priority
		// 3. Highlighted items (if implemented via mouse hover in future)
		// 4. Default style
		style := c.style
		if isChecked && !isCursor {
			// Checked items get checked style, unless they're also the cursor
			style = c.checkedStyle
		}
		if isCursor {
			// Cursor always takes precedence
			style = c.cursorStyle
		}

		// Draw checkbox
		checkChar := c.uncheckedChar
		if isChecked {
			checkChar = c.checkedChar
		}
		ctx.PrintStyled(0, y, checkChar, style)

		// Draw label
		ctx.PrintTruncated(checkW+1, y, item.Label, style)

		// Register clickable region
		if c.onToggle != nil {
			bounds := ctx.AbsoluteBounds()
			itemBounds := image.Rect(
				bounds.Min.X,
				bounds.Min.Y+y,
				bounds.Max.X,
				bounds.Min.Y+y+1,
			)
			ctx.registries().interactive.RegisterButton(itemBounds, func() {
				if c.cursor != nil {
					*c.cursor = idx
				}
				if idx < len(c.checked) {
					c.checked[idx] = !c.checked[idx]
					if c.onToggle != nil {
						c.onToggle(idx, c.checked[idx])
					}
				}
			})
		}
	}
}
