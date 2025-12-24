package tui

import (
	"fmt"
	"image"
	"strings"
)

// ListItemRenderer is a function that renders a list item.
// It receives the item, whether it's selected, and returns a View.
type ListItemRenderer func(item ListItem, selected bool) View

// listView is a flexible list component that supports keyboard navigation,
// scrolling, filtering, and custom rendering of items.
type listView struct {
	// Data
	items        []ListItem
	filteredIdxs []int // indices into items array after filtering
	selected     *int  // pointer to selected index (in filtered list)

	// Filtering
	filterText        *string // pointer to filter text binding
	filterFunc        func(item ListItem, query string) bool
	showFilter        bool
	filterPlaceholder string

	// Rendering
	renderer      ListItemRenderer
	itemHeight    int // height per item (default 1)
	style         Style
	selectedStyle Style
	filterStyle   Style
	width         int
	height        int

	// Scrolling
	scrollOffset *int // pointer to scroll offset

	// Callbacks
	onSelect func(item ListItem, index int)
}

// FilterableList creates a new filterable list view with the given items.
// selected is a pointer to the currently selected index in the filtered list.
//
// Example:
//
//	FilterableList(items, &app.selected).
//	    Filter(&app.filterText).
//	    Height(10).
//	    OnSelect(func(item tui.ListItem, idx int) { ... })
func FilterableList(items []ListItem, selected *int) *listView {
	filteredIdxs := make([]int, len(items))
	for i := range items {
		filteredIdxs[i] = i
	}

	return &listView{
		items:             items,
		filteredIdxs:      filteredIdxs,
		selected:          selected,
		itemHeight:        1,
		style:             NewStyle(),
		selectedStyle:     NewStyle().WithReverse(),
		filterStyle:       NewStyle().WithForeground(ColorBrightBlack),
		filterPlaceholder: "Filter...",
	}
}

// FilterableListStrings creates a filterable list from string labels.
func FilterableListStrings(labels []string, selected *int) *listView {
	items := make([]ListItem, len(labels))
	for i, label := range labels {
		items[i] = ListItem{Label: label, Value: label}
	}
	return FilterableList(items, selected)
}

// OnSelect sets a callback when an item is selected (Enter key or click).
func (l *listView) OnSelect(fn func(item ListItem, index int)) *listView {
	l.onSelect = fn
	return l
}

// Style sets the style for normal items.
func (l *listView) Style(s Style) *listView {
	l.style = s
	return l
}

// SelectedStyle sets the style for the selected item.
func (l *listView) SelectedStyle(s Style) *listView {
	l.selectedStyle = s
	return l
}

// Fg sets the foreground color for normal items.
func (l *listView) Fg(c Color) *listView {
	l.style = l.style.WithForeground(c)
	return l
}

// SelectedFg sets the foreground color for the selected item.
func (l *listView) SelectedFg(c Color) *listView {
	l.selectedStyle = l.selectedStyle.WithForeground(c)
	return l
}

// SelectedBg sets the background color for the selected item.
func (l *listView) SelectedBg(c Color) *listView {
	l.selectedStyle = l.selectedStyle.WithBackground(c)
	return l
}

// Width sets a fixed width for the list.
func (l *listView) Width(w int) *listView {
	l.width = w
	return l
}

// Height sets a fixed height for the list (including filter if shown).
func (l *listView) Height(h int) *listView {
	l.height = h
	return l
}

// Size sets both width and height at once.
func (l *listView) Size(w, h int) *listView {
	l.width = w
	l.height = h
	return l
}

// ItemHeight sets the height of each item in rows.
func (l *listView) ItemHeight(h int) *listView {
	l.itemHeight = h
	return l
}

// Renderer sets a custom renderer for list items.
// The renderer function receives the item and whether it's selected.
func (l *listView) Renderer(fn ListItemRenderer) *listView {
	l.renderer = fn
	return l
}

// Filter enables filtering with the given text binding.
// The filter input is shown at the top of the list.
func (l *listView) Filter(filterText *string) *listView {
	l.showFilter = true
	l.filterText = filterText
	return l
}

// FilterPlaceholder sets the placeholder text for the filter input.
func (l *listView) FilterPlaceholder(text string) *listView {
	l.filterPlaceholder = text
	return l
}

