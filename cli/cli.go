// Package cli provides a flexible CLI framework for building command-line applications
// with rich terminal UI capabilities.
//
// The cli package enables rapid development of CLI tools with:
//
//   - Fluent API for defining commands, subcommands, and command groups
//   - Type-safe flag parsing with support for environment variables
//   - Progressive interactivity: commands adapt between quick one-liners and rich TUIs
//   - Middleware support for cross-cutting concerns (auth, logging, etc.)
//   - Styled help output with customizable themes
//   - Shell completion generation (bash, zsh, fish)
//   - Testing utilities for CLI command verification
//
// # Basic Usage
//
// Create a simple CLI application:
//
//	app := cli.New("myapp").
//	    Description("A sample CLI application").
//	    Version("1.0.0")
//
//	app.Command("greet").
//	    Description("Greet a user").
//	    Args("name").
//	    Run(func(ctx *cli.Context) error {
//	        name := ctx.Arg(0)
//	        ctx.Printf("Hello, %s!\n", name)
//	        return nil
//	    })
//
//	return app.Execute()
//
// # Commands and Groups
//
// Commands can be organized into groups for better structure:
//
//	users := app.Group("users").
//	    Description("Manage users")
//
//	users.Command("list").
//	    Description("List all users").
//	    Run(func(ctx *cli.Context) error {
//	        // List users
//	        return nil
//	    })
//
//	users.Command("create").
//	    Description("Create a new user").
//	    Args("username", "email").
//	    Run(func(ctx *cli.Context) error {
//	        // Create user
//	        return nil
//	    })
//
// # Flags
//
// The package supports type-safe flags with validation and defaults:
//
//	app.Command("deploy").
//	    Flags(
//	        cli.String("env", "e").Default("staging").Help("Deployment environment"),
//	        cli.Bool("force", "f").Help("Force deployment"),
//	        cli.Int("replicas", "r").Default(3).Help("Number of replicas"),
//	    ).
//	    Run(func(ctx *cli.Context) error {
//	        env := ctx.String("env")
//	        force := ctx.Bool("force")
//	        replicas := ctx.Int("replicas")
//	        // Deploy with these settings
//	        return nil
//	    })
//
// Flags can also be bound from environment variables:
//
//	cli.String("token", "t").Env("API_TOKEN").Required()
//
// # Progressive Interactivity
//
// Commands can provide both interactive and non-interactive modes:
//
//	app.Command("delete").
//	    Interactive(func(ctx *cli.Context) error {
//	        // Show rich TUI with confirmation
//	        confirmed, err := ctx.Confirm("Delete all data?")
//	        if err != nil || !confirmed {
//	            return err
//	        }
//	        return performDelete()
//	    }).
//	    NonInteractive(func(ctx *cli.Context) error {
//	        // Require --force flag when piped
//	        if !ctx.Bool("force") {
//	            return cli.Error("--force required in non-interactive mode")
//	        }
//	        return performDelete()
//	    })
//
// # Middleware
//
// Middleware wraps handlers to add reusable behavior:
//
//	// Global middleware applies to all commands
//	app.Use(cli.Recover())
//
//	// Command-specific middleware
//	app.Command("admin").
//	    Use(requireAuth).
//	    Run(func(ctx *cli.Context) error {
//	        // Handle admin command
//	        return nil
//	    })
//
// # Error Handling
//
// The package provides rich error types with hints and details:
//
//	return cli.Errorf("deployment failed: %s", err).
//	    Hint("Check your credentials and try again").
//	    Detail("Server: %s", server).
//	    Detail("Exit code: %d", exitCode)
//
// # Testing
//
// Commands are easy to test with the built-in testing utilities:
//
//	func TestGreetCommand(t *testing.T) {
//	    app := setupApp()
//	    result := app.Test(t, cli.TestArgs("greet", "Alice"))
//
//	    if !result.Success() {
//	        t.Errorf("command failed: %v", result.Err)
//	    }
//	    if !result.Contains("Hello, Alice") {
//	        t.Errorf("unexpected output: %s", result.Stdout)
//	    }
//	}
package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/deepnoodle-ai/wonton/color"
	"github.com/deepnoodle-ai/wonton/tui"
)

