package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/deepnoodle-ai/gooey"
)

// CheckboxDemoApp demonstrates the CheckboxGroup widget using the Runtime architecture.
// It shows how to handle keyboard events and render state without manual loops or goroutines.
type CheckboxDemoApp struct {
	checkbox *gooey.CheckboxGroup
	width    int
	height   int
}

// Init initializes the application by creating the checkbox widget.
func (app *CheckboxDemoApp) Init() error {
	app.checkbox = gooey.NewCheckboxGroup(2, 4, []string{
		"Apple",
		"Banana",
		"Cherry",
		"Date",
		"Elderberry",
	})
	return nil
}

// HandleEvent processes events from the runtime.
func (app *CheckboxDemoApp) HandleEvent(event gooey.Event) []gooey.Cmd {
	switch e := event.(type) {
	case gooey.KeyEvent:
		// Quit on 'q'
		if e.Rune == 'q' || e.Rune == 'Q' {
			return []gooey.Cmd{gooey.Quit()}
		}

		// Pass other keys to checkbox
		app.checkbox.HandleKey(e)

	case gooey.ResizeEvent:
		// Update dimensions on resize
		app.width = e.Width
		app.height = e.Height
	}

	return nil
}

// Render draws the current application state.
func (app *CheckboxDemoApp) Render(frame gooey.RenderFrame) {
	width, height := frame.Size()

	// Clear screen
	frame.Fill(' ', gooey.NewStyle())

	// Draw header
	headerStyle := gooey.NewStyle().WithBold().WithBackground(gooey.ColorBlue).WithForeground(gooey.ColorWhite)
	headerText := " Checkbox Demo "
	for i := 0; i < width; i++ {
		if i >= (width-len(headerText))/2 && i < (width-len(headerText))/2+len(headerText) {
			frame.SetCell(i, 0, rune(headerText[i-(width-len(headerText))/2]), headerStyle)
		} else {
			frame.SetCell(i, 0, ' ', headerStyle)
		}
	}

	// Draw separator
	separatorStyle := gooey.NewStyle().WithForeground(gooey.ColorBrightBlack)
	for i := 0; i < width; i++ {
		frame.SetCell(i, 1, '─', separatorStyle)
	}

	// Draw checkbox
	app.checkbox.Draw(frame)

	// Draw selected items below checkbox
	selected := app.checkbox.GetSelectedItems()
	msg := "Selected: " + strings.Join(selected, ", ")
	if len(selected) == 0 {
		msg = "Selected: (none)"
	}
	msgStyle := gooey.NewStyle().WithForeground(gooey.ColorGreen)
	frame.PrintStyled(2, 10, msg, msgStyle)

	// Draw help text
	helpStyle := gooey.NewStyle().WithDim()
	frame.PrintStyled(2, 12, "Press Space to toggle, Arrows to move, q to quit.", helpStyle)

	// Draw footer
	if height > 0 {
		footerStyle := gooey.NewStyle().WithBackground(gooey.ColorBrightBlack).WithForeground(gooey.ColorWhite)
		footerText := " Press 'q' to quit "
		footerX := (width - len(footerText)) / 2
		if footerX < 0 {
			footerX = 0
		}
		for i := 0; i < width; i++ {
			if i >= footerX && i < footerX+len(footerText) {
				frame.SetCell(i, height-1, rune(footerText[i-footerX]), footerStyle)
			} else {
				frame.SetCell(i, height-1, ' ', footerStyle)
			}
		}
	}
}

func main() {
	// Create and initialize terminal
	terminal, err := gooey.NewTerminal()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create terminal: %v\n", err)
		os.Exit(1)
	}
	defer terminal.Close()

	// Get initial terminal size
	width, height := terminal.Size()

	// Create the application
	app := &CheckboxDemoApp{
		width:  width,
		height: height,
	}

	// Create and run the runtime with 30 FPS
	runtime := gooey.NewRuntime(terminal, app, 30)

	// Run blocks until the application quits
	if err := runtime.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Runtime error: %v\n", err)
		os.Exit(1)
	}
}
