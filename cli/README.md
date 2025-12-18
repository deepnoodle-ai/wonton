# cli

CLI framework with commands, flags, config, middleware, and progressive interactivity.

## Summary

The cli package provides a powerful CLI framework that integrates with Wonton's
TUI capabilities. It supports command registration with hierarchical groups,
struct-based flag definitions with environment variable fallbacks, automatic
help/version commands, shell completion generation, and progressive
interactivity where the same command can work as a quick one-liner or upgrade to
a rich TUI when run in a terminal. The framework includes semantic output
helpers, interactive prompts, middleware support, and comprehensive validation.

## Usage Examples

### Basic Application

```go
package main

import (
    "fmt"
    "os"

    "github.com/deepnoodle-ai/wonton/cli"
)

func main() {
    app := cli.New("myapp").
        Description("A demonstration CLI application").
        Version("1.0.0")

    app.Command("greet").
        Description("Greet someone").
        Args("name?").
        Flags(
            cli.Bool("loud", "l").Help("Greet loudly"),
            cli.Int("times", "t").Default(1).Help("Number of times"),
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

    if err := app.Execute(); err != nil {
        if cli.IsHelpRequested(err) {
            os.Exit(0)
        }
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(cli.GetExitCode(err))
    }
}
```

### Command Groups

```go
package main

import (
    "github.com/deepnoodle-ai/wonton/cli"
)

func main() {
    app := cli.New("myapp").Description("My application")

    // Create a command group
    users := app.Group("users").
        Description("User management commands")

    users.Command("list").
        Description("List all users").
        Alias("ls").
        Run(func(ctx *cli.Context) error {
            ctx.Println("Users:")
            ctx.Println("  - alice (admin)")
            ctx.Println("  - bob (user)")
            return nil
        })

    users.Command("create").
        Description("Create a new user").
        Args("username").
        Flags(
            cli.String("role", "r").
                Default("user").
                Enum("admin", "user", "guest").
                Help("User role"),
        ).
        Run(func(ctx *cli.Context) error {
            username := ctx.Arg(0)
            role := ctx.String("role")
            ctx.Success("Created user '%s' with role '%s'", username, role)
            return nil
        })

    app.Execute()
}
```

Groups support help flags: `myapp users --help` or `myapp users -h` displays help for the group, listing all available subcommands.

### Struct-Based Flags

```go
package main

import (
    "github.com/deepnoodle-ai/wonton/cli"
)

type RunFlags struct {
    Model       string  `flag:"model,m" default:"claude-sonnet" help:"Model to use" env:"MYAPP_MODEL"`
    Temperature float64 `flag:"temperature,t" default:"0.7" help:"Temperature"`
    MaxTokens   int     `flag:"max-tokens" default:"1000" help:"Max tokens"`
    Stream      bool    `flag:"stream,s" help:"Stream output"`
    Format      string  `flag:"format,f" enum:"text,json,markdown" default:"text" help:"Output format"`
}

func main() {
    app := cli.New("myapp")

    runCmd := app.Command("run").
        Description("Run a prompt").
        Args("prompt")

    // Parse struct into flags
    cli.ParseFlags[RunFlags](runCmd)

    runCmd.Run(func(ctx *cli.Context) error {
        // Bind flag values to struct
        flags, err := cli.BindFlags[RunFlags](ctx)
        if err != nil {
            return err
        }

        ctx.Printf("Model: %s\n", flags.Model)
        ctx.Printf("Temperature: %.2f\n", flags.Temperature)
        ctx.Printf("Format: %s\n", flags.Format)
        return nil
    })

    app.Execute()
}
```

### Interactive Prompts

```go
package main

import (
    "github.com/deepnoodle-ai/wonton/cli"
)

func main() {
    app := cli.New("deploy").Description("Deployment tool")

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

    app.Execute()
}
```

### Semantic Output Helpers

```go
package main

import (
    "github.com/deepnoodle-ai/wonton/cli"
)

func main() {
    app := cli.New("status").Description("Check system status")

    app.Command("check").
        Description("Run health check").
        Run(func(ctx *cli.Context) error {
            ctx.Info("Starting health check...")

            if checkDatabase() {
                ctx.Success("Database: OK")
            } else {
                ctx.Fail("Database: Failed")
            }

            usage := getDiskUsage()
            if usage > 90 {
                ctx.Warn("Disk usage at %d%%", usage)
            } else {
                ctx.Info("Disk usage: %d%%", usage)
            }

            return nil
        })

    app.Execute()
}
```

### Validation