// FilterFunc sets a custom filter function.
// Default is case-insensitive substring match on Label.
func (l *listView) FilterFunc(fn func(item ListItem, query string) bool) *listView {
	l.filterFunc = fn
	return l
}

// ScrollY binds an external scroll position for programmatic control.
func (l *listView) ScrollY(scrollY *int) *listView {
	l.scrollOffset = scrollY
	return l
}

// ScrollOffset binds an external scroll offset for programmatic control.
// Deprecated: Use ScrollY instead for consistency with other scrollable components.
func (l *listView) ScrollOffset(offset *int) *listView {
	l.scrollOffset = offset
	return l
}

// applyFilter updates the filteredIdxs based on the current filter text.
func (l *listView) applyFilter() {
	if l.filterText == nil || *l.filterText == "" {
		// No filter - show all items
		l.filteredIdxs = make([]int, len(l.items))
		for i := range l.items {
			l.filteredIdxs[i] = i
		}
		return
	}

	query := *l.filterText
	l.filteredIdxs = l.filteredIdxs[:0]

	filterFn := l.filterFunc
	if filterFn == nil {
		// Default: case-insensitive substring match on label
		filterFn = func(item ListItem, q string) bool {
			return strings.Contains(strings.ToLower(item.Label), strings.ToLower(q))
		}
	}

	for i, item := range l.items {
		if filterFn(item, query) {
			l.filteredIdxs = append(l.filteredIdxs, i)
		}
	}

	// Clamp selected index to filtered list bounds
	if l.selected != nil && *l.selected >= len(l.filteredIdxs) {
		if len(l.filteredIdxs) > 0 {
			*l.selected = len(l.filteredIdxs) - 1
		} else {
			*l.selected = 0
		}
	}
}

func (l *listView) size(maxWidth, maxHeight int) (int, int) {
	l.applyFilter()

	w := l.width
	if w == 0 {
		// Calculate width from items
		for _, idx := range l.filteredIdxs {
			item := l.items[idx]
			itemW, _ := MeasureText(item.Label)
			if item.Icon != "" {
				iconW, _ := MeasureText(item.Icon)
				itemW += iconW + 1
			}
			if itemW > w {
				w = itemW
			}
		}
		if w == 0 {
			w = 20 // minimum width
		}
	}

	h := l.height
	if h == 0 {
		// Auto-size to content
		numItems := len(l.filteredIdxs)
		h = numItems * l.itemHeight
		if l.showFilter {
			h += 2 // filter input + divider
		}
		if h == 0 {
			h = 1
		}
	}

	if maxWidth > 0 && w > maxWidth {
		w = maxWidth
	}
	if maxHeight > 0 && h > maxHeight {
		h = maxHeight
	}
	return w, h
}

func (l *listView) render(ctx *RenderContext) {
	l.applyFilter()

	width, height := ctx.Size()
	if width == 0 || height == 0 {
		return
	}

	yOffset := 0

	// Render filter input if enabled
	if l.showFilter && l.filterText != nil {
		filterHeight := 2 // input + divider
		if height <= filterHeight {
			// Not enough space, just show filter
			filterCtx := ctx.SubContext(image.Rect(0, 0, width, height))
			l.renderFilter(filterCtx)
			return
		}

		filterCtx := ctx.SubContext(image.Rect(0, 0, width, 1))
		l.renderFilter(filterCtx)

		// Draw divider
		dividerStyle := NewStyle().WithForeground(ColorBrightBlack)
		for x := 0; x < width; x++ {
			ctx.SetCell(x, 1, '─', dividerStyle)
		}

		yOffset = filterHeight
		height -= filterHeight
	}

	// Render list items
	listCtx := ctx.SubContext(image.Rect(0, yOffset, width, yOffset+height))
	l.renderItems(listCtx)
}

func (l *listView) renderFilter(ctx *RenderContext) {
	if l.filterText == nil {
		return
	}

	// Simple filter display: "Filter: [text]"
	prefix := "Filter: "
	prefixW, _ := MeasureText(prefix)

	ctx.PrintStyled(0, 0, prefix, l.filterStyle)

	// Show filter text or placeholder
	displayText := *l.filterText
	if displayText == "" {
		displayText = l.filterPlaceholder
		ctx.PrintTruncated(prefixW, 0, displayText, l.filterStyle.WithDim())
	} else {
		ctx.PrintTruncated(prefixW, 0, displayText, l.style)
	}
}

