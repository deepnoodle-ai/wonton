// Example: Syntax-Highlighted Code Viewer
//
// Displays a source file with syntax highlighting in a scrollable view.
//
// Run with: go run ./examples/tui/code ./examples/tui/code/main.go
package main

import (
	"os"

	"github.com/deepnoodle-ai/wonton/cli"
	"github.com/deepnoodle-ai/wonton/tui"
)

func main() {
	app := cli.New("code").
		Description("Display syntax-highlighted code").
		Args("file").
		Run(func(ctx *cli.Context) error {
			data, err := os.ReadFile(ctx.Arg(0))
			if err != nil {
				return err
			}
			return tui.Run(&codeApp{code: string(data)})
		})

	if err := app.Execute(); err != nil {
		app.PrintError(err)
		os.Exit(cli.GetExitCode(err))
	}
}

type codeApp struct {
	code    string
	scrollY int
}

func (app *codeApp) HandleEvent(event tui.Event) []tui.Cmd {
	switch e := event.(type) {
	case tui.KeyEvent:
		switch {
		case e.Rune == 'q' || e.Key == tui.KeyEscape || e.Key == tui.KeyCtrlC:
			return []tui.Cmd{tui.Quit()}
		case e.Key == tui.KeyArrowUp:
			app.scrollBy(-1)
		case e.Key == tui.KeyArrowDown:
			app.scrollBy(1)
		case e.Key == tui.KeyPageUp:
			app.scrollBy(-10)
		case e.Key == tui.KeyPageDown:
			app.scrollBy(10)
		case e.Key == tui.KeyHome:
			app.scrollY = 0
		case e.Key == tui.KeyEnd:
			app.scrollY = 1 << 30 // Clamped to the bottom by the scroll view
		}
	}
	return nil
}

// scrollBy adjusts the scroll offset; the scroll view clamps the maximum.
func (app *codeApp) scrollBy(delta int) {
	app.scrollY += delta
	if app.scrollY < 0 {
		app.scrollY = 0
	}
}

func (app *codeApp) View() tui.View {
	return tui.Stack(
		tui.Scroll(
			tui.Code(app.code, "go"),
			&app.scrollY,
		),
		tui.Text(" ↑↓ scroll | PgUp/PgDn page | Home/End jump | q quit ").Dim(),
	)
}
