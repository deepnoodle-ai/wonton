package main

import (
	"fmt"
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
	Color       gooey.Color
	Complete    bool
}

// ProgressDemoApp demonstrates multiple progress indicators and spinners using Runtime.
type ProgressDemoApp struct {
	items     []*ProgressItem
	frame     uint64
	startTime time.Time
}

// Init initializes the progress demo
func (app *ProgressDemoApp) Init() error {
	app.startTime = time.Now()

	// Initialize progress items
	app.items = []*ProgressItem{
		{
			ID:          "download",
			Message:     "Downloading assets...",
			Progress:    0,
			Total:       100,
			SpinnerOnly: false,
			Color:       gooey.ColorCyan,
		},
		{
			ID:          "db",
			Message:     "Connecting to database...",
			Progress:    0,
			Total:       1,
			SpinnerOnly: true,
			Color:       gooey.ColorYellow,
		},
		{
			ID:          "process",
			Message:     "Waiting to process...",
			Progress:    0,
			Total:       50,
			SpinnerOnly: false,
			Color:       gooey.ColorGreen,
		},
	}

	return nil
}

// HandleEvent processes events from the Runtime.
func (app *ProgressDemoApp) HandleEvent(event gooey.Event) []gooey.Cmd {
	switch e := event.(type) {
	case gooey.TickEvent:
		app.frame = e.Frame

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
		if allComplete && app.frame > 120 { // 120 frames = 4 seconds at 30 FPS
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
		gooey.HeaderBar("Multi-Progress Demo").Bg(gooey.ColorCyan).Fg(gooey.ColorBlack),
		gooey.Spacer().MinHeight(1),

		// Progress items using new declarative views
		gooey.ForEach(app.items, func(item *ProgressItem, i int) gooey.View {
			// Build the item view using Loading (spinner) view
			itemContent := gooey.HStack(
				gooey.Loading(app.frame).Label(item.Message).Fg(item.Color),
			)

			// If not spinner-only, add progress bar below using Progress view
			if !item.SpinnerOnly {
				return gooey.VStack(
					itemContent,
					gooey.Progress(item.Progress, item.Total).Width(30).Fg(item.Color),
				)
			}

			return itemContent
		}).Gap(1),

		gooey.Spacer(),

		// Divider before footer
		gooey.Divider(),
		gooey.Spacer().MinHeight(1),

		// Bottom controls
		gooey.HStack(
			gooey.Text("Press 'q' to quit").Fg(gooey.ColorWhite),
			gooey.Spacer(),
			gooey.Text("Elapsed: %s", elapsed).Dim(),
		),
	).Padding(2)
}

func main() {
	app := &ProgressDemoApp{}

	if err := gooey.Run(app); err != nil {
		log.Fatal(err)
	}

	log.Println("\n✨ All tasks finished!")
}
