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

## Root and group actions

Both the app and command groups can have their own action handlers that run when
invoked without a subcommand. This is useful for commands like `git remote`
which lists remotes when run alone but also has subcommands like `git remote add`.

### App-level action

Set a root handler that runs when the app is invoked without any command:

```go
app := cli.New("cat").
    Description("Concatenate files").
    Args("file?").
    Action(func(ctx *cli.Context) error {
        if ctx.NArg() == 0 {
            // Read from stdin
            return copyStdin(ctx)
        }
        // Read from file
        return readFile(ctx.Arg(0))
    })

// These all work:
// cat              -> reads stdin (root action)
// cat file.txt     -> reads file.txt (root action with arg)
// cat --help       -> shows help
```

### Group-level action

Groups can also have actions that run when the group is invoked without a subcommand:

```go
app := cli.New("myapp").Description("My application")

// Group with both an action and subcommands
users := app.Group("users").
    Description("User management").
    Flags(&cli.BoolFlag{Name: "all", Short: "a", Help: "Show all users"}).
    Action(func(ctx *cli.Context) error {
        // Runs when: myapp users or myapp users -a
        if ctx.Bool("all") {
            return listAllUsers()
        }
        return listActiveUsers()
    })

// Subcommands take precedence over the group action
users.Command("add").
    Description("Add a user").
    Args("name").
    Run(func(ctx *cli.Context) error {
        return addUser(ctx.Arg(0))
    })

users.Command("remove").
    Description("Remove a user").
    Args("name").
    Run(func(ctx *cli.Context) error {
        return removeUser(ctx.Arg(0))
    })

// These all work:
// myapp users           -> lists active users (group action)
// myapp users -a        -> lists all users (group action with flag)
// myapp users add bob   -> adds user bob (subcommand)
// myapp users remove jo -> removes user jo (subcommand)
```

When both an action and subcommands exist, subcommands are matched first. If the
argument doesn't match any subcommand, it's passed to the action as a positional
argument.
