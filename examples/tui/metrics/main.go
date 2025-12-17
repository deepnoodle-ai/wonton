package main

import (
	"fmt"
	"log"
	"time"

	"github.com/deepnoodle-ai/wonton/tui"
)

// MetricsApp demonstrates the performance metrics system using the Runtime architecture.
// It shows live metrics during an animated demo and final metrics after completion.
type MetricsApp struct {
	terminal    *tui.Terminal
	frame       uint64
	demoRunning bool
	startTime   time.Time
	demoDone    bool

	// Animation colors
	colors []tui.Color

	// Current metrics snapshot
	metrics tui.MetricsSnapshot
}

// HandleEvent processes events from the Runtime.
func (app *MetricsApp) HandleEvent(event tui.Event) []tui.Cmd {
	switch e := event.(type) {
	case tui.TickEvent:
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

	case tui.KeyEvent:
		// Handle user input
		if e.Key == tui.KeyCtrlC {
			// Quit the application
			return []tui.Cmd{tui.Quit()}
		}

	case tui.ResizeEvent:
		// Terminal resized - metrics will adapt automatically
	}

	return nil
}

// View returns the declarative view for the current application state.
func (app *MetricsApp) View() tui.View {
	if app.demoRunning {
		return app.viewDemo()
	} else if app.demoDone {
		return app.viewFinalMetrics()
	}
	return tui.Stack()
}

// viewDemo returns the animated demo view with live metrics.
func (app *MetricsApp) viewDemo() tui.View {
	// Build animated content lines
	animatedLines := []tui.View{}
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
		animatedLines = append(animatedLines, tui.Text("%s", line).Fg(app.colors[colorIdx]))
	}

	// Calculate progress
	progressVal := int(float64(time.Since(app.startTime)) / float64(5*time.Second) * 100)
	if progressVal > 100 {
		progressVal = 100
	}

	// Animated header color
	colorIdx := int(app.frame/10) % len(app.colors)

	return tui.Stack(
		tui.Spacer().MinHeight(1),
		tui.Text("=== Performance Metrics Demo ===").Bold().Fg(app.colors[colorIdx]),
		tui.Spacer().MinHeight(1),
		tui.Stack(animatedLines...),
		tui.Spacer().MinHeight(1),
		// Use Progress view instead of manual string building
		tui.Group(
			tui.Text("Progress:").Fg(tui.ColorGreen),
			tui.Progress(progressVal, 100).Width(40).Fg(tui.ColorGreen).HidePercent(),
		).Gap(1),
		tui.Spacer().MinHeight(1),
		tui.Text("Live Metrics:").Bold(),
		tui.Text("  Frames rendered: %d", app.metrics.TotalFrames),
		tui.Text("  Frames skipped:  %d", app.metrics.SkippedFrames),
		tui.Text("  Cells updated:   %d", app.metrics.CellsUpdated),
		tui.Text("  ANSI codes:      %d", app.metrics.ANSICodesEmitted),
		tui.Text("  Bytes written:   %d (%.1f KB)", app.metrics.BytesWritten, float64(app.metrics.BytesWritten)/1024.0),
		tui.Text("  Avg FPS:         %.2f", app.metrics.FPS()),
		tui.Text("  Avg frame time:  %.2fms", app.metrics.AvgTimePerFrame.Seconds()*1000),
		tui.Text("  Last frame:      %.2fms", app.metrics.LastFrameTime.Seconds()*1000),
		tui.Text("  Efficiency:      %.1f%%", app.metrics.Efficiency()),
		tui.Spacer(),
		tui.Text("Animation running... Will show final metrics in a moment"),
	).Padding(2)
}

// viewFinalMetrics returns the final metrics screen view.
func (app *MetricsApp) viewFinalMetrics() tui.View {
	return tui.Stack(
		tui.Spacer().MinHeight(1),
		tui.Text("=== Final Performance Metrics ===").Bold().Fg(tui.ColorCyan),
		tui.Spacer().MinHeight(1),

		tui.Text("Frames:").Fg(tui.ColorYellow),
		tui.Text("  Total rendered:  %d", app.metrics.TotalFrames),
		tui.Text("  Skipped:         %d", app.metrics.SkippedFrames),
		tui.Text("  Average FPS:     %.2f", app.metrics.FPS()),
		tui.Spacer().MinHeight(1),

		tui.Text("Rendering:").Fg(tui.ColorYellow),
		tui.Text("  Cells updated:   %d", app.metrics.CellsUpdated),
		tui.Text("  ANSI codes:      %d", app.metrics.ANSICodesEmitted),
		tui.Text("  Bytes written:   %d (%.2f KB)", app.metrics.BytesWritten, float64(app.metrics.BytesWritten)/1024.0),
		tui.Text("  Efficiency:      %.1f%%", app.metrics.Efficiency()),
		tui.Spacer().MinHeight(1),

		tui.Text("Timing:").Fg(tui.ColorYellow),
		tui.Text("  Min frame time:  %.2fms", app.metrics.MinFrameTime.Seconds()*1000),
		tui.Text("  Max frame time:  %.2fms", app.metrics.MaxFrameTime.Seconds()*1000),
		tui.Text("  Avg frame time:  %.2fms", app.metrics.AvgTimePerFrame.Seconds()*1000),
		tui.Text("  Last frame time: %.2fms", app.metrics.LastFrameTime.Seconds()*1000),
		tui.Spacer().MinHeight(1),

		tui.Text("Dirty Regions:").Fg(tui.ColorYellow),
		tui.Text("  Last size:       %d cells", app.metrics.LastDirtyArea),
		tui.Text("  Max size:        %d cells", app.metrics.MaxDirtyArea),
		tui.Text("  Avg size:        %.0f cells", app.metrics.AvgDirtyArea),
		tui.Spacer().MinHeight(1),

		tui.Text("Compact Summary:").Fg(tui.ColorYellow),
		tui.Text("  %s", app.metrics.Compact()).Fg(tui.ColorGreen),

		tui.Spacer(),
		tui.Text("Press Ctrl+C to exit").Bold().Fg(tui.ColorMagenta),
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
	terminal, err := tui.NewTerminal()
	if err != nil {
		log.Fatalf("Failed to create terminal: %v\n", err)
	}
	defer terminal.Close()

	// Create the metrics application
	app := &MetricsApp{
		terminal: terminal,
		frame:    0,
		colors: []tui.Color{
			tui.ColorRed,
			tui.ColorYellow,
			tui.ColorGreen,
			tui.ColorCyan,
			tui.ColorBlue,
			tui.ColorMagenta,
		},
	}

	// Create and run the runtime with 60 FPS for smooth animation
	runtime := tui.NewRuntime(terminal, app, 60)

	// Run the event loop (blocks until quit)
	if err := runtime.Run(); err != nil {
		log.Fatalf("Runtime error: %v\n", err)
	}

	fmt.Println("\n✨ Metrics demo finished!")
}
