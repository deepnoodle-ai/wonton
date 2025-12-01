package main

import (
	"log"
	"strings"

	"github.com/deepnoodle-ai/gooey/tui"
)

// InputFormsApp demonstrates a multi-field form using declarative UI.
// It shows how to handle multiple input fields with automatic focus management,
// form validation, and submission.
type InputFormsApp struct {
	// Form field values
	name     string
	email    string
	password string

	// Form state
	submitted bool
	errors    []string
}

func (app *InputFormsApp) View() tui.View {
	if app.submitted {
		// Show success screen
		return tui.VStack(
			tui.Text("Gooey Input Forms Demo").Bold().Fg(tui.ColorCyan),
			tui.Text("----------------------").Fg(tui.ColorCyan),
			tui.Spacer().MinHeight(1),
			tui.Text("Form submitted successfully!").Fg(tui.ColorGreen),
			tui.Spacer().MinHeight(1),
			tui.Text("Name:     %s", app.name),
			tui.Text("Email:    %s", app.email),
			tui.Text("Password: %s", strings.Repeat("*", len(app.password))),
			tui.Spacer().MinHeight(1),
			tui.Text("Press Enter to exit").Dim(),
		).Padding(2)
	}

	// Build form view
	var children []tui.View

	// Title
	children = append(children,
		tui.Text("Gooey Input Forms Demo").Bold().Fg(tui.ColorCyan),
		tui.Text("----------------------").Fg(tui.ColorCyan),
		tui.Spacer().MinHeight(1),
	)

	// Name field
	children = append(children,
		tui.HStack(
			tui.Text("Name:     "),
			tui.Input(&app.name).Placeholder("Enter your name").Width(40),
		),
		tui.Spacer().MinHeight(1),
	)

	// Email field
	children = append(children,
		tui.HStack(
			tui.Text("Email:    "),
			tui.Input(&app.email).Placeholder("Enter your email").Width(40),
		),
		tui.Spacer().MinHeight(1),
	)

	// Password field
	children = append(children,
		tui.HStack(
			tui.Text("Password: "),
			tui.Input(&app.password).
				Placeholder("Enter password").
				Width(40).
				Mask('*').
				OnSubmit(func(string) { app.validateAndSubmit() }),
		),
		tui.Spacer().MinHeight(1),
	)

	// Show errors if any
	if len(app.errors) > 0 {
		for _, err := range app.errors {
			children = append(children,
				tui.Text("- %s", err).Fg(tui.ColorRed),
			)
		}
		children = append(children, tui.Spacer().MinHeight(1))
	}

	// Help text
	children = append(children,
		tui.Text("Tab: Navigate | Enter: Submit | Esc: Quit").Dim(),
	)

	return tui.VStack(children...).Padding(2)
}

func (app *InputFormsApp) HandleEvent(event tui.Event) []tui.Cmd {
	switch e := event.(type) {
	case tui.KeyEvent:
		if app.submitted {
			// After submission, quit on Enter
			if e.Key == tui.KeyEnter {
				return []tui.Cmd{tui.Quit()}
			}
		}

		// Handle quit keys
		if e.Key == tui.KeyEscape || e.Key == tui.KeyCtrlC {
			return []tui.Cmd{tui.Quit()}
		}
	}

	return nil
}

func (app *InputFormsApp) validateAndSubmit() {
	app.errors = nil

	// Validate name
	if strings.TrimSpace(app.name) == "" {
		app.errors = append(app.errors, "Name is required")
	}

	// Validate email (simple check)
	email := strings.TrimSpace(app.email)
	if email == "" {
		app.errors = append(app.errors, "Email is required")
	} else if !strings.Contains(email, "@") {
		app.errors = append(app.errors, "Email must contain @")
	}

	// Validate password
	if len(app.password) < 4 {
		app.errors = append(app.errors, "Password must be at least 4 characters")
	}

	if len(app.errors) == 0 {
		app.submitted = true
	}
}

func main() {
	if err := tui.Run(&InputFormsApp{}); err != nil {
		log.Fatal(err)
	}
}
