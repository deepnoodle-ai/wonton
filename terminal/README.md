# Terminal primitives

The `terminal` package is Wonton's low-level abstraction over ANSI-capable
terminals. It exposes double-buffered rendering, hyperlink support, mouse and
keyboard decoding, frame metrics, and helpers for recording sessions or replaying
bugs.

Use it directly when you need total control, or indirectly through the higher
level `tui` runtime.

## Capabilities

- `Terminal.BeginFrame` / `EndFrame` double-buffered drawing with dirty-region
  tracking for minimal writes.
- Raw-mode management with cursor hiding, alternate screen buffers, and mouse
  tracking toggles.
- Event decoding: key presses (including Kitty-kb), mouse press/drag/click, and
  resize notifications.
- Session recording (`Recorder`), hyperlink primitives (OSC 8), render metrics,
  and color utilities (via the `color` package).

## Example

```go
package main

import (
	"log"
	"strconv"
	"time"

	"github.com/deepnoodle-ai/wonton/terminal"
)

func main() {
	term, err := terminal.NewTerminal()
	if err != nil {
		log.Fatal(err)
	}
	defer term.Close()

	term.EnableAlternateScreen()
	term.HideCursor()
	defer term.ShowCursor()

	style := terminal.NewStyle().
		WithForeground(terminal.ColorBrightCyan).
		WithBold()

	for i := 0; i < 100; i++ {
		frame, err := term.BeginFrame()
		if err != nil {
			log.Fatal(err)
		}

		frame.Fill(' ')
		frame.PrintStyled(2, 1, "Hello, Terminal!", style)
		frame.PrintStyled(2, 3, "Frame: "+strconv.Itoa(i), terminal.NewStyle())

		if err := term.EndFrame(frame); err != nil {
			log.Fatal(err)
		}

		time.Sleep(50 * time.Millisecond)
	}
}
```

For input handling, pair a `terminal.Terminal` with a `terminal.EventDecoder` or
the higher-level `tui.Runtime`. Check out `examples/hyperlink_demo` and the
`terminal` package tests for more detailed usage.
