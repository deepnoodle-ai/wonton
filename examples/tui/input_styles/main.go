package main

import (
	"log"

	"github.com/deepnoodle-ai/wonton/tui"
)

// InputStylesApp demonstrates the new InputField styles and cursor options
type InputStylesApp struct {
	input1 string
	input2 string
	input3 string
	input4 string
	input5 string
}

func (app *InputStylesApp) Init() error {
	return nil
}

func (app *InputStylesApp) HandleEvent(event tui.Event) []tui.Cmd {
	switch e := event.(type) {
	case tui.KeyEvent:
		switch e.Key {
		case tui.KeyEscape, tui.KeyCtrlC:
			return []tui.Cmd{tui.Quit()}
		}
	}
	return nil
}

func (app *InputStylesApp) View() tui.View {
	return tui.Stack(
		// Header
		tui.Text("InputField Styles & Cursor Options Demo").Bold().Fg(tui.ColorCyan),
		tui.Text("Various InputField border styles and cursor configurations").Fg(tui.ColorBrightBlack),
		tui.Spacer().MinHeight(1),

		// 1. Standard bordered input with block cursor
		tui.InputField(&app.input1).
			ID("input1").
			Label("Standard Border:").
			Placeholder("Block cursor (default)").
			Bordered().
			Width(60),

		tui.Spacer().MinHeight(1),

		// 2. Horizontal bar border with underline cursor
		tui.InputField(&app.input2).
			ID("input2").
			Label("Horizontal Bar:").
			Placeholder("Underline cursor").
			HorizontalBorderOnly().
			CursorShape(tui.InputCursorUnderline).
			CursorColor(tui.ColorCyan).
			Width(60),

		tui.Spacer().MinHeight(1),

		// 3. Prompt style with bar cursor
		tui.InputField(&app.input3).
			ID("input3").
			Label("With Prompt:").
			Placeholder("Bar/beam cursor").
			Prompt("❯").
			PromptStyle(tui.NewStyle().WithForeground(tui.ColorMagenta)).
			CursorShape(tui.InputCursorBar).
			CursorColor(tui.ColorGreen).
			Width(60),

		tui.Spacer().MinHeight(1),

		// 4. Horizontal bar + prompt + custom cursor color
		tui.InputField(&app.input4).
			ID("input4").
			Label("Bar + Prompt:").
			Placeholder("Cyan block cursor").
			HorizontalBorderOnly().
			Prompt(">").
			PromptStyle(tui.NewStyle().WithForeground(tui.ColorYellow).WithBold()).
			CursorColor(tui.ColorCyan).
			BorderFg(tui.ColorBrightBlack).
			FocusBorderFg(tui.ColorYellow).
			Width(60),

		tui.Spacer().MinHeight(1),

		// 5. Simple input with blinking bar cursor
		tui.InputField(&app.input5).
			ID("input5").
			Label("Blinking Bar:").
			Placeholder("Blinking bar cursor").
			CursorShape(tui.InputCursorBar).
			CursorColor(tui.ColorRed).
			CursorBlink(true).
			Width(60),

		// Help
		tui.Spacer().MinHeight(2),
		tui.Text("Tab/Shift+Tab: navigate | ESC: quit").Fg(tui.ColorBrightBlack),
	)
}

func main() {
	err := tui.Run(&InputStylesApp{})
	if err != nil {
		log.Fatal(err)
	}
}