```go
package main

import (
    "github.com/deepnoodle-ai/wonton/cli"
)

func main() {
    app := cli.New("myapp")

    // Exact argument count
    app.Command("pair").
        Description("Pair two items").
        ExactArgs(2).
        Run(func(ctx *cli.Context) error {
            ctx.Printf("Pairing: %s <-> %s\n", ctx.Arg(0), ctx.Arg(1))
            return nil
        })

    // Argument range
    app.Command("add").
        Description("Add items (1-3 items)").
        ArgsRange(1, 3).
        Run(func(ctx *cli.Context) error {
            ctx.Printf("Adding %d items\n", ctx.NArg())
            return nil
        })

    // No arguments
    app.Command("status").
        Description("Show status").
        NoArgs().
        Run(func(ctx *cli.Context) error {
            ctx.Println("Status: OK")
            return nil
        })

    // Custom validation
    app.Command("set").
        Description("Set a value").
        Args("key", "value").
        Validate(func(ctx *cli.Context) error {
            key := ctx.Arg(0)
            if len(key) > 50 {
                return cli.Error("key too long").
                    Hint("Keys must be 50 characters or less")
            }
            return nil
        }).
        Run(func(ctx *cli.Context) error {
            ctx.Printf("Set %s = %s\n", ctx.Arg(0), ctx.Arg(1))
            return nil
        })

    app.Execute()
}
```

### Middleware

```go
package main

import (
    "fmt"
    "time"

    "github.com/deepnoodle-ai/wonton/cli"
)

func main() {
    app := cli.New("myapp")

    // Global middleware (applies to all commands)
    app.Use(cli.Recover())
    app.Use(LoggingMiddleware())

    // Command-specific middleware
    app.Command("dangerous").
        Description("Dangerous operation").
        Use(cli.Confirm("Are you sure you want to continue?")).
        Run(func(ctx *cli.Context) error {
            ctx.Println("Performing dangerous operation...")
            return nil
        })

    app.Execute()
}

func LoggingMiddleware() cli.Middleware {
    return func(next cli.Handler) cli.Handler {
        return func(ctx *cli.Context) error {
            start := time.Now()
            err := next(ctx)
            duration := time.Since(start)
            fmt.Printf("Command took %s\n", duration)
            return err
        }
    }
}
```

**Middleware execution order:** App-level middleware runs before command-level middleware. If you have:

- App middleware: `[A, B]`
- Command middleware: `[C, D]`

Execution order is: `A-before → B-before → C-before → D-before → handler → D-after → C-after → B-after → A-after`

### Progressive Interactivity

```go
package main

import (
    "github.com/deepnoodle-ai/wonton/cli"
)

func main() {
    app := cli.New("myapp")

    app.Command("process").
        Description("Process files").
        ArgsRange(1, -1). // Require at least one argument
        // Interactive mode (with TUI)
        Interactive(func(ctx *cli.Context) error {
            files := ctx.Args() // All positional args are available
            // Show interactive TUI with progress, selection, etc.
            return runInteractiveProcessor(files)
        }).
        // Non-interactive mode (pipe-friendly)
        NonInteractive(func(ctx *cli.Context) error {
            files := ctx.Args()
            // Simple streaming output for pipes
            return runBatchProcessor(files)
        }).
        // Fallback handler (auto-detects terminal)
        Run(func(ctx *cli.Context) error {
            files := ctx.Args()
            if ctx.Interactive() {
                return runInteractiveProcessor(files)
            }
            return runBatchProcessor(files)
        })

    app.Execute()
}
```

### Root Handler

```go
package main

import (
    "io"
    "github.com/deepnoodle-ai/wonton/cli"
)

func main() {
    // Root handler that runs when no command is specified
    app := cli.New("cat").
        Description("Concatenate files").
        Args("file?").
        Run(func(ctx *cli.Context) error {
            if ctx.NArg() == 0 {
                // Read from stdin
                io.Copy(ctx.Stdout(), ctx.Stdin())
                return nil
            }
            // Read from file
            return readFile(ctx, ctx.Arg(0))
        })

    // Can still have subcommands
    app.Command("version").
        Description("Show version").
        Run(func(ctx *cli.Context) error {
            ctx.Println("cat version 1.0.0")
            return nil
        })

    app.Execute()
}
```

### Group Handler

