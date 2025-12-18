// Example: Syntax-Highlighted Code Viewer
//
// Displays a Go source file with syntax highlighting.
//
// Run with: go run ./examples/tui/code main.go
package main

import (
	"os"

	"github.com/deepnoodle-ai/wonton/cli"
	"github.com/deepnoodle-ai/wonton/tui"
)

func main() {
	cli.New("code").
		Description("Display syntax-highlighted code").
		Args("file").
		Run(func(ctx *cli.Context) error {
			data, err := os.ReadFile(ctx.Arg(0))
			if err != nil {
				return err
			}
			return tui.Run(&codeApp{code: string(data)})
		}).
		Execute()
}

type codeApp struct {
	code    string
	scrollY int
}

func (app *codeApp) HandleEvent(event tui.Event) []tui.Cmd {
	switch e := event.(type) {
	case tui.KeyEvent:
		if e.Rune == 'q' || e.Key == tui.KeyEscape {
			return []tui.Cmd{tui.Quit()}
		}
		if e.Key == tui.KeyArrowDown {
			app.scrollY++
		}
		if e.Key == tui.KeyArrowUp && app.scrollY > 0 {
			app.scrollY--
		}
	}
	return nil
}

func (app *codeApp) View() tui.View {
	return tui.Code(app.code, "go").ScrollY(&app.scrollY)
}
