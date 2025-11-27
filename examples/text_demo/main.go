package main

import (
	"log"

	"github.com/deepnoodle-ai/gooey"
)

// TextDemoApp demonstrates text wrapping and alignment capabilities.
type TextDemoApp struct{}

// HandleEvent processes events from the runtime.
func (app *TextDemoApp) HandleEvent(event gooey.Event) []gooey.Cmd {
	switch e := event.(type) {
	case gooey.KeyEvent:
		if e.Key == gooey.KeyCtrlC || e.Rune == 'q' || e.Rune == 'Q' {
			return []gooey.Cmd{gooey.Quit()}
		}
	}
	return nil
}

// View returns the declarative UI for this app.
func (app *TextDemoApp) View() gooey.View {
	longText := "This is a very long sentence that should automatically wrap when it reaches the boundary of the container. It serves as a demonstration of the text wrapping capability."

	return gooey.VStack(
		// Header
		gooey.Text("Text Wrapping & Alignment Demo").Bold().Fg(gooey.ColorCyan),
		gooey.Text("Press Q or Ctrl+C to exit").Fg(gooey.ColorWhite),
		gooey.Divider(),

		// 2x2 Grid using nested HStack and VStack
		gooey.VStack(
			// Top row
			gooey.HStack(
				// Top Left: Wrapped, Left Aligned
				gooey.WrappedText("WRAPPED LEFT:\n"+longText).
					Bg(gooey.ColorBlue).Fg(gooey.ColorWhite).FillBg(),

				// Top Right: Wrapped, Center Aligned
				gooey.WrappedText("WRAPPED CENTER:\n"+longText).
					Center().
					Bg(gooey.ColorGreen).Fg(gooey.ColorBlack).FillBg(),
			),

			// Bottom row
			gooey.HStack(
				// Bottom Left: Wrapped, Right Aligned
				gooey.WrappedText("WRAPPED RIGHT:\n"+longText).
					Right().
					Bg(gooey.ColorRed).Fg(gooey.ColorWhite).FillBg(),

				// Bottom Right: Truncated (clipped at edge, no wrapping), Center Aligned
				gooey.WrappedText("TRUNCATED (Clipped at edge):\n"+longText).
					Truncate().Center().
					Bg(gooey.ColorYellow).Fg(gooey.ColorBlack).FillBg(),
			),
		),
	)
}

func main() {
	// Run the application (30 FPS is default)
	if err := gooey.Run(&TextDemoApp{}); err != nil {
		log.Fatal(err)
	}
}
