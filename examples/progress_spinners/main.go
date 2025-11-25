package main

import (
	"fmt"
	"time"

	"github.com/deepnoodle-ai/gooey"
)

// ProgressItem represents a single progress indicator
type ProgressItem struct {
	ID          string
	Message     string
	Progress    int
	Total       int
	SpinnerOnly bool
	SpinnerIdx  int
	Style       gooey.Style
	Complete    bool
}

// ProgressDemoApp demonstrates multiple progress indicators and spinners using Runtime.
type ProgressDemoApp struct {
	items      []*ProgressItem
	frame      uint64
	width      int
	height     int
	startTime  time.Time
	spinnerSet []string
}

// Init initializes the progress demo
func (app *ProgressDemoApp) Init() error {
	app.startTime = time.Now()

	// Use a simple spinner set
	app.spinnerSet = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

	// Initialize progress items
	app.items = []*ProgressItem{
		{
			ID:          "download",
			Message:     "Downloading assets...",
			Progress:    0,
			Total:       100,
			SpinnerOnly: false,
			Style:       gooey.NewStyle().WithForeground(gooey.ColorCyan),
		},
		{
			ID:          "db",
			Message:     "Connecting to database...",
			Progress:    0,
			Total:       1,
			SpinnerOnly: true,
			Style:       gooey.NewStyle().WithForeground(gooey.ColorYellow),
		},
		{
			ID:          "process",
			Message:     "Waiting to process...",
			Progress:    0,
			Total:       50,
			SpinnerOnly: false,
			Style:       gooey.NewStyle().WithForeground(gooey.ColorGreen),
		},
	}

	return nil
}

// HandleEvent processes events from the Runtime.
func (app *ProgressDemoApp) HandleEvent(event gooey.Event) []gooey.Cmd {
	switch e := event.(type) {
	case gooey.TickEvent:
		app.frame = e.Frame

		// Update spinner indices for all items
		for _, item := range app.items {
			item.SpinnerIdx = int(app.frame/4) % len(app.spinnerSet)
		}

		// Simulate download progress (item 0)
		if app.items[0].Progress < 100 && !app.items[0].Complete {
			app.items[0].Progress++
			app.items[0].Message = "Downloading assets..."
			if app.items[0].Progress >= 100 {
				app.items[0].Message = "Download complete!"
				app.items[0].Complete = true
			}
		}

		// Simulate database connection (item 1) - completes after 2 seconds (60 frames at 30 FPS)
		if app.frame == 60 && !app.items[1].Complete {
			app.items[1].Message = "Connected to DB!"
			app.items[1].Complete = true

			// Start processing after DB connection
			app.items[2].Message = "Processing records..."
		}

		// Simulate processing progress (item 2) - starts after DB connection
		if app.items[1].Complete && app.items[2].Progress < 50 && !app.items[2].Complete {
			// Update every 3 frames for slower progress
			if app.frame%3 == 0 {
				app.items[2].Progress++
				app.items[2].Message = fmt.Sprintf("Processing records... %d/50", app.items[2].Progress)
				if app.items[2].Progress >= 50 {
					app.items[2].Message = "Processing complete!"
					app.items[2].Complete = true
				}
			}
		}

		// Auto-quit after all tasks complete and 2 seconds have passed
		allComplete := true
		for _, item := range app.items {
			if !item.Complete {
				allComplete = false
				break
			}
		}
		if allComplete && app.frame > 120 { // 120 frames = 4 seconds at 30 FPS (2 sec after start, roughly)
			// Give user a moment to see completion
			if app.frame == 240 { // 8 seconds total
				return []gooey.Cmd{gooey.Quit()}
			}
		}

	case gooey.KeyEvent:
		// Allow manual quit
		if e.Rune == 'q' || e.Rune == 'Q' {
			return []gooey.Cmd{gooey.Quit()}
		}

	case gooey.ResizeEvent:
		app.width = e.Width
		app.height = e.Height
	}

	return nil
}

// Render draws the progress indicators.
func (app *ProgressDemoApp) Render(frame gooey.RenderFrame) {
	width, height := frame.Size()

	// Clear screen
	frame.FillStyled(0, 0, width, height, ' ', gooey.NewStyle())

	// Draw header
	headerStyle := gooey.NewStyle().WithBold().WithForeground(gooey.ColorCyan)
	frame.PrintStyled(2, 1, "Multi-Progress Demo", headerStyle)
	frame.PrintStyled(2, 2, "======================", headerStyle)

	// Draw progress items starting at line 4
	startY := 4
	for i, item := range app.items {
		y := startY + i*2
		if y >= height-2 {
			break
		}

		// Draw spinner
		spinner := app.spinnerSet[item.SpinnerIdx]
		frame.PrintStyled(2, y, spinner, item.Style)

		// Draw message
		frame.PrintStyled(4, y, item.Message, item.Style)

		// Draw progress bar if not spinner-only
		if !item.SpinnerOnly {
			barWidth := 30
			barX := 4
			barY := y + 1

			// Draw progress bar background
			bgStyle := gooey.NewStyle().WithForeground(gooey.ColorBrightBlack)
			for x := 0; x < barWidth; x++ {
				frame.SetCell(barX+x, barY, '░', bgStyle)
			}

			// Draw progress bar fill
			fillWidth := (item.Progress * barWidth) / item.Total
			if fillWidth > barWidth {
				fillWidth = barWidth
			}
			for x := 0; x < fillWidth; x++ {
				frame.SetCell(barX+x, barY, '█', item.Style)
			}

			// Draw percentage
			percentText := fmt.Sprintf(" %d%%", (item.Progress*100)/item.Total)
			frame.PrintStyled(barX+barWidth+1, barY, percentText, item.Style)
		}
	}

	// Draw help text at bottom
	helpStyle := gooey.NewStyle().WithForeground(gooey.ColorWhite)
	frame.PrintStyled(2, height-2, "Press 'q' to quit", helpStyle)

	// Draw elapsed time
	elapsed := time.Since(app.startTime).Truncate(time.Millisecond)
	timeStyle := gooey.NewStyle().WithForeground(gooey.ColorBrightBlack)
	timeText := fmt.Sprintf("Elapsed: %s", elapsed)
	frame.PrintStyled(width-len(timeText)-2, height-1, timeText, timeStyle)
}

func main() {
	// Initialize terminal
	terminal, err := gooey.NewTerminal()
	if err != nil {
		fmt.Printf("Error initializing terminal: %v\n", err)
		return
	}
	defer terminal.Close()

	// Get initial terminal size
	width, height := terminal.Size()

	// Create the application
	app := &ProgressDemoApp{
		width:  width,
		height: height,
	}

	// Create and run the runtime with 30 FPS
	runtime := gooey.NewRuntime(terminal, app, 30)

	// Run the event loop (blocks until quit)
	if err := runtime.Run(); err != nil {
		fmt.Printf("Runtime error: %v\n", err)
		return
	}

	fmt.Println("\n✨ All tasks finished!")
}
