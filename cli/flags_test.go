package cli

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestFloatAliasForFloat64(t *testing.T) {
	flag := Float("ratio", "r").Default(0.75).Help("Test ratio")
	assert.Equal(t, "ratio", flag.GetName())
	assert.Equal(t, "r", flag.GetShort())
	assert.Equal(t, "Test ratio", flag.GetHelp())
	assert.Equal(t, 0.75, flag.GetDefault())
}

func TestBindFlagsFromContext(t *testing.T) {
	t.Run("binds string field from context", func(t *testing.T) {
		type Config struct {
			Name string `flag:"name,n"`
		}

		ctx := newTestContext(map[string]any{
			"name": "Alice",
		})

		cfg, err := BindFlags[Config](ctx)
		assert.NoError(t, err)
		assert.Equal(t, "Alice", cfg.Name)
	})

	t.Run("binds bool field from context", func(t *testing.T) {
		type Config struct {
			Verbose bool `flag:"verbose"`
		}

		ctx := newTestContext(map[string]any{
			"verbose": true,
		})

		cfg, err := BindFlags[Config](ctx)
		assert.NoError(t, err)
		assert.True(t, cfg.Verbose)
	})

	t.Run("binds int field from context", func(t *testing.T) {
		type Config struct {
			Port int `flag:"port"`
		}

		ctx := newTestContext(map[string]any{
			"port": 8080,
		})

		cfg, err := BindFlags[Config](ctx)
		assert.NoError(t, err)
		assert.Equal(t, 8080, cfg.Port)
	})

	t.Run("binds int64 field from context", func(t *testing.T) {
		type Config struct {
			Size int64 `flag:"size"`
		}

		ctx := newTestContext(map[string]any{
			"size": int64(1024),
		})

		cfg, err := BindFlags[Config](ctx)
		assert.NoError(t, err)
		assert.Equal(t, int64(1024), cfg.Size)
	})

	t.Run("binds float64 field from context", func(t *testing.T) {
		type Config struct {
			Ratio float64 `flag:"ratio"`
		}

		ctx := newTestContext(map[string]any{
			"ratio": 0.75,
		})

		cfg, err := BindFlags[Config](ctx)
		assert.NoError(t, err)
		assert.InDelta(t, 0.75, cfg.Ratio, 0.001)
	})

	t.Run("uses default when flag not set", func(t *testing.T) {
		type Config struct {
			Port int `flag:"port" default:"8080"`
		}

		ctx := newTestContext(map[string]any{})

		cfg, err := BindFlags[Config](ctx)
		assert.NoError(t, err)
		assert.Equal(t, 8080, cfg.Port)
	})

	t.Run("converts string to int", func(t *testing.T) {
		type Config struct {
			Port int `flag:"port"`
		}

		ctx := newTestContext(map[string]any{
			"port": "9090",
		})

		cfg, err := BindFlags[Config](ctx)
		assert.NoError(t, err)
		assert.Equal(t, 9090, cfg.Port)
	})

	t.Run("converts string to float64", func(t *testing.T) {
		type Config struct {
			Ratio float64 `flag:"ratio"`
		}

		ctx := newTestContext(map[string]any{
			"ratio": "0.85",
		})

		cfg, err := BindFlags[Config](ctx)
		assert.NoError(t, err)
		assert.InDelta(t, 0.85, cfg.Ratio, 0.001)
	})

	t.Run("converts string to bool", func(t *testing.T) {
		type Config struct {
			Enabled bool `flag:"enabled"`
		}

		ctx := newTestContext(map[string]any{
			"enabled": "true",
		})

		cfg, err := BindFlags[Config](ctx)
		assert.NoError(t, err)
		assert.True(t, cfg.Enabled)
	})

	t.Run("converts int to float64", func(t *testing.T) {
		type Config struct {
			Ratio float64 `flag:"ratio"`
		}

		ctx := newTestContext(map[string]any{
			"ratio": 1,
		})

		cfg, err := BindFlags[Config](ctx)
		assert.NoError(t, err)
		assert.InDelta(t, 1.0, cfg.Ratio, 0.001)
	})

	t.Run("converts int64 to float64", func(t *testing.T) {
		type Config struct {
			Ratio float64 `flag:"ratio"`
		}

		ctx := newTestContext(map[string]any{
			"ratio": int64(2),
		})

		cfg, err := BindFlags[Config](ctx)
		assert.NoError(t, err)
		assert.InDelta(t, 2.0, cfg.Ratio, 0.001)
	})

	t.Run("converts float64 to int", func(t *testing.T) {
		type Config struct {
			Count int `flag:"count"`
		}

		ctx := newTestContext(map[string]any{
			"count": 42.7,
		})

		cfg, err := BindFlags[Config](ctx)
		assert.NoError(t, err)
		assert.Equal(t, 42, cfg.Count)
	})

	t.Run("converts int to int64", func(t *testing.T) {
		type Config struct {
			Size int64 `flag:"size"`
		}

		ctx := newTestContext(map[string]any{
			"size": 1024,
		})

		cfg, err := BindFlags[Config](ctx)
		assert.NoError(t, err)
		assert.Equal(t, int64(1024), cfg.Size)
	})

	t.Run("skips fields without flag tag", func(t *testing.T) {
		type Config struct {
			Name   string `flag:"name"`
			Secret string // no flag tag, should not be populated
		}

		ctx := newTestContext(map[string]any{
			"name":   "Alice",
			"Secret": "should-not-be-set",
		})

		cfg, err := BindFlags[Config](ctx)
		assert.NoError(t, err)
		assert.Equal(t, "Alice", cfg.Name)
		assert.Equal(t, "", cfg.Secret)
	})

	t.Run("handles bool conversions from string", func(t *testing.T) {
		type Config struct {
			Flag1 bool `flag:"flag1"`
			Flag2 bool `flag:"flag2"`
		}

		ctx := newTestContext(map[string]any{
			"flag1": "yes",
			"flag2": "1",
		})

		cfg, err := BindFlags[Config](ctx)
		assert.NoError(t, err)
		assert.True(t, cfg.Flag1)
		assert.True(t, cfg.Flag2)
	})

	t.Run("supports time.Duration, pointers, and unexported fields", func(t *testing.T) {
		type Config struct {
			Timeout  time.Duration  `flag:"timeout,t" default:"5s"`
			Delay    *time.Duration `flag:"delay" default:"250ms"`
			Name     *string        `flag:"name,n"`
			internal string         `flag:"internal"`
		}

		app := New("test")
		cmd := app.Command("run")
		ParseFlags[Config](cmd)

		var got *Config
		cmd.Run(func(ctx *Context) error {
			var err error
			got, err = BindFlags[Config](ctx)
			return err
		})

		err := app.ExecuteArgs([]string{"run", "--timeout", "15s", "--name", "worker"})
		assert.NoError(t, err)
		assert.NotNil(t, got)
		assert.Equal(t, 15*time.Second, got.Timeout)
		assert.NotNil(t, got.Delay)
		assert.Equal(t, 250*time.Millisecond, *got.Delay)
		assert.NotNil(t, got.Name)
		assert.Equal(t, "worker", *got.Name)
		assert.Equal(t, "", got.internal)
	})
}

