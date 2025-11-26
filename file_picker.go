package gooey

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"
)

// FilePicker is a widget for selecting files
type FilePicker struct {
	BaseWidget

	CurrentDir string
	ShowHidden bool
	Filter     string

	// Child widgets
	input *TextInput
	list  *List

	// Internal
	allFiles []ListItem
	OnSelect func(path string)
	focused  bool
}

// NewFilePicker creates a new file picker
func NewFilePicker(startDir string) *FilePicker {
	if startDir == "" {
		startDir, _ = os.Getwd()
	}
	// Ensure absolute path
	absDir, err := filepath.Abs(startDir)
	if err == nil {
		startDir = absDir
	}

	fp := &FilePicker{
		BaseWidget: NewBaseWidget(),
		CurrentDir: startDir,
		ShowHidden: false,
		input:      NewTextInput(),
		list:       NewList(nil),
	}

	fp.input.Placeholder = "Filter..."
	fp.input.SetFocused(true) // Always keep input "focused" for typing
	fp.input.OnChange = fp.handleFilterChange

	fp.list.OnSelect = fp.handleListSelect

	fp.Refresh()
	return fp
}

// Init initializes the widget
func (fp *FilePicker) Init() {
	fp.input.Init()
	fp.list.Init()
}

// Refresh reloads the file list from the current directory
func (fp *FilePicker) Refresh() {
	fp.input.SetValue("") // Reset filter on dir change? Or keep it? usually reset.
	fp.input.CursorPos = 0
	fp.Filter = ""

	files, err := os.ReadDir(fp.CurrentDir)
	if err != nil {
		// Show error item?
		fp.list.SetItems([]ListItem{{Label: fmt.Sprintf("Error: %v", err)}})
		return
	}

	var items []ListItem

	// Add ".." if not at root
	parent := filepath.Dir(fp.CurrentDir)
	if parent != fp.CurrentDir {
		items = append(items, ListItem{Label: "..", Value: parent, Icon: "[DIR]"})
	}

	// Sort: Directories first, then files
	var dirs []os.DirEntry
	var regular []os.DirEntry

	for _, f := range files {
		if !fp.ShowHidden && strings.HasPrefix(f.Name(), ".") {
			continue
		}
		if f.IsDir() {
			dirs = append(dirs, f)
		} else {
			regular = append(regular, f)
		}
	}

	for _, d := range dirs {
		items = append(items, ListItem{
			Label: d.Name(),
			Value: filepath.Join(fp.CurrentDir, d.Name()),
			Icon:  "[DIR]",
		})
	}

	for _, f := range regular {
		items = append(items, ListItem{
			Label: f.Name(),
			Value: filepath.Join(fp.CurrentDir, f.Name()),
			Icon:  "",
		})
	}

	fp.allFiles = items
	fp.filterFiles()
}

func (fp *FilePicker) handleFilterChange(value string) {
	fp.Filter = value
	fp.filterFiles()
}

func (fp *FilePicker) filterFiles() {
	if fp.Filter == "" {
		fp.list.SetItems(fp.allFiles)
		return
	}

	var filtered []ListItem
	for _, item := range fp.allFiles {
		// Simple fuzzy match: all chars must be present in order
		// or use simple string contains
		if FuzzyMatch(fp.Filter, item.Label) {
			filtered = append(filtered, item)
		}
	}
	fp.list.SetItems(filtered)
}

func (fp *FilePicker) handleListSelect(item ListItem) {
	path, ok := item.Value.(string)
	if !ok {
		return
	}

	// Check if it's a directory
	info, err := os.Stat(path)
	if err == nil && info.IsDir() {
		fp.CurrentDir = path
		fp.Refresh()
	} else {
		// It's a file, trigger selection
		if fp.OnSelect != nil {
			fp.OnSelect(path)
		}
	}
}

// Draw renders the file picker
func (fp *FilePicker) Draw(frame RenderFrame) {
	bounds := fp.GetBounds()
	if bounds.Empty() {
		return
	}

	// Input at top (height 1)
	inputHeight := 1
	inputBounds := image.Rect(0, 0, bounds.Dx(), inputHeight)

	// Separator?

	// List below
	listBounds := image.Rect(0, inputHeight+1, bounds.Dx(), bounds.Dy())

	// Update children bounds
	fp.input.SetBounds(inputBounds)
	fp.list.SetBounds(listBounds)

	// Draw Input
	inputFrame := frame.SubFrame(inputBounds)
	fp.input.Draw(inputFrame)

	// Draw Separator
	frame.PrintStyled(0, inputHeight, strings.Repeat("─", bounds.Dx()), NewStyle().WithForeground(ColorBrightBlack))

	// Draw List
	listFrame := frame.SubFrame(listBounds)
	fp.list.Draw(listFrame)

	// Draw current path in footer/header?
	// Maybe overlay at bottom or top right?
	// Let's put it in the separator for now?
	pathLabel := fmt.Sprintf(" %s ", fp.CurrentDir)
	if len(pathLabel) < bounds.Dx() {
		frame.PrintStyled(2, inputHeight, pathLabel, NewStyle().WithBackground(ColorBlack).WithForeground(ColorCyan))
	}
}

// HandleKey handles key events
func (fp *FilePicker) HandleKey(event KeyEvent) bool {
	// Pass navigation to list
	switch event.Key {
	case KeyArrowUp, KeyArrowDown, KeyPageUp, KeyPageDown, KeyHome, KeyEnd:
		return fp.list.HandleKey(event)
	case KeyEnter:
		return fp.list.HandleKey(event)
	}

	// Pass typing to input
	if fp.input.HandleKey(event) {
		return true
	}

	return false
}

// SetBounds sets the widget bounds
func (fp *FilePicker) SetBounds(bounds image.Rectangle) {
	fp.BaseWidget.SetBounds(bounds)
	// Children bounds are updated in Draw or here?
	// Better in Layout if we had one. For now Draw handles it.
}

// SetInputStyle sets the style for the input field
func (fp *FilePicker) SetInputStyle(style Style) {
	fp.input.Style = style
}

// FocusInput explicitly sets focus on the input field
func (fp *FilePicker) FocusInput() {
	fp.input.SetFocused(true)
}

// GetInputValue returns the current value of the input field (for debugging)
func (fp *FilePicker) GetInputValue() string {
	return fp.input.Value()
}

// GetInputFocused returns whether the input is focused (for debugging)
func (fp *FilePicker) GetInputFocused() bool {
	return fp.input.focused
}

// GetInputCursorPos returns the cursor position (for debugging)
func (fp *FilePicker) GetInputCursorPos() int {
	return fp.input.CursorPos
}

// SetInputValueDirect directly sets the input value (for debugging)
func (fp *FilePicker) SetInputValueDirect(val string) {
	fp.input.SetValue(val)
	fp.Filter = val
	fp.filterFiles()
}

// HandleMouse handles mouse events
func (fp *FilePicker) HandleMouse(event MouseEvent) bool {
	// Check bounds? Container checks it, but good practice.
	bounds := fp.GetBounds()
	if event.X < bounds.Min.X || event.X >= bounds.Max.X ||
		event.Y < bounds.Min.Y || event.Y >= bounds.Max.Y {
		return false
	}

	// Delegate to children
	if fp.input.HandleMouse(event) {
		return true
	}
	if fp.list.HandleMouse(event) {
		return true
	}
	return false
}
