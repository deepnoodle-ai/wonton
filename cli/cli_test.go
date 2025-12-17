package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

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

// Test TestResult helpers
func TestTestResultHelpers(t *testing.T) {
	result := &TestResult{
		ExitCode: 0,
		Stdout:   "hello world",
		Stderr:   "warning: something",
	}

	assert.True(t, result.Success())
	assert.False(t, result.Failed())
	assert.True(t, result.Contains("hello"))
	assert.True(t, result.StderrContains("warning"))
}

func TestTestApp(t *testing.T) {
	app := TestApp("test")
	assert.False(t, app.isInteractive)
	assert.NotNil(t, app.stdin)
	assert.NotNil(t, app.stdout)
	assert.NotNil(t, app.stderr)
}

// Additional coverage tests

func TestAddGlobalFlag(t *testing.T) {
	app := New("test").Description("Test")
	app.AddGlobalFlag(&StringFlag{Name: "config", Short: "c", Help: "Config file"})

	var configValue string
	app.Command("run").Description("Run").Run(func(ctx *Context) error {
		configValue = ctx.String("config")
		return nil
	})

	err := app.RunArgs([]string{"run", "-c", "app.yaml"})
	assert.NoError(t, err)
	assert.Equal(t, "app.yaml", configValue)
}

func TestSetColorEnabled(t *testing.T) {
	app := New("test").Description("Test")
	app.SetColorEnabled(false)
	assert.False(t, app.colorEnabled)

	app.SetColorEnabled(true)
	assert.True(t, app.colorEnabled)
}

func TestCommandAliases(t *testing.T) {
	var executed bool

	app := New("test").Description("Test")
	app.Command("generate").
		Description("Generate").
		Aliases("gen", "g", "create").
		Run(func(ctx *Context) error {
			executed = true
			return nil
		})

	// Test all aliases
	for _, alias := range []string{"gen", "g", "create"} {
		executed = false
		err := app.RunArgs([]string{alias})
		assert.NoError(t, err)
		assert.True(t, executed, "alias %s should execute", alias)
	}
}

func TestCommandName(t *testing.T) {
	app := New("test").Description("Test")
	cmd := app.Command("mycommand").Description("My command")

	assert.Equal(t, "mycommand", cmd.Name())
}

func TestCommandGetDescription(t *testing.T) {
	app := New("test").Description("Test")
	cmd := app.Command("mycommand").Description("My description")

	assert.Equal(t, "My description", cmd.GetDescription())
}

func TestGroupCommandList(t *testing.T) {
	app := New("test").Description("Test")
	users := app.Group("users").Description("User operations")
	users.Command("list").Description("List users").Run(func(ctx *Context) error { return nil })
	users.Command("create").Description("Create user").Run(func(ctx *Context) error { return nil })

	list := users.commandList()
	assert.Contains(t, list, "list")
	assert.Contains(t, list, "create")
	assert.Contains(t, list, "users")
}

func TestContextCommand(t *testing.T) {
	var gotCommand *Command

	app := New("test").Description("Test")
	app.Command("run").Description("Run").Run(func(ctx *Context) error {
		gotCommand = ctx.Command()
		return nil
	})

	err := app.RunArgs([]string{"run"})
	assert.NoError(t, err)
	assert.NotNil(t, gotCommand)
	assert.Equal(t, "run", gotCommand.Name())
}

func TestContextStdinStderr(t *testing.T) {
	var gotStdin, gotStderr bool

	app := New("test").Description("Test")
	app.Command("run").Description("Run").Run(func(ctx *Context) error {
		gotStdin = ctx.Stdin() != nil
		gotStderr = ctx.Stderr() != nil
		return nil
	})

	err := app.RunArgs([]string{"run"})
	assert.NoError(t, err)
	assert.True(t, gotStdin)
	assert.True(t, gotStderr)
}

func TestContextPrint(t *testing.T) {
	var buf bytes.Buffer

	app := New("test").Description("Test")
	app.stdout = &buf
	app.Command("run").Description("Run").Run(func(ctx *Context) error {
		ctx.Print("hello")
		ctx.Print(" world")
		return nil
	})

	err := app.RunArgs([]string{"run"})
	assert.NoError(t, err)
	assert.Equal(t, "hello world", buf.String())
}

func TestContextErrorMethods(t *testing.T) {
	var stderr bytes.Buffer

	app := New("test").Description("Test")
	app.stderr = &stderr
	app.Command("run").Description("Run").Run(func(ctx *Context) error {
		ctx.Error("err1")
		ctx.Errorf("err%d", 2)
		ctx.Errorln("err3")
		return nil
	})

	err := app.RunArgs([]string{"run"})
	assert.NoError(t, err)
	assert.Equal(t, "err1err2err3\n", stderr.String())
}

