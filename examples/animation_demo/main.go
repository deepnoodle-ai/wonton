package main

import (
	"fmt"
	"image"
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
	// Build views for each animation line
	views := []gooey.View{
		gooey.Text("Text Animation Styles").Bold().Fg(gooey.ColorCyan),
		gooey.Spacer().MinHeight(1),
	}

	// Add each animation as a canvas
	for _, a := range animations {
		anim := a // Capture for closure
		views = append(views,
			gooey.Canvas(func(frame gooey.RenderFrame, bounds image.Rectangle) {
				app.renderAnimatedLine(frame, bounds, anim)
			}),
			gooey.Spacer().MinHeight(1),
		)
	}

	// Add help text at bottom
	views = append(views,
		gooey.Spacer(),
		gooey.Text("[q] quit").Fg(gooey.ColorBrightBlack),
	)

	return gooey.VStack(views...).Padding(1)
}

func (app *AnimationDemoApp) renderAnimatedLine(frame gooey.RenderFrame, bounds image.Rectangle, anim animDef) {
	// Label
	label := gooey.NewStyle().WithForeground(gooey.ColorBrightBlack)
	labelText := fmt.Sprintf("%-16s", anim.name+":")
	frame.PrintStyled(0, 0, labelText, label)

	// Animated text
	startX := 18
	for j, ch := range anim.text {
		if startX+j >= bounds.Dx() {
			break
		}
		style := anim.anim.GetStyle(app.frame, j, len(anim.text))
		frame.SetCell(startX+j, 0, ch, style)
	}
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
