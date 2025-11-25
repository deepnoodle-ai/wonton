package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/deepnoodle-ai/gooey"
)

// InputFormsApp demonstrates a form with three different input types using Runtime.
type InputFormsApp struct {
	stage    int
	name     string
	password string
	fruit    string
	message  string
}

func (app *InputFormsApp) HandleEvent(event gooey.Event) []gooey.Cmd {
	switch e := event.(type) {
	case gooey.KeyEvent:
		switch app.stage {
		case 0:
			// Intro screen - any key starts
			app.stage = 1
			app.message = "Reading name..."
			return []gooey.Cmd{app.cmdReadName()}

		case 2:
			// After name input, any key continues
			app.stage = 3
			app.message = "Reading password..."
			return []gooey.Cmd{app.cmdReadPassword()}

		case 4:
			// After password input, any key continues
			app.stage = 5
			app.message = "Reading fruit preference..."
			return []gooey.Cmd{app.cmdReadFruit()}

		case 6:
			// After fruit input, any key continues to summary
			app.stage = 7

		case 7:
			// Summary screen - any key quits
			return []gooey.Cmd{gooey.Quit()}
		}

	case FormInputEvent:
		// Handle form input results
		if e.Error != nil {
			app.message = fmt.Sprintf("Error: %v", e.Error)
			return []gooey.Cmd{gooey.Quit()}
		}

		switch e.Field {
		case "name":
			app.name = e.Value
			app.stage = 2
			app.message = "Press any key to continue..."
		case "password":
			app.password = e.Value
			app.stage = 4
			app.message = "Press any key to continue..."
		case "fruit":
			app.fruit = e.Value
			app.stage = 6
			app.message = "Press any key to view summary..."
		}
	}

	return nil
}

