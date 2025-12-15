package cli

import (
	"fmt"
	"strings"
	"time"
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
	flags    []Flag
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
func newCommand(name string, app *App) *Command {
	return &Command{
		name:  name,
		app:   app,
		flags: make([]Flag, 0),
		args:  make([]*Arg, 0),
	}
}

// Description sets the command description.
func (c *Command) Description(desc string) *Command {
	c.description = desc
	return c
}

// Args sets the positional argument names.
// Append "?" to make an argument optional (e.g., "name?").
func (c *Command) Args(names ...string) *Command {
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
	return c
}

// Flags adds typed flags to the command.
func (c *Command) Flags(flags ...Flag) *Command {
	c.flags = append(c.flags, flags...)
	return c
}

// Name returns the command name.
func (c *Command) Name() string {
	return c.name
}

// GetDescription returns the command description.
func (c *Command) GetDescription() string {
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

// Aliases sets command aliases (alias for Alias).
func (c *Command) Aliases(names ...string) *Command {
	return c.Alias(names...)
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

// ArgsRange validates argument count is between min and max.
func (c *Command) ArgsRange(min, max int) *Command {
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
	return c
}

// ExactArgs validates exactly n arguments are provided.
func (c *Command) ExactArgs(n int) *Command {
	c.validators = append(c.validators, func(ctx *Context) error {
		if ctx.NArg() != n {
			return Errorf("requires exactly %d argument(s), got %d", n, ctx.NArg())
		}
		return nil
	})
	return c
}

// NoArgs validates no arguments are provided.
func (c *Command) NoArgs() *Command {
	c.validators = append(c.validators, func(ctx *Context) error {
		if ctx.NArg() > 0 {
			return Errorf("accepts no arguments, got %d", ctx.NArg())
		}
		return nil
	})
	return c
}

// Flag is the interface for typed flags.
type Flag interface {
	GetName() string
	GetShort() string
	GetHelp() string
	GetEnvVar() string
	GetDefault() any
	IsRequired() bool
	IsHidden() bool
	GetEnum() []string
	Validate(value string) error
}

// Arg represents a positional argument.
type Arg struct {
	Name        string
	Description string
	Required    bool
	Default     any
}

// AddArg adds a positional argument to the command.
func (c *Command) AddArg(a *Arg) *Command {
	c.args = append(c.args, a)
	return c
}

// Typed flag implementations

// BoolFlag represents a boolean flag.
type BoolFlag struct {
	Name     string
	Short    string
	Help     string
	Value    bool // default value
	EnvVar   string
	Hidden   bool
	Required bool
}

func (f *BoolFlag) GetName() string     { return f.Name }
func (f *BoolFlag) GetShort() string    { return f.Short }
func (f *BoolFlag) GetHelp() string     { return f.Help }
func (f *BoolFlag) GetEnvVar() string   { return f.EnvVar }
func (f *BoolFlag) GetDefault() any     { return f.Value }
func (f *BoolFlag) IsRequired() bool    { return f.Required }
func (f *BoolFlag) IsHidden() bool      { return f.Hidden }
func (f *BoolFlag) GetEnum() []string   { return nil }
func (f *BoolFlag) Validate(string) error { return nil }

// StringFlag represents a string flag.
type StringFlag struct {
	Name      string
	Short     string
	Help      string
	Value     string // default value
	EnvVar    string
	Required  bool
	Hidden    bool
	Enum      []string
	Validator func(string) error
}

func (f *StringFlag) GetName() string   { return f.Name }
func (f *StringFlag) GetShort() string  { return f.Short }
func (f *StringFlag) GetHelp() string   { return f.Help }
func (f *StringFlag) GetEnvVar() string { return f.EnvVar }
func (f *StringFlag) GetDefault() any   { return f.Value }
func (f *StringFlag) IsRequired() bool  { return f.Required }
func (f *StringFlag) IsHidden() bool    { return f.Hidden }
func (f *StringFlag) GetEnum() []string { return f.Enum }
func (f *StringFlag) Validate(value string) error {
	if f.Validator != nil {
		return f.Validator(value)
	}
	return nil
}

// IntFlag represents an integer flag.
type IntFlag struct {
	Name      string
	Short     string
	Help      string
	Value     int // default value
	EnvVar    string
	Required  bool
	Hidden    bool
	Validator func(int) error
}

func (f *IntFlag) GetName() string   { return f.Name }
func (f *IntFlag) GetShort() string  { return f.Short }
func (f *IntFlag) GetHelp() string   { return f.Help }
func (f *IntFlag) GetEnvVar() string { return f.EnvVar }
func (f *IntFlag) GetDefault() any   { return f.Value }
func (f *IntFlag) IsRequired() bool  { return f.Required }
func (f *IntFlag) IsHidden() bool    { return f.Hidden }
func (f *IntFlag) GetEnum() []string { return nil }
func (f *IntFlag) Validate(value string) error {
	// Validation happens after parsing in parseFlags
	return nil
}

// Float64Flag represents a float64 flag.
type Float64Flag struct {
	Name      string
	Short     string
	Help      string
	Value     float64 // default value
	EnvVar    string
	Required  bool
	Hidden    bool
	Validator func(float64) error
}

func (f *Float64Flag) GetName() string   { return f.Name }
func (f *Float64Flag) GetShort() string  { return f.Short }
func (f *Float64Flag) GetHelp() string   { return f.Help }
func (f *Float64Flag) GetEnvVar() string { return f.EnvVar }
func (f *Float64Flag) GetDefault() any   { return f.Value }
func (f *Float64Flag) IsRequired() bool  { return f.Required }
func (f *Float64Flag) IsHidden() bool    { return f.Hidden }
func (f *Float64Flag) GetEnum() []string { return nil }
func (f *Float64Flag) Validate(string) error { return nil }

// DurationFlag represents a time.Duration flag.
type DurationFlag struct {
	Name     string
	Short    string
	Help     string
	Value    time.Duration // default value
	EnvVar   string
	Required bool
	Hidden   bool
}

func (f *DurationFlag) GetName() string   { return f.Name }
func (f *DurationFlag) GetShort() string  { return f.Short }
func (f *DurationFlag) GetHelp() string   { return f.Help }
func (f *DurationFlag) GetEnvVar() string { return f.EnvVar }
func (f *DurationFlag) GetDefault() any   { return f.Value }
func (f *DurationFlag) IsRequired() bool  { return f.Required }
func (f *DurationFlag) IsHidden() bool    { return f.Hidden }
func (f *DurationFlag) GetEnum() []string { return nil }
func (f *DurationFlag) Validate(string) error { return nil }

// StringSliceFlag represents a string slice flag.
type StringSliceFlag struct {
	Name     string
	Short    string
	Help     string
	Value    []string // default value
	EnvVar   string
	Required bool
	Hidden   bool
}

func (f *StringSliceFlag) GetName() string   { return f.Name }
func (f *StringSliceFlag) GetShort() string  { return f.Short }
func (f *StringSliceFlag) GetHelp() string   { return f.Help }
func (f *StringSliceFlag) GetEnvVar() string { return f.EnvVar }
func (f *StringSliceFlag) GetDefault() any   { return f.Value }
func (f *StringSliceFlag) IsRequired() bool  { return f.Required }
func (f *StringSliceFlag) IsHidden() bool    { return f.Hidden }
func (f *StringSliceFlag) GetEnum() []string { return nil }
func (f *StringSliceFlag) Validate(string) error { return nil }

// IntSliceFlag represents an int slice flag.
type IntSliceFlag struct {
	Name     string
	Short    string
	Help     string
	Value    []int // default value
	EnvVar   string
	Required bool
	Hidden   bool
}

func (f *IntSliceFlag) GetName() string   { return f.Name }
func (f *IntSliceFlag) GetShort() string  { return f.Short }
func (f *IntSliceFlag) GetHelp() string   { return f.Help }
func (f *IntSliceFlag) GetEnvVar() string { return f.EnvVar }
func (f *IntSliceFlag) GetDefault() any   { return f.Value }
func (f *IntSliceFlag) IsRequired() bool  { return f.Required }
func (f *IntSliceFlag) IsHidden() bool    { return f.Hidden }
func (f *IntSliceFlag) GetEnum() []string { return nil }
func (f *IntSliceFlag) Validate(string) error { return nil }

// allFlags returns all flags including global flags from the app.
func (c *Command) allFlags() []Flag {
	var all []Flag
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

	// Track which flags were explicitly set by user
	ctx.setFlags = make(map[string]bool)

	// Initialize with defaults and check env vars
	for _, f := range allFlags {
		name := f.GetName()
		// Check env var first
		if f.GetEnvVar() != "" {
			if val, ok := lookupEnv(f.GetEnvVar()); ok {
				ctx.flags[name] = val
				ctx.setFlags[name] = true
				continue
			}
		}
		// Use default
		if f.GetDefault() != nil {
			ctx.flags[name] = f.GetDefault()
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
				ctx.setFlags[name] = true
			} else if name == "help" {
				return c.showHelp()
			} else {
				// Check if it's a boolean flag or needs a value
				flag := c.findFlag(name)
				if flag == nil {
					return fmt.Errorf("unknown flag: --%s", name)
				}
				if _, ok := flag.GetDefault().(bool); ok {
					ctx.flags[name] = true
					ctx.setFlags[name] = true
				} else if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
					i++
					if err := c.setFlag(ctx, name, args[i]); err != nil {
						return err
					}
					ctx.setFlags[name] = true
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
				if _, ok := flag.GetDefault().(bool); ok {
					ctx.flags[flag.GetName()] = true
					ctx.setFlags[flag.GetName()] = true
				} else if j == len(shorts)-1 && i+1 < len(args) {
					i++
					if err := c.setFlag(ctx, flag.GetName(), args[i]); err != nil {
						return err
					}
					ctx.setFlags[flag.GetName()] = true
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

	// Check required flags
	for _, f := range allFlags {
		if f.IsRequired() && !ctx.setFlags[f.GetName()] {
			return fmt.Errorf("missing required flag: --%s", f.GetName())
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

func (c *Command) findFlag(name string) Flag {
	// Check global flags first
	if c.app != nil {
		for _, f := range c.app.globalFlags {
			if f.GetName() == name {
				return f
			}
		}
	}
	// Then check command flags
	for _, f := range c.flags {
		if f.GetName() == name {
			return f
		}
	}
	return nil
}

func (c *Command) findFlagByShort(short string) Flag {
	// Check global flags first
	if c.app != nil {
		for _, f := range c.app.globalFlags {
			if f.GetShort() == short {
				return f
			}
		}
	}
	// Then check command flags
	for _, f := range c.flags {
		if f.GetShort() == short {
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
	if enum := flag.GetEnum(); len(enum) > 0 {
		valid := false
		for _, e := range enum {
			if e == value {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid value for --%s: %s (allowed: %s)",
				name, value, strings.Join(enum, ", "))
		}
	}

	// Run custom validator
	if err := flag.Validate(value); err != nil {
		return fmt.Errorf("invalid value for --%s: %w", name, err)
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
			if f.IsHidden() {
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
			if f.IsHidden() {
				continue
			}
			writeFlagHelp(&sb, f)
		}
	}

	fmt.Fprint(c.app.stdout, sb.String())
	return &HelpRequested{}
}

// writeFlagHelp writes help text for a single flag.
func writeFlagHelp(sb *strings.Builder, f Flag) {
	flagStr := "  "
	if f.GetShort() != "" {
		flagStr += fmt.Sprintf("-%s, ", f.GetShort())
	} else {
		flagStr += "    "
	}
	flagStr += fmt.Sprintf("--%-12s", f.GetName())
	sb.WriteString(flagStr)
	sb.WriteString(" ")
	sb.WriteString(f.GetHelp())
	def := f.GetDefault()
	if def != nil && def != "" && def != false && def != 0 {
		sb.WriteString(fmt.Sprintf(" (default: %v)", def))
	}
	if f.IsRequired() {
		sb.WriteString(" (required)")
	}
	if enum := f.GetEnum(); len(enum) > 0 {
		sb.WriteString(fmt.Sprintf(" [%s]", strings.Join(enum, "|")))
	}
	sb.WriteString("\n")
}
