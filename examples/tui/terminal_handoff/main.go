// Terminal handoff runs a real editor and preserves the application draft.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"unicode/utf8"

	"github.com/deepnoodle-ai/wonton/tui"
)

type editorApp struct {
	fullscreen    *tui.Runtime
	inline        *tui.InlineApp
	editor        string
	draft, notice string
}

func (a *editorApp) View() tui.View {
	return tui.Stack(
		tui.Text("Draft · Ctrl-O editor · Ctrl-C quit"),
		tui.InputField(&a.draft).ID("draft").Multiline(true).MaxHeight(8).Placeholder("Type a draft"),
		tui.Text("%s", a.notice),
	)
}
func (a *editorApp) LiveView() tui.View { return a.View() }
func (a *editorApp) HandleEvent(event tui.Event) []tui.Cmd {
	key, ok := event.(tui.KeyEvent)
	if !ok {
		return nil
	}
	if key.Key == tui.KeyCtrlC {
		return []tui.Cmd{tui.Quit()}
	}
	if key.Key != tui.KeyCtrlO {
		return nil
	}
	file, err := os.CreateTemp("", "wonton-draft-*.txt")
	if err != nil {
		a.notice = err.Error()
		return nil
	}
	path := file.Name()
	defer os.Remove(path)
	_, writeErr := file.WriteString(a.draft)
	err = errors.Join(writeErr, file.Close())
	if err != nil {
		a.notice = err.Error()
		return nil
	}
	callback := func() error {
		cmd := exec.Command(a.editor, path)
		cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stdout
		return cmd.Run()
	}
	var operationErr, restoreErr error
	if a.fullscreen != nil {
		operationErr, restoreErr = a.fullscreen.Handoff(callback)
	} else {
		operationErr, restoreErr = a.inline.Handoff(callback)
	}
	if restoreErr != nil {
		return nil
	} // Run returns the fatal restoration error.
	if operationErr != nil {
		a.notice = fmt.Sprintf("Editor failed; draft retained: %v", operationErr)
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil || !utf8.Valid(data) {
		a.notice = "Editor output unreadable; draft retained"
		return nil
	}
	a.draft = string(data)
	a.notice = "Editor returned; draft restored without submission"
	return nil
}
func main() {
	inline := flag.Bool("inline", false, "use inline mode")
	editor := flag.String("editor", "vi", "editor executable; arguments are not parsed")
	flag.Parse()
	app := &editorApp{editor: *editor, draft: "Keep this draft"}
	var err error
	if *inline {
		app.inline = tui.NewInlineApp(tui.WithInlineBracketedPaste(true))
		err = app.inline.Run(app)
	} else {
		terminal, openErr := tui.NewTerminal()
		if openErr != nil {
			fmt.Fprintln(os.Stderr, openErr)
			os.Exit(1)
		}
		terminal.EnableAlternateScreen()
		terminal.HideCursor()
		terminal.EnableBracketedPaste()
		app.fullscreen = tui.NewRuntime(terminal, app, 30)
		err = app.fullscreen.Run()
		err = errors.Join(err, terminal.Close())
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
