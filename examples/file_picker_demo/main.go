package main

import (
	"fmt"
	"image"
	"log"
	"os"
	"path/filepath"

	"github.com/deepnoodle-ai/gooey"
)

// FilePickerDemoApp demonstrates the FilePicker widget using the Runtime architecture.
// It shows file browsing with filtering, mouse support, and keyboard navigation.
//
// Features:
// - Type to filter files (fuzzy matching)
// - Arrow keys to navigate
// - Enter to select file or open directory
// - F2 to toggle hidden files
// - Mouse click to select
// - Escape or Ctrl+C to quit

type FilePickerDemoApp struct {
	picker    *gooey.FilePicker
	statusMsg string
	width     int
	height    int
}

// Init initializes the application by creating the file picker.
func (app *FilePickerDemoApp) Init() error {
	pwd, err := os.Getwd()
	if err != nil {
		pwd = "/"
	}

	app.picker = gooey.NewFilePicker(pwd)

	// Make the input more visible
	app.picker.SetInputStyle(gooey.NewStyle().
		WithBackground(gooey.ColorBlue).
		WithForeground(gooey.ColorYellow).
		WithBold())

	app.picker.Init()
	app.picker.FocusInput()

	// Set initial bounds
	pickerHeight := app.height - 4
	if pickerHeight < 5 {
		pickerHeight = 5
	}
	app.picker.SetBounds(image.Rect(0, 0, app.width, pickerHeight))

	// Set up selection callback
	app.picker.OnSelect = func(path string) {
		info, err := os.Stat(path)
		if err != nil {
			app.statusMsg = fmt.Sprintf("Error: %v", err)
		} else {
			if info.IsDir() {
				app.statusMsg = fmt.Sprintf("Opened directory: %s", path)
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
	}

	app.statusMsg = "Type to filter, arrows to navigate, Enter to select, F2 toggle hidden, Esc/Ctrl+C quit"

	return nil
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
			app.picker.ShowHidden = !app.picker.ShowHidden
			app.picker.Refresh()
			if app.picker.ShowHidden {
				app.statusMsg = "Hidden files: ON"
			} else {
				app.statusMsg = "Hidden files: OFF"
			}
			return nil
		}

		// Pass keys to picker
		app.picker.HandleKey(e)

	case gooey.MouseEvent:
		// Pass mouse events to picker
		app.picker.HandleMouse(e)

	case gooey.ResizeEvent:
		// Update dimensions and picker bounds on resize
		app.width = e.Width
		app.height = e.Height

		pickerHeight := e.Height - 4
		if pickerHeight < 5 {
			pickerHeight = 5
		}
		app.picker.SetBounds(image.Rect(0, 0, e.Width, pickerHeight))
	}

	return nil
}

// Render draws the current application state.
func (app *FilePickerDemoApp) Render(frame gooey.RenderFrame) {
	width, height := frame.Size()

	// Clear screen
	frame.Fill(' ', gooey.NewStyle())

	// Draw title
	titleStyle := gooey.NewStyle().WithBold().WithForeground(gooey.ColorCyan)
	title := "FILE PICKER DEMO"
	titleX := (width - len(title)) / 2
	if titleX < 0 {
		titleX = 0
	}
	frame.PrintStyled(titleX, 0, title, titleStyle)

	// Draw separator
	separatorStyle := gooey.NewStyle().WithBackground(gooey.ColorBrightBlack)
	for i := 0; i < width; i++ {
		frame.SetCell(i, 1, ' ', separatorStyle)
	}

	// Draw file picker
	bounds := app.picker.GetBounds()
	pickerFrame := frame.SubFrame(image.Rect(0, 2, width, 2+bounds.Dy()))
	app.picker.Draw(pickerFrame)

	// Draw separator before status
	for i := 0; i < width; i++ {
		frame.SetCell(i, height-2, ' ', separatorStyle)
	}

	// Draw status
	statusStyle := gooey.NewStyle().WithForeground(gooey.ColorGreen)
	statusText := app.statusMsg
	if len(statusText) > width {
		statusText = statusText[:width-3] + "..."
	}
	frame.PrintStyled(0, height-1, statusText, statusStyle)
}

func main() {
	if err := gooey.Run(&FilePickerDemoApp{}, gooey.WithMouseTracking(true)); err != nil {
		log.Fatal(err)
	}
}