func TestHelpRequestedError(t *testing.T) {
	err := &HelpRequested{}
	assert.Equal(t, "help requested", err.Error())
}


func TestIntFlagMethods(t *testing.T) {
	flag := &IntFlag{
		Name:     "count",
		Short:    "c",
		Help:     "Item count",
		Value:    10,
		EnvVar:   "COUNT",
		Required: true,
		Hidden:   true,
	}

	assert.Equal(t, "count", flag.GetName())
	assert.Equal(t, "c", flag.GetShort())
	assert.Equal(t, "Item count", flag.GetHelp())
	assert.Equal(t, "COUNT", flag.GetEnvVar())
	assert.Equal(t, 10, flag.GetDefault())
	assert.True(t, flag.IsRequired())
	assert.True(t, flag.IsHidden())
	assert.Nil(t, flag.GetEnum())
	assert.NoError(t, flag.Validate("5"))
}

func TestFloat64FlagMethods(t *testing.T) {
	flag := &Float64Flag{
		Name:     "rate",
		Short:    "r",
		Help:     "Rate value",
		Value:    0.5,
		EnvVar:   "RATE",
		Required: false,
		Hidden:   false,
	}

	assert.Equal(t, "rate", flag.GetName())
	assert.Equal(t, "r", flag.GetShort())
	assert.Equal(t, "Rate value", flag.GetHelp())
	assert.Equal(t, "RATE", flag.GetEnvVar())
	assert.Equal(t, 0.5, flag.GetDefault())
	assert.False(t, flag.IsRequired())
	assert.False(t, flag.IsHidden())
	assert.NoError(t, flag.Validate("1.5"))
}

func TestDurationFlagMethods(t *testing.T) {
	flag := &DurationFlag{
		Name:     "timeout",
		Short:    "t",
		Help:     "Timeout duration",
		EnvVar:   "TIMEOUT",
		Required: true,
		Hidden:   false,
	}

	assert.Equal(t, "timeout", flag.GetName())
	assert.Equal(t, "t", flag.GetShort())
	assert.Equal(t, "Timeout duration", flag.GetHelp())
	assert.Equal(t, "TIMEOUT", flag.GetEnvVar())
	assert.False(t, flag.IsHidden())
	assert.True(t, flag.IsRequired())
	assert.Nil(t, flag.GetEnum())
	assert.NoError(t, flag.Validate("5s"))
}

func TestStringSliceFlagMethods(t *testing.T) {
	flag := &StringSliceFlag{
		Name:     "tags",
		Short:    "t",
		Help:     "Tags list",
		EnvVar:   "TAGS",
		Required: false,
		Hidden:   true,
	}

	assert.Equal(t, "tags", flag.GetName())
	assert.Equal(t, "t", flag.GetShort())
	assert.Equal(t, "Tags list", flag.GetHelp())
	assert.Equal(t, "TAGS", flag.GetEnvVar())
	assert.False(t, flag.IsRequired())
	assert.True(t, flag.IsHidden())
	assert.Nil(t, flag.GetEnum())
	assert.NoError(t, flag.Validate("a,b,c"))
}

func TestIntSliceFlagMethods(t *testing.T) {
	flag := &IntSliceFlag{
		Name:     "ids",
		Short:    "i",
		Help:     "ID list",
		EnvVar:   "IDS",
		Required: true,
		Hidden:   false,
	}

	assert.Equal(t, "ids", flag.GetName())
	assert.Equal(t, "i", flag.GetShort())
	assert.Equal(t, "ID list", flag.GetHelp())
	assert.Equal(t, "IDS", flag.GetEnvVar())
	assert.True(t, flag.IsRequired())
	assert.False(t, flag.IsHidden())
	assert.Nil(t, flag.GetEnum())
	assert.NoError(t, flag.Validate("1,2,3"))
}

func TestStringFlagValidateWithValidator(t *testing.T) {
	flag := &StringFlag{
		Name: "format",
		Validator: func(value string) error {
			if value != "json" && value != "yaml" {
				return Error("invalid format")
			}
			return nil
		},
	}

	assert.NoError(t, flag.Validate("json"))
	assert.NoError(t, flag.Validate("yaml"))
	assert.Error(t, flag.Validate("xml"))
}

