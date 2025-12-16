package main

import (
	"log"

	"github.com/deepnoodle-ai/wonton/tui"
)

// AdvancedAnimationsApp demonstrates the advanced animation system.
type AdvancedAnimationsApp struct {
	controller *tui.AnimationController
	pulseAnim  *tui.Animation
	fadeAnim   *tui.Animation
	bounceAnim *tui.Animation
}

func (app *AdvancedAnimationsApp) Init() []tui.Cmd {
	// Initialize animation controller
	app.controller = tui.NewAnimationController()

	// Create various animations with different easing functions
	app.pulseAnim = tui.NewAnimation(60).
		WithEasing(tui.EaseInOutSine).
		WithLoop(true).
		WithPingPong(true)
	app.pulseAnim.Start(0)

	app.fadeAnim = tui.NewAnimation(120).
		WithEasing(tui.EaseInOutQuad).
		WithLoop(true).
		WithPingPong(true)
	app.fadeAnim.Start(0)

	app.bounceAnim = tui.NewAnimation(90).
		WithEasing(tui.EaseOutBounce).
		WithLoop(true)
	app.bounceAnim.Start(0)

	// Register animations with controller
	app.controller.Register("pulse", app.pulseAnim)
	app.controller.Register("fade", app.fadeAnim)
	app.controller.Register("bounce", app.bounceAnim)

	// Register global effects
	globalPulse := tui.NewGlobalEffect(tui.GlobalEffectPulse, 80).
		WithEasing(tui.EaseInOutSine)
	globalPulse.Start(0)
	tui.GlobalAnimationCoordinator.RegisterEffect("global-pulse", globalPulse)

	return nil
}