// App represents a CLI application. It manages commands, groups, global flags,
// and application-wide configuration.
//
// Use New to create an App, then configure it with the fluent builder methods:
//
//	app := cli.New("myapp").
//	    Description("My awesome CLI").
//	    Version("1.0.0")
//
// Register commands with Command or organize them with Group:
//
//	app.Command("serve").Description("Start server").Run(handler)
//	app.Group("users").Command("list").Run(listUsersHandler)
//
// The App automatically provides built-in help and version commands.
type App struct {
	name        string
	description string
	version     string

	commands   map[string]*Command
	groups     map[string]*Group
	middleware []Middleware

	// Global flags
	globalFlags []Flag

	// Root handler (runs when no command specified)
	handler    Handler
	args       []*Arg
	validators []func(*Context) error

	// I/O
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer

	// Runtime state
	isInteractive    bool
	forceInteractive *bool // For testing - if set, overrides TTY detection
	colorEnabled     bool  // Whether to use colored output

	// Help styling
	helpTheme *HelpTheme
}

// New creates a new CLI application with the given name.
//
// The name is used in help output and error messages. Configure the app
// using the fluent builder methods:
//
//	app := cli.New("myapp").
//	    Description("A sample application").
//	    Version("1.0.0")
//
// By default, the app uses os.Stdin, os.Stdout, and os.Stderr for I/O,
// and detects interactivity automatically based on TTY presence.
func New(name string) *App {
	return &App{
		name:         name,
		commands:     make(map[string]*Command),
		groups:       make(map[string]*Group),
		stdin:        os.Stdin,
		stdout:       os.Stdout,
		stderr:       os.Stderr,
		colorEnabled: color.ShouldColorize(os.Stdout),
	}
}

// Description sets the application description.
func (a *App) Description(desc string) *App {
	a.description = desc
	return a
}

// Version sets the application version.
func (a *App) Version(v string) *App {
	a.version = v
	return a
}

// Command registers a new command or returns an existing one.
// Use builder methods like Description(), Args(), and Flags() to configure the command.
func (a *App) Command(name string) *Command {
	if existing, ok := a.commands[name]; ok {
		return existing
	}
	cmd := newCommand(name, a)
	a.commands[name] = cmd
	return cmd
}

// Main returns a command builder for the main/root command.
// This command runs when no subcommand is specified.
//
// Example:
//
//	app.Main().
//	    Args("url").
//	    Flags(cli.String("output", "o").Help("Output file")).
//	    Run(handler)
//
// This is equivalent to app.Command("") but more explicit.
func (a *App) Main() *Command {
	return a.Command("")
}

// Group creates a new command group for organizing related commands.
//
// Groups help organize commands hierarchically. For example:
//
//	users := app.Group("users").Description("User management")
//	users.Command("list").Run(listHandler)
//	users.Command("create").Args("username").Run(createHandler)
//
// Users can invoke grouped commands as "myapp users list" or "myapp users:list".
func (a *App) Group(name string) *Group {
	g := &Group{
		name:     name,
		app:      a,
		commands: make(map[string]*Command),
	}
	a.groups[name] = g
	return g
}

// Use adds middleware that will be applied to all commands in the application.
//
// Middleware wraps command handlers to add cross-cutting behavior like
// logging, authentication, or error recovery:
//
//	app.Use(cli.Recover())  // Recover from panics
//	app.Use(loggingMiddleware)
//
// Middleware is applied in the order registered, with app-level middleware
// executing before command-level middleware.
func (a *App) Use(mw ...Middleware) *App {
	a.middleware = append(a.middleware, mw...)
	return a
}

// AddGlobalFlag adds a global flag available to all commands.
func (a *App) AddGlobalFlag(f Flag) *App {
	a.globalFlags = append(a.globalFlags, f)
	return a
}

// GlobalFlags adds multiple global flags available to all commands.
func (a *App) GlobalFlags(flags ...Flag) *App {
	a.globalFlags = append(a.globalFlags, flags...)
	return a
}

// Run sets the root handler that executes when no command is specified.
func (a *App) Run(h Handler) *App {
	a.handler = h
	return a
}

// Args sets the positional argument names for the root command.
// Append "?" to make an argument optional (e.g., "name?").
func (a *App) Args(names ...string) *App {
	for _, name := range names {
		required := true
		if strings.HasSuffix(name, "?") {
			name = strings.TrimSuffix(name, "?")
			required = false
		}
		a.args = append(a.args, &Arg{
			Name:     name,
			Required: required,
		})
	}
	return a
}

// Validate adds a validation function for the root command.
func (a *App) Validate(v func(*Context) error) *App {
	a.validators = append(a.validators, v)
	return a
}

