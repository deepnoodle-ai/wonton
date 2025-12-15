package main

import (
	"log"

	"github.com/deepnoodle-ai/wonton/tui"
)

// AnimatedApp demonstrates smooth animation using the declarative View system.
// It renders animated blocks moving across the screen with rainbow color cycling.
// This example uses CanvasContext to access the animation frame counter.
type AnimatedApp struct {
	width  int
	height int
}

// HandleEvent processes events from the Runtime.
// For this demo, we handle:
// - KeyEvent: Handle user input (q to quit)
// - ResizeEvent: Handle terminal resize
func (app *AnimatedApp) HandleEvent(event tui.Event) []tui.Cmd {
	switch e := event.(type) {
	case tui.KeyEvent:
		// Handle user input
		if e.Rune == 'q' || e.Rune == 'Q' || e.Key == tui.KeyEscape || e.Key == tui.KeyCtrlC {
			return []tui.Cmd{tui.Quit()}
		}

	case tui.ResizeEvent:
		// Update canvas size on terminal resize
		app.width = e.Width
		app.height = e.Height
	}

	return nil
}

// View returns the declarative view structure.
// This is called automatically by the Runtime to render the UI.
func (app *AnimatedApp) View() tui.View {
	return tui.VStack(
		// Title
		tui.Text("Animated Blocks Demo (60 FPS)").Bold().Fg(tui.ColorCyan),

		tui.Spacer(),

		// Animation canvas using CanvasContext for frame counter access
		tui.CanvasContext(func(ctx *tui.RenderContext) {
			width, height := ctx.Size()
			frame := ctx.Frame() // Get animation frame from context

			// Draw animated blocks
			numBlocks := 5
			blockHeight := height / numBlocks

			for blockIdx := 0; blockIdx < numBlocks; blockIdx++ {
				y := blockIdx * blockHeight
				if y >= height {
					break
				}

				// Calculate position from frame counter
				speed := uint64(blockIdx+1) * 2
				offset := uint64(blockIdx * 15)
				x := int((frame*speed + offset) % uint64(width))

				// Ensure position is within bounds
				if x < 0 || x >= width {
					continue
				}

				// Draw a solid block (█)
				blockChar := '█'
				blockStyle := rainbowStyle(frame, blockIdx)
				ctx.SetCell(x, y, blockChar, blockStyle)

				// Draw a trail of semi-filled blocks for visual effect
				trailLength := 4
				for trail := 1; trail <= trailLength; trail++ {
					trailX := x - trail
					if trailX < 0 {
						break
					}
					// Avoid uint64 underflow when frame < trail
					trailFrame := frame
					if trailFrame >= uint64(trail) {
						trailFrame -= uint64(trail)
					}
					trailStyle := rainbowStyle(trailFrame, blockIdx)
					trailStyle = trailStyle.WithFgRGB(tui.NewRGB(
						trailStyle.FgRGB.R/uint8(trail+1),
						trailStyle.FgRGB.G/uint8(trail+1),
						trailStyle.FgRGB.B/uint8(trail+1),
					))
					ctx.SetCell(trailX, y, '▒', trailStyle)
				}
			}
		}),

		// Footer with controls
		tui.HStack(
			tui.Text("Press 'q' to quit").Fg(tui.ColorYellow),
			tui.Spacer(),
			tui.Text("Using ctx.Frame() for animation").Info(),
		),
	)
}

// rainbowStyle generates a rainbow color based on frame and block index.
// This creates a smooth color cycling effect across the blocks.
func rainbowStyle(frame uint64, blockIndex int) tui.Style {
	// Create a smooth rainbow gradient with enough colors for smooth animation
	rainbowLength := 60
	colors := tui.SmoothRainbow(rainbowLength)

	// Calculate color index based on frame and block index
	// The offset ensures each block has a different color
	offset := int(frame) % rainbowLength
	colorIndex := (blockIndex*10 + offset) % len(colors)
	rgb := colors[colorIndex]

	return tui.NewStyle().WithFgRGB(rgb).WithBold()
}

func main() {
	app := &AnimatedApp{}

	if err := tui.Run(app, tui.WithFPS(60)); err != nil {
		log.Fatal(err)
	}
}
