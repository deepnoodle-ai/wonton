// Example: sseview - Server-Sent Events stream viewer
//
// A TUI for viewing SSE event streams in real-time. Perfect for debugging
// AI streaming APIs (OpenAI, Anthropic), webhooks, and real-time data feeds.
//
// Run with:
//
//	go run ./examples/sseview https://example.com/events
//	go run ./examples/sseview --header "Authorization: Bearer token" https://api.example.com/stream
//	go run ./examples/sseview --json https://api.openai.com/v1/chat/completions
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/deepnoodle-ai/wonton/cli"
	"github.com/deepnoodle-ai/wonton/humanize"
	"github.com/deepnoodle-ai/wonton/retry"
	"github.com/deepnoodle-ai/wonton/sse"
	"github.com/deepnoodle-ai/wonton/tui"
)

// SSEEvent represents a received event with metadata
type SSEEvent struct {
	Event     sse.Event
	Timestamp time.Time
	Index     int
}

// SSEViewApp is the TUI application state
type SSEViewApp struct {
	mu sync.Mutex

	// Connection state
	url         string
	headers     map[string]string
	connected   bool
	connecting  bool
	error       error
	startTime   time.Time
	lastEventAt time.Time

	// Events
	events       []SSEEvent
	maxEvents    int
	totalEvents  int
	scrollOffset int
	selected     int
	autoScroll   bool

	// Display options
	prettyJSON bool
	showRaw    bool
	width      int
	height     int

	// Control
	cancel context.CancelFunc
}

