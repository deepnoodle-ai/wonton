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
	// Focus management
	id      string
	bounds  image.Rectangle
	focused bool

	// Data
	items        []ListItem
	filteredIdxs []int // indices into items array after filtering
	selected     *int  // pointer to selected index (in filtered list)

	// Chosen items (items confirmed with Enter)
	chosen        map[int]bool // map of original item indices that are chosen
	chosenPtr     *[]int       // optional external binding for chosen indices
	multiSelect   bool         // true = toggle multiple, false = single selection
	chosenMarker  string       // marker shown for chosen items (e.g., "[x]")
	defaultMarker string       // marker shown for unchosen items (e.g., "[ ]")

	// Filtering
	filterText        *string // pointer to filter text binding
	filterFunc        func(item ListItem, query string) bool
	showFilter        bool
	filterPlaceholder string

	// Rendering
	renderer      ListItemRenderer
	itemHeight    int // height per item (default 1)
	style         Style
	selectedStyle Style // style for active/cursor item
	chosenStyle   Style // style for chosen items
	filterStyle   Style
	width         int
	height        int

	// Scrolling
	scrollOffset *int // pointer to scroll offset

	// Callbacks
	onSelect func(item ListItem, index int)
}

// FilterableList creates a new filterable list view with the given items.
// selected is a pointer to the currently selected index (cursor position) in
// the filtered list.
//
// The component handles keyboard navigation (arrow keys), filtering (typing),
// and selection (Enter) automatically when focused. Use Tab to focus the list.
//
// To track chosen items (items confirmed with Enter), use Chosen() to bind
// an external slice. Use MultiSelect(true) to allow multiple items to be
// chosen, or leave as single-select mode (default) where choosing a new item
// clears the previous choice. Use Markers() to display chosen/unchosen
// indicators on the right side of each item.
//
// Example:
//
//	FilterableList(items, &app.selected).
//	    Filter(&app.filterText).
//	    Height(10).
//	    MultiSelect(true).
//	    Chosen(&app.chosen).
//	    Markers("[ ]", "[x]").
//	    OnSelect(func(item tui.ListItem, idx int) { ... })
func FilterableList(items []ListItem, selected *int) *listView {
	filteredIdxs := make([]int, len(items))
	for i := range items {
		filteredIdxs[i] = i
	}

	// Generate ID from selected pointer address
	id := fmt.Sprintf("list_%p", selected)

	return &listView{
		id:                id,
		items:             items,
		filteredIdxs:      filteredIdxs,
		selected:          selected,
		chosen:            make(map[int]bool),
		itemHeight:        1,
		style:             NewStyle(),
		selectedStyle:     NewStyle().WithReverse(),
		chosenStyle:       NewStyle().WithForeground(ColorGreen),
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

// ID sets a custom ID for this list (for focus management).
func (l *listView) ID(id string) *listView {
	l.id = id
	return l
}

// Focusable interface implementation
func (l *listView) FocusID() string {
	return l.id
}

func (l *listView) IsFocused() bool {
	return l.focused
}

func (l *listView) SetFocused(focused bool) {
	l.focused = focused
}

func (l *listView) FocusBounds() image.Rectangle {
	return l.bounds
}

func (l *listView) HandleKeyEvent(event KeyEvent) bool {
	// Get visible height for scroll calculations
	visibleHeight := l.height
	if visibleHeight == 0 {
		visibleHeight = 10 // default
	}
	if l.showFilter && l.filterText != nil {
		visibleHeight -= 2 // account for filter input and divider
	}

	// Handle arrow keys for navigation
	switch event.Key {
	case KeyArrowUp:
		if l.selected != nil && *l.selected > 0 {
			*l.selected--
			// Adjust scroll if needed
			if l.scrollOffset != nil && *l.scrollOffset > *l.selected {
				*l.scrollOffset = *l.selected
			}
			return true
		}
	case KeyArrowDown:
		if l.selected != nil && *l.selected < len(l.filteredIdxs)-1 {
			*l.selected++
			// Adjust scroll if needed
			if l.scrollOffset != nil && *l.selected-*l.scrollOffset >= visibleHeight {
				*l.scrollOffset = *l.selected - visibleHeight + 1
			}
			return true
		}
	case KeyEnter:
		if l.selected != nil && *l.selected >= 0 && *l.selected < len(l.filteredIdxs) {
			origIdx := l.filteredIdxs[*l.selected]

			// Update chosen items
			if l.multiSelect {
				// Toggle in multi-select mode
				if l.chosen[origIdx] {
					delete(l.chosen, origIdx)
				} else {
					l.chosen[origIdx] = true
				}
			} else {
				// Single select mode - clear others and set this one
				l.chosen = make(map[int]bool)
				l.chosen[origIdx] = true
			}

			// Sync to external binding if provided
			l.syncChosenToPtr()

			// Fire callback
			if l.onSelect != nil {
				l.onSelect(l.items[origIdx], *l.selected)
			}
			return true
		}
	case KeyBackspace:
		// Handle filter text deletion
		if l.showFilter && l.filterText != nil && len(*l.filterText) > 0 {
			*l.filterText = (*l.filterText)[:len(*l.filterText)-1]
			return true
		}
	}

	// Handle printable characters for filtering
	if l.showFilter && l.filterText != nil && event.Rune >= 32 && event.Rune < 127 {
		*l.filterText += string(event.Rune)
		return true
	}

	return false
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
// This also disables the default reverse video effect to allow explicit color control.
func (l *listView) SelectedBg(c Color) *listView {
	l.selectedStyle = l.selectedStyle.WithBackground(c)
	// Disable reverse when setting explicit background - otherwise colors get inverted
	l.selectedStyle.Reverse = false
	return l
}

// ChosenStyle sets the style for chosen items (items confirmed with Enter).
// This style is applied to items that have been selected via the Enter key.
// Note: When an item is both chosen and under the cursor, the selected style
// takes precedence.
func (l *listView) ChosenStyle(s Style) *listView {
	l.chosenStyle = s
	return l
}

// ChosenFg sets the foreground color for chosen items (items confirmed with Enter).
func (l *listView) ChosenFg(c Color) *listView {
	l.chosenStyle = l.chosenStyle.WithForeground(c)
	return l
}

// ChosenBg sets the background color for chosen items (items confirmed with Enter).
func (l *listView) ChosenBg(c Color) *listView {
	l.chosenStyle = l.chosenStyle.WithBackground(c)
	return l
}

// MultiSelect enables multi-selection mode where Enter toggles items.
// When enabled, pressing Enter on an item toggles its chosen state without
// affecting other chosen items. When disabled (default), pressing Enter
// clears any previously chosen items and selects only the current item.
func (l *listView) MultiSelect(enabled bool) *listView {
	l.multiSelect = enabled
	return l
}

// Markers sets the markers displayed on the right side of items to indicate
// chosen state. The defaultMarker is shown for unchosen items, and chosenMarker
// is shown for chosen items. Pass empty strings to disable markers.
//
// Example:
//
//	Markers("[ ]", "[x]")  // checkbox style
//	Markers("○", "●")      // radio style
//	Markers("", "✓")       // checkmark only when chosen
func (l *listView) Markers(defaultMarker, chosenMarker string) *listView {
	l.defaultMarker = defaultMarker
	l.chosenMarker = chosenMarker
	return l
}

// Chosen binds the chosen items to an external slice of indices.
// The slice is updated whenever items are chosen or unchosen via the Enter key.
// Indices refer to the original item positions (not filtered indices).
// Use with MultiSelect(true) to allow multiple chosen items, or leave as
// single-select mode where choosing a new item clears the previous choice.
func (l *listView) Chosen(chosen *[]int) *listView {
	l.chosenPtr = chosen
	// Initialize internal map from provided slice
	if chosen != nil {
		l.chosen = make(map[int]bool)
		for _, idx := range *chosen {
			l.chosen[idx] = true
		}
	}
	return l
}

// syncChosenToPtr updates the external chosen pointer from the internal map.
func (l *listView) syncChosenToPtr() {
	if l.chosenPtr == nil {
		return
	}
	// Rebuild slice from map
	result := make([]int, 0, len(l.chosen))
	for idx := range l.chosen {
		result = append(result, idx)
	}
	*l.chosenPtr = result
}

// isChosen checks if an item (by original index) is chosen.
func (l *listView) isChosen(origIdx int) bool {
	return l.chosen[origIdx]
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
			var fullText string
			if item.Icon != "" {
				fullText = item.Icon + "  " + item.Label // Double space after icon
			} else {
				fullText = item.Label
			}
			itemW, _ := MeasureText(fullText)
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

	// Register with focus manager for keyboard input
	l.bounds = ctx.AbsoluteBounds()
	focusManager.Register(l)

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

	// Helper to calculate visible items and indicator needs
	calcLayout := func(scrollOff int) (visibleItems int, hasAbove, hasBelow bool) {
		available := height
		hasAbove = scrollOff > 0
		if hasAbove {
			available--
		}
		visibleItems = available / l.itemHeight
		if visibleItems < 1 {
			visibleItems = 1
		}
		itemsBelow := numItems - scrollOff - visibleItems
		hasBelow = itemsBelow > 0
		if hasBelow {
			available--
			visibleItems = available / l.itemHeight
			if visibleItems < 1 {
				visibleItems = 1
			}
			// Recheck after reducing visible items
			itemsBelow = numItems - scrollOff - visibleItems
			hasBelow = itemsBelow > 0
		}
		return
	}

	// Initial layout calculation
	visibleItems, hasItemsAbove, hasItemsBelow := calcLayout(scrollOffset)

	// Auto-scroll to keep selected item visible
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

	// Recalculate layout with final scroll offset
	visibleItems, hasItemsAbove, hasItemsBelow = calcLayout(scrollOffset)

	// Update scroll offset binding
	if l.scrollOffset != nil {
		*l.scrollOffset = scrollOffset
	}

	indicatorStyle := NewStyle().WithDim()
	y := 0

	// Render top scroll indicator on its own line
	if hasItemsAbove {
		indicator := fmt.Sprintf("↑ %d more", scrollOffset)
		indicatorW, _ := MeasureText(indicator)
		x := (width - indicatorW) / 2
		if x < 0 {
			x = 0
		}
		ctx.PrintStyled(x, y, indicator, indicatorStyle)
		y++
	}

	// Render visible items
	for i := scrollOffset; i < numItems && i < scrollOffset+visibleItems; i++ {
		itemIdx := l.filteredIdxs[i]
		item := l.items[itemIdx]
		isSelected := i == selectedIdx
		isChosen := l.isChosen(itemIdx)

		itemCtx := ctx.SubContext(image.Rect(0, y, width, y+l.itemHeight))
		l.renderItem(itemCtx, item, isSelected, isChosen, i, itemIdx)

		y += l.itemHeight
	}

	// Render bottom scroll indicator on its own line
	if hasItemsBelow {
		itemsBelow := numItems - scrollOffset - visibleItems
		indicator := fmt.Sprintf("↓ %d more", itemsBelow)
		indicatorW, _ := MeasureText(indicator)
		x := (width - indicatorW) / 2
		if x < 0 {
			x = 0
		}
		ctx.PrintStyled(x, height-1, indicator, indicatorStyle)
	}
}

func (l *listView) renderItem(ctx *RenderContext, item ListItem, selected bool, chosen bool, index int, origIdx int) {
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

	// Default rendering - determine style based on state
	style := l.style
	if chosen {
		style = l.chosenStyle
	}
	if selected {
		// When both selected and chosen, use selected background with chosen foreground
		// so user can still see the item is chosen while highlighted
		if chosen {
			style = l.selectedStyle.WithForeground(l.chosenStyle.Foreground)
		} else {
			style = l.selectedStyle
		}
	}

	// Build the full text to render (without checkbox - that goes on right)
	var fullText string
	if item.Icon != "" {
		fullText = item.Icon + "  " + item.Label // Double space after icon for safety
	} else {
		fullText = item.Label
	}

	// Fill background for selected or chosen items
	if (selected || chosen) && height > 0 {
		for row := 0; row < height; row++ {
			ctx.FillStyled(0, row, width, 1, ' ', style)
		}
	}

	// Determine which marker to show (if any)
	marker := l.defaultMarker
	if chosen {
		marker = l.chosenMarker
	}
	markerWidth := 0
	if marker != "" {
		mw, _ := MeasureText(marker)
		markerWidth = mw + 1 // +1 for space before marker
	}

	// Render the item text (truncated to leave room for marker)
	maxTextWidth := width - markerWidth
	if maxTextWidth > 0 {
		ctx.PrintTruncated(0, 0, fullText, style)
	}

	// Render marker on the right side if present
	if marker != "" {
		markerX := width - markerWidth
		if markerX < 0 {
			markerX = 0
		}
		ctx.PrintStyled(markerX, 0, " "+marker, style)
	}

	// Register click handler
	if l.onSelect != nil {
		bounds := ctx.AbsoluteBounds()
		idx := index
		oi := origIdx
		interactiveRegistry.RegisterButton(bounds, func() {
			if l.selected != nil {
				*l.selected = idx
			}
			l.onSelect(l.items[oi], oi)
		})
	}
}

// checkboxListView displays a list with checkable items
type checkboxListView struct {
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
	bounds         image.Rectangle
	focused        bool
}

// CheckboxList creates a list with checkable items.
// checked should be a slice tracking which items are checked.
// cursor should be a pointer to the current cursor position.
//
// The component handles keyboard navigation (arrow keys) and toggling (space)
// automatically when focused. Use Tab to focus the list.
//
// Example:
//
//	CheckboxList(items, app.checked, &app.cursor).OnToggle(func(i int, c bool) { ... })
func CheckboxList(items []ListItem, checked []bool, cursor *int) *checkboxListView {
	// Generate ID from cursor pointer address
	id := fmt.Sprintf("checkbox_%p", cursor)
	return &checkboxListView{
		id:            id,
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

// ID sets a custom ID for this checkbox list (for focus management).
func (c *checkboxListView) ID(id string) *checkboxListView {
	c.id = id
	return c
}

// Focusable interface implementation
func (c *checkboxListView) FocusID() string {
	return c.id
}

func (c *checkboxListView) IsFocused() bool {
	return c.focused
}

func (c *checkboxListView) SetFocused(focused bool) {
	c.focused = focused
}

func (c *checkboxListView) FocusBounds() image.Rectangle {
	return c.bounds
}

func (c *checkboxListView) HandleKeyEvent(event KeyEvent) bool {
	// Handle arrow keys for navigation
	switch event.Key {
	case KeyArrowUp:
		if c.cursor != nil && *c.cursor > 0 {
			*c.cursor--
			return true
		}
	case KeyArrowDown:
		if c.cursor != nil && *c.cursor < len(c.items)-1 {
			*c.cursor++
			return true
		}
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

	// Register with focus manager for keyboard input
	c.bounds = ctx.AbsoluteBounds()
	focusManager.Register(c)

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
	bounds         image.Rectangle
	focused        bool
}

// RadioList creates a list with radio button items (single selection).
// selected should be a pointer to the currently selected index.
//
// The component handles keyboard navigation (arrow keys) and selection (Enter/Space)
// automatically when focused. Use Tab to focus the list.
//
// Example:
//
//	RadioList(items, &app.selected).OnSelect(func(i int) { ... })
func RadioList(items []ListItem, selected *int) *radioListView {
	// Generate ID from selected pointer address
	id := fmt.Sprintf("radio_%p", selected)
	return &radioListView{
		id:             id,
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

// ID sets a custom ID for this radio list (for focus management).
func (r *radioListView) ID(id string) *radioListView {
	r.id = id
	return r
}

// Focusable interface implementation
func (r *radioListView) FocusID() string {
	return r.id
}

func (r *radioListView) IsFocused() bool {
	return r.focused
}

func (r *radioListView) SetFocused(focused bool) {
	r.focused = focused
}

func (r *radioListView) FocusBounds() image.Rectangle {
	return r.bounds
}

func (r *radioListView) HandleKeyEvent(event KeyEvent) bool {
	// Handle arrow keys for navigation
	switch event.Key {
	case KeyArrowUp:
		if r.selected != nil && *r.selected > 0 {
			*r.selected--
			return true
		}
	case KeyArrowDown:
		if r.selected != nil && *r.selected < len(r.items)-1 {
			*r.selected++
			return true
		}
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

	// Register with focus manager for keyboard input
	r.bounds = ctx.AbsoluteBounds()
	focusManager.Register(r)

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
