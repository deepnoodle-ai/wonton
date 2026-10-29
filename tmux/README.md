# tmux

Run real terminals inside a Go TUI by letting tmux do the terminal emulation.
A `Terminal` attaches to a tmux session over a control-mode connection
(`tmux -C`), captures the session's pane into a `Screen` snapshot whenever its
program writes, and sends keys, pastes, and mouse events back. Shells, editors,
and full-screen programs such as `vim`, `htop`, or a coding agent work as they
would in any terminal.

The sessions live in the tmux server, not in your program, so they outlive it.
A program that exits and starts again reattaches to the shells it left
running.

## Features

- Live mirror of a pane: cells, colors (16, 256, and 24-bit), wide characters,
  OSC 8 hyperlinks, cursor, and modes (alternate screen, mouse reporting, copy
  mode)
- Key input by name, so tmux encodes keys for the pane's current modes
- Bracketed paste, and mouse forwarding in SGR or X10 encoding when the
  program asks for the mouse
- Scrollback through copy mode, the alternate-screen wheel convention, or the
  program's own mouse handling
- Captures coalesced and rate limited, so a pane spewing output stays cheap
- A control-mode client (`Control`) and one-shot commands (`Server.Run`) for
  everything else tmux can do

## Usage Examples

### A terminal in a TUI

```go
package main

import (
	"context"
	"log"
	"time"

	"github.com/deepnoodle-ai/wonton/tmux"
	"github.com/deepnoodle-ai/wonton/tui"
)

type redraw struct{}

func (redraw) Timestamp() time.Time { return time.Now() }

type app struct {
	rt   *tui.Runtime
	term *tmux.Terminal
}

func (a *app) SetRuntime(rt *tui.Runtime) { a.rt = rt }

func (a *app) Init() error {
	server := &tmux.Server{Socket: "myapp"} // a private server: tmux -L myapp
	term, err := server.Open(context.Background(), tmux.TerminalOptions{
		Session: "shell",
		Width:   80,
		Height:  24,
		// Called from a tmux goroutine whenever the screen changes.
		OnUpdate: func() { go a.rt.SendEvent(redraw{}) },
	})
	a.term = term
	return err
}

func (a *app) Destroy() { a.term.Detach() } // the shell keeps running

func (a *app) View() tui.View {
	s := a.term.Screen()
	return tui.Bordered(tui.Cells(s.Cells).Cursor(s.CursorX, s.CursorY)).Title(s.Command)
}

func (a *app) HandleEvent(ev tui.Event) []tui.Cmd {
	switch e := ev.(type) {
	case tui.KeyEvent:
		a.term.SendKey(e) // pastes too, when e.Paste is set
	case tui.ResizeEvent:
		a.term.Resize(e.Width-2, e.Height-2)
	}
	select {
	case <-a.term.Done():
		return []tui.Cmd{tui.Quit()} // the shell exited
	default:
		return nil
	}
}

func main() {
	if err := tui.Run(&app{}, tui.WithBracketedPaste(true)); err != nil {
		log.Fatal(err)
	}
}
```

`examples/corral` builds a full workspace on this: resizable splits of
terminals, a sidebar, context menus, and a layout that survives restarts.

### Mouse and scrollback

```go
// Forward a click if the pane's program reads the mouse (vim with mouse=a,
// htop, ...). x and y are cells inside the pane.
if !term.SendMouse(ev, x, y) {
	// The program does not want it: use the wheel to scroll history.
	if ev.Button == tui.MouseButtonWheelUp {
		term.Scroll(-3) // enters copy mode; scrolling back down leaves it
	}
}
```

While the pane is in copy mode, `Screen().InMode` is set, `Cells` shows the
history lines being viewed, and `SendKey` scrolls for the navigation keys and
leaves the mode for any other key.

### Sessions and options

```go
server := &tmux.Server{Socket: "myapp"}
ctx := context.Background()

sessions, _ := server.Sessions(ctx) // none, not an error, if the server is down
for _, s := range sessions {
	fmt.Println(s.Name, s.Attached, s.Path)
}

// User options live as long as the server: a good home for state that
// belongs with the sessions, such as how they were laid out.
server.SetOption(ctx, "@layout", layoutJSON)
saved, _ := server.GetOption(ctx, "@layout")

server.KillSession(ctx, "shell")
```

### Control mode

```go
ctl, err := server.Attach(ctx, "shell", func(n tmux.Notification) {
	if n.Name == "output" {
		fmt.Printf("%s wrote %d bytes\n", n.Pane, len(n.Data))
	}
})
out, err := ctl.Command("list-windows -t " + tmux.Quote("=shell"))
ctl.Close()
```

`Command` waits for tmux's reply; several commands separated by `" ; "` reply
together. `Quote` makes any string safe to splice into a command, newlines and
control characters included.

## API Reference

| Type / Function           | Description                                                        |
| ------------------------- | ------------------------------------------------------------------ |
| `Server`                  | A tmux server, chosen by `Socket` (-L) or `SocketPath` (-S)         |
| `Server.Open`             | Attach a `Terminal` to a session, creating it if needed             |
| `Server.Attach`           | Open a control-mode connection to a session                         |
| `Server.Run`              | Run one tmux command and return its output                          |
| `Server.NewSession`       | Start a detached, manually sized session with no status line        |
| `Server.Sessions`         | List sessions                                                       |
| `Server.GetOption/SetOption` | Read and write global user options (`@name`)                     |
| `Terminal.Screen`         | The latest `Screen` snapshot; never nil                             |
| `Terminal.SendKey`        | Type a `terminal.KeyEvent` (or paste it, when `Paste` is set)       |
| `Terminal.SendText/Paste` | Type text, or paste it with bracketed paste when the program wants it |
| `Terminal.SendMouse`      | Forward a mouse event if the program has turned on mouse reporting  |
| `Terminal.Scroll`         | Scroll history (copy mode) or send cursor keys on the alternate screen |
| `Terminal.WrappedRows`    | Which rows wrap onto the next, so copied text joins them             |
| `Terminal.Resize`         | Set the pane's size in cells                                        |
| `Terminal.Detach/Kill`    | Leave the session running, or end it                                |
| `Terminal.Done/Exited`    | Learn that the terminal ended, and whether on its own               |
| `Screen`                  | Cells, cursor, modes, title, command, and working directory         |
| `ParseCapture`            | Turn `capture-pane -p -e` output into cells                         |
| `KeyName`                 | The tmux key name for a `terminal.KeyEvent`                         |
| `MouseSequence`           | The bytes a terminal sends a program for a mouse event              |
| `Quote`                   | Quote a string for tmux's command language                          |

## Notes

- Requires tmux on `PATH` (`Server.Binary` overrides it). Paste and mouse
  forwarding pass control characters through verbatim on tmux 3.5 and later;
  older versions show them as `^X`.
- `NewSession` and `Open` turn the session's status line off and size its
  window manually, so the pane is exactly the size you ask for no matter what
  other clients attach.
- Give your program its own `Socket`. Its sessions then stay out of the user's
  `tmux ls`, and the user's configuration still applies unless you set
  `Config` (use `"/dev/null"` for none).

## Related Packages

- [tui](../tui) - `Cells` draws a `Screen`; `Split`, `Menu`, and `HitMap` build a workspace around it
- [terminal](../terminal) - The key and mouse events a `Terminal` accepts
