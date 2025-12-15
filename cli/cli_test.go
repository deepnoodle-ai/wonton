package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestAppBasic(t *testing.T) {
	app := New("test").Description("Test application")
	assert.Equal(t, "test", app.name)
	assert.Equal(t, "Test application", app.description)
}

func TestCommand(t *testing.T) {
	var executed bool
	var receivedArg string

	app := New("test").Description("Test application")
	app.Command("greet").
		Description("Greet someone").
		Args("name").
		Run(func(ctx *Context) error {
			executed = true
			receivedArg = ctx.Arg(0)
			return nil
		})

	err := app.RunArgs([]string{"greet", "World"})
	assert.NoError(t, err)
	assert.True(t, executed)
	assert.Equal(t, "World", receivedArg)
}

func TestFlags(t *testing.T) {
	var model string
	var temp float64
	var verbose bool

	app := New("test").Description("Test")
	app.Command("run").
		Description("Run something").
		Flags(
			&StringFlag{Name: "model", Short: "m", Value: "default"},
			&Float64Flag{Name: "temp", Short: "t", Value: 0.7},
			&BoolFlag{Name: "verbose", Short: "v"},
		).
		Run(func(ctx *Context) error {
			model = ctx.String("model")
			temp = ctx.Float64("temp")
			verbose = ctx.Bool("verbose")
			return nil
		})

	// Test defaults
	err := app.RunArgs([]string{"run"})
	assert.NoError(t, err)
	assert.Equal(t, "default", model)
	assert.InDelta(t, 0.7, temp, 0.001)
	assert.False(t, verbose)

	// Test with flags
	err = app.RunArgs([]string{"run", "--model", "gpt-4", "-t", "0.9", "-v"})
	assert.NoError(t, err)
	assert.Equal(t, "gpt-4", model)
	assert.InDelta(t, 0.9, temp, 0.001)
	assert.True(t, verbose)
}

func TestFlagsEqualsStyle(t *testing.T) {
	var value string

	app := New("test").Description("Test")
	app.Command("run").
		Description("Run").
		Flags(&StringFlag{Name: "config"}).
		Run(func(ctx *Context) error {
			value = ctx.String("config")
			return nil
		})

	err := app.RunArgs([]string{"run", "--config=myfile.yaml"})
	assert.NoError(t, err)
	assert.Equal(t, "myfile.yaml", value)
}

