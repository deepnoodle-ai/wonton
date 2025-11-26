package main

import (
	"fmt"
	"log"

	"github.com/deepnoodle-ai/gooey"
)

// ModalDemoApp demonstrates modal dialogs using the declarative View API.
// It shows how to display overlay modals and handle modal-specific interactions.
type ModalDemoApp struct {
	modal          *gooey.Modal
	showModal      bool
	lastButtonText string
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
	}

	return nil
}

// View returns the declarative UI for this app.
func (app *ModalDemoApp) View() gooey.View {
	// Main content without modal
	mainContent := gooey.VStack(
		// Header
		gooey.Text(" Modal Demo ").
			Bold().
			Fg(gooey.ColorWhite).
			Bg(gooey.ColorBlue).
			Width(0), // Full width
		gooey.Spacer(),
		// Main message
		gooey.Text("Press 'm' to open modal. Press 'q' to quit.").
			Fg(gooey.ColorWhite),
		gooey.Spacer().MinHeight(2),
		// Last button clicked
		gooey.If(app.lastButtonText != "",
			gooey.Text("Last clicked: %s", app.lastButtonText).
				Fg(gooey.ColorGreen),
		),
		gooey.Spacer(),
		// Footer
		gooey.Padding(1,
			gooey.HStack(
				gooey.Text("Left"),
				gooey.Spacer(),
				gooey.Text("Center"),
				gooey.Spacer(),
				gooey.Text("Right"),
			).Bg(gooey.ColorWhite),
		),
	).Align(gooey.AlignCenter)

	// If modal is shown, overlay it on top using ZStack
	if app.showModal && app.modal != nil {
		return gooey.ZStack(
			mainContent,
			app.modalView(),
		)
	}

	return mainContent
}

// modalView creates the modal dialog view
func (app *ModalDemoApp) modalView() gooey.View {
	if app.modal == nil {
		return gooey.Text("")
	}

	// Create buttons
	var buttons []gooey.View
	for i, buttonText := range app.modal.Buttons {
		idx := i // capture for closure
		buttons = append(buttons,
			gooey.Clickable(fmt.Sprintf("[ %s ]", buttonText), func() {
				if app.modal.Callback != nil {
					app.modal.Callback(idx)
				}
			}).Fg(gooey.ColorWhite).Bg(gooey.ColorBlue),
		)
	}

	// Modal content
	return gooey.Bordered(
		gooey.Padding(2,
			gooey.VStack(
				gooey.Text("%s", app.modal.Title).Bold().Fg(gooey.ColorCyan),
				gooey.Spacer().MinHeight(1),
				gooey.Text("%s", app.modal.Content).Fg(gooey.ColorWhite),
				gooey.Spacer().MinHeight(2),
				gooey.HStack(buttons...).Gap(2).Align(gooey.AlignCenter),
			).Bg(gooey.ColorBlack),
		),
	)
}

func main() {
	if err := gooey.Run(&ModalDemoApp{}, gooey.WithMouseTracking(true)); err != nil {
		log.Fatal(err)
	}
}
