package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/deepnoodle-ai/gooey"
)

func main() {
	// Create terminal
	terminal, err := gooey.NewTerminal()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create terminal: %v\n", err)
		os.Exit(1)
	}
	defer terminal.Close()

	// Enable raw mode and alternate screen
	terminal.EnableRawMode()
	terminal.EnableAlternateScreen()
	terminal.HideCursor()

	// Enable performance metrics
	terminal.EnableMetrics()

	// Enable automatic resize handling
	terminal.WatchResize()
	defer terminal.StopWatchResize()

	// Run animation for a few seconds
	runDemo(terminal, 5*time.Second)

	// Show final metrics and wait for Ctrl+C
	showFinalMetrics(terminal)
}

func runDemo(terminal *gooey.Terminal, duration time.Duration) {
	start := time.Now()
	frame := uint64(0)

	// Animation colors
	colors := []gooey.Color{
		gooey.ColorRed,
		gooey.ColorYellow,
		gooey.ColorGreen,
		gooey.ColorCyan,
		gooey.ColorBlue,
		gooey.ColorMagenta,
	}

	for time.Since(start) < duration {
		// Get current metrics BEFORE BeginFrame to avoid deadlock
		snapshot := terminal.GetMetrics()

		renderFrame, err := terminal.BeginFrame()
		if err != nil {
			continue
		}

		_, height := renderFrame.Size()

		// Clear screen
		renderFrame.Fill(' ', gooey.NewStyle())

		// Draw animated header
		colorIdx := int(frame/10) % len(colors)
		style := gooey.NewStyle().
			WithForeground(colors[colorIdx]).
			WithBold()
		renderFrame.PrintStyled(2, 2, "=== Performance Metrics Demo ===", style)

		// Draw some animated content
		for i := 0; i < 10; i++ {
			x := 2
			y := 4 + i
			offset := int(frame) + i*5
			char := "▓▒░ "[offset%4]
			colorIdx := (i + int(frame/5)) % len(colors)
			style := gooey.NewStyle().WithForeground(colors[colorIdx])
			line := ""
			for j := 0; j < 40; j++ {
				if (j+offset)%4 == 0 {
					line += string(char)
				} else {
					line += " "
				}
			}
			renderFrame.PrintStyled(x, y, line, style)
		}

		// Draw progress bar
		barWidth := 40
		progress := float64(time.Since(start)) / float64(duration)
		filled := int(progress * float64(barWidth))

		renderFrame.PrintStyled(2, 16, "Progress: ", gooey.NewStyle())
		renderFrame.PrintStyled(12, 16, "[", gooey.NewStyle())
		for i := 0; i < barWidth; i++ {
			if i < filled {
				renderFrame.PrintStyled(13+i, 16, "█", gooey.NewStyle().WithForeground(gooey.ColorGreen))
			} else {
				renderFrame.PrintStyled(13+i, 16, "░", gooey.NewStyle().WithForeground(gooey.ColorWhite))
			}
		}
		renderFrame.PrintStyled(13+barWidth, 16, "]", gooey.NewStyle())

		// Display live metrics
		y := 18
		renderFrame.PrintStyled(2, y, "Live Metrics:", gooey.NewStyle().WithBold())
		y++

		renderFrame.PrintStyled(2, y, fmt.Sprintf("  Frames rendered: %d", snapshot.TotalFrames), gooey.NewStyle())
		y++
		renderFrame.PrintStyled(2, y, fmt.Sprintf("  Frames skipped:  %d", snapshot.SkippedFrames), gooey.NewStyle())
		y++
		renderFrame.PrintStyled(2, y, fmt.Sprintf("  Cells updated:   %d", snapshot.CellsUpdated), gooey.NewStyle())
		y++
		renderFrame.PrintStyled(2, y, fmt.Sprintf("  ANSI codes:      %d", snapshot.ANSICodesEmitted), gooey.NewStyle())
		y++
		renderFrame.PrintStyled(2, y, fmt.Sprintf("  Bytes written:   %d (%.1f KB)", snapshot.BytesWritten, float64(snapshot.BytesWritten)/1024.0), gooey.NewStyle())
		y++
		renderFrame.PrintStyled(2, y, fmt.Sprintf("  Avg FPS:         %.2f", snapshot.FPS()), gooey.NewStyle())
		y++
		renderFrame.PrintStyled(2, y, fmt.Sprintf("  Avg frame time:  %.2fms", snapshot.AvgTimePerFrame.Seconds()*1000), gooey.NewStyle())
		y++
		renderFrame.PrintStyled(2, y, fmt.Sprintf("  Last frame:      %.2fms", snapshot.LastFrameTime.Seconds()*1000), gooey.NewStyle())
		y++
		renderFrame.PrintStyled(2, y, fmt.Sprintf("  Efficiency:      %.1f%%", snapshot.Efficiency()), gooey.NewStyle())

		// Footer
		renderFrame.PrintStyled(2, height-2, "Animation running... Press Ctrl+C to exit",
			gooey.NewStyle().WithForeground(gooey.ColorWhite))

		terminal.EndFrame(renderFrame)

		frame++
		time.Sleep(16 * time.Millisecond) // ~60 FPS
	}
}