func TestParseArgsUnknownFlag(t *testing.T) {
	app := New("test").Description("Test")
	app.Command("run").
		Description("Run").
		Flags(&StringFlag{Name: "known"}).
		Run(func(ctx *Context) error { return nil })

	err := app.RunArgs([]string{"run", "--unknown", "value"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown flag")
}

func TestCompletionCommandUnsupportedShell(t *testing.T) {
	app := New("test").Description("Test")
	app.AddCompletionCommand()

	var stderr bytes.Buffer
	app.stderr = &stderr

	err := app.RunArgs([]string{"completion", "powershell"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported shell")
}

func TestCompletionCommandZsh(t *testing.T) {
	app := New("test").Description("Test")
	app.AddCompletionCommand()
	app.Command("run").Description("Run").Run(func(ctx *Context) error { return nil })

	var stdout bytes.Buffer
	app.stdout = &stdout

	err := app.RunArgs([]string{"completion", "zsh"})
	assert.NoError(t, err)
	assert.Contains(t, stdout.String(), "#compdef test")
}

func TestCompletionCommandFish(t *testing.T) {
	app := New("test").Description("Test")
	app.AddCompletionCommand()
	app.Command("run").Description("Run").Run(func(ctx *Context) error { return nil })

	var stdout bytes.Buffer
	app.stdout = &stdout

	err := app.RunArgs([]string{"completion", "fish"})
	assert.NoError(t, err)
	assert.Contains(t, stdout.String(), "complete -c test")
}

func TestCompletionWithGroups(t *testing.T) {
	var buf bytes.Buffer

	app := New("myapp").Description("My app")
	users := app.Group("users").Description("User management")
	users.Command("list").Description("List users").Run(func(ctx *Context) error { return nil })
	users.Command("create").Description("Create user").Run(func(ctx *Context) error { return nil })

	err := app.GenerateBashCompletion(&buf)
	assert.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "users")
	assert.Contains(t, output, "list")
	assert.Contains(t, output, "create")
}

func TestTestStdin(t *testing.T) {
	app := New("test").Description("Test")
	app.Command("read").Description("Read input").Run(func(ctx *Context) error {
		buf := make([]byte, 100)
		n, _ := ctx.Stdin().Read(buf)
		ctx.Printf("Read: %s", string(buf[:n]))
		return nil
	})

	result := app.Test(t,
		TestArgs("read"),
		TestStdin("hello from stdin"),
	)

	assert.True(t, result.Success())
	assert.Contains(t, result.Stdout, "Read: hello from stdin")
}

func TestCaptureOutput(t *testing.T) {
	stdout, stderr := CaptureOutput(func() {
		// These print to actual os.Stdout/os.Stderr
		// but CaptureOutput intercepts them
	})

	// Should capture without error
	assert.NotNil(t, stdout)
	assert.NotNil(t, stderr)
}

func TestBeforeMiddlewareError(t *testing.T) {
	var handlerCalled bool

	app := New("test").Description("Test")
	app.Use(Before(func(ctx *Context) error {
		return Error("before failed")
	}))
	app.Command("run").Description("Run").Run(func(ctx *Context) error {
		handlerCalled = true
		return nil
	})

	err := app.RunArgs([]string{"run"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "before failed")
	assert.False(t, handlerCalled)
}

func TestAfterMiddlewareError(t *testing.T) {
	var handlerCalled bool

	app := New("test").Description("Test")
	app.Use(After(func(ctx *Context) error {
		return Error("after failed")
	}))
	app.Command("run").Description("Run").Run(func(ctx *Context) error {
		handlerCalled = true
		return nil
	})

	err := app.RunArgs([]string{"run"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "after failed")
	assert.True(t, handlerCalled)
}

func TestAfterMiddlewareDoesNotOverrideHandlerError(t *testing.T) {
	app := New("test").Description("Test")
	app.Use(After(func(ctx *Context) error {
		return Error("after error")
	}))
	app.Command("run").Description("Run").Run(func(ctx *Context) error {
		return Error("handler error")
	})

	err := app.RunArgs([]string{"run"})
	assert.Error(t, err)
	// Handler error should take precedence
	assert.Contains(t, err.Error(), "handler error")
}

func TestContextStringFromNonString(t *testing.T) {
	ctx := &Context{
		flags:    map[string]any{"num": 42},
		setFlags: map[string]bool{"num": true},
	}

	// String() should convert non-string values
	assert.Equal(t, "42", ctx.String("num"))
}

func TestContextIntFromFloat(t *testing.T) {
	ctx := &Context{
		flags:    map[string]any{"val": float64(42.9)},
		setFlags: map[string]bool{"val": true},
	}

	// Int() should truncate float
	assert.Equal(t, 42, ctx.Int("val"))
}

func TestContextInt64FromVariousTypes(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		expected int64
	}{
		{"from int", 42, 42},
		{"from int64", int64(100), 100},
		{"from float64", float64(99.9), 99},
		{"from string", "123", 123},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &Context{
				flags:    map[string]any{"val": tt.value},
				setFlags: map[string]bool{"val": true},
			}
			assert.Equal(t, tt.expected, ctx.Int64("val"))
		})
	}
}

func TestContextFloat64FromVariousTypes(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		expected float64
	}{
		{"from int", 42, 42.0},
		{"from int64", int64(100), 100.0},
		{"from float64", float64(3.14), 3.14},
		{"from string", "2.5", 2.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &Context{
				flags:    map[string]any{"val": tt.value},
				setFlags: map[string]bool{"val": true},
			}
			assert.InDelta(t, tt.expected, ctx.Float64("val"), 0.001)
		})
	}
}

