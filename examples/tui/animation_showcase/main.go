package main

import (
	"log"

	"github.com/deepnoodle-ai/wonton/tui"
)

// AnimationShowcaseApp demonstrates using animation presets and builders.
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
			name:        "Preset: Fade In",
			description: "Smooth fade in with EaseInQuad",
			view: func() tui.View {
				anim := tui.AnimationPresets.FadeIn(120)
				anim.Start(0)
				return tui.Fade(
					app.demoPanel("Fading in smoothly"),
					anim,
					0.0,
					1.0,
				)
			},
		},
		{
			name:        "Preset: Pulse",
			description: "Continuous pulsing effect",
			view: func() tui.View {
				anim := tui.AnimationPresets.Pulse(80)
				anim.Start(0)
				return tui.Brightness(
					app.demoPanel("Pulsing brightness"),
					anim,
					0.6,
					1.4,
				)
			},
		},
		{
			name:        "Preset: Bounce",
			description: "Bouncing animation",
			view: func() tui.View {
				anim := tui.AnimationPresets.Bounce(90)
				anim.Start(0)
				return tui.Brightness(
					app.demoPanel("Bouncing!"),
					anim,
					0.5,
					1.5,
				)
			},
		},
		{
			name:        "Preset: Elastic",
			description: "Elastic spring effect",
			view: func() tui.View {
				anim := tui.AnimationPresets.Elastic(100)
				anim.Start(0)
				return tui.Brightness(
					app.demoPanel("Elastic spring"),
					anim,
					0.7,
					1.3,
				)
			},
		},
		{
			name:        "Preset: Attention",
			description: "Attention-grabbing effect",
			view: func() tui.View {
				anim := tui.AnimationPresets.Attention(60)
				anim.Start(0)
				return tui.Brightness(
					app.demoPanel("Look at me!"),
					anim,
					0.8,
					1.2,
				)
			},
		},
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
			name:        "Complex: Multiple Effects",
			description: "Combining fade, brightness, and animated border",
			view: func() tui.View {
				fadeAnim := tui.AnimationPresets.Pulse(100)
				fadeAnim.Start(0)

				brightAnim := tui.AnimationPresets.Pulse(70)
				brightAnim.Start(0)

				chain := tui.NewViewAnimationChain().
					WithAnimatedBorder(tui.BorderAnimationPresets.Rainbow(3, false)).
					WithFade(fadeAnim, 0.7, 1.0).
					WithBrightness(brightAnim, 0.9, 1.1)

				return chain.Apply(app.demoPanel("Multiple effects\ncombined together!"))
			},
		},
		{
			name:        "Wave: Horizontal",
			description: "Brightness wave moving left to right",
			view: func() tui.View {
				return tui.GlobalWave(
					app.demoPanel("Horizontal wave\nwashing over"),
					0.2,
					0.1,
					5,
					tui.WaveHorizontal,
				)
			},
		},
		{
			name:        "Wave: Radial",
			description: "Wave emanating from center",
			view: func() tui.View {
				return tui.GlobalWave(
					app.demoPanel("Radial wave\nfrom center"),
					0.15,
					0.008,
					8,
					tui.WaveRadial,
				)
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
			name:        "Everything Combined",
			description: "The kitchen sink of animations",
			view: func() tui.View {
				pulseAnim := tui.AnimationPresets.Pulse(60)
				pulseAnim.Start(0)

				return tui.GlobalWave(
					tui.Brightness(
						tui.AnimatedBordered(
							tui.Stack(
								tui.Text("Ultimate Combo!").Bold().Rainbow(2),
								tui.Spacer().MinHeight(1),
								tui.Text("Rainbow border").Pulse(tui.NewRGB(255, 200, 100), 15),
								tui.Text("Global wave effect").Sparkle(3, tui.NewRGB(200, 200, 255), tui.NewRGB(255, 255, 255)),
								tui.Text("Brightness pulse").Glitch(2, tui.NewRGB(100, 255, 200), tui.NewRGB(255, 100, 255)),
								tui.Text("All at once!").Wave(10),
							).Padding(2),
							tui.BorderAnimationPresets.Rainbow(2, false),
						).Title("Kitchen Sink"),
						pulseAnim,
						0.85,
						1.15,
					),
					0.12,
					0.08,
					7,
					tui.WaveRadial,
				)
			},
		},
	}

	return nil
}

func (app *AnimationShowcaseApp) demoPanel(content string) tui.View {
	return tui.Panel(
		tui.Stack(
			tui.Text("%s", content).Fg(tui.ColorWhite),
		).Padding(2),
	).Border(tui.BorderSingle).BorderColor(tui.ColorCyan).Size(50, 10)
}

func (app *AnimationShowcaseApp) View() tui.View {
	if app.currentEffect >= len(app.effects) {
		app.currentEffect = 0
	}

	effect := app.effects[app.currentEffect]

	return tui.Stack(
		tui.Text("Animation Showcase").Bold().Fg(tui.ColorCyan).Rainbow(2),

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
				tui.Text("[space] Play Again").Fg(tui.ColorYellow),
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
		if e.Rune == ' ' {
			// Reinitialize current effect (restart animation)
			app.Init()
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
