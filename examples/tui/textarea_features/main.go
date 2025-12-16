package main

import (
	"log"
	"strings"

	"github.com/deepnoodle-ai/wonton/tui"
)

// TextAreaFeaturesApp demonstrates the new TextArea features:
// 1. Left border only
// 2. Line numbers
// 3. Current line highlighting
type TextAreaFeaturesApp struct {
	content    string
	cursorLine int
	scrollY    int
}

func (app *TextAreaFeaturesApp) Init() error {
	app.content = `package main

import (
	"fmt"
	"log"
)

func main() {
	fmt.Println("Hello, World!")
	log.Println("This is a demo")

	// More lines to demonstrate scrolling
	for i := 0; i < 10; i++ {
		fmt.Printf("Line %d\n", i)
	}

	fmt.Println("End of demo")
}`
	return nil
}

func (app *TextAreaFeaturesApp) HandleEvent(event tui.Event) []tui.Cmd {
	switch e := event.(type) {
	case tui.KeyEvent:
		switch e.Key {
		case tui.KeyEscape, tui.KeyCtrlC:
			return []tui.Cmd{tui.Quit()}
		}
	}
	return nil
}

func (app *TextAreaFeaturesApp) View() tui.View {
	lineCount := len(strings.Split(app.content, "\n"))

	return tui.Stack(
		// Header
		tui.Text("TextArea Features Demo").Bold().Fg(tui.ColorCyan),
		tui.Text("Use arrow keys, PageUp/PageDown, Home/End to navigate").Fg(tui.ColorBrightBlack),
		tui.Text("Tab/Shift+Tab to switch between viewers | ESC: quit").Fg(tui.ColorBrightBlack),
		tui.Spacer().MinHeight(1),

		// Example 1: Line numbers only
		tui.Text("1. Line Numbers (no border)").Bold().Fg(tui.ColorYellow),
		tui.TextArea(&app.content).
			ID("viewer-1").
			LineNumbers(true).
			LineNumberFg(tui.ColorBrightBlack).
			Size(70, 8),

		tui.Spacer().MinHeight(1),

		// Example 2: Left border only with line numbers
		tui.Text("2. Left Border + Line Numbers").Bold().Fg(tui.ColorYellow),
		tui.TextArea(&app.content).
			ID("viewer-2").
			LeftBorderOnly().
			BorderFg(tui.ColorCyan).
			LineNumbers(true).
			LineNumberFg(tui.ColorBrightBlue).
			Size(70, 8),

		tui.Spacer().MinHeight(1),

		// Example 3: Current line highlighting with line numbers
		tui.Text("3. Current Line Highlighting + Line Numbers (arrow keys move cursor)").Bold().Fg(tui.ColorYellow),
		tui.Text("Current line: %d/%d", app.cursorLine+1, lineCount).Fg(tui.ColorBrightBlack),
		tui.TextArea(&app.content).
			ID("viewer-3").
			LeftBorderOnly().
			FocusBorderFg(tui.ColorGreen).
			LineNumbers(true).
			LineNumberFg(tui.ColorBrightBlue).
			HighlightCurrentLine(true).
			CurrentLineStyle(tui.NewStyle().WithBackground(tui.ColorBrightBlack)).
			CursorLine(&app.cursorLine).
			ScrollY(&app.scrollY).
			Size(70, 10),

		tui.Spacer().MinHeight(1),

		// Example 4: Full border with all features
		tui.Text("4. Full Border + All Features").Bold().Fg(tui.ColorYellow),
		tui.TextArea(&app.content).
			ID("viewer-4").
			Title("Code Viewer").
			Bordered().
			FocusBorderFg(tui.ColorMagenta).
			LineNumbers(true).
			HighlightCurrentLine(true).
			CursorLine(&app.cursorLine).
			Size(70, 8),
	)
}

func main() {
	err := tui.Run(&TextAreaFeaturesApp{})
	if err != nil {
		log.Fatal(err)
	}
}
