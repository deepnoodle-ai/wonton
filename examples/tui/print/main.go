// Package main demonstrates inline view printing without a full TUI application.
//
// The Print function renders views directly to the terminal without:
//   - Enabling alternate screen mode
//   - Enabling raw mode or handling keyboard input
//   - Clearing the screen
//   - Starting an event loop
//
// This is useful for CLI tools that want styled output once and then exit.
//
// Run with: go run ./examples/tui/print
package main

import (
	"fmt"
	"time"

	"github.com/deepnoodle-ai/wonton/tui"
)

func main() {
	// Simple text with styling
	fmt.Println("=== Simple Styled Text ===")
	tui.Print(tui.Text("Hello from Print!").Bold().Fg(tui.ColorCyan))
	fmt.Println()

	// Multiple lines with a Stack
	fmt.Println("=== Stacked Content ===")
	tui.Print(tui.Stack(
		tui.Text("Status Report").Bold().Underline(),
		tui.Text(""),
		tui.Group(tui.Text("  Database: "), tui.Text("Connected").Fg(tui.ColorGreen)),
		tui.Group(tui.Text("  Cache:    "), tui.Text("Ready").Fg(tui.ColorGreen)),
		tui.Group(tui.Text("  Queue:    "), tui.Text("3 pending").Fg(tui.ColorYellow)),
	))
	fmt.Println()

	// Bordered content
	fmt.Println("=== Bordered Box ===")
	tui.Print(
		tui.Bordered(
			tui.Stack(
				tui.Text("Important Notice").Bold().Fg(tui.ColorYellow),
				tui.Text(""),
				tui.Text("This message is displayed inline"),
				tui.Text("without taking over the terminal."),
			).Padding(1),
		).Border(&tui.RoundedBorder).BorderFg(tui.ColorCyan),
		tui.WithWidth(45),
	)
	fmt.Println()

	// Horizontal layout with Group
	fmt.Println("=== Horizontal Layout ===")
	tui.Print(
		tui.Group(
			tui.Text("Left").Fg(tui.ColorRed),
			tui.Spacer(),
			tui.Text("Center").Fg(tui.ColorGreen),
			tui.Spacer(),
			tui.Text("Right").Fg(tui.ColorBlue),
		),
		tui.WithWidth(50),
	)
	fmt.Println()

	// Live updating progress bar
	fmt.Println("=== Live Progress Bar ===")
	live := tui.NewLivePrinter(tui.WithWidth(60))

	for i := 0; i <= 100; i += 5 {
		filled := i / 5
		empty := 20 - filled

		live.Update(tui.Stack(
			tui.Group(
				tui.Text("Downloading:").Fg(tui.ColorWhite),
				tui.Text(" archive.tar.gz").Fg(tui.ColorCyan),
			),
			tui.Group(
				tui.Text("[").Fg(tui.ColorWhite),
				tui.Text("%s", repeat("█", filled)).Fg(tui.ColorGreen),
				tui.Text("%s", repeat("░", empty)).Fg(tui.ColorBrightBlack),
				tui.Text("]").Fg(tui.ColorWhite),
				tui.Text(" %3d%%", i).Fg(tui.ColorYellow),
			),
		))
		time.Sleep(50 * time.Millisecond)
	}
	live.Stop()
	fmt.Println()

	// Live updating with changing height
	fmt.Println("=== Live Multi-line Status ===")
	tui.Live(func(update func(tui.View)) {
		tasks := []string{"Compiling", "Linking", "Optimizing", "Packaging"}
		for i := range tasks {
			// Build the status display
			var lines []tui.View
			lines = append(lines, tui.Text("Build Progress").Bold().Fg(tui.ColorCyan))
			lines = append(lines, tui.Text(""))

			for j, t := range tasks {
				var status tui.View
				if j < i {
					status = tui.Group(
						tui.Text("  ✓ ").Fg(tui.ColorGreen),
						tui.Text("%s", t).Fg(tui.ColorGreen),
					)
				} else if j == i {
					status = tui.Group(
						tui.Text("  ◐ ").Fg(tui.ColorYellow),
						tui.Text("%s", t).Fg(tui.ColorYellow),
						tui.Text("..."),
					)
				} else {
					status = tui.Group(
						tui.Text("  ○ ").Fg(tui.ColorBrightBlack),
						tui.Text("%s", t).Fg(tui.ColorBrightBlack),
					)
				}
				lines = append(lines, status)
			}

			update(tui.Stack(lines...))
			time.Sleep(400 * time.Millisecond)
		}

		// Final state - all complete
		var lines []tui.View
		lines = append(lines, tui.Text("Build Progress").Bold().Fg(tui.ColorCyan))
		lines = append(lines, tui.Text(""))
		for _, t := range tasks {
			lines = append(lines, tui.Group(
				tui.Text("  ✓ ").Fg(tui.ColorGreen),
				tui.Text("%s", t).Fg(tui.ColorGreen),
			))
		}
		lines = append(lines, tui.Text(""))
		lines = append(lines, tui.Text("  Build complete!").Bold().Fg(tui.ColorGreen))
		update(tui.Stack(lines...))
	}, tui.WithWidth(40))
	fmt.Println()

	// Error message example
	fmt.Println("=== Error Display ===")
	tui.Print(
		tui.Bordered(
			tui.Stack(
				tui.Text(" Error ").Bg(tui.ColorRed).Fg(tui.ColorWhite).Bold(),
				tui.Text(""),
				tui.Text("Connection refused: localhost:5432").Fg(tui.ColorRed),
				tui.Text("Please check if the database is running.").Dim(),
			).Padding(1),
		).Border(&tui.SingleBorder).BorderFg(tui.ColorRed),
		tui.WithWidth(50),
	)
	fmt.Println()

	fmt.Println("Done! Notice the terminal was not cleared.")
}

func repeat(s string, n int) string {
	result := ""
	for i := 0; i < n; i++ {
		result += s
	}
	return result
}
