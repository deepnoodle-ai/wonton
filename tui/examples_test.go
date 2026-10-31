package tui_test

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/deepnoodle-ai/wonton/termtest"
	"github.com/deepnoodle-ai/wonton/tui"
)

// ExampleApplication demonstrates the basic Application interface
// for building declarative TUI applications.
func ExampleApplication() {
	type CounterApp struct{}

	// View returns the UI tree based on current state
	// func (app *CounterApp) View() tui.View {
	// 	return tui.Stack(
	// 		tui.Text("Count: %d", app.count),
	// 		tui.Button("Increment", func() { app.count++ }),
	// 	)
	// }

	app := &CounterApp{}
	_ = app

	// Run the application
	// tui.Run(app)
}

// ExampleStack demonstrates vertical layout of views.
func ExampleStack() {
	// Stack arranges children vertically
	view := tui.Stack(
		tui.Text("Header").Bold(),
		tui.Text("Body content"),
		tui.Spacer(),
		tui.Text("Footer").Dim(),
	).Gap(1) // 1 row between children

	_ = view
	// Output:
}

// ExampleText demonstrates text rendering with styling.
func ExampleText() {
	// Basic text
	_ = tui.Text("Hello, World!")

	// Text with formatting
	name := "Alice"
	_ = tui.Text("Hello, %s!", name)

	// Text with styling
	_ = tui.Text("Success").Fg(tui.ColorGreen).Bold()

	// Semantic styling
	_ = tui.Text("Error occurred").Error()
	_ = tui.Text("Warning message").Warning()
	_ = tui.Text("Helpful hint").Hint()

	// Output:
}

// ExampleSpacer demonstrates flexible space distribution.
func ExampleSpacer() {
	// Push content to opposite edges
	view := tui.Stack(
		tui.Text("Top"),
		tui.Spacer(), // Fills all available space
		tui.Text("Bottom"),
	)

	_ = view
	// Output:
}

// ExampleButton demonstrates interactive buttons.
func ExampleButton() {
	type App struct {
		count int
	}
	app := &App{}

	// Button with callback
	button := tui.Button("Click Me", func() {
		app.count++
	})

	// Styled button
	_ = tui.Button("Submit", func() {}).
		Fg(tui.ColorGreen).
		Bold()

	_ = button
	// Output:
}

// ExampleInputField demonstrates text input fields.
func ExampleInputField() {
	type App struct {
		name  string
		email string
	}
	app := &App{}

	// Basic input field
	input := tui.InputField(&app.name).
		Label("Name:").
		Placeholder("Enter your name")

	// Input with border and custom styling
	_ = tui.InputField(&app.email).
		Label("Email").
		Placeholder("user@example.com").
		Bordered().
		FocusBorderFg(tui.ColorCyan).
		Width(40)

	_ = input
	// Output:
}

// ExampleCmd demonstrates async command execution.
func ExampleCmd() {
	// type DataEvent struct {
	// 	Time time.Time
	// 	Data string
	// }

	// Custom event type needs Timestamp method
	// func (e DataEvent) Timestamp() time.Time { return e.Time }

	// Command that performs async work
	// fetchData := func() tui.Cmd {
	// 	return func() tui.Event {
	// 		// Simulate async operation
	// 		data := "fetched data"
	// 		return DataEvent{Time: time.Now(), Data: data}
	// 	}
	// }

	// Output:
}

// ExamplePadding demonstrates adding space around views.
func ExamplePadding() {
	content := tui.Text("Padded content")

	// Equal padding on all sides
	_ = tui.Padding(2, content)

	// Different horizontal and vertical padding
	_ = tui.PaddingHV(4, 1, content)

	// Specific padding on each side
	_ = tui.PaddingLTRB(1, 2, 3, 4, content)

	// Output:
}

// ExampleBordered demonstrates bordered views.
func ExampleBordered() {
	content := tui.Text("Inside box")

	// Simple border
	_ = tui.Bordered(content)

	// Border with title and styling
	_ = tui.Bordered(content).
		Border(&tui.RoundedBorder).
		Title("Box Title").
		BorderFg(tui.ColorCyan)

	// Focus-aware border
	_ = tui.Bordered(content).
		FocusID("my-element").
		FocusBorderFg(tui.ColorGreen)
}

