// Package main demonstrates animation with the declarative API.
// Compare with examples/runtime_animation which uses the imperative API.
package main

import (
	"fmt"
	"image"
	"log"

	"github.com/deepnoodle-ai/gooey"
)

type App struct {
	frame     uint64
	positions []int
	width     int
}

func (app *App) View() gooey.View {
	return gooey.VStack(
		gooey.Text("Animated Blocks Demo (60 FPS)").Bold().Fg(gooey.ColorCyan),
		gooey.Spacer().MinHeight(1),
		// Use Canvas for custom animation drawing
		gooey.Canvas(func(frame gooey.RenderFrame, bounds image.Rectangle) {
			app.drawBlocks(frame, bounds)
		}),
		gooey.HStack(
			gooey.Text("Press 'q' to quit").Fg(gooey.ColorYellow),
			gooey.Spacer(),
			gooey.Text("Frame: %d", app.frame).Fg(gooey.ColorGreen),
		),
	).Padding(1)
}

func (app *App) drawBlocks(frame gooey.RenderFrame, bounds image.Rectangle) {
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
			trailStyle := style.WithFgRGB(gooey.NewRGB(
				style.FgRGB.R/uint8(trail+1),
				style.FgRGB.G/uint8(trail+1),
				style.FgRGB.B/uint8(trail+1),
			))
			frame.SetCell(trailX, y, '▒', trailStyle)
		}
	}
}

func (app *App) rainbowStyle(blockIndex int) gooey.Style {
	colors := gooey.SmoothRainbow(60)
	offset := int(app.frame) % 60
	colorIndex := (blockIndex*10 + offset) % len(colors)
	return gooey.NewStyle().WithFgRGB(colors[colorIndex]).WithBold()
}

func (app *App) HandleEvent(event gooey.Event) []gooey.Cmd {
	switch e := event.(type) {
	case gooey.TickEvent:
		app.frame++
		for i := range app.positions {
			speed := uint64(i+1) * 2
			offset := uint64(i * 15)
			app.positions[i] = int((app.frame*speed + offset) % uint64(app.width))
		}

	case gooey.KeyEvent:
		if e.Rune == 'q' || e.Key == gooey.KeyCtrlC {
			return []gooey.Cmd{gooey.Quit()}
		}

	case gooey.ResizeEvent:
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

	if err := gooey.Run(app, gooey.WithFPS(60)); err != nil {
		log.Fatal(err)
	}
}