func TestRequiredFlag(t *testing.T) {
	app := New("test").Description("Test")
	app.Command("run").
		Description("Run").
		Flags(&StringFlag{Name: "required", Required: true}).
		Run(func(ctx *Context) error {
			return nil
		})

	err := app.RunArgs([]string{"run"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing required flag")
}

func TestEnumFlag(t *testing.T) {
	app := New("test").Description("Test")
	app.Command("run").
		Description("Run").
		Flags(&StringFlag{Name: "format", Enum: []string{"json", "yaml", "text"}, Value: "text"}).
		Run(func(ctx *Context) error {
			return nil
		})

	// Valid value
	err := app.RunArgs([]string{"run", "--format", "json"})
	assert.NoError(t, err)

	// Invalid value
	err = app.RunArgs([]string{"run", "--format", "xml"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid value")
}

func TestRequiredArg(t *testing.T) {
	app := New("test").Description("Test")
	app.Command("greet").
		Description("Greet").
		Args("name").
		Run(func(ctx *Context) error {
			return nil
		})

	err := app.RunArgs([]string{"greet"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing required argument")
}

func TestOptionalArg(t *testing.T) {
	var name string

	app := New("test").Description("Test")
	app.Command("greet").
		Description("Greet").
		Args("name?").
		Run(func(ctx *Context) error {
			name = ctx.Arg(0)
			return nil
		})

	err := app.RunArgs([]string{"greet"})
	assert.NoError(t, err)
	assert.Empty(t, name)

	err = app.RunArgs([]string{"greet", "World"})
	assert.NoError(t, err)
	assert.Equal(t, "World", name)
}

func TestGroup(t *testing.T) {
	var executed bool

	app := New("test").Description("Test")
	users := app.Group("users").Description("User management")
	users.Command("list").
		Description("List users").
		Run(func(ctx *Context) error {
			executed = true
			return nil
		})

	err := app.RunArgs([]string{"users:list"})
	assert.NoError(t, err)
	assert.True(t, executed)
}

func TestMiddleware(t *testing.T) {
	var order []string

	app := New("test").Description("Test")
	app.Use(func(next Handler) Handler {
		return func(ctx *Context) error {
			order = append(order, "global-before")
			err := next(ctx)
			order = append(order, "global-after")
			return err
		}
	})

	app.Command("run").
		Description("Run").
		Use(func(next Handler) Handler {
			return func(ctx *Context) error {
				order = append(order, "cmd-before")
				err := next(ctx)
				order = append(order, "cmd-after")
				return err
			}
		}).
		Run(func(ctx *Context) error {
			order = append(order, "handler")
			return nil
		})

	err := app.RunArgs([]string{"run"})
	assert.NoError(t, err)
	// Middleware is applied in reverse order: cmd middleware wraps first, then global
	// So execution is: cmd-before -> global-before -> handler -> global-after -> cmd-after
	assert.Equal(t, []string{
		"cmd-before",
		"global-before",
		"handler",
		"global-after",
		"cmd-after",
	}, order)
}

func TestHelp(t *testing.T) {
	var buf bytes.Buffer

	app := New("test").Description("Test application")
	app.Version("1.0.0")
	app.stdout = &buf
	app.Command("run").Description("Run something")

	app.RunArgs([]string{"help"})

	output := buf.String()
	assert.Contains(t, output, "test")
	assert.Contains(t, output, "Test application")
	assert.Contains(t, output, "1.0.0")
	assert.Contains(t, output, "run")
}

func TestCommandHelp(t *testing.T) {
	var buf bytes.Buffer

	app := New("test").Description("Test")
	app.stdout = &buf
	app.Command("run").
		Description("Run something").
		Args("file").
		Flags(&BoolFlag{Name: "verbose", Short: "v", Help: "Verbose output"}).
		Run(func(ctx *Context) error {
			return nil
		})

	err := app.RunArgs([]string{"run", "--help"})
	assert.True(t, IsHelpRequested(err))

	output := buf.String()
	assert.Contains(t, output, "run")
	assert.Contains(t, output, "Run something")
	assert.Contains(t, output, "--verbose")
	assert.Contains(t, output, "<file>")
}

func TestToolSchema(t *testing.T) {
	app := New("agent").Description("AI Agent")
	app.Command("create-file").
		Description("Create a file").
		Tool().
		AddArg(&Arg{Name: "path", Description: "File path", Required: true}).
		AddArg(&Arg{Name: "content", Description: "File content", Required: true}).
		Run(func(ctx *Context) error {
			return nil
		})

	schemas := app.GetToolSchemas()
	assert.Len(t, schemas, 1)

	schema := schemas[0]
	assert.Equal(t, "create-file", schema.Name)
	assert.Equal(t, "Create a file", schema.Description)
	assert.Contains(t, schema.Required, "path")
	assert.Contains(t, schema.Required, "content")
}

func TestContext(t *testing.T) {
	var gotCtx *Context

	app := New("test").Description("Test")
	app.Command("run").
		Description("Run").
		Run(func(ctx *Context) error {
			gotCtx = ctx
			return nil
		})

	err := app.RunContext(context.Background(), []string{"run"})
	assert.NoError(t, err)
	assert.NotNil(t, gotCtx)
	assert.NotNil(t, gotCtx.Context())
}

func TestContextOutput(t *testing.T) {
	var buf bytes.Buffer

	app := New("test").Description("Test")
	app.stdout = &buf
	app.Command("run").
		Description("Run").
		Run(func(ctx *Context) error {
			ctx.Println("Hello")
			ctx.Printf("Count: %d\n", 42)
			return nil
		})

	app.RunArgs([]string{"run"})

	assert.Equal(t, "Hello\nCount: 42\n", buf.String())
}

func TestError(t *testing.T) {
	err := Error("something failed").
		Hint("try again").
		Code("FAILED").
		Detail("detail 1").
		Detail("detail 2")

	msg := err.Error()
	assert.Contains(t, msg, "something failed")
	assert.Contains(t, msg, "try again")
	assert.Contains(t, msg, "detail 1")
	assert.Equal(t, "FAILED", err.ErrorCode())
}

func TestValidation(t *testing.T) {
	app := New("test").Description("Test")
	app.Command("run").
		Description("Run").
		Flags(&IntFlag{Name: "count", Value: 0}).
		Validate(func(ctx *Context) error {
			if ctx.Int("count") > 10 {
				return Error("count must be <= 10")
			}
			return nil
		}).
		Run(func(ctx *Context) error {
			return nil
		})

	err := app.RunArgs([]string{"run", "--count", "15"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "count must be <= 10")
}

func TestDoubleDash(t *testing.T) {
	var args []string

	app := New("test").Description("Test")
	app.Command("run").
		Description("Run").
		Run(func(ctx *Context) error {
			args = ctx.Args()
			return nil
		})

	err := app.RunArgs([]string{"run", "--", "-flag-like", "--another"})
	assert.NoError(t, err)
	assert.Equal(t, []string{"-flag-like", "--another"}, args)
}

func TestMultipleShortFlags(t *testing.T) {
	var verbose, debug bool

	app := New("test").Description("Test")
	app.Command("run").
		Description("Run").
		Flags(
			&BoolFlag{Name: "verbose", Short: "v"},
			&BoolFlag{Name: "debug", Short: "d"},
		).
		Run(func(ctx *Context) error {
			verbose = ctx.Bool("verbose")
			debug = ctx.Bool("debug")
			return nil
		})

	err := app.RunArgs([]string{"run", "-vd"})
	assert.NoError(t, err)
	assert.True(t, verbose)
	assert.True(t, debug)
}

func TestRecoverMiddleware(t *testing.T) {
	app := New("test").Description("Test")
	app.Use(Recover())
	app.Command("panic").
		Description("Panic").
		Run(func(ctx *Context) error {
			panic("test panic")
		})

	err := app.RunArgs([]string{"panic"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "panic")
}

func TestBeforeAfterMiddleware(t *testing.T) {
	var order []string

	app := New("test").Description("Test")
	app.Use(
		Before(func(ctx *Context) error {
			order = append(order, "before")
			return nil
		}),
		After(func(ctx *Context) error {
			order = append(order, "after")
			return nil
		}),
	)
	app.Command("run").
		Description("Run").
		Run(func(ctx *Context) error {
			order = append(order, "run")
			return nil
		})

	app.RunArgs([]string{"run"})
	assert.Equal(t, []string{"before", "run", "after"}, order)
}

func TestUnknownCommand(t *testing.T) {
	app := New("test").Description("Test")
	app.Command("run").
		Description("Run").
		Run(func(ctx *Context) error {
			return nil
		})

	err := app.RunArgs([]string{"unknown"})
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "unknown command"))
}

func TestVersion(t *testing.T) {
	var buf bytes.Buffer

	app := New("test").Description("Test")
	app.Version("1.2.3")
	app.stdout = &buf

	app.RunArgs([]string{"version"})
	assert.Equal(t, "1.2.3\n", buf.String())
}

func TestCommandAlias(t *testing.T) {
	var executed bool

	app := New("test").Description("Test")
	app.Command("generate").
		Description("Generate something").
		Alias("gen", "g").
		Run(func(ctx *Context) error {
			executed = true
			return nil
		})

	// Test using alias
	err := app.RunArgs([]string{"gen"})
	assert.NoError(t, err)
	assert.True(t, executed)

	// Test using another alias
	executed = false
	err = app.RunArgs([]string{"g"})
	assert.NoError(t, err)
	assert.True(t, executed)

	// Test original name still works
	executed = false
	err = app.RunArgs([]string{"generate"})
	assert.NoError(t, err)
	assert.True(t, executed)
}

func TestGroupCommandAlias(t *testing.T) {
	var executed bool

	app := New("test").Description("Test")
	users := app.Group("users").Description("User management")
	users.Command("list").
		Description("List users").
		Alias("ls", "l").
		Run(func(ctx *Context) error {
			executed = true
			return nil
		})

	// Test using alias
	err := app.RunArgs([]string{"users:ls"})
	assert.NoError(t, err)
	assert.True(t, executed)

	// Test short alias
	executed = false
	err = app.RunArgs([]string{"users:l"})
	assert.NoError(t, err)
	assert.True(t, executed)
}

// RunFlags is a test struct for flag parsing
type RunFlags struct {
	Model   string  `flag:"model,m" default:"claude-sonnet" help:"Model to use"`
	Temp    float64 `flag:"temp,t" default:"0.7" help:"Temperature"`
	Verbose bool    `flag:"verbose,v" help:"Verbose output"`
	Count   int     `flag:"count,c" default:"5" help:"Count"`
	Format  string  `flag:"format,f" enum:"json,text,yaml" default:"text" help:"Output format"`
}

func TestParseFlagsGeneric(t *testing.T) {
	app := New("test").Description("Test")
	cmd := app.Command("run").Description("Run something")

	// Use ParseFlags to set up flags from struct
	ParseFlags[RunFlags](cmd)
	cmd.Run(func(ctx *Context) error {
		return nil
	})

	// Verify flags were registered
	assert.Len(t, cmd.flags, 5)

	// Find model flag
	var modelFlag Flag
	for _, f := range cmd.flags {
		if f.GetName() == "model" {
			modelFlag = f
			break
		}
	}
	assert.NotNil(t, modelFlag)
	assert.Equal(t, "m", modelFlag.GetShort())
	assert.Equal(t, "claude-sonnet", modelFlag.GetDefault())
	assert.Equal(t, "Model to use", modelFlag.GetHelp())

	// Find format flag and check enum
	var formatFlag Flag
	for _, f := range cmd.flags {
		if f.GetName() == "format" {
			formatFlag = f
			break
		}
	}
	assert.NotNil(t, formatFlag)
	assert.Equal(t, []string{"json", "text", "yaml"}, formatFlag.GetEnum())
}

func TestBindFlagsGeneric(t *testing.T) {
	app := New("test").Description("Test")
	cmd := app.Command("run").Description("Run")

	ParseFlags[RunFlags](cmd)

	var boundFlags *RunFlags
	cmd.Run(func(ctx *Context) error {
		var err error
		boundFlags, err = BindFlags[RunFlags](ctx)
		return err
	})

	// Test with custom values
	err := app.RunArgs([]string{"run", "--model", "gpt-4", "-t", "0.9", "-v", "--count", "10"})
	assert.NoError(t, err)
	assert.NotNil(t, boundFlags)
	assert.Equal(t, "gpt-4", boundFlags.Model)
	assert.InDelta(t, 0.9, boundFlags.Temp, 0.001)
	assert.True(t, boundFlags.Verbose)
	assert.Equal(t, 10, boundFlags.Count)
	assert.Equal(t, "text", boundFlags.Format) // default
}

func TestBindFlagsDefaults(t *testing.T) {
	app := New("test").Description("Test")
	cmd := app.Command("run").Description("Run")

	ParseFlags[RunFlags](cmd)

	var boundFlags *RunFlags
	cmd.Run(func(ctx *Context) error {
		var err error
		boundFlags, err = BindFlags[RunFlags](ctx)
		return err
	})

	// Run with no flags - should use defaults
	err := app.RunArgs([]string{"run"})
	assert.NoError(t, err)
	assert.NotNil(t, boundFlags)
	assert.Equal(t, "claude-sonnet", boundFlags.Model)
	assert.InDelta(t, 0.7, boundFlags.Temp, 0.001)
	assert.False(t, boundFlags.Verbose)
	assert.Equal(t, 5, boundFlags.Count)
	assert.Equal(t, "text", boundFlags.Format)
}

func TestRequireFlagsMiddleware(t *testing.T) {
	app := New("test").Description("Test")
	app.Command("run").
		Description("Run").
		Flags(&StringFlag{Name: "config"}).
		Use(RequireFlags("config")).
		Run(func(ctx *Context) error {
			return nil
		})

	// Missing required flag
	err := app.RunArgs([]string{"run"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "required flag not set")

	// With required flag
	err = app.RunArgs([]string{"run", "--config", "test.yaml"})
	assert.NoError(t, err)
}

func TestRequireInteractiveMiddleware(t *testing.T) {
	app := New("test").Description("Test")
	app.isInteractive = false // Force non-interactive
	app.Command("run").
		Description("Run").
		Use(RequireInteractive()).
		Run(func(ctx *Context) error {
			return nil
		})

	err := app.RunArgs([]string{"run"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "interactive terminal")
}

func TestLoggerMiddleware(t *testing.T) {
	var stderr bytes.Buffer

	app := New("test").Description("Test")
	app.stderr = &stderr
	app.Use(Logger())
	app.Command("run").Description("Run").Run(func(ctx *Context) error {
		return nil
	})

	err := app.RunArgs([]string{"run"})
	assert.NoError(t, err)

	output := stderr.String()
	assert.Contains(t, output, "Running: run")
	assert.Contains(t, output, "Done: run")
}

func TestLoggerMiddlewareWithError(t *testing.T) {
	var stderr bytes.Buffer

	app := New("test").Description("Test")
	app.stderr = &stderr
	app.Use(Logger())
	app.Command("run").Description("Run").Run(func(ctx *Context) error {
		return Error("test error")
	})

	err := app.RunArgs([]string{"run"})
	assert.Error(t, err)

	output := stderr.String()
	assert.Contains(t, output, "Running: run")
	assert.Contains(t, output, "Failed: run")
}

func TestToolSchemaFromStruct(t *testing.T) {
	type CreateFileParams struct {
		Path    string `flag:"path,p" help:"File path" required:"true"`
		Content string `flag:"content,c" help:"File content" required:"true"`
		Force   bool   `flag:"force,f" help:"Overwrite existing file"`
	}

	schema := GenerateToolSchemaFromStruct[CreateFileParams]("create-file", "Create a file")

	assert.Equal(t, "create-file", schema.Name)
	assert.Equal(t, "Create a file", schema.Description)

	// Check parameters
	assert.Contains(t, schema.Parameters, "path")
	assert.Equal(t, "string", schema.Parameters["path"].Type)
	assert.Equal(t, "File path", schema.Parameters["path"].Description)

	assert.Contains(t, schema.Parameters, "content")
	assert.Equal(t, "string", schema.Parameters["content"].Type)

	assert.Contains(t, schema.Parameters, "force")
	assert.Equal(t, "boolean", schema.Parameters["force"].Type)

	// Check required
	assert.Contains(t, schema.Required, "path")
	assert.Contains(t, schema.Required, "content")
	assert.NotContains(t, schema.Required, "force")
}

func TestToolSchemaWithFlags(t *testing.T) {
	app := New("agent").Description("AI Agent")
	app.Command("run").
		Description("Run a prompt").
		Tool().
		Flags(
			&StringFlag{Name: "model", Help: "Model to use", Value: "claude-sonnet"},
			&Float64Flag{Name: "temperature", Help: "Temperature", Value: 0.7},
			&BoolFlag{Name: "stream", Help: "Stream output"},
		).
		Run(func(ctx *Context) error { return nil })

	schemas := app.GetToolSchemas()
	assert.Len(t, schemas, 1)

	schema := schemas[0]
	assert.Equal(t, "run", schema.Name)

	// Check parameter types
	assert.Equal(t, "string", schema.Parameters["model"].Type)
	assert.Equal(t, "claude-sonnet", schema.Parameters["model"].Default)

	assert.Equal(t, "number", schema.Parameters["temperature"].Type)

	assert.Equal(t, "boolean", schema.Parameters["stream"].Type)
}

func TestToolSchemaInGroup(t *testing.T) {
	app := New("agent").Description("AI Agent")
	files := app.Group("files").Description("File operations")
	files.Command("create").
		Description("Create a file").
		Tool().
		AddArg(&Arg{Name: "path", Required: true}).
		Run(func(ctx *Context) error { return nil })

	schemas := app.GetToolSchemas()
	assert.Len(t, schemas, 1)

	// Should be prefixed with group name
	assert.Equal(t, "files:create", schemas[0].Name)
}

func TestToolsJSON(t *testing.T) {
	app := New("agent").Description("AI Agent")
	app.Command("read").
		Description("Read a file").
		Tool().
		AddArg(&Arg{Name: "path", Required: true}).
		Run(func(ctx *Context) error { return nil })

	jsonStr, err := app.ToolsJSON()
	assert.NoError(t, err)
	assert.Contains(t, jsonStr, "read")
	assert.Contains(t, jsonStr, "path")
}

func TestContextNArg(t *testing.T) {
	var narg int
	var args []string

	app := New("test").Description("Test")
	app.Command("run").Description("Run").Run(func(ctx *Context) error {
		narg = ctx.NArg()
		args = ctx.Args()
		return nil
	})

	app.RunArgs([]string{"run", "a", "b", "c"})
	assert.Equal(t, 3, narg)
	assert.Equal(t, []string{"a", "b", "c"}, args)
}

func TestContextFlagTypes(t *testing.T) {
	app := New("test").Description("Test")
	app.Command("run").
		Description("Run").
		Flags(
			&StringFlag{Name: "str"},
			&IntFlag{Name: "num"},
			&Float64Flag{Name: "float"},
			&BoolFlag{Name: "bool"},
		).
		Run(func(ctx *Context) error {
			return nil
		})

	var str string
	var num int
	var num64 int64
	var flt float64
	var b bool

	app.Command("run").Run(func(ctx *Context) error {
		str = ctx.String("str")
		num = ctx.Int("num")
		num64 = ctx.Int64("num")
		flt = ctx.Float64("float")
		b = ctx.Bool("bool")
		return nil
	})

	err := app.RunArgs([]string{"run", "--str", "hello", "--num", "42", "--float", "3.14", "--bool"})
	assert.NoError(t, err)
	assert.Equal(t, "hello", str)
	assert.Equal(t, 42, num)
	assert.Equal(t, int64(42), num64)
	assert.InDelta(t, 3.14, flt, 0.001)
	assert.True(t, b)
}

func TestExitError(t *testing.T) {
	err := Exit(42)
	assert.Equal(t, 42, GetExitCode(err))

	exitErr, ok := err.(*ExitError)
	assert.True(t, ok)
	assert.Equal(t, 42, exitErr.Code)
}

func TestHiddenCommand(t *testing.T) {
	var buf bytes.Buffer

	app := New("test").Description("Test")
	app.stdout = &buf
	app.Command("visible").Description("Visible command").Run(func(ctx *Context) error { return nil })
	app.Command("hidden").Description("Hidden command").Hidden().Run(func(ctx *Context) error { return nil })

	app.RunArgs([]string{"help"})

	output := buf.String()
	assert.Contains(t, output, "visible")
	// Hidden command should still exist but help display logic would hide it
	// (Note: current implementation doesn't filter hidden in showHelp - could be added)
}

func TestDeprecatedCommand(t *testing.T) {
	var buf bytes.Buffer

	app := New("test").Description("Test")
	app.stdout = &buf
	app.Command("old").
		Description("Old command").
		Deprecated("Use 'new' instead").
		Run(func(ctx *Context) error { return nil })

	// Command should still work
	err := app.RunArgs([]string{"old"})
	assert.NoError(t, err)

	// Help should show deprecation
	app.RunArgs([]string{"old", "--help"})
	output := buf.String()
	assert.Contains(t, output, "DEPRECATED")
	assert.Contains(t, output, "Use 'new' instead")
}

func TestLongDescription(t *testing.T) {
	var buf bytes.Buffer

	app := New("test").Description("Test")
	app.stdout = &buf
	app.Command("run").
		Description("Run something").
		Long("This is a longer description that provides more detail about what the command does.").
		Run(func(ctx *Context) error { return nil })

	app.RunArgs([]string{"run", "--help"})
	output := buf.String()
	assert.Contains(t, output, "longer description")
}

func TestHiddenFlag(t *testing.T) {
	var buf bytes.Buffer

	app := New("test").Description("Test")
	app.stdout = &buf
	app.Command("run").
		Description("Run").
		Flags(
			&StringFlag{Name: "visible", Help: "Visible flag"},
			&StringFlag{Name: "hidden", Help: "Hidden flag", Hidden: true},
		).
		Run(func(ctx *Context) error { return nil })

	app.RunArgs([]string{"run", "--help"})
	output := buf.String()
	assert.Contains(t, output, "visible")
	assert.NotContains(t, output, "Hidden flag")
}

func TestIsTTY(t *testing.T) {
	// This test just verifies the function exists and doesn't panic
	// Actual TTY detection depends on environment
	_ = IsTTY()
	_ = IsPiped()
}

func TestEnvVarForFlag(t *testing.T) {
	// Set env var
	t.Setenv("TEST_API_KEY", "secret-key")

	var key string

	app := New("test").Description("Test")
	app.Command("run").
		Description("Run").
		Flags(&StringFlag{Name: "api-key", EnvVar: "TEST_API_KEY", Required: true}).
		Run(func(ctx *Context) error {
			key = ctx.String("api-key")
			return nil
		})

	// Should use env var when flag not provided
	err := app.RunArgs([]string{"run"})
	assert.NoError(t, err)
	assert.Equal(t, "secret-key", key)
}

func TestErrorf(t *testing.T) {
	err := Errorf("failed with value: %d", 42)
	assert.Contains(t, err.Error(), "failed with value: 42")
}

// Test app.Test() infrastructure
func TestAppTestInfrastructure(t *testing.T) {
	app := New("test").Description("Test app")
	app.Command("echo").
		Description("Echo input").
		Args("message").
		Run(func(ctx *Context) error {
			ctx.Printf("Echo: %s\n", ctx.Arg(0))
			return nil
		})

	result := app.Test(t, TestArgs("echo", "hello"))

	assert.True(t, result.Success())
	assert.Equal(t, 0, result.ExitCode)
	assert.Contains(t, result.Stdout, "Echo: hello")
	assert.True(t, result.Contains("hello"))
}

func TestAppTestWithEnv(t *testing.T) {
	app := New("test").Description("Test")
	app.Command("run").
		Description("Run").
		Flags(&StringFlag{Name: "key", EnvVar: "TEST_KEY", Required: true}).
		Run(func(ctx *Context) error {
			ctx.Printf("Key: %s\n", ctx.String("key"))
			return nil
		})

	result := app.Test(t,
		TestArgs("run"),
		TestEnv("TEST_KEY", "secret-value"),
	)

	assert.True(t, result.Success())
	assert.Contains(t, result.Stdout, "Key: secret-value")
}

func TestAppTestFailure(t *testing.T) {
	app := New("test").Description("Test")
	app.Command("fail").Description("Fail").Run(func(ctx *Context) error {
		return Error("intentional failure")
	})

	result := app.Test(t, TestArgs("fail"))

	assert.True(t, result.Failed())
	assert.Equal(t, 1, result.ExitCode)
	assert.NotNil(t, result.Err)
}

// Test hidden commands filtering
func TestHiddenCommandsNotInHelp(t *testing.T) {
	var buf bytes.Buffer

	app := New("test").Description("Test")
	app.stdout = &buf
	app.Command("public").Description("Public command").Run(func(ctx *Context) error { return nil })
	app.Command("secret").Description("Secret command").Hidden().Run(func(ctx *Context) error { return nil })

	app.RunArgs([]string{"help"})

	output := buf.String()
	assert.Contains(t, output, "public")
	assert.NotContains(t, output, "secret")
}

// Test Interactive/NonInteractive dispatch
func TestInteractiveDispatch(t *testing.T) {
	var handlerCalled string

	app := New("test").Description("Test")
	app.ForceInteractive(true) // Force interactive mode for testing

	app.Command("run").
		Description("Run").
		Interactive(func(ctx *Context) error {
			handlerCalled = "interactive"
			return nil
		}).
		NonInteractive(func(ctx *Context) error {
			handlerCalled = "non-interactive"
			return nil
		})

	err := app.RunArgs([]string{"run"})
	assert.NoError(t, err)
	assert.Equal(t, "interactive", handlerCalled)
}

func TestNonInteractiveDispatch(t *testing.T) {
	var handlerCalled string

	app := New("test").Description("Test")
	app.ForceInteractive(false) // Force non-interactive mode for testing

	app.Command("run").
		Description("Run").
		Interactive(func(ctx *Context) error {
			handlerCalled = "interactive"
			return nil
		}).
		NonInteractive(func(ctx *Context) error {
			handlerCalled = "non-interactive"
			return nil
		})

	err := app.RunArgs([]string{"run"})
	assert.NoError(t, err)
	assert.Equal(t, "non-interactive", handlerCalled)
}

func TestFallbackToDefaultHandler(t *testing.T) {
	var handlerCalled string

	app := New("test").Description("Test")
	app.isInteractive = true

	// Only set default handler, no interactive/non-interactive
	app.Command("run").
		Description("Run").
		Run(func(ctx *Context) error {
			handlerCalled = "default"
			return nil
		})

	err := app.RunArgs([]string{"run"})
	assert.NoError(t, err)
	assert.Equal(t, "default", handlerCalled)
}

// Test ArgsRange validation
func TestArgsRange(t *testing.T) {
	app := New("test").Description("Test")
	app.Command("add").
		Description("Add items").
		ArgsRange(1, 3).
		Run(func(ctx *Context) error { return nil })

	// Too few args
	err := app.RunArgs([]string{"add"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least 1 argument")

	// Correct number
	err = app.RunArgs([]string{"add", "a"})
	assert.NoError(t, err)

	err = app.RunArgs([]string{"add", "a", "b", "c"})
	assert.NoError(t, err)

	// Too many args
	err = app.RunArgs([]string{"add", "a", "b", "c", "d"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at most 3 argument")
}

func TestExactArgs(t *testing.T) {
	app := New("test").Description("Test")
	app.Command("pair").
		Description("Pair two items").
		ExactArgs(2).
		Run(func(ctx *Context) error { return nil })

	// Too few
	err := app.RunArgs([]string{"pair", "a"})
	assert.Error(t, err)

	// Exact
	err = app.RunArgs([]string{"pair", "a", "b"})
	assert.NoError(t, err)

	// Too many
	err = app.RunArgs([]string{"pair", "a", "b", "c"})
	assert.Error(t, err)
}

func TestNoArgs(t *testing.T) {
	app := New("test").Description("Test")
	app.Command("status").
		Description("Show status").
		NoArgs().
		Run(func(ctx *Context) error { return nil })

	// No args - ok
	err := app.RunArgs([]string{"status"})
	assert.NoError(t, err)

	// With args - error
	err = app.RunArgs([]string{"status", "extra"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "accepts no arguments")
}

func TestWithValidation(t *testing.T) {
	app := New("test").Description("Test")
	app.Command("set").
		Description("Set value").
		Args("value").
		Validate(func(ctx *Context) error {
			if ctx.Arg(0) == "invalid" {
				return Error("invalid value")
			}
			return nil
		}).
		Run(func(ctx *Context) error { return nil })

	err := app.RunArgs([]string{"set", "valid"})
	assert.NoError(t, err)

	err = app.RunArgs([]string{"set", "invalid"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid value")
}

// Test global flags
func TestGlobalFlags(t *testing.T) {
	var verbose bool
	var config string

	app := New("test").Description("Test")
	app.GlobalFlags(
		&BoolFlag{Name: "verbose", Short: "v", Help: "Verbose output"},
		&StringFlag{Name: "config", Short: "c", Help: "Config file"},
	)

	app.Command("run").Description("Run").Run(func(ctx *Context) error {
		verbose = ctx.Bool("verbose")
		config = ctx.String("config")
		return nil
	})

	// Use global flags
	err := app.RunArgs([]string{"run", "-v", "-c", "myconfig.yaml"})
	assert.NoError(t, err)
	assert.True(t, verbose)
	assert.Equal(t, "myconfig.yaml", config)
}

func TestGlobalFlagsInHelp(t *testing.T) {
	var buf bytes.Buffer

	app := New("test").Description("Test")
	app.stdout = &buf
	app.GlobalFlags(&BoolFlag{Name: "verbose", Short: "v", Help: "Verbose output"})

	app.Command("run").
		Description("Run something").
		Flags(&StringFlag{Name: "output", Short: "o", Help: "Output file"}).
		Run(func(ctx *Context) error { return nil })

	app.RunArgs([]string{"run", "--help"})

	output := buf.String()
	assert.Contains(t, output, "Flags:")
	assert.Contains(t, output, "output")
	assert.Contains(t, output, "Global Flags:")
	assert.Contains(t, output, "verbose")
}

// Test shell completions
func TestBashCompletion(t *testing.T) {
	var buf bytes.Buffer

	app := New("myapp").Description("My application")
	app.Command("run").Description("Run something").Run(func(ctx *Context) error { return nil })
	app.Command("build").Description("Build something").Run(func(ctx *Context) error { return nil })

	err := app.GenerateBashCompletion(&buf)
	assert.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "myapp")
	assert.Contains(t, output, "run")
	assert.Contains(t, output, "build")
	assert.Contains(t, output, "complete -F")
}

func TestZshCompletion(t *testing.T) {
	var buf bytes.Buffer

	app := New("myapp").Description("My application")
	app.Command("run").Description("Run something").Run(func(ctx *Context) error { return nil })

	err := app.GenerateZshCompletion(&buf)
	assert.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "#compdef myapp")
	assert.Contains(t, output, "run")
}

func TestFishCompletion(t *testing.T) {
	var buf bytes.Buffer

	app := New("myapp").Description("My application")
	app.Command("run").
		Description("Run something").
		Flags(&BoolFlag{Name: "verbose", Short: "v", Help: "Be verbose"}).
		Run(func(ctx *Context) error { return nil })

	err := app.GenerateFishCompletion(&buf)
	assert.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "complete -c myapp")
	assert.Contains(t, output, "run")
	assert.Contains(t, output, "verbose")
}

func TestAddCompletionCommand(t *testing.T) {
	app := New("myapp").Description("My app")
	app.AddCompletionCommand()

	assert.Contains(t, app.commands, "completion")

	var buf bytes.Buffer
	app.stdout = &buf

	err := app.RunArgs([]string{"completion", "bash"})
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "complete -F")
}

// Test conversation basics
func TestConversationHistory(t *testing.T) {
	ctx := &Context{
		stdin:       strings.NewReader(""),
		stdout:      &bytes.Buffer{},
		stderr:      &bytes.Buffer{},
		interactive: false,
		flags:       make(map[string]any),
	}

	conv := NewConversation(ctx)
	conv.System("You are a helpful assistant")
	conv.AddMessage("user", "Hello")
	conv.AddMessage("assistant", "Hi there!")

	history := conv.History()
	assert.Len(t, history, 3)
	assert.Equal(t, "system", history[0].Role)
	assert.Equal(t, "user", history[1].Role)
	assert.Equal(t, "assistant", history[2].Role)
}

func TestConversationReply(t *testing.T) {
	var buf bytes.Buffer

	ctx := &Context{
		stdin:       strings.NewReader(""),
		stdout:      &buf,
		stderr:      &bytes.Buffer{},
		interactive: false,
		flags:       make(map[string]any),
	}

	conv := NewConversation(ctx)
	conv.Reply("", "Hello, world!")

	assert.Contains(t, buf.String(), "Hello, world!")
	assert.Len(t, conv.History(), 1)
	assert.Equal(t, "assistant", conv.History()[0].Role)
}

// Test TestResult helpers
func TestTestResultHelpers(t *testing.T) {
	result := &TestResult{
		ExitCode: 0,
		Stdout:   "hello world",
		Stderr:   "warning: something",
		Events: []map[string]any{
			{"type": "start"},
			{"type": "end"},
		},
	}

	assert.True(t, result.Success())
	assert.False(t, result.Failed())
	assert.True(t, result.Contains("hello"))
	assert.True(t, result.StderrContains("warning"))
	assert.Equal(t, 2, result.EventCount())
	assert.NotNil(t, result.GetEvent(0))
	assert.Nil(t, result.GetEvent(10))
}

func TestTestApp(t *testing.T) {
	app := TestApp("test")
	assert.False(t, app.isInteractive)
	assert.NotNil(t, app.stdin)
	assert.NotNil(t, app.stdout)
	assert.NotNil(t, app.stderr)
}
