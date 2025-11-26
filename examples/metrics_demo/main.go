package main

import (
	"fmt"
	"log"
	"time"

	"github.com/deepnoodle-ai/gooey"
)

// MetricsApp demonstrates the performance metrics system using the Runtime architecture.
// It shows live metrics during an animated demo and final metrics after completion.
type MetricsApp struct {
	terminal    *gooey.Terminal
	frame       uint64
	demoRunning bool
	startTime   time.Time
	demoDone    bool

	// Animation colors
	colors []gooey.Color

	// Current metrics snapshot
	metrics gooey.MetricsSnapshot
}

// HandleEvent processes events from the Runtime.
func (app *MetricsApp) HandleEvent(event gooey.Event) []gooey.Cmd {
	switch e := event.(type) {
	case gooey.TickEvent:
		app.frame = e.Frame

		// Get current metrics before rendering
		app.metrics = app.terminal.GetMetrics()

		// Check if demo duration has elapsed (5 seconds)
		if app.demoRunning && time.Since(app.startTime) >= 5*time.Second {
			app.demoRunning = false
			app.demoDone = true

			// Disable raw mode so Ctrl+C works for final screen
			app.terminal.DisableRawMode()
		}

	case gooey.KeyEvent:
		// Handle user input
		if e.Key == gooey.KeyCtrlC {
			// Quit the application
			return []gooey.Cmd{gooey.Quit()}
		}

	case gooey.ResizeEvent:
		// Terminal resized - metrics will adapt automatically
	}

	return nil
}

// View returns the declarative view for the current application state.
func (app *MetricsApp) View() gooey.View {
	if app.demoRunning {
		return app.viewDemo()
	} else if app.demoDone {
		return app.viewFinalMetrics()
	}
	return gooey.VStack()
}

// viewDemo returns the animated demo view with live metrics.
func (app *MetricsApp) viewDemo() gooey.View {
	// Build animated content lines
	animatedLines := []gooey.View{}
	for i := 0; i < 10; i++ {
		offset := int(app.frame) + i*5
		char := "▓▒░ "[offset%4]
		colorIdx := (i + int(app.frame/5)) % len(app.colors)
		line := ""
		for j := 0; j < 40; j++ {
			if (j+offset)%4 == 0 {
				line += string(char)
			} else {
				line += " "
			}
		}
		animatedLines = append(animatedLines, gooey.Text("%s", line).Fg(app.colors[colorIdx]))
	}

	// Build progress bar string
	barWidth := 40
	progress := float64(time.Since(app.startTime)) / float64(5*time.Second)
	filled := int(progress * float64(barWidth))

	progressBar := "["
	for i := 0; i < barWidth; i++ {
		if i < filled {
			progressBar += "█"
		} else {
			progressBar += "░"
		}
	}
	progressBar += "]"

	// Animated header color
	colorIdx := int(app.frame/10) % len(app.colors)

	return gooey.VStack(
		gooey.Spacer().MinHeight(1),
		gooey.Text("=== Performance Metrics Demo ===").Bold().Fg(app.colors[colorIdx]),
		gooey.Spacer().MinHeight(1),
		gooey.VStack(animatedLines...),
		gooey.Spacer().MinHeight(1),
		gooey.Text("Progress: %s", progressBar).Fg(gooey.ColorGreen),
		gooey.Spacer().MinHeight(1),
		gooey.Text("Live Metrics:").Bold(),
		gooey.Text("  Frames rendered: %d", app.metrics.TotalFrames),
		gooey.Text("  Frames skipped:  %d", app.metrics.SkippedFrames),
		gooey.Text("  Cells updated:   %d", app.metrics.CellsUpdated),
		gooey.Text("  ANSI codes:      %d", app.metrics.ANSICodesEmitted),
		gooey.Text("  Bytes written:   %d (%.1f KB)", app.metrics.BytesWritten, float64(app.metrics.BytesWritten)/1024.0),
		gooey.Text("  Avg FPS:         %.2f", app.metrics.FPS()),
		gooey.Text("  Avg frame time:  %.2fms", app.metrics.AvgTimePerFrame.Seconds()*1000),
		gooey.Text("  Last frame:      %.2fms", app.metrics.LastFrameTime.Seconds()*1000),
		gooey.Text("  Efficiency:      %.1f%%", app.metrics.Efficiency()),
		gooey.Spacer(),
		gooey.Text("Animation running... Will show final metrics in a moment"),
	).Padding(2)
}