// rootCommand returns a Command that wraps the app's root handler for execution.
func (a *App) rootCommand() *Command {
	return &Command{
		name:        a.name,
		description: a.description,
		app:         a,
		handler:     a.handler,
		flags:       nil, // global flags are automatically included
		args:        a.args,
		validators:  a.validators,
	}
}

// Execute runs the CLI application with os.Args.
//
// This is the typical entry point for CLI applications:
//
//	func main() {
//	    app := setupApp()
//	    if err := app.Execute(); err != nil {
//	        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
//	        os.Exit(1)
//	    }
//	}
//
// Execute automatically strips the program name from os.Args and passes
// the remaining arguments to ExecuteArgs.
func (a *App) Execute() error {
	return a.ExecuteContext(context.Background(), os.Args[1:])
}

// ExecuteArgs runs the CLI application with the given arguments.
func (a *App) ExecuteArgs(args []string) error {
	return a.ExecuteContext(context.Background(), args)
}

// ForceInteractive sets the interactive mode for testing purposes.
// Pass true to force interactive, false to force non-interactive.
func (a *App) ForceInteractive(interactive bool) *App {
	a.forceInteractive = &interactive
	return a
}

// ExecuteContext runs the CLI application with context and arguments.
func (a *App) ExecuteContext(ctx context.Context, args []string) error {
	// Detect interactivity (can be overridden for testing)
	if a.forceInteractive != nil {
		a.isInteractive = *a.forceInteractive
	} else {
		a.isInteractive = isTerminal(os.Stdin) && isTerminal(os.Stdout)
	}

	// Parse command and arguments using definition-driven parser
	p := newParser(a)
	result, err := p.parse(args)
	if err != nil {
		return err
	}

	// Check for help/version in global flags
	for _, gf := range result.GlobalFlags {
		if gf == "--help" || gf == "-h" {
			return a.showHelp()
		}
		if gf == "--version" {
			if a.version != "" {
				fmt.Fprintln(a.stdout, a.version)
			}
			return nil
		}
	}

	// Handle built-in commands
	switch result.Command {
	case "help":
		return a.showHelp()
	case "version":
		if a.version != "" {
			fmt.Fprintln(a.stdout, a.version)
		}
		return nil
	}

	// Build the full argument list for the command.
	// This includes any global flags (which the command's parseFlags will process)
	// followed by the command's own arguments.
	cmdArgs := append(result.GlobalFlags, result.CommandArgs...)

	// Find the command (or use root handler)
	var cmd *Command
	if result.Command == "" && result.Group == "" {
		if a.handler == nil && a.commands[""] == nil {
			return a.showHelp()
		}
		// Use root handler (prefer explicit Command("") over app handler)
		if rootCmd := a.commands[""]; rootCmd != nil {
			cmd = rootCmd
		} else {
			cmd = a.rootCommand()
		}
	} else if result.Group != "" {
		// Group command
		group := a.groups[result.Group]
		if group == nil {
			return fmt.Errorf("unknown group: %s", result.Group)
		}
		if result.Command == "" {
			// Check if there are remaining args that might be an unknown subcommand
			if len(result.CommandArgs) > 0 {
				firstArg := result.CommandArgs[0]
				// Check if it's a help flag
				if firstArg == "--help" || firstArg == "-h" {
					return group.showHelp()
				}
				// Not a flag - could be an unknown subcommand
				if !looksLikeFlag(firstArg) {
					if group.handler == nil {
						return fmt.Errorf("unknown subcommand '%s' for group '%s'", firstArg, result.Group)
					}
					// Group has a handler, treat as positional arg
				} else if group.handler == nil {
					// First arg is a flag but group has no handler - requires a subcommand
					return fmt.Errorf("group '%s' requires a subcommand", result.Group)
				}
			} else if group.handler == nil {
				// No args and no handler - requires a subcommand
				return fmt.Errorf("group '%s' requires a subcommand", result.Group)
			}
			// Group with handler
			cmd = &Command{
				name:       result.Group,
				app:        a,
				handler:    group.handler,
				flags:      group.flags,
				args:       group.args,
				middleware: group.middleware,
				validators: group.validators,
			}
		} else {
			// Group with subcommand
			cmd = group.commands[result.Command]
			if cmd == nil {
				return fmt.Errorf("unknown command: %s %s", result.Group, result.Command)
			}
		}
	} else {
		// Direct command
		cmd = a.commands[result.Command]
		if cmd == nil {
			// Check aliases
			for _, c := range a.commands {
				for _, alias := range c.aliases {
					if alias == result.Command {
						cmd = c
						break
					}
				}
				if cmd != nil {
					break
				}
			}
		}
		if cmd == nil {
			// Command not found - if app has root handler, treat as positional arg
			if rootCmd := a.commands[""]; rootCmd != nil {
				cmd = rootCmd
				cmdArgs = append([]string{result.Command}, cmdArgs...)
			} else if a.handler != nil {
				cmd = a.rootCommand()
				cmdArgs = append([]string{result.Command}, cmdArgs...)
			} else {
				return fmt.Errorf("unknown command: %s", result.Command)
			}
		}
	}

	// Create execution context
	execCtx := &Context{
		context:     ctx,
		app:         a,
		command:     cmd,
		args:        cmdArgs,
		flags:       make(map[string]any),
		interactive: a.isInteractive,
		stdin:       a.stdin,
		stdout:      a.stdout,
		stderr:      a.stderr,
	}

	// Parse flags for this command
	if err := cmd.parseFlags(execCtx, cmdArgs); err != nil {
		return err
	}

	// Select handler based on interactivity
	handler := cmd.handler
	if a.isInteractive && cmd.interactive != nil {
		handler = cmd.interactive
	} else if !a.isInteractive && cmd.nonInteract != nil {
		handler = cmd.nonInteract
	}

	// Ensure we have a handler
	if handler == nil {
		return fmt.Errorf("no handler defined for command: %s", cmd.name)
	}

	// Build middleware chain: app middleware runs before (wraps) command middleware
	for i := len(cmd.middleware) - 1; i >= 0; i-- {
		handler = cmd.middleware[i](handler)
	}
	for i := len(a.middleware) - 1; i >= 0; i-- {
		handler = a.middleware[i](handler)
	}

	// Execute
	return handler(execCtx)
}

