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
	longDesc    string
	version     string

	commands     map[string]*Command
	commandOrder []string // insertion order for stable help output
	groups       map[string]*Group
	groupOrder   []string // insertion order for stable help output
	middleware   []Middleware

	// Global flags
	globalFlags []Flag

	// Root handler (runs when no command specified)
	handler    Handler
	args       []*Arg
	validators []func(*Context) error

	// Examples for help output
	examples []Example

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

	// expandGroups controls whether command groups show their subcommands
	// inline in the main help output. Defaults to false (collapsed).
	// Individual groups can override this via Group.Expand.
	expandGroups bool
}

// Example represents a usage example for help output.
type Example struct {
	Description string
	Command     string
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

// Long sets a longer description or quickstart text for help output.
// This text appears after the usage section and before commands.
//
// Example:
//
//	app.Long("Quick start:\n  myapp login\n  myapp \"Do something\"")
func (a *App) Long(desc string) *App {
	a.longDesc = desc
	return a
}

// Examples adds usage examples to the application help output.
// Examples appear after the description and before the commands section.
//
// Example:
//
//	app.Examples(
//	    cli.NewExample("Start interactively", "myapp"),
//	    cli.NewExample("With a prompt", "myapp \"Design a logo\""),
//	)
func (a *App) Examples(examples ...Example) *App {
	a.examples = append(a.examples, examples...)
	return a
}

// NewExample creates a new usage example for help output.
func NewExample(description, command string) Example {
	return Example{Description: description, Command: command}
}

// Command registers a new command or returns an existing one.
// Use builder methods like Description(), Args(), and Flags() to configure the command.
func (a *App) Command(name string) *Command {
	if existing, ok := a.commands[name]; ok {
		return existing
	}
	cmd := newCommand(name, a)
	a.commands[name] = cmd
	a.commandOrder = append(a.commandOrder, name)
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
	if existing, ok := a.groups[name]; ok {
		return existing
	}
	g := &Group{
		name:     name,
		app:      a,
		commands: make(map[string]*Command),
	}
	a.groups[name] = g
	a.groupOrder = append(a.groupOrder, name)
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
//
// Each name is a small DSL describing the slot's cardinality:
//
//	"name"     required, exactly one
//	"name?"    optional, zero or one
//	"name..."  variadic, one or more (trailing)
//	"name?..." variadic, zero or more (trailing)
//
// Extra positionals beyond declared slots are rejected at parse time
// unless the last slot is variadic. See Command.Args for details.
func (a *App) Args(names ...string) *App {
	a.args = append(a.args, parseArgSpecs(names)...)
	validateArgSlots(a.args)
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
//	        app.PrintError(err)
//	        os.Exit(cli.GetExitCode(err))
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

	// Apply --no-color / --color / --force-color command-line overrides.
	// These take precedence over env vars and TTY detection, matching the
	// common CLI precedence convention.
	args = a.applyColorOverrides(args)

	// Parse command and arguments using definition-driven parser
	p := newParser(a)
	result, err := p.parse(args)
	if err != nil {
		return err
	}

	// Check for help/version in global flags
	for _, gf := range result.GlobalFlags {
		if gf == "--help" || (gf == "-h" && a.isHelpShort()) {
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
			return a.unknownCommandError(result.Group)
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
						return a.unknownSubcommandError(result.Group, firstArg)
					}
					// Group has a handler, treat as positional arg
				} else if group.handler == nil {
					// First arg is a flag but group has no handler - requires a subcommand
					return a.groupRequiresSubcommandError(result.Group)
				}
			} else if group.handler == nil {
				// No args and no handler - requires a subcommand
				return a.groupRequiresSubcommandError(result.Group)
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
				return a.unknownSubcommandError(result.Group, result.Command)
			}
		}
	} else {
		// Direct command
		cmd = a.commands[result.Command]
		if cmd == nil {
			// Check aliases
			for _, name := range orderedCommandKeys(a.commands, a.commandOrder) {
				c := a.commands[name]
				if c == nil {
					continue
				}
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
				return a.unknownCommandError(result.Command)
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

// isHelpShort reports whether -h should trigger help at the app level.
// Returns false if a global flag has explicitly claimed -h as its short name.
func (a *App) isHelpShort() bool {
	return a.findGlobalFlagByShort("h") == nil
}

// applyColorOverrides scans args for --no-color, --color, and --force-color
// flags and updates the app's color setting accordingly. The recognized
// flags are stripped from the returned args so downstream parsing is not
// affected. Command-line flags take precedence over env vars and TTY
// detection.
func (a *App) applyColorOverrides(args []string) []string {
	out := make([]string, 0, len(args))
	for i, arg := range args {
		// Stop interpreting flags once we hit the end-of-flags marker so
		// positional arguments like "--no-color" can be passed verbatim.
		if arg == "--" {
			out = append(out, args[i:]...)
			return out
		}
		switch arg {
		case "--no-color", "--no-colour":
			a.colorEnabled = false
			// Mirror into the color package so direct color.Apply calls in
			// command handlers follow the user's explicit choice too.
			color.Enabled = false
			continue
		case "--color", "--colour", "--force-color":
			a.colorEnabled = true
			color.Enabled = true
			continue
		}
		out = append(out, arg)
	}
	return out
}

// suggestName returns the closest match to `attempted` from `candidates`
// using Levenshtein distance, or "" if no candidate is close enough. The
// threshold is max(2, len(attempted)/3) so short typos are tolerated
// while unrelated strings don't get matched.
func suggestName(attempted string, candidates []string) string {
	if len(candidates) == 0 || attempted == "" {
		return ""
	}
	threshold := len(attempted) / 3
	if threshold < 2 {
		threshold = 2
	}
	best := ""
	bestDist := threshold + 1
	for _, c := range candidates {
		if c == "" {
			continue
		}
		d := levenshtein(attempted, c)
		if d < bestDist {
			bestDist = d
			best = c
		}
	}
	if bestDist > threshold {
		return ""
	}
	return best
}

// levenshtein returns the edit distance between two strings.
func levenshtein(a, b string) int {
	ar := []rune(a)
	br := []rune(b)
	la, lb := len(ar), len(br)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}
	prev := make([]int, lb+1)
	curr := make([]int, lb+1)
	for j := 0; j <= lb; j++ {
		prev[j] = j
	}
	for i := 1; i <= la; i++ {
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if ar[i-1] == br[j-1] {
				cost = 0
			}
			del := prev[j] + 1
			ins := curr[j-1] + 1
			sub := prev[j-1] + cost
			m := del
			if ins < m {
				m = ins
			}
			if sub < m {
				m = sub
			}
			curr[j] = m
		}
		prev, curr = curr, prev
	}
	return prev[lb]
}

// topLevelCommandNames returns visible, invokable top-level names: direct
// commands (excluding the empty root name), their aliases, and group names.
func (a *App) topLevelCommandNames() []string {
	out := make([]string, 0, len(a.commands)+len(a.groups))
	for _, name := range orderedCommandKeys(a.commands, a.commandOrder) {
		cmd := a.commands[name]
		if name == "" || cmd == nil || cmd.hidden {
			continue
		}
		out = append(out, name)
		out = append(out, cmd.aliases...)
	}
	for _, name := range orderedGroupKeys(a.groups, a.groupOrder) {
		g := a.groups[name]
		if g == nil {
			continue
		}
		out = append(out, name)
		out = append(out, g.aliases...)
	}
	return out
}

// groupSubcommandNames returns visible subcommand names and aliases for a group.
func groupSubcommandNames(g *Group) []string {
	out := make([]string, 0, len(g.commands))
	for _, name := range orderedCommandKeys(g.commands, g.commandOrder) {
		cmd := g.commands[name]
		if cmd == nil || cmd.hidden {
			continue
		}
		out = append(out, name)
		out = append(out, cmd.aliases...)
	}
	return out
}

func writePlainCommandRows(sb *strings.Builder, commands map[string]*Command, order []string, indent int) {
	names := orderedCommandKeys(commands, order)
	width := maxVisibleCommandNameLen(commands, names)
	prefix := strings.Repeat(" ", indent)
	for _, name := range names {
		cmd := commands[name]
		if cmd == nil || cmd.hidden || name == "" {
			continue
		}
		sb.WriteString(fmt.Sprintf("%s%-*s  %s\n", prefix, width, name, cmd.description))
	}
}

// unknownCommandError builds an error for an unknown top-level command,
// including a did-you-mean suggestion when one is available and a footer
// pointing to --help.
func (a *App) unknownCommandError(name string) error {
	msg := fmt.Sprintf("unknown command: %s", name)
	if suggestion := suggestName(name, a.topLevelCommandNames()); suggestion != "" {
		msg += fmt.Sprintf("\n\nDid you mean '%s'?", suggestion)
	}
	msg += fmt.Sprintf("\n\nRun '%s --help' to see available commands.", a.name)
	return fmt.Errorf("%s", msg)
}

// unknownSubcommandError builds an error for an unknown subcommand of a
// group, with a did-you-mean suggestion and a pointer to the group's help.
func (a *App) unknownSubcommandError(groupName, subName string) error {
	msg := fmt.Sprintf("unknown subcommand '%s' for group '%s'", subName, groupName)
	if g := a.groups[groupName]; g != nil {
		if suggestion := suggestName(subName, groupSubcommandNames(g)); suggestion != "" {
			msg += fmt.Sprintf("\n\nDid you mean '%s %s'?", groupName, suggestion)
		}
	}
	msg += fmt.Sprintf("\n\nRun '%s %s --help' to see available subcommands.", a.name, groupName)
	return fmt.Errorf("%s", msg)
}

// groupRequiresSubcommandError builds an error for the case where a user
// invoked a group that has no handler and provided no subcommand. Points
// the user at the group's --help for the list of available subcommands.
func (a *App) groupRequiresSubcommandError(groupName string) error {
	msg := fmt.Sprintf("group '%s' requires a subcommand", groupName)
	msg += fmt.Sprintf("\n\nRun '%s %s --help' to see available subcommands.", a.name, groupName)
	return fmt.Errorf("%s", msg)
}

// findGroup looks up a group by its canonical name or any of its aliases.
// It returns the group and its canonical name. The canonical name is the
// name the group was registered under with App.Group.
func (a *App) findGroup(name string) (*Group, string, bool) {
	if g, ok := a.groups[name]; ok {
		return g, name, true
	}
	for _, canonical := range orderedGroupKeys(a.groups, a.groupOrder) {
		g := a.groups[canonical]
		if g == nil {
			continue
		}
		for _, alias := range g.aliases {
			if alias == name {
				return g, canonical, true
			}
		}
	}
	return nil, "", false
}

// showHelp displays the application help.
func (a *App) showHelp() error {
	if a.colorEnabled {
		// Use the styled tui-based help
		view := a.renderAppHelp()
		if err := tui.Fprint(a.stdout, view); err != nil {
			return err
		}
		if err := writeHelpNewline(a.stdout); err != nil {
			return err
		}
		return &HelpRequested{}
	}

	// Fallback to plain text for non-color terminals
	var sb strings.Builder

	// App name, description, and version
	sb.WriteString(a.name)
	if a.description != "" {
		sb.WriteString(" - ")
		sb.WriteString(a.description)
	}
	if a.version != "" {
		sb.WriteString(" (v")
		sb.WriteString(a.version)
		sb.WriteString(")")
	}
	sb.WriteString("\n\n")

	// Long description
	if a.longDesc != "" {
		sb.WriteString(a.longDesc)
		sb.WriteString("\n\n")
	}

	// Usage section
	hasSubcmds := a.hasSubcommands()
	sb.WriteString("Usage:\n")
	if hasSubcmds {
		rootCmd := a.commands[""]
		hasRootWithArgs := rootCmd != nil && (len(rootCmd.args) > 0 || rootCmd.handler != nil)
		if hasRootWithArgs {
			// Show both usage lines
			sb.WriteString(a.buildRootUsageString())
			if rootCmd.usageSummary != "" {
				sb.WriteString("  " + rootCmd.usageSummary)
			}
			sb.WriteString("\n")
			sb.WriteString(fmt.Sprintf("  %s <command> [flags]", a.name))
			sb.WriteString("  Run a subcommand")
			sb.WriteString("\n\n")
		} else {
			sb.WriteString(fmt.Sprintf("  %s <command> [flags] [args]\n\n", a.name))
		}
	} else {
		sb.WriteString(a.buildRootUsageString())
		sb.WriteString("\n\n")
	}

	// Examples section
	if len(a.examples) > 0 {
		sb.WriteString("Examples:\n")
		for i, ex := range a.examples {
			if i > 0 {
				sb.WriteString("\n")
			}
			sb.WriteString(fmt.Sprintf("  %s\n  $ %s\n", ex.Description, ex.Command))
		}
		sb.WriteString("\n")
	}

	// Commands section (in insertion order)
	if len(a.commands) > 0 && hasSubcmds {
		sb.WriteString("Commands:\n")
		writePlainCommandRows(&sb, a.commands, a.commandOrder, 2)
		sb.WriteString("\n")
	}

	// Flat-routed groups appear as their own named sections
	for _, name := range a.groupOrder {
		group := a.groups[name]
		if group == nil || !group.flatRouting {
			continue
		}
		sb.WriteString(fmt.Sprintf("%s:\n", strings.ToUpper(name)))
		writePlainCommandRows(&sb, group.commands, group.commandOrder, 2)
		sb.WriteString("\n")
	}

	// Non-flat-routed command groups section (in insertion order)
	if a.hasNonFlatGroups() {
		sb.WriteString("Command Groups:\n")
		order := a.groupOrder
		first := true
		for _, name := range order {
			group := a.groups[name]
			if group == nil || group.flatRouting {
				continue
			}
			if !first && group.isExpanded() {
				sb.WriteString("\n")
			}
			first = false
			if group.description != "" {
				sb.WriteString(fmt.Sprintf("  %s - %s\n", name, group.description))
			} else {
				sb.WriteString(fmt.Sprintf("  %s\n", name))
			}
			if group.isExpanded() {
				writePlainCommandRows(&sb, group.commands, group.commandOrder, 4)
			}
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

	if err := writeHelpOutput(a.stdout, sb.String()); err != nil {
		return err
	}
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
	name         string
	description  string
	app          *App
	commands     map[string]*Command
	commandOrder []string // insertion order for stable help output

	// Handler for running group without subcommand
	handler    Handler
	flags      []Flag
	args       []*Arg
	middleware []Middleware
	validators []func(*Context) error

	// aliases are alternate names the group can be invoked by.
	aliases []string

	// flatRouting makes group subcommands invocable without the group prefix.
	// For example, with FlatRouting enabled on a "transform" group containing
	// a "resize" command, users can type "app resize" instead of "app transform resize".
	// The group name is still used for visual grouping in help output.
	flatRouting bool

	// expand overrides App.expandGroups for this group. nil means inherit.
	expand *bool
}

// Alias adds group aliases. Aliases can be used interchangeably with the
// group name, including in the "group subcommand" and "group:subcommand"
// forms (e.g., if "workflows" has alias "workflow", both "app workflows run"
// and "app workflow run" resolve to the same command).
func (g *Group) Alias(names ...string) *Group {
	g.aliases = append(g.aliases, names...)
	return g
}

// Description sets the group description.
func (g *Group) Description(desc string) *Group {
	g.description = desc
	return g
}

// Command adds a command to the group.
// Use builder methods like Description(), Args(), and Flags() to configure the command.
func (g *Group) Command(name string) *Command {
	if existing, ok := g.commands[name]; ok {
		return existing
	}
	cmd := newCommand(name, g.app)
	cmd.group = g
	g.commands[name] = cmd
	g.commandOrder = append(g.commandOrder, name)
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
//
// See Command.Args for the full DSL. Extra positionals beyond declared
// slots are rejected at parse time unless the last slot is variadic.
func (g *Group) Args(names ...string) *Group {
	g.args = append(g.args, parseArgSpecs(names)...)
	validateArgSlots(g.args)
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

// FlatRouting makes the group's subcommands directly invocable without the
// group prefix. The group name becomes a help-only section header.
//
// For example, with a "transform" group containing "resize" and "crop":
//
//	app.Group("transform").FlatRouting(true).Description("Image transforms")
//	// Users can now invoke: app resize (instead of app transform resize)
//	// Help output shows commands grouped under a TRANSFORM section header
//
// Direct top-level commands always take priority over flat-routed group commands.
// The group prefix ("app transform resize") and colon syntax ("app transform:resize")
// continue to work as well.
func (g *Group) FlatRouting(enabled bool) *Group {
	g.flatRouting = enabled
	return g
}

// Expand overrides App.ExpandGroups for this group. When true, the group's
// subcommands are shown inline in the main help output; when false, only the
// group name and description are shown. If not set, the app-level default
// applies.
func (g *Group) Expand(enabled bool) *Group {
	g.expand = &enabled
	return g
}

// isExpanded reports whether this group should be rendered expanded in help.
func (g *Group) isExpanded() bool {
	if g.expand != nil {
		return *g.expand
	}
	if g.app != nil {
		return g.app.expandGroups
	}
	return false
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
	for _, name := range g.commandOrder {
		cmd := g.commands[name]
		if cmd == nil {
			continue
		}
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
	if g.app.colorEnabled {
		view := g.renderGroupHelp()
		if err := tui.Fprint(g.app.stdout, view); err != nil {
			return err
		}
		if err := writeHelpNewline(g.app.stdout); err != nil {
			return err
		}
		return &HelpRequested{}
	}

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

	// Aliases
	if len(g.aliases) > 0 {
		sb.WriteString("Aliases:\n  ")
		sb.WriteString(strings.Join(g.aliases, ", "))
		sb.WriteString("\n\n")
	}

	// Commands
	sb.WriteString("Commands:\n")
	writePlainCommandRows(&sb, g.commands, g.commandOrder, 2)
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

	if err := writeHelpOutput(g.app.stdout, sb.String()); err != nil {
		return err
	}
	return &HelpRequested{}
}

// SetColorEnabled enables or disables colored output. The setting is also
// mirrored into color.Enabled so that direct color.Apply calls in command
// handlers follow the same choice.
func (a *App) SetColorEnabled(enabled bool) *App {
	a.colorEnabled = enabled
	color.Enabled = enabled
	return a
}

// PrintError writes a formatted error message to the app's stderr. The
// message is prefixed with "Error: " and colorized red when color is
// enabled (respecting NO_COLOR, --no-color, and TTY detection).
//
// Nil errors and HelpRequested errors are ignored so callers can use the
// typical pattern:
//
//	if err := app.Execute(); err != nil {
//	    app.PrintError(err)
//	    os.Exit(cli.GetExitCode(err))
//	}
func (a *App) PrintError(err error) {
	if err == nil || IsHelpRequested(err) {
		return
	}
	prefix := "Error: "
	msg := err.Error()

	// Split off any trailing "tail" — did-you-mean suggestions and help
	// hints — so we can color only the primary error message red and
	// leave the rest in the default style. The convention used by this
	// package is to separate the tail with a blank line.
	head, tail, _ := strings.Cut(msg, "\n\n")

	if a.colorEnabled {
		fmt.Fprintln(a.stderr, color.Red.Apply(prefix+head))
	} else {
		fmt.Fprintln(a.stderr, prefix+head)
	}
	if tail != "" {
		fmt.Fprintln(a.stderr)
		fmt.Fprintln(a.stderr, tail)
	}
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

// ExpandGroups controls whether command groups show their subcommands inline
// in the main help output. Defaults to false (groups are collapsed, showing
// only the group name and description). Individual groups can override this
// via Group.Expand.
func (a *App) ExpandGroups(enabled bool) *App {
	a.expandGroups = enabled
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
