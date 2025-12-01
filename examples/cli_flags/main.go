// Example: Flags and Validation
//
// Demonstrates advanced flag handling and argument validation:
// - Struct-based flag definitions with generics
// - Environment variable fallbacks
// - Enum validation
// - Argument count validation (exact, range, none)
// - Custom validation functions
//
// Run with:
//
//	go run examples/cli_flags/main.go run "Hello world"
//	go run examples/cli_flags/main.go run --model gpt-4 -t 0.9 "Hello"
//	go run examples/cli_flags/main.go run --format json "Test"
//	MYAPP_MODEL=claude-opus go run examples/cli_flags/main.go run "Test"
//	go run examples/cli_flags/main.go add a b c
//	go run examples/cli_flags/main.go pair x y
//	go run examples/cli_flags/main.go status
package main

import (
	"fmt"
	"os"

	"github.com/deepnoodle-ai/wonton/cli"
)

// RunFlags demonstrates struct-based flag definitions.
// Tags supported: flag, default, help, env, enum, required, hidden
type RunFlags struct {
	Model       string  `flag:"model,m" default:"claude-sonnet" help:"Model to use" env:"MYAPP_MODEL"`
	Temperature float64 `flag:"temperature,t" default:"0.7" help:"Temperature for generation"`
	MaxTokens   int     `flag:"max-tokens" default:"1000" help:"Maximum tokens"`
	Stream      bool    `flag:"stream,s" help:"Stream output"`
	Format      string  `flag:"format,f" enum:"text,json,markdown" default:"text" help:"Output format"`
}

func main() {
	app := cli.New("flagdemo", "Demonstrates flags and validation")
	app.Version("1.0.0")

	// Command using struct-based flags
	runCmd := app.Command("run", "Run a prompt", cli.WithArgs("prompt"))
	cli.ParseFlags[RunFlags](runCmd)
	runCmd.Run(func(ctx *cli.Context) error {
		// Bind flags to struct
		flags, err := cli.BindFlags[RunFlags](ctx)
		if err != nil {
			return err
		}

		ctx.Println("Configuration:")
		ctx.Printf("  Model:       %s\n", flags.Model)
		ctx.Printf("  Temperature: %.2f\n", flags.Temperature)
		ctx.Printf("  Max Tokens:  %d\n", flags.MaxTokens)
		ctx.Printf("  Stream:      %v\n", flags.Stream)
		ctx.Printf("  Format:      %s\n", flags.Format)
		ctx.Printf("  Prompt:      %s\n", ctx.Arg(0))

		return nil
	})

	// Command with required flag
	app.Command("deploy", "Deploy to environment").
		AddFlag(&cli.Flag{
			Name:        "environment",
			Short:       "e",
			Description: "Target environment",
			Required:    true,
			Enum:        []string{"dev", "staging", "prod"},
		}).
		AddFlag(&cli.Flag{
			Name:        "force",
			Short:       "f",
			Description: "Force deployment",
			Default:     false,
		}).
		Run(func(ctx *cli.Context) error {
			env := ctx.String("environment")
			force := ctx.Bool("force")
			ctx.Printf("Deploying to %s (force=%v)\n", env, force)
			return nil
		})

	// Command with argument range validation (1-3 args)
	app.Command("add", "Add items (1-3 items)", cli.WithArgsRange(1, 3)).
		Run(func(ctx *cli.Context) error {
			ctx.Printf("Adding %d items: %v\n", ctx.NArg(), ctx.Args())
			return nil
		})

	// Command with exact args validation
	app.Command("pair", "Pair two items", cli.WithExactArgs(2)).
		Run(func(ctx *cli.Context) error {
			ctx.Printf("Pairing: %s <-> %s\n", ctx.Arg(0), ctx.Arg(1))
			return nil
		})

	// Command with no args validation
	app.Command("status", "Show status (no arguments)", cli.WithNoArgs()).
		Run(func(ctx *cli.Context) error {
			ctx.Println("Status: OK")
			ctx.Println("Version: 1.0.0")
			ctx.Println("Uptime: 42 hours")
			return nil
		})

	// Command with custom validation
	app.Command("set", "Set a value",
		cli.WithArgs("key", "value"),
		cli.WithValidation(func(ctx *cli.Context) error {
			key := ctx.Arg(0)
			if key == "" {
				return cli.Error("key cannot be empty")
			}
			if len(key) > 50 {
				return cli.Error("key too long").
					Hint("Keys must be 50 characters or less").
					Detail("Got %d characters", len(key))
			}
			// Reserved keys
			reserved := []string{"system", "root", "admin"}
			for _, r := range reserved {
				if key == r {
					return cli.Errorf("'%s' is a reserved key", key).
						Hint("Choose a different key name")
				}
			}
			return nil
		}),
	).Run(func(ctx *cli.Context) error {
		ctx.Printf("Set %s = %s\n", ctx.Arg(0), ctx.Arg(1))
		return nil
	})

	// Command with env var for sensitive data
	app.Command("auth", "Authenticate").
		AddFlag(&cli.Flag{
			Name:        "token",
			Description: "API token",
			EnvVar:      "MYAPP_TOKEN",
			Required:    true,
		}).
		Run(func(ctx *cli.Context) error {
			token := ctx.String("token")
			// Don't print the actual token!
			ctx.Printf("Authenticating with token: %s***\n", token[:min(3, len(token))])
			return nil
		})

	if err := app.Run(); err != nil {
		if cli.IsHelpRequested(err) {
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(cli.GetExitCode(err))
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
