package cli_test

import (
	"fmt"

	"github.com/deepnoodle-ai/wonton/cli"
)

// ExampleNew demonstrates creating a basic CLI application.
func ExampleNew() {
	app := cli.New("myapp").
		Description("A sample CLI application").
		Version("1.0.0")

	app.Command("hello").
		Description("Say hello").
		Args("name").
		Run(func(ctx *cli.Context) error {
			name := ctx.Arg(0)
			ctx.Printf("Hello, %s!\n", name)
			return nil
		})

	// In a real application, you would call:
	// app.Execute()

	fmt.Println("Application created successfully")
	// Output: Application created successfully
}

// ExampleApp_Command demonstrates creating commands with flags.
func ExampleApp_Command() {
	app := cli.New("greet")

	app.Command("hello").
		Description("Greet someone").
		Args("name").
		Flags(
			cli.Bool("loud", "l").Help("Greet loudly"),
			cli.Int("times", "t").Default(1).Help("Number of times"),
		).
		Run(func(ctx *cli.Context) error {
			name := ctx.Arg(0)
			times := ctx.Int("times")
			loud := ctx.Bool("loud")

			greeting := fmt.Sprintf("Hello, %s!", name)
			if loud {
				greeting = fmt.Sprintf("%s!!!", greeting)
			}

			for i := 0; i < times; i++ {
				ctx.Println(greeting)
			}
			return nil
		})

	fmt.Println("Command registered")
	// Output: Command registered
}

// ExampleApp_Group demonstrates organizing commands into groups.
func ExampleApp_Group() {
	app := cli.New("myapp")

	// Create a group for user management commands
	users := app.Group("users").
		Description("User management commands")

	users.Command("list").
		Description("List all users").
		Run(func(ctx *cli.Context) error {
			ctx.Println("Listing users...")
			return nil
		})

	users.Command("create").
		Description("Create a new user").
		Args("username", "email").
		Run(func(ctx *cli.Context) error {
			username := ctx.Arg(0)
			email := ctx.Arg(1)
			ctx.Printf("Creating user: %s <%s>\n", username, email)
			return nil
		})

	fmt.Println("Command group created")
	// Output: Command group created
}

// ExampleString demonstrates using string flags.
func ExampleString() {
	app := cli.New("deploy")

	app.Command("run").
		Flags(
			cli.String("env", "e").
				Default("staging").
				Help("Deployment environment").
				Enum("dev", "staging", "prod"),
			cli.String("region", "r").
				Default("us-east-1").
				Help("AWS region"),
		).
		Run(func(ctx *cli.Context) error {
			env := ctx.String("env")
			region := ctx.String("region")
			ctx.Printf("Deploying to %s in %s\n", env, region)
			return nil
		})

	fmt.Println("String flags configured")
	// Output: String flags configured
}

// ExampleBool demonstrates using boolean flags.
func ExampleBool() {
	app := cli.New("build")

	app.Command("run").
		Flags(
			cli.Bool("verbose", "v").Help("Enable verbose output"),
			cli.Bool("force", "f").Help("Force rebuild"),
		).
		Run(func(ctx *cli.Context) error {
			if ctx.Bool("verbose") {
				ctx.Println("Verbose mode enabled")
			}
			if ctx.Bool("force") {
				ctx.Println("Force rebuild enabled")
			}
			return nil
		})

	fmt.Println("Boolean flags configured")
	// Output: Boolean flags configured
}

// ExampleInt demonstrates using integer flags.
func ExampleInt() {
	app := cli.New("server")

	app.Command("start").
		Flags(
			cli.Int("port", "p").Default(8080).Help("Server port"),
			cli.Int("workers", "w").Default(4).Help("Number of workers"),
		).
		Run(func(ctx *cli.Context) error {
			port := ctx.Int("port")
			workers := ctx.Int("workers")
			ctx.Printf("Starting server on port %d with %d workers\n", port, workers)
			return nil
		})

	fmt.Println("Integer flags configured")
	// Output: Integer flags configured
}

// ExampleCommand_Interactive demonstrates progressive interactivity.
func ExampleCommand_Interactive() {
	app := cli.New("delete")

	app.Command("data").
		Description("Delete all data").
		Flags(cli.Bool("force", "f").Help("Force deletion without confirmation")).
		Interactive(func(ctx *cli.Context) error {
			// In interactive mode, use prompts
			ctx.Info("Interactive mode detected")
			return nil
		}).
		NonInteractive(func(ctx *cli.Context) error {
			// In non-interactive mode, require --force flag
			if !ctx.Bool("force") {
				return cli.Error("--force flag required in non-interactive mode")
			}
			ctx.Info("Non-interactive mode, proceeding with force flag")
			return nil
		})

	fmt.Println("Progressive interactivity configured")
	// Output: Progressive interactivity configured
}

