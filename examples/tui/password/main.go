package main

import (
	"log"

	"github.com/deepnoodle-ai/wonton/tui"
)

// PasswordDemoApp demonstrates password input using declarative View style.
type PasswordDemoApp struct {
	password  string
	submitted bool
}

func (app *PasswordDemoApp) HandleEvent(event tui.Event) []tui.Cmd {
	switch e := event.(type) {
	case tui.KeyEvent:
		if app.submitted {
			return []tui.Cmd{tui.Quit()}
		}

		switch e.Key {
		case tui.KeyEnter:
			app.submitted = true
			return nil
		case tui.KeyEscape, tui.KeyCtrlC:
			return []tui.Cmd{tui.Quit()}
		}
	}
	return nil
}

func (app *PasswordDemoApp) View() tui.View {
	headerStyle := tui.NewStyle().WithForeground(tui.ColorCyan).WithBold()
	promptStyle := tui.NewStyle().WithForeground(tui.ColorYellow)
	infoStyle := tui.NewStyle().WithForeground(tui.ColorBrightBlack)
	successStyle := tui.NewStyle().WithForeground(tui.ColorGreen)

	if app.submitted {
		return tui.VStack(
			tui.Text("Password Input Demo").Style(headerStyle),
			tui.Spacer(),
			tui.Text("Password received (%d chars)", len(app.password)).Style(successStyle),
			tui.Spacer(),
			tui.Text("Press any key to exit").Style(infoStyle),
		)
	}

	return tui.VStack(
		tui.Text("Password Input Demo").Style(headerStyle),
		tui.Spacer(),
		tui.HStack(
			tui.Text("Password: ").Style(promptStyle),
			tui.Input(&app.password).Mask('*').Placeholder("enter password").Width(30),
		),
		tui.Spacer(),
		tui.Text("Enter to submit, Esc to quit").Style(infoStyle),
	)
}

func main() {
	if err := tui.Run(&PasswordDemoApp{}); err != nil {
		log.Fatal(err)
	}
}
