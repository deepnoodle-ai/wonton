package main

import (
	"fmt"
	"os"

	"github.com/deepnoodle-ai/gooey"
)

// ModalDemoApp demonstrates modal dialogs using the Runtime architecture.
// It shows how to display overlay modals and handle modal-specific interactions.
type ModalDemoApp struct {
	modal          *gooey.Modal
	showModal      bool
	lastButtonText string
	width          int
	height         int
}

// HandleEvent processes events from the runtime.
func (app *ModalDemoApp) HandleEvent(event gooey.Event) []gooey.Cmd {
	switch e := event.(type) {
	case gooey.KeyEvent:
		// If modal is shown, let it handle keys first
		if app.showModal && app.modal != nil {
			if app.modal.HandleKey(e) {
				return nil
			}
		}

		// Handle keys when no modal is shown
		if e.Rune == 'm' || e.Rune == 'M' {
			// Open modal
			app.modal = gooey.NewModal(
				"Confirm Action",
				"Are you sure you want to perform this action?\nIt cannot be undone.",
				[]string{"OK", "Cancel"},
			)
			app.modal.SetCallback(func(buttonIndex int) {
				// Save which button was clicked
				if buttonIndex < len(app.modal.Buttons) {
					app.lastButtonText = app.modal.Buttons[buttonIndex]
				}
				// Close modal
				app.showModal = false
			})
			app.showModal = true
			return nil
		}

		if e.Rune == 'q' || e.Rune == 'Q' || e.Key == gooey.KeyEscape || e.Key == gooey.KeyCtrlC {
			return []gooey.Cmd{gooey.Quit()}
		}

	case gooey.ResizeEvent:
		// Update dimensions on resize
		app.width = e.Width
		app.height = e.Height
	}

	return nil
}

// Render draws the current application state.
func (app *ModalDemoApp) Render(frame gooey.RenderFrame) {
	width, height := frame.Size()

	// Clear screen
	frame.Fill(' ', gooey.NewStyle())

	// Draw header
	headerStyle := gooey.NewStyle().WithBold().WithForeground(gooey.ColorWhite).WithBackground(gooey.ColorBlue)
	headerText := " Modal Demo "
	for i := 0; i < width; i++ {
		if i >= (width-len(headerText))/2 && i < (width-len(headerText))/2+len(headerText) {
			frame.SetCell(i, 0, rune(headerText[i-(width-len(headerText))/2]), headerStyle)
		} else {
			frame.SetCell(i, 0, ' ', headerStyle)
		}
	}

	// Draw main content
	contentStyle := gooey.NewStyle().WithForeground(gooey.ColorWhite)
	msg := "Press 'm' to open modal. Press 'q' to quit."
	msgX := (width - len(msg)) / 2
	msgY := height / 2
	if msgX < 0 {
		msgX = 0
	}
	frame.PrintStyled(msgX, msgY, msg, contentStyle)

	// Show last button clicked
	if app.lastButtonText != "" {
		resultMsg := fmt.Sprintf("Last clicked: %s", app.lastButtonText)
		resultX := (width - len(resultMsg)) / 2
		if resultX < 0 {
			resultX = 0
		}
		resultStyle := gooey.NewStyle().WithForeground(gooey.ColorGreen)
		frame.PrintStyled(resultX, msgY+2, resultMsg, resultStyle)
	}

	// Draw footer
	footerStyle := gooey.NewStyle().WithForeground(gooey.ColorBlack).WithBackground(gooey.ColorWhite)
	leftText := "Left"
	centerText := "Center"
	rightText := "Right"

	if height > 0 {
		// Left aligned
		frame.PrintStyled(1, height-1, leftText, footerStyle)

		// Center aligned
		centerX := (width - len(centerText)) / 2
		if centerX < 0 {
			centerX = 0
		}
		frame.PrintStyled(centerX, height-1, centerText, footerStyle)

		// Right aligned
		rightX := width - len(rightText) - 1
		if rightX < 0 {
			rightX = 0
		}
		frame.PrintStyled(rightX, height-1, rightText, footerStyle)

		// Fill remaining footer space
		for i := 0; i < width; i++ {
			// Skip positions where we've already drawn text
			hasText := false
			if i >= 1 && i < 1+len(leftText) {
				hasText = true
			}
			if i >= centerX && i < centerX+len(centerText) {
				hasText = true
			}
			if i >= rightX && i < rightX+len(rightText) {
				hasText = true
			}
			if !hasText {
				frame.SetCell(i, height-1, ' ', footerStyle)
			}
		}
	}

	// Draw modal overlay if shown
	if app.showModal && app.modal != nil {
		app.modal.Draw(frame)
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
	app := &ModalDemoApp{
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
