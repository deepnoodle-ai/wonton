package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
)

// Common middleware implementations

// Logger returns middleware that logs command execution.
func Logger() Middleware {
	return func(next Handler) Handler {
		return func(ctx *Context) error {
			start := time.Now()
			fmt.Fprintf(ctx.Stderr(), "[%s] Running: %s\n",
				start.Format("15:04:05"), ctx.Command().Name())

			err := next(ctx)

			elapsed := time.Since(start)
			if err != nil {
				fmt.Fprintf(ctx.Stderr(), "[%s] Failed: %s (%v) - %v\n",
					time.Now().Format("15:04:05"), ctx.Command().Name(), elapsed, err)
			} else {
				fmt.Fprintf(ctx.Stderr(), "[%s] Done: %s (%v)\n",
					time.Now().Format("15:04:05"), ctx.Command().Name(), elapsed)
			}
			return err
		}
	}
}

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

// RequireInteractive returns middleware that requires an interactive terminal.
func RequireInteractive() Middleware {
	return func(next Handler) Handler {
		return func(ctx *Context) error {
			if !ctx.Interactive() {
				return fmt.Errorf("this command requires an interactive terminal")
			}
			return next(ctx)
		}
	}
}

// WithTimeout returns middleware that sets a context timeout.
func WithTimeout(d time.Duration) Middleware {
	return func(next Handler) Handler {
		return func(ctx *Context) error {
			// Note: This would need context.WithTimeout to be useful
			// For now, just pass through
			return next(ctx)
		}
	}
}

// Auth returns middleware that ensures authentication.
// The authFn should return an API key or error.
func Auth(authFn func(*Context) (string, error)) Middleware {
	return func(next Handler) Handler {
		return func(ctx *Context) error {
			key, err := authFn(ctx)
			if err != nil {
				return fmt.Errorf("authentication failed: %w", err)
			}
			if key == "" {
				return fmt.Errorf("authentication required")
			}
			// Store the key in context for the handler
			ctx.flags["_auth_key"] = key
			return next(ctx)
		}
	}
}

// EnvAuth returns middleware that requires an environment variable for auth.
func EnvAuth(envVar string) Middleware {
	return Auth(func(ctx *Context) (string, error) {
		if key := os.Getenv(envVar); key != "" {
			return key, nil
		}
		return "", fmt.Errorf("environment variable %s not set", envVar)
	})
}
