// Example: Multi-step Wizard with TUI
//
// Demonstrates building a configuration wizard with the CLI framework:
// - Multi-step form flow with validation
// - Input fields, selections, and confirmations
// - Progress indication through steps
// - Summary and confirmation before completion
//
// Run with:
//
//	go run examples/cli_wizard/main.go --help
//	go run examples/cli_wizard/main.go init           # Run the setup wizard
//	go run examples/cli_wizard/main.go init --quick   # Quick setup with defaults
//	go run examples/cli_wizard/main.go config show    # Show current config
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/deepnoodle-ai/gooey/cli"
	"github.com/deepnoodle-ai/gooey/tui"
)

// Config represents the application configuration
type Config struct {
	ProjectName string
	Language    string
	Framework   string
	Database    string
	Features    []string
	Port        int
	EnableTLS   bool
}

func main() {
	app := cli.New("wizard", "Configuration wizard demo")
	app.Version("1.0.0")

	// Init command - runs the setup wizard
	app.Command("init", "Initialize a new project with a wizard").
		AddFlag(&cli.Flag{
			Name:        "quick",
			Short:       "q",
			Description: "Use defaults without prompts",
			Default:     false,
		}).
		AddFlag(&cli.Flag{
			Name:        "name",
			Short:       "n",
			Description: "Project name (for quick mode)",
			Default:     "",
		}).
		Interactive(func(ctx *cli.Context) error {
			// Full interactive wizard
			wizard := &WizardApp{
				step: 0,
				config: Config{
					Port: 8080,
				},
			}
			return cli.RunInteractive(ctx, wizard, tui.WithMouseTracking(true))
		}).
		NonInteractive(func(ctx *cli.Context) error {
			// Non-interactive: use quick mode or fail
			if !ctx.Bool("quick") {
				return cli.Error("wizard requires an interactive terminal").
					Hint("Use --quick for non-interactive setup with defaults")
			}

			name := ctx.String("name")
			if name == "" {
				name = "my-project"
			}

			config := Config{
				ProjectName: name,
				Language:    "go",
				Framework:   "gooey",
				Database:    "sqlite",
				Features:    []string{"api", "tui"},
				Port:        8080,
				EnableTLS:   false,
			}

			ctx.Println("Quick setup complete!")
			ctx.Printf("Project:   %s\n", config.ProjectName)
			ctx.Printf("Language:  %s\n", config.Language)
			ctx.Printf("Framework: %s\n", config.Framework)
			ctx.Printf("Database:  %s\n", config.Database)
			return nil
		})

	// Step-by-step command using prompts
	app.Command("setup", "Step-by-step setup using prompts").
		Run(func(ctx *cli.Context) error {
			if !ctx.Interactive() {
				return cli.Error("setup requires an interactive terminal")
			}

			ctx.Println("Project Setup Wizard")
			ctx.Println("====================")
			ctx.Println("")

			// Step 1: Project name
			name, err := cli.NewInput(ctx, "Project name:").
				Placeholder("my-awesome-project").
				Default("my-project").
				Show()
			if err != nil {
				return err
			}

			// Step 2: Language
			language, err := cli.Select(ctx, "Programming language:", "go", "python", "rust", "typescript")
			if err != nil {
				return err
			}

			// Step 3: Framework based on language
			var frameworks []string
			switch language {
			case "go":
				frameworks = []string{"gooey", "gin", "echo", "fiber"}
			case "python":
				frameworks = []string{"fastapi", "flask", "django"}
			case "rust":
				frameworks = []string{"actix", "rocket", "axum"}
			case "typescript":
				frameworks = []string{"express", "fastify", "nest"}
			}

			framework, err := cli.Select(ctx, "Framework:", frameworks...)
			if err != nil {
				return err
			}

			// Step 4: Database
			database, err := cli.Select(ctx, "Database:", "sqlite", "postgres", "mysql", "mongodb")
			if err != nil {
				return err
			}

			// Step 5: Confirm
			ctx.Println("")
			ctx.Println("Configuration Summary:")
			ctx.Println("----------------------")
			ctx.Printf("  Project:   %s\n", name)
			ctx.Printf("  Language:  %s\n", language)
			ctx.Printf("  Framework: %s\n", framework)
			ctx.Printf("  Database:  %s\n", database)
			ctx.Println("")

			confirmed, err := cli.ConfirmPrompt(ctx, "Create project with these settings?")
			if err != nil {
				return err
			}

			if confirmed {
				ctx.Println("\nCreating project...")
				ctx.Printf("Project '%s' created successfully!\n", name)
			} else {
				ctx.Println("\nSetup cancelled.")
			}

			return nil
		})

	// Config group
	config := app.Group("config", "Configuration management")

	config.Command("show", "Show current configuration").
		Run(func(ctx *cli.Context) error {
			// Simulated config
			ctx.Println("Current Configuration:")
			ctx.Println("----------------------")
			ctx.Println("  project_name: my-project")
			ctx.Println("  language: go")
			ctx.Println("  framework: gooey")
			ctx.Println("  database: sqlite")
			ctx.Println("  port: 8080")
			ctx.Println("  tls_enabled: false")
			return nil
		})

	config.Command("edit", "Edit configuration interactively").
		Run(func(ctx *cli.Context) error {
			if !ctx.Interactive() {
				return cli.Error("edit requires an interactive terminal")
			}

			editor := &ConfigEditor{
				items: []configItem{
					{key: "project_name", value: "my-project", editable: true},
					{key: "language", value: "go", options: []string{"go", "python", "rust", "typescript"}},
					{key: "framework", value: "gooey", options: []string{"gooey", "gin", "echo"}},
					{key: "database", value: "sqlite", options: []string{"sqlite", "postgres", "mysql"}},
					{key: "port", value: "8080", editable: true},
					{key: "tls_enabled", value: "false", options: []string{"true", "false"}},
				},
			}
			return cli.RunInteractive(ctx, editor, tui.WithMouseTracking(true))
		})

	if err := app.Run(); err != nil {
		if cli.IsHelpRequested(err) {
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(cli.GetExitCode(err))
	}
}

// WizardApp is the multi-step wizard TUI
type WizardApp struct {
	step    int
	config  Config
	message string

	// Form state
	projectName string
	portInput   string
}

const (
	stepWelcome = iota
	stepName
	stepLanguage
	stepFramework
	stepDatabase
	stepFeatures
	stepPort
	stepTLS
	stepSummary
	stepComplete
)

var stepTitles = []string{
	"Welcome",
	"Project Name",
	"Language",
	"Framework",
	"Database",
	"Features",
	"Port",
	"TLS",
	"Summary",
	"Complete",
}

func (app *WizardApp) View() tui.View {
	views := []tui.View{
		// Header
		tui.HeaderBar("Project Setup Wizard"),
		tui.Spacer().MinHeight(1),

		// Progress indicator
		app.progressBar(),
		tui.Spacer().MinHeight(1),
	}

	// Step content
	switch app.step {
	case stepWelcome:
		views = append(views, app.welcomeView()...)
	case stepName:
		views = append(views, app.nameView()...)
	case stepLanguage:
		views = append(views, app.languageView()...)
	case stepFramework:
		views = append(views, app.frameworkView()...)
	case stepDatabase:
		views = append(views, app.databaseView()...)
	case stepFeatures:
		views = append(views, app.featuresView()...)
	case stepPort:
		views = append(views, app.portView()...)
	case stepTLS:
		views = append(views, app.tlsView()...)
	case stepSummary:
		views = append(views, app.summaryView()...)
	case stepComplete:
		views = append(views, app.completeView()...)
	}

	views = append(views, tui.Spacer())

	// Message area
	if app.message != "" {
		views = append(views, tui.Text("%s", app.message).Fg(tui.ColorYellow))
	}

	// Footer
	views = append(views, tui.Divider())
	if app.step == stepComplete {
		views = append(views, tui.Text("Press Enter or q to exit").Dim())
	} else {
		views = append(views, tui.Text("Enter: Continue  Backspace: Go Back  q: Quit").Dim())
	}

	return tui.VStack(views...).Padding(1)
}

func (app *WizardApp) progressBar() tui.View {
	total := len(stepTitles) - 1 // Exclude complete
	current := app.step
	if current >= total {
		current = total
	}

	return tui.VStack(
		tui.Progress(current, total).Width(50).Fg(tui.ColorCyan),
		tui.Text("Step %d of %d: %s", current+1, total, stepTitles[app.step]).Dim(),
	)
}

func (app *WizardApp) welcomeView() []tui.View {
	return []tui.View{
		tui.Text("Welcome to the Project Setup Wizard!").Bold().Fg(tui.ColorCyan),
		tui.Spacer().MinHeight(1),
		tui.Text("This wizard will help you configure your new project."),
		tui.Text("You'll be asked to provide:"),
		tui.Spacer().MinHeight(1),
		tui.Text("  - Project name"),
		tui.Text("  - Programming language"),
		tui.Text("  - Framework selection"),
		tui.Text("  - Database choice"),
		tui.Text("  - Optional features"),
		tui.Text("  - Server configuration"),
		tui.Spacer().MinHeight(1),
		tui.Text("Press Enter to begin...").Fg(tui.ColorGreen),
	}
}

func (app *WizardApp) nameView() []tui.View {
	return []tui.View{
		tui.Text("Project Name").Bold().Fg(tui.ColorCyan),
		tui.Spacer().MinHeight(1),
		tui.Text("Enter a name for your project:"),
		tui.Spacer().MinHeight(1),
		tui.Input(&app.projectName).
			Placeholder("my-awesome-project").
			Width(40).
			OnSubmit(func(s string) {
				if s != "" {
					app.config.ProjectName = s
					app.step++
					app.message = ""
				} else {
					app.message = "Project name cannot be empty"
				}
			}),
	}
}

func (app *WizardApp) languageView() []tui.View {
	languages := []string{"go", "python", "rust", "typescript"}
	views := []tui.View{
		tui.Text("Programming Language").Bold().Fg(tui.ColorCyan),
		tui.Spacer().MinHeight(1),
		tui.Text("Select a programming language:"),
		tui.Spacer().MinHeight(1),
	}

	for i, lang := range languages {
		selected := app.config.Language == lang
		views = append(views, app.radioButton(lang, selected, func() {
			app.config.Language = languages[i]
		}))
	}

	return views
}

func (app *WizardApp) frameworkView() []tui.View {
	var frameworks []string
	switch app.config.Language {
	case "go":
		frameworks = []string{"gooey", "gin", "echo", "fiber"}
	case "python":
		frameworks = []string{"fastapi", "flask", "django"}
	case "rust":
		frameworks = []string{"actix", "rocket", "axum"}
	case "typescript":
		frameworks = []string{"express", "fastify", "nest"}
	default:
		frameworks = []string{"default"}
	}

	views := []tui.View{
		tui.Text("Framework").Bold().Fg(tui.ColorCyan),
		tui.Spacer().MinHeight(1),
		tui.Text("Select a framework for %s:", app.config.Language),
		tui.Spacer().MinHeight(1),
	}

	for i, fw := range frameworks {
		selected := app.config.Framework == fw
		views = append(views, app.radioButton(fw, selected, func() {
			app.config.Framework = frameworks[i]
		}))
	}

	return views
}

func (app *WizardApp) databaseView() []tui.View {
	databases := []string{"sqlite", "postgres", "mysql", "mongodb"}
	views := []tui.View{
		tui.Text("Database").Bold().Fg(tui.ColorCyan),
		tui.Spacer().MinHeight(1),
		tui.Text("Select a database:"),
		tui.Spacer().MinHeight(1),
	}

	for i, db := range databases {
		selected := app.config.Database == db
		views = append(views, app.radioButton(db, selected, func() {
			app.config.Database = databases[i]
		}))
	}

	return views
}

func (app *WizardApp) featuresView() []tui.View {
	features := []string{"api", "tui", "auth", "logging", "metrics"}
	views := []tui.View{
		tui.Text("Features").Bold().Fg(tui.ColorCyan),
		tui.Spacer().MinHeight(1),
		tui.Text("Select features to enable (click to toggle):"),
		tui.Spacer().MinHeight(1),
	}

	for i, feat := range features {
		enabled := contains(app.config.Features, feat)
		views = append(views, app.checkboxButton(feat, enabled, func() {
			if contains(app.config.Features, features[i]) {
				app.config.Features = remove(app.config.Features, features[i])
			} else {
				app.config.Features = append(app.config.Features, features[i])
			}
		}))
	}

	return views
}

func (app *WizardApp) portView() []tui.View {
	if app.portInput == "" {
		app.portInput = fmt.Sprintf("%d", app.config.Port)
	}
	return []tui.View{
		tui.Text("Server Port").Bold().Fg(tui.ColorCyan),
		tui.Spacer().MinHeight(1),
		tui.Text("Enter the server port:"),
		tui.Spacer().MinHeight(1),
		tui.Input(&app.portInput).
			Placeholder("8080").
			Width(10).
			OnSubmit(func(s string) {
				var port int
				_, err := fmt.Sscanf(s, "%d", &port)
				if err != nil || port < 1 || port > 65535 {
					app.message = "Please enter a valid port (1-65535)"
				} else {
					app.config.Port = port
					app.step++
					app.message = ""
				}
			}),
	}
}

func (app *WizardApp) tlsView() []tui.View {
	return []tui.View{
		tui.Text("TLS Configuration").Bold().Fg(tui.ColorCyan),
		tui.Spacer().MinHeight(1),
		tui.Text("Enable TLS/HTTPS?"),
		tui.Spacer().MinHeight(1),
		app.radioButton("Yes - Enable TLS", app.config.EnableTLS, func() {
			app.config.EnableTLS = true
		}),
		app.radioButton("No - HTTP only", !app.config.EnableTLS, func() {
			app.config.EnableTLS = false
		}),
	}
}

func (app *WizardApp) summaryView() []tui.View {
	features := "none"
	if len(app.config.Features) > 0 {
		features = strings.Join(app.config.Features, ", ")
	}

	tls := "Disabled"
	if app.config.EnableTLS {
		tls = "Enabled"
	}

	return []tui.View{
		tui.Text("Summary").Bold().Fg(tui.ColorCyan),
		tui.Spacer().MinHeight(1),
		tui.Text("Please review your configuration:"),
		tui.Spacer().MinHeight(1),
		tui.Bordered(tui.VStack(
			tui.KeyValue("Project Name", app.config.ProjectName),
			tui.KeyValue("Language", app.config.Language),
			tui.KeyValue("Framework", app.config.Framework),
			tui.KeyValue("Database", app.config.Database),
			tui.KeyValue("Features", features),
			tui.KeyValue("Port", fmt.Sprintf("%d", app.config.Port)),
			tui.KeyValue("TLS", tls),
		).Padding(1)),
		tui.Spacer().MinHeight(1),
		tui.Text("Press Enter to create the project...").Fg(tui.ColorGreen),
	}
}

func (app *WizardApp) completeView() []tui.View {
	return []tui.View{
		tui.Text("Setup Complete!").Bold().Fg(tui.ColorGreen),
		tui.Spacer().MinHeight(1),
		tui.Text("Your project has been configured successfully."),
		tui.Spacer().MinHeight(1),
		tui.Text("Project: %s", app.config.ProjectName).Fg(tui.ColorCyan),
		tui.Spacer().MinHeight(1),
		tui.Text("Next steps:"),
		tui.Text("  1. cd %s", app.config.ProjectName),
		tui.Text("  2. %s install", getPackageManager(app.config.Language)),
		tui.Text("  3. %s run .", getRunCommand(app.config.Language)),
	}
}

func (app *WizardApp) radioButton(label string, selected bool, onClick func()) tui.View {
	prefix := "( ) "
	style := tui.NewStyle()
	if selected {
		prefix = "(*) "
		style = style.WithForeground(tui.ColorCyan)
	}
	return tui.Clickable(fmt.Sprintf("%s%s", prefix, label), onClick).Style(style)
}

func (app *WizardApp) checkboxButton(label string, checked bool, onClick func()) tui.View {
	prefix := "[ ] "
	style := tui.NewStyle()
	if checked {
		prefix = "[x] "
		style = style.WithForeground(tui.ColorGreen)
	}
	return tui.Clickable(fmt.Sprintf("%s%s", prefix, label), onClick).Style(style)
}

func (app *WizardApp) HandleEvent(event tui.Event) []tui.Cmd {
	switch e := event.(type) {
	case tui.KeyEvent:
		switch e.Key {
		case tui.KeyCtrlC:
			return []tui.Cmd{tui.Quit()}
		case tui.KeyEnter:
			app.nextStep()
		case tui.KeyBackspace:
			if app.step > 0 && app.step < stepComplete {
				app.step--
				app.message = ""
			}
		case tui.KeyEscape:
			return []tui.Cmd{tui.Quit()}
		}
		if e.Rune == 'q' {
			return []tui.Cmd{tui.Quit()}
		}
	}
	return nil
}

func (app *WizardApp) nextStep() {
	switch app.step {
	case stepWelcome:
		app.step++
	case stepName:
		if app.projectName == "" {
			app.message = "Please enter a project name"
		} else {
			app.config.ProjectName = app.projectName
			app.step++
			app.message = ""
		}
	case stepLanguage:
		if app.config.Language == "" {
			app.config.Language = "go" // Default
		}
		app.step++
	case stepFramework:
		if app.config.Framework == "" {
			app.config.Framework = "gooey" // Default
		}
		app.step++
	case stepDatabase:
		if app.config.Database == "" {
			app.config.Database = "sqlite" // Default
		}
		app.step++
	case stepFeatures:
		app.step++
	case stepPort:
		// Handled by input OnSubmit
		if app.portInput != "" {
			var port int
			fmt.Sscanf(app.portInput, "%d", &port)
			if port >= 1 && port <= 65535 {
				app.config.Port = port
				app.step++
			}
		}
	case stepTLS:
		app.step++
	case stepSummary:
		app.step++
	case stepComplete:
		// Exit
	}
}

// ConfigEditor is a TUI for editing configuration
type ConfigEditor struct {
	items   []configItem
	cursor  int
	editing bool
	input   string
}

type configItem struct {
	key      string
	value    string
	options  []string // If set, this is a select field
	editable bool     // If true and no options, this is a text field
}

func (app *ConfigEditor) View() tui.View {
	views := []tui.View{
		tui.Text("Configuration Editor").Bold().Fg(tui.ColorCyan),
		tui.Divider(),
		tui.Spacer().MinHeight(1),
	}

	for i, item := range app.items {
		selected := i == app.cursor
		views = append(views, app.configRow(item, selected))
	}

	views = append(views, tui.Spacer())
	views = append(views, tui.Divider())
	views = append(views, tui.Text("j/k: Navigate  Enter: Edit  s: Save  q: Quit").Dim())

	return tui.VStack(views...).Padding(1)
}

func (app *ConfigEditor) configRow(item configItem, selected bool) tui.View {
	prefix := "  "
	style := tui.NewStyle()
	if selected {
		prefix = "> "
		style = style.WithForeground(tui.ColorCyan)
	}

	return tui.HStack(
		tui.Text("%s", prefix).Style(style),
		tui.Text("%-15s", item.key).Bold(),
		tui.Text(": "),
		tui.Text("%s", item.value).Fg(tui.ColorGreen),
	)
}

func (app *ConfigEditor) HandleEvent(event tui.Event) []tui.Cmd {
	switch e := event.(type) {
	case tui.KeyEvent:
		switch e.Key {
		case tui.KeyCtrlC:
			return []tui.Cmd{tui.Quit()}
		case tui.KeyArrowUp:
			if app.cursor > 0 {
				app.cursor--
			}
		case tui.KeyArrowDown:
			if app.cursor < len(app.items)-1 {
				app.cursor++
			}
		}
		switch e.Rune {
		case 'q':
			return []tui.Cmd{tui.Quit()}
		case 'j':
			if app.cursor < len(app.items)-1 {
				app.cursor++
			}
		case 'k':
			if app.cursor > 0 {
				app.cursor--
			}
		case 's':
			// Save would go here
		}
	}
	return nil
}

// Helper functions

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func remove(slice []string, item string) []string {
	result := make([]string, 0, len(slice))
	for _, s := range slice {
		if s != item {
			result = append(result, s)
		}
	}
	return result
}

func getPackageManager(lang string) string {
	switch lang {
	case "go":
		return "go mod"
	case "python":
		return "pip"
	case "rust":
		return "cargo"
	case "typescript":
		return "npm"
	default:
		return "package-manager"
	}
}

func getRunCommand(lang string) string {
	switch lang {
	case "go":
		return "go"
	case "python":
		return "python"
	case "rust":
		return "cargo"
	case "typescript":
		return "npx ts-node"
	default:
		return "run"
	}
}
