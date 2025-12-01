package main

import (
	"image"
	"log"

	"github.com/deepnoodle-ai/gooey/tui"
)

// AnimatedApp demonstrates smooth animation using the declarative View system.
// It renders animated blocks moving across the screen with rainbow color cycling.
type AnimatedApp struct {
	frame     uint64
	positions []int
	width     int
	height    int
}

// HandleEvent processes events from the Runtime.
// For this demo, we handle:
// - TickEvent: Update animation state
// - KeyEvent: Handle user input (q to quit)
// - ResizeEvent: Handle terminal resize
func (app *AnimatedApp) HandleEvent(event tui.Event) []tui.Cmd {
	switch e := event.(type) {
	case tui.TickEvent:
		// Update animation state on each tick
		app.frame++

		// Update positions for each animated block
		// Each block moves at a different speed and offset
		for i := range app.positions {
			speed := uint64(i+1) * 2 // Different speeds for visual interest
			offset := uint64(i * 15) // Stagger the starting positions
			app.positions[i] = int((app.frame*speed + offset) % uint64(app.width))
		}

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

		// Animation canvas
		tui.Canvas(func(frame tui.RenderFrame, bounds image.Rectangle) {
			width := bounds.Dx()
			height := bounds.Dy()

			// Draw animated blocks
			// We'll draw multiple blocks at different vertical positions
			numBlocks := 5
			blockHeight := height / numBlocks

			for blockIdx := 0; blockIdx < numBlocks; blockIdx++ {
				y := blockIdx * blockHeight
				if y >= height {
					break
				}

				// Get the position for this block
				if blockIdx < len(app.positions) {
					x := app.positions[blockIdx]

					// Ensure position is within bounds
					if x < 0 || x >= width {
						continue
					}

					// Draw a solid block (█)
					blockChar := '█'
					blockStyle := rainbowStyle(app.frame, blockIdx)
					frame.SetCell(bounds.Min.X+x, bounds.Min.Y+y, blockChar, blockStyle)

					// Draw a trail of semi-filled blocks for visual effect
					trailLength := 4
					for trail := 1; trail <= trailLength; trail++ {
						trailX := x - trail
						if trailX < 0 {
							break
						}
						// Avoid uint64 underflow when frame < trail
						trailFrame := app.frame
						if trailFrame >= uint64(trail) {
							trailFrame -= uint64(trail)
						}
						trailStyle := rainbowStyle(trailFrame, blockIdx)
						trailStyle = trailStyle.WithFgRGB(tui.NewRGB(
							trailStyle.FgRGB.R/uint8(trail+1),
							trailStyle.FgRGB.G/uint8(trail+1),
							trailStyle.FgRGB.B/uint8(trail+1),
						))
						frame.SetCell(bounds.Min.X+trailX, bounds.Min.Y+y, '▒', trailStyle)
					}
				}
			}
		}),

		// Footer with controls and frame counter
		tui.HStack(
			tui.Text("Press 'q' to quit").Fg(tui.ColorYellow),
			tui.Spacer(),
			tui.Text("Frame: %d", app.frame).Fg(tui.ColorGreen),
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
	app := &AnimatedApp{
		frame:     0,
		positions: make([]int, 5), // 5 animated blocks
	}

	if err := tui.Run(app, tui.WithFPS(60)); err != nil {
		log.Fatal(err)
	}
}