func TestCaptureOutputLargeAndPanicRestore(t *testing.T) {
	large := strings.Repeat("x", 256*1024)
	out, errOut := CaptureOutput(func() {
		_, _ = fmt.Fprint(os.Stdout, large)
		_, _ = fmt.Fprint(os.Stderr, large)
	})
	assert.Equal(t, len(large), len(out))
	assert.Equal(t, len(large), len(errOut))

	origOut, origErr := os.Stdout, os.Stderr
	assert.Panics(t, func() {
		CaptureOutput(func() {
			panic("boom")
		})
	})
	assert.Equal(t, origOut, os.Stdout)
	assert.Equal(t, origErr, os.Stderr)
}

func TestEndOfFlagsWithFlagLikePositionalOnRoot(t *testing.T) {
	var gotArgs []string
	app := New("test").
		Args("args?...").
		GlobalFlags(Bool("verbose", "v")).
		Run(func(ctx *Context) error {
			gotArgs = ctx.Args()
			return nil
		})

	err := app.ExecuteArgs([]string{"--", "-v"})
	assert.NoError(t, err)
	assert.Equal(t, []string{"-v"}, gotArgs)
}

func TestGroupMiddlewareAndFlagsOnSubcommand(t *testing.T) {
	var order []string
	var envVal string

	app := New("test")
	app.Use(func(next Handler) Handler {
		return func(ctx *Context) error {
			order = append(order, "app")
			return next(ctx)
		}
	})

	grp := app.Group("deploy").
		Flags(String("env", "e").Default("staging")).
		Use(func(next Handler) Handler {
			return func(ctx *Context) error {
				order = append(order, "group")
				return next(ctx)
			}
		})

	grp.Command("service").
		Use(func(next Handler) Handler {
			return func(ctx *Context) error {
				order = append(order, "cmd")
				return next(ctx)
			}
		}).
		Run(func(ctx *Context) error {
			order = append(order, "handler")
			envVal = ctx.String("env")
			return nil
		})

	err := app.ExecuteArgs([]string{"deploy", "service", "--env", "prod"})
	assert.NoError(t, err)
	assert.Equal(t, []string{"app", "group", "cmd", "handler"}, order)
	assert.Equal(t, "prod", envVal)

	// Subcommand flag takes precedence over group flag of the same name (Q1).
	var countVal int
	grp.Command("override").
		Flags(Int("env", "e").Default(42)).
		Run(func(ctx *Context) error {
			countVal = ctx.Int("env")
			return nil
		})
	err = app.ExecuteArgs([]string{"deploy", "override", "--env", "7"})
	assert.NoError(t, err)
	assert.Equal(t, 7, countVal)
}
