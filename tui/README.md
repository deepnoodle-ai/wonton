# TUI engine

The `tui` package provides Wonton's declarative terminal UI engine. It sits on
top of the `terminal` package and offers high-level layout primitives, ready-made
controls (tables, lists, inputs, markdown, file pickers), and an event-driven
runtime that keeps your application logic single-threaded.

## Mental model

- **Application**: implement `tui.Application` by providing a `View() tui.View`.
  Optionally implement `tui.EventHandler` to respond to keyboard, mouse, tick,
  and custom events.
- **Views**: compose UI using helpers like `tui.Stack`, `tui.HStack`,
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

	return tui.Stack(
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

## Animation

The TUI library provides two approaches for animation:

### Fluent Text Animations

Use chainable methods on text views for common animation effects:

```go
// Rainbow color cycling
tui.Text("Hello World").Rainbow(3)      // speed: lower = faster
tui.Text("Reverse!").RainbowReverse(3)

// Pulsing brightness
tui.Text("Alert!").Pulse(tui.NewRGB(255, 0, 0), 12)

// Wave color effect (speed, then colors)
tui.Text("Wave").Wave(12,
    tui.NewRGB(255, 50, 0),
    tui.NewRGB(255, 150, 0),
)

// Sliding highlight effect (speed, base color, highlight color)
tui.Text("Shine").Slide(2, tui.NewRGB(100, 100, 100), tui.NewRGB(255, 255, 255))
tui.Text("Shine").SlideReverse(2, baseColor, highlightColor)  // right to left

// Sparkle effect - random characters twinkle like stars
tui.Text("Stars").Sparkle(3, tui.NewRGB(180, 180, 220), tui.NewRGB(255, 255, 255))

// Typewriter effect - characters reveal one by one with blinking cursor
tui.Text("Loading...").Typewriter(3, tui.NewRGB(0, 255, 100), tui.NewRGB(255, 255, 255))

// Glitch effect - cyberpunk-style digital corruption
tui.Text("SIGNAL").Glitch(2, tui.NewRGB(255, 0, 100), tui.NewRGB(0, 255, 255))
```

### Semantic Text Styles

Common text patterns have convenient builder methods:

```go
tui.Text("Operation successful").Success()  // green, bold
tui.Text("Error occurred").Error()          // red, bold
tui.Text("Warning: low disk").Warning()     // yellow, bold
tui.Text("More info").Info()                // cyan
tui.Text("Secondary text").Muted()          // dim, gray
tui.Text("Press Enter to continue").Hint()  // dim, italic
```

### Custom Canvas Animations

For complex animations, use `CanvasContext` to access the animation frame counter:

```go
tui.CanvasContext(func(ctx *tui.RenderContext) {
    w, h := ctx.Size()
    frame := ctx.Frame()  // Animation frame counter (increments each tick)

    // Animate a moving block
    x := int(frame) % w
    ctx.SetCell(x, 0, '█', tui.NewStyle().WithForeground(tui.ColorCyan))
})
```

The `RenderContext` provides:
- `Frame()` - animation frame counter (60 FPS by default)
- `Size()` - drawing area dimensions
- `SetCell()`, `PrintStyled()`, etc. - drawing methods

See the `examples/` directory for advanced scenarios: HTTP-driven updates
(`runtime_http`), markdown rendering (`markdown_demo`), file pickers, forms, and
animation samples (`animation_demo`, `declarative_animation`).
