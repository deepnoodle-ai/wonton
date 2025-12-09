# CLI framework

The `cli` package powers Wonton's command-line framework. It pairs a familiar
flag/argument API with progressive interactivity: the same command can behave as
a fast one-shot tool when piped and upgrade into a full TUI workflow when the
user runs it in a real terminal.

## Key capabilities

- Command registration with groups, aliases, middleware, and validation hooks.
- Struct-based flag declarations (`cli.WithFlags` and `cli.GlobalFlags`) and
  Env/JSON config loading via the `env` package.
- Automatic `help`/`version` commands plus shell-completion generation.
- Progressive handlers: `Command.Interactive` vs `Command.NonInteractive`.
- Rich output helpers: streaming chunks (`ctx.Stream`), progress indicators,
  tables/forms via the `tui` helpers, and JSON event mode when piping.

## Quick example

```go
package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/deepnoodle-ai/wonton/cli"
)

func main() {
	app := cli.New("lab", "Demo of the Wonton CLI framework")
	app.Version("1.0.0")
	app.AddCompletionCommand()
	app.AddGlobalFlag(&cli.Flag{
		Name:        "profile",
		Short:       "p",
		Description: "Active profile name",
		Default:     "default",
	})

	app.Command("greet", "Say hi", cli.WithArgs("name?")).
		AddFlag(&cli.Flag{
			Name:        "loud",
			Short:       "l",
			Description: "Uppercase the greeting",
		}).
		Run(func(ctx *cli.Context) error {
			name := ctx.Arg(0)
			if name == "" {
				name = "friend"
			}

			msg := fmt.Sprintf("Hello, %s (profile=%s)", name, ctx.String("profile"))
			if ctx.Bool("loud") {
				msg = strings.ToUpper(msg)
			}
			ctx.Println(msg)
			return nil
		})

	app.Command("deploy", "Long-running task").
		Interactive(func(ctx *cli.Context) error {
			return ctx.WithProgress("Deploying to "+ctx.String("profile"), func(p *cli.Progress) error {
				for step := 0; step <= 100; step += 10 {
					p.Set(step, fmt.Sprintf("Step %d/10", step/10))
					time.Sleep(200 * time.Millisecond)
				}
				return nil
			})
		}).
		NonInteractive(func(ctx *cli.Context) error {
			return ctx.Stream(func(yield func(string)) error {
				yield("starting deploy...\n")
				yield("done\n")
				return nil
			})
		})

	if err := app.Run(); err != nil {
		// Convert CLI errors to proper exit codes.
		os.Exit(cli.GetExitCode(err))
	}
}
```

Use `go run ./examples/cli_basic` for a more comprehensive walkthrough that
covers groups, nested commands, and config loading.