// viewFinalMetrics returns the final metrics screen view.
func (app *MetricsApp) viewFinalMetrics() gooey.View {
	return gooey.VStack(
		gooey.Spacer().MinHeight(1),
		gooey.Text("=== Final Performance Metrics ===").Bold().Fg(gooey.ColorCyan),
		gooey.Spacer().MinHeight(1),

		gooey.Text("Frames:").Fg(gooey.ColorYellow),
		gooey.Text("  Total rendered:  %d", app.metrics.TotalFrames),
		gooey.Text("  Skipped:         %d", app.metrics.SkippedFrames),
		gooey.Text("  Average FPS:     %.2f", app.metrics.FPS()),
		gooey.Spacer().MinHeight(1),

		gooey.Text("Rendering:").Fg(gooey.ColorYellow),
		gooey.Text("  Cells updated:   %d", app.metrics.CellsUpdated),
		gooey.Text("  ANSI codes:      %d", app.metrics.ANSICodesEmitted),
		gooey.Text("  Bytes written:   %d (%.2f KB)", app.metrics.BytesWritten, float64(app.metrics.BytesWritten)/1024.0),
		gooey.Text("  Efficiency:      %.1f%%", app.metrics.Efficiency()),
		gooey.Spacer().MinHeight(1),

		gooey.Text("Timing:").Fg(gooey.ColorYellow),
		gooey.Text("  Min frame time:  %.2fms", app.metrics.MinFrameTime.Seconds()*1000),
		gooey.Text("  Max frame time:  %.2fms", app.metrics.MaxFrameTime.Seconds()*1000),
		gooey.Text("  Avg frame time:  %.2fms", app.metrics.AvgTimePerFrame.Seconds()*1000),
		gooey.Text("  Last frame time: %.2fms", app.metrics.LastFrameTime.Seconds()*1000),
		gooey.Spacer().MinHeight(1),

		gooey.Text("Dirty Regions:").Fg(gooey.ColorYellow),
		gooey.Text("  Last size:       %d cells", app.metrics.LastDirtyArea),
		gooey.Text("  Max size:        %d cells", app.metrics.MaxDirtyArea),
		gooey.Text("  Avg size:        %.0f cells", app.metrics.AvgDirtyArea),
		gooey.Spacer().MinHeight(1),

		gooey.Text("Compact Summary:").Fg(gooey.ColorYellow),
		gooey.Text("  %s", app.metrics.Compact()).Fg(gooey.ColorGreen),

		gooey.Spacer(),
		gooey.Text("Press Ctrl+C to exit").Bold().Fg(gooey.ColorMagenta),
	).Padding(2)
}

// Init initializes the application.
func (app *MetricsApp) Init() error {
	// Enable performance metrics
	app.terminal.EnableMetrics()

	// Enable alternate screen, hide cursor
	// Note: Runtime automatically enables raw mode
	app.terminal.EnableAlternateScreen()
	app.terminal.HideCursor()

	// Start the demo
	app.demoRunning = true
	app.startTime = time.Now()

	return nil
}

// Destroy cleans up the application.
func (app *MetricsApp) Destroy() {
	// Restore terminal state
	app.terminal.ShowCursor()
	app.terminal.DisableAlternateScreen()
	// Note: Runtime automatically disables raw mode
}

func main() {
	// Create terminal
	terminal, err := gooey.NewTerminal()
	if err != nil {
		log.Fatalf("Failed to create terminal: %v\n", err)
	}
	defer terminal.Close()

	// Create the metrics application
	app := &MetricsApp{
		terminal: terminal,
		frame:    0,
		colors: []gooey.Color{
			gooey.ColorRed,
			gooey.ColorYellow,
			gooey.ColorGreen,
			gooey.ColorCyan,
			gooey.ColorBlue,
			gooey.ColorMagenta,
		},
	}

	// Create and run the runtime with 60 FPS for smooth animation
	runtime := gooey.NewRuntime(terminal, app, 60)

	// Run the event loop (blocks until quit)
	if err := runtime.Run(); err != nil {
		log.Fatalf("Runtime error: %v\n", err)
	}

	fmt.Println("\n✨ Metrics demo finished!")
}
