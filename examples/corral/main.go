// Example: Corral
//
// A terminal workspace for running many shells and coding agents side by
// side. The main area is split into columns of live terminals, each column
// divided into rows as needed; a sidebar lists every space and pane. Every
// pane is a tmux session on a private tmux server, so quitting corral leaves
// them running and starting it again picks up where you left off.
//
// Mouse:
//
//   - Click a pane, or a pane in the sidebar, to focus it.
//   - Drag the edge between two panes (or the sidebar's edge) to resize.
//   - Right-click anything — a pane, the sidebar, an edge, the top bar — for
//     what you can do with it.
//   - Scroll a pane to page back through its history.
//   - Drag across text in a pane to select it and copy it.
//
// Keys go to the focused pane. Ctrl+B is the prefix for corral's own keys;
// press Ctrl+B ? to list them, or Ctrl+B twice to send Ctrl+B to the pane.
//
// Run with:
//
//	go run ./examples/corral
//	go run ./examples/corral -L work      # a separate set of sessions
//
// Requires tmux.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/deepnoodle-ai/wonton/tmux"
	"github.com/deepnoodle-ai/wonton/tui"
)

func main() {
	socket := flag.String("L", "corral", "name of the private tmux server the sessions live on")
	config := flag.String("f", "", "tmux configuration file (default: your own)")
	flag.Parse()

	server := &tmux.Server{Socket: *socket, Config: *config}
	if !server.Available() {
		fmt.Fprintln(os.Stderr, "corral runs its terminals in tmux, which is not installed.")
		fmt.Fprintln(os.Stderr, "Install it (brew install tmux, apt install tmux) and try again.")
		os.Exit(1)
	}

	a := newApp(server)
	err := tui.Run(a,
		tui.WithMouseTracking(true),
		tui.WithBracketedPaste(true),
		tui.WithFPS(15),
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, "corral:", err)
		os.Exit(1)
	}
	switch {
	case a.ended:
		fmt.Println("corral: ended every session.")
	case len(a.panes) > 0:
		fmt.Printf("corral: %d session(s) still running. Run corral again to pick up where you left off.\n", len(a.panes))
	}
}
