package cli

import (
	"context"
	"fmt"
	"io"
	"strconv"

	"github.com/deepnoodle-ai/wonton/color"
)

// Context provides access to the command execution environment.
//
// Context is passed to every command handler and provides:
//   - Parsed flags and arguments
//   - Standard I/O streams
//   - Application and command metadata
//   - Helper methods for output and prompts
//
// Example usage in a handler:
//
//	func handler(ctx *cli.Context) error {
//	    name := ctx.Arg(0)
//	    verbose := ctx.Bool("verbose")
//	    if verbose {
//	        ctx.Info("Processing %s...", name)
//	    }
//	    return nil
//	}
type Context struct {
	context     context.Context
	app         *App
	command     *Command
	args        []string
	positional  []string
	flags       map[string]any
	setFlags    map[string]bool // Flags explicitly set by user (or env var)
	interactive bool

	// I/O
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
}

// Context returns the Go context.
func (c *Context) Context() context.Context {
	return c.context
}

// App returns the application.
func (c *Context) App() *App {
	return c.app
}

// Command returns the current command.
func (c *Context) Command() *Command {
	return c.command
}

// Interactive returns true if running in an interactive terminal.
func (c *Context) Interactive() bool {
	return c.interactive
}

// Stdin returns the stdin reader.
func (c *Context) Stdin() io.Reader {
	return c.stdin
}

// Stdout returns the stdout writer.
func (c *Context) Stdout() io.Writer {
	return c.stdout
}

// Stderr returns the stderr writer.
func (c *Context) Stderr() io.Writer {
	return c.stderr
}

// Args returns all positional arguments.
func (c *Context) Args() []string {
	return c.positional
}

// NArg returns the number of positional arguments.
func (c *Context) NArg() int {
	return len(c.positional)
}

// Arg returns the positional argument at index i, or empty string if not present.
func (c *Context) Arg(i int) string {
	if i >= 0 && i < len(c.positional) {
		return c.positional[i]
	}
	return ""
}

// Flag accessors

// String returns a flag value as a string.
//
// If the flag is not set, returns an empty string. To check if a flag
// was explicitly set, use IsSet.
func (c *Context) String(name string) string {
	if v, ok := c.flags[name]; ok {
		switch val := v.(type) {
		case string:
			return val
		default:
			return fmt.Sprint(val)
		}
	}
	return ""
}

// Int returns a flag value as an int.
func (c *Context) Int(name string) int {
	if v, ok := c.flags[name]; ok {
		switch val := v.(type) {
		case int:
			return val
		case int64:
			return int(val)
		case float64:
			return int(val)
		case string:
			if i, err := strconv.Atoi(val); err == nil {
				return i
			}
		}
	}
	return 0
}

// Int64 returns a flag value as an int64.
func (c *Context) Int64(name string) int64 {
	if v, ok := c.flags[name]; ok {
		switch val := v.(type) {
		case int64:
			return val
		case int:
			return int64(val)
		case float64:
			return int64(val)
		case string:
			if i, err := strconv.ParseInt(val, 10, 64); err == nil {
				return i
			}
		}
	}
	return 0
}

// Float64 returns a flag value as a float64.
func (c *Context) Float64(name string) float64 {
	if v, ok := c.flags[name]; ok {
		switch val := v.(type) {
		case float64:
			return val
		case int:
			return float64(val)
		case int64:
			return float64(val)
		case string:
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				return f
			}
		}
	}
	return 0
}

// Bool returns a flag value as a bool.
func (c *Context) Bool(name string) bool {
	if v, ok := c.flags[name]; ok {
		switch val := v.(type) {
		case bool:
			return val
		case string:
			return val == "true" || val == "1" || val == "yes"
		}
	}
	return false
}

// IsSet returns true if a flag was explicitly set by the user or environment.
//
// This distinguishes between a flag using its default value versus being
// explicitly set to that value:
//
//	if ctx.IsSet("force") {
//	    // User explicitly passed --force
//	}
func (c *Context) IsSet(name string) bool {
	if c.setFlags == nil {
		return false
	}
	return c.setFlags[name]
}

// Output helpers

// Print writes to stdout.
func (c *Context) Print(a ...any) {
	fmt.Fprint(c.stdout, a...)
}

// Printf writes formatted output to stdout.
func (c *Context) Printf(format string, a ...any) {
	fmt.Fprintf(c.stdout, format, a...)
}

// Println writes to stdout with a newline.
func (c *Context) Println(a ...any) {
	fmt.Fprintln(c.stdout, a...)
}

// Error writes to stderr.
func (c *Context) Error(a ...any) {
	fmt.Fprint(c.stderr, a...)
}

// Errorf writes formatted output to stderr.
func (c *Context) Errorf(format string, a ...any) {
	fmt.Fprintf(c.stderr, format, a...)
}

// Errorln writes to stderr with a newline.
func (c *Context) Errorln(a ...any) {
	fmt.Fprintln(c.stderr, a...)
}

// Semantic output helpers with colors

// Success prints a success message to stdout in green (if color enabled).
//
//	ctx.Success("Deployment completed successfully!")
func (c *Context) Success(format string, a ...any) {
	msg := fmt.Sprintf(format, a...)
	if c.app != nil && c.app.colorEnabled {
		msg = color.Green.Apply(msg)
	}
	fmt.Fprintln(c.stdout, msg)
}

// Fail prints an error message to stderr in red (if color enabled).
//
//	ctx.Fail("Deployment failed: %v", err)
func (c *Context) Fail(format string, a ...any) {
	msg := fmt.Sprintf(format, a...)
	if c.app != nil && c.app.colorEnabled {
		msg = color.Red.Apply(msg)
	}
	fmt.Fprintln(c.stderr, msg)
}

// Warn prints a warning message to stderr in yellow (if color enabled).
//
//	ctx.Warn("Configuration file not found, using defaults")
func (c *Context) Warn(format string, a ...any) {
	msg := fmt.Sprintf(format, a...)
	if c.app != nil && c.app.colorEnabled {
		msg = color.Yellow.Apply(msg)
	}
	fmt.Fprintln(c.stderr, msg)
}

// Info prints an informational message to stdout in cyan (if color enabled).
//
//	ctx.Info("Processing %d files...", count)
func (c *Context) Info(format string, a ...any) {
	msg := fmt.Sprintf(format, a...)
	if c.app != nil && c.app.colorEnabled {
		msg = color.Cyan.Apply(msg)
	}
	fmt.Fprintln(c.stdout, msg)
}