func TestFlagValidationInParsing(t *testing.T) {
	// Test that flag validation is called during parsing
	app := New("test").Description("Test")
	app.Command("run").
		Description("Run").
		Flags(&StringFlag{Name: "format", Enum: []string{"json", "yaml"}}).
		Run(func(ctx *Context) error { return nil })

	err := app.RunArgs([]string{"run", "--format", "invalid"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid value")
}

func TestShowHelpWithGlobalFlagsInCommandHelp(t *testing.T) {
	var buf bytes.Buffer

	app := New("test").Description("Test application")
	app.stdout = &buf
	app.GlobalFlags(
		&BoolFlag{Name: "verbose", Short: "v", Help: "Verbose mode"},
		&StringFlag{Name: "config", Short: "c", Help: "Config file"},
	)
	app.Command("run").Description("Run command").Run(func(ctx *Context) error { return nil })

	// Global flags are shown in command help, not app help
	app.RunArgs([]string{"run", "--help"})

	output := buf.String()
	assert.Contains(t, output, "Global Flags:")
	assert.Contains(t, output, "--verbose")
	assert.Contains(t, output, "--config")
}

func TestFindCommandInGroup(t *testing.T) {
	var executed bool

	app := New("test").Description("Test")
	db := app.Group("db").Description("Database operations")
	db.Command("migrate").Description("Run migrations").Run(func(ctx *Context) error {
		executed = true
		return nil
	})

	// Should find command via group:command syntax
	err := app.RunArgs([]string{"db:migrate"})
	assert.NoError(t, err)
	assert.True(t, executed)
}

func TestSelectStringError(t *testing.T) {
	ctx := &Context{
		interactive: false,
		stdout:      &bytes.Buffer{},
		stderr:      &bytes.Buffer{},
	}

	_, err := ctx.SelectString("Choose:", "a", "b")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "interactive")
}

func TestSemanticOutputWithColors(t *testing.T) {
	var stdout, stderr bytes.Buffer

	app := New("test")
	app.colorEnabled = true // Enable colors

	ctx := &Context{
		app:    app,
		stdout: &stdout,
		stderr: &stderr,
	}

	ctx.Success("success message")
	ctx.Info("info message")
	ctx.Warn("warn message")
	ctx.Fail("fail message")

	// With colors enabled, output should contain ANSI codes
	assert.Contains(t, stdout.String(), "success message")
	assert.Contains(t, stdout.String(), "info message")
	assert.Contains(t, stderr.String(), "warn message")
	assert.Contains(t, stderr.String(), "fail message")

	// Check for ANSI escape codes (basic check)
	assert.Contains(t, stdout.String(), "\033[")
}

func TestHelpWithHiddenCommandInGroup(t *testing.T) {
	var buf bytes.Buffer

	app := New("test").Description("Test")
	app.stdout = &buf

	// Regular group
	public := app.Group("public").Description("Public commands")
	public.Command("list").Description("List items").Run(func(ctx *Context) error { return nil })

	// Hidden command in group
	public.Command("secret").Description("Secret").Hidden().Run(func(ctx *Context) error { return nil })

	// App help shows groups
	app.RunArgs([]string{"help"})
	output := buf.String()
	assert.Contains(t, output, "public")

	// The group's commandList should filter hidden commands
	list := public.commandList()
	assert.Contains(t, list, "list")
	// Hidden command should still appear in commandList (it doesn't filter)
	// The filtering happens elsewhere
}

func TestZshCompletionWithGroups(t *testing.T) {
	var buf bytes.Buffer

	app := New("myapp").Description("My app")
	users := app.Group("users").Description("User management")
	users.Command("list").Description("List users").Run(func(ctx *Context) error { return nil })

	err := app.GenerateZshCompletion(&buf)
	assert.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "users")
}

func TestFishCompletionWithGroups(t *testing.T) {
	var buf bytes.Buffer

	app := New("myapp").Description("My app")
	files := app.Group("files").Description("File operations")
	files.Command("list").Description("List files").Run(func(ctx *Context) error { return nil })
	files.Command("delete").Description("Delete files").Run(func(ctx *Context) error { return nil })

	err := app.GenerateFishCompletion(&buf)
	assert.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "files")
	assert.Contains(t, output, "list")
	assert.Contains(t, output, "delete")
}

