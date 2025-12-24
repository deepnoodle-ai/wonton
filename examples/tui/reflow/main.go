package main

import (
	"fmt"
	"log"

	"github.com/deepnoodle-ai/wonton/tui"
)

// ReflowApp demonstrates text reflow with animated width changes
// using the declarative View API.
type ReflowApp struct {
	width     int
	direction int
	frame     uint64
}

// HandleEvent processes events from the runtime.
func (app *ReflowApp) HandleEvent(event tui.Event) []tui.Cmd {
	switch e := event.(type) {
	case tui.KeyEvent:
		// Exit on any key press or Ctrl+C
		return []tui.Cmd{tui.Quit()}

	case tui.TickEvent:
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
			return []tui.Cmd{tui.Quit()}
		}
	}

	return nil
}

// View returns the declarative view tree.
func (app *ReflowApp) View() tui.View {
	// Debug info at top
	debugInfo := fmt.Sprintf("Width: %d | Frame: %d/200 | Press any key to exit", app.width, app.frame)

	// The long text that will wrap based on width
	longText := "This is a long text that should wrap automatically when the container width decreases. It demonstrates text reflow in Wonton! The height of the box should adjust to fit the text."

	// Wrap in a border
	borderedBox := tui.Bordered(tui.MaxWidth(app.width-4, tui.Text("%s", longText).Wrap().Flex(1))).
		Border(&tui.RoundedBorder).
		BorderFg(tui.ColorCyan)

	// Add a button below to show it moves
	button := tui.Text("I move down!").
		Fg(tui.ColorGreen).
		Bold()

	// Main layout with everything centered
	return tui.Stack(
		tui.Text("%s", debugInfo),
		tui.Stack(
			borderedBox,
			button,
		).Gap(1),
	).Gap(1)
}

func main() {
	app := &ReflowApp{
		width:     40,
		direction: 1,
	}
	if err := tui.Run(app, tui.WithFPS(20)); err != nil {
		log.Fatal(err)
	}
}
