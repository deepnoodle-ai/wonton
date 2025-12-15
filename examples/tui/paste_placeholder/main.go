package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/deepnoodle-ai/wonton/tui"
)

// PastePlaceholderApp demonstrates the Input component's paste placeholder mode.
//
// When paste placeholder mode is enabled, multi-line pastes are collapsed into
// a single "[pasted N lines]" placeholder in the display. The actual content is
// preserved and returned by Value(). The placeholder can be deleted atomically
// with a single backspace.
type PastePlaceholderApp struct {
	input        string
	status       string
	scrollY      int  // scroll position for content viewer
	contentFocus bool // true when content viewer has focus
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
		case tui.KeyTab:
			// Toggle focus between input and content viewer
			app.contentFocus = !app.contentFocus
		case tui.KeyCtrlU:
			app.input = ""
			app.scrollY = 0
			app.status = "Try pasting multi-line text (e.g. some code)"
		case tui.KeyArrowUp:
			if app.contentFocus && app.scrollY > 0 {
				app.scrollY--
			}
		case tui.KeyArrowDown:
			if app.contentFocus {
				app.scrollY++
			}
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

	// Build content lines for the scrollable view
	var contentView tui.View
	if app.input == "" {
		contentView = tui.Text("(empty)").Fg(tui.ColorBrightBlack)
	} else {
		lines := strings.Split(app.input, "\n")
		lineViews := make([]tui.View, len(lines))
		for i, line := range lines {
			if line == "" {
				lineViews[i] = tui.Text(" ") // show empty lines
			} else {
				lineViews[i] = tui.Text("%s", line).Fg(tui.ColorWhite)
			}
		}
		contentView = tui.VStack(lineViews...)
	}

	return tui.VStack(
		// Header
		tui.Text("Paste Placeholder Mode").Bold().Fg(tui.ColorCyan),
		tui.Text("Paste multi-line text to see it collapse into a placeholder.").Fg(tui.ColorBrightBlack),
		tui.Text("You can type before and after the placeholder.").Fg(tui.ColorBrightBlack),
		tui.Spacer().MinHeight(1),

		// Input using declarative Input component
		tui.HStack(
			tui.Text("Input: ").Fg(tui.ColorCyan),
			tui.Input(&app.input).
				Placeholder("Type or paste here...").
				PastePlaceholder(true).
				Multiline(true).
				Width(50),
		),
		tui.Spacer().MinHeight(1),

		// Status message
		tui.Text("%s", app.status).Fg(tui.ColorGreen),
		tui.Spacer().MinHeight(1),

		// Scrollable bordered text area for actual content
		app.contentViewer(contentView, actualLines),

		// Help
		tui.Spacer().MinHeight(1),
		tui.Text("Shift+Enter: newline | Tab: switch focus | ↑/↓: scroll | Ctrl+U: reset | ESC: quit").Fg(tui.ColorBrightBlack),
	)
}

func (app *PastePlaceholderApp) contentViewer(content tui.View, lineCount int) tui.View {
	borderColor := tui.ColorBrightBlack
	if app.contentFocus {
		borderColor = tui.ColorCyan
	}
	return tui.Size(60, 10, tui.Bordered(
		tui.Scroll(content, &app.scrollY),
	).Border(&tui.RoundedBorder).
		Title(fmt.Sprintf(" Actual Content (%d chars, %d lines) ", len(app.input), lineCount)).
		TitleStyle(tui.NewStyle().WithForeground(tui.ColorYellow)).
		BorderFg(borderColor))
}

func main() {
	err := tui.Run(&PastePlaceholderApp{}, tui.WithBracketedPaste(true))
	if err != nil {
		log.Fatal(err)
	}
}