```go
package main

import (
    "github.com/deepnoodle-ai/wonton/cli"
)

func main() {
    app := cli.New("myapp")

    // Group with handler and subcommands
    users := app.Group("users").
        Description("User management").
        Flags(cli.Bool("all", "a").Help("Show all users")).
        Run(func(ctx *cli.Context) error {
            // Runs when: myapp users or myapp users -a
            if ctx.Bool("all") {
                return listAllUsers()
            }
            return listActiveUsers()
        })

    // Subcommands take precedence
    users.Command("add").
        Description("Add a user").
        Args("name").
        Run(func(ctx *cli.Context) error {
            return addUser(ctx.Arg(0))
        })

    app.Execute()
}
```

### Global Flags

```go
package main

import (
    "github.com/deepnoodle-ai/wonton/cli"
)

func main() {
    app := cli.New("myapp")

    // Global flags available to all commands
    app.GlobalFlags(
        cli.Bool("verbose", "v").Help("Verbose output"),
        cli.String("config", "c").Help("Config file path"),
    )

    app.Command("build").
        Description("Build the project").
        Run(func(ctx *cli.Context) error {
            if ctx.Bool("verbose") {
                ctx.Info("Verbose mode enabled")
            }
            config := ctx.String("config")
            if config != "" {
                ctx.Info("Using config: %s", config)
            }
            return nil
        })

    app.Execute()
}
```

Global flags can be placed before or after the command name:

```bash
myapp -v build           # Global flag before command
myapp build -v           # Global flag after command
myapp --config=app.yaml build  # Both work
```

### Slice Flags

Slice flags accumulate values when specified multiple times:

```go
package main

import (
    "github.com/deepnoodle-ai/wonton/cli"
)

func main() {
    app := cli.New("docker")

    app.Command("run").
        Description("Run a container").
        Args("image").
        Flags(
            cli.Strings("env", "e").Help("Environment variables"),
            cli.Strings("volume", "v").Help("Volume mounts"),
            cli.Ints("port", "p").Help("Port mappings"),
        ).
        Run(func(ctx *cli.Context) error {
            image := ctx.Arg(0)
            envVars := ctx.Strings("env")    // []string
            volumes := ctx.Strings("volume") // []string
            ports := ctx.Ints("port")        // []int

            ctx.Printf("Running %s with %d env vars, %d volumes, %d ports\n",
                image, len(envVars), len(volumes), len(ports))
            return nil
        })

    app.Execute()
}
```

Usage:

```bash
# Multiple values accumulate into slices
myapp run -e FOO=bar -e BAZ=qux --volume /host:/container nginx
# envVars = ["FOO=bar", "BAZ=qux"]
# volumes = ["/host:/container"]

# Defaults are replaced when user provides values
myapp run --port 8080 --port 443 nginx
# ports = [8080, 443]
```

## API Reference

### Application

| Method                     | Description                      | Parameters                    | Returns    |
| -------------------------- | -------------------------------- | ----------------------------- | ---------- |
| `New(name)`                | Create new CLI app               | `string`                      | `*App`     |
| `Description(desc)`        | Set app description              | `string`                      | `*App`     |
| `Version(version)`         | Set app version                  | `string`                      | `*App`     |
| `Command(name)`            | Register/get command             | `string`                      | `*Command` |
| `Group(name)`              | Create command group             | `string`                      | `*Group`   |
| `Use(mw...)`               | Add middleware                   | `...Middleware`               | `*App`     |
| `GlobalFlags(flags...)`    | Add global flags                 | `...Flag`                     | `*App`     |
| `Run(handler)`             | Set root handler                 | `Handler`                     | `*App`     |
| `Args(names...)`           | Set root args                    | `...string`                   | `*App`     |
| `Validate(fn)`             | Add validation                   | `func(*Context) error`        | `*App`     |
| `Execute()`                | Execute with os.Args             | None                          | `error`    |
| `ExecuteArgs(args)`        | Execute with args                | `[]string`                    | `error`    |
| `ExecuteContext(ctx, args)`| Execute with context             | `context.Context`, `[]string` | `error`    |
| `SetColorEnabled(enabled)` | Enable/disable colors            | `bool`                        | `*App`     |
| `HelpTheme(theme)`         | Set help theme                   | `HelpTheme`                   | `*App`     |
| `ForceInteractive(val)`    | Force interactive mode (testing) | `bool`                        | `*App`     |
| `SetStdin(reader)`         | Set input reader                 | `io.Reader`                   | `*App`     |
| `SetStdout(writer)`        | Set output writer                | `io.Writer`                   | `*App`     |
| `SetStderr(writer)`        | Set error output writer          | `io.Writer`                   | `*App`     |

### Command

