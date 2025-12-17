// Package cli provides a CLI framework that integrates with Wonton's TUI capabilities.
// It supports command registration, flag parsing, hierarchical configuration,
// and progressive interactivity (same command works as quick one-liner or rich TUI).
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

// App is the main CLI application.
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

// New creates a new CLI application.
// Use builder methods like Description() and Version() to configure the app.
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

// Group creates a command group for organizing subcommands.
// Use builder methods like Description() to configure the group.
func (a *App) Group(name string) *Group {
	g := &Group{
		name:     name,
		app:      a,
		commands: make(map[string]*Command),
	}
	a.groups[name] = g
	return g
}

// Use adds middleware to the application.
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

// Action sets the root handler that executes when no command is specified.
func (a *App) Action(h Handler) *App {
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

// Run executes the CLI application with os.Args.
func (a *App) Run() error {
	return a.RunContext(context.Background(), os.Args[1:])
}

// RunArgs executes the CLI application with the given arguments.
func (a *App) RunArgs(args []string) error {
	return a.RunContext(context.Background(), args)
}

// ForceInteractive sets the interactive mode for testing purposes.
// Pass true to force interactive, false to force non-interactive.
func (a *App) ForceInteractive(interactive bool) *App {
	a.forceInteractive = &interactive
	return a
}

// RunContext executes the CLI application with context and arguments.
func (a *App) RunContext(ctx context.Context, args []string) error {
	// Detect interactivity (can be overridden for testing)
	if a.forceInteractive != nil {
		a.isInteractive = *a.forceInteractive
	} else {
		a.isInteractive = isTerminal(os.Stdin) && isTerminal(os.Stdout)
	}

	// Parse command and flags
	cmdName, cmdArgs, err := a.parseArgs(args)
	if err != nil {
		return err
	}

	// Handle built-in commands
	switch cmdName {
	case "help":
		return a.showHelp()
	case "version":
		if a.version != "" {
			fmt.Fprintln(a.stdout, a.version)
		}
		return nil
	}

	// Find the command (or use root handler)
	var cmd *Command
	if cmdName == "" {
		if a.handler == nil {
			return a.showHelp()
		}
		// Use root handler
		cmd = a.rootCommand()
	} else {
		var subCmdArgs []string
		var err error
		cmd, subCmdArgs, err = a.findCommand(cmdName, cmdArgs)
		if err != nil {
			// If command not found but app has root handler,
			// treat first arg as positional arg to root handler
			if a.handler != nil {
				cmd = a.rootCommand()
				// Put cmdName back as first arg
				cmdArgs = append([]string{cmdName}, cmdArgs...)
			} else {
				return err
			}
		} else {
			// Update cmdArgs if we consumed a subcommand name
			cmdArgs = subCmdArgs
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

	// Build middleware chain
	for i := len(a.middleware) - 1; i >= 0; i-- {
		handler = a.middleware[i](handler)
	}
	for i := len(cmd.middleware) - 1; i >= 0; i-- {
		handler = cmd.middleware[i](handler)
	}

	// Execute
	return handler(execCtx)
}

// parseArgs separates the command name from its arguments.
func (a *App) parseArgs(args []string) (string, []string, error) {
	if len(args) == 0 {
		return "", nil, nil
	}

	// Check if first arg is a flag (global flag)
	if strings.HasPrefix(args[0], "-") {
		return "", args, nil
	}

	return args[0], args[1:], nil
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
		return tui.Fprint(a.stdout, view)
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
	sb.WriteString(" <command> [flags] [args]\n\n")

	// Commands section
	if len(a.commands) > 0 {
		sb.WriteString("Commands:\n")
		for name, cmd := range a.commands {
			if cmd.hidden {
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

	// Global flags section
	if len(a.globalFlags) > 0 {
		sb.WriteString("Global Flags:\n")
		for _, f := range a.globalFlags {
			if f.IsHidden() {
				continue
			}
			writeFlagHelp(&sb, f)
		}
		sb.WriteString("\n")
	}

	// Help hint
	sb.WriteString("Run '")
	sb.WriteString(a.name)
	sb.WriteString(" <command> --help' for more information on a command.\n")

	fmt.Fprint(a.stdout, sb.String())
	return nil
}

// Group organizes related commands.
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

// Action sets the handler that runs when the group is invoked without a subcommand.
func (g *Group) Action(h Handler) *Group {
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
