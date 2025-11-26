package main

import (
	"fmt"
	"image"
	"log"
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
		if e.Rune == 'q' || e.Rune == 'Q' || e.Key == gooey.KeyEscape || e.Key == gooey.KeyCtrlC {
			return []gooey.Cmd{gooey.Quit()}
		}
	}

	return nil
}

// View returns the declarative view hierarchy.
func (app *ProgressDemoApp) View() gooey.View {
	elapsed := time.Since(app.startTime).Truncate(time.Millisecond)

	return gooey.VStack(
		// Header
		gooey.Text("Multi-Progress Demo").Bold().Fg(gooey.ColorCyan),
		gooey.Text("======================").Bold().Fg(gooey.ColorCyan),
		gooey.Spacer().MinHeight(1),

		// Progress items
		gooey.ForEach(app.items, func(item *ProgressItem, i int) gooey.View {
			spinner := app.spinnerSet[item.SpinnerIdx]

			// Build the item view
			itemContent := gooey.HStack(
				gooey.Text("%s", spinner).Style(item.Style),
				gooey.Text("%s", item.Message).Style(item.Style),
			).Gap(1)

			// If not spinner-only, add progress bar below
			if !item.SpinnerOnly {
				progressBar := gooey.Canvas(func(frame gooey.RenderFrame, bounds image.Rectangle) {
					app.drawProgressBar(frame, bounds, item)
				}).Height(1)

				return gooey.VStack(
					itemContent,
					progressBar,
				)
			}

			return itemContent
		}).Gap(1),

		gooey.Spacer(),

		// Bottom controls
		gooey.HStack(
			gooey.Text("Press 'q' to quit").Fg(gooey.ColorWhite),
			gooey.Spacer(),
			gooey.Text("Elapsed: %s", elapsed).Fg(gooey.ColorBrightBlack),
		),
	).Padding(2)
}

// drawProgressBar draws a progress bar in imperative style using Canvas.
func (app *ProgressDemoApp) drawProgressBar(frame gooey.RenderFrame, bounds image.Rectangle, item *ProgressItem) {
	barWidth := 30
	width := bounds.Dx()

	// Ensure we have space for the bar
	if width < barWidth+2 {
		barWidth = width - 10
		if barWidth < 5 {
			return
		}
	}

	// Draw progress bar background
	bgStyle := gooey.NewStyle().WithForeground(gooey.ColorBrightBlack)
	for x := 0; x < barWidth; x++ {
		frame.SetCell(x, 0, '░', bgStyle)
	}

	// Draw progress bar fill
	fillWidth := (item.Progress * barWidth) / item.Total
	if fillWidth > barWidth {
		fillWidth = barWidth
	}
	for x := 0; x < fillWidth; x++ {
		frame.SetCell(x, 0, '█', item.Style)
	}

	// Draw percentage
	percentText := fmt.Sprintf(" %d%%", (item.Progress*100)/item.Total)
	frame.PrintStyled(barWidth+1, 0, percentText, item.Style)
}

func main() {
	app := &ProgressDemoApp{}

	if err := gooey.Run(app); err != nil {
		log.Fatal(err)
	}

	log.Println("\n✨ All tasks finished!")
}
