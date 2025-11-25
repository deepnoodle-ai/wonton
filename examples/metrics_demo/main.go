package main

import (
	"fmt"
	"os"
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

// Render draws the current application state.
func (app *MetricsApp) Render(frame gooey.RenderFrame) {
	if app.demoRunning {
		app.renderDemo(frame)
	} else if app.demoDone {
		app.renderFinalMetrics(frame)
	}
}

// renderDemo draws the animated demo with live metrics.
func (app *MetricsApp) renderDemo(frame gooey.RenderFrame) {
	_, height := frame.Size()

	// Clear screen
	frame.Fill(' ', gooey.NewStyle())

	// Draw animated header
	colorIdx := int(app.frame/10) % len(app.colors)
	style := gooey.NewStyle().
		WithForeground(app.colors[colorIdx]).
		WithBold()
	frame.PrintStyled(2, 2, "=== Performance Metrics Demo ===", style)

	// Draw some animated content
	for i := 0; i < 10; i++ {
		x := 2
		y := 4 + i
		if y >= height-10 {
			break
		}
		offset := int(app.frame) + i*5
		char := "▓▒░ "[offset%4]
		colorIdx := (i + int(app.frame/5)) % len(app.colors)
		style := gooey.NewStyle().WithForeground(app.colors[colorIdx])
		line := ""
		for j := 0; j < 40; j++ {
			if (j+offset)%4 == 0 {
				line += string(char)
			} else {
				line += " "
			}
		}
		frame.PrintStyled(x, y, line, style)
	}

	// Draw progress bar
	barWidth := 40
	progress := float64(time.Since(app.startTime)) / float64(5*time.Second)
	filled := int(progress * float64(barWidth))

	frame.PrintStyled(2, 16, "Progress: ", gooey.NewStyle())
	frame.PrintStyled(12, 16, "[", gooey.NewStyle())
	for i := 0; i < barWidth; i++ {
		if i < filled {
			frame.PrintStyled(13+i, 16, "█", gooey.NewStyle().WithForeground(gooey.ColorGreen))
		} else {
			frame.PrintStyled(13+i, 16, "░", gooey.NewStyle().WithForeground(gooey.ColorWhite))
		}
	}
	frame.PrintStyled(13+barWidth, 16, "]", gooey.NewStyle())

	// Display live metrics
	y := 18
	frame.PrintStyled(2, y, "Live Metrics:", gooey.NewStyle().WithBold())
	y++

	frame.PrintStyled(2, y, fmt.Sprintf("  Frames rendered: %d", app.metrics.TotalFrames), gooey.NewStyle())
	y++
	frame.PrintStyled(2, y, fmt.Sprintf("  Frames skipped:  %d", app.metrics.SkippedFrames), gooey.NewStyle())
	y++
	frame.PrintStyled(2, y, fmt.Sprintf("  Cells updated:   %d", app.metrics.CellsUpdated), gooey.NewStyle())
	y++
	frame.PrintStyled(2, y, fmt.Sprintf("  ANSI codes:      %d", app.metrics.ANSICodesEmitted), gooey.NewStyle())
	y++
	frame.PrintStyled(2, y, fmt.Sprintf("  Bytes written:   %d (%.1f KB)", app.metrics.BytesWritten, float64(app.metrics.BytesWritten)/1024.0), gooey.NewStyle())
	y++
	frame.PrintStyled(2, y, fmt.Sprintf("  Avg FPS:         %.2f", app.metrics.FPS()), gooey.NewStyle())
	y++
	frame.PrintStyled(2, y, fmt.Sprintf("  Avg frame time:  %.2fms", app.metrics.AvgTimePerFrame.Seconds()*1000), gooey.NewStyle())
	y++
	frame.PrintStyled(2, y, fmt.Sprintf("  Last frame:      %.2fms", app.metrics.LastFrameTime.Seconds()*1000), gooey.NewStyle())
	y++
	frame.PrintStyled(2, y, fmt.Sprintf("  Efficiency:      %.1f%%", app.metrics.Efficiency()), gooey.NewStyle())

	// Footer
	frame.PrintStyled(2, height-2, "Animation running... Will show final metrics in a moment",
		gooey.NewStyle().WithForeground(gooey.ColorWhite))
}

// renderFinalMetrics draws the final metrics screen.
func (app *MetricsApp) renderFinalMetrics(frame gooey.RenderFrame) {
	_, height := frame.Size()

	// Clear screen
	frame.Fill(' ', gooey.NewStyle())

	// Title
	titleStyle := gooey.NewStyle().WithBold().WithForeground(gooey.ColorCyan)
	frame.PrintStyled(2, 2, "=== Final Performance Metrics ===", titleStyle)

	// Display detailed metrics
	y := 4
	labelStyle := gooey.NewStyle().WithForeground(gooey.ColorYellow)
	valueStyle := gooey.NewStyle().WithForeground(gooey.ColorWhite)

	frame.PrintStyled(2, y, "Frames:", labelStyle)
	y++
	frame.PrintStyled(4, y, fmt.Sprintf("Total rendered:  %d", app.metrics.TotalFrames), valueStyle)
	y++
	frame.PrintStyled(4, y, fmt.Sprintf("Skipped:         %d", app.metrics.SkippedFrames), valueStyle)
	y++
	frame.PrintStyled(4, y, fmt.Sprintf("Average FPS:     %.2f", app.metrics.FPS()), valueStyle)
	y += 2

	frame.PrintStyled(2, y, "Rendering:", labelStyle)
	y++
	frame.PrintStyled(4, y, fmt.Sprintf("Cells updated:   %d", app.metrics.CellsUpdated), valueStyle)
	y++
	frame.PrintStyled(4, y, fmt.Sprintf("ANSI codes:      %d", app.metrics.ANSICodesEmitted), valueStyle)
	y++
	frame.PrintStyled(4, y, fmt.Sprintf("Bytes written:   %d (%.2f KB)", app.metrics.BytesWritten, float64(app.metrics.BytesWritten)/1024.0), valueStyle)
	y++
	frame.PrintStyled(4, y, fmt.Sprintf("Efficiency:      %.1f%%", app.metrics.Efficiency()), valueStyle)
	y += 2

	frame.PrintStyled(2, y, "Timing:", labelStyle)
	y++
	frame.PrintStyled(4, y, fmt.Sprintf("Min frame time:  %.2fms", app.metrics.MinFrameTime.Seconds()*1000), valueStyle)
	y++
	frame.PrintStyled(4, y, fmt.Sprintf("Max frame time:  %.2fms", app.metrics.MaxFrameTime.Seconds()*1000), valueStyle)
	y++
	frame.PrintStyled(4, y, fmt.Sprintf("Avg frame time:  %.2fms", app.metrics.AvgTimePerFrame.Seconds()*1000), valueStyle)
	y++
	frame.PrintStyled(4, y, fmt.Sprintf("Last frame time: %.2fms", app.metrics.LastFrameTime.Seconds()*1000), valueStyle)
	y += 2

	frame.PrintStyled(2, y, "Dirty Regions:", labelStyle)
	y++
	frame.PrintStyled(4, y, fmt.Sprintf("Last size:       %d cells", app.metrics.LastDirtyArea), valueStyle)
	y++
	frame.PrintStyled(4, y, fmt.Sprintf("Max size:        %d cells", app.metrics.MaxDirtyArea), valueStyle)
	y++
	frame.PrintStyled(4, y, fmt.Sprintf("Avg size:        %.0f cells", app.metrics.AvgDirtyArea), valueStyle)
	y += 2

	// Compact summary
	frame.PrintStyled(2, y, "Compact Summary:", labelStyle)
	y++
	frame.PrintStyled(4, y, app.metrics.Compact(), gooey.NewStyle().WithForeground(gooey.ColorGreen))

	// Footer instruction
	footerStyle := gooey.NewStyle().WithForeground(gooey.ColorMagenta).WithBold()
	frame.PrintStyled(2, height-2, "Press Ctrl+C to exit", footerStyle)
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
		fmt.Fprintf(os.Stderr, "Failed to create terminal: %v\n", err)
		os.Exit(1)
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
		fmt.Fprintf(os.Stderr, "Runtime error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n✨ Metrics demo finished!")
}
