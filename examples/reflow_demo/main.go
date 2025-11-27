package main

import (
	"fmt"
	"image"
	"log"
	"strings"

	"github.com/deepnoodle-ai/gooey"
)

// ReflowApp demonstrates text reflow with animated width changes
// using the declarative View API.
type ReflowApp struct {
	width     int
	direction int
	frame     uint64
}

// HandleEvent processes events from the runtime.
func (app *ReflowApp) HandleEvent(event gooey.Event) []gooey.Cmd {
	switch e := event.(type) {
	case gooey.KeyEvent:
		// Exit on any key press or Ctrl+C
		return []gooey.Cmd{gooey.Quit()}

	case gooey.TickEvent:
		// Animate width on each tick
		app.frame = e.Frame
		app.width += app.direction
		if app.width > 60 {
			app.direction = -1
		} else if app.width < 20 {
			app.direction = 1
		}

		// Auto-quit after 200 frames
		if app.frame >= 200 {
			return []gooey.Cmd{gooey.Quit()}
		}
	}

	return nil
}

// View returns the declarative view tree.
func (app *ReflowApp) View() gooey.View {
	// Debug info at top
	debugInfo := fmt.Sprintf("Width: %d | Frame: %d/200 | Press any key to exit", app.width, app.frame)

	// The long text that will wrap based on width
	longText := "This is a long text that should wrap automatically when the container width decreases. It demonstrates text reflow in Gooey! The height of the box should adjust to fit the text."

	// Wrap the text to the current width
	wrapped := gooey.WrapText(longText, app.width-4) // -4 for border and padding

	// Create the wrapping text box using Canvas for custom width control
	wrappingBox := gooey.Canvas(func(frame gooey.RenderFrame, bounds image.Rectangle) {
		// Split into lines and render
		lines := strings.Split(wrapped, "\n")
		for i, line := range lines {
			if i >= bounds.Dy() {
				break
			}
			frame.PrintStyled(0, i, line, gooey.NewStyle())
		}
	}).Size(app.width-4, len(strings.Split(wrapped, "\n")))

	// Wrap in a border
	borderedBox := gooey.Bordered(wrappingBox).
		Border(&gooey.RoundedBorder).
		BorderFg(gooey.ColorCyan)

	// Add a button below to show it moves
	button := gooey.Text("I move down!").
		Fg(gooey.ColorGreen).
		Bold()

	// Main layout with everything centered
	return gooey.VStack(
		gooey.Text("%s", debugInfo),
		gooey.Spacer(),
		gooey.HStack(
			gooey.Spacer(),
			gooey.VStack(
				borderedBox,
				gooey.Spacer().MinHeight(1),
				button,
			),
			gooey.Spacer(),
		),
		gooey.Spacer(),
	)
}

func main() {
	app := &ReflowApp{
		width:     40,
		direction: 1,
	}
	if err := gooey.Run(app, gooey.WithFPS(20)); err != nil {
		log.Fatal(err)
	}
}
