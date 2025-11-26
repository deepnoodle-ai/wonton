package main

import (
	"log"

	"github.com/deepnoodle-ai/gooey"
)

// Minimal composition test with deeply nested containers.
type App struct {
	status string
}

func (app *App) View() gooey.View {
	return gooey.Bordered(
		gooey.VStack(
			gooey.Text("Nested Layout Test").Fg(gooey.ColorCyan).Bold(),
			gooey.Spacer().MinHeight(1),
			gooey.Text(app.status).Fg(gooey.ColorYellow),
			gooey.Spacer().MinHeight(1),
			gooey.HStack(
				gooey.Bordered(
					gooey.VStack(
						gooey.Text("Buttons").Fg(gooey.ColorGreen),
						gooey.Spacer().MinHeight(1),
						gooey.Clickable("Button 1", func() {
							app.status = "Button 1 clicked!"
						}),
						gooey.Spacer().MinHeight(1),
						gooey.Clickable("Button 2", func() {
							app.status = "Button 2 clicked!"
						}),
						gooey.Spacer().MinHeight(1),
						gooey.Clickable("Button 3", func() {
							app.status = "Button 3 clicked!"
						}),
					).Padding(1),
				).Border(&gooey.SingleBorder),
				gooey.Spacer().MinWidth(2),
				gooey.Bordered(
					gooey.VStack(
						gooey.Text("Labels").Fg(gooey.ColorMagenta),
						gooey.Text("Label A"),
						gooey.Text("Label B"),
						gooey.Text("Label C"),
					).Padding(1),
				).Border(&gooey.RoundedBorder),
			),
			gooey.Spacer().MinHeight(1),
			gooey.Text("Press 'q' to quit").Fg(gooey.ColorBrightBlack),
		).Padding(1),
	).Border(&gooey.DoubleBorder)
}

func (app *App) HandleEvent(event gooey.Event) []gooey.Cmd {
	switch e := event.(type) {
	case gooey.KeyEvent:
		if e.Rune == 'q' || e.Rune == 'Q' || e.Key == gooey.KeyEscape || e.Key == gooey.KeyCtrlC {
			return []gooey.Cmd{gooey.Quit()}
		}
	}
	return nil
}

func main() {
	if err := gooey.Run(&App{status: "Click a button..."}, gooey.WithMouseTracking(true)); err != nil {
		log.Fatal(err)
	}
}
