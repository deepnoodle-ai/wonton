// Example: Session Viewer
//
// A TUI application for browsing and replaying asciinema-format terminal
// recordings. Navigate sessions, search commands, and replay at variable
// speed with proper styling.
//
// Run with:
//
//	go run examples/sessview/main.go play <recording.cast>
//	go run examples/sessview/main.go play --speed 2.0 session.cast
//	go run examples/sessview/main.go browse .
//	go run examples/sessview/main.go info session.cast
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/deepnoodle-ai/wonton/cli"
	"github.com/deepnoodle-ai/wonton/humanize"
	"github.com/deepnoodle-ai/wonton/termsession"
	"github.com/deepnoodle-ai/wonton/tui"
)

func main() {
	app := cli.New("sessview").
		Description("Browse and replay terminal session recordings").
		Version("0.1.0")

	// Play command - replay a recording
	app.Command("play").
		Description("Play a terminal recording").
		Args("file").
		Flags(
			cli.Float("speed", "s").
				Default(1.0).
				Help("Playback speed multiplier"),
			cli.Bool("loop", "l").
				Help("Loop playback"),
			cli.Float("max-idle", "m").
				Default(0.0).
				Help("Maximum idle time between events (seconds)"),
		).
		Run(func(ctx *cli.Context) error {
			filename := ctx.Arg(0)
			speed := ctx.Float64("speed")
			loop := ctx.Bool("loop")
			maxIdle := ctx.Float64("max-idle")

			return playRecording(filename, speed, loop, maxIdle)
		})

	// Browse command - interactive TUI browser
	app.Command("browse").
		Description("Browse recordings in a directory").
		Args("directory?").
		Run(func(ctx *cli.Context) error {
			dir := ctx.Arg(0)
			if dir == "" {
				dir = "."
			}
			return browseRecordings(dir)
		})

	// Info command - show recording metadata
	app.Command("info").
		Description("Show recording information").
		Args("file").
		Run(func(ctx *cli.Context) error {
			filename := ctx.Arg(0)
			return showInfo(filename)
		})

	if err := app.Execute(); err != nil {
		if cli.IsHelpRequested(err) {
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// playRecording plays back a recording with the specified options
func playRecording(filename string, speed float64, loop bool, maxIdle float64) error {
	player, err := termsession.NewPlayer(filename, termsession.PlayerOptions{
		Speed:   speed,
		Loop:    loop,
		MaxIdle: maxIdle,
		Output:  os.Stdout,
	})
	if err != nil {
		return fmt.Errorf("failed to load recording: %w", err)
	}

	// Show header info
	header := player.GetHeader()
	fmt.Printf("\033[2J\033[H") // Clear screen
	fmt.Printf("Playing: %s\n", filename)
	if header.Title != "" {
		fmt.Printf("Title: %s\n", header.Title)
	}
	fmt.Printf("Size: %dx%d\n", header.Width, header.Height)
	fmt.Printf("Duration: %s\n", humanize.Duration(time.Duration(player.GetDuration()*float64(time.Second))))
	fmt.Printf("Speed: %.1fx\n", speed)
	if loop {
		fmt.Printf("Loop: enabled\n")
	}
	fmt.Printf("\n")

	// Start playback
	return player.Play()
}

// showInfo displays metadata about a recording
func showInfo(filename string) error {
	header, events, err := termsession.LoadCastFile(filename)
	if err != nil {
		return fmt.Errorf("failed to load recording: %w", err)
	}

	fmt.Printf("File: %s\n", filename)
	if header.Title != "" {
		fmt.Printf("Title: %s\n", header.Title)
	}
	fmt.Printf("Dimensions: %dx%d\n", header.Width, header.Height)
	fmt.Printf("Version: %d\n", header.Version)
	fmt.Printf("Timestamp: %s\n", time.Unix(header.Timestamp, 0).Format(time.RFC3339))

	duration := termsession.Duration(events)
	fmt.Printf("Duration: %s (%.2fs)\n", humanize.Duration(time.Duration(duration*float64(time.Second))), duration)
	fmt.Printf("Events: %d\n", len(events))

	outputEvents := termsession.OutputEvents(events)
	fmt.Printf("Output events: %d\n", len(outputEvents))
	fmt.Printf("Input events: %d\n", len(events)-len(outputEvents))

	if len(header.Env) > 0 {
		fmt.Printf("\nEnvironment:\n")
		for k, v := range header.Env {
			fmt.Printf("  %s=%s\n", k, v)
		}
	}

	return nil
}

// browseRecordings opens an interactive browser for recordings in a directory
func browseRecordings(dir string) error {
	// Find all .cast files in the directory
	files, err := findCastFiles(dir)
	if err != nil {
		return err
	}

	if len(files) == 0 {
		return fmt.Errorf("no .cast files found in %s", dir)
	}

	app := &BrowserApp{
		directory: dir,
		files:     files,
	}

	return tui.Run(app)
}

// findCastFiles searches for .cast files in a directory
func findCastFiles(dir string) ([]string, error) {
	var files []string

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".cast") || strings.HasSuffix(name, ".cast.gz") {
			files = append(files, filepath.Join(dir, name))
		}
	}

	return files, nil
}

// BrowserApp is the TUI application for browsing recordings
type BrowserApp struct {
	directory    string
	files        []string
	selected     int
	scrollOffset int
	preview      *RecordingPreview
	width        int
	height       int
	statusMsg    string
}

// RecordingPreview holds preview information about a recording
type RecordingPreview struct {
	filename string
	header   *termsession.RecordingHeader
	duration float64
	events   int
}

// Init initializes the browser
func (app *BrowserApp) Init() error {
	if len(app.files) > 0 {
		app.loadPreview(app.files[app.selected])
	}
	app.statusMsg = "Use arrows to navigate, Enter to play, i for info, q to quit"
	return nil
}

// HandleEvent processes user input
func (app *BrowserApp) HandleEvent(event tui.Event) []tui.Cmd {
	switch e := event.(type) {
	case tui.ResizeEvent:
		app.width = e.Width
		app.height = e.Height

	case tui.KeyEvent:
		// Quit
		if e.Rune == 'q' || e.Rune == 'Q' || e.Key == tui.KeyCtrlC || e.Key == tui.KeyEscape {
			return []tui.Cmd{tui.Quit()}
		}

		// Navigation
		switch e.Key {
		case tui.KeyArrowUp:
			if app.selected > 0 {
				app.selected--
				app.loadPreview(app.files[app.selected])
			}
		case tui.KeyArrowDown:
			if app.selected < len(app.files)-1 {
				app.selected++
				app.loadPreview(app.files[app.selected])
			}
		case tui.KeyHome:
			app.selected = 0
			app.loadPreview(app.files[app.selected])
		case tui.KeyEnd:
			app.selected = len(app.files) - 1
			app.loadPreview(app.files[app.selected])
		case tui.KeyEnter:
			// Play the selected recording
			return app.playSelected()
		}

		// Commands
		switch e.Rune {
		case 'i', 'I':
			app.showInfoSelected()
		case 'r', 'R':
			app.loadPreview(app.files[app.selected])
			app.statusMsg = "Preview reloaded"
		}
	}

	return nil
}

// View renders the TUI
func (app *BrowserApp) View() tui.View {
	// Convert files to list items
	items := make([]tui.ListItem, len(app.files))
	for i, file := range app.files {
		name := filepath.Base(file)
		items[i] = tui.ListItem{
			Label: name,
			Icon:  "📼",
			Value: file,
		}
	}

	// Build preview content
	var previewLines []tui.View
	if app.preview != nil {
		title := app.preview.header.Title
		if title == "" {
			title = "(no title)"
		}

		previewLines = []tui.View{
			tui.Text("File:").Bold(),
			tui.Text("  %s", filepath.Base(app.preview.filename)).Fg(tui.ColorCyan),
			tui.Spacer().MinHeight(1),

			tui.Text("Title:").Bold(),
			tui.Text("  %s", title).Fg(tui.ColorYellow),
			tui.Spacer().MinHeight(1),

			tui.Text("Dimensions:").Bold(),
			tui.Text("  %dx%d", app.preview.header.Width, app.preview.header.Height),
			tui.Spacer().MinHeight(1),

			tui.Text("Duration:").Bold(),
			tui.Text("  %s", humanize.Duration(time.Duration(app.preview.duration*float64(time.Second)))).Fg(tui.ColorGreen),
			tui.Spacer().MinHeight(1),

			tui.Text("Events:").Bold(),
			tui.Text("  %d", app.preview.events),
			tui.Spacer().MinHeight(1),

			tui.Text("Recorded:").Bold(),
			tui.Text("  %s", time.Unix(app.preview.header.Timestamp, 0).Format("2006-01-02 15:04:05")),

			tui.Spacer(),
		}
	} else {
		previewLines = []tui.View{
			tui.Text("No preview available").Dim(),
			tui.Spacer(),
		}
	}

	// Calculate list height
	listHeight := app.height - 6
	if listHeight < 5 {
		listHeight = 5
	}

	return tui.Stack(
		// Header
		tui.HeaderBar("Session Viewer - Browse Recordings").
			Bg(tui.ColorBlue).
			Fg(tui.ColorWhite),

		// Main content area
		tui.Group(
			// File list panel
			tui.Stack(
				tui.Text(" Recordings in %s", app.directory),
				tui.Bordered(
					tui.FilterableList(items, &app.selected).
						Height(listHeight).
						SelectedFg(tui.ColorBlack).
						SelectedBg(tui.ColorCyan).
						ScrollOffset(&app.scrollOffset),
				).Title("Files").BorderFg(tui.ColorCyan),
			),

			// Preview panel
			tui.Stack(
				tui.Text(" Preview").Bold(),
				tui.Bordered(
					tui.Stack(previewLines...).Padding(1),
				).BorderFg(tui.ColorYellow),
			),
		),

		// Status bar
		tui.StatusBar(app.statusMsg),
	)
}

// loadPreview loads metadata for the selected file
func (app *BrowserApp) loadPreview(filename string) {
	header, events, err := termsession.LoadCastFile(filename)
	if err != nil {
		app.statusMsg = fmt.Sprintf("Error loading preview: %v", err)
		app.preview = nil
		return
	}

	app.preview = &RecordingPreview{
		filename: filename,
		header:   header,
		duration: termsession.Duration(events),
		events:   len(events),
	}
	app.statusMsg = fmt.Sprintf("Loaded preview for %s", filepath.Base(filename))
}

// playSelected plays the currently selected recording
func (app *BrowserApp) playSelected() []tui.Cmd {
	if app.selected >= len(app.files) {
		return nil
	}

	filename := app.files[app.selected]

	// Quit the TUI and then play
	return []tui.Cmd{
		tui.Quit(),
		func() tui.Event {
			// Small delay to ensure terminal is restored
			time.Sleep(100 * time.Millisecond)

			// Clear screen before playing
			fmt.Printf("\033[2J\033[H")

			if err := playRecording(filename, 1.0, false, 0.0); err != nil {
				fmt.Fprintf(os.Stderr, "Error playing recording: %v\n", err)
			}

			// Return a dummy event (we're quitting anyway)
			return tui.QuitEvent{Time: time.Now()}
		},
	}
}

// showInfoSelected shows detailed info for the selected recording
func (app *BrowserApp) showInfoSelected() {
	if app.selected >= len(app.files) {
		return
	}

	filename := app.files[app.selected]
	header, events, err := termsession.LoadCastFile(filename)
	if err != nil {
		app.statusMsg = fmt.Sprintf("Error: %v", err)
		return
	}

	// Build a detailed info message
	var info strings.Builder
	info.WriteString(fmt.Sprintf("File: %s | ", filepath.Base(filename)))
	if header.Title != "" {
		info.WriteString(fmt.Sprintf("Title: %s | ", header.Title))
	}
	info.WriteString(fmt.Sprintf("Size: %dx%d | ", header.Width, header.Height))
	info.WriteString(fmt.Sprintf("Duration: %s | ", humanize.Duration(time.Duration(termsession.Duration(events)*float64(time.Second)))))
	info.WriteString(fmt.Sprintf("Events: %d", len(events)))

	app.statusMsg = info.String()
}
