// Example: prompt_choice
//
// This example demonstrates PromptChoice - a Claude Code-style selection widget
// with numbered options and optional inline text input.
//
// Features demonstrated:
//   - Fixed options with number key shortcuts
//   - Arrow key navigation
//   - Inline text input option
//   - Enter to confirm, Escape to cancel
//   - Custom styling and cursor
//
// Run with: go run ./examples/prompt_choice
package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/deepnoodle-ai/wonton/tui"
)

type App struct {
	runner    *tui.InlineApp
	selected  int
	inputText string
	state     string // "prompt", "result"
	result    string
}

func (app *App) LiveView() tui.View {
	switch app.state {
	case "result":
		return app.resultView()
	default:
		return app.promptView()
	}
}

func (app *App) promptView() tui.View {
	return tui.Stack(
		tui.Text(" Claude wants to edit files in your project").Bold(),
		tui.Text(""),
		tui.Text(" This will modify the following files:").Dim(),
		tui.Text("   • src/main.go").Dim(),
		tui.Text("   • src/config.go").Dim(),
		tui.Text("   • tests/main_test.go").Dim(),
		tui.Text(""),
		tui.PromptChoice(&app.selected, &app.inputText).
			Option("Yes, allow this edit").
			Option("Yes, and trust this project").
			InputOption("Tell Claude what to do differently...").
			OnSelect(app.handleSelect).
			OnCancel(app.handleCancel).
			CursorStyle(tui.NewStyle().WithForeground(tui.ColorCyan)),
	)
}

func (app *App) resultView() tui.View {
	icon := "✓"
	style := tui.NewStyle().WithForeground(tui.ColorGreen)
	if strings.HasPrefix(app.result, "Cancelled") {
		icon = "✗"
		style = tui.NewStyle().WithForeground(tui.ColorRed)
	}

	return tui.Stack(
		tui.Group(
			tui.Text("%s", icon).Style(style),
			tui.Text(" %s", app.result),
		),
		tui.Text(""),
		tui.Text(" Press Enter or 'q' to exit").Dim(),
	)
}

func (app *App) handleSelect(idx int, inputText string) {
	app.state = "result"
	switch idx {
	case 0:
		app.result = "Approved: Allowing this edit"
		app.runner.Print(tui.Text("User approved the edit").Fg(tui.ColorGreen))
	case 1:
		app.result = "Approved: Project trusted for future edits"
		app.runner.Print(tui.Text("User trusted the project").Fg(tui.ColorGreen))
	case 2:
		app.result = fmt.Sprintf("Custom instructions: %q", inputText)
		app.runner.Print(tui.Text("User provided instructions: %s", inputText).Fg(tui.ColorYellow))
	}
}

func (app *App) handleCancel() {
	app.state = "result"
	app.result = "Cancelled by user"
	app.runner.Print(tui.Text("User cancelled the operation").Fg(tui.ColorRed))
}

func (app *App) HandleEvent(event tui.Event) []tui.Cmd {
	if app.state == "result" {
		if key, ok := event.(tui.KeyEvent); ok {
			if key.Key == tui.KeyEnter || key.Rune == 'q' || key.Key == tui.KeyCtrlC {
				return []tui.Cmd{tui.Quit()}
			}
		}
	}
	return nil
}

func main() {
	app := &App{state: "prompt"}
	app.runner = tui.NewInlineApp(tui.InlineAppConfig{Width: 60})

	fmt.Println()
	fmt.Println("  PromptChoice Demo - Claude Code Style Confirmation")
	fmt.Println("  " + strings.Repeat("─", 46))
	fmt.Println()
	fmt.Println("  Controls:")
	fmt.Println("    ↑/↓     Navigate options")
	fmt.Println("    1-3     Jump to option")
	fmt.Println("    Enter   Confirm selection")
	fmt.Println("    Esc     Cancel")
	fmt.Println("    (type)  Enter custom text on option 3")
	fmt.Println()

	if err := app.runner.Run(app); err != nil {
		log.Fatal(err)
	}
}