func main() {
	app := cli.New("sseview").
		Description("View Server-Sent Events streams in real-time").
		Version("1.0.0")

	app.Main().
		Args("url").
		Flags(
			cli.String("header", "H").
				Help("Add header (format: 'Name: Value') - use multiple times for multiple headers"),
			cli.Bool("json", "j").
				Default(true).
				Help("Pretty-print JSON data"),
			cli.Bool("raw", "r").
				Help("Show raw event data"),
			cli.Int("max-events", "m").
				Default(100).
				Help("Maximum events to keep in buffer"),
			cli.Bool("reconnect", "").
				Default(true).
				Help("Auto-reconnect on disconnect"),
			cli.Int("timeout", "t").
				Default(30).
				Help("Connection timeout in seconds"),
		).
		Run(func(ctx *cli.Context) error {
			url := ctx.Arg(0)
			if url == "" {
				return cli.Error("URL is required").
					Hint("Usage: sseview https://example.com/events")
			}

			// Parse header (single header for simplicity)
			headers := make(map[string]string)
			headerStr := ctx.String("header")
			if headerStr != "" {
				parts := strings.SplitN(headerStr, ":", 2)
				if len(parts) == 2 {
					headers[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
				}
			}

			tuiApp := &SSEViewApp{
				url:        url,
				headers:    headers,
				prettyJSON: ctx.Bool("json"),
				showRaw:    ctx.Bool("raw"),
				maxEvents:  ctx.Int("max-events"),
				autoScroll: true,
				startTime:  time.Now(),
			}

			// Start connection in background
			connCtx, cancel := context.WithCancel(context.Background())
			tuiApp.cancel = cancel

			go tuiApp.connect(connCtx, ctx.Bool("reconnect"), ctx.Int("timeout"))

			// Run TUI
			if err := tui.Run(tuiApp); err != nil {
				cancel()
				return err
			}

			cancel()
			return nil
		})

	if err := app.Execute(); err != nil {
		if cli.IsHelpRequested(err) {
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(cli.GetExitCode(err))
	}
}

func (app *SSEViewApp) connect(ctx context.Context, reconnect bool, timeout int) {
	app.mu.Lock()
	app.connecting = true
	app.mu.Unlock()

	client := sse.NewClient(app.url)
	client.HTTPClient = &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
	}

	// Set headers
	for k, v := range app.headers {
		client.Headers.Set(k, v)
	}

	connectFn := func() error {
		app.mu.Lock()
		app.connecting = true
		app.connected = false
		app.error = nil
		app.mu.Unlock()

		events, errs := client.Connect(ctx)

		app.mu.Lock()
		app.connecting = false
		app.connected = true
		app.mu.Unlock()

		// Process events
		for event := range events {
			app.mu.Lock()
			app.totalEvents++
			app.lastEventAt = time.Now()

			sseEvent := SSEEvent{
				Event:     event,
				Timestamp: time.Now(),
				Index:     app.totalEvents,
			}

			app.events = append(app.events, sseEvent)
			if len(app.events) > app.maxEvents {
				app.events = app.events[1:]
			}

			// Auto-scroll to bottom
			if app.autoScroll {
				app.selected = len(app.events) - 1
			}

			app.mu.Unlock()
		}

		// Check for errors
		if err := <-errs; err != nil {
			app.mu.Lock()
			app.connected = false
			app.error = err
			app.mu.Unlock()
			return err
		}

		return nil
	}

	if reconnect {
		// Retry with backoff
		_ = retry.DoSimple(ctx, connectFn,
			retry.WithMaxAttempts(0), // Unlimited retries
			retry.WithBackoff(time.Second, 30*time.Second),
		)
	} else {
		if err := connectFn(); err != nil {
			app.mu.Lock()
			app.error = err
			app.mu.Unlock()
		}
	}

	app.mu.Lock()
	app.connected = false
	app.connecting = false
	app.mu.Unlock()
}

func (app *SSEViewApp) HandleEvent(event tui.Event) []tui.Cmd {
	switch e := event.(type) {
	case tui.ResizeEvent:
		app.width = e.Width
		app.height = e.Height

	case tui.KeyEvent:
		// Quit
		if e.Rune == 'q' || e.Rune == 'Q' || e.Key == tui.KeyCtrlC || e.Key == tui.KeyEscape {
			if app.cancel != nil {
				app.cancel()
			}
			return []tui.Cmd{tui.Quit()}
		}

		app.mu.Lock()
		defer app.mu.Unlock()

		// Calculate page size for navigation
		listHeight := app.height - 10
		if listHeight < 5 {
			listHeight = 5
		}

		// Navigation (less-compatible)
		switch e.Key {
		case tui.KeyArrowUp:
			if app.selected > 0 {
				app.selected--
				app.autoScroll = false
			}
		case tui.KeyArrowDown:
			if app.selected < len(app.events)-1 {
				app.selected++
			}
			if app.selected == len(app.events)-1 {
				app.autoScroll = true
			}
		case tui.KeyHome:
			app.selected = 0
			app.autoScroll = false
		case tui.KeyEnd:
			if len(app.events) > 0 {
				app.selected = len(app.events) - 1
			}
			app.autoScroll = true
		case tui.KeyPageUp, tui.KeyCtrlB:
			app.selected -= listHeight
			if app.selected < 0 {
				app.selected = 0
			}
			app.autoScroll = false
		case tui.KeyPageDown, tui.KeyCtrlF:
			app.selected += listHeight
			if app.selected >= len(app.events) {
				app.selected = len(app.events) - 1
			}
			if app.selected == len(app.events)-1 {
				app.autoScroll = true
			}
		case tui.KeyCtrlD:
			// Half page down
			app.selected += listHeight / 2
			if app.selected >= len(app.events) {
				app.selected = len(app.events) - 1
			}
			if app.selected == len(app.events)-1 {
				app.autoScroll = true
			}
		case tui.KeyCtrlU:
			// Half page up
			app.selected -= listHeight / 2
			if app.selected < 0 {
				app.selected = 0
			}
			app.autoScroll = false
		}

		// Less-compatible rune keys (note: 'j' is used for JSON toggle)
		switch e.Rune {
		case 'k':
			// Up one line
			if app.selected > 0 {
				app.selected--
				app.autoScroll = false
			}
		case ' ', 'f':
			// Page down
			app.selected += listHeight
			if app.selected >= len(app.events) {
				app.selected = len(app.events) - 1
			}
			if app.selected == len(app.events)-1 {
				app.autoScroll = true
			}
		case 'b':
			// Page up
			app.selected -= listHeight
			if app.selected < 0 {
				app.selected = 0
			}
			app.autoScroll = false
		case 'd':
			// Half page down
			app.selected += listHeight / 2
			if app.selected >= len(app.events) {
				app.selected = len(app.events) - 1
			}
			if app.selected == len(app.events)-1 {
				app.autoScroll = true
			}
		case 'u':
			// Half page up
			app.selected -= listHeight / 2
			if app.selected < 0 {
				app.selected = 0
			}
			app.autoScroll = false
		case 'g':
			// Go to top
			app.selected = 0
			app.autoScroll = false
		case 'G':
			// Go to bottom
			if len(app.events) > 0 {
				app.selected = len(app.events) - 1
			}
			app.autoScroll = true
		}

		// Commands
		switch e.Rune {
		case 'j':
			app.prettyJSON = !app.prettyJSON
		case 'r':
			app.showRaw = !app.showRaw
		case 'a':
			app.autoScroll = !app.autoScroll
		case 'c':
			app.events = nil
			app.selected = 0
		}
	}

	return nil
}

func (app *SSEViewApp) View() tui.View {
	app.mu.Lock()
	defer app.mu.Unlock()

	// Header with connection status
	var statusIcon string
	var statusText string

	if app.connecting {
		statusIcon = "◌"
		statusText = "Connecting..."
	} else if app.connected {
		statusIcon = "●"
		statusText = "Connected"
	} else if app.error != nil {
		statusIcon = "✗"
		statusText = fmt.Sprintf("Error: %v", app.error)
	} else {
		statusIcon = "○"
		statusText = "Disconnected"
	}

	header := tui.HeaderBar(fmt.Sprintf("SSE Viewer  %s %s  [%d events]",
		statusIcon, statusText, app.totalEvents)).
		Bg(tui.ColorBlue).
		Fg(tui.ColorWhite)

	// URL bar
	urlBar := tui.Text(" %s", app.url).Fg(tui.ColorBrightBlack).MaxWidth(app.width - 4)

	// Event list
	var eventViews []tui.View

	if len(app.events) == 0 {
		eventViews = append(eventViews,
			tui.Text("Waiting for events...").Fg(tui.ColorBrightBlack))
	} else {
		// Calculate visible range
		listHeight := app.height - 10
		if listHeight < 5 {
			listHeight = 5
		}

		start := app.scrollOffset
		end := start + listHeight
		if end > len(app.events) {
			end = len(app.events)
		}

		// Adjust scroll offset to keep selected visible
		if app.selected < start {
			app.scrollOffset = app.selected
			start = app.scrollOffset
			end = start + listHeight
			if end > len(app.events) {
				end = len(app.events)
			}
		} else if app.selected >= end {
			app.scrollOffset = app.selected - listHeight + 1
			if app.scrollOffset < 0 {
				app.scrollOffset = 0
			}
			start = app.scrollOffset
			end = start + listHeight
			if end > len(app.events) {
				end = len(app.events)
			}
		}

		for i := start; i < end; i++ {
			evt := app.events[i]
			eventViews = append(eventViews, app.formatEvent(evt, i == app.selected))
		}
	}

	// Detail panel (selected event)
	var detailViews []tui.View
	if app.selected >= 0 && app.selected < len(app.events) {
		evt := app.events[app.selected]
		detailViews = app.formatEventDetail(evt)
	} else {
		detailViews = []tui.View{
			tui.Text("Select an event to view details").Fg(tui.ColorBrightBlack),
		}
	}

	// Stats bar
	elapsed := time.Since(app.startTime)
	var lastEventAgo string
	if !app.lastEventAt.IsZero() {
		lastEventAgo = humanize.Duration(time.Since(app.lastEventAt)) + " ago"
	} else {
		lastEventAgo = "-"
	}

	statsBar := tui.Group(
		tui.Text("Elapsed: %s", humanize.Duration(elapsed)).Fg(tui.ColorBrightBlack),
		tui.Spacer().MinWidth(2),
		tui.Text("Last: %s", lastEventAgo).Fg(tui.ColorBrightBlack),
		tui.Spacer(),
		tui.Text("Auto-scroll: ").Fg(tui.ColorBrightBlack),
		tui.Text("%v", app.autoScroll).Fg(tui.ColorCyan),
	)

	// Help
	helpText := "↑↓ navigate | j toggle JSON | r toggle raw | c clear | a auto-scroll | q quit"

	return tui.Stack(
		header,
		urlBar,
		tui.Spacer().MinHeight(1),
		tui.Group(
			// Event list
			tui.Stack(
				tui.Bordered(
					tui.Stack(eventViews...),
				).Title("Events").BorderFg(tui.ColorCyan),
			),
			// Detail panel
			tui.Stack(
				tui.Bordered(
					tui.Stack(detailViews...).Padding(1),
				).Title("Detail").BorderFg(tui.ColorYellow),
			),
		),
		statsBar,
		tui.StatusBar(helpText),
	)
}

func (app *SSEViewApp) formatEvent(evt SSEEvent, selected bool) tui.View {
	// Event type
	eventType := evt.Event.Event
	if eventType == "" {
		eventType = "message"
	}

	// Truncate data for list view
	data := evt.Event.Data
	if len(data) > 60 {
		data = data[:57] + "..."
	}
	data = strings.ReplaceAll(data, "\n", "↵")

	// Time
	timeStr := evt.Timestamp.Format("15:04:05")

	var bg tui.Color
	var fg tui.Color
	if selected {
		bg = tui.ColorCyan
		fg = tui.ColorBlack
	} else {
		bg = tui.ColorDefault
		fg = tui.ColorWhite
	}

	return tui.Group(
		tui.Text(" %s ", timeStr).Fg(tui.ColorBrightBlack).Bg(bg),
		tui.Text(" %s ", eventType).Fg(tui.ColorYellow).Bg(bg).Bold(),
		tui.Text(" %s", data).Fg(fg).Bg(bg),
	)
}

func (app *SSEViewApp) formatEventDetail(evt SSEEvent) []tui.View {
	var views []tui.View

	// Metadata
	eventType := evt.Event.Event
	if eventType == "" {
		eventType = "message"
	}

	views = append(views,
		tui.Text("Event #%d", evt.Index).Bold().Fg(tui.ColorCyan),
		tui.Spacer().MinHeight(1),
		tui.Text("Type: %s", eventType).Fg(tui.ColorYellow),
		tui.Text("Time: %s", evt.Timestamp.Format("15:04:05.000")),
	)

	if evt.Event.ID != "" {
		views = append(views, tui.Text("ID: %s", evt.Event.ID).Fg(tui.ColorBrightBlack))
	}
	if evt.Event.Retry > 0 {
		views = append(views, tui.Text("Retry: %dms", evt.Event.Retry).Fg(tui.ColorBrightBlack))
	}

	views = append(views, tui.Spacer().MinHeight(1))

	// Data
	views = append(views, tui.Text("Data:").Bold())

	data := evt.Event.Data
	if app.prettyJSON && !app.showRaw {
		// Try to pretty-print JSON
		var jsonData any
		if err := json.Unmarshal([]byte(data), &jsonData); err == nil {
			pretty, err := json.MarshalIndent(jsonData, "", "  ")
			if err == nil {
				data = string(pretty)
			}
		}
	}

	// Split data into lines
	lines := strings.Split(data, "\n")
	for _, line := range lines {
		if len(line) > 80 {
			line = line[:77] + "..."
		}
		views = append(views, tui.Text("  %s", line).Fg(tui.ColorWhite))
	}

	return views
}
