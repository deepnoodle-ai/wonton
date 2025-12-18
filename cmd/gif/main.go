// Command gif converts terminal recordings (.cast files) to animated GIFs
// and provides utilities for creating animated GIFs.
package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/deepnoodle-ai/wonton/cli"
	"github.com/deepnoodle-ai/wonton/gif"
)

func main() {
	app := cli.New("gif").
		Description("Terminal recording to GIF converter").
		Version("1.0.0")

	// Cast command - convert .cast files to GIFs
	app.Command("cast").
		Description("Convert asciinema recording to GIF").
		Args("input", "output").
		Flags(
			&cli.IntFlag{Name: "cols", Short: "c", Help: "Terminal columns (0 = from file)", Value: 0},
			&cli.IntFlag{Name: "rows", Short: "r", Help: "Terminal rows (0 = from file)", Value: 0},
			&cli.Float64Flag{Name: "speed", Short: "s", Help: "Playback speed multiplier", Value: 1.0},
			&cli.Float64Flag{Name: "maxidle", Short: "m", Help: "Max idle time between events (seconds)", Value: 2.0},
			&cli.IntFlag{Name: "padding", Short: "p", Help: "Padding around terminal (pixels)", Value: 8},
			&cli.IntFlag{Name: "fps", Short: "f", Help: "Target frames per second", Value: 10},
			&cli.Float64Flag{Name: "fontsize", Help: "Font size in points", Value: 14},
			&cli.BoolFlag{Name: "bitmap", Short: "b", Help: "Use bitmap font instead of TTF"},
		).
		Run(castCmd)

	// Info command - show cast file information
	app.Command("info").
		Description("Show information about a cast file").
		Args("file").
		Run(infoCmd)

	// Demo command - create a demo animation
	app.Command("demo").
		Description("Create a demo bouncing ball animation").
		Args("output").
		Flags(
			&cli.IntFlag{Name: "width", Short: "w", Help: "GIF width in pixels", Value: 200},
			&cli.IntFlag{Name: "height", Short: "h", Help: "GIF height in pixels", Value: 150},
			&cli.IntFlag{Name: "frames", Short: "f", Help: "Number of frames", Value: 60},
		).
		Run(demoCmd)

	if err := app.Execute(); err != nil {
		if !cli.IsHelpRequested(err) {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}

func castCmd(ctx *cli.Context) error {
	inputFile := ctx.Arg(0)
	outputFile := ctx.Arg(1)

	if !strings.HasSuffix(strings.ToLower(outputFile), ".gif") {
		return fmt.Errorf("output file must have .gif extension: %s", outputFile)
	}

	// Get cast info for display
	info, err := gif.GetCastInfo(inputFile)
	if err != nil {
		return fmt.Errorf("loading %s: %w", inputFile, err)
	}

	cols := ctx.Int("cols")
	rows := ctx.Int("rows")

	termCols := info.Width
	termRows := info.Height
	if cols > 0 {
		termCols = cols
	}
	if rows > 0 {
		termRows = rows
	}

	ctx.Printf("Converting %s (%dx%d terminal, %d events, %.1fs)...\n",
		inputFile, termCols, termRows, info.EventCount, info.Duration)

	opts := gif.CastOptions{
		Cols:      cols,
		Rows:      rows,
		Speed:     ctx.Float64("speed"),
		MaxIdle:   ctx.Float64("maxidle"),
		FPS:       ctx.Int("fps"),
		Padding:   ctx.Int("padding"),
		FontSize:  ctx.Float64("fontsize"),
		UseBitmap: ctx.Bool("bitmap"),
	}

	g, err := gif.RenderCast(inputFile, opts)
	if err != nil {
		return fmt.Errorf("rendering: %w", err)
	}

	if err := g.Save(outputFile); err != nil {
		return fmt.Errorf("saving %s: %w", outputFile, err)
	}

	ctx.Success("Created %s (%d frames)", outputFile, g.FrameCount())
	return nil
}

func infoCmd(ctx *cli.Context) error {
	filename := ctx.Arg(0)

	info, err := gif.GetCastInfo(filename)
	if err != nil {
		return err
	}

	ctx.Printf("File:       %s\n", filename)
	ctx.Printf("Dimensions: %dx%d\n", info.Width, info.Height)
	ctx.Printf("Duration:   %.2fs\n", info.Duration)
	ctx.Printf("Events:     %d\n", info.EventCount)
	if info.Title != "" {
		ctx.Printf("Title:      %s\n", info.Title)
	}
	if info.Timestamp > 0 {
		t := time.Unix(info.Timestamp, 0)
		ctx.Printf("Recorded:   %s\n", t.Format(time.RFC3339))
	}
	return nil
}

func demoCmd(ctx *cli.Context) error {
	outputFile := ctx.Arg(0)

	if !strings.HasSuffix(strings.ToLower(outputFile), ".gif") {
		return fmt.Errorf("output file must have .gif extension: %s", outputFile)
	}

	width := ctx.Int("width")
	height := ctx.Int("height")
	frames := ctx.Int("frames")

	ctx.Println("Creating demo animation...")

	// Create a simple bouncing ball animation
	g := gif.New(width, height)
	g.SetLoopCount(0)

	ballX, ballY := width/4, height/3
	velX, velY := 5, 3
	radius := 15

	for frame := 0; frame < frames; frame++ {
		x, y := ballX, ballY // Capture for closure
		g.AddFrameWithDelay(func(f *gif.Frame) {
			// Background
			f.Fill(gif.RGB(32, 32, 48))

			// Border
			f.DrawRect(5, 5, width-10, height-10, gif.RGB(100, 100, 120))

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
		if ballX-radius <= 5 || ballX+radius >= width-5 {
			velX = -velX
		}
		if ballY-radius <= 5 || ballY+radius >= height-5 {
			velY = -velY
		}
	}

	if err := g.Save(outputFile); err != nil {
		return err
	}

	ctx.Success("Created %s (%d frames)", outputFile, g.FrameCount())
	return nil
}
