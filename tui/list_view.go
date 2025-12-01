package tui

import (
	"fmt"
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

func (l *selectListView) render(frame RenderFrame, bounds image.Rectangle) {
	if bounds.Empty() || len(l.items) == 0 {
		return
	}

	subFrame := frame.SubFrame(bounds)
	height := bounds.Dy()

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
				subFrame.PrintStyled(0, y, l.cursorChar, style)
			}
			x = cursorW
		}

		// Draw item label
		subFrame.PrintTruncated(x, y, item.Label, style)

		// Register clickable region
		if l.onSelect != nil {
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

// checkboxListView displays a list with checkable items
type checkboxListView struct {
	items         []ListItem
	checked       []bool
	cursor        *int
	onToggle      func(index int, checked bool)
	style         Style
	cursorStyle   Style
	checkedChar   string
	uncheckedChar string
	width         int
	height        int
}

// CheckboxList creates a list with checkable items.
// checked should be a slice tracking which items are checked.
// cursor should be a pointer to the current cursor position.
//
// Example:
//
//	CheckboxList(items, app.checked, &app.cursor).OnToggle(func(i int, c bool) { ... })
func CheckboxList(items []ListItem, checked []bool, cursor *int) *checkboxListView {
	return &checkboxListView{
		items:         items,
		checked:       checked,
		cursor:        cursor,
		style:         NewStyle(),
		cursorStyle:   NewStyle().WithBold(),
		checkedChar:   "☑",
		uncheckedChar: "☐",
	}
}

// CheckboxListStrings creates a checkbox list from string labels.
func CheckboxListStrings(labels []string, checked []bool, cursor *int) *checkboxListView {
	items := make([]ListItem, len(labels))
	for i, label := range labels {
		items[i] = ListItem{Label: label, Value: label}
	}
	return CheckboxList(items, checked, cursor)
}

// OnToggle sets a callback when an item is toggled.
func (c *checkboxListView) OnToggle(fn func(index int, checked bool)) *checkboxListView {
	c.onToggle = fn
	return c
}

// Fg sets the foreground color.
func (c *checkboxListView) Fg(col Color) *checkboxListView {
	c.style = c.style.WithForeground(col)
	return c
}

// CursorFg sets the foreground color for the cursor line.
func (c *checkboxListView) CursorFg(col Color) *checkboxListView {
	c.cursorStyle = c.cursorStyle.WithForeground(col)
	return c
}

// Style sets the style for normal items.
func (c *checkboxListView) Style(s Style) *checkboxListView {
	c.style = s
	return c
}

// CursorStyle sets the style for the cursor line.
func (c *checkboxListView) CursorStyle(s Style) *checkboxListView {
	c.cursorStyle = s
	return c
}

// CheckedChar sets the checked checkbox character.
func (c *checkboxListView) CheckedChar(ch string) *checkboxListView {
	c.checkedChar = ch
	return c
}

// UncheckedChar sets the unchecked checkbox character.
func (c *checkboxListView) UncheckedChar(ch string) *checkboxListView {
	c.uncheckedChar = ch
	return c
}

// Width sets a fixed width.
func (c *checkboxListView) Width(w int) *checkboxListView {
	c.width = w
	return c
}

// Height sets a fixed height.
func (c *checkboxListView) Height(h int) *checkboxListView {
	c.height = h
	return c
}

func (c *checkboxListView) size(maxWidth, maxHeight int) (int, int) {
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

func (c *checkboxListView) render(frame RenderFrame, bounds image.Rectangle) {
	if bounds.Empty() || len(c.items) == 0 {
		return
	}

	subFrame := frame.SubFrame(bounds)
	height := bounds.Dy()

	cursorIdx := 0
	if c.cursor != nil {
		cursorIdx = *c.cursor
	}

	checkW, _ := MeasureText(c.checkedChar)

	for y := 0; y < height && y < len(c.items); y++ {
		item := c.items[y]
		isCursor := y == cursorIdx
		isChecked := y < len(c.checked) && c.checked[y]

		style := c.style
		if isCursor {
			style = c.cursorStyle
		}

		// Draw checkbox
		checkChar := c.uncheckedChar
		if isChecked {
			checkChar = c.checkedChar
		}
		subFrame.PrintStyled(0, y, checkChar, style)

		// Draw label
		subFrame.PrintTruncated(checkW+1, y, item.Label, style)

		// Register clickable region
		if c.onToggle != nil {
			itemBounds := image.Rect(
				bounds.Min.X,
				bounds.Min.Y+y,
				bounds.Max.X,
				bounds.Min.Y+y+1,
			)
			idx := y // capture for closure
			interactiveRegistry.RegisterButton(itemBounds, func() {
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

// radioListView displays a list with radio button items
type radioListView struct {
	items          []ListItem
	selected       *int
	onSelect       func(index int)
	style          Style
	cursorStyle    Style
	selectedChar   string
	unselectedChar string
	width          int
	height         int
}

// RadioList creates a list with radio button items (single selection).
// selected should be a pointer to the currently selected index.
//
// Example:
//
//	RadioList(items, &app.selected).OnSelect(func(i int) { ... })
func RadioList(items []ListItem, selected *int) *radioListView {
	return &radioListView{
		items:          items,
		selected:       selected,
		style:          NewStyle(),
		cursorStyle:    NewStyle().WithBold(),
		selectedChar:   "●",
		unselectedChar: "○",
	}
}

// RadioListStrings creates a radio list from string labels.
func RadioListStrings(labels []string, selected *int) *radioListView {
	items := make([]ListItem, len(labels))
	for i, label := range labels {
		items[i] = ListItem{Label: label, Value: label}
	}
	return RadioList(items, selected)
}

// OnSelect sets a callback when an item is selected.
func (r *radioListView) OnSelect(fn func(index int)) *radioListView {
	r.onSelect = fn
	return r
}

// Fg sets the foreground color.
func (r *radioListView) Fg(c Color) *radioListView {
	r.style = r.style.WithForeground(c)
	return r
}

// CursorFg sets the foreground color for the focused item.
func (r *radioListView) CursorFg(c Color) *radioListView {
	r.cursorStyle = r.cursorStyle.WithForeground(c)
	return r
}

// Style sets the style for normal items.
func (r *radioListView) Style(s Style) *radioListView {
	r.style = s
	return r
}

// CursorStyle sets the style for the focused item.
func (r *radioListView) CursorStyle(s Style) *radioListView {
	r.cursorStyle = s
	return r
}

// SelectedChar sets the selected radio character.
func (r *radioListView) SelectedChar(ch string) *radioListView {
	r.selectedChar = ch
	return r
}

// UnselectedChar sets the unselected radio character.
func (r *radioListView) UnselectedChar(ch string) *radioListView {
	r.unselectedChar = ch
	return r
}

// Width sets a fixed width.
func (r *radioListView) Width(w int) *radioListView {
	r.width = w
	return r
}

// Height sets a fixed height.
func (r *radioListView) Height(h int) *radioListView {
	r.height = h
	return r
}

func (r *radioListView) size(maxWidth, maxHeight int) (int, int) {
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

func (r *radioListView) render(frame RenderFrame, bounds image.Rectangle) {
	if bounds.Empty() || len(r.items) == 0 {
		return
	}

	subFrame := frame.SubFrame(bounds)
	height := bounds.Dy()

	selectedIdx := 0
	if r.selected != nil {
		selectedIdx = *r.selected
	}

	radioW, _ := MeasureText(r.selectedChar)

	for y := 0; y < height && y < len(r.items); y++ {
		item := r.items[y]
		isSelected := y == selectedIdx
		style := r.style
		if isSelected {
			style = r.cursorStyle
		}

		// Draw radio button
		radioChar := r.unselectedChar
		if isSelected {
			radioChar = r.selectedChar
		}
		subFrame.PrintStyled(0, y, radioChar, style)

		// Draw label
		subFrame.PrintTruncated(radioW+1, y, item.Label, style)

		// Register clickable region
		if r.onSelect != nil || r.selected != nil {
			itemBounds := image.Rect(
				bounds.Min.X,
				bounds.Min.Y+y,
				bounds.Max.X,
				bounds.Min.Y+y+1,
			)
			idx := y // capture for closure
			interactiveRegistry.RegisterButton(itemBounds, func() {
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

// meterView displays a labeled meter/gauge
type meterView struct {
	label      string
	value      int
	max        int
	width      int
	filledChar rune
	emptyChar  rune
	style      Style
	labelStyle Style
	showValue  bool
}

// Meter creates a labeled meter/gauge view.
//
// Example:
//
//	Meter("CPU", 75, 100).Width(20)
func Meter(label string, value, max int) *meterView {
	return &meterView{
		label:      label,
		value:      value,
		max:        max,
		width:      10,
		filledChar: '█',
		emptyChar:  '·',
		style:      NewStyle().WithForeground(ColorGreen),
		labelStyle: NewStyle(),
		showValue:  true,
	}
}

// Width sets the width of the bar portion.
func (m *meterView) Width(w int) *meterView {
	m.width = w
	return m
}

// FilledChar sets the filled character.
func (m *meterView) FilledChar(c rune) *meterView {
	m.filledChar = c
	return m
}

// EmptyChar sets the empty character.
func (m *meterView) EmptyChar(c rune) *meterView {
	m.emptyChar = c
	return m
}

// Fg sets the bar foreground color.
func (m *meterView) Fg(c Color) *meterView {
	m.style = m.style.WithForeground(c)
	return m
}

// LabelFg sets the label foreground color.
func (m *meterView) LabelFg(c Color) *meterView {
	m.labelStyle = m.labelStyle.WithForeground(c)
	return m
}

// Style sets the bar style.
func (m *meterView) Style(s Style) *meterView {
	m.style = s
	return m
}

// ShowValue enables/disables value display.
func (m *meterView) ShowValue(show bool) *meterView {
	m.showValue = show
	return m
}

func (m *meterView) size(maxWidth, maxHeight int) (int, int) {
	labelW, _ := MeasureText(m.label)
	w := labelW + 2 + m.width // label + ": " + bar
	if m.showValue {
		w += len(fmt.Sprintf(" %d%%", 100))
	}
	if maxWidth > 0 && w > maxWidth {
		w = maxWidth
	}
	return w, 1
}

func (m *meterView) render(frame RenderFrame, bounds image.Rectangle) {
	if bounds.Empty() {
		return
	}

	subFrame := frame.SubFrame(bounds)
	x := 0

	// Draw label
	if m.label != "" {
		subFrame.PrintStyled(x, 0, m.label+": ", m.labelStyle)
		labelW, _ := MeasureText(m.label)
		x += labelW + 2
	}

	// Calculate fill
	barWidth := m.width
	fillWidth := 0
	if m.max > 0 {
		fillWidth = (m.value * barWidth) / m.max
		if fillWidth > barWidth {
			fillWidth = barWidth
		}
	}

	// Draw empty background
	emptyStyle := NewStyle().WithForeground(ColorBrightBlack)
	for i := 0; i < barWidth; i++ {
		subFrame.SetCell(x+i, 0, m.emptyChar, emptyStyle)
	}

	// Draw filled portion
	for i := 0; i < fillWidth; i++ {
		subFrame.SetCell(x+i, 0, m.filledChar, m.style)
	}
	x += barWidth

	// Draw value
	if m.showValue && m.max > 0 {
		percent := (m.value * 100) / m.max
		subFrame.PrintStyled(x, 0, fmt.Sprintf(" %d%%", percent), m.style)
	}
}
