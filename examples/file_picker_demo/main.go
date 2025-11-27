package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/deepnoodle-ai/gooey"
)

// FilePickerDemoApp demonstrates the declarative FilePicker view.
type FilePickerDemoApp struct {
	currentDir string
	files      []gooey.ListItem
	filter     string
	selected   int
	showHidden bool
	statusMsg  string
	width      int
	height     int
}

// Init initializes the application.
func (app *FilePickerDemoApp) Init() error {
	pwd, err := os.Getwd()
	if err != nil {
		pwd = "/"
	}
	app.currentDir = pwd
	app.refreshFiles()
	app.statusMsg = "Type to filter, arrows to navigate, Enter to select, F2 toggle hidden, Esc quit"
	return nil
}

// refreshFiles reads the current directory and updates the file list.
func (app *FilePickerDemoApp) refreshFiles() {
	app.filter = "" // Reset filter on dir change
	app.selected = 0

	files, err := os.ReadDir(app.currentDir)
	if err != nil {
		app.files = []gooey.ListItem{{Label: fmt.Sprintf("Error: %v", err)}}
		return
	}

	var items []gooey.ListItem

	// Add ".." if not at root
	parent := filepath.Dir(app.currentDir)
	if parent != app.currentDir {
		items = append(items, gooey.ListItem{Label: "..", Value: parent, Icon: "[DIR]"})
	}

	// Sort: Directories first, then files
	var dirs []os.DirEntry
	var regular []os.DirEntry

	for _, f := range files {
		if !app.showHidden && strings.HasPrefix(f.Name(), ".") {
			continue
		}
		if f.IsDir() {
			dirs = append(dirs, f)
		} else {
			regular = append(regular, f)
		}
	}

	for _, d := range dirs {
		items = append(items, gooey.ListItem{
			Label: "[DIR] " + d.Name(),
			Value: filepath.Join(app.currentDir, d.Name()),
			Icon:  "[DIR]",
		})
	}

	for _, f := range regular {
		items = append(items, gooey.ListItem{
			Label: f.Name(),
			Value: filepath.Join(app.currentDir, f.Name()),
		})
	}

	app.files = items
}

// HandleEvent processes events from the runtime.
func (app *FilePickerDemoApp) HandleEvent(event gooey.Event) []gooey.Cmd {
	switch e := event.(type) {
	case gooey.KeyEvent:
		// Quit on Ctrl+C or Escape
		if e.Key == gooey.KeyCtrlC || e.Key == gooey.KeyEscape {
			return []gooey.Cmd{gooey.Quit()}
		}

		// Toggle hidden files on F2
		if e.Key == gooey.KeyF2 {
			app.showHidden = !app.showHidden
			app.refreshFiles()
			if app.showHidden {
				app.statusMsg = "Hidden files: ON"
			} else {
				app.statusMsg = "Hidden files: OFF"
			}
			return nil
		}

		// Handle navigation (these are passed to SelectList via InputRegistry)
		switch e.Key {
		case gooey.KeyArrowUp:
			if app.selected > 0 {
				app.selected--
			}
			return nil
		case gooey.KeyArrowDown:
			if app.selected < len(app.files)-1 {
				app.selected++
			}
			return nil
		case gooey.KeyEnter:
			app.handleSelect()
			return nil
		}

	case gooey.ResizeEvent:
		app.width = e.Width
		app.height = e.Height
	}

	return nil
}

// handleSelect handles item selection.
func (app *FilePickerDemoApp) handleSelect() {
	if app.selected >= len(app.files) {
		return
	}

	item := app.files[app.selected]
	path, ok := item.Value.(string)
	if !ok {
		return
	}

	info, err := os.Stat(path)
	if err != nil {
		app.statusMsg = fmt.Sprintf("Error: %v", err)
		return
	}

	if info.IsDir() {
		app.currentDir = path
		app.refreshFiles()
		app.statusMsg = fmt.Sprintf("Opened: %s", path)
	} else {
		size := info.Size()
		var sizeStr string
		if size < 1024 {
			sizeStr = fmt.Sprintf("%d B", size)
		} else if size < 1024*1024 {
			sizeStr = fmt.Sprintf("%.1f KB", float64(size)/1024)
		} else {
			sizeStr = fmt.Sprintf("%.1f MB", float64(size)/(1024*1024))
		}
		app.statusMsg = fmt.Sprintf("Selected: %s (%s)", filepath.Base(path), sizeStr)
	}
}

// View returns the declarative view structure.
func (app *FilePickerDemoApp) View() gooey.View {
	pickerHeight := app.height - 6
	if pickerHeight < 5 {
		pickerHeight = 5
	}

	return gooey.VStack(
		gooey.Text("FILE PICKER DEMO").Bold().Fg(gooey.ColorCyan),
		gooey.Spacer().MinHeight(1),
		gooey.FilePicker(app.files, &app.filter, &app.selected).
			CurrentPath(app.currentDir).
			Height(pickerHeight).
			OnSelect(func(item gooey.ListItem) {
				app.handleSelect()
			}),
		gooey.Spacer().MinHeight(1),
		gooey.Text("%s", app.statusMsg).Fg(gooey.ColorGreen),
	)
}

func main() {
	app := &FilePickerDemoApp{}
	if err := app.Init(); err != nil {
		log.Fatal(err)
	}
	if err := gooey.Run(app, gooey.WithMouseTracking(true)); err != nil {
		log.Fatal(err)
	}
}