// findGlobalFlag looks up a global flag by name.
func (a *App) findGlobalFlag(name string) Flag {
	for _, f := range a.globalFlags {
		if f.GetName() == name {
			return f
		}
	}
	return nil
}

// findGlobalFlagByShort looks up a global flag by short name.
func (a *App) findGlobalFlagByShort(short string) Flag {
	for _, f := range a.globalFlags {
		if f.GetShort() == short {
			return f
		}
	}
	return nil
}

// findCommand looks up a command by name, including group commands and aliases.
// It returns the command, the remaining args (after consuming subcommand name if applicable), and any error.
func (a *App) findCommand(name string, args []string) (*Command, []string, error) {
	// Check direct commands first
	if cmd, ok := a.commands[name]; ok {
		return cmd, args, nil
	}

	// Check aliases for direct commands
	for _, cmd := range a.commands {
		for _, alias := range cmd.aliases {
			if alias == name {
				return cmd, args, nil
			}
		}
	}

	// Check for group:command pattern
	parts := strings.SplitN(name, ":", 2)
	if len(parts) == 2 {
		if g, ok := a.groups[parts[0]]; ok {
			if cmd, ok := g.commands[parts[1]]; ok {
				return cmd, args, nil
			}
			// Check aliases within the group
			for _, cmd := range g.commands {
				for _, alias := range cmd.aliases {
					if alias == parts[1] {
						return cmd, args, nil
					}
				}
			}
		}
	}

	// Check groups with space-separated subcommand (e.g., "users list" as args ["users", "list"])
	if group, ok := a.groups[name]; ok {
		if len(args) > 0 {
			subName := args[0]
			// Handle help flags for the group
			if subName == "--help" || subName == "-h" {
				return nil, nil, group.showHelp()
			}
			// Check direct subcommand
			if cmd, ok := group.commands[subName]; ok {
				return cmd, args[1:], nil
			}
			// Check aliases within the group
			for _, cmd := range group.commands {
				for _, alias := range cmd.aliases {
					if alias == subName {
						return cmd, args[1:], nil
					}
				}
			}
			// Not a subcommand - if group has handler, treat as positional args
			if group.handler != nil {
				return group.asCommand(), args, nil
			}
			// No handler - unknown subcommand
			return nil, nil, fmt.Errorf("unknown subcommand '%s' for group '%s'\n\nAvailable commands:\n%s",
				subName, name, group.commandList())
		}
		// No args provided
		if group.handler != nil {
			return group.asCommand(), args, nil
		}
		// No handler - requires a subcommand
		return nil, nil, fmt.Errorf("group '%s' requires a subcommand\n\nAvailable commands:\n%s",
			name, group.commandList())
	}

	return nil, nil, fmt.Errorf("unknown command: %s\n\nRun '%s help' for usage", name, a.name)
}

