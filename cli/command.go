package cli

import (
	"fmt"
	"strings"
)

// Handler is the function type for command handlers.
type Handler func(*Context) error

// Middleware wraps a handler to add behavior.
type Middleware func(Handler) Handler

// Command represents a CLI command.
type Command struct {
	name        string
	description string
	longDesc    string
	app         *App
	group       *Group

	// Handler
	handler Handler

	// Flags and args
	flags    []*Flag
	args     []*Arg
	flagDefs any // Struct type for flag parsing

	// Options
	middleware  []Middleware
	hidden      bool
	deprecated  string
	aliases     []string
	isTool      bool // Marks as AI-callable tool
	interactive Handler
	nonInteract Handler

	// Validation
	validators []func(*Context) error
}

// newCommand creates a new command.
func newCommand(name, description string, app *App) *Command {
	return &Command{
		name:        name,
		description: description,
		app:         app,
		flags:       make([]*Flag, 0),
		args:        make([]*Arg, 0),
	}
}

// Name returns the command name.
func (c *Command) Name() string {
	return c.name
}

// Description returns the command description.
func (c *Command) Description() string {
	return c.description
}

// Run sets the command handler.
func (c *Command) Run(h Handler) *Command {
	c.handler = h
	return c
}

// Long sets a longer description for help output.
func (c *Command) Long(desc string) *Command {
	c.longDesc = desc
	return c
}

// Hidden hides the command from help output.
func (c *Command) Hidden() *Command {
	c.hidden = true
	return c
}

// Deprecated marks the command as deprecated.
func (c *Command) Deprecated(msg string) *Command {
	c.deprecated = msg
	return c
}

// Alias adds command aliases.
func (c *Command) Alias(names ...string) *Command {
	c.aliases = append(c.aliases, names...)
	return c
}

// Use adds middleware to the command.
func (c *Command) Use(mw ...Middleware) *Command {
	c.middleware = append(c.middleware, mw...)
	return c
}

// Tool marks the command as an AI-callable tool.
func (c *Command) Tool() *Command {
	c.isTool = true
	return c
}

// Interactive sets the interactive mode handler.
func (c *Command) Interactive(h Handler) *Command {
	c.interactive = h
	return c
}

// NonInteractive sets the non-interactive mode handler.
func (c *Command) NonInteractive(h Handler) *Command {
	c.nonInteract = h
	return c
}

// Validate adds a validation function.
func (c *Command) Validate(v func(*Context) error) *Command {
	c.validators = append(c.validators, v)
	return c
}

// Flag configuration

// Flag represents a command-line flag.
type Flag struct {
	Name        string
	Short       string // Single character shorthand
	Description string
	Default     any
	Required    bool
	Enum        []string // Allowed values
	EnvVar      string   // Environment variable to check
	Hidden      bool
}

// Arg represents a positional argument.
type Arg struct {
	Name        string
	Description string
	Required    bool
	Default     any
}

// AddFlag adds a flag to the command.
func (c *Command) AddFlag(f *Flag) *Command {
	c.flags = append(c.flags, f)
	return c
}

// AddArg adds a positional argument to the command.
func (c *Command) AddArg(a *Arg) *Command {
	c.args = append(c.args, a)
	return c
}

// CommandOption is a functional option for configuring commands.
type CommandOption func(*Command)

// WithFlags sets flags from a struct type.
func WithFlags(flagStruct any) CommandOption {
	return func(c *Command) {
		c.flagDefs = flagStruct
	}
}

// WithArgs configures positional arguments.
func WithArgs(names ...string) CommandOption {
	return func(c *Command) {
		for _, name := range names {
			required := true
			if strings.HasSuffix(name, "?") {
				name = strings.TrimSuffix(name, "?")
				required = false
			}
			c.args = append(c.args, &Arg{
				Name:     name,
				Required: required,
			})
		}
	}
}

// WithArgsRange sets minimum and maximum argument count.
func WithArgsRange(min, max int) CommandOption {
	return func(c *Command) {
		c.validators = append(c.validators, func(ctx *Context) error {
			n := ctx.NArg()
			if n < min {
				return Errorf("requires at least %d argument(s), got %d", min, n)
			}
			if max >= 0 && n > max {
				return Errorf("accepts at most %d argument(s), got %d", max, n)
			}
			return nil
		})
	}
}

