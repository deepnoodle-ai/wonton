package main

import (
	"image"
	"log"
	"strings"

	"github.com/deepnoodle-ai/gooey"
)

// CheckboxDemoApp demonstrates the CheckboxGroup widget using declarative View() style.
// It shows how to handle keyboard events and render state without manual loops or goroutines.
type CheckboxDemoApp struct {
	checkbox *gooey.CheckboxGroup
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
		if e.Rune == 'q' || e.Rune == 'Q' || e.Key == gooey.KeyEscape || e.Key == gooey.KeyCtrlC {
			return []gooey.Cmd{gooey.Quit()}
		}
		app.checkbox.HandleKey(e)
	}

	return nil
}

// View returns the declarative view structure.
func (app *CheckboxDemoApp) View() gooey.View {
	// Get selected items for display
	selected := app.checkbox.GetSelectedItems()
	selectedMsg := "Selected: " + strings.Join(selected, ", ")
	if len(selected) == 0 {
		selectedMsg = "Selected: (none)"
	}

	return gooey.VStack(
		// Header
		gooey.Canvas(func(frame gooey.RenderFrame, bounds image.Rectangle) {
			width := bounds.Dx()
			headerStyle := gooey.NewStyle().WithBold().WithBackground(gooey.ColorBlue).WithForeground(gooey.ColorWhite)
			headerText := " Checkbox Demo "
			// Center the header text
			startX := (width - len(headerText)) / 2
			if startX < 0 {
				startX = 0
			}
			// Fill entire width with header background
			for i := 0; i < width; i++ {
				if i >= startX && i < startX+len(headerText) {
					frame.SetCell(i, 0, rune(headerText[i-startX]), headerStyle)
				} else {
					frame.SetCell(i, 0, ' ', headerStyle)
				}
			}
		}).Height(1),

		// Separator
		gooey.Canvas(func(frame gooey.RenderFrame, bounds image.Rectangle) {
			width := bounds.Dx()
			separatorStyle := gooey.NewStyle().WithForeground(gooey.ColorBrightBlack)
			for i := 0; i < width; i++ {
				frame.SetCell(i, 0, '─', separatorStyle)
			}
		}).Height(1),

		// Spacing
		gooey.Spacer().MinHeight(1),

		// Checkbox widget
		gooey.Canvas(func(frame gooey.RenderFrame, bounds image.Rectangle) {
			app.checkbox.Draw(frame)
		}).Height(len(app.checkbox.Options)).Width(30),

		// Spacing
		gooey.Spacer().MinHeight(1),

		// Selected items display
		gooey.Text(selectedMsg).Fg(gooey.ColorGreen),

		// Spacing
		gooey.Spacer().MinHeight(1),

		// Help text
		gooey.Text("Press Space to toggle, Arrows to move, q to quit.").Dim(),

		// Flexible spacer to push footer to bottom
		gooey.Spacer(),

		// Footer
		gooey.Canvas(func(frame gooey.RenderFrame, bounds image.Rectangle) {
			width := bounds.Dx()
			footerStyle := gooey.NewStyle().WithBackground(gooey.ColorBrightBlack).WithForeground(gooey.ColorWhite)
			footerText := " Press 'q' to quit "
			footerX := (width - len(footerText)) / 2
			if footerX < 0 {
				footerX = 0
			}
			for i := 0; i < width; i++ {
				if i >= footerX && i < footerX+len(footerText) {
					frame.SetCell(i, 0, rune(footerText[i-footerX]), footerStyle)
				} else {
					frame.SetCell(i, 0, ' ', footerStyle)
				}
			}
		}).Height(1),
	).PaddingLTRB(2, 0, 2, 0)
}

func main() {
	if err := gooey.Run(&CheckboxDemoApp{}); err != nil {
		log.Fatal(err)
	}
}
