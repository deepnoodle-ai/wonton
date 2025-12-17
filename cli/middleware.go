package cli

import (
	"bufio"
	"fmt"
	"strings"
)

// Common middleware implementations

// Recover returns middleware that recovers from panics.
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

// Confirm returns middleware that prompts for confirmation.
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

// Before returns middleware that runs before the command.
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

// After returns middleware that runs after the command.
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
