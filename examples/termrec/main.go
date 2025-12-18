// Example: termrec - Terminal session recorder and GIF exporter
//
// Records terminal sessions and exports them to animated GIFs. Perfect for
// creating demos, documentation, and README visuals.
//
// Run with:
//
//	go run ./examples/termrec record demo.cast          # Record a session
//	go run ./examples/termrec export demo.cast demo.gif # Convert to GIF
//	go run ./examples/termrec info demo.cast            # Show recording info
//	go run ./examples/termrec play demo.cast            # Play back recording
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/deepnoodle-ai/wonton/cli"
	"github.com/deepnoodle-ai/wonton/color"
	"github.com/deepnoodle-ai/wonton/gif"
	"github.com/deepnoodle-ai/wonton/humanize"
	"github.com/deepnoodle-ai/wonton/termsession"
	"github.com/deepnoodle-ai/wonton/tui"
)

func main() {
	app := cli.New("termrec").
		Description("Record terminal sessions and export to GIF").
		Version("1.0.0")

	// Record command
	app.Command("record").
		Description("Record a terminal session").
		Args("output").
		Flags(
			cli.String("title", "t").
				Help("Recording title"),
			cli.String("shell", "s").
				Default("").
				Help("Shell to use (default: $SHELL or /bin/sh)"),
			cli.Float("max-idle", "i").
				Default(2.0).
				Help("Max idle time between events (seconds)"),
			cli.Bool("compress", "c").
				Default(true).
				Help("Compress output with gzip"),
		).
		Run(func(ctx *cli.Context) error {
			output := ctx.Arg(0)
			title := ctx.String("title")
			shell := ctx.String("shell")
			maxIdle := ctx.Float64("max-idle")
			compress := ctx.Bool("compress")

			return recordSession(output, title, shell, maxIdle, compress)
		})

	// Export command
	app.Command("export").
		Description("Convert recording to animated GIF").
		Args("input", "output?").
		Flags(
			cli.Float("speed", "s").
				Default(1.0).
				Help("Playback speed multiplier"),
			cli.Float("max-idle", "i").
				Default(2.0).
				Help("Max idle time between frames (seconds)"),
			cli.Int("fps", "f").
				Default(10).
				Help("Frames per second"),
			cli.Float("font-size", "").
				Default(14.0).
				Help("Font size in points"),
			cli.Int("padding", "p").
				Default(8).
				Help("Padding around content (pixels)"),
			cli.Int("cols", "").
				Default(0).
				Help("Override terminal columns (0 = auto)"),
			cli.Int("rows", "").
				Default(0).
				Help("Override terminal rows (0 = auto)"),
		).
		Run(func(ctx *cli.Context) error {
			input := ctx.Arg(0)
			output := ctx.Arg(1)
			if output == "" {
				output = strings.TrimSuffix(input, filepath.Ext(input)) + ".gif"
			}

			opts := gif.CastOptions{
				Speed:    ctx.Float64("speed"),
				MaxIdle:  ctx.Float64("max-idle"),
				FPS:      ctx.Int("fps"),
				FontSize: ctx.Float64("font-size"),
				Padding:  ctx.Int("padding"),
				Cols:     ctx.Int("cols"),
				Rows:     ctx.Int("rows"),
			}

			return exportToGIF(input, output, opts)
		})

	// Info command
	app.Command("info").
		Description("Show recording information").
		Args("file").
		Run(func(ctx *cli.Context) error {
			file := ctx.Arg(0)
			return showInfo(file)
		})

	// Play command
	app.Command("play").
		Description("Play back a recording").
		Args("file").
		Flags(
			cli.Float("speed", "s").
				Default(1.0).
				Help("Playback speed multiplier"),
			cli.Bool("loop", "l").
				Help("Loop playback"),
			cli.Float("max-idle", "i").
				Default(0.0).
				Help("Max idle time (0 = no limit)"),
		).
		Run(func(ctx *cli.Context) error {
			file := ctx.Arg(0)
			speed := ctx.Float64("speed")
			loop := ctx.Bool("loop")
			maxIdle := ctx.Float64("max-idle")

			return playRecording(file, speed, loop, maxIdle)
		})

	// Interactive TUI command
	app.Command("interactive").
		Alias("i").
		Description("Launch interactive TUI").
		Args("file?").
		Run(func(ctx *cli.Context) error {
			file := ctx.Arg(0)
			return runInteractiveTUI(file)
		})

	if err := app.Execute(); err != nil {
		if cli.IsHelpRequested(err) {
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(cli.GetExitCode(err))
	}
}

func recordSession(output, title, shell string, maxIdle float64, compress bool) error {
	// Determine shell
	if shell == "" {
		shell = os.Getenv("SHELL")
		if shell == "" {
			shell = "/bin/sh"
		}
	}

	// Add .cast extension if missing
	if !strings.HasSuffix(output, ".cast") && !strings.HasSuffix(output, ".cast.gz") {
		if compress {
			output += ".cast.gz"
		} else {
			output += ".cast"
		}
	}

	fmt.Printf("%s Recording to %s\n", color.Green.Apply("●"), output)
	fmt.Printf("  Shell: %s\n", shell)
	if title != "" {
		fmt.Printf("  Title: %s\n", title)
	}
	fmt.Printf("  Press %s to exit the shell and stop recording\n\n", color.Cyan.Apply("Ctrl+D"))

	// Create session
	session, err := termsession.NewSession(termsession.SessionOptions{
		Command: []string{shell},
	})
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	// Handle interrupt gracefully
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		session.Close()
	}()

	// Start recording
	opts := termsession.RecordingOptions{
		Compress:      compress,
		Title:         title,
		IdleTimeLimit: maxIdle,
	}

	startTime := time.Now()
	if err := session.Record(output, opts); err != nil {
		return fmt.Errorf("failed to start recording: %w", err)
	}

	// Wait for session to complete
	if err := session.Wait(); err != nil {
		// Session ended (possibly by user)
	}

	duration := time.Since(startTime)
	fmt.Printf("\n%s Recording saved to %s (%s)\n",
		color.Green.Apply("✓"),
		output,
		humanize.Duration(duration))

	return nil
}

