package main

import (
	"log"

	"github.com/deepnoodle-ai/wonton/tui"
)

// AnimationDemoApp shows different text animation styles, one per line.
type AnimationDemoApp struct{}

func (app *AnimationDemoApp) View() tui.View {
	return tui.Stack(
		tui.Text("Text Animation Styles").Bold().Fg(tui.ColorCyan),
		tui.Spacer().MinHeight(1),

		// Rainbow animations
		tui.Group(
			tui.Text("Rainbow:         ").Fg(tui.ColorBrightBlack),
			tui.Text("Smooth rainbow color cycling").Animate(tui.Rainbow(3)),
		),
		tui.Group(
			tui.Text("Reverse Rainbow: ").Fg(tui.ColorBrightBlack),
			tui.Text("Rainbow cycling backwards").Animate(tui.Rainbow(3).Reverse()),
		),
		tui.Group(
			tui.Text("Fast Rainbow:    ").Fg(tui.ColorBrightBlack),
			tui.Text("Faster rainbow animation").Animate(tui.Rainbow(1)),
		),

		tui.Spacer().MinHeight(1),

		// Pulse animations
		tui.Group(
			tui.Text("Cyan Pulse:      ").Fg(tui.ColorBrightBlack),
			tui.Text("Pulsing brightness effect").Animate(tui.Pulse(tui.NewRGB(0, 255, 255), 12)),
		),
		tui.Group(
			tui.Text("Orange Pulse:    ").Fg(tui.ColorBrightBlack),
			tui.Text("Warm pulsing glow").Animate(tui.Pulse(tui.NewRGB(255, 128, 0), 12)),
		),
		tui.Group(
			tui.Text("Green Pulse:     ").Fg(tui.ColorBrightBlack),
			tui.Text("Matrix-style pulse").Animate(tui.Pulse(tui.NewRGB(0, 255, 128), 10)),
		),

		tui.Spacer().MinHeight(1),

		// Sparkle animations - twinkling star effect
		tui.Group(
			tui.Text("Sparkle:         ").Fg(tui.ColorBrightBlack),
			tui.Text("Twinkling like distant stars").Animate(tui.Sparkle(3, tui.NewRGB(180, 180, 220), tui.NewRGB(255, 255, 255))),
		),
		tui.Group(
			tui.Text("Gold Sparkle:    ").Fg(tui.ColorBrightBlack),
			tui.Text("Glittering treasure").Animate(tui.Sparkle(2, tui.NewRGB(180, 140, 0), tui.NewRGB(255, 240, 150))),
		),

		tui.Spacer().MinHeight(1),

		// Typewriter animations - text reveal effect
		tui.Group(
			tui.Text("Typewriter:      ").Fg(tui.ColorBrightBlack),
			tui.Text("Characters appear one by one...").Animate(tui.Typewriter(3, tui.NewRGB(0, 255, 100), tui.NewRGB(255, 255, 255)).WithLoop(true)),
		),
		tui.Group(
			tui.Text("Terminal:        ").Fg(tui.ColorBrightBlack),
			tui.Text("System initialized successfully").Animate(tui.Typewriter(2, tui.NewRGB(255, 180, 0), tui.NewRGB(255, 100, 0)).WithLoop(true)),
		),

		tui.Spacer().MinHeight(1),

		// Glitch animations - cyberpunk effect
		tui.Group(
			tui.Text("Glitch:          ").Fg(tui.ColorBrightBlack),
			tui.Text("SIGNAL CORRUPTED").Animate(tui.Glitch(2, tui.NewRGB(255, 0, 100), tui.NewRGB(0, 255, 255))),
		),
		tui.Group(
			tui.Text("Cyber:           ").Fg(tui.ColorBrightBlack),
			tui.Text("NEURAL LINK ACTIVE").Animate(tui.Glitch(2, tui.NewRGB(0, 200, 255), tui.NewRGB(255, 0, 255))),
		),

		tui.Spacer().MinHeight(1),

		// Slide animations
		tui.Group(
			tui.Text("Slide:           ").Fg(tui.ColorBrightBlack),
			tui.Text("Highlight slides across").Animate(tui.Slide(2, tui.NewRGB(100, 100, 100), tui.NewRGB(255, 255, 255))),
		),
		tui.Group(
			tui.Text("Slide Reverse:   ").Fg(tui.ColorBrightBlack),
			tui.Text("Right to left shine").Animate(tui.Slide(2, tui.NewRGB(0, 100, 200), tui.NewRGB(100, 200, 255)).Reversed()),
		),

		tui.Spacer().MinHeight(1),

		// Wave animations
		tui.Group(
			tui.Text("Default Wave:    ").Fg(tui.ColorBrightBlack),
			tui.Text("Multi-color wave effect").Animate(tui.Wave(12)),
		),
		tui.Group(
			tui.Text("Fire Wave:       ").Fg(tui.ColorBrightBlack),
			tui.Text("Hot fiery colors").Animate(tui.Wave(12,
				tui.NewRGB(255, 50, 0),
				tui.NewRGB(255, 150, 0),
				tui.NewRGB(255, 255, 0),
			)),
		),
		tui.Group(
			tui.Text("Ocean Wave:      ").Fg(tui.ColorBrightBlack),
			tui.Text("Cool ocean blues").Animate(tui.Wave(12,
				tui.NewRGB(0, 100, 200),
				tui.NewRGB(0, 200, 255),
				tui.NewRGB(100, 255, 255),
			)),
		),

		tui.Spacer(),
		tui.Text("[q] quit").Fg(tui.ColorBrightBlack),
	).Gap(1).Padding(1)
}

func (app *AnimationDemoApp) HandleEvent(event tui.Event) []tui.Cmd {
	switch e := event.(type) {
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
