# gooey

A sophisticated Terminal GUI library for Go that provides flicker-free rendering, advanced animations (30+ FPS), and a race-free message-driven architecture.

## Features

- **Race-Free Message-Driven Architecture** - Single-threaded event loop eliminates concurrency bugs
- **Flicker-Free Rendering** - Double-buffered rendering with dirty region tracking
- **Smooth Animations** - Support for 30-60 FPS animations with minimal CPU usage
- **Advanced Input Handling** - Full keyboard and mouse support with paste handling
- **Composition System** - Build complex UIs with containers and layout managers
- **No Manual Synchronization** - No need for mutexes or locks in application code

## Quick Start

### Message-Driven Architecture (Recommended)

The Runtime provides a race-free, event-driven architecture similar to The Elm Architecture but preserving Gooey's superior rendering system:

```go
package main

import (
    "fmt"
    "github.com/yourusername/gooey"
)

type CounterApp struct {
    count int
}

func (app *CounterApp) HandleEvent(event gooey.Event) []gooey.Cmd {
    switch e := event.(type) {
    case gooey.KeyEvent:
        switch e.Rune {
        case '+':
            app.count++
        case '-':
            app.count--
        case 'q':
            return []gooey.Cmd{gooey.Quit()}
        }
    }
    return nil
}

func (app *CounterApp) Render(frame gooey.RenderFrame) {
    style := gooey.NewStyle().WithForeground(gooey.ColorGreen)
    frame.PrintStyled(0, 0, fmt.Sprintf("Count: %d", app.count), style)
    frame.PrintStyled(0, 1, "Press +/- to change, q to quit", gooey.NewStyle())
}

func main() {
    terminal, _ := gooey.NewTerminal()
    defer terminal.Close()

    runtime := gooey.NewRuntime(terminal, &CounterApp{}, 30)
    runtime.Run()
}
```

**Key Benefits:**
- No race conditions - HandleEvent and Render are never called concurrently
- No locks needed - State can be mutated freely
- Async operations via commands - HTTP requests don't block the UI
- Automatic rendering - Screen updates after each event

See `MESSAGE_DRIVEN_ARCHITECTURE.md` for complete details.

### Examples

```bash
# Counter with keyboard input
go run examples/runtime_counter/main.go

# HTTP requests without blocking UI
go run examples/runtime_http/main.go

# Smooth 60 FPS animation
go run examples/runtime_animation/main.go

# Composition system with containers
go run examples/runtime_composition/main.go

# Comprehensive demo of all features
go run examples/all/main.go
```

## Documentation

- `MESSAGE_DRIVEN_ARCHITECTURE.md` - Complete guide to the message-driven architecture
- `documentation/plan.md` - Implementation plan and design decisions
- `CLAUDE.md` - Developer guide and architecture overview
- `documentation/composition_guide.md` - Guide to the composition system
- `documentation/animations.md` - Animation system guide

## Migration from Manual Threading

If you're currently using Animator and ScreenManager with manual mutexes:

**Before:**
```go
var mu sync.Mutex

animator := gooey.NewAnimator(terminal, 60)
animator.Start()

go func() {
    for {
        mu.Lock()
        frame, _ := terminal.BeginFrame()
        render(frame)
        terminal.EndFrame(frame)
        mu.Unlock()
    }
}()
```

**After:**
```go
type MyApp struct {
    // Your state here (no mutex needed!)
}

func (app *MyApp) HandleEvent(event gooey.Event) []gooey.Cmd {
    // Handle events and modify state
    return nil
}

func (app *MyApp) Render(frame gooey.RenderFrame) {
    // Render using existing code
}

runtime := gooey.NewRuntime(terminal, &MyApp{}, 60)
runtime.Run()
```

See the migration guide in `MESSAGE_DRIVEN_ARCHITECTURE.md` for complete details.

## License

MIT
