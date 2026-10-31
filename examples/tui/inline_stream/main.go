// Example: inline_stream
//
// This example streams log lines into scrollback while a spinner and an
// input line stay at the bottom of the terminal, as an agent CLI does.
//
// InlineApp.Print inserts each line above the live region without
// redrawing it. Scroll back after the stream ends, or resize the window
// while it runs: every line is there, in order.
//
// Keys:
//   - Enter: stream 500 more lines (the text you typed is printed first)
//   - Ctrl+C: quit
//
// Run with: go run ./examples/tui/inline_stream
//
// Flags: -lines sets how many lines Enter streams (500), and -delay the
// pause between them (2ms). For slow output: -lines 20 -delay 500ms.
package main

import (
	"flag"
	"log"
	"time"

	"github.com/deepnoodle-ai/wonton/tui"
)

var (
	batch = 500
	delay = 2 * time.Millisecond
)

type streamApp struct {
	runner  *tui.InlineApp
	input   string
	next    int // number of the next line to print
	remain  int // lines left in the current stream
	started time.Time
}

type lineEvent struct{ time time.Time }

func (e lineEvent) Timestamp() time.Time { return e.time }

// nextLine asks for the next line after a short pause.
func nextLine() tui.Cmd {
	return func() tui.Event {
		time.Sleep(delay)
		return lineEvent{time: time.Now()}
	}
}

func (app *streamApp) LiveView() tui.View {
	var status tui.View = tui.Text(" idle: press Enter to stream %d lines", batch).Dim()
	if app.remain > 0 {
		status = tui.Group(
			tui.Loading(uint64(app.next)).CharSet(tui.SpinnerDots.Frames).Speed(4).Fg(tui.ColorCyan),
			tui.Text(" streaming… %d lines left (%s)", app.remain, time.Since(app.started).Round(time.Millisecond)),
		)
	}
	return tui.Stack(
		status,
		tui.Divider(),
		tui.InputField(&app.input).ID("prompt").Label("> ").Placeholder("type a note, then Enter"),
		tui.Divider(),
		tui.Text("  Ctrl+C to quit").Dim(),
	)
}

func (app *streamApp) HandleEvent(event tui.Event) []tui.Cmd {
	switch e := event.(type) {
	case tui.KeyEvent:
		switch e.Key {
		case tui.KeyCtrlC:
			return []tui.Cmd{tui.Quit()}
		case tui.KeyEnter:
			if app.remain > 0 {
				return nil
			}
			if app.input != "" {
				app.runner.Print(tui.Text("> %s", app.input).Bold())
				app.input = ""
			}
			app.remain = batch
			app.started = time.Now()
			return []tui.Cmd{nextLine()}
		}
	case lineEvent:
		if app.remain == 0 {
			return nil
		}
		color := []tui.Color{tui.ColorGreen, tui.ColorBlue, tui.ColorYellow}[app.next%3]
		app.runner.Print(tui.Group(
			tui.Text("%04d ", app.next).Fg(color),
			tui.Text("compiled package ./pkg/mod%03d in %dms", app.next%997, 3+app.next%40),
		))
		app.next++
		app.remain--
		if app.remain == 0 {
			app.runner.Printf("streamed %d lines in %s", batch, time.Since(app.started).Round(time.Millisecond))
			return nil
		}
		return []tui.Cmd{nextLine()}
	}
	return nil
}

func main() {
	flag.IntVar(&batch, "lines", batch, "lines to stream when Enter is pressed")
	flag.DurationVar(&delay, "delay", delay, "pause between streamed lines")
	flag.Parse()
	app := &streamApp{}
	app.runner = tui.NewInlineApp(tui.WithInlineFPS(20))
	if err := app.runner.Run(app); err != nil {
		log.Fatal(err)
	}
}
