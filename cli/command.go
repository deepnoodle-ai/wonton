package cli

import (
	"fmt"
	"strconv"
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
	name         string
	description  string
	longDesc     string
	usageSummary string
	app          *App
	group        *Group

	// Handler
	handler Handler

	// Flags and args
	flags              []Flag
	args               []*Arg
	omittedGlobalFlags map[string]bool

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
// Each name is a small DSL describing the slot's cardinality:
//
//	"name"     required, exactly one
//	"name?"    optional, zero or one
//	"name..."  variadic, one or more (trailing)
//	"name?..." variadic, zero or more (trailing)
//
// Example:
//
//	cmd.Args("source", "dest?")        // source required, dest optional
//	cmd.Args("src", "dsts...")         // src required, one or more dsts
//	cmd.Args("paths?...")              // zero or more paths
//
// Extra positional arguments beyond the declared slots are rejected at
// parse time unless the last slot is variadic.
//
// Invalid declarations (duplicate names, variadic not last, required
// slot following an optional one) panic at registration — these are
// programmer errors, not user errors.
//
// Access arguments in the handler using ctx.Arg(index) or ctx.Args().
func (c *Command) Args(names ...string) *Command {
	c.args = append(c.args, parseArgSpecs(names)...)
	validateArgSlots(c.args)
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

// UsageSummary sets a short summary that appears next to the usage line in help.
// This is useful for the root command when the app has both a root handler and subcommands.
//
// Example:
//
//	app.Main().
//	    Args("prompt?").
//	    UsageSummary("Start designing (interactive)").
//	    Run(runMain)
func (c *Command) UsageSummary(summary string) *Command {
	c.usageSummary = summary
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
//
// Variadic arguments (collecting all remaining positionals) must be the
// last arg in the list. A variadic arg that is also Required means the
// slot must receive at least one value.
type Arg struct {
	Name        string
	Description string
	Required    bool
	Variadic    bool
	Default     any
}

// AddArg adds a positional argument to the command.
//
// Panics if the resulting arg slot layout is invalid (duplicate name,
// variadic not last, required slot following an optional one).
func (c *Command) AddArg(a *Arg) *Command {
	c.args = append(c.args, a)
	validateArgSlots(c.args)
	return c
}

// parseArgSpec parses a single arg spec string like "name", "name?",
// "name...", or "name?..." / "name...?" into an *Arg. Panics on malformed
// input — these are programmer errors caught at registration.
func parseArgSpec(spec string) *Arg {
	if spec == "" {
		panic("cli: empty arg name")
	}
	name := spec
	variadic := false
	required := true

	// Strip trailing "..." and "?" in either order; both may appear.
	for {
		switch {
		case strings.HasSuffix(name, "..."):
			if variadic {
				panic(fmt.Sprintf("cli: arg %q has multiple variadic markers", spec))
			}
			name = strings.TrimSuffix(name, "...")
			variadic = true
		case strings.HasSuffix(name, "?"):
			if !required {
				panic(fmt.Sprintf("cli: arg %q has multiple optional markers", spec))
			}
			name = strings.TrimSuffix(name, "?")
			required = false
		default:
			goto done
		}
	}
done:
	if name == "" {
		panic(fmt.Sprintf("cli: arg %q has no name", spec))
	}
	if !isValidArgName(name) {
		panic(fmt.Sprintf("cli: arg %q has invalid name %q (use letters, digits, _ or -)", spec, name))
	}
	// A variadic arg without a "?" means "one or more" — the slot is
	// required even though it accepts multiple values.
	return &Arg{
		Name:     name,
		Required: required,
		Variadic: variadic,
	}
}

func parseArgSpecs(specs []string) []*Arg {
	out := make([]*Arg, 0, len(specs))
	for _, s := range specs {
		out = append(out, parseArgSpec(s))
	}
	return out
}

// validateArgSlots enforces the layout invariants for a command's arg list.
// Runs on every Args/AddArg call so bad declarations fail fast at startup.
func validateArgSlots(args []*Arg) {
	seen := make(map[string]bool, len(args))
	sawOptional := false
	for i, a := range args {
		if a == nil {
			panic(fmt.Sprintf("cli: arg slot %d is nil", i))
		}
		if a.Name == "" {
			panic(fmt.Sprintf("cli: arg slot %d has empty name", i))
		}
		if !isValidArgName(a.Name) {
			panic(fmt.Sprintf("cli: invalid arg name %q (use letters, digits, _ or -)", a.Name))
		}
		if seen[a.Name] {
			panic(fmt.Sprintf("cli: duplicate arg name %q", a.Name))
		}
		seen[a.Name] = true
		if a.Variadic && i != len(args)-1 {
			panic(fmt.Sprintf("cli: variadic arg %q must be the last slot", a.Name))
		}
		if sawOptional && a.Required {
			panic(fmt.Sprintf("cli: required arg %q cannot follow an optional arg", a.Name))
		}
		if !a.Required {
			sawOptional = true
		}
	}
}

func isValidArgName(s string) bool {
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '_' || r == '-':
		default:
			return false
		}
	}
	return true
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
	if f.Validator == nil {
		return nil
	}
	n, err := parseInt(value)
	if err != nil {
		return err
	}
	return f.Validator(n)
}

// Int64Flag represents an int64 flag.
type Int64Flag struct {
	Name      string
	Short     string
	Help      string
	Value     int64 // default value
	EnvVar    string
	Required  bool
	Hidden    bool
	Validator func(int64) error
}

func (f *Int64Flag) GetName() string   { return f.Name }
func (f *Int64Flag) GetShort() string  { return f.Short }
func (f *Int64Flag) GetHelp() string   { return f.Help }
func (f *Int64Flag) GetEnvVar() string { return f.EnvVar }
func (f *Int64Flag) GetDefault() any   { return f.Value }
func (f *Int64Flag) IsRequired() bool  { return f.Required }
func (f *Int64Flag) IsHidden() bool    { return f.Hidden }
func (f *Int64Flag) GetEnum() []string { return nil }
func (f *Int64Flag) Validate(value string) error {
	if f.Validator == nil {
		return nil
	}
	v, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil {
		return err
	}
	return f.Validator(v)
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
func (f *Float64Flag) Validate(value string) error {
	if f.Validator == nil {
		return nil
	}
	v, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil {
		return err
	}
	return f.Validator(v)
}

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

// allFlags returns all flags including global flags from the app,
// minus any global flags omitted for this command via OmitGlobalFlag.
func (c *Command) allFlags() []Flag {
	var all []Flag
	if c.app != nil {
		for _, f := range c.app.globalFlags {
			if c.omittedGlobalFlags[f.GetName()] {
				continue
			}
			all = append(all, f)
		}
	}
	all = append(all, c.flags...)
	return all
}

// visibleGlobalFlags returns the app's global flags minus any omitted by
// this command. Used for help rendering.
func (c *Command) visibleGlobalFlags() []Flag {
	if c.app == nil {
		return nil
	}
	if len(c.omittedGlobalFlags) == 0 {
		return c.app.globalFlags
	}
	out := make([]Flag, 0, len(c.app.globalFlags))
	for _, f := range c.app.globalFlags {
		if c.omittedGlobalFlags[f.GetName()] {
			continue
		}
		out = append(out, f)
	}
	return out
}

// OmitGlobalFlag hides one or more app-level global flags from this
// command. Omitted flags are not parsed, not validated (so their
// Required() constraint is skipped), and not shown in the command's help.
//
// Useful when a global flag is required for most commands but a handful
// of commands shouldn't need it:
//
//	app.GlobalFlags(cli.String("api-key", "k").Env("API_KEY").Required())
//	app.Command("health").OmitGlobalFlag("api-key").Run(...)
func (c *Command) OmitGlobalFlag(names ...string) *Command {
	if c.omittedGlobalFlags == nil {
		c.omittedGlobalFlags = make(map[string]bool, len(names))
	}
	for _, n := range names {
		c.omittedGlobalFlags[n] = true
	}
	return c
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
				typed, err := convertFlagValue(f, val)
				if err != nil {
					return c.errorWithHelpHint(fmt.Sprintf("invalid value %q for --%s from environment variable %s: %s",
						val, name, f.GetEnvVar(), err))
				}
				ctx.flags[name] = typed
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
					return c.errorWithHelpHint(fmt.Sprintf("unknown flag: --%s", name))
				}
				if _, ok := flag.GetDefault().(bool); ok {
					ctx.flags[name] = true
					ctx.setFlags[name] = true
				} else if i+1 < len(args) && !looksLikeFlag(args[i+1]) {
					i++
					if err := c.setFlag(ctx, name, args[i]); err != nil {
						return err
					}
					ctx.setFlags[name] = true
				} else {
					return c.errorWithHelpHint(fmt.Sprintf("flag --%s requires a value", name))
				}
			}
		} else if strings.HasPrefix(arg, "-") && len(arg) > 1 {
			// Short flag(s)
			shorts := arg[1:]
			for j, r := range shorts {
				// -h shows help unless a flag explicitly claims the -h short
				if r == 'h' && c.isHelpShort() {
					return c.showHelp()
				}
				flag := c.findFlagByShort(string(r))
				if flag == nil {
					return c.errorWithHelpHint(fmt.Sprintf("unknown flag: -%c", r))
				}
				if _, ok := flag.GetDefault().(bool); ok {
					ctx.flags[flag.GetName()] = true
					ctx.setFlags[flag.GetName()] = true
				} else if j == len(shorts)-1 && i+1 < len(args) && !looksLikeFlag(args[i+1]) {
					i++
					if err := c.setFlag(ctx, flag.GetName(), args[i]); err != nil {
						return err
					}
					ctx.setFlags[flag.GetName()] = true
				} else {
					return c.errorWithHelpHint(fmt.Sprintf("flag -%c requires a value", r))
				}
			}
		} else {
			positional = append(positional, arg)
		}
	}

	// Assign positionals to declared slots. If the last slot is variadic,
	// it absorbs all remaining positionals; otherwise extras are an error.
	hasVariadic := len(c.args) > 0 && c.args[len(c.args)-1].Variadic
	fixedLen := len(c.args)
	if hasVariadic {
		fixedLen--
	}

	for i := 0; i < fixedLen; i++ {
		arg := c.args[i]
		if i < len(positional) {
			ctx.positional = append(ctx.positional, positional[i])
		} else if arg.Required {
			return c.errorWithUsageHint(fmt.Sprintf("missing required argument: %s", arg.Name))
		} else if arg.Default != nil {
			ctx.positional = append(ctx.positional, fmt.Sprint(arg.Default))
		}
	}

	if hasVariadic {
		vararg := c.args[len(c.args)-1]
		var remaining []string
		if len(positional) > fixedLen {
			remaining = positional[fixedLen:]
		}
		if len(remaining) == 0 && vararg.Required {
			return c.errorWithUsageHint(fmt.Sprintf("missing required argument: %s", vararg.Name))
		}
		ctx.positional = append(ctx.positional, remaining...)
	} else if len(positional) > fixedLen {
		return c.errorWithUsageHint(fmt.Sprintf("unexpected argument: %s", positional[fixedLen]))
	}

	// Check required flags
	for _, f := range allFlags {
		if f.IsRequired() && !ctx.setFlags[f.GetName()] {
			return missingFlagError("missing required flag", c, f)
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
	// Check global flags first (skipping omitted ones)
	for _, f := range c.visibleGlobalFlags() {
		if f.GetName() == name {
			return f
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
	// Check global flags first (skipping omitted ones)
	for _, f := range c.visibleGlobalFlags() {
		if f.GetShort() == short {
			return f
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

// isHelpShort reports whether -h should trigger help for this command.
// Returns false if the command or its app's global flags have explicitly
// claimed -h as a short flag name.
func (c *Command) isHelpShort() bool {
	return c.findFlagByShort("h") == nil
}

func (c *Command) setFlag(ctx *Context, name, value string) error {
	flag := c.findFlag(name)
	if flag == nil {
		return c.errorWithHelpHint(fmt.Sprintf("unknown flag: %s", name))
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
			return c.errorWithHelpHint(fmt.Sprintf("invalid value for --%s: %s (allowed: %s)",
				name, value, strings.Join(enum, ", ")))
		}
	}

	// Run custom validator
	if err := flag.Validate(value); err != nil {
		return c.errorWithHelpHint(fmt.Sprintf("invalid value for --%s: %s", name, err))
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
		typed, err := convertFlagValue(flag, value)
		if err != nil {
			return fmt.Errorf("invalid value for --%s: %s", name, err)
		}
		ctx.flags[name] = typed
	}
	return nil
}

func parseInt(s string) (int, error) {
	return strconv.Atoi(strings.TrimSpace(s))
}

// convertFlagValue converts a raw string (e.g. from an environment variable)
// to the flag's value type, as indicated by the type of its default value.
// Slice flags accept comma-separated values.
func convertFlagValue(f Flag, val string) (any, error) {
	switch f.GetDefault().(type) {
	case bool:
		switch strings.ToLower(strings.TrimSpace(val)) {
		case "true", "1", "yes", "on":
			return true, nil
		case "false", "0", "no", "off", "":
			return false, nil
		default:
			return nil, fmt.Errorf("invalid boolean")
		}
	case int:
		return parseInt(val)
	case int64:
		return strconv.ParseInt(strings.TrimSpace(val), 10, 64)
	case float64:
		return strconv.ParseFloat(strings.TrimSpace(val), 64)
	case time.Duration:
		return time.ParseDuration(strings.TrimSpace(val))
	case []string:
		if val == "" {
			return []string(nil), nil
		}
		parts := strings.Split(val, ",")
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		return parts, nil
	case []int:
		if val == "" {
			return []int(nil), nil
		}
		parts := strings.Split(val, ",")
		ints := make([]int, len(parts))
		for i, p := range parts {
			n, err := parseInt(p)
			if err != nil {
				return nil, fmt.Errorf("invalid integer %q", strings.TrimSpace(p))
			}
			ints[i] = n
		}
		return ints, nil
	default:
		return val, nil
	}
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
		if err := writeHelpNewline(c.app.stdout); err != nil {
			return err
		}
		return &HelpRequested{}
	}

	// Fallback to plain text for non-color terminals
	var sb strings.Builder

	// Command name and description
	if c.group != nil && !c.group.flatRouting {
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
	sb.WriteString("Usage:\n")
	sb.WriteString(buildUsageString(c))
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

	// Global Flags (minus any omitted for this command)
	if globals := c.visibleGlobalFlags(); len(globals) > 0 {
		sb.WriteString("Global Flags:\n")
		writeFlagsHelp(&sb, globals)
	}

	if err := writeHelpOutput(c.app.stdout, sb.String()); err != nil {
		return err
	}
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

// missingFlagError builds a user-facing error for a missing required flag,
// mentioning the backing environment variable if one is configured and
// pointing the user at the command's --help for the full flag list.
func missingFlagError(prefix string, c *Command, f Flag) error {
	head := fmt.Sprintf("%s: --%s", prefix, f.GetName())
	if env := f.GetEnvVar(); env != "" {
		head += fmt.Sprintf(" (or set %s environment variable)", env)
	}
	return c.errorWithHelpHint(head)
}

// errorWithHelpHint wraps a head message with a trailing "Run 'X --help'"
// pointer to this command's help. Used for flag-parsing errors where the
// relevant reference material is the flag list.
func (c *Command) errorWithHelpHint(head string) error {
	if c == nil {
		return fmt.Errorf("%s", head)
	}
	return fmt.Errorf("%s\n\nRun '%s --help' to see available flags.", head, c.helpInvocation())
}

// errorWithUsageHint is the positional-argument counterpart to
// errorWithHelpHint. Used when the error is about positional args, so the
// hint points the user at the usage and ARGUMENTS sections rather than
// the flag list.
func (c *Command) errorWithUsageHint(head string) error {
	if c == nil {
		return fmt.Errorf("%s", head)
	}
	return fmt.Errorf("%s\n\nRun '%s --help' to see available arguments.", head, c.helpInvocation())
}

// helpInvocation returns the "app [group] name" prefix used when pointing
// the user at this command's --help.
func (c *Command) helpInvocation() string {
	parts := []string{}
	if c.app != nil {
		parts = append(parts, c.app.name)
	}
	if c.group != nil {
		parts = append(parts, c.group.name)
	}
	if c.name != "" {
		parts = append(parts, c.name)
	}
	return strings.Join(parts, " ")
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
	if !isZeroDefault(def) {
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

// isZeroDefault reports whether a flag default is the zero value for its type
// and should therefore be omitted from help text.
func isZeroDefault(def any) bool {
	switch v := def.(type) {
	case nil:
		return true
	case string:
		return v == ""
	case bool:
		return !v
	case int:
		return v == 0
	case int64:
		return v == 0
	case float64:
		return v == 0
	case time.Duration:
		return v == 0
	case []string:
		return len(v) == 0
	case []int:
		return len(v) == 0
	}
	return false
}