// WithExactArgs requires exactly n arguments.
func WithExactArgs(n int) CommandOption {
	return func(c *Command) {
		c.validators = append(c.validators, func(ctx *Context) error {
			if ctx.NArg() != n {
				return Errorf("requires exactly %d argument(s), got %d", n, ctx.NArg())
			}
			return nil
		})
	}
}

// WithNoArgs requires no arguments.
func WithNoArgs() CommandOption {
	return func(c *Command) {
		c.validators = append(c.validators, func(ctx *Context) error {
			if ctx.NArg() > 0 {
				return Errorf("accepts no arguments, got %d", ctx.NArg())
			}
			return nil
		})
	}
}

// WithValidation adds a custom validation function.
func WithValidation(fn func(*Context) error) CommandOption {
	return func(c *Command) {
		c.validators = append(c.validators, fn)
	}
}

// WithTool marks the command as an AI-callable tool.
func WithTool() CommandOption {
	return func(c *Command) {
		c.isTool = true
	}
}

// allFlags returns all flags including global flags from the app.
func (c *Command) allFlags() []*Flag {
	var all []*Flag
	if c.app != nil {
		all = append(all, c.app.globalFlags...)
	}
	all = append(all, c.flags...)
	return all
}

// parseFlags parses flags from arguments into the context.
func (c *Command) parseFlags(ctx *Context, args []string) error {
	// Get all flags (global + command-specific)
	allFlags := c.allFlags()

	// Initialize with defaults
	for _, f := range allFlags {
		if f.Default != nil {
			ctx.flags[f.Name] = f.Default
		}
	}

	// Parse arguments
	var positional []string
	for i := 0; i < len(args); i++ {
		arg := args[i]

		if arg == "--" {
			// Everything after -- is positional
			positional = append(positional, args[i+1:]...)
			break
		}

		if strings.HasPrefix(arg, "--") {
			// Long flag
			name := strings.TrimPrefix(arg, "--")
			if strings.Contains(name, "=") {
				parts := strings.SplitN(name, "=", 2)
				name = parts[0]
				if err := c.setFlag(ctx, name, parts[1]); err != nil {
					return err
				}
			} else if name == "help" {
				return c.showHelp()
			} else {
				// Check if it's a boolean flag or needs a value
				flag := c.findFlag(name)
				if flag == nil {
					return fmt.Errorf("unknown flag: --%s", name)
				}
				if _, ok := flag.Default.(bool); ok {
					ctx.flags[name] = true
				} else if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
					i++
					if err := c.setFlag(ctx, name, args[i]); err != nil {
						return err
					}
				} else {
					return fmt.Errorf("flag --%s requires a value", name)
				}
			}
		} else if strings.HasPrefix(arg, "-") && len(arg) > 1 {
			// Short flag(s)
			shorts := arg[1:]
			for j, r := range shorts {
				flag := c.findFlagByShort(string(r))
				if flag == nil {
					return fmt.Errorf("unknown flag: -%c", r)
				}
				if _, ok := flag.Default.(bool); ok {
					ctx.flags[flag.Name] = true
				} else if j == len(shorts)-1 && i+1 < len(args) {
					i++
					if err := c.setFlag(ctx, flag.Name, args[i]); err != nil {
						return err
					}
				} else {
					return fmt.Errorf("flag -%c requires a value", r)
				}
			}
		} else {
			positional = append(positional, arg)
		}
	}

	// Set positional arguments
	for i, arg := range c.args {
		if i < len(positional) {
			ctx.positional = append(ctx.positional, positional[i])
		} else if arg.Required {
			return fmt.Errorf("missing required argument: %s", arg.Name)
		} else if arg.Default != nil {
			ctx.positional = append(ctx.positional, fmt.Sprint(arg.Default))
		}
	}

	// Extra positional args beyond defined ones
	if len(positional) > len(c.args) {
		ctx.positional = positional
	}

	// Check required flags (including global flags)
	for _, f := range allFlags {
		if f.Required {
			if _, ok := ctx.flags[f.Name]; !ok {
				// Check env var
				if f.EnvVar != "" {
					if val, ok := lookupEnv(f.EnvVar); ok {
						ctx.flags[f.Name] = val
						continue
					}
				}
				return fmt.Errorf("missing required flag: --%s", f.Name)
			}
		}
	}

	// Run validators
	for _, v := range c.validators {
		if err := v(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (c *Command) findFlag(name string) *Flag {
	// Check global flags first
	if c.app != nil {
		for _, f := range c.app.globalFlags {
			if f.Name == name {
				return f
			}
		}
	}
	// Then check command flags
	for _, f := range c.flags {
		if f.Name == name {
			return f
		}
	}
	return nil
}

func (c *Command) findFlagByShort(short string) *Flag {
	// Check global flags first
	if c.app != nil {
		for _, f := range c.app.globalFlags {
			if f.Short == short {
				return f
			}
		}
	}
	// Then check command flags
	for _, f := range c.flags {
		if f.Short == short {
			return f
		}
	}
	return nil
}

func (c *Command) setFlag(ctx *Context, name, value string) error {
	flag := c.findFlag(name)
	if flag == nil {
		return fmt.Errorf("unknown flag: %s", name)
	}

	// Validate enum
	if len(flag.Enum) > 0 {
		valid := false
		for _, e := range flag.Enum {
			if e == value {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid value for --%s: %s (allowed: %s)",
				name, value, strings.Join(flag.Enum, ", "))
		}
	}

	ctx.flags[name] = value
	return nil
}

func (c *Command) showHelp() error {
	var sb strings.Builder

	// Command name and description
	if c.group != nil {
		sb.WriteString(fmt.Sprintf("%s %s", c.group.name, c.name))
	} else {
		sb.WriteString(c.name)
	}
	sb.WriteString(" - ")
	sb.WriteString(c.description)
	sb.WriteString("\n\n")

	if c.longDesc != "" {
		sb.WriteString(c.longDesc)
		sb.WriteString("\n\n")
	}

	if c.deprecated != "" {
		sb.WriteString("DEPRECATED: ")
		sb.WriteString(c.deprecated)
		sb.WriteString("\n\n")
	}

	// Usage
	sb.WriteString("Usage:\n  ")
	sb.WriteString(c.app.name)
	if c.group != nil {
		sb.WriteString(" ")
		sb.WriteString(c.group.name)
	}
	sb.WriteString(" ")
	sb.WriteString(c.name)
	if len(c.flags) > 0 {
		sb.WriteString(" [flags]")
	}
	for _, arg := range c.args {
		if arg.Required {
			sb.WriteString(" <")
			sb.WriteString(arg.Name)
			sb.WriteString(">")
		} else {
			sb.WriteString(" [")
			sb.WriteString(arg.Name)
			sb.WriteString("]")
		}
	}
	sb.WriteString("\n\n")

	// Arguments
	if len(c.args) > 0 {
		sb.WriteString("Arguments:\n")
		for _, arg := range c.args {
			sb.WriteString(fmt.Sprintf("  %-15s %s", arg.Name, arg.Description))
			if !arg.Required {
				sb.WriteString(" (optional)")
			}
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	// Command-specific Flags
	if len(c.flags) > 0 {
		sb.WriteString("Flags:\n")
		for _, f := range c.flags {
			if f.Hidden {
				continue
			}
			writeFlagHelp(&sb, f)
		}
		sb.WriteString("\n")
	}

	// Global Flags
	if c.app != nil && len(c.app.globalFlags) > 0 {
		sb.WriteString("Global Flags:\n")
		for _, f := range c.app.globalFlags {
			if f.Hidden {
				continue
			}
			writeFlagHelp(&sb, f)
		}
	}

	fmt.Fprint(c.app.stdout, sb.String())
	return &HelpRequested{}
}

// writeFlagHelp writes help text for a single flag.
func writeFlagHelp(sb *strings.Builder, f *Flag) {
	flagStr := "  "
	if f.Short != "" {
		flagStr += fmt.Sprintf("-%s, ", f.Short)
	} else {
		flagStr += "    "
	}
	flagStr += fmt.Sprintf("--%-12s", f.Name)
	sb.WriteString(flagStr)
	sb.WriteString(" ")
	sb.WriteString(f.Description)
	if f.Default != nil && f.Default != "" && f.Default != false {
		sb.WriteString(fmt.Sprintf(" (default: %v)", f.Default))
	}
	if f.Required {
		sb.WriteString(" (required)")
	}
	if len(f.Enum) > 0 {
		sb.WriteString(fmt.Sprintf(" [%s]", strings.Join(f.Enum, "|")))
	}
	sb.WriteString("\n")
}
