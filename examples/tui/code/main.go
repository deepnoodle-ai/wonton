// Command code displays a syntax-highlighted file in a scrollable view.
package main

import (
	"fmt"
	"os"

	"github.com/deepnoodle-ai/wonton/cli"
	"github.com/deepnoodle-ai/wonton/tui"
)

func main() {
	app := cli.New("code").
		Description("Display syntax-highlighted code").
		Args("file").
		Run(func(ctx *cli.Context) error {
			path := ctx.Arg(0)
			if path == "" {
				return fmt.Errorf("usage: code <file>")
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			return tui.Run(&codeApp{code: string(data)})
		})

	// Execute returns the handler's error rather than reporting it, so an
	// unreadable file has to be printed here or the program exits 0 in silence.
	if err := app.Execute(); err != nil {
		app.PrintError(err)
		os.Exit(cli.GetExitCode(err))
	}
}

type codeApp struct {
	code          string
	scrollY       int
	width, height int
}

func (app *codeApp) HandleEvent(event tui.Event) []tui.Cmd {
	switch e := event.(type) {
	case tui.ResizeEvent:
		app.width, app.height = e.Width, e.Height
	case tui.KeyEvent:
		if e.Rune == 'q' || e.Key == tui.KeyEscape || e.Key == tui.KeyCtrlC {
			return []tui.Cmd{tui.Quit()}
		}
		page := max(app.viewHeight()-1, 1)
		switch e.Key {
		case tui.KeyArrowUp:
			app.scrollBy(-1)
		case tui.KeyArrowDown:
			app.scrollBy(1)
		case tui.KeyPageUp:
			app.scrollBy(-page)
		case tui.KeyPageDown:
			app.scrollBy(page)
		case tui.KeyHome:
			app.scrollY = 0
		case tui.KeyEnd:
			app.scrollY = app.maxScroll()
		}
	}
	return nil
}

// viewHeight is the space the code has, once the status line is taken out.
func (app *codeApp) viewHeight() int { return max(app.height-1, 1) }

// maxScroll is roughly how far the content can move before its last line
// reaches the bottom. Measure reports the height the code would render at,
// without a terminal to render it to. It measures the content alone, so the
// answer is an upper bound — ScrollView clamps scrollY to the exact bottom
// when it renders, and writes the clamped value back.
func (app *codeApp) maxScroll() int {
	if app.width == 0 {
		return 0
	}
	_, contentHeight := tui.Measure(app.codeView(), app.width, 0)
	return max(contentHeight-app.viewHeight(), 0)
}

func (app *codeApp) scrollBy(lines int) {
	app.scrollY = min(max(app.scrollY+lines, 0), app.maxScroll())
}

func (app *codeApp) codeView() tui.View { return tui.Code(app.code, "go") }

func (app *codeApp) View() tui.View {
	return tui.Stack(
		tui.Scroll(app.codeView(), &app.scrollY),
		tui.Text(" q quit · ↑↓ scroll · PgUp/PgDn page · Home/End jump ").
			Style(tui.NewStyle().WithBackground(tui.ColorBlue).WithForeground(tui.ColorWhite)),
	)
}
