package main

import (
	"fmt"
	"image"
	"log"

	"github.com/deepnoodle-ai/gooey"
)

// PasswordDemoApp demonstrates password input using TextInput with masking.
type PasswordDemoApp struct {
	input     *gooey.TextInput
	submitted bool
	password  string
}

func (app *PasswordDemoApp) Init() error {
	app.input = gooey.NewTextInput().
		WithMask('*').
		WithPlaceholder("enter password")
	app.input.SetFocused(true)
	app.input.SetBounds(image.Rect(0, 0, 30, 1))
	return nil
}

func (app *PasswordDemoApp) HandleEvent(event gooey.Event) []gooey.Cmd {
	switch e := event.(type) {
	case gooey.KeyEvent:
		if app.submitted {
			return []gooey.Cmd{gooey.Quit()}
		}

		switch e.Key {
		case gooey.KeyEnter:
			app.password = app.input.Value()
			app.submitted = true
			return nil
		case gooey.KeyEscape, gooey.KeyCtrlC:
			return []gooey.Cmd{gooey.Quit()}
		default:
			app.input.HandleKey(e)
		}
	}
	return nil
}

func (app *PasswordDemoApp) Render(frame gooey.RenderFrame) {
	width, height := frame.Size()
	frame.FillStyled(0, 0, width, height, ' ', gooey.NewStyle())

	headerStyle := gooey.NewStyle().WithForeground(gooey.ColorCyan).WithBold()
	promptStyle := gooey.NewStyle().WithForeground(gooey.ColorYellow)
	infoStyle := gooey.NewStyle().WithForeground(gooey.ColorBrightBlack)
	successStyle := gooey.NewStyle().WithForeground(gooey.ColorGreen)

	frame.PrintStyled(0, 0, "Password Input Demo", headerStyle)

	if app.submitted {
		frame.PrintStyled(0, 2, fmt.Sprintf("Password received (%d chars)", len(app.password)), successStyle)
		frame.PrintStyled(0, 4, "Press any key to exit", infoStyle)
	} else {
		prompt := "Password: "
		frame.PrintStyled(0, 2, prompt, promptStyle)

		inputX := len(prompt)
		app.input.SetBounds(image.Rect(inputX, 2, inputX+30, 3))
		app.input.Draw(frame)

		frame.PrintStyled(0, 4, "Enter to submit, Esc to quit", infoStyle)
	}
}

func main() {
	if err := gooey.Run(&PasswordDemoApp{}); err != nil {
		log.Fatal(err)
	}
}