// showHelp displays the application help.
func (a *App) showHelp() error {
	if a.colorEnabled {
		// Use the styled tui-based help
		view := a.renderAppHelp()
		if err := tui.Fprint(a.stdout, view); err != nil {
			return err
		}
		return &HelpRequested{}
	}

	// Fallback to plain text for non-color terminals
	var sb strings.Builder

	// App name and description
	sb.WriteString(a.name)
	if a.description != "" {
		sb.WriteString(" - ")
		sb.WriteString(a.description)
	}
	sb.WriteString("\n\n")

	// Version
	if a.version != "" {
		sb.WriteString("Version: ")
		sb.WriteString(a.version)
		sb.WriteString("\n\n")
	}

	// Usage section
	sb.WriteString("Usage:\n  ")
	sb.WriteString(a.name)
	hasSubcmds := a.hasSubcommands()
	if hasSubcmds {
		sb.WriteString(" <command> [flags] [args]\n\n")
	} else {
		// Root-only app
		rootCmd := a.commands[""]
		hasFlags := len(a.globalFlags) > 0 || (rootCmd != nil && len(rootCmd.flags) > 0)
		if hasFlags {
			sb.WriteString(" [flags]")
		}
		// Add root command args
		if rootCmd != nil {
			for _, arg := range rootCmd.args {
				if arg.Required {
					sb.WriteString(" <" + arg.Name + ">")
				} else {
					sb.WriteString(" [" + arg.Name + "]")
				}
			}
		} else {
			for _, arg := range a.args {
				if arg.Required {
					sb.WriteString(" <" + arg.Name + ">")
				} else {
					sb.WriteString(" [" + arg.Name + "]")
				}
			}
		}
		sb.WriteString("\n\n")
	}

	// Commands section
	if len(a.commands) > 0 && hasSubcmds {
		sb.WriteString("Commands:\n")
		for name, cmd := range a.commands {
			if cmd.hidden || name == "" {
				continue
			}
			sb.WriteString(fmt.Sprintf("  %-15s %s\n", name, cmd.description))
		}
		sb.WriteString("\n")
	}

	// Command groups section
	if len(a.groups) > 0 {
		sb.WriteString("Command Groups:\n")
		for name, group := range a.groups {
			sb.WriteString(fmt.Sprintf("  %-15s %s\n", name, group.description))
		}
		sb.WriteString("\n")
	}

	// Root command flags section (always show if root command has flags)
	if rootCmd := a.commands[""]; rootCmd != nil && len(rootCmd.flags) > 0 {
		sb.WriteString("Flags:\n")
		writeFlagsHelp(&sb, rootCmd.flags)
		sb.WriteString("\n")
	}

	// Global flags section
	if len(a.globalFlags) > 0 {
		sb.WriteString("Global Flags:\n")
		writeFlagsHelp(&sb, a.globalFlags)
		sb.WriteString("\n")
	}

	// Help hint (only show if there are subcommands)
	if hasSubcmds {
		sb.WriteString("Run '")
		sb.WriteString(a.name)
		sb.WriteString(" <command> --help' for more information on a command.\n")
	}

	fmt.Fprint(a.stdout, sb.String())
	return &HelpRequested{}
}

// Group organizes related commands under a common namespace.
//
// Groups provide hierarchical organization for complex CLIs:
//
//	users := app.Group("users").Description("User management commands")
//	users.Command("list").Run(listHandler)
//	users.Command("create").Args("username").Run(createHandler)
//
// Groups can have their own handler that runs when invoked without a subcommand,
// their own flags, and middleware that applies to all subcommands.
type Group struct {
	name        string
	description string
	app         *App
	commands    map[string]*Command

	// Handler for running group without subcommand
	handler    Handler
	flags      []Flag
	args       []*Arg
	middleware []Middleware
	validators []func(*Context) error
}

// Description sets the group description.
func (g *Group) Description(desc string) *Group {
	g.description = desc
	return g
}

// Command adds a command to the group.
// Use builder methods like Description(), Args(), and Flags() to configure the command.
func (g *Group) Command(name string) *Command {
	cmd := newCommand(name, g.app)
	cmd.group = g
	g.commands[name] = cmd
	return cmd
}

// Run sets the handler that runs when the group is invoked without a subcommand.
func (g *Group) Run(h Handler) *Group {
	g.handler = h
	return g
}

