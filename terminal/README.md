# terminal

Low-level terminal control with double-buffered rendering, input decoding, styling, and mouse/keyboard event handling. Provides the foundation for building rich terminal user interfaces.

## Features

- Double-buffered rendering with dirty region tracking
- Frame-based rendering API with atomic updates
- ANSI escape sequence generation and parsing
- Wide character support (CJK, emojis)
- Mouse event parsing (SGR extended mode)
- Keyboard event decoding with modifiers
- Raw mode and alternate screen buffer
- Cursor control and visibility
- RGB and 256-color support
- OSC 8 hyperlink support
- Terminal resize handling with callbacks
- Performance metrics collection
- Bracketed paste support
- Kitty keyboard protocol detection

## Usage Examples

### Basic Rendering

```go
package main

import (
    "github.com/deepnoodle-ai/wonton/terminal"
)

func main() {
    term, _ := terminal.NewTerminal()
    defer term.Close()

    // Enable raw mode and alternate screen
    term.EnableRawMode()
    term.EnableAlternateScreen()
    term.HideCursor()

    // Begin frame (locks terminal)
    frame, _ := term.BeginFrame()

    // Draw content
    style := terminal.NewStyle().WithForeground(terminal.ColorGreen).WithBold()
    frame.PrintStyled(0, 0, "Hello, World!", style)

    // End frame (flushes to terminal)
    term.EndFrame(frame)
}
```

### Styled Text

```go
frame, _ := term.BeginFrame()

// Create styles
headerStyle := terminal.NewStyle().
    WithForeground(terminal.ColorBlue).
    WithBold()

warningStyle := terminal.NewStyle().
    WithForeground(terminal.ColorYellow).
    WithBgRGB(terminal.RGB{40, 40, 40})

// Print styled text
frame.PrintStyled(0, 0, "Header", headerStyle)
frame.PrintStyled(0, 1, "Warning!", warningStyle)

term.EndFrame(frame)
```

### RGB Colors

```go
// True color support
rgb := terminal.NewRGB(100, 200, 255)
style := terminal.NewStyle().WithFgRGB(rgb)

frame.PrintStyled(0, 0, "True color text", style)
```

### Filling Regions

```go
frame, _ := term.BeginFrame()

// Fill rectangle with character and style
boxStyle := terminal.NewStyle().
    WithBackground(terminal.ColorBlue)

frame.FillStyled(5, 5, 20, 10, ' ', boxStyle)

term.EndFrame(frame)
```

### SubFrames for Layout

```go
import "image"

frame, _ := term.BeginFrame()

// Create subframe for a panel
panelRect := image.Rect(10, 5, 50, 20)
panel := frame.SubFrame(panelRect)

// Draw in panel using (0,0) relative coordinates
panel.PrintStyled(0, 0, "Panel Header", headerStyle)
panel.PrintStyled(0, 1, "Content", textStyle)

// SubFrame automatically handles clipping and translation
term.EndFrame(frame)
```

### Keyboard Input

```go
import "os"

decoder := terminal.NewKeyDecoder(os.Stdin)

for {
    event, err := decoder.ReadEvent()
    if err != nil {
        break
    }

    switch e := event.(type) {
    case terminal.KeyEvent:
        if e.Key == terminal.KeyCtrlC {
            return
        }
        if e.Rune != 0 {
            fmt.Printf("Char: %c\n", e.Rune)
        }
        if e.Paste != "" {
            fmt.Printf("Pasted: %s\n", e.Paste)
        }
    }
}
```

### Mouse Input

```go
term.EnableMouseTracking()
defer term.DisableMouseTracking()

decoder := terminal.NewKeyDecoder(os.Stdin)

for {
    event, err := decoder.ReadEvent()
    if err != nil {
        break
    }

    switch e := event.(type) {
    case terminal.MouseEvent:
        if e.Type == terminal.MousePress && e.Button == terminal.MouseButtonLeft {
            fmt.Printf("Clicked at (%d, %d)\n", e.X, e.Y)
        }
        if e.Type == terminal.MouseScroll {
            fmt.Printf("Scrolled: %d\n", e.DeltaY)
        }
    }
}
```

### Hyperlinks

```go
// OSC 8 hyperlink (works in iTerm2, WezTerm, kitty)
link := terminal.Hyperlink{
    Text: "Click here",
    URL:  "https://example.com",
    Style: terminal.NewStyle().
        WithForeground(terminal.ColorBlue).
        WithUnderline(),
}

frame.PrintHyperlink(5, 5, link)

// Fallback format: "Text (URL)"
frame.PrintHyperlinkFallback(5, 6, link)
```

### Terminal Resize Handling

```go
term.WatchResize()
defer term.StopWatchResize()

unregister := term.OnResize(func(width, height int) {
    log.Printf("Terminal resized to %dx%d\n", width, height)
    // Trigger re-render
    render()
})
defer unregister()
```

