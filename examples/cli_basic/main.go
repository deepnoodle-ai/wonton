// Example: Basic CLI
//
// Demonstrates the fundamentals of the Gooey CLI framework:
// - Creating an app with commands
// - Adding flags and arguments
// - Command groups for organization
// - Command aliases
//
// Run with:
//   go run examples/cli_basic/main.go --help
//   go run examples/cli_basic/main.go greet World
//   go run examples/cli_basic/main.go greet --loud World
//   go run examples/cli_basic/main.go users:list
//   go run examples/cli_basic/main.go users:create alice
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/deepnoodle-ai/gooey/cli"
)

func main() {
	// Create a new CLI application
	app := cli.New("myapp", "A demonstration CLI application")
	app.Version("1.0.0")

	// Add shell completion support
	app.AddCompletionCommand()

	// Simple command with an argument
	app.Command("greet", "Greet someone", cli.WithArgs("name")).
		AddFlag(&cli.Flag{
			Name:        "loud",
			Short:       "l",
			Description: "Greet loudly",
			Default:     false,
		}).
		AddFlag(&cli.Flag{
			Name:        "times",
			Short:       "t",
			Description: "Number of times to greet",
			Default:     1,
		}).
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
	app.Command("generate", "Generate something").
		Alias("gen", "g").
		Long("Generate various outputs. This command has aliases 'gen' and 'g'.").
		Run(func(ctx *cli.Context) error {
			ctx.Println("Generating... (use 'gen' or 'g' as shortcuts)")
			return nil
		})

	// Command group for organizing related commands
	users := app.Group("users", "User management commands")

	users.Command("list", "List all users").
		Alias("ls").
		Run(func(ctx *cli.Context) error {
			ctx.Println("Users:")
			ctx.Println("  - alice (admin)")
			ctx.Println("  - bob (user)")
			ctx.Println("  - charlie (user)")
			return nil
		})

	users.Command("create", "Create a new user", cli.WithArgs("username")).
		AddFlag(&cli.Flag{
			Name:        "role",
			Short:       "r",
			Description: "User role",
			Default:     "user",
			Enum:        []string{"admin", "user", "guest"},
		}).
		Run(func(ctx *cli.Context) error {
			username := ctx.Arg(0)
			role := ctx.String("role")
			ctx.Printf("Created user '%s' with role '%s'\n", username, role)
			return nil
		})

	users.Command("delete", "Delete a user", cli.WithArgs("username")).
		Run(func(ctx *cli.Context) error {
			username := ctx.Arg(0)
			ctx.Printf("Deleted user '%s'\n", username)
			return nil
		})

	// Hidden command (won't show in help)
	app.Command("secret", "Secret command").
		Hidden().
		Run(func(ctx *cli.Context) error {
			ctx.Println("You found the secret command!")
			return nil
		})

	// Deprecated command
	app.Command("old-greet", "Old greeting command").
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