func (app *AdvancedAnimationsApp) View() tui.View {
	// Update animations
	app.controller.Update(0) // Frame will be updated by context

	return tui.Stack(
		// Title with rainbow animation
		tui.Text("Advanced Animation System Demo").Bold().Rainbow(2),

		tui.Spacer().MinHeight(1),

		// Section 1: Animated Borders
		tui.Text("1. Animated Borders").Bold().Fg(tui.ColorCyan),
		tui.Spacer().MinHeight(1),

		tui.Group(
			// Rainbow border
			tui.AnimatedBordered(
				tui.Stack(
					tui.Text("Rainbow Border").Bold(),
					tui.Text("Cycles through colors"),
				).Padding(1),
				&tui.RainbowBorderAnimation{Speed: 2, Reversed: false},
			).Title("Rainbow"),

			tui.Spacer(),

			// Pulsing border
			tui.AnimatedBordered(
				tui.Stack(
					tui.Text("Pulsing Border").Bold(),
					tui.Text("Smooth brightness pulse"),
				).Padding(1),
				&tui.PulseBorderAnimation{
					Speed:         15,
					Color:         tui.NewRGB(0, 200, 255),
					MinBrightness: 0.3,
					MaxBrightness: 1.0,
					Easing:        tui.EaseInOutSine,
				},
			).Title("Pulse"),

			tui.Spacer(),

			// Marquee border
			tui.AnimatedBordered(
				tui.Stack(
					tui.Text("Marquee Border").Bold(),
					tui.Text("Classic marquee effect"),
				).Padding(1),
				&tui.MarqueeBorderAnimation{
					Speed:         3,
					OnColor:       tui.NewRGB(255, 255, 0),
					OffColor:      tui.NewRGB(50, 50, 0),
					SegmentLength: 2,
				},
			).Title("Marquee"),
		),

		tui.Spacer().MinHeight(1),

		// More animated borders
		tui.Group(
			// Wave border
			tui.AnimatedBordered(
				tui.Stack(
					tui.Text("Wave Border").Bold(),
					tui.Text("Flowing colors"),
				).Padding(1),
				&tui.WaveBorderAnimation{
					Speed:     3,
					WaveWidth: 8,
					Colors: []tui.RGB{
						tui.NewRGB(255, 0, 100),
						tui.NewRGB(0, 255, 100),
						tui.NewRGB(100, 0, 255),
					},
				},
			).Title("Wave"),

			tui.Spacer(),

			// Sparkle border
			tui.AnimatedBordered(
				tui.Stack(
					tui.Text("Sparkle Border").Bold(),
					tui.Text("Random sparkles"),
				).Padding(1),
				&tui.SparkleBorderAnimation{
					Speed:      2,
					BaseColor:  tui.NewRGB(100, 100, 150),
					SparkColor: tui.NewRGB(255, 255, 255),
					Density:    3,
				},
			).Title("Sparkle"),

			tui.Spacer(),

			// Fire border
			tui.AnimatedBordered(
				tui.Stack(
					tui.Text("Fire Border").Bold(),
					tui.Text("Flickering flames"),
				).Padding(1),
				tui.BorderAnimationPresets.Fire(1),
			).Title("Fire"),
		),

		tui.Spacer().MinHeight(1),

		// Section 2: View Animations
		tui.Text("2. View Brightness & Fade Animations").Bold().Fg(tui.ColorCyan),
		tui.Spacer().MinHeight(1),

		tui.Group(
			// Fade animation
			tui.Fade(
				tui.Panel(
					tui.Stack(
						tui.Text("Fading Panel").Bold(),
						tui.Text("Opacity changes"),
					).Padding(1),
				).Border(tui.BorderSingle).BorderColor(tui.ColorGreen).Size(25, 5),
				app.fadeAnim,
				0.3,
				1.0,
			),

			tui.Spacer(),

			// Brightness animation
			tui.Brightness(
				tui.Panel(
					tui.Stack(
						tui.Text("Brightness Panel").Bold(),
						tui.Text("Brightness pulse"),
					).Padding(1),
				).Border(tui.BorderSingle).BorderColor(tui.ColorYellow).Size(25, 5),
				app.pulseAnim,
				0.5,
				1.3,
			),

			tui.Spacer(),

			// Brightness with color tint
			func() tui.View {
				bright := tui.Brightness(
					tui.Panel(
						tui.Stack(
							tui.Text("Tinted Brightness").Bold(),
							tui.Text("With color tint"),
						).Padding(1),
					).Border(tui.BorderSingle).BorderColor(tui.ColorMagenta).Size(25, 5),
					app.bounceAnim,
					0.7,
					1.4,
				)
				return bright.WithColorTint(tui.NewRGB(255, 100, 200))
			}(),
		),

		tui.Spacer().MinHeight(1),

		// Section 3: Global Wave Effects
		tui.Text("3. Global Wave Effects").Bold().Fg(tui.ColorCyan),
		tui.Spacer().MinHeight(1),

		tui.Group(
			// Horizontal wave
			tui.GlobalWave(
				tui.Panel(
					tui.Stack(
						tui.Text("Horizontal Wave").Bold(),
						tui.Text("Left to right"),
					).Padding(1),
				).Border(tui.BorderSingle).Bg(tui.ColorBlue).Size(22, 5),
				0.2,
				0.15,
				5,
				tui.WaveHorizontal,
			),

			tui.Spacer(),

			// Vertical wave
			tui.GlobalWave(
				tui.Panel(
					tui.Stack(
						tui.Text("Vertical Wave").Bold(),
						tui.Text("Top to bottom"),
					).Padding(1),
				).Border(tui.BorderSingle).Bg(tui.ColorBlue).Size(22, 5),
				0.2,
				0.15,
				5,
				tui.WaveVertical,
			),

			tui.Spacer(),

			// Diagonal wave
			tui.GlobalWave(
				tui.Panel(
					tui.Stack(
						tui.Text("Diagonal Wave").Bold(),
						tui.Text("Corner to corner"),
					).Padding(1),
				).Border(tui.BorderSingle).Bg(tui.ColorBlue).Size(22, 5),
				0.2,
				0.15,
				5,
				tui.WaveDiagonal,
			),

			tui.Spacer(),

			// Radial wave
			tui.GlobalWave(
				tui.Panel(
					tui.Stack(
						tui.Text("Radial Wave").Bold(),
						tui.Text("From center"),
					).Padding(1),
				).Border(tui.BorderSingle).Bg(tui.ColorBlue).Size(22, 5),
				0.15,
				0.01,
				8,
				tui.WaveRadial,
			),
		),

		tui.Spacer().MinHeight(1),

		// Section 4: Composable Animations
		tui.Text("4. Composable Animation Chain").Bold().Fg(tui.ColorCyan),
		tui.Spacer().MinHeight(1),

		// Complex chained animation
		func() tui.View {
			chain := tui.NewViewAnimationChain().
				WithAnimatedBorder(tui.BorderAnimationPresets.Rainbow(2, false)).
				WithBrightness(app.pulseAnim, 0.8, 1.2).
				WithGlobalWave(0.15, 0.1, 6, tui.WaveHorizontal)

			return chain.Apply(
				tui.Panel(
					tui.Stack(
						tui.Text("Chained Effects").Bold(),
						tui.Text("Rainbow + Pulse + Wave"),
						tui.Text("All combined together!"),
					).Padding(1),
				).Size(40, 7),
			)
		}(),

		tui.Spacer().MinHeight(1),

		// Section 5: Different Easing Functions
		tui.Text("5. Easing Function Showcase").Bold().Fg(tui.ColorCyan),
		tui.Spacer().MinHeight(1),

		tui.Group(
			app.easingDemo("Linear", tui.EaseLinear),
			tui.Spacer(),
			app.easingDemo("Quad", tui.EaseInOutQuad),
			tui.Spacer(),
			app.easingDemo("Cubic", tui.EaseInOutCubic),
		),

		tui.Spacer().MinHeight(1),

		tui.Group(
			app.easingDemo("Sine", tui.EaseInOutSine),
			tui.Spacer(),
			app.easingDemo("Elastic", tui.EaseInOutElastic),
			tui.Spacer(),
			app.easingDemo("Bounce", tui.EaseInOutBounce),
		),

		tui.Spacer(),

		// Instructions
		tui.Text("[q] quit  •  [r] reset animations").Fg(tui.ColorBrightBlack),
	).Padding(1)
}