func exportToGIF(input, output string, opts gif.CastOptions) error {
	// Show export settings
	fmt.Printf("%s Exporting to GIF\n", color.Blue.Apply("●"))
	fmt.Printf("  Input:    %s\n", input)
	fmt.Printf("  Output:   %s\n", output)
	fmt.Printf("  Speed:    %.1fx\n", opts.Speed)
	fmt.Printf("  FPS:      %d\n", opts.FPS)
	fmt.Printf("  Max idle: %.1fs\n", opts.MaxIdle)
	fmt.Println()

	// Get info first
	info, err := gif.GetCastInfo(input)
	if err != nil {
		return fmt.Errorf("failed to read recording: %w", err)
	}

	fmt.Printf("  Recording: %dx%d, %.1fs duration, %d events\n",
		info.Width, info.Height, info.Duration, info.EventCount)

	// Render
	startTime := time.Now()
	g, err := gif.RenderCast(input, opts)
	if err != nil {
		return fmt.Errorf("failed to render: %w", err)
	}

	// Save
	if err := g.Save(output); err != nil {
		return fmt.Errorf("failed to save GIF: %w", err)
	}

	// Get file size
	stat, _ := os.Stat(output)
	size := "unknown"
	if stat != nil {
		size = humanize.Bytes(stat.Size())
	}

	fmt.Printf("\n%s Created %s (%d frames, %s) in %s\n",
		color.Green.Apply("✓"),
		output,
		g.FrameCount(),
		size,
		humanize.Duration(time.Since(startTime)))

	return nil
}

