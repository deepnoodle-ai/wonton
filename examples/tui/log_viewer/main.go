package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/deepnoodle-ai/wonton/tui"
)

// LogViewerApp demonstrates a practical log viewer with text selection.
// This shows how selection can be used in a real application scenario.
type LogViewerApp struct {
	logs       string
	selection  tui.TextSelection
	scrollY    int
	lineCount  int
	copied     bool
	copiedTime time.Time
}

func (app *LogViewerApp) Init() error {
	// Generate sample log entries
	var logs strings.Builder
	levels := []string{"INFO", "DEBUG", "WARN", "ERROR", "INFO", "INFO", "DEBUG"}
	messages := []string{
		"Application started successfully",
		"Loading configuration from /etc/app/config.yaml",
		"Database connection established: postgres://db:5432/app",
		"Cache warmed up with 1,523 entries",
		"HTTP server listening on :8080",
		"Request received: GET /api/users",
		"Query executed in 12ms: SELECT * FROM users LIMIT 100",
		"Response sent: 200 OK (245 bytes)",
		"Request received: POST /api/orders",
		"Validation passed for order #12345",
		"Payment processed: $99.99 via Stripe",
		"Order created successfully: #12345",
		"Email queued: order_confirmation to user@example.com",
		"Request received: GET /api/health",
		"Health check passed: all services operational",
		"Slow query detected (>100ms): complex aggregation",
		"Memory usage: 256MB / 512MB (50%)",
		"Goroutines active: 42",
		"Connection pool: 8/20 connections in use",
		"Background job completed: cleanup_old_sessions",
		"Request received: DELETE /api/cache",
		"Cache invalidated: 89 entries removed",
		"Rate limit approaching for IP 192.168.1.100",
		"Authentication failed: invalid token",
		"Retry attempt 1/3 for external API call",
		"External API responded: 200 OK",
		"Webhook delivered successfully to https://hooks.example.com",
		"Scheduled task started: daily_report_generation",
		"Report generated: 2,341 rows exported to CSV",
		"File uploaded to S3: reports/daily/2024-01-15.csv",
	}

	baseTime := time.Now().Add(-30 * time.Minute)
	for i := 0; i < 30; i++ {
		timestamp := baseTime.Add(time.Duration(i) * time.Minute)
		level := levels[i%len(levels)]
		message := messages[i%len(messages)]
		logs.WriteString(fmt.Sprintf("[%s] %s: %s\n",
			timestamp.Format("15:04:05"),
			level,
			message))
	}

	app.logs = strings.TrimSuffix(logs.String(), "\n")
	app.lineCount = strings.Count(app.logs, "\n") + 1
	return nil
}

func (app *LogViewerApp) HandleEvent(event tui.Event) []tui.Cmd {
	switch e := event.(type) {
	case tui.KeyEvent:
		switch e.Key {
		case tui.KeyCtrlQ, tui.KeyCtrlC:
			if e.Key == tui.KeyCtrlC && app.selection.Active && !app.selection.IsEmpty() {
				app.copied = true
				app.copiedTime = time.Now()
				return nil
			}
			if e.Key == tui.KeyCtrlQ {
				return []tui.Cmd{tui.Quit()}
			}
		case tui.KeyEscape:
			app.selection.Clear()
		}

	case tui.MouseEvent:
		tui.TextAreaHandleMouseEvent("logs", &e)

	case tui.TickEvent:
		// Clear "copied" message after 2 seconds
		if app.copied && time.Since(app.copiedTime) > 2*time.Second {
			app.copied = false
		}
	}
	return nil
}

func (app *LogViewerApp) View() tui.View {
	// Build status line
	var status string
	if app.copied {
		status = "Copied to clipboard!"
	} else if app.selection.Active && !app.selection.IsEmpty() {
		start, end := app.selection.Normalized()
		lineCount := end.Line - start.Line + 1
		if lineCount == 1 {
			status = fmt.Sprintf("1 line selected (Ctrl+C to copy)")
		} else {
			status = fmt.Sprintf("%d lines selected (Ctrl+C to copy)", lineCount)
		}
	} else {
		status = "Select log entries to copy"
	}

	statusColor := tui.ColorBrightBlack
	if app.copied {
		statusColor = tui.ColorGreen
	} else if app.selection.Active {
		statusColor = tui.ColorYellow
	}

	return tui.Stack(
		// Header
		tui.Group(
			tui.Text("Log Viewer").Bold().Fg(tui.ColorCyan),
			tui.Spacer(),
			tui.Text("%d lines", app.lineCount).Fg(tui.ColorBrightBlack),
		),
		tui.Text("Ctrl+Q: quit | Double-click: word | Triple-click: line").
			Fg(tui.ColorBrightBlack),
		tui.Spacer().MinHeight(1),

		// Status
		tui.Text("%s", status).Fg(statusColor),
		tui.Spacer().MinHeight(1),

		// Log viewer
		tui.TextArea(&app.logs).
			ID("logs").
			LeftBorderOnly().
			BorderFg(tui.ColorBrightBlack).
			FocusBorderFg(tui.ColorCyan).
			LineNumbers(true).
			LineNumberFg(tui.ColorBrightBlack).
			EnableSelection().
			Selection(&app.selection).
			ScrollY(&app.scrollY).
			Size(80, 20),

		tui.Spacer().MinHeight(1),

		// Footer with scroll position
		tui.Text("Scroll: line %d | Arrow keys to scroll", app.scrollY+1).
			Fg(tui.ColorBrightBlack),
	)
}

func main() {
	err := tui.Run(&LogViewerApp{}, tui.WithFPS(30), tui.WithMouseTracking(true))
	if err != nil {
		log.Fatal(err)
	}
}
