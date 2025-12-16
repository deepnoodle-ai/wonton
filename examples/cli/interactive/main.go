// Example: Interactive Mode Dispatch
//
// Demonstrates the progressive interactivity feature:
// - Different handlers for TTY vs pipe/non-interactive
// - Interactive prompts using Wonton views
// - Confirmation dialogs
// - Selection menus
//
// Run interactively:
//
//	go run examples/cli_interactive/main.go select
//	go run examples/cli_interactive/main.go confirm
//	go run examples/cli_interactive/main.go input
//
// Run non-interactively (pipe):
//
//	echo "option1" | go run examples/cli_interactive/main.go select
package main

import (
	"fmt"
	"os"

	"github.com/deepnoodle-ai/wonton/cli"
	"github.com/deepnoodle-ai/wonton/tui"
)

func main() {
	app := cli.New("interactive").
		Description("Demonstrates interactive mode dispatch").
		Version("1.0.0")

	// Command with separate interactive and non-interactive handlers
	app.Command("select").
		Description("Select a model").
		Flags(
			&cli.StringFlag{Name: "model", Short: "m", Help: "Model to use (for non-interactive mode)"},
		).
		Interactive(func(ctx *cli.Context) error {
			// Rich interactive TUI selection
			models := []string{
				"claude-sonnet-4",
				"claude-opus-4",
				"gpt-4o",
				"gpt-4o-mini",
				"gemini-pro",
			}

			selected, err := cli.Select(ctx, "Select a model:", models...)
			if err != nil {
				return err
			}

			ctx.Printf("\nYou selected: %s\n", selected)
			return nil
		}).
		NonInteractive(func(ctx *cli.Context) error {
			// Non-interactive: require --model flag
			model := ctx.String("model")
			if model == "" {
				return cli.Error("--model flag required in non-interactive mode").
					Hint("Use --model to specify the model, or run interactively")
			}
			ctx.Printf("Using model: %s\n", model)
			return nil
		})

	// Confirmation prompt
	app.Command("confirm").
		Description("Demonstrate confirmation").
		Interactive(func(ctx *cli.Context) error {
			confirmed, err := cli.ConfirmPrompt(ctx, "Are you sure you want to proceed?")
			if err != nil {
				return err
			}

			if confirmed {
				ctx.Println("\nProceeding with action...")
			} else {
				ctx.Println("\nAction cancelled.")
			}
			return nil
		}).
		NonInteractive(func(ctx *cli.Context) error {
			return cli.Error("confirmation required").
				Hint("Run interactively to confirm, or use --yes flag")
		})

	// Text input prompt
	app.Command("input").
		Description("Demonstrate text input").
		Interactive(func(ctx *cli.Context) error {
			name, err := cli.NewInput(ctx, "Enter your name:").
				Placeholder("John Doe").
				Default("Anonymous").
				Show()
			if err != nil {
				return err
			}

			ctx.Printf("\nHello, %s!\n", name)
			return nil
		}).
		NonInteractive(func(ctx *cli.Context) error {
			return cli.Error("interactive input required").
				Hint("Run in a terminal for interactive input")
		})

	// Command that works both ways but adapts
	app.Command("info").
		Description("Show information (adapts to mode)").
		Run(func(ctx *cli.Context) error {
			if ctx.Interactive() {
				// Rich formatted output for terminal
				ctx.Println("System Information")
				ctx.Println("==================")
				ctx.Println("")
				ctx.Println("  OS:      darwin")
				ctx.Println("  Arch:    arm64")
				ctx.Println("  Version: 1.0.0")
				ctx.Println("")
				ctx.Println("Press any key to continue...")
			} else {
				// Simple output for piping/scripting
				ctx.Println("os=darwin")
				ctx.Println("arch=arm64")
				ctx.Println("version=1.0.0")
			}
			return nil
		})

	// Full interactive app using Wonton
	app.Command("dashboard").
		Description("Interactive dashboard").
		Run(func(ctx *cli.Context) error {
			if !ctx.Interactive() {
				return cli.Error("dashboard requires an interactive terminal")
			}

			dashboard := &DashboardApp{
				items: []string{"Status", "Logs", "Settings", "Quit"},
			}
			return cli.RunInteractive(ctx, dashboard, tui.WithMouseTracking(true))
		})

	if err := app.Run(); err != nil {
		if cli.IsHelpRequested(err) {
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(cli.GetExitCode(err))
	}
}

// DashboardApp is a simple interactive dashboard
type DashboardApp struct {
	items    []string
	selected int
	message  string
}

func (app *DashboardApp) View() tui.View {
	views := []tui.View{
		tui.Text("Dashboard").Bold().Fg(tui.ColorCyan),
		tui.Divider(),
		tui.Spacer().MinHeight(1),
	}

	for i, item := range app.items {
		prefix := "  "
		style := tui.NewStyle()
		if i == app.selected {
			prefix = "> "
			style = style.WithForeground(tui.ColorGreen).WithBold()
		}
		views = append(views, tui.Text("%s%s", prefix, item).Style(style))
	}

	views = append(views, tui.Spacer().MinHeight(1))

	if app.message != "" {
		views = append(views, tui.Text("%s", app.message).Fg(tui.ColorYellow))
	}

	views = append(views, tui.Text("Use arrow keys to navigate, Enter to select, q to quit").Dim())

	return tui.Stack(views...).Padding(1)
}

func (app *DashboardApp) HandleEvent(event tui.Event) []tui.Cmd {
	switch e := event.(type) {
	case tui.KeyEvent:
		switch e.Key {
		case tui.KeyArrowUp:
			if app.selected > 0 {
				app.selected--
			}
		case tui.KeyArrowDown:
			if app.selected < len(app.items)-1 {
				app.selected++
			}
		case tui.KeyEnter:
			selected := app.items[app.selected]
			if selected == "Quit" {
				return []tui.Cmd{tui.Quit()}
			}
			app.message = fmt.Sprintf("Selected: %s", selected)
		case tui.KeyCtrlC:
			return []tui.Cmd{tui.Quit()}
		}
		if e.Rune == 'q' {
			return []tui.Cmd{tui.Quit()}
		}
	}
	return nil
}