// ExampleWidth demonstrates size constraints.
func ExampleWidth() {
	content := tui.Text("This is some text content")

	// Fixed width
	_ = tui.Width(40, content)

	// Maximum width (can be smaller)
	_ = tui.MaxWidth(80, content)

	// Fixed width and height
	_ = tui.Size(40, 10, content)
}

// exampleRunApp is a minimal Application for ExampleRun.
type exampleRunApp struct{}

func (a *exampleRunApp) View() tui.View { return tui.Text("Hello") }

// ExampleRun demonstrates running a TUI application with options.
// No Output comment: this example is compiled but not executed, since
// Run takes over the terminal.
func ExampleRun() {
	app := &exampleRunApp{}

	// Run with default options
	// tui.Run(app)

	// Run with custom options
	_ = tui.Run(app,
		tui.WithFPS(60),               // 60 FPS for smooth animations
		tui.WithMouseTracking(true),   // Enable mouse events
		tui.WithAlternateScreen(true), // Use alternate screen buffer
		tui.WithBracketedPaste(true),  // Handle pasted text properly
	)
}

// inboxApp counts the keys it reads and quits on 'q'.
type inboxApp struct{ keys int }

func (a *inboxApp) View() tui.View { return tui.Text("Inbox (%d)", a.keys) }

func (a *inboxApp) HandleEvent(event tui.Event) []tui.Cmd {
	if key, ok := event.(tui.KeyEvent); ok {
		a.keys++
		if key.Rune == 'q' {
			return []tui.Cmd{tui.Quit()}
		}
	}
	return nil
}

// typedKeys is a tui.InputSource that types its runes, then ends.
type typedKeys []rune

func (k *typedKeys) ReadEvent() (tui.Event, error) {
	if len(*k) == 0 {
		return nil, io.EOF
	}
	r := (*k)[0]
	*k = (*k)[1:]
	return tui.KeyEvent{Rune: r, Time: time.Now()}, nil
}

func (k *typedKeys) SetPasteTabWidth(int) {}

// ExampleWithTerminal runs an app with tui.Run, as a user runs it, but on an
// in-memory screen. The app reads its keys from a tui.InputSource, and Run closes the
// terminal on exit, so the screen shows what the user sees afterwards.
func ExampleWithTerminal() {
	screen := termtest.NewScreen(40, 5)
	screen.Write([]byte("$ inbox\r\n")) // the shell prompt

	term := tui.NewTestTerminal(40, 5, screen)
	keys := &typedKeys{'j', 'k', 'q'}
	app := &inboxApp{}
	if err := tui.Run(app, tui.WithTerminal(term), tui.WithInput(keys)); err != nil {
		fmt.Println(err)
	}

	fmt.Println("keys read:", app.keys)
	fmt.Println("alternate screen:", screen.IsAlternateScreen())
	fmt.Printf("row 0: %q\n", screen.Row(0))
	x, y := screen.Cursor()
	fmt.Println("cursor:", x, y)
	// Output:
	// keys read: 3
	// alternate screen: false
	// row 0: "$ inbox"
	// cursor: 0 1
}

// ExampleBatch demonstrates running multiple commands.
func ExampleBatch() {
	cmd1 := func() tui.Cmd {
		return func() tui.Event {
			return tui.TickEvent{}
		}
	}

	cmd2 := func() tui.Cmd {
		return func() tui.Event {
			return tui.TickEvent{}
		}
	}

	// Run multiple commands in parallel
	cmds := tui.Batch(
		cmd1(),
		cmd2(),
	)

	fmt.Printf("Queued %d commands", len(cmds))
	// Output: Queued 2 commands
}

// ExampleFprintln prints a display-only table for CLI output. With color off,
// as when stdout is piped or NO_COLOR is set, the output is plain text.
func ExampleFprintln() {
	columns := []tui.TableColumn{{Title: "Name"}, {Title: "Role"}}
	rows := [][]string{{"Alice", "Admin"}, {"Bob", "Viewer"}}

	// A nil selection pointer highlights no row.
	table := tui.Table(columns, nil).Rows(rows)
	tui.Fprintln(os.Stdout, table, tui.WithColor(false), tui.WithWidth(20))
	// Output:
	// Name     Role
	// ─────────────────
	// Alice    Admin
	// Bob      Viewer
}
