// Demonstrates shell-style input affordances on InputField:
//
//   - History(items): Up/Down recall previous submissions. In multiline
//     mode, arrows navigate the text first and recall history only at the
//     first/last visual line (like zsh/fish).
//   - OnComplete(fn): Tab completion with in-buffer cycling. Tab/Shift+Tab
//     or Down/Up cycle candidates, Esc restores the original text.
//   - OnKey(fn): claim specific keys before the input processes them
//     (here, Ctrl+L clears the input).
//
// Try: type "/he", press Tab repeatedly, then Enter. Press Up to recall.
// Compose a multiline entry with Shift+Enter and notice Up moves the cursor
// before it recalls history.
package main

import (
	"strings"

	"github.com/deepnoodle-ai/wonton/tui"
)

var commands = []string{
	"/help", "/hello", "/health", "/history", "/clear", "/quit",
}

type replApp struct {
	runner  *tui.InlineApp
	input   string
	history []string
}

func (a *replApp) LiveView() tui.View {
	return tui.Stack(
		tui.Divider(),
		tui.InputField(&a.input).
			ID("repl-input").
			Prompt("❯").
			Multiline(true).
			MaxHeight(8).
			Placeholder("Type a command (Tab completes, Up recalls)...").
			History(a.history).
			OnComplete(a.complete).
			OnKey(a.handleKey).
			OnSubmit(a.submit),
		tui.Divider(),
		tui.Text(" Tab complete · Up/Down history · Shift+Enter newline · Ctrl+L clear · Ctrl+C quit").Hint(),
	).Gap(0)
}

func (a *replApp) complete(value string) []string {
	if value == "" {
		return nil
	}
	var matches []string
	for _, cmd := range commands {
		if strings.HasPrefix(cmd, value) {
			matches = append(matches, cmd)
		}
	}
	return matches
}

// handleKey runs before the input's own handling (completion, history,
// submit, editing), so any key claimed here wins over the built-ins.
func (a *replApp) handleKey(e tui.KeyEvent) bool {
	if e.Key == tui.KeyCtrlL {
		a.input = ""
		return true
	}
	return false
}

func (a *replApp) submit(value string) {
	if value == "" {
		return
	}
	a.history = append(a.history, value)
	a.input = ""
	switch value {
	case "/quit":
		a.runner.Stop()
	case "/clear":
		a.history = nil
		a.runner.Printf("history cleared")
	case "/history":
		for i, h := range a.history {
			a.runner.Printf("%3d  %s", i+1, h)
		}
	default:
		a.runner.Printf("you ran: %s", value)
	}
}

func (a *replApp) HandleEvent(event tui.Event) []tui.Cmd {
	if key, ok := event.(tui.KeyEvent); ok && key.Ctrl && key.Rune == 'c' {
		return []tui.Cmd{tui.Quit()}
	}
	return nil
}

func main() {
	app := &replApp{}
	app.runner = tui.NewInlineApp(tui.WithInlineFPS(30))
	if err := app.runner.Run(app); err != nil {
		panic(err)
	}
}