// ExampleCommand_Validate demonstrates argument validation.
func ExampleCommand_Validate() {
	app := cli.New("copy")

	app.Command("file").
		Args("source", "dest").
		Validate(func(ctx *cli.Context) error {
			// Custom validation logic
			source := ctx.Arg(0)
			if source == "" {
				return cli.Error("source cannot be empty")
			}
			return nil
		}).
		Run(func(ctx *cli.Context) error {
			source := ctx.Arg(0)
			dest := ctx.Arg(1)
			ctx.Printf("Copying %s to %s\n", source, dest)
			return nil
		})

	fmt.Println("Validation configured")
	// Output: Validation configured
}

// ExampleCommand_ExactArgs demonstrates exact argument count validation.
func ExampleCommand_ExactArgs() {
	app := cli.New("add")

	app.Command("numbers").
		ExactArgs(2).
		Run(func(ctx *cli.Context) error {
			// This handler will only run if exactly 2 arguments are provided
			ctx.Println("Processing exactly 2 numbers")
			return nil
		})

	fmt.Println("Exact args validation configured")
	// Output: Exact args validation configured
}

// ExampleRecover demonstrates panic recovery middleware.
func ExampleRecover() {
	app := cli.New("myapp")

	// Add recovery middleware to all commands
	app.Use(cli.Recover())

	app.Command("risky").
		Run(func(ctx *cli.Context) error {
			// If this panics, Recover() will catch it and return an error
			ctx.Println("Running risky operation...")
			return nil
		})

	fmt.Println("Recovery middleware configured")
	// Output: Recovery middleware configured
}

// ExampleBefore demonstrates Before middleware.
func ExampleBefore() {
	app := cli.New("myapp")

	app.Command("secure").
		Use(cli.Before(func(ctx *cli.Context) error {
			// Validation runs before the main handler
			if !ctx.IsSet("token") {
				return cli.Error("authentication token required")
			}
			return nil
		})).
		Flags(cli.String("token", "t").Help("Auth token")).
		Run(func(ctx *cli.Context) error {
			ctx.Println("Authenticated command running")
			return nil
		})

	fmt.Println("Before middleware configured")
	// Output: Before middleware configured
}

// ExampleError demonstrates creating rich errors.
func ExampleError() {
	// Create a rich error with hints and details
	err := cli.Errorf("failed to connect to %s", "database").
		Hint("Check your connection string and network settings").
		Detail("Host: %s", "localhost:5432").
		Detail("Timeout: %s", "30s").
		Code("ERR_CONNECTION")

	fmt.Printf("Error created: %s\n", err.ErrorCode())
	// Output: Error created: ERR_CONNECTION
}

// ExampleContext_Success demonstrates semantic output helpers.
func ExampleContext_Success() {
	app := cli.New("deploy")

	app.Command("run").
		Run(func(ctx *cli.Context) error {
			// These methods provide colored output when colors are enabled
			ctx.Success("Deployment completed successfully!")
			ctx.Info("Processing 10 items...")
			ctx.Warn("Configuration file not found, using defaults")
			return nil
		})

	fmt.Println("Semantic output configured")
	// Output: Semantic output configured
}

// ExampleApp_Test demonstrates testing CLI commands.
func ExampleApp_Test() {
	// Create a test app
	app := cli.TestApp("myapp")

	app.Command("greet").
		Args("name").
		Run(func(ctx *cli.Context) error {
			name := ctx.Arg(0)
			ctx.Printf("Hello, %s!\n", name)
			return nil
		})

	// In a real test, you would use testing.T:
	// result := app.Test(t, cli.TestArgs("greet", "Alice"))
	// assert.True(t, result.Success())
	// assert.True(t, result.Contains("Hello, Alice"))

	fmt.Println("Test app created")
	// Output: Test app created
}

// ExampleParseFlags demonstrates struct-based flag definition.
func ExampleParseFlags() {
	type ServerConfig struct {
		Port    int    `flag:"port,p" default:"8080" help:"Server port"`
		Host    string `flag:"host,h" default:"localhost" help:"Server host"`
		Debug   bool   `flag:"debug,d" help:"Enable debug mode"`
		Workers int    `flag:"workers,w" default:"4" help:"Number of workers"`
	}

	app := cli.New("server")
	cmd := app.Command("start")
	cli.ParseFlags[ServerConfig](cmd)

	cmd.Run(func(ctx *cli.Context) error {
		config, err := cli.BindFlags[ServerConfig](ctx)
		if err != nil {
			return err
		}
		ctx.Printf("Server: %s:%d (workers=%d, debug=%v)\n",
			config.Host, config.Port, config.Workers, config.Debug)
		return nil
	})

	fmt.Println("Struct-based flags configured")
	// Output: Struct-based flags configured
}
