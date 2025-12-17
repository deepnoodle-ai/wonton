package cli

import (
	"bufio"
	"fmt"
	"strings"
)

// This file provides common middleware implementations for CLI commands.

// Recover returns middleware that recovers from panics in handlers.
//
// The panic is converted to an error and returned normally:
//
//	app.Use(cli.Recover())
func Recover() Middleware {
	return func(next Handler) Handler {
		return func(ctx *Context) (err error) {
			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("panic: %v", r)
				}
			}()
			return next(ctx)
		}
	}
}

// RequireFlags returns middleware that requires certain flags to be set.
//
// This validates that specified flags were explicitly provided:
//
//	cmd.Use(cli.RequireFlags("config", "api-key"))
func RequireFlags(names ...string) Middleware {
	return func(next Handler) Handler {
		return func(ctx *Context) error {
			for _, name := range names {
				if !ctx.IsSet(name) {
					return fmt.Errorf("required flag not set: --%s", name)
				}
			}
			return next(ctx)
		}
	}
}

// Confirm returns middleware that prompts for user confirmation before proceeding.
//
// Only works in interactive mode. In non-interactive mode, returns an error:
//
//	cmd.Use(cli.Confirm("Are you sure you want to delete everything?"))
func Confirm(message string) Middleware {
	return func(next Handler) Handler {
		return func(ctx *Context) error {
			if !ctx.Interactive() {
				return fmt.Errorf("confirmation required but running non-interactively")
			}

			fmt.Fprintf(ctx.Stdout(), "%s [y/N]: ", message)
			reader := bufio.NewReader(ctx.Stdin())
			response, err := reader.ReadString('\n')
			if err != nil {
				return err
			}

			response = strings.ToLower(strings.TrimSpace(response))
			if response != "y" && response != "yes" {
				return fmt.Errorf("operation cancelled")
			}

			return next(ctx)
		}
	}
}

// Before returns middleware that runs a function before the command handler.
//
// If the function returns an error, the handler is not executed:
//
//	cmd.Use(cli.Before(func(ctx *cli.Context) error {
//	    return validateConfig(ctx.String("config"))
//	}))
func Before(fn func(*Context) error) Middleware {
	return func(next Handler) Handler {
		return func(ctx *Context) error {
			if err := fn(ctx); err != nil {
				return err
			}
			return next(ctx)
		}
	}
}

// After returns middleware that runs a function after the command handler.
//
// The after function runs regardless of whether the handler succeeded:
//
//	cmd.Use(cli.After(func(ctx *cli.Context) error {
//	    return cleanup()
//	}))
func After(fn func(*Context) error) Middleware {
	return func(next Handler) Handler {
		return func(ctx *Context) error {
			err := next(ctx)
			if afterErr := fn(ctx); afterErr != nil && err == nil {
				return afterErr
			}
			return err
		}
	}
}
