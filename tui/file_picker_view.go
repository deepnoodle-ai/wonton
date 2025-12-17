package tui

import (
	"image"
	"strings"
)

// filePickerView displays a file picker with filter input and file list.
type filePickerView struct {
	items       []ListItem
	filter      *string
	selected    *int
	currentPath string
	onSelect    func(item ListItem)
	showHidden  bool
	style       Style
	inputStyle  Style
	pathStyle   Style
	height      int
}

// FilePicker creates a file picker view with filter and list.
// items should contain the files/directories to display.
// filter should be a pointer to the filter text.
// selected should be a pointer to the selected index.
//
// Example:
//
//	FilePicker(app.files, &app.filter, &app.selected).
//	    CurrentPath(app.currentDir).
//	    OnSelect(func(item ListItem) { app.handleSelect(item) })
func FilePicker(items []ListItem, filter *string, selected *int) *filePickerView {
	return &filePickerView{
		items:      items,
		filter:     filter,
		selected:   selected,
		style:      NewStyle(),
		inputStyle: NewStyle().WithBackground(ColorBlue).WithForeground(ColorYellow).WithBold(),
		pathStyle:  NewStyle().WithForeground(ColorCyan),
	}
}

// CurrentPath sets the current directory path to display.
func (f *filePickerView) CurrentPath(path string) *filePickerView {
	f.currentPath = path
	return f
}

// OnSelect sets a callback when an item is selected.
func (f *filePickerView) OnSelect(fn func(item ListItem)) *filePickerView {
	f.onSelect = fn
	return f
}

// ShowHidden enables or disables showing hidden files.
func (f *filePickerView) ShowHidden(show bool) *filePickerView {
	f.showHidden = show
	return f
}

// Fg sets the foreground color for list items.
func (f *filePickerView) Fg(c Color) *filePickerView {
	f.style = f.style.WithForeground(c)
	return f
}

// Bg sets the background color for list items.
func (f *filePickerView) Bg(c Color) *filePickerView {
	f.style = f.style.WithBackground(c)
	return f
}

// Style sets the style for list items.
func (f *filePickerView) Style(s Style) *filePickerView {
	f.style = s
	return f
}

// InputStyle sets the style for the filter input.
func (f *filePickerView) InputStyle(s Style) *filePickerView {
	f.inputStyle = s
	return f
}

// PathStyle sets the style for the current path display.
func (f *filePickerView) PathStyle(s Style) *filePickerView {
	f.pathStyle = s
	return f
}

// Height sets a fixed height for the file picker.
func (f *filePickerView) Height(h int) *filePickerView {
	f.height = h
	return f
}

// filteredItems returns items that match the current filter.
func (f *filePickerView) filteredItems() []ListItem {
	filterText := ""
	if f.filter != nil {
		filterText = *f.filter
	}

	if filterText == "" {
		return f.items
	}

	var filtered []ListItem
	for _, item := range f.items {
		if FuzzyMatch(filterText, item.Label) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func (f *filePickerView) size(maxWidth, maxHeight int) (int, int) {
	h := f.height
	if h == 0 {
		// Default: input (1) + divider (1) + list (items or 10)
		h = 2 + len(f.items)
		if h > 20 {
			h = 20
		}
	}

	w := maxWidth
	if w == 0 {
		w = 40
	}

	if maxHeight > 0 && h > maxHeight {
		h = maxHeight
	}
	return w, h
}

func (f *filePickerView) render(ctx *RenderContext) {
	width, height := ctx.Size()
	if width == 0 || height == 0 {
		return
	}

	// Layout: input (1 line) + divider (1 line) + list (remaining)
	inputHeight := 1
	dividerHeight := 1
	listHeight := height - inputHeight - dividerHeight
	if listHeight < 1 {
		listHeight = 1
	}

	// Create input view
	inputView := Input(f.filter).
		Placeholder("Filter...").
		Width(width)

	// Render input
	inputCtx := ctx.SubContext(image.Rect(0, 0, width, inputHeight))
	inputView.render(inputCtx)

	// Render divider with path
	dividerY := inputHeight
	dividerStyle := NewStyle().WithForeground(ColorBrightBlack)
	ctx.PrintStyled(0, dividerY, strings.Repeat("─", width), dividerStyle)

	// Overlay path on divider
	if f.currentPath != "" {
		pathLabel := " " + f.currentPath + " "
		if len(pathLabel) > width-4 {
			pathLabel = " ..." + pathLabel[len(pathLabel)-width+7:] + " "
		}
		ctx.PrintStyled(2, dividerY, pathLabel, f.pathStyle)
	}

	// Get filtered items
	items := f.filteredItems()

	// Adjust selected index if out of bounds
	if f.selected != nil && *f.selected >= len(items) {
		if len(items) > 0 {
			*f.selected = len(items) - 1
		} else {
			*f.selected = 0
		}
	}

	// Create list view
	listView := SelectList(items, f.selected).
		Style(f.style).
		Height(listHeight).
		OnSelect(func(index int) {
			if f.onSelect != nil && index < len(items) {
				f.onSelect(items[index])
			}
		})

	// Render list
	listCtx := ctx.SubContext(image.Rect(0, dividerY+dividerHeight, width, height))
	listView.render(listCtx)
}
