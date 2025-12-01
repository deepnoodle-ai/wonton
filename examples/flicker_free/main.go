package main

import (
	"fmt"
	"image"
	"log"
	"math/rand"
	"time"

	"github.com/deepnoodle-ai/wonton/tui"
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
	colors []tui.RGB
}

// Init initializes the application state.
func (app *FlickerFreeApp) Init() error {
	// Initialize random data
	app.chars = []string{"@", "#", "$", "%", "&", "*", "+", "=", "?", "!"}
	app.colors = []tui.RGB{
		tui.NewRGB(255, 0, 0),
		tui.NewRGB(0, 255, 0),
		tui.NewRGB(0, 0, 255),
		tui.NewRGB(255, 255, 0),
		tui.NewRGB(0, 255, 255),
		tui.NewRGB(255, 0, 255),
	}

	// Initialize empty quadrant data
	for i := range app.quadrantData {
		app.quadrantData[i] = make([]string, 0)
	}

	return nil
}

// HandleEvent processes events from the runtime.
func (app *FlickerFreeApp) HandleEvent(event tui.Event) []tui.Cmd {
	switch e := event.(type) {
	case tui.KeyEvent:
		// Handle keyboard input
		if e.Key == tui.KeyCtrlC || e.Rune == 'q' || e.Rune == 'Q' {
			return []tui.Cmd{tui.Quit()}
		}

	case tui.ResizeEvent:
		// Update dimensions on resize
		app.width = e.Width
		app.height = e.Height
		app.updateLayout()

	case tui.TickEvent:
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

// View returns the declarative view structure.
func (app *FlickerFreeApp) View() tui.View {
	return tui.VStack(
		// Header
		tui.Text("⚡ Flicker-Free Update Demo ⚡").Bold().Fg(tui.ColorCyan),
		tui.Text("Updates occur rapidly. Press Ctrl+C or Q to exit.").Fg(tui.ColorWhite),
		tui.Text("--------------------------------------------------").Fg(tui.ColorWhite),

		tui.Spacer(),

		// Quadrants canvas
		tui.Canvas(func(frame tui.RenderFrame, bounds image.Rectangle) {
			width := bounds.Dx()
			height := bounds.Dy()

			// Update dimensions if changed
			if app.width != width || app.height != height {
				app.width = width
				app.height = height
				app.updateLayout()
			}

			// Calculate quadrant positions
			midX := width / 2
			midY := height / 2

			// Draw quadrants with colored borders
			quadrantColors := []tui.Color{
				tui.ColorRed,
				tui.ColorGreen,
				tui.ColorBlue,
				tui.ColorYellow,
			}

			quadrantPositions := [][2]int{
				{2, 1},               // Q1: top-left
				{midX + 2, 1},        // Q2: top-right
				{2, midY + 2},        // Q3: bottom-left
				{midX + 2, midY + 2}, // Q4: bottom-right
			}

			// Draw each quadrant
			for q := 0; q < 4; q++ {
				x := quadrantPositions[q][0]
				y := quadrantPositions[q][1]

				// Draw quadrant label
				label := fmt.Sprintf("Q%d", q+1)
				labelStyle := tui.NewStyle().WithForeground(quadrantColors[q]).WithBold()
				frame.PrintStyled(bounds.Min.X+x, bounds.Min.Y+y-1, label, labelStyle)

				// Draw quadrant data
				for i := 0; i < app.quadrantHeight && i < len(app.quadrantData[q]); i++ {
					line := app.quadrantData[q][i]
					if line == "" {
						continue
					}

					// Create animated style for this line
					color := app.colors[rand.Intn(len(app.colors))]
					lineStyle := tui.NewStyle().WithFgRGB(color)

					// Draw the line (truncate if too long)
					maxWidth := midX - 6
					if maxWidth < 1 {
						maxWidth = 10
					}
					if len(line) > maxWidth {
						line = line[:maxWidth]
					}

					frame.PrintStyled(bounds.Min.X+x, bounds.Min.Y+y+i, line, lineStyle)
				}
			}
		}),
	)
}

func main() {
	if err := tui.Run(&FlickerFreeApp{}); err != nil {
		log.Fatal(err)
	}
}
