package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/deepnoodle-ai/wonton/tui"
)

// Handler is the function type for command handlers.
//
// Handlers receive a Context containing parsed flags, arguments, and
// I/O streams. They should return nil on success or an error on failure:
//
//	func myHandler(ctx *cli.Context) error {
//	    name := ctx.Arg(0)
//	    verbose := ctx.Bool("verbose")
//	    ctx.Printf("Processing %s (verbose=%v)\n", name, verbose)
//	    return nil
//	}
type Handler func(*Context) error

// Middleware wraps a handler to add behavior.
//
// Middleware can run code before and after the handler executes, modify
// the context, or intercept errors:
//
//	func loggingMiddleware(next cli.Handler) cli.Handler {
//	    return func(ctx *cli.Context) error {
//	        start := time.Now()
//	        err := next(ctx)
//	        duration := time.Since(start)
//	        log.Printf("Command %s took %v", ctx.Command().Name(), duration)
//	        return err
//	    }
//	}
type Middleware func(Handler) Handler

// Command represents a CLI command with its configuration and handler.
//
// Commands are created through App.Command() or Group.Command() and configured
// using the fluent builder pattern:
//
//	app.Command("deploy").
//	    Description("Deploy the application").
//	    Args("environment").
//	    Flags(
//	        cli.Bool("force", "f").Help("Force deployment"),
//	        cli.String("version", "v").Help("Version to deploy"),
//	    ).
//	    Run(func(ctx *cli.Context) error {
//	        env := ctx.Arg(0)
//	        force := ctx.Bool("force")
//	        // Perform deployment
//	        return nil
//	    })
type Command struct {
	name        string
	description string
	longDesc    string
	app         *App
	group       *Group

	// Handler
	handler Handler

	// Flags and args
	flags []Flag
	args  []*Arg

	// Options
	middleware  []Middleware
	hidden      bool
	deprecated  string
	aliases     []string
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

// Args sets the positional argument names for the command.
//
// Arguments are processed in order. Append "?" to make an argument optional:
//
//	cmd.Args("source", "dest?")  // source required, dest optional
//
// Access arguments in the handler using ctx.Arg(index) or ctx.Args().
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
//
// Use the flag builder functions to create type-safe flags:
//
//	cmd.Flags(
//	    cli.String("name", "n").Required().Help("User name"),
//	    cli.Int("port", "p").Default(8080).Help("Port number"),
//	    cli.Bool("verbose", "v").Help("Verbose output"),
//	)
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

// Run sets the command handler that executes when the command is invoked.
//
// The handler receives a Context with parsed flags and arguments:
//
//	cmd.Run(func(ctx *cli.Context) error {
//	    name := ctx.String("name")
//	    ctx.Printf("Hello, %s!\n", name)
//	    return nil
//	})
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

// Interactive sets a handler that runs when the command is executed in a TTY.
//
// Use this for rich interactive experiences with prompts and TUI components:
//
//	cmd.Interactive(func(ctx *cli.Context) error {
//	    name, err := ctx.Input("Enter your name: ")
//	    if err != nil {
//	        return err
//	    }
//	    ctx.Success("Welcome, %s!", name)
//	    return nil
//	})
func (c *Command) Interactive(h Handler) *Command {
	c.interactive = h
	return c
}

// NonInteractive sets a handler that runs when stdin/stdout are not TTYs.
//
// Use this for piped or automated execution where interactivity isn't available:
//
//	cmd.NonInteractive(func(ctx *cli.Context) error {
//	    if !ctx.IsSet("name") {
//	        return cli.Error("--name is required in non-interactive mode")
//	    }
//	    // Process with flags only
//	    return nil
//	})
func (c *Command) NonInteractive(h Handler) *Command {
	c.nonInteract = h
	return c
}

// Validate adds a validation function that runs before the handler.
//
// Validators can check arguments and flags, returning an error if invalid:
//
//	cmd.Validate(func(ctx *cli.Context) error {
//	    if ctx.Int("port") < 1024 {
//	        return cli.Error("port must be >= 1024")
//	    }
//	    return nil
//	})
func (c *Command) Validate(v func(*Context) error) *Command {
	c.validators = append(c.validators, v)
	return c
}

// ArgsRange validates that the number of arguments is between min and max.
//
// Pass -1 for max to allow unlimited arguments above min:
//
//	cmd.ArgsRange(1, 3)   // Require 1-3 arguments
//	cmd.ArgsRange(2, -1)  // Require at least 2 arguments
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

// ExactArgs validates that exactly n arguments are provided.
//
//	cmd.ExactArgs(2)  // Require exactly 2 arguments
func (c *Command) ExactArgs(n int) *Command {
	c.validators = append(c.validators, func(ctx *Context) error {
		if ctx.NArg() != n {
			return Errorf("requires exactly %d argument(s), got %d", n, ctx.NArg())
		}
		return nil
	})
	return c
}

// NoArgs validates that no arguments are provided.
//
// Useful for commands that only accept flags:
//
//	cmd.NoArgs()  // Reject any positional arguments
func (c *Command) NoArgs() *Command {
	c.validators = append(c.validators, func(ctx *Context) error {
		if ctx.NArg() > 0 {
			return Errorf("accepts no arguments, got %d", ctx.NArg())
		}
		return nil
	})
	return c
}

// Flag is the interface implemented by all typed flag types.
//
// The cli package provides concrete implementations like BoolFlag, StringFlag,
// IntFlag, etc. Most users will use the flag builder functions (Bool, String, Int)
// rather than implementing this interface directly.
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

// Arg represents a positional argument configuration.
//
// Arguments are typically defined using the Args method:
//
//	cmd.Args("source", "destination")
//
// For more control, use AddArg with an explicit Arg struct.
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

func (f *BoolFlag) GetName() string       { return f.Name }
func (f *BoolFlag) GetShort() string      { return f.Short }
func (f *BoolFlag) GetHelp() string       { return f.Help }
func (f *BoolFlag) GetEnvVar() string     { return f.EnvVar }
func (f *BoolFlag) GetDefault() any       { return f.Value }
func (f *BoolFlag) IsRequired() bool      { return f.Required }
func (f *BoolFlag) IsHidden() bool        { return f.Hidden }
func (f *BoolFlag) GetEnum() []string     { return nil }
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

func (f *Float64Flag) GetName() string       { return f.Name }
func (f *Float64Flag) GetShort() string      { return f.Short }
func (f *Float64Flag) GetHelp() string       { return f.Help }
func (f *Float64Flag) GetEnvVar() string     { return f.EnvVar }
func (f *Float64Flag) GetDefault() any       { return f.Value }
func (f *Float64Flag) IsRequired() bool      { return f.Required }
func (f *Float64Flag) IsHidden() bool        { return f.Hidden }
func (f *Float64Flag) GetEnum() []string     { return nil }
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

func (f *DurationFlag) GetName() string       { return f.Name }
func (f *DurationFlag) GetShort() string      { return f.Short }
func (f *DurationFlag) GetHelp() string       { return f.Help }
func (f *DurationFlag) GetEnvVar() string     { return f.EnvVar }
func (f *DurationFlag) GetDefault() any       { return f.Value }
func (f *DurationFlag) IsRequired() bool      { return f.Required }
func (f *DurationFlag) IsHidden() bool        { return f.Hidden }
func (f *DurationFlag) GetEnum() []string     { return nil }
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

func (f *StringSliceFlag) GetName() string       { return f.Name }
func (f *StringSliceFlag) GetShort() string      { return f.Short }
func (f *StringSliceFlag) GetHelp() string       { return f.Help }
func (f *StringSliceFlag) GetEnvVar() string     { return f.EnvVar }
func (f *StringSliceFlag) GetDefault() any       { return f.Value }
func (f *StringSliceFlag) IsRequired() bool      { return f.Required }
func (f *StringSliceFlag) IsHidden() bool        { return f.Hidden }
func (f *StringSliceFlag) GetEnum() []string     { return nil }
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

func (f *IntSliceFlag) GetName() string       { return f.Name }
func (f *IntSliceFlag) GetShort() string      { return f.Short }
func (f *IntSliceFlag) GetHelp() string       { return f.Help }
func (f *IntSliceFlag) GetEnvVar() string     { return f.EnvVar }
func (f *IntSliceFlag) GetDefault() any       { return f.Value }
func (f *IntSliceFlag) IsRequired() bool      { return f.Required }
func (f *IntSliceFlag) IsHidden() bool        { return f.Hidden }
func (f *IntSliceFlag) GetEnum() []string     { return nil }
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
				} else if i+1 < len(args) && !c.looksLikeFlag(args[i+1]) {
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
				// -h is always help
				if r == 'h' {
					return c.showHelp()
				}
				flag := c.findFlagByShort(string(r))
				if flag == nil {
					return fmt.Errorf("unknown flag: -%c", r)
				}
				if _, ok := flag.GetDefault().(bool); ok {
					ctx.flags[flag.GetName()] = true
					ctx.setFlags[flag.GetName()] = true
				} else if j == len(shorts)-1 && i+1 < len(args) && !c.looksLikeFlag(args[i+1]) {
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

// looksLikeFlag returns true if the string looks like a flag rather than a value.
// This allows values like "-1" (negative numbers) while still treating "-v" as a flag.
func (c *Command) looksLikeFlag(s string) bool {
	if !strings.HasPrefix(s, "-") {
		return false
	}
	if len(s) == 1 {
		return false // just "-"
	}
	// Check if it could be a negative number
	if len(s) >= 2 {
		// -N or -.N where N is a digit
		second := s[1]
		if second >= '0' && second <= '9' {
			return false // looks like negative number
		}
		if second == '.' && len(s) > 2 {
			return false // looks like negative decimal
		}
	}
	// Otherwise it's probably a flag
	return true
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

	// Handle slice flags by accumulating values
	// On first user-provided value, clear defaults and start fresh
	switch flag.GetDefault().(type) {
	case []string:
		var existing []string
		if ctx.setFlags[name] {
			// Already set by user, accumulate
			existing, _ = ctx.flags[name].([]string)
		}
		// Otherwise start with empty slice (replacing defaults)
		ctx.flags[name] = append(existing, value)
	case []int:
		var existing []int
		if ctx.setFlags[name] {
			// Already set by user, accumulate
			existing, _ = ctx.flags[name].([]int)
		}
		// Otherwise start with empty slice (replacing defaults)
		intVal, err := parseInt(value)
		if err != nil {
			return fmt.Errorf("invalid integer for --%s: %s", name, value)
		}
		ctx.flags[name] = append(existing, intVal)
	default:
		ctx.flags[name] = value
	}
	return nil
}

func parseInt(s string) (int, error) {
	var i int
	_, err := fmt.Sscanf(s, "%d", &i)
	return i, err
}

func (c *Command) showHelp() error {
	// For the root command (name == ""), use the app's help which shows
	// full application help including the root command's flags
	if c.name == "" {
		return c.app.showHelp()
	}

	if c.app.colorEnabled {
		// Use the styled tui-based help
		view := c.renderCommandHelp()
		if err := tui.Fprint(c.app.stdout, view); err != nil {
			return err
		}
		return &HelpRequested{}
	}

	// Fallback to plain text for non-color terminals
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
		writeFlagsHelp(&sb, c.flags)
		sb.WriteString("\n")
	}

	// Global Flags
	if c.app != nil && len(c.app.globalFlags) > 0 {
		sb.WriteString("Global Flags:\n")
		writeFlagsHelp(&sb, c.app.globalFlags)
	}

	fmt.Fprint(c.app.stdout, sb.String())
	return &HelpRequested{}
}

// writeFlagsHelp writes help text for a slice of flags with proper alignment.
func writeFlagsHelp(sb *strings.Builder, flags []Flag) {
	// Calculate max flag name length for alignment
	maxNameLen := 0
	for _, f := range flags {
		if f.IsHidden() {
			continue
		}
		if len(f.GetName()) > maxNameLen {
			maxNameLen = len(f.GetName())
		}
	}

	for _, f := range flags {
		if f.IsHidden() {
			continue
		}
		writeFlagHelp(sb, f, maxNameLen)
	}
}

// writeFlagHelp writes help text for a single flag with the given name width.
func writeFlagHelp(sb *strings.Builder, f Flag, nameWidth int) {
	flagStr := "  "
	if f.GetShort() != "" {
		flagStr += fmt.Sprintf("-%s, ", f.GetShort())
	} else {
		flagStr += "    "
	}
	flagStr += fmt.Sprintf("--%-*s", nameWidth, f.GetName())
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
