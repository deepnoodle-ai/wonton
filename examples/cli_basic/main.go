// Example: Basic CLI
//
// Demonstrates the fundamentals of the Wonton CLI framework:
// - Creating an app with commands
// - Adding flags and arguments
// - Command groups for organization
// - Command aliases
//
// Run with:
//
//	go run examples/cli_basic/main.go --help
//	go run examples/cli_basic/main.go greet World
//	go run examples/cli_basic/main.go greet --loud World
//	go run examples/cli_basic/main.go users:list
//	go run examples/cli_basic/main.go users:create alice
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/deepnoodle-ai/wonton/cli"
)

func main() {
	// Create a new CLI application
	app := cli.New("myapp").
		Description("A demonstration CLI application").
		Version("1.0.0")

	// Add shell completion support
	app.AddCompletionCommand()

	// Simple command with an argument
	app.Command("greet").
		Description("Greet someone").
		Args("name?").
		Flags(
			&cli.BoolFlag{Name: "loud", Short: "l", Help: "Greet loudly"},
			&cli.IntFlag{Name: "times", Short: "t", Help: "Number of times to greet", Value: 1},
		).
		Run(func(ctx *cli.Context) error {
			name := ctx.Arg(0)
			if name == "" {
				name = "World"
			}

			greeting := fmt.Sprintf("Hello, %s!", name)
			if ctx.Bool("loud") {
				greeting = strings.ToUpper(greeting)
			}

			times := ctx.Int("times")
			for i := 0; i < times; i++ {
				ctx.Println(greeting)
			}
			return nil
		})

	// Command with multiple aliases
	app.Command("generate").
		Description("Generate something").
		Aliases("gen", "g").
		Long("Generate various outputs. This command has aliases 'gen' and 'g'.").
		Run(func(ctx *cli.Context) error {
			ctx.Println("Generating... (use 'gen' or 'g' as shortcuts)")
			return nil
		})

	// Command group for organizing related commands
	users := app.Group("users").
		Description("User management commands")

	users.Command("list").
		Description("List all users").
		Alias("ls").
		Run(func(ctx *cli.Context) error {
			ctx.Println("Users:")
			ctx.Println("  - alice (admin)")
			ctx.Println("  - bob (user)")
			ctx.Println("  - charlie (user)")
			return nil
		})

	users.Command("create").
		Description("Create a new user").
		Args("username").
		Flags(
			&cli.StringFlag{
				Name:  "role",
				Short: "r",
				Help:  "User role",
				Value: "user",
				Enum:  []string{"admin", "user", "guest"},
			},
		).
		Run(func(ctx *cli.Context) error {
			username := ctx.Arg(0)
			role := ctx.String("role")
			ctx.Printf("Created user '%s' with role '%s'\n", username, role)
			return nil
		})

	users.Command("delete").
		Description("Delete a user").
		Args("username").
		Run(func(ctx *cli.Context) error {
			username := ctx.Arg(0)
			ctx.Printf("Deleted user '%s'\n", username)
			return nil
		})

	// Hidden command (won't show in help)
	app.Command("secret").
		Description("Secret command").
		Hidden().
		Run(func(ctx *cli.Context) error {
			ctx.Println("You found the secret command!")
			return nil
		})

	// Deprecated command
	app.Command("old-greet").
		Description("Old greeting command").
		Deprecated("Use 'greet' instead").
		Run(func(ctx *cli.Context) error {
			ctx.Println("Hello from the old command!")
			return nil
		})

	// Run the application
	if err := app.Run(); err != nil {
		if cli.IsHelpRequested(err) {
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(cli.GetExitCode(err))
	}
}