func showInfo(file string) error {
	header, events, err := termsession.LoadCastFile(file)
	if err != nil {
		return fmt.Errorf("failed to load recording: %w", err)
	}

	duration := termsession.Duration(events)
	outputEvents := termsession.OutputEvents(events)

	fmt.Printf("%s %s\n\n", color.Blue.Apply("●"), file)

	if header.Title != "" {
		fmt.Printf("  Title:      %s\n", header.Title)
	}
	fmt.Printf("  Dimensions: %dx%d\n", header.Width, header.Height)
	fmt.Printf("  Duration:   %s (%.2fs)\n", humanize.Duration(time.Duration(duration*float64(time.Second))), duration)
	fmt.Printf("  Events:     %d total (%d output, %d input)\n",
		len(events), len(outputEvents), len(events)-len(outputEvents))
	fmt.Printf("  Recorded:   %s\n", time.Unix(header.Timestamp, 0).Format("2006-01-02 15:04:05"))
	fmt.Printf("  Version:    %d\n", header.Version)

	if len(header.Env) > 0 {
		fmt.Printf("\n  Environment:\n")
		for k, v := range header.Env {
			fmt.Printf("    %s=%s\n", k, v)
		}
	}

	return nil
}

func playRecording(file string, speed float64, loop bool, maxIdle float64) error {
	player, err := termsession.NewPlayer(file, termsession.PlayerOptions{
		Speed:   speed,
		Loop:    loop,
		MaxIdle: maxIdle,
		Output:  os.Stdout,
	})
	if err != nil {
		return fmt.Errorf("failed to load recording: %w", err)
	}

	header := player.GetHeader()
	duration := player.GetDuration()

	// Clear screen and show info
	fmt.Print("\033[2J\033[H")
	fmt.Printf("Playing: %s", file)
	if header.Title != "" {
		fmt.Printf(" - %s", header.Title)
	}
	fmt.Printf("\n")
	fmt.Printf("Size: %dx%d | Duration: %s | Speed: %.1fx\n\n",
		header.Width, header.Height,
		humanize.Duration(time.Duration(duration*float64(time.Second))),
		speed)

	return player.Play()
}

// InteractiveApp is the TUI for the interactive mode
type InteractiveApp struct {
	files        []string
	selected     int
	currentFile  string
	info         *gif.CastInfo
	statusMsg    string
	exportOpts   gif.CastOptions
	width        int
	height       int
	scrollOffset int
}

func runInteractiveTUI(initialFile string) error {
	app := &InteractiveApp{
		exportOpts: gif.DefaultCastOptions(),
	}

	// Find .cast files in current directory
	entries, err := os.ReadDir(".")
	if err == nil {
		for _, e := range entries {
			name := e.Name()
			if strings.HasSuffix(name, ".cast") || strings.HasSuffix(name, ".cast.gz") {
				app.files = append(app.files, name)
			}
		}
	}

	// If initial file provided, select it
	if initialFile != "" {
		for i, f := range app.files {
			if f == initialFile {
				app.selected = i
				break
			}
		}
		app.loadInfo(initialFile)
	} else if len(app.files) > 0 {
		app.loadInfo(app.files[0])
	}

	app.statusMsg = "↑↓ select | Enter play | e export | q quit"

	return tui.Run(app)
}

func (app *InteractiveApp) loadInfo(file string) {
	app.currentFile = file
	info, err := gif.GetCastInfo(file)
	if err != nil {
		app.statusMsg = fmt.Sprintf("Error: %v", err)
		app.info = nil
		return
	}
	app.info = info
}

func (app *InteractiveApp) HandleEvent(event tui.Event) []tui.Cmd {
	switch e := event.(type) {
	case tui.ResizeEvent:
		app.width = e.Width
		app.height = e.Height

	case tui.KeyEvent:
		if e.Rune == 'q' || e.Rune == 'Q' || e.Key == tui.KeyCtrlC || e.Key == tui.KeyEscape {
			return []tui.Cmd{tui.Quit()}
		}

		switch e.Key {
		case tui.KeyArrowUp:
			if app.selected > 0 {
				app.selected--
				if len(app.files) > 0 {
					app.loadInfo(app.files[app.selected])
				}
			}
		case tui.KeyArrowDown:
			if app.selected < len(app.files)-1 {
				app.selected++
				app.loadInfo(app.files[app.selected])
			}
		case tui.KeyEnter:
			if len(app.files) > 0 {
				return app.playSelected()
			}
		}

		switch e.Rune {
		case 'e', 'E':
			if len(app.files) > 0 {
				return app.exportSelected()
			}
		case 'r', 'R':
			if len(app.files) > 0 {
				app.loadInfo(app.files[app.selected])
			}
		}
	}
	return nil
}