func (app *InputFormsApp) Render(frame gooey.RenderFrame) {
	width, height := frame.Size()

	// Clear screen
	frame.FillStyled(0, 0, width, height, ' ', gooey.NewStyle())

	titleStyle := gooey.NewStyle().WithForeground(gooey.ColorCyan).WithBold()

	// Title
	frame.PrintStyled(0, 0, "📝 Gooey Input Form Demo", titleStyle)
	frame.PrintStyled(0, 1, "------------------------", titleStyle)

	y := 3

	switch app.stage {
	case 0:
		// Welcome screen
		frame.PrintStyled(0, y, "This demo shows a multi-step form with different input types:", gooey.NewStyle())
		y += 2
		frame.PrintStyled(0, y, "1. Basic input (name)", gooey.NewStyle())
		y++
		frame.PrintStyled(0, y, "2. Secure password input", gooey.NewStyle())
		y++
		frame.PrintStyled(0, y, "3. Input with suggestions (fruit)", gooey.NewStyle())
		y += 2
		frame.PrintStyled(0, y, "Press any key to start", gooey.NewStyle().WithForeground(gooey.ColorYellow))

	case 1:
		// Reading name
		promptStyle := gooey.NewStyle().WithForeground(gooey.ColorCyan).WithBold()
		frame.PrintStyled(0, y, "What is your name? ", promptStyle)
		y++
		frame.PrintStyled(0, y, "(Simulated input in Runtime)", gooey.NewStyle().WithForeground(gooey.ColorBrightBlack))

	case 2:
		// Show name result
		frame.PrintStyled(0, y, fmt.Sprintf("Name: %s", app.name), gooey.NewStyle().WithForeground(gooey.ColorGreen))
		y += 2
		frame.PrintStyled(0, y, app.message, gooey.NewStyle().WithForeground(gooey.ColorYellow))

	case 3:
		// Reading password
		promptStyle := gooey.NewStyle().WithForeground(gooey.ColorRed).WithBold()
		frame.PrintStyled(0, y, "Enter a password: ", promptStyle)
		y++
		frame.PrintStyled(0, y, "(Simulated secure input)", gooey.NewStyle().WithForeground(gooey.ColorBrightBlack))

	case 4:
		// Show password result
		frame.PrintStyled(0, y, fmt.Sprintf("Name: %s", app.name), gooey.NewStyle())
		y++
		frame.PrintStyled(0, y, fmt.Sprintf("Password: %s (len=%d)", strings.Repeat("*", len(app.password)), len(app.password)), gooey.NewStyle().WithForeground(gooey.ColorGreen))
		y += 2
		frame.PrintStyled(0, y, app.message, gooey.NewStyle().WithForeground(gooey.ColorYellow))

	case 5:
		// Reading fruit
		promptStyle := gooey.NewStyle().WithForeground(gooey.ColorGreen).WithBold()
		frame.PrintStyled(0, y, "Favorite fruit? ", promptStyle)
		y++
		frame.PrintStyled(0, y, "(Simulated input with suggestions)", gooey.NewStyle().WithForeground(gooey.ColorBrightBlack))
		y++
		frame.PrintStyled(0, y, "Options: Apple, Banana, Cherry, Date, Elderberry...", gooey.NewStyle().WithForeground(gooey.ColorBrightBlack))

	case 6:
		// Show all results
		frame.PrintStyled(0, y, fmt.Sprintf("Name: %s", app.name), gooey.NewStyle())
		y++
		frame.PrintStyled(0, y, fmt.Sprintf("Password: %s (len=%d)", strings.Repeat("*", len(app.password)), len(app.password)), gooey.NewStyle())
		y++
		frame.PrintStyled(0, y, fmt.Sprintf("Fruit: %s", app.fruit), gooey.NewStyle().WithForeground(gooey.ColorGreen))
		y += 2
		frame.PrintStyled(0, y, app.message, gooey.NewStyle().WithForeground(gooey.ColorYellow))

	case 7:
		// Summary
		frame.PrintStyled(0, y, "✅ Form Complete!", titleStyle)
		y += 2
		frame.PrintStyled(0, y, fmt.Sprintf("Name:     %s", app.name), gooey.NewStyle())
		y++
		frame.PrintStyled(0, y, fmt.Sprintf("Password: %s (len=%d)", strings.Repeat("*", len(app.password)), len(app.password)), gooey.NewStyle())
		y++
		frame.PrintStyled(0, y, fmt.Sprintf("Fruit:    %s", app.fruit), gooey.NewStyle())
		y += 2
		frame.PrintStyled(0, y, "Press any key to exit", gooey.NewStyle().WithForeground(gooey.ColorYellow))
	}
}

// Commands for async form input operations (simulated)
func (app *InputFormsApp) cmdReadName() gooey.Cmd {
	return func() gooey.Event {
		// Simulate reading name
		return FormInputEvent{
			Field: "name",
			Value: "John Doe",
			Error: nil,
		}
	}
}

func (app *InputFormsApp) cmdReadPassword() gooey.Cmd {
	return func() gooey.Event {
		// Simulate reading password
		return FormInputEvent{
			Field: "password",
			Value: "securepass123",
			Error: nil,
		}
	}
}

func (app *InputFormsApp) cmdReadFruit() gooey.Cmd {
	return func() gooey.Event {
		// Simulate reading fruit with suggestions
		return FormInputEvent{
			Field: "fruit",
			Value: "Apple",
			Error: nil,
		}
	}
}

// FormInputEvent is returned when form field input completes
type FormInputEvent struct {
	Field string
	Value string
	Error error
}

func (e FormInputEvent) Timestamp() time.Time {
	return time.Now()
}

func main() {
	// Create and initialize terminal
	terminal, err := gooey.NewTerminal()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create terminal: %v\n", err)
		os.Exit(1)
	}
	defer terminal.Close()

	// Create the application
	app := &InputFormsApp{
		stage:   0,
		message: "Welcome!",
	}

	// Create and run the runtime
	runtime := gooey.NewRuntime(terminal, app, 30)

	// Run blocks until the application quits
	if err := runtime.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Runtime error: %v\n", err)
		os.Exit(1)
	}
}