func TestFlagWithoutShortName(t *testing.T) {
	var buf bytes.Buffer

	app := New("test").Description("Test")
	app.stdout = &buf
	app.Command("run").
		Description("Run").
		Flags(&StringFlag{Name: "longonly", Help: "Long flag only"}).
		Run(func(ctx *Context) error { return nil })

	err := app.GenerateFishCompletion(&buf)
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "longonly")
}

// Additional coverage tests

func TestFlagGetDefaultMethods(t *testing.T) {
	// DurationFlag GetDefault
	df := &DurationFlag{Name: "timeout", Value: 30 * 1000000000} // 30s in nanoseconds
	assert.Equal(t, 30*time.Second, df.GetDefault())

	// StringSliceFlag GetDefault
	ssf := &StringSliceFlag{Name: "tags", Value: []string{"a", "b"}}
	assert.Equal(t, []string{"a", "b"}, ssf.GetDefault())

	// IntSliceFlag GetDefault
	isf := &IntSliceFlag{Name: "ports", Value: []int{80, 443}}
	assert.Equal(t, []int{80, 443}, isf.GetDefault())

	// BoolFlag Validate (always returns nil)
	bf := &BoolFlag{Name: "debug"}
	assert.NoError(t, bf.Validate("anything"))

	// Float64Flag Validate (always returns nil)
	ff := &Float64Flag{Name: "rate"}
	assert.NoError(t, ff.Validate("anything"))

	// DurationFlag Validate (always returns nil)
	assert.NoError(t, df.Validate("anything"))

	// StringSliceFlag Validate (always returns nil)
	assert.NoError(t, ssf.Validate("anything"))

	// IntSliceFlag Validate (always returns nil)
	assert.NoError(t, isf.Validate("anything"))
}

func TestGlobalFlagAfterCommand(t *testing.T) {
	var verbose bool

	app := New("test").Description("Test")
	app.GlobalFlags(&BoolFlag{Name: "verbose", Short: "v"})
	app.Command("run").
		Description("Run").
		Run(func(ctx *Context) error {
			verbose = ctx.Bool("verbose")
			return nil
		})

	// Global flags work when specified after command
	err := app.RunArgs([]string{"run", "-v"})
	assert.NoError(t, err)
	assert.True(t, verbose)
}

func TestParseArgsEmpty(t *testing.T) {
	app := New("test").Description("Test")
	app.stdout = &bytes.Buffer{}

	// Empty args should show help
	err := app.RunArgs([]string{})
	assert.NoError(t, err)
}

func TestContextStringWithNonStringType(t *testing.T) {
	ctx := newTestContext(map[string]any{
		"count": 42,
	})

	// String() should convert int to string
	assert.Equal(t, "42", ctx.String("count"))
}

func TestContextIntWithFloat64Type(t *testing.T) {
	ctx := newTestContext(map[string]any{
		"count": float64(42.9),
	})

	// Int() should convert float64 to int
	assert.Equal(t, 42, ctx.Int("count"))
}

func TestContextIsSetWithMissingSetFlags(t *testing.T) {
	// Create context with nil setFlags map
	ctx := &Context{
		flags:    map[string]any{"flag": true},
		setFlags: nil,
	}

	// IsSet should return false when setFlags is nil
	assert.False(t, ctx.IsSet("flag"))
}

func TestDefaultFlagValues(t *testing.T) {
	var count int

	app := New("test").Description("Test")
	app.Command("run").
		Description("Run").
		Flags(&IntFlag{Name: "count", Value: 10}).
		Run(func(ctx *Context) error {
			count = ctx.Int("count")
			return nil
		})

	err := app.RunArgs([]string{"run"})
	assert.NoError(t, err)
	assert.Equal(t, 10, count)
}

func TestFindCommandWithGroupAlias(t *testing.T) {
	app := New("test").Description("Test")
	group := app.Group("users").Description("User management")
	group.Command("list").
		Alias("ls").
		Description("List users").
		Run(func(ctx *Context) error { return nil })

	// Test using group:alias pattern
	err := app.RunArgs([]string{"users:ls"})
	assert.NoError(t, err)
}