func (l *listView) renderItems(ctx *RenderContext) {
	width, height := ctx.Size()
	if width == 0 || height == 0 {
		return
	}

	numItems := len(l.filteredIdxs)
	if numItems == 0 {
		// Show "no items" message
		msg := "No items"
		if l.showFilter && l.filterText != nil && *l.filterText != "" {
			msg = fmt.Sprintf("No items matching '%s'", *l.filterText)
		}
		ctx.PrintStyled(0, 0, msg, l.style.WithDim())
		return
	}

	selectedIdx := 0
	if l.selected != nil {
		selectedIdx = *l.selected
		if selectedIdx >= numItems {
			selectedIdx = numItems - 1
		}
		if selectedIdx < 0 {
			selectedIdx = 0
		}
	}

	// Calculate scroll offset
	scrollOffset := 0
	if l.scrollOffset != nil {
		scrollOffset = *l.scrollOffset
	}

	// Auto-scroll to keep selected item visible
	visibleItems := height / l.itemHeight
	if selectedIdx < scrollOffset {
		scrollOffset = selectedIdx
	}
	if selectedIdx >= scrollOffset+visibleItems {
		scrollOffset = selectedIdx - visibleItems + 1
	}

	// Clamp scroll offset
	maxScroll := numItems - visibleItems
	if maxScroll < 0 {
		maxScroll = 0
	}
	if scrollOffset > maxScroll {
		scrollOffset = maxScroll
	}
	if scrollOffset < 0 {
		scrollOffset = 0
	}

	// Update scroll offset binding
	if l.scrollOffset != nil {
		*l.scrollOffset = scrollOffset
	}

	// Render visible items
	y := 0
	for i := scrollOffset; i < numItems && y < height; i++ {
		itemIdx := l.filteredIdxs[i]
		item := l.items[itemIdx]
		isSelected := i == selectedIdx

		itemHeight := l.itemHeight
		if y+itemHeight > height {
			itemHeight = height - y
		}

		itemCtx := ctx.SubContext(image.Rect(0, y, width, y+itemHeight))
		l.renderItem(itemCtx, item, isSelected, i)

		y += l.itemHeight
	}
}

func (l *listView) renderItem(ctx *RenderContext, item ListItem, selected bool, index int) {
	width, height := ctx.Size()
	if width == 0 || height == 0 {
		return
	}

	// Use custom renderer if provided
	if l.renderer != nil {
		itemView := l.renderer(item, selected)
		itemView.render(ctx)

		// Register click handler
		if l.onSelect != nil {
			bounds := ctx.AbsoluteBounds()
			idx := index
			interactiveRegistry.RegisterButton(bounds, func() {
				if l.selected != nil {
					*l.selected = idx
				}
				// Get the actual item from the filtered indices
				actualIdx := l.filteredIdxs[idx]
				l.onSelect(l.items[actualIdx], actualIdx)
			})
		}
		return
	}

	// Default rendering
	style := l.style
	if selected {
		style = l.selectedStyle
	}

	// Render item with icon if present
	x := 0
	if item.Icon != "" {
		ctx.PrintStyled(x, 0, item.Icon+" ", style)
		iconW, _ := MeasureText(item.Icon)
		x += iconW + 1
	}

	// Fill background for selected item
	if selected && height > 0 {
		for row := 0; row < height; row++ {
			ctx.FillStyled(0, row, width, 1, ' ', style)
		}
	}

	// Render label
	ctx.PrintTruncated(x, 0, item.Label, style)

	// Register click handler
	if l.onSelect != nil {
		bounds := ctx.AbsoluteBounds()
		idx := index
		interactiveRegistry.RegisterButton(bounds, func() {
			if l.selected != nil {
				*l.selected = idx
			}
			// Get the actual item from the filtered indices
			actualIdx := l.filteredIdxs[idx]
			l.onSelect(l.items[actualIdx], actualIdx)
		})
	}
}

