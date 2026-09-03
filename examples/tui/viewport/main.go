// Command viewport demonstrates tui.Viewport: a virtualized list that stays
// still while content streams into it, and that a user can select text out of
// with the mouse the way they would in the terminal itself.
//
// Run it, then drag across the messages, double-click a word, triple-click a
// line, and press c to copy. Watch a message stream in while a selection is
// held: the highlight stays on its own words.
package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/deepnoodle-ai/wonton/clipboard"
	"github.com/deepnoodle-ai/wonton/tui"
)

// message is one entry in the transcript. Streaming appends to the last one.
type message struct {
	who  string
	body string
}

// transcript is the ViewportItems the viewport reads every frame. Item is
// called at most once per index until Invalidate, so building the view here is
// as expensive as it looks and no more.
type transcript struct {
	messages []message
}

func (t *transcript) Len() int { return len(t.messages) }

func (t *transcript) Item(i int) tui.View {
	m := t.messages[i]
	who := tui.ColorCyan
	if m.who == "assistant" {
		who = tui.ColorGreen
	}
	return tui.Stack(
		tui.Text("%s", m.who).Bold().Fg(who),
		tui.Text("%s", m.body),
	)
}

type app struct {
	items    *transcript
	viewport tui.ViewportState

	// runtime is handed over by SetRuntime, which is how an application
	// reaches its own terminal to write an escape sequence — OSC 52, here.
	runtime *tui.Runtime

	status  string
	pending string // what is left to stream into the last message
}

func (a *app) HandleEvent(event tui.Event) []tui.Cmd {
	switch e := event.(type) {
	case tui.MouseEvent:
		// The viewport gets first refusal on the left button. It returns false
		// for anything it did not consume — the wheel, and a plain click with
		// no selection to dismiss — which is what leaves room for an
		// application's own click targets inside a viewport.
		if a.viewport.HandleMouse(e) {
			return nil
		}
		switch e.Button {
		case tui.MouseButtonWheelUp:
			a.viewport.ScrollBy(-3)
		case tui.MouseButtonWheelDown:
			a.viewport.ScrollBy(3)
		}

	case tui.TickEvent:
		// A pointer held still outside the viewport sends no further mouse
		// events, so a drag past the edge only keeps growing if something
		// nudges it once a frame.
		a.viewport.DragAutoScroll()
		a.stream()

	case tui.KeyEvent:
		switch {
		case e.Rune == 'q' || e.Key == tui.KeyCtrlC:
			return []tui.Cmd{tui.Quit()}
		case e.Rune == 'c':
			a.copySelection()
		case e.Key == tui.KeyEscape:
			a.viewport.ClearSelection()
			a.status = "Selection cleared"
		case e.Key == tui.KeyPageUp:
			a.viewport.PageUp()
		case e.Key == tui.KeyPageDown:
			a.viewport.PageDown()
		case e.Key == tui.KeyEnd:
			a.viewport.ScrollToBottom()
		case e.Key == tui.KeyHome:
			a.viewport.ScrollToTop()
		}
	}
	return nil
}

// copySelection puts the selection on the system clipboard, and asks the
// terminal for it too.
//
// Two rungs, because neither alone is enough: the native clipboard addresses
// this machine, which is the wrong one over SSH, and OSC 52 addresses the
// terminal the user is sitting at but has no reply and is not always enabled.
// Neither can confirm it landed, so the status line says what was attempted.
func (a *app) copySelection() {
	text := a.viewport.SelectedText()
	if text == "" {
		a.status = "Nothing selected"
		return
	}
	nativeErr := clipboard.Write(text)

	// *tui.Terminal is an io.Writer, so it goes straight in. This runs on the
	// goroutine that calls HandleEvent, which is the one that owns the
	// terminal, so the sequence cannot land inside a frame.
	osc52Err := clipboard.WriteOSC52(a.runtime.Terminal(), text)

	// Each rung is tried and reported on its own. Reporting them together
	// would claim the clipboard succeeded whenever the terminal did, which is
	// the opposite of what a copy with no confirmation should tell the user.
	switch {
	case nativeErr != nil && osc52Err != nil:
		a.status = fmt.Sprintf("Copy failed — clipboard: %v; OSC 52: %v", nativeErr, osc52Err)
	case nativeErr != nil:
		a.status = fmt.Sprintf("Sent %d bytes to the terminal; clipboard failed: %v", len(text), nativeErr)
	case osc52Err != nil:
		a.status = fmt.Sprintf("Copied %d bytes to the clipboard; OSC 52 failed: %v", len(text), osc52Err)
	default:
		a.status = fmt.Sprintf("Copied %d bytes (sent to clipboard and terminal)", len(text))
	}
}

// SetRuntime implements tui.RuntimeAware. The runtime calls it before Init, so
// the terminal is available from the first frame.
func (a *app) SetRuntime(r *tui.Runtime) { a.runtime = r }

// stream appends a few characters to the last message each frame, so the
// viewport has content moving under a selection.
func (a *app) stream() {
	if a.pending == "" {
		return
	}
	n := min(3, len(a.pending))
	last := len(a.items.messages) - 1
	a.items.messages[last].body += a.pending[:n]
	a.pending = a.pending[n:]
	// The item changed in place, so its cached view, height and rendered rows
	// are all stale. Appending a new message would need no call.
	a.viewport.Invalidate(last)
}

func (a *app) View() tui.View {
	sel := "none"
	if a.viewport.HasSelection() {
		sel = fmt.Sprintf("%d bytes", len(a.viewport.SelectedText()))
	}
	position := "following"
	if !a.viewport.Follow {
		position = fmt.Sprintf("%d lines below", a.viewport.LinesBelow)
	}

	return tui.Stack(
		tui.Text("Viewport & Selection").Bold().Fg(tui.ColorCyan),
		tui.Text("drag to select · double-click a word · triple-click a line · c copies · Esc clears · q quits").Dim(),
		tui.Bordered(tui.Viewport(&a.viewport, a.items).Gap(1)).
			BorderFg(tui.ColorBlue).
			Title("Transcript"),
		tui.Group(
			tui.Text("Selected:").Fg(tui.ColorCyan),
			tui.Text(" %s  ", sel),
			tui.Text("Position:").Fg(tui.ColorCyan),
			tui.Text(" %s  ", position),
			tui.Text("%s", a.status).Dim(),
		),
	).Padding(1)
}

func main() {
	items := &transcript{messages: seed()}
	a := &app{
		items:   items,
		status:  "Ready",
		pending: strings.Repeat("This reply is still streaming in. ", 6),
	}
	a.viewport.Follow = true
	a.items.messages = append(a.items.messages, message{who: "assistant", body: ""})

	// Drag reporting, not full hover tracking: a selection needs motion while a
	// button is held and nothing more.
	if err := tui.Run(a, tui.WithMouseDrag(true)); err != nil {
		log.Fatal(err)
	}
}

func seed() []message {
	var out []message
	for i := 1; i <= 12; i++ {
		out = append(out,
			message{who: "user", body: fmt.Sprintf("Question %d: where does tui/viewport_selection.go put the endpoints?", i)},
			message{who: "assistant", body: fmt.Sprintf(
				"Answer %d: on (item, line, column), not screen rows.\nThat is what keeps a selection on its own words\nwhile a reply streams in above it.", i)},
		)
	}
	return out
}
