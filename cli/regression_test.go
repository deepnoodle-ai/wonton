package cli

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/deepnoodle-ai/wonton/assert"
)

// BUG-031: "--" ends flag parsing for the root command too.
func TestDoubleDashOnRootCommand(t *testing.T) {
	app := TestApp("app")
	var got []string
	app.Main().Args("args...").Flags(Bool("verbose", "v")).Run(func(ctx *Context) error {
		got = ctx.Args()
		assert.False(t, ctx.Bool("verbose"))
		return nil
	})
	assert.NoError(t, app.ExecuteArgs([]string{"--", "-v", "--help"}))
	assert.Equal(t, []string{"-v", "--help"}, got)

	// Also with subcommands defined.
	app.Command("other").Run(func(*Context) error { return errors.New("wrong command") })
	got = nil
	assert.NoError(t, app.ExecuteArgs([]string{"--", "-v"}))
	assert.Equal(t, []string{"-v"}, got)
}

// BUG-031: group middleware runs for group subcommands.
func TestGroupMiddlewareRunsForSubcommands(t *testing.T) {
	app := TestApp("app")
	var order []string
	mw := func(name string) Middleware {
		return func(next Handler) Handler {
			return func(ctx *Context) error {
				order = append(order, name)
				return next(ctx)
			}
		}
	}
	app.Use(mw("app"))
	admin := app.Group("admin").Use(mw("group"))
	admin.Command("users").Use(mw("cmd")).Run(func(*Context) error {
		order = append(order, "handler")
		return nil
	})
	assert.NoError(t, app.ExecuteArgs([]string{"admin", "users"}))
	assert.Equal(t, []string{"app", "group", "cmd", "handler"}, order)
}

