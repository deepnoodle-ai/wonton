package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/deepnoodle-ai/wonton/tui"
)

// PastePlaceholderApp demonstrates the Input component's paste placeholder mode
// and the unified focus system with Tab/Shift+Tab navigation.
//
// When paste placeholder mode is enabled, multi-line pastes are collapsed into
// a single "[pasted N lines]" placeholder in the display. The actual content is
// preserved and returned by Value(). The placeholder can be deleted atomically
// with a single backspace.
type PastePlaceholderApp struct {
	input  string
	status string
}

func (app *PastePlaceholderApp) Init() error {
	app.status = "Try pasting multi-line text (e.g. some code)"
	return nil
}

func (app *PastePlaceholderApp) HandleEvent(event tui.Event) []tui.Cmd {
	switch e := event.(type) {
	case tui.KeyEvent:
		// Update status on paste
		if e.Paste != "" {
			lines := strings.Split(e.Paste, "\n")
			if len(lines) > 1 {
				app.status = "Pasted as placeholder — Backspace deletes it all at once"
			} else {
				app.status = "Pasted (single line, no placeholder)"
			}
		}

		switch e.Key {
		case tui.KeyEscape, tui.KeyCtrlC:
			return []tui.Cmd{tui.Quit()}
		}
	}
	return nil
}

func (app *PastePlaceholderApp) View() tui.View {
	// Count actual lines
	actualLines := 0
	if app.input != "" {
		actualLines = strings.Count(app.input, "\n") + 1
	}

	return tui.Stack(
		// Header
		tui.Text("Paste Placeholder Mode").Bold().Fg(tui.ColorCyan),
		tui.Text("Paste multi-line text to see it collapse into a placeholder.").Fg(tui.ColorBrightBlack),
		tui.Text("Input has MaxHeight(3) - content scrolls with ▲/▼ indicators.").Fg(tui.ColorBrightBlack),
		tui.Text("Tab/Shift+Tab to navigate between elements.").Fg(tui.ColorBrightBlack),
		tui.Spacer().MinHeight(1),

		// InputField combines label, input, and border with automatic focus styling
		// The label and border automatically highlight when the input is focused
		tui.InputField(&app.input).
			ID("paste-input").
			Label("Input: ").
			Placeholder("Type or paste here...").
			PastePlaceholder(true).
			Multiline(true).
			Bordered().
			MaxHeight(3). // Scrollable input with overflow indicators
			Width(60),

		// Status message
		tui.Spacer().MinHeight(1),
		tui.Text("%s", app.status).Fg(tui.ColorGreen),
		tui.Spacer().MinHeight(1),

		// TextArea is a high-level scrollable text viewer with automatic
		// focus-aware styling and keyboard scroll handling
		tui.TextArea(&app.input).
			ID("content-viewer").
			Title(fmt.Sprintf(" Actual Content (%d chars, %d lines) ", len(app.input), actualLines)).
			TitleStyle(tui.NewStyle().WithForeground(tui.ColorYellow)).
			FocusTitleStyle(tui.NewStyle().WithForeground(tui.ColorCyan).WithBold()).
			BorderFg(tui.ColorBrightBlack).
			FocusBorderFg(tui.ColorCyan).
			Bordered().
			Size(60, 10),

		// Help
		tui.Spacer().MinHeight(1),
		tui.Text("Shift+Enter: newline | Tab/Shift+Tab: switch focus | ↑/↓: scroll | ESC: quit").Fg(tui.ColorBrightBlack),
	)
}

func main() {
	err := tui.Run(&PastePlaceholderApp{}, tui.WithBracketedPaste(true))
	if err != nil {
		log.Fatal(err)
	}
}