func (app *InteractiveApp) View() tui.View {
	if len(app.files) == 0 {
		return tui.Stack(
			tui.HeaderBar("termrec - Terminal Recorder").Bg(tui.ColorMagenta).Fg(tui.ColorWhite),
			tui.Spacer(),
			tui.Text("No .cast files found in current directory").Fg(tui.ColorYellow),
			tui.Spacer().MinHeight(1),
			tui.Text("Use 'termrec record <output.cast>' to create a recording").Fg(tui.ColorBrightBlack),
			tui.Spacer(),
			tui.StatusBar("Press 'q' to quit"),
		)
	}

	// File list
	items := make([]tui.ListItem, len(app.files))
	for i, f := range app.files {
		items[i] = tui.ListItem{
			Label: f,
			Icon:  "🎬",
			Value: f,
		}
	}

	// Info panel
	var infoViews []tui.View
	if app.info != nil {
		title := app.info.Title
		if title == "" {
			title = "(no title)"
		}
		infoViews = []tui.View{
			tui.Text("Title:").Bold(),
			tui.Text("  %s", title).Fg(tui.ColorYellow),
			tui.Spacer().MinHeight(1),
			tui.Text("Dimensions:").Bold(),
			tui.Text("  %d × %d", app.info.Width, app.info.Height),
			tui.Spacer().MinHeight(1),
			tui.Text("Duration:").Bold(),
			tui.Text("  %s", humanize.Duration(time.Duration(app.info.Duration*float64(time.Second)))).Fg(tui.ColorGreen),
			tui.Spacer().MinHeight(1),
			tui.Text("Events:").Bold(),
			tui.Text("  %d", app.info.EventCount),
			tui.Spacer().MinHeight(1),
			tui.Text("Recorded:").Bold(),
			tui.Text("  %s", time.Unix(app.info.Timestamp, 0).Format("2006-01-02 15:04")),
		}
	} else {
		infoViews = []tui.View{
			tui.Text("Select a file to view info").Dim(),
		}
	}

	listHeight := app.height - 6
	if listHeight < 5 {
		listHeight = 5
	}

	return tui.Stack(
		tui.HeaderBar("termrec - Terminal Recorder").Bg(tui.ColorMagenta).Fg(tui.ColorWhite),
		tui.Group(
			tui.Stack(
				tui.Bordered(
					tui.FilterableList(items, &app.selected).
						Height(listHeight).
						SelectedFg(tui.ColorBlack).
						SelectedBg(tui.ColorMagenta).
						ScrollOffset(&app.scrollOffset),
				).Title("Recordings").BorderFg(tui.ColorMagenta),
			),
			tui.Stack(
				tui.Bordered(
					tui.Stack(infoViews...).Padding(1),
				).Title("Info").BorderFg(tui.ColorCyan),
			),
		),
		tui.StatusBar(app.statusMsg),
	)
}

func (app *InteractiveApp) playSelected() []tui.Cmd {
	if app.selected >= len(app.files) {
		return nil
	}

	file := app.files[app.selected]

	return []tui.Cmd{
		tui.Quit(),
		func() tui.Event {
			time.Sleep(100 * time.Millisecond)
			fmt.Print("\033[2J\033[H")

			if err := playRecording(file, 1.0, false, 0.0); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			}

			return tui.QuitEvent{Time: time.Now()}
		},
	}
}

func (app *InteractiveApp) exportSelected() []tui.Cmd {
	if app.selected >= len(app.files) {
		return nil
	}

	file := app.files[app.selected]
	output := strings.TrimSuffix(file, filepath.Ext(file))
	output = strings.TrimSuffix(output, ".cast") + ".gif"

	return []tui.Cmd{
		tui.Quit(),
		func() tui.Event {
			time.Sleep(100 * time.Millisecond)
			fmt.Print("\033[2J\033[H")

			if err := exportToGIF(file, output, app.exportOpts); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			}

			fmt.Println("\nPress Enter to continue...")
			var input string
			fmt.Scanln(&input)

			return tui.QuitEvent{Time: time.Now()}
		},
	}
}

// Background context for signal handling
var _ context.Context = context.Background()