| Method                    | Description                                | Parameters             | Returns    |
| ------------------------- | ------------------------------------------ | ---------------------- | ---------- |
| `Description(desc)`       | Set description                            | `string`               | `*Command` |
| `Long(desc)`              | Set long description                       | `string`               | `*Command` |
| `Args(names...)`          | Set positional args (use "?" for optional) | `...string`            | `*Command` |
| `Flags(flags...)`         | Add flags                                  | `...Flag`              | `*Command` |
| `Run(handler)`            | Set handler                                | `Handler`              | `*Command` |
| `Interactive(handler)`    | Set interactive handler                    | `Handler`              | `*Command` |
| `NonInteractive(handler)` | Set non-interactive handler                | `Handler`              | `*Command` |
| `Use(mw...)`              | Add middleware                             | `...Middleware`        | `*Command` |
| `Validate(fn)`            | Add validation                             | `func(*Context) error` | `*Command` |
| `ArgsRange(min, max)`     | Validate arg count range                   | `int`, `int`           | `*Command` |
| `ExactArgs(n)`            | Require exact arg count                    | `int`                  | `*Command` |
| `NoArgs()`                | Require no args                            | None                   | `*Command` |
| `Alias(names...)`         | Add aliases                                | `...string`            | `*Command` |
| `Hidden()`                | Hide from help                             | None                   | `*Command` |
| `Deprecated(msg)`         | Mark as deprecated                         | `string`               | `*Command` |

### Context

| Method                            | Description                   | Parameters            | Returns           |
| --------------------------------- | ----------------------------- | --------------------- | ----------------- |
| `Context()`                       | Get Go context                | None                  | `context.Context` |
| `App()`                           | Get app                       | None                  | `*App`            |
| `Command()`                       | Get command                   | None                  | `*Command`        |
| `Interactive()`                   | Check if interactive          | None                  | `bool`            |
| `Args()`                          | Get all args                  | None                  | `[]string`        |
| `Arg(i)`                          | Get arg at index              | `int`                 | `string`          |
| `NArg()`                          | Get arg count                 | None                  | `int`             |
| `String(name)`                    | Get string flag               | `string`              | `string`          |
| `Strings(name)`                   | Get string slice flag         | `string`              | `[]string`        |
| `Int(name)`                       | Get int flag                  | `string`              | `int`             |
| `Ints(name)`                      | Get int slice flag            | `string`              | `[]int`           |
| `Int64(name)`                     | Get int64 flag                | `string`              | `int64`           |
| `Float64(name)`                   | Get float64 flag              | `string`              | `float64`         |
| `Bool(name)`                      | Get bool flag                 | `string`              | `bool`            |
| `IsSet(name)`                     | Check if flag was set         | `string`              | `bool`            |
| `Stdin()`                         | Get stdin reader              | None                  | `io.Reader`       |
| `Stdout()`                        | Get stdout writer             | None                  | `io.Writer`       |
| `Stderr()`                        | Get stderr writer             | None                  | `io.Writer`       |
| `Print(args...)`                  | Print to stdout               | `...any`              | None              |
| `Printf(format, args...)`         | Printf to stdout              | `string`, `...any`    | None              |
| `Println(args...)`                | Println to stdout             | `...any`              | None              |
| `Error(args...)`                  | Print to stderr               | `...any`              | None              |
| `Errorf(format, args...)`         | Printf to stderr              | `string`, `...any`    | None              |
| `Errorln(args...)`                | Println to stderr             | `...any`              | None              |
| `Success(format, args...)`        | Green message to stdout       | `string`, `...any`    | None              |
| `Fail(format, args...)`           | Red message to stderr         | `string`, `...any`    | None              |
| `Warn(format, args...)`           | Yellow message to stderr      | `string`, `...any`    | None              |
| `Info(format, args...)`           | Cyan message to stdout        | `string`, `...any`    | None              |
| `Select(title, options...)`       | Show selection prompt         | `string`, `...string` | `int`, `error`    |
| `SelectString(title, options...)` | Show selection, return string | `string`, `...string` | `string`, `error` |
| `Input(prompt)`                   | Show text input prompt        | `string`              | `string`, `error` |
| `Confirm(message)`                | Show yes/no confirmation      | `string`              | `bool`, `error`   |

### Flag Builders

