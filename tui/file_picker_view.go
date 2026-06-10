package tui

import (
	"image"
	"strings"
)

// FilePickerView displays a file picker with filter input and file list.
type FilePickerView struct {
	items       []ListItem
	filter      *string
	selected    *int
	currentPath string
	onSelect    func(item ListItem, index int)
	showHidden  bool
	style       Style
	inputStyle  Style
	pathStyle   Style
	width       int
	height      int
}

// FilePicker creates a file picker view with filter and list.
// items should contain the files/directories to display.
// filter should be a pointer to the filter text.
// selected should be a pointer to the selected index.
//
// The filter input and list are keyboard navigable. Use Tab to switch
// between them, type to filter, and arrow keys to navigate the list.
//
// Example:
//
//	FilePicker(app.files, &app.filter, &app.selected).
//	    CurrentPath(app.currentDir).
//	    OnSelect(func(item ListItem, index int) { app.handleSelect(item, index) })
func FilePicker(items []ListItem, filter *string, selected *int) *FilePickerView {
	return &FilePickerView{
		items:      items,
		filter:     filter,
		selected:   selected,
		style:      NewStyle(),
		inputStyle: NewStyle().WithBackground(ColorBlue).WithForeground(ColorYellow).WithBold(),
		pathStyle:  NewStyle().WithForeground(ColorCyan),
	}
}

// CurrentPath sets the current directory path to display.
func (f *FilePickerView) CurrentPath(path string) *FilePickerView {
	f.currentPath = path
	return f
}

// OnSelect sets a callback when an item is selected.
// The callback receives the selected item and its index.
func (f *FilePickerView) OnSelect(fn func(item ListItem, index int)) *FilePickerView {
	f.onSelect = fn
	return f
}

// ShowHidden enables or disables showing hidden files.
func (f *FilePickerView) ShowHidden(show bool) *FilePickerView {
	f.showHidden = show
	return f
}

// Fg sets the foreground color for list items.
func (f *FilePickerView) Fg(c Color) *FilePickerView {
	f.style = f.style.WithForeground(c)
	return f
}

// Bg sets the background color for list items.
func (f *FilePickerView) Bg(c Color) *FilePickerView {
	f.style = f.style.WithBackground(c)
	return f
}

// Style sets the style for list items.
func (f *FilePickerView) Style(s Style) *FilePickerView {
	f.style = s
	return f
}

// InputStyle sets the style for the filter input.
func (f *FilePickerView) InputStyle(s Style) *FilePickerView {
	f.inputStyle = s
	return f
}

// PathStyle sets the style for the current path display.
func (f *FilePickerView) PathStyle(s Style) *FilePickerView {
	f.pathStyle = s
	return f
}

// Width sets a fixed width for the file picker.
func (f *FilePickerView) Width(w int) *FilePickerView {
	f.width = w
	return f
}

// Height sets a fixed height for the file picker.
func (f *FilePickerView) Height(h int) *FilePickerView {
	f.height = h
	return f
}

// Size sets both width and height at once.
func (f *FilePickerView) Size(w, h int) *FilePickerView {
	f.width = w
	f.height = h
	return f
}

// filteredItems returns items that match the current filter.
func (f *FilePickerView) filteredItems() []ListItem {
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

func (f *FilePickerView) size(maxWidth, maxHeight int) (int, int) {
	h := f.height
	if h == 0 {
		// Default: input (1) + divider (1) + list (items or 10)
		h = 2 + len(f.items)
		if h > 20 {
			h = 20
		}
	}

	w := f.width
	if w == 0 {
		w = maxWidth
		if w == 0 {
			w = 40
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

func (f *FilePickerView) render(ctx *RenderContext) {
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
	InputView := Input(f.filter).
		Placeholder("Filter...").
		Width(width)

	// Render input
	inputCtx := ctx.SubContext(image.Rect(0, 0, width, inputHeight))
	InputView.render(inputCtx)

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
	FilterableListView := SelectList(items, f.selected).
		Style(f.style).
		Height(listHeight).
		OnSelect(func(item ListItem, index int) {
			if f.onSelect != nil {
				f.onSelect(item, index)
			}
		})

	// Render list
	listCtx := ctx.SubContext(image.Rect(0, dividerY+dividerHeight, width, height))
	FilterableListView.render(listCtx)
}
