// Package main demonstrates animation with the declarative API.
// Compare with examples/runtime_animation which uses the imperative API.
package main

import (
	"fmt"
	"image"
	"log"

	"github.com/deepnoodle-ai/gooey/tui"
)

type App struct {
	frame     uint64
	positions []int
	width     int
}

func (app *App) View() tui.View {
	return tui.VStack(
		tui.Text("Animated Blocks Demo (60 FPS)").Bold().Fg(tui.ColorCyan),
		tui.Spacer().MinHeight(1),
		// Use Canvas for custom animation drawing
		tui.Canvas(func(frame tui.RenderFrame, bounds image.Rectangle) {
			app.drawBlocks(frame, bounds)
		}),
		tui.HStack(
			tui.Text("Press 'q' to quit").Fg(tui.ColorYellow),
			tui.Spacer(),
			tui.Text("Frame: %d", app.frame).Fg(tui.ColorGreen),
		),
	).Padding(1)
}

func (app *App) drawBlocks(frame tui.RenderFrame, bounds image.Rectangle) {
	width, height := bounds.Dx(), bounds.Dy()
	numBlocks := 5
	blockHeight := height / numBlocks

	for i := 0; i < numBlocks && i < len(app.positions); i++ {
		y := i * blockHeight
		x := app.positions[i]

		if x < 0 || x >= width {
			continue
		}

		// Draw block with rainbow color
		style := app.rainbowStyle(i)
		frame.SetCell(x, y, '█', style)

		// Draw trail
		for trail := 1; trail <= 4; trail++ {
			trailX := x - trail
			if trailX < 0 {
				break
			}
			trailStyle := style.WithFgRGB(tui.NewRGB(
				style.FgRGB.R/uint8(trail+1),
				style.FgRGB.G/uint8(trail+1),
				style.FgRGB.B/uint8(trail+1),
			))
			frame.SetCell(trailX, y, '▒', trailStyle)
		}
	}
}

func (app *App) rainbowStyle(blockIndex int) tui.Style {
	colors := tui.SmoothRainbow(60)
	offset := int(app.frame) % 60
	colorIndex := (blockIndex*10 + offset) % len(colors)
	return tui.NewStyle().WithFgRGB(colors[colorIndex]).WithBold()
}

func (app *App) HandleEvent(event tui.Event) []tui.Cmd {
	switch e := event.(type) {
	case tui.TickEvent:
		app.frame++
		for i := range app.positions {
			speed := uint64(i+1) * 2
			offset := uint64(i * 15)
			app.positions[i] = int((app.frame*speed + offset) % uint64(app.width))
		}

	case tui.KeyEvent:
		if e.Rune == 'q' || e.Key == tui.KeyCtrlC {
			return []tui.Cmd{tui.Quit()}
		}

	case tui.ResizeEvent:
		app.width = e.Width
	}
	return nil
}

func main() {
	app := &App{
		positions: make([]int, 5),
		width:     80,
	}

	// Suppress unused variable warning
	_ = fmt.Sprint

	if err := tui.Run(app, tui.WithFPS(60)); err != nil {
		log.Fatal(err)
	}
}
