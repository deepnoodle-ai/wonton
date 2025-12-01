package main

import (
	"log"

	"github.com/deepnoodle-ai/gooey/tui"
)

// Minimal composition test with deeply nested containers.
type App struct {
	status string
}

func (app *App) View() tui.View {
	return tui.Bordered(
		tui.VStack(
			tui.Text("Nested Layout Test").Fg(tui.ColorCyan).Bold(),
			tui.Spacer().MinHeight(1),
			tui.Text("%s", app.status).Fg(tui.ColorYellow),
			tui.Spacer().MinHeight(1),
			tui.HStack(
				tui.Bordered(
					tui.VStack(
						tui.Text("Buttons").Fg(tui.ColorGreen),
						tui.Spacer().MinHeight(1),
						tui.Clickable("Button 1", func() {
							app.status = "Button 1 clicked!"
						}),
						tui.Spacer().MinHeight(1),
						tui.Clickable("Button 2", func() {
							app.status = "Button 2 clicked!"
						}),
						tui.Spacer().MinHeight(1),
						tui.Clickable("Button 3", func() {
							app.status = "Button 3 clicked!"
						}),
					).Padding(1),
				).Border(&tui.SingleBorder),
				tui.Spacer().MinWidth(2),
				tui.Bordered(
					tui.VStack(
						tui.Text("Labels").Fg(tui.ColorMagenta),
						tui.Text("Label A"),
						tui.Text("Label B"),
						tui.Text("Label C"),
					).Padding(1),
				).Border(&tui.RoundedBorder),
			),
			tui.Spacer().MinHeight(1),
			tui.Text("Press 'q' to quit").Fg(tui.ColorBrightBlack),
		).Padding(1),
	).Border(&tui.DoubleBorder)
}

func (app *App) HandleEvent(event tui.Event) []tui.Cmd {
	switch e := event.(type) {
	case tui.KeyEvent:
		if e.Rune == 'q' || e.Rune == 'Q' || e.Key == tui.KeyEscape || e.Key == tui.KeyCtrlC {
			return []tui.Cmd{tui.Quit()}
		}
	}
	return nil
}

func main() {
	if err := tui.Run(&App{status: "Click a button..."}, tui.WithMouseTracking(true)); err != nil {
		log.Fatal(err)
	}
}