// Flags adds typed flags to the group.
func (g *Group) Flags(flags ...Flag) *Group {
	g.flags = append(g.flags, flags...)
	return g
}

// Args sets the positional argument names for the group.
// Append "?" to make an argument optional (e.g., "name?").
func (g *Group) Args(names ...string) *Group {
	for _, name := range names {
		required := true
		if strings.HasSuffix(name, "?") {
			name = strings.TrimSuffix(name, "?")
			required = false
		}
		g.args = append(g.args, &Arg{
			Name:     name,
			Required: required,
		})
	}
	return g
}

// Use adds middleware to the group.
func (g *Group) Use(mw ...Middleware) *Group {
	g.middleware = append(g.middleware, mw...)
	return g
}

// Validate adds a validation function for the group.
func (g *Group) Validate(v func(*Context) error) *Group {
	g.validators = append(g.validators, v)
	return g
}

// asCommand returns a Command that wraps the group for execution.
func (g *Group) asCommand() *Command {
	return &Command{
		name:        g.name,
		description: g.description,
		app:         g.app,
		handler:     g.handler,
		flags:       g.flags,
		args:        g.args,
		middleware:  g.middleware,
		validators:  g.validators,
	}
}

func (g *Group) commandList() string {
	var sb strings.Builder
	for name, cmd := range g.commands {
		if g.app.colorEnabled {
			sb.WriteString(fmt.Sprintf("  %s %s - %s\n", color.Green.Apply(g.name), color.Green.Apply(name), cmd.description))
		} else {
			sb.WriteString(fmt.Sprintf("  %s %s - %s\n", g.name, name, cmd.description))
		}
	}
	return sb.String()
}

// showHelp displays help for the group.
func (g *Group) showHelp() error {
	var sb strings.Builder

	// Group name and description
	sb.WriteString(g.name)
	if g.description != "" {
		sb.WriteString(" - ")
		sb.WriteString(g.description)
	}
	sb.WriteString("\n\n")

	// Usage
	sb.WriteString("Usage:\n  ")
	sb.WriteString(g.app.name)
	sb.WriteString(" ")
	sb.WriteString(g.name)
	sb.WriteString(" <command> [flags] [args]\n\n")

	// Commands
	sb.WriteString("Commands:\n")
	for name, cmd := range g.commands {
		if cmd.hidden {
			continue
		}
		sb.WriteString(fmt.Sprintf("  %-15s %s\n", name, cmd.description))
	}
	sb.WriteString("\n")

	// Group flags
	if len(g.flags) > 0 {
		sb.WriteString("Flags:\n")
		writeFlagsHelp(&sb, g.flags)
		sb.WriteString("\n")
	}

	// Help hint
	sb.WriteString("Run '")
	sb.WriteString(g.app.name)
	sb.WriteString(" ")
	sb.WriteString(g.name)
	sb.WriteString(" <command> --help' for more information on a command.\n")

	fmt.Fprint(g.app.stdout, sb.String())
	return &HelpRequested{}
}

// SetColorEnabled enables or disables colored output.
func (a *App) SetColorEnabled(enabled bool) *App {
	a.colorEnabled = enabled
	return a
}

// HelpTheme sets a custom theme for help output styling.
// Use DefaultHelpTheme() to get the default theme and modify it.
//
// Example:
//
//	theme := cli.DefaultHelpTheme()
//	theme.TitleStart = color.NewRGB(255, 100, 100)
//	theme.TitleEnd = color.NewRGB(255, 100, 100)
//	app.HelpTheme(theme)
func (a *App) HelpTheme(theme HelpTheme) *App {
	a.helpTheme = &theme
	return a
}

// SetStdin sets the input reader for the application.
//
// This is useful for programmatic invocation or testing:
//
//	app.SetStdin(strings.NewReader("user input"))
func (a *App) SetStdin(r io.Reader) *App {
	a.stdin = r
	return a
}

// SetStdout sets the output writer for the application.
//
// This is useful for capturing output programmatically:
//
//	var buf bytes.Buffer
//	app.SetStdout(&buf)
//	app.ExecuteArgs([]string{"greet"})
//	output := buf.String()
func (a *App) SetStdout(w io.Writer) *App {
	a.stdout = w
	return a
}

// SetStderr sets the error output writer for the application.
//
// This is useful for capturing error output programmatically:
//
//	var buf bytes.Buffer
//	app.SetStderr(&buf)
func (a *App) SetStderr(w io.Writer) *App {
	a.stderr = w
	return a
}
