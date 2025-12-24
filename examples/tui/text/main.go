package main

import (
	"log"

	"github.com/deepnoodle-ai/wonton/tui"
)

// TextDemoApp demonstrates text wrapping and alignment capabilities.
type TextDemoApp struct{}

// HandleEvent processes events from the runtime.
func (app *TextDemoApp) HandleEvent(event tui.Event) []tui.Cmd {
	switch e := event.(type) {
	case tui.KeyEvent:
		if e.Key == tui.KeyCtrlC || e.Rune == 'q' || e.Rune == 'Q' {
			return []tui.Cmd{tui.Quit()}
		}
	}
	return nil
}

// View returns the declarative UI for this app.
func (app *TextDemoApp) View() tui.View {
	longText := "This is a very long sentence that should automatically wrap when it reaches the boundary of the container. It serves as a demonstration of the text wrapping capability."

	return tui.Stack(
		// Header
		tui.Text("Text Wrapping & Alignment Demo").Bold().Fg(tui.ColorCyan),
		tui.Text("Press Q or Ctrl+C to exit").Fg(tui.ColorWhite),
		tui.Divider(),

		// 2x2 Grid using nested Group and Stack
		tui.Stack(
			// Top row
			tui.Group(
				// Top Left: Wrapped, Left Aligned
				tui.Text("WRAPPED LEFT:\n%s", longText).
					Wrap().Flex(1).
					Bg(tui.ColorBlue).Fg(tui.ColorWhite).FillBg(),

				// Top Right: Wrapped, Center Aligned
				tui.Text("WRAPPED CENTER:\n%s", longText).
					Wrap().Flex(1).Center().
					Bg(tui.ColorGreen).Fg(tui.ColorBlack).FillBg(),
			),

			// Bottom row
			tui.Group(
				// Bottom Left: Wrapped, Right Aligned
				tui.Text("WRAPPED RIGHT:\n%s", longText).
					Wrap().Flex(1).Right().
					Bg(tui.ColorRed).Fg(tui.ColorWhite).FillBg(),

				// Bottom Right: Truncated (clipped at edge, no wrapping), Center Aligned
				tui.Text("TRUNCATED (Clipped at edge):\n%s", longText).
					Flex(1).Center().
					Bg(tui.ColorYellow).Fg(tui.ColorBlack).FillBg(),
			),
		),
	)
}

func main() {
	// Run the application (30 FPS is default)
	if err := tui.Run(&TextDemoApp{}); err != nil {
		log.Fatal(err)
	}
}
