# TUI engine

The `tui` package provides Wonton's declarative terminal UI engine. It sits on
top of the `terminal` package and offers high-level layout primitives, ready-made
controls (tables, lists, inputs, markdown, file pickers), and an event-driven
runtime that keeps your application logic single-threaded.

## Mental model

- **Application**: implement `tui.Application` by providing a `View() tui.View`.
  Optionally implement `tui.EventHandler` to respond to keyboard, mouse, tick,
  and custom events.
- **Views**: compose UI using helpers like `tui.VStack`, `tui.HStack`,
  `tui.Text`, `tui.Table`, `tui.SelectList`, `tui.Markdown`, and `tui.Spacer`.
  Views are immutable; rebuild them on every `View()` call from your state.
- **Events & commands**: `HandleEvent` receives events (e.g., `tui.KeyEvent`,
  `tui.MouseEvent`, `tui.TickEvent`). Return `tui.Cmd` values (like `tui.Quit()`,
  `tui.Tick`, or custom async operations) to run work off the UI thread.
- **Runtime**: `tui.Run(app, opts...)` wires up the terminal, alternate screen,
  mouse tracking, and render loop. Use options like `tui.WithFPS(60)` or
  `tui.WithMouseTracking(true)` for finer control.

## Quick example

```go
package main

import (
	"log"
	"strings"

	"github.com/deepnoodle-ai/wonton/tui"
)

type inbox struct {
	filter   string
	selected int
	items    []string
}

func (app *inbox) View() tui.View {
	visible := make([]string, 0, len(app.items))
	for _, item := range app.items {
		if app.filter == "" || strings.Contains(strings.ToLower(item), app.filter) {
			visible = append(visible, item)
		}
	}

	return tui.VStack(
		tui.Text("Inbox (%d)", len(visible)).Bold(),
		tui.Text("Type to filter, Enter to open, q to quit").Dim(),
		tui.SelectListStrings(visible, &app.selected).
			OnSelect(func(idx int) {
				app.selected = idx
			}),
		tui.Clickable("Refresh", func() {
			// Trigger async refresh via HandleEvent.
		}).Width(12),
	).Gap(1)
}

func (app *inbox) HandleEvent(ev tui.Event) []tui.Cmd {
	switch e := ev.(type) {
	case tui.KeyEvent:
		switch e.Rune {
		case 'q':
			return []tui.Cmd{tui.Quit()}
		default:
			if e.Rune != 0 && e.Modifiers == 0 {
				app.filter += string(e.Rune)
			}
		}
	case tui.TickEvent:
		// Periodic refresh.
	}
	return nil
}

func main() {
	app := &inbox{
		items: []string{"Prod rollout", "Weekly sync", "Incident review"},
	}

	if err := tui.Run(app, tui.WithFPS(30)); err != nil {
		log.Fatal(err)
	}
}
```

See the `examples/` directory for advanced scenarios: HTTP-driven updates
(`runtime_http`), markdown rendering (`markdown_demo`), file pickers, forms, and
animation samples.
