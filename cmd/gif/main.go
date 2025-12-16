// Command gif converts terminal recordings (.cast files) to animated GIFs
// and provides utilities for creating simple animated GIFs.
//
// Usage:
//
//	gif cast <input.cast> <output.gif>   Convert asciinema recording to GIF
//	gif demo <output.gif>                Create a demo animation
//	gif info <input.cast>                Show information about a cast file
//
// Options for cast conversion:
//
//	-cols     Number of terminal columns (default: from .cast file)
//	-rows     Number of terminal rows (default: from .cast file)
//	-speed    Playback speed multiplier (default: 1.0)
//	-maxidle  Max idle time in seconds (default: 2.0)
//	-padding  Padding around terminal in pixels (default: 8)
//	-fps      Target frames per second (default: 10)
//	-fontsize Font size in points (default: 14)
//	-bitmap   Use bitmap font instead of TTF
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/deepnoodle-ai/wonton/gif"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	switch cmd {
	case "cast":
		if len(os.Args) < 4 {
			fmt.Fprintln(os.Stderr, "Usage: gif cast [options] <input.cast> <output.gif>")
			os.Exit(1)
		}
		castCmd(os.Args[2:])
	case "demo":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: gif demo <output.gif>")
			os.Exit(1)
		}
		demoCmd(os.Args[2])
	case "info":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: gif info <input.cast>")
			os.Exit(1)
		}
		infoCmd(os.Args[2])
	case "-h", "--help", "help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`gif - Terminal recording to GIF converter

Usage:
  gif cast [options] <input.cast> <output.gif>   Convert .cast to GIF
  gif demo <output.gif>                          Create demo animation
  gif info <input.cast>                          Show cast file info

Cast options:
  -cols int       Terminal columns (default: from file)
  -rows int       Terminal rows (default: from file)
  -speed float    Playback speed multiplier (default: 1.0)
  -maxidle float  Max idle time between events in seconds (default: 2.0)
  -padding int    Padding around terminal in pixels (default: 8)
  -fps int        Target frames per second (default: 10)
  -fontsize float Font size in points (default: 14)
  -bitmap         Use bitmap font instead of TTF

Examples:
  gif cast session.cast output.gif
  gif cast -speed 2.0 -maxidle 1.0 session.cast output.gif
  gif info session.cast
  gif demo animation.gif`)
}

func castCmd(args []string) {
	fs := flag.NewFlagSet("cast", flag.ExitOnError)
	cols := fs.Int("cols", 0, "Terminal columns (0 = from file)")
	rows := fs.Int("rows", 0, "Terminal rows (0 = from file)")
	speed := fs.Float64("speed", 1.0, "Playback speed multiplier")
	maxIdle := fs.Float64("maxidle", 2.0, "Max idle time between events")
	padding := fs.Int("padding", 8, "Padding in pixels")
	fps := fs.Int("fps", 10, "Target frames per second")
	fontSize := fs.Float64("fontsize", 14, "Font size in points")
	useBitmap := fs.Bool("bitmap", false, "Use bitmap font instead of TTF")
	fs.Parse(args)

	if fs.NArg() < 2 {
		fmt.Fprintln(os.Stderr, "Usage: gif cast [options] <input.cast> <output.gif>")
		os.Exit(1)
	}

	inputFile := fs.Arg(0)
	outputFile := fs.Arg(1)

	// Get cast info for display
	info, err := gif.GetCastInfo(inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading %s: %v\n", inputFile, err)
		os.Exit(1)
	}

	termCols := info.Width
	termRows := info.Height
	if *cols > 0 {
		termCols = *cols
	}
	if *rows > 0 {
		termRows = *rows
	}

	fmt.Printf("Converting %s (%dx%d terminal, %d events, %.1fs)...\n",
		inputFile, termCols, termRows, info.EventCount, info.Duration)

	opts := gif.CastOptions{
		Cols:      *cols,
		Rows:      *rows,
		Speed:     *speed,
		MaxIdle:   *maxIdle,
		FPS:       *fps,
		Padding:   *padding,
		FontSize:  *fontSize,
		UseBitmap: *useBitmap,
	}

	g, err := gif.RenderCast(inputFile, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error rendering: %v\n", err)
		os.Exit(1)
	}

	if err := g.Save(outputFile); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving %s: %v\n", outputFile, err)
		os.Exit(1)
	}

	fmt.Printf("Created %s (%d frames)\n", outputFile, g.FrameCount())
}

func infoCmd(filename string) {
	info, err := gif.GetCastInfo(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("File:       %s\n", filename)
	fmt.Printf("Dimensions: %dx%d\n", info.Width, info.Height)
	fmt.Printf("Duration:   %.2fs\n", info.Duration)
	fmt.Printf("Events:     %d\n", info.EventCount)
	if info.Title != "" {
		fmt.Printf("Title:      %s\n", info.Title)
	}
	if info.Timestamp > 0 {
		t := time.Unix(info.Timestamp, 0)
		fmt.Printf("Recorded:   %s\n", t.Format(time.RFC3339))
	}
}

func demoCmd(outputFile string) {
	fmt.Println("Creating demo animation...")

	// Create a simple bouncing ball animation
	g := gif.New(200, 150)
	g.SetLoopCount(0)

	ballX, ballY := 50, 50
	velX, velY := 5, 3
	radius := 15

	for frame := 0; frame < 60; frame++ {
		x, y := ballX, ballY // Capture for closure
		g.AddFrameWithDelay(func(f *gif.Frame) {
			// Background
			f.Fill(gif.RGB(32, 32, 48))

			// Border
			f.DrawRect(5, 5, 190, 140, gif.RGB(100, 100, 120))

			// Ball shadow
			f.FillCircle(x+3, y+3, radius-2, gif.RGB(20, 20, 30))

			// Ball
			f.FillCircle(x, y, radius, gif.RGB(255, 100, 100))
			f.DrawCircle(x, y, radius, gif.RGB(255, 200, 200))
		}, 5) // 50ms per frame

		// Update ball position
		ballX += velX
		ballY += velY

		// Bounce off walls
		if ballX-radius <= 5 || ballX+radius >= 195 {
			velX = -velX
		}
		if ballY-radius <= 5 || ballY+radius >= 145 {
			velY = -velY
		}
	}

	if err := g.Save(outputFile); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Created %s (%d frames)\n", outputFile, g.FrameCount())
}
