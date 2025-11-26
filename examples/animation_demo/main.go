package main

import (
	"fmt"
	"log"

	"github.com/deepnoodle-ai/gooey"
)

// AnimationDemoApp shows different text animation styles, one per line.
type AnimationDemoApp struct {
	frame  uint64
	width  int
	height int
}

func (app *AnimationDemoApp) HandleEvent(event gooey.Event) []gooey.Cmd {
	switch e := event.(type) {
	case gooey.TickEvent:
		app.frame = e.Frame

	case gooey.KeyEvent:
		if e.Rune == 'q' || e.Rune == 'Q' || e.Key == gooey.KeyEscape || e.Key == gooey.KeyCtrlC {
			return []gooey.Cmd{gooey.Quit()}
		}

	case gooey.ResizeEvent:
		app.width = e.Width
		app.height = e.Height
	}

	return nil
}

func (app *AnimationDemoApp) Render(frame gooey.RenderFrame) {
	width, height := frame.Size()

	// Styles
	title := gooey.NewStyle().WithBold().WithForeground(gooey.ColorCyan)
	label := gooey.NewStyle().WithForeground(gooey.ColorBrightBlack)
	dim := gooey.NewStyle().WithForeground(gooey.ColorBrightBlack)

	// Clear
	frame.FillStyled(0, 0, width, height, ' ', gooey.NewStyle())

	// Title
	frame.PrintStyled(2, 1, "Text Animation Styles", title)

	// Animation definitions
	animations := []struct {
		name string
		anim gooey.TextAnimation
		text string
	}{
		{"Rainbow", gooey.CreateRainbowText("", 3), "Smooth rainbow color cycling"},
		{"Reverse Rainbow", gooey.CreateReverseRainbowText("", 3), "Rainbow cycling backwards"},
		{"Fast Rainbow", gooey.CreateRainbowText("", 1), "Faster rainbow animation"},
		{"Cyan Pulse", gooey.CreatePulseText(gooey.NewRGB(0, 255, 255), 12), "Pulsing brightness effect"},
		{"Orange Pulse", gooey.CreatePulseText(gooey.NewRGB(255, 128, 0), 12), "Warm pulsing glow"},
		{"Green Pulse", gooey.CreatePulseText(gooey.NewRGB(0, 255, 128), 10), "Matrix-style pulse"},
	}

	// Render each animation on its own line
	startY := 4
	for i, a := range animations {
		y := startY + i*2
		if y >= height-2 {
			break
		}

		// Label
		frame.PrintStyled(2, y, fmt.Sprintf("%-16s", a.name+":"), label)

		// Animated text
		for j, ch := range a.text {
			style := a.anim.GetStyle(app.frame, j, len(a.text))
			frame.SetCell(20+j, y, ch, style)
		}
	}

	// Help
	frame.PrintStyled(2, height-1, "[q] quit", dim)
}

func main() {
	if err := gooey.Run(&AnimationDemoApp{}); err != nil {
		log.Fatal(err)
	}
}