### Performance Metrics

```go
term.EnableMetrics()

// ... render frames ...

metrics := term.GetMetrics()
fmt.Printf("Frames: %d\n", metrics.FrameCount)
fmt.Printf("Avg render time: %v\n", metrics.AvgRenderTime)
fmt.Printf("Cells updated: %d\n", metrics.TotalCellsUpdated)
```

### Enhanced Keyboard Protocol

```go
// Detect Kitty keyboard protocol support
if term.DetectKittyProtocol() {
    term.EnableEnhancedKeyboard()
    defer term.DisableEnhancedKeyboard()
}

// Now Shift+Enter, Ctrl+Enter, etc. are distinguishable
decoder := terminal.NewKeyDecoder(os.Stdin)
for {
    event, err := decoder.ReadEvent()
    if err != nil {
        break
    }

    if key, ok := event.(terminal.KeyEvent); ok {
        if key.Key == terminal.KeyEnter && key.Shift {
            fmt.Println("Shift+Enter pressed")
        }
    }
}
```

## API Reference

### Terminal Management

| Function          | Description                  | Inputs                             | Outputs             |
| ----------------- | ---------------------------- | ---------------------------------- | ------------------- |
| `NewTerminal`     | Create new terminal instance | None                               | `*Terminal, error`  |
| `NewTestTerminal` | Create terminal for testing  | `width, height int, out io.Writer` | `*Terminal`         |
| `Close`           | Clean up terminal state      | None                               | `error`             |
| `Size`            | Get terminal dimensions      | None                               | `width, height int` |
| `RefreshSize`     | Update cached terminal size  | None                               | `error`             |

### Frame Rendering

| Method       | Description                            | Inputs              | Outputs              |
| ------------ | -------------------------------------- | ------------------- | -------------------- |
| `BeginFrame` | Start frame rendering (locks terminal) | None                | `RenderFrame, error` |
| `EndFrame`   | Finish frame and flush changes         | `frame RenderFrame` | `error`              |
| `Flush`      | Manually flush buffer changes          | None                | None                 |

### RenderFrame Methods

| Method           | Description                         | Inputs                                            | Outputs             |
| ---------------- | ----------------------------------- | ------------------------------------------------- | ------------------- |
| `SetCell`        | Set character and style at position | `x, y int, char rune, style Style`                | `error`             |
| `PrintStyled`    | Print text with wrapping            | `x, y int, text string, style Style`              | `error`             |
| `PrintTruncated` | Print text with truncation          | `x, y int, text string, style Style`              | `error`             |
| `FillStyled`     | Fill rectangle with character       | `x, y, width, height int, char rune, style Style` | `error`             |
| `Fill`           | Fill entire frame                   | `char rune, style Style`                          | `error`             |
| `Size`           | Get frame dimensions                | None                                              | `width, height int` |
| `GetBounds`      | Get frame bounds                    | None                                              | `image.Rectangle`   |
| `SubFrame`       | Create subframe for region          | `rect image.Rectangle`                            | `RenderFrame`       |
| `PrintHyperlink` | Print clickable hyperlink (OSC 8)   | `x, y int, link Hyperlink`                        | `error`             |

### Input Decoding

| Function          | Description                  | Inputs        | Outputs              |
| ----------------- | ---------------------------- | ------------- | -------------------- |
| `NewKeyDecoder`   | Create input decoder         | `r io.Reader` | `*KeyDecoder`        |
| `ReadEvent`       | Read next input event        | None          | `Event, error`       |
| `ParseMouseEvent` | Parse mouse event from bytes | `seq []byte`  | `*MouseEvent, error` |

### Styles

| Function   | Description      | Inputs          | Outputs |
| ---------- | ---------------- | --------------- | ------- |
| `NewStyle` | Create new style | None            | `Style` |
| `NewRGB`   | Create RGB color | `r, g, b uint8` | `RGB`   |

See full API reference in the [package godoc](https://pkg.go.dev/github.com/deepnoodle-ai/wonton/terminal).

## Related Packages

- [tui](../tui) - Declarative TUI library built on terminal
- [termtest](../termtest) - Testing utilities for terminal output
- [termsession](../termsession) - Terminal session recording/playback
- [color](../color) - Color manipulation and gradients

## Design Notes

The terminal package uses double-buffering to minimize flicker and optimize rendering. Only cells that changed since the last frame are redrawn. The dirty region tracking further optimizes by skipping unchanged screen areas entirely.

Frame-based rendering ensures atomic updates: either the entire frame renders successfully, or nothing changes. This prevents partial renders that can occur with immediate-mode APIs.

SubFrames provide coordinate translation and clipping automatically. When drawing to a SubFrame, always use coordinates relative to (0,0), not the bounds returned by GetBounds(). The SubFrame handles translation internally.
