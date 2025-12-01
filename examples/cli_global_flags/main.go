// Example: Global Flags and Configuration
//
// Demonstrates global flags and config management:
// - Global flags available to all commands
// - Config file loading (YAML)
// - Environment variable fallbacks
// - Middleware for common operations
//
// Run with:
//   go run examples/cli_global_flags/main.go --help
//   go run examples/cli_global_flags/main.go run
//   go run examples/cli_global_flags/main.go -v --config myconfig.yaml run
//   go run examples/cli_global_flags/main.go --output json list
//   go run examples/cli_global_flags/main.go -v users list
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/deepnoodle-ai/gooey/cli"
)

func main() {
	app := cli.New("globalflags", "Demonstrates global flags and configuration")
	app.Version("1.0.0")

	// Add global flags available to ALL commands
	app.AddGlobalFlag(&cli.Flag{
		Name:        "verbose",
		Short:       "v",
		Description: "Enable verbose output",
		Default:     false,
	})
	app.AddGlobalFlag(&cli.Flag{
		Name:        "config",
		Short:       "c",
		Description: "Config file path",
		Default:     "",
		EnvVar:      "MYAPP_CONFIG",
	})
	app.AddGlobalFlag(&cli.Flag{
		Name:        "output",
		Short:       "o",
		Description: "Output format",
		Default:     "text",
		Enum:        []string{"text", "json", "yaml"},
	})
	app.AddGlobalFlag(&cli.Flag{
		Name:        "quiet",
		Short:       "q",
		Description: "Suppress non-essential output",
		Default:     false,
	})

	// Global middleware that runs before every command
	app.Use(
		// Verbose logging middleware
		func(next cli.Handler) cli.Handler {
			return func(ctx *cli.Context) error {
				if ctx.Bool("verbose") {
					ctx.Errorf("[%s] Starting command: %s\n",
						time.Now().Format("15:04:05"),
						ctx.Command().Name())
				}

				err := next(ctx)

				if ctx.Bool("verbose") {
					if err != nil {
						ctx.Errorf("[%s] Command failed: %v\n",
							time.Now().Format("15:04:05"), err)
					} else {
						ctx.Errorf("[%s] Command completed\n",
							time.Now().Format("15:04:05"))
					}
				}

				return err
			}
		},

		// Config loading middleware
		cli.Before(func(ctx *cli.Context) error {
			configPath := ctx.String("config")
			if configPath != "" && ctx.Bool("verbose") {
				ctx.Errorf("[config] Loading from: %s\n", configPath)
			}
			return nil
		}),
	)

	// Simple command that uses global flags
	app.Command("run", "Run the main operation").
		Run(func(ctx *cli.Context) error {
			verbose := ctx.Bool("verbose")
			output := ctx.String("output")
			quiet := ctx.Bool("quiet")

			if !quiet {
				ctx.Printf("Running with output format: %s\n", output)
			}

			if verbose {
				ctx.Println("Verbose mode enabled - showing extra details")
			}

			// Simulate some work
			if verbose {
				ctx.Println("Step 1: Initializing...")
				ctx.Println("Step 2: Processing...")
				ctx.Println("Step 3: Finalizing...")
			}

			switch output {
			case "json":
				ctx.Println(`{"status": "success", "message": "Operation completed"}`)
			case "yaml":
				ctx.Println("status: success")
				ctx.Println("message: Operation completed")
			default:
				ctx.Println("Operation completed successfully!")
			}

			return nil
		})

	// List command
	app.Command("list", "List items").
		Run(func(ctx *cli.Context) error {
			output := ctx.String("output")
			verbose := ctx.Bool("verbose")

			items := []map[string]string{
				{"name": "item1", "status": "active"},
				{"name": "item2", "status": "inactive"},
				{"name": "item3", "status": "active"},
			}

			switch output {
			case "json":
				ctx.Println("[")
				for i, item := range items {
					comma := ","
					if i == len(items)-1 {
						comma = ""
					}
					ctx.Printf(`  {"name": "%s", "status": "%s"}%s`+"\n",
						item["name"], item["status"], comma)
				}
				ctx.Println("]")
			case "yaml":
				ctx.Println("items:")
				for _, item := range items {
					ctx.Printf("  - name: %s\n", item["name"])
					ctx.Printf("    status: %s\n", item["status"])
				}
			default:
				ctx.Println("Items:")
				for _, item := range items {
					if verbose {
						ctx.Printf("  - %s (status: %s)\n", item["name"], item["status"])
					} else {
						ctx.Printf("  - %s\n", item["name"])
					}
				}
			}

			return nil
		})

	// Command group that also uses global flags
	users := app.Group("users", "User management")

	users.Command("list", "List users").
		Run(func(ctx *cli.Context) error {
			output := ctx.String("output")

			switch output {
			case "json":
				ctx.Println(`[{"name": "alice"}, {"name": "bob"}]`)
			default:
				ctx.Println("Users: alice, bob")
			}
			return nil
		})

	users.Command("get", "Get user details", cli.WithArgs("username")).
		Run(func(ctx *cli.Context) error {
			username := ctx.Arg(0)
			output := ctx.String("output")
			verbose := ctx.Bool("verbose")

			if verbose {
				ctx.Errorf("[debug] Looking up user: %s\n", username)
			}

			switch output {
			case "json":
				ctx.Printf(`{"name": "%s", "email": "%s@example.com"}`+"\n", username, username)
			default:
				ctx.Printf("User: %s\n", username)
				ctx.Printf("Email: %s@example.com\n", username)
			}
			return nil
		})

	// Show how to check global flags in any command
	app.Command("debug", "Show current configuration").
		Run(func(ctx *cli.Context) error {
			ctx.Println("Current Configuration:")
			ctx.Printf("  --verbose: %v\n", ctx.Bool("verbose"))
			ctx.Printf("  --config:  %s\n", ctx.String("config"))
			ctx.Printf("  --output:  %s\n", ctx.String("output"))
			ctx.Printf("  --quiet:   %v\n", ctx.Bool("quiet"))
			ctx.Printf("  Interactive: %v\n", ctx.Interactive())
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
