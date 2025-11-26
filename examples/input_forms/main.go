package main

import (
	"log"
	"strings"

	"github.com/deepnoodle-ai/gooey"
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

func (app *InputFormsApp) View() gooey.View {
	if app.submitted {
		// Show success screen
		return gooey.VStack(
			gooey.Text("Gooey Input Forms Demo").Bold().Fg(gooey.ColorCyan),
			gooey.Text("----------------------").Fg(gooey.ColorCyan),
			gooey.Spacer().MinHeight(1),
			gooey.Text("Form submitted successfully!").Fg(gooey.ColorGreen),
			gooey.Spacer().MinHeight(1),
			gooey.Text("Name:     %s", app.name),
			gooey.Text("Email:    %s", app.email),
			gooey.Text("Password: %s", strings.Repeat("*", len(app.password))),
			gooey.Spacer().MinHeight(1),
			gooey.Text("Press Enter to exit").Dim(),
		).Padding(2)
	}

	// Build form view
	var children []gooey.View

	// Title
	children = append(children,
		gooey.Text("Gooey Input Forms Demo").Bold().Fg(gooey.ColorCyan),
		gooey.Text("----------------------").Fg(gooey.ColorCyan),
		gooey.Spacer().MinHeight(1),
	)

	// Name field
	children = append(children,
		gooey.HStack(
			gooey.Text("Name:     "),
			gooey.Input(&app.name).Placeholder("Enter your name").Width(40),
		),
		gooey.Spacer().MinHeight(1),
	)

	// Email field
	children = append(children,
		gooey.HStack(
			gooey.Text("Email:    "),
			gooey.Input(&app.email).Placeholder("Enter your email").Width(40),
		),
		gooey.Spacer().MinHeight(1),
	)

	// Password field
	children = append(children,
		gooey.HStack(
			gooey.Text("Password: "),
			gooey.Input(&app.password).
				Placeholder("Enter password").
				Width(40).
				Mask('*').
				OnSubmit(func(string) { app.validateAndSubmit() }),
		),
		gooey.Spacer().MinHeight(1),
	)

	// Show errors if any
	if len(app.errors) > 0 {
		for _, err := range app.errors {
			children = append(children,
				gooey.Text("- "+err).Fg(gooey.ColorRed),
			)
		}
		children = append(children, gooey.Spacer().MinHeight(1))
	}

	// Help text
	children = append(children,
		gooey.Text("Tab: Navigate | Enter: Submit | Esc: Quit").Dim(),
	)

	return gooey.VStack(children...).Padding(2)
}

func (app *InputFormsApp) HandleEvent(event gooey.Event) []gooey.Cmd {
	switch e := event.(type) {
	case gooey.KeyEvent:
		if app.submitted {
			// After submission, quit on Enter
			if e.Key == gooey.KeyEnter {
				return []gooey.Cmd{gooey.Quit()}
			}
		}

		// Handle quit keys
		if e.Key == gooey.KeyEscape || e.Key == gooey.KeyCtrlC {
			return []gooey.Cmd{gooey.Quit()}
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
	if err := gooey.Run(&InputFormsApp{}); err != nil {
		log.Fatal(err)
	}
}
