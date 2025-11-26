package main

import (
	"fmt"
	"log"

	"github.com/deepnoodle-ai/gooey"
)

// AnimatedApp demonstrates smooth animation using the Runtime and TickEvents.
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
func (app *AnimatedApp) HandleEvent(event gooey.Event) []gooey.Cmd {
	switch e := event.(type) {
	case gooey.TickEvent:
		// Update animation state on each tick
		app.frame++

		// Update positions for each animated block
		// Each block moves at a different speed and offset
		for i := range app.positions {
			speed := uint64(i+1) * 2 // Different speeds for visual interest
			offset := uint64(i * 15) // Stagger the starting positions
			app.positions[i] = int((app.frame*speed + offset) % uint64(app.width))
		}

	case gooey.KeyEvent:
		// Handle user input
		if e.Rune == 'q' || e.Rune == 'Q' || e.Key == gooey.KeyEscape || e.Key == gooey.KeyCtrlC {
			return []gooey.Cmd{gooey.Quit()}
		}

	case gooey.ResizeEvent:
		// Update canvas size on terminal resize
		app.width = e.Width
		app.height = e.Height
	}

	return nil
}

// Render draws the current animation frame.
// This is called automatically after each HandleEvent by the Runtime.
func (app *AnimatedApp) Render(frame gooey.RenderFrame) {
	// Clear background and draw header
	width, height := frame.Size()

	// Draw title
	titleStyle := gooey.NewStyle().WithBold().WithForeground(gooey.ColorCyan)
	frame.PrintStyled(0, 0, "Animated Blocks Demo (60 FPS)", titleStyle)

	// Draw animated blocks
	// We'll draw multiple blocks at different vertical positions
	numBlocks := 5
	blockHeight := (height - 4) / numBlocks

	for blockIdx := 0; blockIdx < numBlocks; blockIdx++ {
		y := 2 + blockIdx*blockHeight
		if y >= height-1 {
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
			frame.SetCell(x, y, blockChar, blockStyle)

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
				trailStyle = trailStyle.WithFgRGB(gooey.NewRGB(
					trailStyle.FgRGB.R/uint8(trail+1),
					trailStyle.FgRGB.G/uint8(trail+1),
					trailStyle.FgRGB.B/uint8(trail+1),
				))
				frame.SetCell(trailX, y, '▒', trailStyle)
			}
		}
	}

	// Draw info text at the bottom
	infoY := height - 1
	infoStyle := gooey.NewStyle().WithForeground(gooey.ColorYellow)
	frame.PrintStyled(0, infoY, "Press 'q' to quit", infoStyle)

	// Display frame count
	frameCounterStyle := gooey.NewStyle().WithForeground(gooey.ColorGreen)
	frameCountStr := fmt.Sprintf("Frame: %d", app.frame)
	frame.PrintStyled(width-len(frameCountStr), infoY, frameCountStr, frameCounterStyle)
}

// rainbowStyle generates a rainbow color based on frame and block index.
// This creates a smooth color cycling effect across the blocks.
func rainbowStyle(frame uint64, blockIndex int) gooey.Style {
	// Create a smooth rainbow gradient with enough colors for smooth animation
	rainbowLength := 60
	colors := gooey.SmoothRainbow(rainbowLength)

	// Calculate color index based on frame and block index
	// The offset ensures each block has a different color
	offset := int(frame) % rainbowLength
	colorIndex := (blockIndex*10 + offset) % len(colors)
	rgb := colors[colorIndex]

	return gooey.NewStyle().WithFgRGB(rgb).WithBold()
}

func main() {
	app := &AnimatedApp{
		frame:     0,
		positions: make([]int, 5), // 5 animated blocks
	}

	if err := gooey.Run(app, gooey.WithFPS(60)); err != nil {
		log.Fatal(err)
	}
}