| Function                | Description                      | Parameters         | Returns            |
| ----------------------- | -------------------------------- | ------------------ | ------------------ |
| `String(name, short)`   | Create string flag builder       | `string`, `string` | `*stringBuilder`   |
| `Bool(name, short)`     | Create bool flag builder         | `string`, `string` | `*boolBuilder`     |
| `Int(name, short)`      | Create int flag builder          | `string`, `string` | `*intBuilder`      |
| `Float(name, short)`    | Create float64 flag builder      | `string`, `string` | `*floatBuilder`    |
| `Duration(name, short)` | Create duration flag builder     | `string`, `string` | `*durationBuilder` |
| `Strings(name, short)`  | Create string slice flag builder | `string`, `string` | `*stringsBuilder`  |
| `Ints(name, short)`     | Create int slice flag builder    | `string`, `string` | `*intsBuilder`     |

### Flag Builder Methods

All flag builders support:

- `.Default(value)` - Set default value
- `.Help(text)` - Set help text
- `.Env(varName)` - Set environment variable name
- `.Required()` - Mark as required
- `.Hidden()` - Hide from help

String flags additionally support:

- `.Enum(values...)` - Restrict to enum values
- `.ValidateWith(fn)` - Custom validation

### Middleware Functions

| Function                 | Description             | Parameters             | Returns      |
| ------------------------ | ----------------------- | ---------------------- | ------------ |
| `Recover()`              | Recover from panics     | None                   | `Middleware` |
| `RequireFlags(names...)` | Require flags to be set | `...string`            | `Middleware` |
| `Confirm(message)`       | Prompt for confirmation | `string`               | `Middleware` |
| `Before(fn)`             | Run before command      | `func(*Context) error` | `Middleware` |
| `After(fn)`              | Run after command       | `func(*Context) error` | `Middleware` |

### Struct Flag Functions

| Function             | Description                  | Parameters | Returns       |
| -------------------- | ---------------------------- | ---------- | ------------- |
| `ParseFlags[T](cmd)` | Parse struct tags into flags | `*Command` | `*T`          |
| `BindFlags[T](ctx)`  | Bind flag values to struct   | `*Context` | `*T`, `error` |

### Error Functions

| Function                  | Description                                           | Parameters         | Returns         |
| ------------------------- | ----------------------------------------------------- | ------------------ | --------------- |
| `Error(msg)`              | Create error                                          | `string`           | `*CommandError` |
| `Errorf(format, args...)` | Create formatted error                                | `string`, `...any` | `*CommandError` |
| `Exit(code)`              | Create exit error with code                           | `int`              | `error`         |
| `IsHelpRequested(err)`    | Check if help was requested (supports wrapped errors) | `error`            | `bool`          |
| `GetExitCode(err)`        | Get exit code from error (supports wrapped errors)    | `error`            | `int`           |

Note: `IsHelpRequested` and `GetExitCode` use `errors.As` internally, so they work correctly with wrapped errors (e.g., `fmt.Errorf("failed: %w", cli.Exit(1))`).

## Tips

### Flag Values Starting with Dash

Flag values that look like negative numbers are handled correctly:

```bash
myapp --count -5      # count = -5
myapp --offset -10    # offset = -10
myapp --rate -.5      # rate = -0.5
```

Use `--` to separate flags from positional arguments that start with `-`:

```bash
myapp -- -filename    # "-filename" is a positional arg, not a flag
```

### Help Everywhere

Help is available at multiple levels:

- `myapp help` or `myapp --help` - App help
- `myapp command --help` - Command help
- `myapp group --help` - Group help (lists subcommands)
- `myapp group command --help` - Subcommand help

### Testing Commands

Use the built-in test infrastructure for clean command testing:

```go
func TestMyCommand(t *testing.T) {
    app := setupApp()
    result := app.Test(t, cli.TestArgs("mycommand", "--flag", "value"))

    if !result.Success() {
        t.Fatalf("command failed: %v", result.Err)
    }
    if !result.Contains("expected output") {
        t.Errorf("unexpected output: %s", result.Stdout)
    }
}
```

### Programmatic Invocation

Use `ExecuteArgs` with custom I/O for programmatic CLI usage:

```go
var stdout, stderr bytes.Buffer

app := cli.New("myapp")
app.SetStdout(&stdout)
app.SetStderr(&stderr)
app.SetStdin(strings.NewReader("user input\n"))

app.Command("greet").
    Args("name").
    Run(func(ctx *cli.Context) error {
        ctx.Printf("Hello, %s!\n", ctx.Arg(0))
        return nil
    })

err := app.ExecuteArgs([]string{"greet", "Alice"})
output := stdout.String() // "Hello, Alice!\n"
```

## Related Packages

- **[tui](../tui/)** - Terminal UI library for interactive commands
- **[color](../color/)** - Color utilities for output formatting
- **[env](../env/)** - Environment variable and config loading
- **[terminal](../terminal/)** - Terminal control and detection
