package main

import (
	"log"

	"github.com/deepnoodle-ai/gooey"
)

// AnimationDemoApp shows different text animation styles, one per line.
type AnimationDemoApp struct {
	frame uint64
}

// Animation definitions
type animDef struct {
	name string
	anim gooey.TextAnimation
	text string
}

var animations = []animDef{
	{"Rainbow", gooey.CreateRainbowText("", 3), "Smooth rainbow color cycling"},
	{"Reverse Rainbow", gooey.CreateReverseRainbowText("", 3), "Rainbow cycling backwards"},
	{"Fast Rainbow", gooey.CreateRainbowText("", 1), "Faster rainbow animation"},
	{"Cyan Pulse", gooey.CreatePulseText(gooey.NewRGB(0, 255, 255), 12), "Pulsing brightness effect"},
	{"Orange Pulse", gooey.CreatePulseText(gooey.NewRGB(255, 128, 0), 12), "Warm pulsing glow"},
	{"Green Pulse", gooey.CreatePulseText(gooey.NewRGB(0, 255, 128), 10), "Matrix-style pulse"},
}

func (app *AnimationDemoApp) View() gooey.View {
	return gooey.VStack(
		gooey.Text("Text Animation Styles").Bold().Fg(gooey.ColorCyan),
		gooey.Spacer().MinHeight(1),

		// Use ForEach with AnimatedTextView - no more Canvas needed!
		gooey.ForEach(animations, func(anim animDef, i int) gooey.View {
			return gooey.HStack(
				gooey.Text("%-16s", anim.name+":").Fg(gooey.ColorBrightBlack),
				gooey.AnimatedTextView(anim.text, anim.anim, app.frame),
			)
		}).Gap(1),

		gooey.Spacer(),
		gooey.Text("[q] quit").Fg(gooey.ColorBrightBlack),
	).Padding(1)
}

func (app *AnimationDemoApp) HandleEvent(event gooey.Event) []gooey.Cmd {
	switch e := event.(type) {
	case gooey.TickEvent:
		app.frame = e.Frame

	case gooey.KeyEvent:
		if e.Rune == 'q' || e.Rune == 'Q' || e.Key == gooey.KeyEscape || e.Key == gooey.KeyCtrlC {
			return []gooey.Cmd{gooey.Quit()}
		}
	}

	return nil
}

func main() {
	if err := gooey.Run(&AnimationDemoApp{}); err != nil {
		log.Fatal(err)
	}
}