func (app *AdvancedAnimationsApp) easingDemo(name string, easing tui.Easing) tui.View {
	anim := tui.NewAnimation(90).WithEasing(easing).WithLoop(true).WithPingPong(true)
	anim.Start(0)

	return tui.Brightness(
		tui.Panel(
			tui.Text("%s", name).Bold(),
		).Border(tui.BorderSingle).Size(18, 3),
		anim,
		0.4,
		1.2,
	)
}

func (app *AdvancedAnimationsApp) HandleEvent(event tui.Event) []tui.Cmd {
	switch e := event.(type) {
	case tui.KeyEvent:
		if e.Rune == 'q' || e.Rune == 'Q' || e.Key == tui.KeyEscape || e.Key == tui.KeyCtrlC {
			return []tui.Cmd{tui.Quit()}
		}
		if e.Rune == 'r' || e.Rune == 'R' {
			// Reset all animations
			app.Init()
		}
	case tui.TickEvent:
		// Update animations on each tick
		app.controller.Update(uint64(e.Frame))
		tui.GlobalAnimationCoordinator.Update(uint64(e.Frame))

		app.pulseAnim.Update(uint64(e.Frame))
		app.fadeAnim.Update(uint64(e.Frame))
		app.bounceAnim.Update(uint64(e.Frame))
	}

	return nil
}

func main() {
	app := &AdvancedAnimationsApp{}
	app.Init()

	if err := tui.Run(app, tui.WithFPS(30)); err != nil {
		log.Fatal(err)
	}
}
