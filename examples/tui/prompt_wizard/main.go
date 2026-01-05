// Example: prompt_wizard
//
// This example demonstrates using PromptChoice for a multi-step setup wizard.
// Each step presents options with an optional custom input field.
//
// Features demonstrated:
//   - Multi-step wizard flow
//   - Different option configurations per step
//   - Progress indicator
//   - State management across steps
//   - Summary at completion
//
// Run with: go run ./examples/prompt_wizard
package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/deepnoodle-ai/wonton/tui"
)

// WizardStep defines a single step in the wizard
type WizardStep struct {
	Title       string
	Description string
	Options     []string
	InputLabel  string // Empty means no input option
}

var steps = []WizardStep{
	{
		Title:       "Project Type",
		Description: "What kind of project are you creating?",
		Options:     []string{"Web Application", "CLI Tool", "Library"},
		InputLabel:  "Other (specify)...",
	},
	{
		Title:       "Database",
		Description: "Select your database preference:",
		Options:     []string{"PostgreSQL", "SQLite", "None"},
		InputLabel:  "",
	},
	{
		Title:       "Testing Framework",
		Description: "How would you like to handle testing?",
		Options:     []string{"Standard library (testing)", "Testify", "Ginkgo"},
		InputLabel:  "Custom setup...",
	},
	{
		Title:       "CI/CD",
		Description: "Set up continuous integration?",
		Options:     []string{"GitHub Actions", "GitLab CI", "Skip for now"},
		InputLabel:  "",
	},
}

type App struct {
	runner    *tui.InlineApp
	step      int
	selected  int
	inputText string
	answers   []string // Stores the answer for each step
	done      bool
}

func (app *App) LiveView() tui.View {
	if app.done {
		return app.summaryView()
	}
	return app.stepView()
}

func (app *App) stepView() tui.View {
	currentStep := steps[app.step]

	// Progress bar
	progress := fmt.Sprintf("Step %d of %d", app.step+1, len(steps))
	progressBar := app.renderProgress()

	// Build the prompt choice
	prompt := tui.PromptChoice(&app.selected, &app.inputText)
	for _, opt := range currentStep.Options {
		prompt = prompt.Option(opt)
	}
	if currentStep.InputLabel != "" {
		prompt = prompt.InputOption(currentStep.InputLabel)
	}
	prompt = prompt.
		OnSelect(app.handleSelect).
		OnCancel(app.handleCancel).
		HintText("Esc to go back, Ctrl+C to exit")

	return tui.Stack(
		tui.Group(
			tui.Text(" %s", progress).Dim(),
			tui.Text("  "),
			tui.Text("%s", progressBar).Fg(tui.ColorCyan),
		),
		tui.Text(""),
		tui.Text(" %s", currentStep.Title).Bold(),
		tui.Text(" %s", currentStep.Description).Dim(),
		tui.Text(""),
		prompt,
	)
}

func (app *App) renderProgress() string {
	total := len(steps)
	filled := app.step
	width := 20

	filledCount := (filled * width) / total
	emptyCount := width - filledCount

	return "[" + strings.Repeat("█", filledCount) + strings.Repeat("░", emptyCount) + "]"
}

func (app *App) summaryView() tui.View {
	items := []tui.View{
		tui.Text(" Setup Complete!").Bold().Fg(tui.ColorGreen),
		tui.Text(""),
		tui.Text(" Your selections:").Dim(),
	}

	for i, step := range steps {
		answer := "skipped"
		if i < len(app.answers) {
			answer = app.answers[i]
		}
		items = append(items,
			tui.Text("   • %s: %s", step.Title, answer),
		)
	}

	items = append(items,
		tui.Text(""),
		tui.Text(" Press Enter to exit").Dim(),
	)

	return tui.Stack(items...)
}

func (app *App) handleSelect(idx int, inputText string) {
	currentStep := steps[app.step]

	// Determine the selected value
	var answer string
	if currentStep.InputLabel != "" && idx == len(currentStep.Options) {
		// Input option selected
		if inputText != "" {
			answer = inputText
		} else {
			answer = "(custom - no text entered)"
		}
	} else if idx < len(currentStep.Options) {
		answer = currentStep.Options[idx]
	}

	// Store the answer
	if len(app.answers) <= app.step {
		app.answers = append(app.answers, answer)
	} else {
		app.answers[app.step] = answer
	}

	// Log to scrollback
	app.runner.Print(tui.Group(
		tui.Text("✓").Fg(tui.ColorGreen),
		tui.Text(" %s: %s", currentStep.Title, answer),
	))

	// Move to next step or finish
	if app.step < len(steps)-1 {
		app.step++
		app.selected = 0
		app.inputText = ""
	} else {
		app.done = true
	}
}

func (app *App) handleCancel() {
	if app.step > 0 {
		// Go back to previous step
		app.step--
		app.selected = 0
		app.inputText = ""
		app.runner.Print(tui.Text("← Going back to %s", steps[app.step].Title).Dim())
	}
}

func (app *App) HandleEvent(event tui.Event) []tui.Cmd {
	if key, ok := event.(tui.KeyEvent); ok {
		if key.Key == tui.KeyCtrlC {
			return []tui.Cmd{tui.Quit()}
		}
		if app.done && key.Key == tui.KeyEnter {
			return []tui.Cmd{tui.Quit()}
		}
	}
	return nil
}

func main() {
	app := &App{
		answers: make([]string, 0, len(steps)),
	}
	app.runner = tui.NewInlineApp(tui.InlineAppConfig{Width: 60})

	fmt.Println()
	fmt.Println("  ╭────────────────────────────────────────╮")
	fmt.Println("  │       Project Setup Wizard             │")
	fmt.Println("  ╰────────────────────────────────────────╯")
	fmt.Println()
	fmt.Println("  This wizard will help you configure your new project.")
	fmt.Println("  Use arrow keys or numbers to select options.")
	fmt.Println()

	if err := app.runner.Run(app); err != nil {
		log.Fatal(err)
	}

	fmt.Println()
}