// BUG-063: values from environment variables are validated, and command-line
// values replace (not extend) slice values from the environment.
func TestEnvFlagValidationAndSliceOverride(t *testing.T) {
	app := TestApp("app")
	app.Command("run").Flags(
		String("env", "e").Env("APP_ENV").Enum("dev", "prod"),
		String("name").Env("APP_NAME").ValidateWith(func(s string) error {
			if strings.Contains(s, " ") {
				return errors.New("no spaces")
			}
			return nil
		}),
	).Run(func(*Context) error { return nil })

	t.Setenv("APP_ENV", "invalid")
	err := app.ExecuteArgs([]string{"run"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "APP_ENV")

	t.Setenv("APP_ENV", "dev")
	t.Setenv("APP_NAME", "has space")
	err = app.ExecuteArgs([]string{"run"})
	assert.Error(t, err)

	app2 := TestApp("app")
	var tags []string
	app2.Command("run").Flags(Strings("tag", "t").Env("TAGS")).Run(func(ctx *Context) error {
		tags = ctx.Strings("tag")
		return nil
	})
	t.Setenv("TAGS", "a,b")
	assert.NoError(t, app2.ExecuteArgs([]string{"run"}))
	assert.Equal(t, []string{"a", "b"}, tags)
	assert.NoError(t, app2.ExecuteArgs([]string{"run", "--tag", "c", "-t", "d"}))
	assert.Equal(t, []string{"c", "d"}, tags)
}

// BUG-064: struct flags support time.Duration and slices, and BindFlags
// rejects bad types instead of panicking.
func TestStructFlagsDurationSlicesAndUnexported(t *testing.T) {
	type Flags struct {
		Timeout time.Duration `flag:"timeout" default:"30s"`
		Tags    []string      `flag:"tag" default:"a,b"`
		Ports   []int         `flag:"port"`
		hidden  string        `flag:"hidden"`
	}
	app := TestApp("app")
	var got *Flags
	cmd := app.Command("run")
	ParseFlags[Flags](cmd).Run(func(ctx *Context) error {
		var err error
		got, err = BindFlags[Flags](ctx)
		return err
	})

	assert.NoError(t, app.ExecuteArgs([]string{"run"}))
	assert.Equal(t, 30*time.Second, got.Timeout)
	assert.Equal(t, []string{"a", "b"}, got.Tags)
	assert.Equal(t, 0, len(got.Ports))

	assert.NoError(t, app.ExecuteArgs([]string{"run", "--timeout", "10s", "--tag", "x", "--port", "80", "--port", "443"}))
	assert.Equal(t, 10*time.Second, got.Timeout)
	assert.Equal(t, []string{"x"}, got.Tags)
	assert.Equal(t, []int{80, 443}, got.Ports)
	assert.Equal(t, "", got.hidden)

	_, err := BindFlags[*Flags](newTestContext(map[string]any{}))
	assert.Error(t, err)
}

// BUG-065: short flags accept "=" values.
func TestShortFlagWithEquals(t *testing.T) {
	app := TestApp("app")
	var out string
	var verbose bool
	app.Command("run").Flags(String("output", "o"), Bool("verbose", "v")).Run(func(ctx *Context) error {
		out = ctx.String("output")
		verbose = ctx.Bool("verbose")
		return nil
	})
	assert.NoError(t, app.ExecuteArgs([]string{"run", "-o=out.txt", "-v=false"}))
	assert.Equal(t, "out.txt", out)
	assert.False(t, verbose)
	assert.NoError(t, app.ExecuteArgs([]string{"run", "-vo=x"}))
	assert.Equal(t, "x", out)
	assert.True(t, verbose)

	app.GlobalFlags(String("config", "c"))
	var cfg string
	app.Command("cfg").Run(func(ctx *Context) error {
		cfg = ctx.String("config")
		return nil
	})
	assert.NoError(t, app.ExecuteArgs([]string{"-c=file.yaml", "cfg"}))
	assert.Equal(t, "file.yaml", cfg)
}

// BUG-065: a root boolean flag before positionals keeps their order when the
// app also has subcommands.
func TestRootPositionalsKeepOrderAfterBoolFlag(t *testing.T) {
	app := TestApp("app")
	var args []string
	app.Main().Args("files...").Flags(Bool("verbose")).Run(func(ctx *Context) error {
		args = ctx.Args()
		assert.True(t, ctx.Bool("verbose"))
		return nil
	})
	app.Command("sub").Run(func(*Context) error { return nil })
	assert.NoError(t, app.ExecuteArgs([]string{"--verbose", "first.txt", "second.txt"}))
	assert.Equal(t, []string{"first.txt", "second.txt"}, args)
}

// BUG-065: interactivity is detected from the app's own stdin/stdout.
func TestInteractivityUsesAppStreams(t *testing.T) {
	// Pretend every *os.File is a terminal, as when go test runs in one.
	orig := isTerminal
	isTerminal = func(*os.File) bool { return true }
	defer func() { isTerminal = orig }()

	app := TestApp("app")
	ran := ""
	app.Command("run").
		Interactive(func(*Context) error { ran = "interactive"; return nil }).
		NonInteractive(func(*Context) error { ran = "non-interactive"; return nil })
	assert.NoError(t, app.ExecuteArgs([]string{"run"}))
	assert.Equal(t, "non-interactive", ran, "app stdin is not a file, so the app is not interactive")

	// With the app's streams on terminal files, it is interactive.
	app.SetStdin(os.Stdin)
	app.SetStdout(os.Stdout)
	assert.NoError(t, app.ExecuteArgs([]string{"run"}))
	assert.Equal(t, "interactive", ran)
}

// An empty environment variable counts as unset, and a command-line value
// overrides a bad environment value.
func TestEnvFlagEmptyAndOverride(t *testing.T) {
	app := TestApp("app")
	var env string
	var debug bool
	app.Command("run").Flags(
		String("env", "e").Env("APP_ENV").Enum("dev", "prod").Default("dev"),
		Bool("debug").Env("APP_DEBUG"),
	).Run(func(ctx *Context) error {
		env = ctx.String("env")
		debug = ctx.Bool("debug")
		return nil
	})

	t.Setenv("APP_ENV", "")
	assert.NoError(t, app.ExecuteArgs([]string{"run"}))
	assert.Equal(t, "dev", env)

	t.Setenv("APP_ENV", "invalid")
	assert.NoError(t, app.ExecuteArgs([]string{"run", "--env", "prod"}))
	assert.Equal(t, "prod", env)
	assert.Error(t, app.ExecuteArgs([]string{"run"}))

	t.Setenv("APP_ENV", "dev")
	t.Setenv("APP_DEBUG", "maybe")
	assert.NoError(t, app.ExecuteArgs([]string{"run", "--debug"}))
	assert.True(t, debug)
	assert.Error(t, app.ExecuteArgs([]string{"run"}))
}

// BUG-074: IsPiped does not panic when stdin cannot be stat'ed.
func TestIsPipedWithClosedStdin(t *testing.T) {
	r, w, err := os.Pipe()
	assert.NoError(t, err)
	w.Close()
	r.Close()
	orig := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = orig }()
	assert.False(t, IsPiped())
}

// BUG-075: fish descriptions escape backslashes, and shell function names
// are valid identifiers.
func TestCompletionEscaping(t *testing.T) {
	assert.Equal(t, `ends with \\`, escapeFish(`ends with \`))
	assert.Equal(t, `it\'s \\ ok`, escapeFish(`it's \ ok`))

	app := New("my-app.v2")
	app.Group("dev-tools").Command("run").Run(func(*Context) error { return nil })
	var bash, zsh strings.Builder
	assert.NoError(t, app.GenerateBashCompletion(&bash))
	assert.NoError(t, app.GenerateZshCompletion(&zsh))
	assert.Contains(t, bash.String(), "_my_app_v2_completions()")
	assert.Contains(t, bash.String(), "complete -F _my_app_v2_completions my-app.v2")
	assert.Contains(t, zsh.String(), "__my_app_v2_group_dev_tools()")
}
