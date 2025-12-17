package main

import (
	"log"

	"github.com/deepnoodle-ai/wonton/tui"
)

// AnimationShowcaseApp demonstrates border and text animations.
type AnimationShowcaseApp struct {
	currentEffect int
	effects       []Effect
}

type Effect struct {
	name        string
	description string
	view        func() tui.View
}

func (app *AnimationShowcaseApp) Init() []tui.Cmd {
	app.currentEffect = 0

	// Define all effects to showcase
	app.effects = []Effect{
		{
			name:        "Border: Rainbow",
			description: "Smooth rainbow cycling around border",
			view: func() tui.View {
				return tui.AnimatedBordered(
					app.demoPanel("Rainbow border"),
					tui.BorderAnimationPresets.Rainbow(2, false),
				).Title("Rainbow")
			},
		},
		{
			name:        "Border: Pulsing",
			description: "Border pulses with breathing effect",
			view: func() tui.View {
				return tui.AnimatedBordered(
					app.demoPanel("Pulsing border"),
					tui.BorderAnimationPresets.Pulsing(tui.NewRGB(255, 100, 200), 12),
				).Title("Pulse")
			},
		},
		{
			name:        "Border: Marquee",
			description: "Classic theater marquee lights",
			view: func() tui.View {
				return tui.AnimatedBordered(
					app.demoPanel("Marquee lights"),
					tui.BorderAnimationPresets.Marquee(
						tui.NewRGB(255, 255, 0),
						tui.NewRGB(100, 100, 0),
						2,
						3,
					),
				).Title("Marquee")
			},
		},
		{
			name:        "Border: Fire",
			description: "Flickering fire effect",
			view: func() tui.View {
				return tui.AnimatedBordered(
					app.demoPanel("Burning border"),
					tui.BorderAnimationPresets.Fire(1),
				).Title("Fire")
			},
		},
		{
			name:        "Corner Highlight",
			description: "Highlights travel around corners",
			view: func() tui.View {
				return tui.AnimatedBordered(
					app.demoPanel("Watch the corners\nlight up in sequence"),
					&tui.CornerHighlightAnimation{
						Speed:          2,
						BaseColor:      tui.NewRGB(100, 100, 100),
						HighlightColor: tui.NewRGB(255, 255, 100),
						Duration:       40,
					},
				).Title("Corners")
			},
		},
		{
			name:        "Gradient Border",
			description: "Rotating color gradient",
			view: func() tui.View {
				return tui.AnimatedBordered(
					app.demoPanel("Gradient rotates\naround border"),
					&tui.GradientBorderAnimation{
						Speed: 3,
						Colors: []tui.RGB{
							tui.NewRGB(255, 0, 0),
							tui.NewRGB(255, 127, 0),
							tui.NewRGB(255, 255, 0),
							tui.NewRGB(0, 255, 0),
							tui.NewRGB(0, 0, 255),
							tui.NewRGB(75, 0, 130),
							tui.NewRGB(148, 0, 211),
						},
					},
				).Title("Gradient")
			},
		},
		{
			name:        "Text: Rainbow",
			description: "Rainbow colors cycling through text",
			view: func() tui.View {
				return tui.AnimatedBordered(
					tui.Stack(
						tui.Text("Rainbow text animation").Bold().Rainbow(2),
						tui.Spacer().MinHeight(1),
						tui.Text("Each character gets a different color"),
						tui.Text("that cycles smoothly over time").Rainbow(3),
					).Padding(2),
					tui.BorderAnimationPresets.Rainbow(4, false),
				).Title("Rainbow Text")
			},
		},
		{
			name:        "Text: Pulse",
			description: "Text brightness pulsing",
			view: func() tui.View {
				return tui.AnimatedBordered(
					tui.Stack(
						tui.Text("Pulsing text").Bold().Pulse(tui.NewRGB(255, 200, 100), 15),
						tui.Spacer().MinHeight(1),
						tui.Text("Brightness fades in and out").Pulse(tui.NewRGB(100, 200, 255), 20),
					).Padding(2),
					tui.BorderAnimationPresets.Pulsing(tui.NewRGB(255, 200, 100), 15),
				).Title("Pulse")
			},
		},
		{
			name:        "Text: Sparkle",
			description: "Random sparkles on text",
			view: func() tui.View {
				return tui.AnimatedBordered(
					tui.Stack(
						tui.Text("Sparkling text effect").Bold().Sparkle(3, tui.NewRGB(200, 200, 255), tui.NewRGB(255, 255, 255)),
						tui.Spacer().MinHeight(1),
						tui.Text("Random characters flash bright").Sparkle(2, tui.NewRGB(255, 200, 100), tui.NewRGB(255, 255, 200)),
					).Padding(2),
					&tui.SparkleBorderAnimation{
						Speed:      2,
						BaseColor:  tui.NewRGB(100, 100, 150),
						SparkColor: tui.NewRGB(255, 255, 255),
						Density:    3,
					},
				).Title("Sparkle")
			},
		},
		{
			name:        "Text: Glitch",
			description: "Digital glitch effect",
			view: func() tui.View {
				return tui.AnimatedBordered(
					tui.Stack(
						tui.Text("GLITCH EFFECT").Bold().Glitch(2, tui.NewRGB(0, 255, 0), tui.NewRGB(255, 0, 255)),
						tui.Spacer().MinHeight(1),
						tui.Text("Digital corruption style").Glitch(3, tui.NewRGB(255, 100, 100), tui.NewRGB(100, 100, 255)),
					).Padding(2),
					tui.BorderAnimationPresets.Fire(2),
				).Title("Glitch")
			},
		},
		{
			name:        "Text: Wave",
			description: "Sine wave color effect",
			view: func() tui.View {
				return tui.AnimatedBordered(
					tui.Stack(
						tui.Text("Wavy text animation").Bold().Wave(8),
						tui.Spacer().MinHeight(1),
						tui.Text("Colors flow like a wave").Wave(12),
					).Padding(2),
					&tui.WaveBorderAnimation{
						Speed:     3,
						WaveWidth: 8,
						Colors: []tui.RGB{
							tui.NewRGB(255, 0, 100),
							tui.NewRGB(0, 255, 100),
							tui.NewRGB(100, 0, 255),
						},
					},
				).Title("Wave")
			},
		},
		{
			name:        "Combined Effects",
			description: "Multiple animations together",
			view: func() tui.View {
				return tui.AnimatedBordered(
					tui.Stack(
						tui.Text("Ultimate Combo!").Bold().Rainbow(2),
						tui.Spacer().MinHeight(1),
						tui.Text("Rainbow text").Rainbow(3),
						tui.Text("Pulsing text").Pulse(tui.NewRGB(255, 200, 100), 15),
						tui.Text("Sparkling text").Sparkle(3, tui.NewRGB(200, 200, 255), tui.NewRGB(255, 255, 255)),
						tui.Text("Glitchy text").Glitch(2, tui.NewRGB(100, 255, 200), tui.NewRGB(255, 100, 255)),
						tui.Text("Wavy text").Wave(10),
					).Padding(2),
					tui.BorderAnimationPresets.Rainbow(2, false),
				).Title("Kitchen Sink")
			},
		},
	}

	return nil
}

