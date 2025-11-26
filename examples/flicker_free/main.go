package main

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/deepnoodle-ai/gooey"
)

// FlickerFreeApp demonstrates flicker-free rendering of rapidly updating content.
// It shows how the Runtime's double-buffered rendering prevents visual artifacts
// when updating multiple regions at high frequency.
type FlickerFreeApp struct {
	width          int
	height         int
	quadrantHeight int
	tickCount      int

	// Data for each quadrant
	quadrantData [4][]string

	// Random generation
	chars  []string
	colors []gooey.RGB
}

// Init initializes the application state.
func (app *FlickerFreeApp) Init() error {
	// Initialize random data
	app.chars = []string{"@", "#", "$", "%", "&", "*", "+", "=", "?", "!"}
	app.colors = []gooey.RGB{
		gooey.NewRGB(255, 0, 0),
		gooey.NewRGB(0, 255, 0),
		gooey.NewRGB(0, 0, 255),
		gooey.NewRGB(255, 255, 0),
		gooey.NewRGB(0, 255, 255),
		gooey.NewRGB(255, 0, 255),
	}

	// Initialize empty quadrant data
	for i := range app.quadrantData {
		app.quadrantData[i] = make([]string, 0)
	}

	return nil
}

// HandleEvent processes events from the runtime.
func (app *FlickerFreeApp) HandleEvent(event gooey.Event) []gooey.Cmd {
	switch e := event.(type) {
	case gooey.KeyEvent:
		// Handle keyboard input
		if e.Key == gooey.KeyCtrlC || e.Rune == 'q' || e.Rune == 'Q' {
			return []gooey.Cmd{gooey.Quit()}
		}

	case gooey.ResizeEvent:
		// Update dimensions on resize
		app.width = e.Width
		app.height = e.Height
		app.updateLayout()

	case gooey.TickEvent:
		// Update quadrant data every 3 ticks (~100ms at 30fps, faster than original 10ms but still rapid)
		app.tickCount++
		if app.tickCount%3 == 0 {
			// Randomly update lines in quadrants
			app.updateQuadrant(0)
			app.updateQuadrant(1)
			app.updateQuadrant(2)
			app.updateQuadrant(3)
		}
	}

	return nil
}

// updateLayout recalculates quadrant dimensions based on terminal size.
func (app *FlickerFreeApp) updateLayout() {
	midY := (app.height-3)/2 + 3
	app.quadrantHeight = midY - 4

	// Ensure we have enough space for at least a few lines
	if app.quadrantHeight < 1 {
		app.quadrantHeight = 1
	}

	// Resize quadrant data arrays
	for i := range app.quadrantData {
		if len(app.quadrantData[i]) < app.quadrantHeight {
			// Expand
			newData := make([]string, app.quadrantHeight)
			copy(newData, app.quadrantData[i])
			app.quadrantData[i] = newData
		} else if len(app.quadrantData[i]) > app.quadrantHeight {
			// Truncate
			app.quadrantData[i] = app.quadrantData[i][:app.quadrantHeight]
		}
	}
}

// updateQuadrant randomly updates a line in the specified quadrant.
func (app *FlickerFreeApp) updateQuadrant(quadrant int) {
	if app.quadrantHeight < 1 {
		return
	}

	// Update a random line
	line := rand.Intn(app.quadrantHeight)

	// Build a random string
	str := ""
	for i := 0; i < 20; i++ {
		str += app.chars[rand.Intn(len(app.chars))]
	}

	// Add a timestamp
	str += fmt.Sprintf(" %d", time.Now().UnixNano())

	// Store the data
	app.quadrantData[quadrant][line] = str
}

// Render draws the current application state.
func (app *FlickerFreeApp) Render(frame gooey.RenderFrame) {
	width, height := frame.Size()

	// Update dimensions if changed
	if app.width != width || app.height != height {
		app.width = width
		app.height = height
		app.updateLayout()
	}

	// Define styles
	rainbowStyle := gooey.NewStyle().WithBold().WithForeground(gooey.ColorCyan)
	normalStyle := gooey.NewStyle().WithForeground(gooey.ColorWhite)

	// Header
	title := "⚡ Flicker-Free Update Demo ⚡"
	frame.PrintStyled(0, 0, title, rainbowStyle)
	frame.PrintStyled(0, 1, "Updates occur rapidly. Press Ctrl+C or Q to exit.", normalStyle)
	frame.PrintStyled(0, 2, "--------------------------------------------------", normalStyle)

	// Calculate quadrant positions
	midX := width / 2
	midY := (height-3)/2 + 3

	// Draw quadrants with colored borders
	quadrantColors := []gooey.Color{
		gooey.ColorRed,
		gooey.ColorGreen,
		gooey.ColorBlue,
		gooey.ColorYellow,
	}

	quadrantPositions := [][2]int{
		{2, 4},               // Q1: top-left
		{midX + 2, 4},        // Q2: top-right
		{2, midY + 2},        // Q3: bottom-left
		{midX + 2, midY + 2}, // Q4: bottom-right
	}

	// Draw each quadrant
	for q := 0; q < 4; q++ {
		x := quadrantPositions[q][0]
		y := quadrantPositions[q][1]

		// Draw quadrant label
		label := fmt.Sprintf("Q%d", q+1)
		labelStyle := gooey.NewStyle().WithForeground(quadrantColors[q]).WithBold()
		frame.PrintStyled(x, y-1, label, labelStyle)

		// Draw quadrant data
		for i := 0; i < app.quadrantHeight && i < len(app.quadrantData[q]); i++ {
			line := app.quadrantData[q][i]
			if line == "" {
				continue
			}

			// Create animated style for this line
			color := app.colors[rand.Intn(len(app.colors))]
			lineStyle := gooey.NewStyle().WithFgRGB(color)

			// Draw the line (truncate if too long)
			maxWidth := midX - 6
			if maxWidth < 1 {
				maxWidth = 10
			}
			if len(line) > maxWidth {
				line = line[:maxWidth]
			}

			frame.PrintStyled(x, y+i, line, lineStyle)
		}
	}
}

func main() {
	if err := gooey.Run(&FlickerFreeApp{}); err != nil {
		log.Fatal(err)
	}
}
