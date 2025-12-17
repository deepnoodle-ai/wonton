# CLI framework

The `cli` package powers Wonton's command-line framework. It pairs a familiar
flag/argument API with progressive interactivity: the same command can behave as
a fast one-shot tool when piped and upgrade into a full TUI workflow when the
user runs it in a real terminal.

## Key capabilities

- Command registration with groups, aliases, middleware, and validation hooks.
- Struct-based flag declarations with environment variable support.
- Automatic `help`/`version` commands plus shell-completion generation.
- Progressive handlers: `Command.Interactive` vs `Command.NonInteractive`.
- Semantic output helpers with automatic color support.
- Interactive prompts (select, input, confirm) via the `tui` package.

## Quick example

```go
package main

import (
	"os"

	"github.com/deepnoodle-ai/wonton/cli"
)

func main() {
	app := cli.New("deploy").Description("Deployment tool")
	app.Version("1.0.0")

	app.Command("run").
		Description("Deploy to environment").
		Args("env?").
		Run(func(ctx *cli.Context) error {
			env := ctx.Arg(0)

			// Interactive prompt if no env provided
			if env == "" {
				var err error
				env, err = ctx.SelectString("Choose environment:", "dev", "staging", "prod")
				if err != nil {
					return err
				}
			}

			// Confirmation for production
			if env == "prod" {
				ok, err := ctx.Confirm("Deploy to production?")
				if err != nil || !ok {
					ctx.Warn("Deployment cancelled")
					return nil
				}
			}

			ctx.Info("Deploying to %s...", env)
			// ... deployment logic ...
			ctx.Success("Deployed to %s", env)
			return nil
		})

	if err := app.Run(); err != nil {
		os.Exit(cli.GetExitCode(err))
	}
}
```

## Output helpers

The context provides semantic output methods with automatic color support:

```go
ctx.Success("Deployed to %s", env)   // Green, stdout
ctx.Info("Processing %d items", n)   // Cyan, stdout
ctx.Warn("Disk usage at %d%%", pct)  // Yellow, stderr
ctx.Fail("Connection failed: %v", e) // Red, stderr
```

## Interactive prompts

For interactive commands, use the built-in prompt helpers:

```go
// Selection prompt (returns index)
idx, err := ctx.Select("Choose:", "Option A", "Option B", "Option C")

// Selection prompt (returns string)
choice, err := ctx.SelectString("Environment:", "dev", "staging", "prod")

// Text input
name, err := ctx.Input("Enter name:")

// Yes/No confirmation
ok, err := ctx.Confirm("Continue?")
```

Prompts require an interactive terminal and return an error if run non-interactively.
