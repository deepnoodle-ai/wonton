package main

import (
	"fmt"
	"image"
	"log"
	"strings"

	"github.com/deepnoodle-ai/gooey"
)

// InputFormsApp demonstrates a multi-field form using TextInput widgets.
// It shows how to handle multiple input fields with focus management,
// form validation, and submission.
type InputFormsApp struct {
	// Form fields
	nameInput     *gooey.TextInput
	emailInput    *gooey.TextInput
	passwordInput *gooey.TextInput

	// Which field is focused (0=name, 1=email, 2=password)
	focusedField int

	// Form state
	submitted bool
	errors    []string
}

func (app *InputFormsApp) Init() error {
	// Create name input
	app.nameInput = gooey.NewTextInput().
		WithPlaceholder("Enter your name")
	app.nameInput.SetFocused(true) // Start with name focused

	// Create email input
	app.emailInput = gooey.NewTextInput().
		WithPlaceholder("Enter your email")

	// Create password input with masking
	app.passwordInput = gooey.NewTextInput().
		WithPlaceholder("Enter password").
		WithMask('*')

	return nil
}

func (app *InputFormsApp) HandleEvent(event gooey.Event) []gooey.Cmd {
	switch e := event.(type) {
	case gooey.KeyEvent:
		// Handle form-level keys first
		switch e.Key {
		case gooey.KeyTab:
			// Move to next field
			app.cycleFocus(1)
			return nil

		case gooey.KeyArrowDown:
			// Move to next field
			app.cycleFocus(1)
			return nil

		case gooey.KeyArrowUp:
			// Move to previous field
			app.cycleFocus(-1)
			return nil

		case gooey.KeyEnter:
			if app.submitted {
				// Already submitted, quit on Enter
				return []gooey.Cmd{gooey.Quit()}
			}
			// Move to next field, or submit if on last field
			if app.focusedField < 2 {
				app.cycleFocus(1)
			} else {
				app.validateAndSubmit()
			}
			return nil

		case gooey.KeyEscape, gooey.KeyCtrlC:
			return []gooey.Cmd{gooey.Quit()}
		}

		// Pass key events to the focused input
		app.getFocusedInput().HandleKey(e)
	}

	return nil
}

func (app *InputFormsApp) cycleFocus(direction int) {
	// Clear focus on current
	app.getFocusedInput().SetFocused(false)

	// Move to next/prev field
	app.focusedField = (app.focusedField + direction + 3) % 3

	// Set focus on new field
	app.getFocusedInput().SetFocused(true)
}

func (app *InputFormsApp) getFocusedInput() *gooey.TextInput {
	switch app.focusedField {
	case 0:
		return app.nameInput
	case 1:
		return app.emailInput
	case 2:
		return app.passwordInput
	default:
		return app.nameInput
	}
}

func (app *InputFormsApp) validateAndSubmit() {
	app.errors = nil

	// Validate name
	if strings.TrimSpace(app.nameInput.Value()) == "" {
		app.errors = append(app.errors, "Name is required")
	}

	// Validate email (simple check)
	email := strings.TrimSpace(app.emailInput.Value())
	if email == "" {
		app.errors = append(app.errors, "Email is required")
	} else if !strings.Contains(email, "@") {
		app.errors = append(app.errors, "Email must contain @")
	}

	// Validate password
	if len(app.passwordInput.Value()) < 4 {
		app.errors = append(app.errors, "Password must be at least 4 characters")
	}

	if len(app.errors) == 0 {
		app.submitted = true
	}
}

func (app *InputFormsApp) Render(frame gooey.RenderFrame) {
	width, height := frame.Size()

	// Styles
	titleStyle := gooey.NewStyle().WithForeground(gooey.ColorCyan).WithBold()
	labelStyle := gooey.NewStyle().WithForeground(gooey.ColorWhite)
	focusedLabelStyle := gooey.NewStyle().WithForeground(gooey.ColorGreen).WithBold()
	errorStyle := gooey.NewStyle().WithForeground(gooey.ColorRed)
	successStyle := gooey.NewStyle().WithForeground(gooey.ColorGreen)
	helpStyle := gooey.NewStyle().WithForeground(gooey.ColorBrightBlack)

	// Clear screen
	frame.FillStyled(0, 0, width, height, ' ', gooey.NewStyle())

	// Title
	frame.PrintStyled(2, 1, "Gooey Input Forms Demo", titleStyle)
	frame.PrintStyled(2, 2, "----------------------", titleStyle)

	if app.submitted {
		// Show success message
		frame.PrintStyled(2, 4, "Form submitted successfully!", successStyle)
		frame.PrintStyled(2, 6, fmt.Sprintf("Name:     %s", app.nameInput.Value()), labelStyle)
		frame.PrintStyled(2, 7, fmt.Sprintf("Email:    %s", app.emailInput.Value()), labelStyle)
		frame.PrintStyled(2, 8, fmt.Sprintf("Password: %s", strings.Repeat("*", len(app.passwordInput.Value()))), labelStyle)
		frame.PrintStyled(2, 10, "Press Enter to exit", helpStyle)
		return
	}

	y := 4
	labelX := 2
	inputX := 14 // Fixed position for all inputs

	// Name field
	if app.focusedField == 0 {
		frame.PrintStyled(labelX, y, "> Name:", focusedLabelStyle)
	} else {
		frame.PrintStyled(labelX, y, "  Name:", labelStyle)
	}
	app.nameInput.SetBounds(image.Rect(inputX, y, inputX+40, y+1))
	app.nameInput.Draw(frame)
	y += 2

	// Email field
	if app.focusedField == 1 {
		frame.PrintStyled(labelX, y, "> Email:", focusedLabelStyle)
	} else {
		frame.PrintStyled(labelX, y, "  Email:", labelStyle)
	}
	app.emailInput.SetBounds(image.Rect(inputX, y, inputX+40, y+1))
	app.emailInput.Draw(frame)
	y += 2

	// Password field
	if app.focusedField == 2 {
		frame.PrintStyled(labelX, y, "> Password:", focusedLabelStyle)
	} else {
		frame.PrintStyled(labelX, y, "  Password:", labelStyle)
	}
	app.passwordInput.SetBounds(image.Rect(inputX, y, inputX+40, y+1))
	app.passwordInput.Draw(frame)
	y += 2

	// Show errors if any
	if len(app.errors) > 0 {
		y++
		for _, err := range app.errors {
			frame.PrintStyled(2, y, "- "+err, errorStyle)
			y++
		}
	}

	// Help text
	y += 2
	frame.PrintStyled(2, y, "Tab/Arrow keys: Navigate | Enter: Submit | Esc: Quit", helpStyle)
}

func main() {
	if err := gooey.Run(&InputFormsApp{}); err != nil {
		log.Fatal(err)
	}
}