func (app *AnimationShowcaseApp) demoPanel(content string) tui.View {
	return tui.Stack(
		tui.Text("%s", content).FgRGB(255, 255, 255),
	).Padding(2)
}

func (app *AnimationShowcaseApp) View() tui.View {
	if app.currentEffect >= len(app.effects) {
		app.currentEffect = 0
	}

	effect := app.effects[app.currentEffect]

	return tui.Stack(
		tui.Text("Animation Showcase").Bold().Rainbow(2),

		tui.Spacer().MinHeight(1),

		// Effect info
		tui.Stack(
			tui.Group(
				tui.Text("Effect:").Bold().Fg(tui.ColorYellow),
				tui.Spacer(),
				tui.Text("%s", effect.name).Fg(tui.ColorWhite),
			),
			tui.Group(
				tui.Text("Description:").Bold().Fg(tui.ColorYellow),
				tui.Spacer(),
				tui.Text("%s", effect.description).Fg(tui.ColorBrightBlack),
			),
		).Gap(1),

		tui.Spacer().MinHeight(2),

		// Current effect
		tui.Stack(
			effect.view(),
		),

		tui.Spacer(),

		// Navigation
		tui.Stack(
			tui.Text("─────────────────────────────────────────────────────────────────").Fg(tui.ColorBrightBlack),
			tui.Group(
				tui.Text("[←] Previous").Fg(tui.ColorGreen),
				tui.Spacer(),
				tui.Text("[→] Next").Fg(tui.ColorGreen),
				tui.Spacer(),
				tui.Text("[q] Quit").Fg(tui.ColorRed),
			),
			tui.Text("─────────────────────────────────────────────────────────────────").Fg(tui.ColorBrightBlack),

			tui.Spacer().MinHeight(1),

			tui.Group(
				tui.Text("Effect").Fg(tui.ColorBrightBlack),
				tui.Spacer(),
				tui.Text("%d / %d", app.currentEffect+1, len(app.effects)).Fg(tui.ColorWhite),
			),
		),
	).Padding(2)
}

func (app *AnimationShowcaseApp) HandleEvent(event tui.Event) []tui.Cmd {
	switch e := event.(type) {
	case tui.KeyEvent:
		if e.Rune == 'q' || e.Rune == 'Q' || e.Key == tui.KeyEscape || e.Key == tui.KeyCtrlC {
			return []tui.Cmd{tui.Quit()}
		}
		if e.Key == tui.KeyArrowRight || e.Rune == 'n' {
			app.currentEffect = (app.currentEffect + 1) % len(app.effects)
		}
		if e.Key == tui.KeyArrowLeft || e.Rune == 'p' {
			app.currentEffect--
			if app.currentEffect < 0 {
				app.currentEffect = len(app.effects) - 1
			}
		}
	}

	return nil
}

func main() {
	app := &AnimationShowcaseApp{}
	app.Init()

	if err := tui.Run(app, tui.WithFPS(30)); err != nil {
		log.Fatal(err)
	}
}
