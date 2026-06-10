package tui

import (
	"fmt"
	"image"
	"strings"
)

// ListItemRenderer is a function that renders a list item.
// It receives the item, whether it's selected, and returns a View.
type ListItemRenderer func(item ListItem, selected bool) View

// FilterableListView is a flexible list component that supports keyboard navigation,
// scrolling, filtering, and custom rendering of items.
type FilterableListView struct {
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
// The component handles keyboard navigation (arrows, PageUp/PageDown, Home/End), filtering (typing),
// and selection (Enter) automatically when focused. Use Tab to focus the list.
//
// For focus management to work, you must set an ID using the ID() method.
// Without an ID, the list will not be focusable via Tab key navigation.
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
//	    ID("my-list").
//	    Filter(&app.filterText).
//	    Height(10).
//	    MultiSelect(true).
//	    Chosen(&app.chosen).
//	    Markers("[ ]", "[x]").
//	    OnSelect(func(item tui.ListItem, idx int) { ... })
func FilterableList(items []ListItem, selected *int) *FilterableListView {
	filteredIdxs := make([]int, len(items))
	for i := range items {
		filteredIdxs[i] = i
	}

	return &FilterableListView{
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
func FilterableListStrings(labels []string, selected *int) *FilterableListView {
	items := make([]ListItem, len(labels))
	for i, label := range labels {
		items[i] = ListItem{Label: label, Value: label}
	}
	return FilterableList(items, selected)
}

// OnSelect sets a callback when an item is selected (Enter key or click).
func (l *FilterableListView) OnSelect(fn func(item ListItem, index int)) *FilterableListView {
	l.onSelect = fn
	return l
}

// ID sets a custom ID for this list (for focus management).
func (l *FilterableListView) ID(id string) *FilterableListView {
	l.id = id
	return l
}

// Focusable interface implementation
func (l *FilterableListView) FocusID() string {
	return l.id
}

func (l *FilterableListView) IsFocused() bool {
	return l.focused
}

func (l *FilterableListView) SetFocused(focused bool) {
	l.focused = focused
}

func (l *FilterableListView) FocusBounds() image.Rectangle {
	return l.bounds
}

func (l *FilterableListView) HandleKeyEvent(event KeyEvent) bool {
	// Get visible height for scroll calculations
	visibleHeight := l.height
	if visibleHeight == 0 {
		visibleHeight = 10 // default
	}
	if l.showFilter && l.filterText != nil {
		visibleHeight -= 2 // account for filter input and divider
	}

	// Vi-style navigation keys are only active when typing is not being
	// captured by the filter input.
	allowVi := !(l.showFilter && l.filterText != nil)
	if nav := listNavForKey(event, allowVi); nav != listNavNone && l.selected != nil {
		if next, moved := moveListCursor(nav, *l.selected, len(l.filteredIdxs), visibleHeight); moved {
			*l.selected = next
			scrollIntoView(l.scrollOffset, *l.selected, visibleHeight)
			return true
		}
	}

	switch event.Key {
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
func (l *FilterableListView) Style(s Style) *FilterableListView {
	l.style = s
	return l
}

// SelectedStyle sets the style for the selected item.
func (l *FilterableListView) SelectedStyle(s Style) *FilterableListView {
	l.selectedStyle = s
	return l
}

// Fg sets the foreground color for normal items.
func (l *FilterableListView) Fg(c Color) *FilterableListView {
	l.style = l.style.WithForeground(c)
	return l
}

// SelectedFg sets the foreground color for the selected item.
func (l *FilterableListView) SelectedFg(c Color) *FilterableListView {
	l.selectedStyle = l.selectedStyle.WithForeground(c)
	return l
}

// SelectedBg sets the background color for the selected item.
// This also disables the default reverse video effect to allow explicit color control.
func (l *FilterableListView) SelectedBg(c Color) *FilterableListView {
	l.selectedStyle = l.selectedStyle.WithBackground(c)
	// Disable reverse when setting explicit background - otherwise colors get inverted
	l.selectedStyle.Reverse = false
	return l
}

// ChosenStyle sets the style for chosen items (items confirmed with Enter).
// This style is applied to items that have been selected via the Enter key.
// Note: When an item is both chosen and under the cursor, the selected style
// takes precedence.
func (l *FilterableListView) ChosenStyle(s Style) *FilterableListView {
	l.chosenStyle = s
	return l
}

// ChosenFg sets the foreground color for chosen items (items confirmed with Enter).
func (l *FilterableListView) ChosenFg(c Color) *FilterableListView {
	l.chosenStyle = l.chosenStyle.WithForeground(c)
	return l
}

// ChosenBg sets the background color for chosen items (items confirmed with Enter).
func (l *FilterableListView) ChosenBg(c Color) *FilterableListView {
	l.chosenStyle = l.chosenStyle.WithBackground(c)
	return l
}

// MultiSelect enables multi-selection mode where Enter toggles items.
// When enabled, pressing Enter on an item toggles its chosen state without
// affecting other chosen items. When disabled (default), pressing Enter
// clears any previously chosen items and selects only the current item.
func (l *FilterableListView) MultiSelect(enabled bool) *FilterableListView {
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
func (l *FilterableListView) Markers(defaultMarker, chosenMarker string) *FilterableListView {
	l.defaultMarker = defaultMarker
	l.chosenMarker = chosenMarker
	return l
}

// Chosen binds the chosen items to an external slice of indices.
// The slice is updated whenever items are chosen or unchosen via the Enter key.
// Indices refer to the original item positions (not filtered indices).
// Use with MultiSelect(true) to allow multiple chosen items, or leave as
// single-select mode where choosing a new item clears the previous choice.
func (l *FilterableListView) Chosen(chosen *[]int) *FilterableListView {
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
func (l *FilterableListView) syncChosenToPtr() {
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
func (l *FilterableListView) isChosen(origIdx int) bool {
	return l.chosen[origIdx]
}

// Width sets a fixed width for the list.
func (l *FilterableListView) Width(w int) *FilterableListView {
	l.width = w
	return l
}

// Height sets a fixed height for the list (including filter if shown).
func (l *FilterableListView) Height(h int) *FilterableListView {
	l.height = h
	return l
}

// Size sets both width and height at once.
func (l *FilterableListView) Size(w, h int) *FilterableListView {
	l.width = w
	l.height = h
	return l
}

// ItemHeight sets the height of each item in rows. Values less than 1 are
// ignored, since layout math divides by the item height.
func (l *FilterableListView) ItemHeight(h int) *FilterableListView {
	if h >= 1 {
		l.itemHeight = h
	}
	return l
}

// Renderer sets a custom renderer for list items.
// The renderer function receives the item and whether it's selected.
func (l *FilterableListView) Renderer(fn ListItemRenderer) *FilterableListView {
	l.renderer = fn
	return l
}

// Filter enables filtering with the given text binding.
// The filter input is shown at the top of the list.
func (l *FilterableListView) Filter(filterText *string) *FilterableListView {
	l.showFilter = true
	l.filterText = filterText
	return l
}

// FilterPlaceholder sets the placeholder text for the filter input.
func (l *FilterableListView) FilterPlaceholder(text string) *FilterableListView {
	l.filterPlaceholder = text
	return l
}

// FilterFunc sets a custom filter function.
// Default is case-insensitive substring match on Label.
func (l *FilterableListView) FilterFunc(fn func(item ListItem, query string) bool) *FilterableListView {
	l.filterFunc = fn
	return l
}

// ScrollY binds an external scroll position for programmatic control.
func (l *FilterableListView) ScrollY(scrollY *int) *FilterableListView {
	l.scrollOffset = scrollY
	return l
}

// ScrollOffset binds an external scroll offset for programmatic control.
// Deprecated: Use ScrollY instead for consistency with other scrollable components.
func (l *FilterableListView) ScrollOffset(offset *int) *FilterableListView {
	l.scrollOffset = offset
	return l
}

// applyFilter updates the filteredIdxs based on the current filter text.
func (l *FilterableListView) applyFilter() {
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

func (l *FilterableListView) size(maxWidth, maxHeight int) (int, int) {
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

func (l *FilterableListView) render(ctx *RenderContext) {
	l.applyFilter()

	width, height := ctx.Size()
	if width == 0 || height == 0 {
		return
	}

	// Register with focus manager for keyboard input (if available)
	l.bounds = ctx.AbsoluteBounds()
	if fm := ctx.FocusManager(); fm != nil {
		fm.Register(l)
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

func (l *FilterableListView) renderFilter(ctx *RenderContext) {
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

func (l *FilterableListView) renderItems(ctx *RenderContext) {
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

func (l *FilterableListView) renderItem(ctx *RenderContext, item ListItem, selected bool, chosen bool, index int, origIdx int) {
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
			ctx.registries().interactive.RegisterButton(bounds, func() {
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
		ctx.registries().interactive.RegisterButton(bounds, func() {
			if l.selected != nil {
				*l.selected = idx
			}
			l.onSelect(l.items[oi], oi)
		})
	}
}
