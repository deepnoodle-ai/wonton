// Package cli provides a CLI framework that integrates with Wonton's TUI capabilities.
// It supports command registration, flag parsing, hierarchical configuration,
// and progressive interactivity (same command works as quick one-liner or rich TUI).
package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"

	"github.com/deepnoodle-ai/wonton/color"
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
	globalFlags     []*Flag
	globalFlagsDefs any

	// Config resolution
	configPaths []string
	configType  string

	// I/O
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer

	// Runtime state
	isInteractive    bool
	forceInteractive *bool // For testing - if set, overrides TTY detection
	colorEnabled     bool  // Whether to use colored output
}

// New creates a new CLI application.
func New(name, description string) *App {
	return &App{
		name:         name,
		description:  description,
		commands:     make(map[string]*Command),
		groups:       make(map[string]*Group),
		stdin:        os.Stdin,
		stdout:       os.Stdout,
		stderr:       os.Stderr,
		colorEnabled: color.IsTerminal(os.Stdout),
	}
}

// Version sets the application version.
func (a *App) Version(v string) *App {
	a.version = v
	return a
}

// Command registers a new command.
func (a *App) Command(name, description string, opts ...CommandOption) *Command {
	cmd := newCommand(name, description, a)
	for _, opt := range opts {
		opt(cmd)
	}
	a.commands[name] = cmd
	return cmd
}

// Group creates a command group for organizing subcommands.
func (a *App) Group(name, description string) *Group {
	g := &Group{
		name:        name,
		description: description,
		app:         a,
		commands:    make(map[string]*Command),
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
func (a *App) AddGlobalFlag(f *Flag) *App {
	a.globalFlags = append(a.globalFlags, f)
	return a
}

// GlobalFlags adds global flags from a struct type using reflection.
func GlobalFlags[T any]() func(*App) {
	return func(a *App) {
		var t T
		// Use parseStructFlags to extract flags from the struct
		dummyCmd := &Command{flags: make([]*Flag, 0)}
		parseStructFlags(dummyCmd, reflect.TypeOf(t))
		a.globalFlags = append(a.globalFlags, dummyCmd.flags...)
		a.globalFlagsDefs = t
	}
}

// WithGlobalFlags is an app option that adds global flags from a struct.
func WithGlobalFlags[T any]() func(*App) {
	return GlobalFlags[T]()
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
	case "", "help":
		return a.showHelp()
	case "version":
		if a.version != "" {
			fmt.Fprintln(a.stdout, a.version)
		}
		return nil
	}

	// Find the command
	cmd, err := a.findCommand(cmdName)
	if err != nil {
		return err
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
func (a *App) findCommand(name string) (*Command, error) {
	// Check direct commands first
	if cmd, ok := a.commands[name]; ok {
		return cmd, nil
	}

	// Check aliases for direct commands
	for _, cmd := range a.commands {
		for _, alias := range cmd.aliases {
			if alias == name {
				return cmd, nil
			}
		}
	}

	// Check for group:command pattern
	parts := strings.SplitN(name, ":", 2)
	if len(parts) == 2 {
		if g, ok := a.groups[parts[0]]; ok {
			if cmd, ok := g.commands[parts[1]]; ok {
				return cmd, nil
			}
			// Check aliases within the group
			for _, cmd := range g.commands {
				for _, alias := range cmd.aliases {
					if alias == parts[1] {
						return cmd, nil
					}
				}
			}
		}
	}

	// Check groups (e.g., "users list" becomes ["users", "list"])
	// This is handled by looking for space-separated args
	for groupName, group := range a.groups {
		if name == groupName {
			// Show group help
			return nil, fmt.Errorf("group '%s' requires a subcommand\n\nAvailable commands:\n%s",
				groupName, group.commandList())
		}
	}

	return nil, fmt.Errorf("unknown command: %s\n\nRun '%s help' for usage", name, a.name)
}

// showHelp displays the application help.
func (a *App) showHelp() error {
	var sb strings.Builder

	// App name and description
	if a.colorEnabled {
		sb.WriteString(color.Bold(color.BrightCyan.Apply(a.name)))
	} else {
		sb.WriteString(a.name)
	}
	if a.description != "" {
		sb.WriteString(" - ")
		sb.WriteString(a.description)
	}
	sb.WriteString("\n\n")

	// Version
	if a.version != "" {
		if a.colorEnabled {
			sb.WriteString(color.Yellow.Apply("Version:"))
		} else {
			sb.WriteString("Version:")
		}
		sb.WriteString(" ")
		sb.WriteString(a.version)
		sb.WriteString("\n\n")
	}

	// Usage section
	if a.colorEnabled {
		sb.WriteString(color.Yellow.Apply("Usage:"))
	} else {
		sb.WriteString("Usage:")
	}
	sb.WriteString("\n  ")
	sb.WriteString(a.name)
	sb.WriteString(" <command> [flags] [args]\n\n")

	// Commands section
	if len(a.commands) > 0 {
		if a.colorEnabled {
			sb.WriteString(color.Yellow.Apply("Commands:"))
		} else {
			sb.WriteString("Commands:")
		}
		sb.WriteString("\n")
		for name, cmd := range a.commands {
			if cmd.hidden {
				continue
			}
			if a.colorEnabled {
				sb.WriteString(fmt.Sprintf("  %-15s %s\n", color.Green.Apply(name), cmd.description))
			} else {
				sb.WriteString(fmt.Sprintf("  %-15s %s\n", name, cmd.description))
			}
		}
		sb.WriteString("\n")
	}

	// Command groups section
	if len(a.groups) > 0 {
		if a.colorEnabled {
			sb.WriteString(color.Yellow.Apply("Command Groups:"))
		} else {
			sb.WriteString("Command Groups:")
		}
		sb.WriteString("\n")
		for name, group := range a.groups {
			if a.colorEnabled {
				sb.WriteString(fmt.Sprintf("  %-15s %s\n", color.Green.Apply(name), group.description))
			} else {
				sb.WriteString(fmt.Sprintf("  %-15s %s\n", name, group.description))
			}
		}
		sb.WriteString("\n")
	}

	// Help hint
	sb.WriteString("Run '")
	if a.colorEnabled {
		sb.WriteString(color.Cyan.Apply(a.name + " <command> --help"))
	} else {
		sb.WriteString(a.name)
		sb.WriteString(" <command> --help")
	}
	sb.WriteString("' for more information on a command.\n")

	fmt.Fprint(a.stdout, sb.String())
	return nil
}

// Group organizes related commands.
type Group struct {
	name        string
	description string
	app         *App
	commands    map[string]*Command
}

// Command adds a command to the group.
func (g *Group) Command(name, description string, opts ...CommandOption) *Command {
	cmd := newCommand(name, description, g.app)
	cmd.group = g
	for _, opt := range opts {
		opt(cmd)
	}
	g.commands[name] = cmd
	return cmd
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