func showFinalMetrics(terminal *gooey.Terminal) {
	metrics := terminal.GetMetrics()

	// Clear the screen and show final metrics
	frame, err := terminal.BeginFrame()
	if err != nil {
		return
	}

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
	frame.PrintStyled(4, y, fmt.Sprintf("Total rendered:  %d", metrics.TotalFrames), valueStyle)
	y++
	frame.PrintStyled(4, y, fmt.Sprintf("Skipped:         %d", metrics.SkippedFrames), valueStyle)
	y++
	frame.PrintStyled(4, y, fmt.Sprintf("Average FPS:     %.2f", metrics.FPS()), valueStyle)
	y += 2

	frame.PrintStyled(2, y, "Rendering:", labelStyle)
	y++
	frame.PrintStyled(4, y, fmt.Sprintf("Cells updated:   %d", metrics.CellsUpdated), valueStyle)
	y++
	frame.PrintStyled(4, y, fmt.Sprintf("ANSI codes:      %d", metrics.ANSICodesEmitted), valueStyle)
	y++
	frame.PrintStyled(4, y, fmt.Sprintf("Bytes written:   %d (%.2f KB)", metrics.BytesWritten, float64(metrics.BytesWritten)/1024.0), valueStyle)
	y++
	frame.PrintStyled(4, y, fmt.Sprintf("Efficiency:      %.1f%%", metrics.Efficiency()), valueStyle)
	y += 2

	frame.PrintStyled(2, y, "Timing:", labelStyle)
	y++
	frame.PrintStyled(4, y, fmt.Sprintf("Min frame time:  %.2fms", metrics.MinFrameTime.Seconds()*1000), valueStyle)
	y++
	frame.PrintStyled(4, y, fmt.Sprintf("Max frame time:  %.2fms", metrics.MaxFrameTime.Seconds()*1000), valueStyle)
	y++
	frame.PrintStyled(4, y, fmt.Sprintf("Avg frame time:  %.2fms", metrics.AvgTimePerFrame.Seconds()*1000), valueStyle)
	y++
	frame.PrintStyled(4, y, fmt.Sprintf("Last frame time: %.2fms", metrics.LastFrameTime.Seconds()*1000), valueStyle)
	y += 2

	frame.PrintStyled(2, y, "Dirty Regions:", labelStyle)
	y++
	frame.PrintStyled(4, y, fmt.Sprintf("Last size:       %d cells", metrics.LastDirtyArea), valueStyle)
	y++
	frame.PrintStyled(4, y, fmt.Sprintf("Max size:        %d cells", metrics.MaxDirtyArea), valueStyle)
	y++
	frame.PrintStyled(4, y, fmt.Sprintf("Avg size:        %.0f cells", metrics.AvgDirtyArea), valueStyle)
	y += 2

	// Compact summary
	frame.PrintStyled(2, y, "Compact Summary:", labelStyle)
	y++
	frame.PrintStyled(4, y, metrics.Compact(), gooey.NewStyle().WithForeground(gooey.ColorGreen))

	// Footer instruction
	footerStyle := gooey.NewStyle().WithForeground(gooey.ColorMagenta).WithBold()
	frame.PrintStyled(2, height-2, "Press Ctrl+C to exit", footerStyle)

	terminal.EndFrame(frame)

	// Disable raw mode so Ctrl+C works
	terminal.DisableRawMode()

	// Wait for Ctrl+C
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	// Clean up
	terminal.ShowCursor()
	terminal.DisableAlternateScreen()
}