// checkboxListView displays a list with checkable items
type checkboxListView struct {
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
		checkedStyle:  NewStyle().WithForeground(ColorGreen),
		checkedChar:   "[x]",
		uncheckedChar: "[ ]",
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

// Fg sets the foreground color for normal items.
func (c *checkboxListView) Fg(col Color) *checkboxListView {
	c.style = c.style.WithForeground(col)
	return c
}

// Bg sets the background color for normal items.
func (c *checkboxListView) Bg(col Color) *checkboxListView {
	c.style = c.style.WithBackground(col)
	return c
}

// CursorFg sets the foreground color for the cursor line.
func (c *checkboxListView) CursorFg(col Color) *checkboxListView {
	c.cursorStyle = c.cursorStyle.WithForeground(col)
	return c
}

// CursorBg sets the background color for the cursor line.
func (c *checkboxListView) CursorBg(col Color) *checkboxListView {
	c.cursorStyle = c.cursorStyle.WithBackground(col)
	return c
}

// CheckedFg sets the foreground color for checked items.
func (c *checkboxListView) CheckedFg(col Color) *checkboxListView {
	c.checkedStyle = c.checkedStyle.WithForeground(col)
	return c
}

// CheckedBg sets the background color for checked items.
func (c *checkboxListView) CheckedBg(col Color) *checkboxListView {
	c.checkedStyle = c.checkedStyle.WithBackground(col)
	return c
}

// HighlightFg sets the foreground color for highlighted items (when hovered).
func (c *checkboxListView) HighlightFg(col Color) *checkboxListView {
	if c.highlightStyle == nil {
		s := NewStyle()
		c.highlightStyle = &s
	}
	*c.highlightStyle = c.highlightStyle.WithForeground(col)
	return c
}

// HighlightBg sets the background color for highlighted items (when hovered).
func (c *checkboxListView) HighlightBg(col Color) *checkboxListView {
	if c.highlightStyle == nil {
		s := NewStyle()
		c.highlightStyle = &s
	}
	*c.highlightStyle = c.highlightStyle.WithBackground(col)
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

// CheckedStyle sets the style for checked items.
func (c *checkboxListView) CheckedStyle(s Style) *checkboxListView {
	c.checkedStyle = s
	return c
}

// HighlightStyle sets the style for highlighted items (when hovered).
func (c *checkboxListView) HighlightStyle(s Style) *checkboxListView {
	c.highlightStyle = &s
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

// Size sets both width and height at once.
func (c *checkboxListView) Size(w, h int) *checkboxListView {
	c.width = w
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

func (c *checkboxListView) render(ctx *RenderContext) {
	width, height := ctx.Size()
	if width == 0 || height == 0 || len(c.items) == 0 {
		return
	}

	cursorIdx := 0
	if c.cursor != nil {
		cursorIdx = *c.cursor
	}

	checkW, _ := MeasureText(c.checkedChar)

	for y := 0; y < height && y < len(c.items); y++ {
		item := c.items[y]
		isCursor := y == cursorIdx
		isChecked := y < len(c.checked) && c.checked[y]

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

// Size sets both width and height at once.
func (r *radioListView) Size(w, h int) *radioListView {
	r.width = w
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

func (r *radioListView) render(ctx *RenderContext) {
	width, height := ctx.Size()
	if width == 0 || height == 0 || len(r.items) == 0 {
		return
	}

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

func (m *meterView) render(ctx *RenderContext) {
	width, height := ctx.Size()
	if width == 0 || height == 0 {
		return
	}

	x := 0

	// Draw label
	if m.label != "" {
		ctx.PrintStyled(x, 0, m.label+": ", m.labelStyle)
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
		ctx.SetCell(x+i, 0, m.emptyChar, emptyStyle)
	}

	// Draw filled portion
	for i := 0; i < fillWidth; i++ {
		ctx.SetCell(x+i, 0, m.filledChar, m.style)
	}
	x += barWidth

	// Draw value
	if m.showValue && m.max > 0 {
		percent := (m.value * 100) / m.max
		ctx.PrintStyled(x, 0, fmt.Sprintf(" %d%%", percent), m.style)
	}
}
