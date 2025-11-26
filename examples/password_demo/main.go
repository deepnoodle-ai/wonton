package main

import (
	"fmt"
	"log"

	"github.com/deepnoodle-ai/gooey"
)

// PasswordDemoApp demonstrates password input using declarative View style.
type PasswordDemoApp struct {
	password  string
	submitted bool
}

func (app *PasswordDemoApp) HandleEvent(event gooey.Event) []gooey.Cmd {
	switch e := event.(type) {
	case gooey.KeyEvent:
		if app.submitted {
			return []gooey.Cmd{gooey.Quit()}
		}

		switch e.Key {
		case gooey.KeyEnter:
			app.submitted = true
			return nil
		case gooey.KeyEscape, gooey.KeyCtrlC:
			return []gooey.Cmd{gooey.Quit()}
		}
	}
	return nil
}

func (app *PasswordDemoApp) View() gooey.View {
	headerStyle := gooey.NewStyle().WithForeground(gooey.ColorCyan).WithBold()
	promptStyle := gooey.NewStyle().WithForeground(gooey.ColorYellow)
	infoStyle := gooey.NewStyle().WithForeground(gooey.ColorBrightBlack)
	successStyle := gooey.NewStyle().WithForeground(gooey.ColorGreen)

	if app.submitted {
		return gooey.VStack(
			gooey.Text("Password Input Demo").Style(headerStyle),
			gooey.Spacer(),
			gooey.Text(fmt.Sprintf("Password received (%d chars)", len(app.password))).Style(successStyle),
			gooey.Spacer(),
			gooey.Text("Press any key to exit").Style(infoStyle),
		)
	}

	return gooey.VStack(
		gooey.Text("Password Input Demo").Style(headerStyle),
		gooey.Spacer(),
		gooey.HStack(
			gooey.Text("Password: ").Style(promptStyle),
			gooey.Input(&app.password).Mask('*').Placeholder("enter password").Width(30),
		),
		gooey.Spacer(),
		gooey.Text("Enter to submit, Esc to quit").Style(infoStyle),
	)
}

func main() {
	if err := gooey.Run(&PasswordDemoApp{}); err != nil {
		log.Fatal(err)
	}
}
