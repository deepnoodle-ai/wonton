// Package main demonstrates animation with the declarative API.
// Compare with examples/runtime_animation which uses the imperative API.
package main

import (
	"log"

	"github.com/deepnoodle-ai/wonton/tui"
)

type App struct{}

func (app *App) View() tui.View {
	return tui.Stack(
		tui.Text("Animated Blocks Demo (60 FPS)").Bold().Fg(tui.ColorCyan),
		tui.Spacer().MinHeight(1),
		// Use CanvasContext to access the animation frame counter
		tui.CanvasContext(func(ctx *tui.RenderContext) {
			app.drawBlocks(ctx)
		}),
		tui.Spacer().MinHeight(1),
		tui.Group(
			tui.Text("Press 'q' to quit").Fg(tui.ColorYellow),
			tui.Spacer(),
			tui.Text("CanvasContext demo").Info(),
		),
	).Padding(1)
}

func (app *App) drawBlocks(ctx *tui.RenderContext) {
	width, height := ctx.Size()
	frame := ctx.Frame() // Animation frame counter from RenderContext
	numBlocks := 5
	blockHeight := height / numBlocks

	for i := 0; i < numBlocks; i++ {
		y := i * blockHeight
		if y >= height {
			break
		}

		// Calculate position directly from frame counter
		speed := uint64(i+1) * 2
		offset := uint64(i * 15)
		x := int((frame*speed + offset) % uint64(width))

		if x < 0 || x >= width {
			continue
		}

		// Draw block with rainbow color
		style := rainbowStyle(frame, i)
		ctx.SetCell(x, y, '█', style)

		// Draw trail
		for trail := 1; trail <= 4; trail++ {
			trailX := x - trail
			if trailX < 0 {
				break
			}
			trailFrame := frame
			if trailFrame >= uint64(trail) {
				trailFrame -= uint64(trail)
			}
			trailStyle := rainbowStyle(trailFrame, i)
			trailStyle = trailStyle.WithFgRGB(tui.NewRGB(
				trailStyle.FgRGB.R/uint8(trail+1),
				trailStyle.FgRGB.G/uint8(trail+1),
				trailStyle.FgRGB.B/uint8(trail+1),
			))
			ctx.SetCell(trailX, y, '▒', trailStyle)
		}
	}
}

func rainbowStyle(frame uint64, blockIndex int) tui.Style {
	colors := tui.SmoothRainbow(60)
	offset := int(frame) % 60
	colorIndex := (blockIndex*10 + offset) % len(colors)
	return tui.NewStyle().WithFgRGB(colors[colorIndex]).WithBold()
}

func (app *App) HandleEvent(event tui.Event) []tui.Cmd {
	switch e := event.(type) {
	case tui.KeyEvent:
		if e.Rune == 'q' || e.Key == tui.KeyCtrlC {
			return []tui.Cmd{tui.Quit()}
		}
	}
	return nil
}

func main() {
	if err := tui.Run(&App{}, tui.WithFPS(60)); err != nil {
		log.Fatal(err)
	}
}