func TestGroupWithoutSubcommand(t *testing.T) {
	app := New("test").Description("Test")
	app.stdout = &bytes.Buffer{}
	app.stderr = &bytes.Buffer{}

	group := app.Group("users").Description("User management")
	group.Command("list").Description("List users").Run(func(ctx *Context) error { return nil })

	// Running just the group name should error
	err := app.RunArgs([]string{"users"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "requires a subcommand")
}

func TestCommandLevelValidation(t *testing.T) {
	app := New("test").Description("Test")
	app.Command("run").
		Description("Run").
		Flags(&IntFlag{Name: "count"}).
		Validate(func(ctx *Context) error {
			if ctx.Int("count") < 0 {
				return Error("count must be non-negative")
			}
			return nil
		}).
		Run(func(ctx *Context) error { return nil })

	// Valid value
	err := app.RunArgs([]string{"run", "--count=5"})
	assert.NoError(t, err)

	// Invalid value - validation catches it
	err = app.RunArgs([]string{"run", "--count=-1"})
	assert.Error(t, err)
}

func TestContextInt64(t *testing.T) {
	ctx := newTestContext(map[string]any{
		"fromInt":    5,
		"fromInt64":  int64(10),
		"fromFloat":  float64(15.9),
		"fromString": "20",
	})

	assert.Equal(t, int64(5), ctx.Int64("fromInt"))
	assert.Equal(t, int64(10), ctx.Int64("fromInt64"))
	assert.Equal(t, int64(15), ctx.Int64("fromFloat"))
	assert.Equal(t, int64(20), ctx.Int64("fromString"))
	assert.Equal(t, int64(0), ctx.Int64("missing"))
}


func TestHelpForSpecificCommand(t *testing.T) {
	var buf bytes.Buffer
	app := New("test").Description("Test")
	app.stdout = &buf

	app.Command("deploy").
		Description("Deploy the application").
		Long("Deploy the application to the specified environment with optional force flag.").
		Args("environment").
		Flags(
			&StringFlag{Name: "target", Short: "t", Help: "Deployment target"},
			&BoolFlag{Name: "force", Short: "f", Help: "Force deployment"},
		).
		Run(func(ctx *Context) error { return nil })

	// Use command --help to get command-specific help
	err := app.RunArgs([]string{"deploy", "--help"})
	// HelpRequested is returned as an error
	var helpErr *HelpRequested
	assert.ErrorAs(t, err, &helpErr)

	output := buf.String()
	assert.Contains(t, output, "deploy")
	assert.Contains(t, output, "force")
}

func TestDeprecatedCommandHelp(t *testing.T) {
	var stdout bytes.Buffer
	app := New("test").Description("Test")
	app.stdout = &stdout

	app.Command("old").
		Description("Old command").
		Deprecated("Use 'new' instead").
		Run(func(ctx *Context) error { return nil })

	// Deprecation notice is shown in help output
	_ = app.RunArgs([]string{"old", "--help"})
	assert.Contains(t, stdout.String(), "DEPRECATED")
	assert.Contains(t, stdout.String(), "Use 'new' instead")
}

func TestMiddlewareBeforeError(t *testing.T) {
	app := New("test").Description("Test")
	app.stderr = &bytes.Buffer{}

	app.Command("run").
		Description("Run").
		Use(Before(func(ctx *Context) error {
			return Error("before error")
		})).
		Run(func(ctx *Context) error { return nil })

	err := app.RunArgs([]string{"run"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "before error")
}

func TestMiddlewareAfterWithCommandError(t *testing.T) {
	var afterRan bool
	app := New("test").Description("Test")
	app.stderr = &bytes.Buffer{}

	app.Command("run").
		Description("Run").
		Use(After(func(ctx *Context) error {
			afterRan = true
			return nil
		})).
		Run(func(ctx *Context) error {
			return Error("command error")
		})

	err := app.RunArgs([]string{"run"})
	assert.Error(t, err)
	// After should still run even if command errors
	assert.True(t, afterRan)
}

func TestCommandWithContext(t *testing.T) {
	var receivedCtx context.Context
	app := New("test").Description("Test")

	app.Command("run").
		Description("Run").
		Run(func(ctx *Context) error {
			receivedCtx = ctx.Context()
			return nil
		})

	err := app.RunArgs([]string{"run"})
	assert.NoError(t, err)
	assert.NotNil(t, receivedCtx)
}

func TestInvalidIntFlagFallsBackToZero(t *testing.T) {
	var count int
	app := New("test").Description("Test")

	app.Command("run").
		Description("Run").
		Flags(&IntFlag{Name: "count"}).
		Run(func(ctx *Context) error {
			count = ctx.Int("count")
			return nil
		})

	// Invalid int values are stored as strings, and ctx.Int returns 0
	err := app.RunArgs([]string{"run", "--count=notanumber"})
	assert.NoError(t, err)
	assert.Equal(t, 0, count)
}


func TestCommandShowHelpFlag(t *testing.T) {
	var buf bytes.Buffer
	app := New("test").Description("Test")
	app.stdout = &buf

	app.Command("run").
		Description("Run the task").
		Flags(&StringFlag{Name: "config", Help: "Config file"}).
		Run(func(ctx *Context) error { return nil })

	// --help returns HelpRequested error
	err := app.RunArgs([]string{"run", "--help"})
	var helpErr *HelpRequested
	assert.ErrorAs(t, err, &helpErr)
	assert.Contains(t, buf.String(), "Run the task")
	assert.Contains(t, buf.String(), "config")
}

func TestShortFlagCombined(t *testing.T) {
	var a, b, c bool

	app := New("test").Description("Test")
	app.Command("run").
		Description("Run").
		Flags(
			&BoolFlag{Name: "alpha", Short: "a"},
			&BoolFlag{Name: "beta", Short: "b"},
			&BoolFlag{Name: "gamma", Short: "c"},
		).
		Run(func(ctx *Context) error {
			a = ctx.Bool("alpha")
			b = ctx.Bool("beta")
			c = ctx.Bool("gamma")
			return nil
		})

	err := app.RunArgs([]string{"run", "-abc"})
	assert.NoError(t, err)
	assert.True(t, a)
	assert.True(t, b)
	assert.True(t, c)
}

func TestExitErrorMessage(t *testing.T) {
	err := Exit(42)
	exitErr, ok := err.(*ExitError)
	assert.True(t, ok)
	assert.Equal(t, 42, exitErr.Code)
	assert.Equal(t, "", exitErr.Error())

	// GetExitCode should return the code
	assert.Equal(t, 42, GetExitCode(err))
}

func TestCommandErrorWithDetails(t *testing.T) {
	err := Error("something failed").
		Hint("Try checking the config").
		Code("ERR_001").
		Detail("File not found: %s", "config.yaml")

	assert.Contains(t, err.Error(), "something failed")
	assert.Contains(t, err.Error(), "Hint: Try checking the config")
	assert.Contains(t, err.Error(), "File not found: config.yaml")
	assert.Equal(t, "ERR_001", err.ErrorCode())
}

func TestErrorfWithFormatting(t *testing.T) {
	err := Errorf("failed to connect to %s:%d", "localhost", 8080)
	assert.Contains(t, err.Error(), "failed to connect to localhost:8080")
}

func TestIsHelpRequested(t *testing.T) {
	helpErr := &HelpRequested{}
	assert.True(t, IsHelpRequested(helpErr))
	assert.Equal(t, "help requested", helpErr.Error())

	otherErr := Error("some error")
	assert.False(t, IsHelpRequested(otherErr))
}

func TestGetExitCodeForNil(t *testing.T) {
	assert.Equal(t, 0, GetExitCode(nil))
}

func TestGetExitCodeForHelpRequested(t *testing.T) {
	assert.Equal(t, 0, GetExitCode(&HelpRequested{}))
}

func TestGetExitCodeForGenericError(t *testing.T) {
	assert.Equal(t, 1, GetExitCode(Error("generic")))
}

func TestAppVersionCommand(t *testing.T) {
	var buf bytes.Buffer
	app := New("test").Description("Test")
	app.Version("1.2.3")
	app.stdout = &buf

	err := app.RunArgs([]string{"version"})
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "1.2.3")
}

func TestFlagEnumValidation(t *testing.T) {
	app := New("test").Description("Test")
	app.Command("run").
		Description("Run").
		Flags(&StringFlag{
			Name: "env",
			Enum: []string{"dev", "staging", "prod"},
		}).
		Run(func(ctx *Context) error { return nil })

	// Valid enum value
	err := app.RunArgs([]string{"run", "--env=dev"})
	assert.NoError(t, err)

	// Invalid enum value
	err = app.RunArgs([]string{"run", "--env=invalid"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid value")
}

func TestHiddenCommandNotInHelp(t *testing.T) {
	var buf bytes.Buffer
	app := New("test").Description("Test")
	app.stdout = &buf

	app.Command("visible").Description("Visible command").Run(func(ctx *Context) error { return nil })
	app.Command("hidden").Description("Hidden command").Hidden().Run(func(ctx *Context) error { return nil })

	// Show help
	_ = app.RunArgs([]string{})

	output := buf.String()
	assert.Contains(t, output, "visible")
	assert.NotContains(t, output, "hidden")
}

func TestDoubleDashStopsFlags(t *testing.T) {
	var args []string
	app := New("test").Description("Test")
	app.Command("run").
		Description("Run").
		Args("files...").
		Run(func(ctx *Context) error {
			args = ctx.Args()
			return nil
		})

	err := app.RunArgs([]string{"run", "--", "--not-a-flag", "-also-not"})
	assert.NoError(t, err)
	assert.Equal(t, []string{"--not-a-flag", "-also-not"}, args)
}

func TestHelpHint(t *testing.T) {
	var buf bytes.Buffer
	app := New("test").Description("Test app")
	app.stdout = &buf

	app.Command("run").Description("Run something").Run(func(ctx *Context) error { return nil })

	_ = app.RunArgs([]string{})
	assert.Contains(t, buf.String(), "test <command> --help")
}

func TestBindFlagsWithVariousConversions(t *testing.T) {
	type Config struct {
		Name    string  `flag:"name"`
		Count   int     `flag:"count"`
		Count64 int64   `flag:"count64"`
		Rate    float64 `flag:"rate"`
		Enabled bool    `flag:"enabled"`
	}

	var boundCfg *Config
	app := New("test").Description("Test")
	app.Command("run").
		Description("Run").
		Flags(
			&StringFlag{Name: "name"},
			&IntFlag{Name: "count"},
			&IntFlag{Name: "count64"},
			&Float64Flag{Name: "rate"},
			&BoolFlag{Name: "enabled"},
		).
		Run(func(ctx *Context) error {
			var err error
			boundCfg, err = BindFlags[Config](ctx)
			return err
		})

	err := app.RunArgs([]string{"run", "--name=test", "--count=5", "--count64=100", "--rate=3.14", "--enabled"})
	assert.NoError(t, err)
	assert.NotNil(t, boundCfg)
	assert.Equal(t, "test", boundCfg.Name)
	assert.Equal(t, 5, boundCfg.Count)
	assert.Equal(t, int64(100), boundCfg.Count64)
	assert.InDelta(t, 3.14, boundCfg.Rate, 0.001)
	assert.True(t, boundCfg.Enabled)
}

func TestUnknownShortFlag(t *testing.T) {
	app := New("test").Description("Test")
	app.Command("run").
		Description("Run").
		Run(func(ctx *Context) error { return nil })

	err := app.RunArgs([]string{"run", "-x"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown flag")
}

func TestFlagRequiresValue(t *testing.T) {
	app := New("test").Description("Test")
	app.Command("run").
		Description("Run").
		Flags(&StringFlag{Name: "config"}).
		Run(func(ctx *Context) error { return nil })

	// Flag at end without value
	err := app.RunArgs([]string{"run", "--config"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "requires a value")
}

func TestShortFlagRequiresValue(t *testing.T) {
	app := New("test").Description("Test")
	app.Command("run").
		Description("Run").
		Flags(&StringFlag{Name: "config", Short: "c"}).
		Run(func(ctx *Context) error { return nil })

	// Combined short flags where non-last needs value
	err := app.RunArgs([]string{"run", "-c"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "requires a value")
}

func TestCommandWithLongDescription(t *testing.T) {
	var buf bytes.Buffer
	app := New("test").Description("Test")
	app.stdout = &buf

	app.Command("deploy").
		Description("Deploy the application").
		Long("This is a longer description that provides more detail about what the deploy command does, including examples and caveats.").
		Run(func(ctx *Context) error { return nil })

	_ = app.RunArgs([]string{"deploy", "--help"})
	output := buf.String()
	assert.Contains(t, output, "longer description")
}

func TestFlagWithDefaultShownInHelp(t *testing.T) {
	var buf bytes.Buffer
	app := New("test").Description("Test")
	app.stdout = &buf

	app.Command("run").
		Description("Run").
		Flags(&StringFlag{Name: "env", Value: "development", Help: "Environment"}).
		Run(func(ctx *Context) error { return nil })

	_ = app.RunArgs([]string{"run", "--help"})
	output := buf.String()
	assert.Contains(t, output, "default: development")
}

func TestEnumShownInHelp(t *testing.T) {
	var buf bytes.Buffer
	app := New("test").Description("Test")
	app.stdout = &buf

	app.Command("run").
		Description("Run").
		Flags(&StringFlag{Name: "env", Enum: []string{"dev", "prod"}, Help: "Environment"}).
		Run(func(ctx *Context) error { return nil })

	_ = app.RunArgs([]string{"run", "--help"})
	output := buf.String()
	assert.Contains(t, output, "dev|prod")
}

func TestRequiredFlagShownInHelp(t *testing.T) {
	var buf bytes.Buffer
	app := New("test").Description("Test")
	app.stdout = &buf

	app.Command("run").
		Description("Run").
		Flags(&StringFlag{Name: "config", Required: true, Help: "Config file"}).
		Run(func(ctx *Context) error { return nil })

	_ = app.RunArgs([]string{"run", "--help"})
	output := buf.String()
	assert.Contains(t, output, "(required)")
}
