package main

import (
	"log"

	"github.com/deepnoodle-ai/gooey/tui"
)

// AnimationDemoApp shows different text animation styles, one per line.
type AnimationDemoApp struct {
	frame uint64
}

// Animation definitions
type animDef struct {
	name string
	anim tui.TextAnimation
	text string
}

var animations = []animDef{
	{"Rainbow", tui.CreateRainbowText("", 3), "Smooth rainbow color cycling"},
	{"Reverse Rainbow", tui.CreateReverseRainbowText("", 3), "Rainbow cycling backwards"},
	{"Fast Rainbow", tui.CreateRainbowText("", 1), "Faster rainbow animation"},
	{"Cyan Pulse", tui.CreatePulseText(tui.NewRGB(0, 255, 255), 12), "Pulsing brightness effect"},
	{"Orange Pulse", tui.CreatePulseText(tui.NewRGB(255, 128, 0), 12), "Warm pulsing glow"},
	{"Green Pulse", tui.CreatePulseText(tui.NewRGB(0, 255, 128), 10), "Matrix-style pulse"},
}

func (app *AnimationDemoApp) View() tui.View {
	return tui.VStack(
		tui.Text("Text Animation Styles").Bold().Fg(tui.ColorCyan),
		tui.Spacer().MinHeight(1),

		// Use ForEach with AnimatedTextView - no more Canvas needed!
		tui.ForEach(animations, func(anim animDef, i int) tui.View {
			return tui.HStack(
				tui.Text("%-16s", anim.name+":").Fg(tui.ColorBrightBlack),
				tui.AnimatedTextView(anim.text, anim.anim, app.frame),
			)
		}).Gap(1),

		tui.Spacer(),
		tui.Text("[q] quit").Fg(tui.ColorBrightBlack),
	).Padding(1)
}

func (app *AnimationDemoApp) HandleEvent(event tui.Event) []tui.Cmd {
	switch e := event.(type) {
	case tui.TickEvent:
		app.frame = e.Frame

	case tui.KeyEvent:
		if e.Rune == 'q' || e.Rune == 'Q' || e.Key == tui.KeyEscape || e.Key == tui.KeyCtrlC {
			return []tui.Cmd{tui.Quit()}
		}
	}

	return nil
}

func main() {
	if err := tui.Run(&AnimationDemoApp{}); err != nil {
		log.Fatal(err)
	}
}
