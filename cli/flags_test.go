package cli

import (
	"testing"

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
}

func TestFlagOnce(t *testing.T) {
	run := func(t *testing.T, f Flag, args ...string) (*Context, error) {
		t.Helper()
		var got *Context
		app := New("test")
		app.Command("run").Flags(f).Run(func(ctx *Context) error {
			got = ctx
			return nil
		})
		err := app.ExecuteArgs(append([]string{"run"}, args...))
		return got, err
	}

	t.Run("second use is a usage error", func(t *testing.T) {
		cases := []struct {
			name string
			flag Flag
			args []string
		}{
			{"string", String("name", "n").Once(), []string{"--name", "a", "--name", "b"}},
			{"string short and long", String("name", "n").Once(), []string{"-n", "a", "--name=b"}},
			{"enum", String("format").Enum("json", "text").Once(), []string{"--format", "json", "--format", "text"}},
			{"int", Int("max").Once(), []string{"--max", "1", "--max", "2"}},
			{"int64", Int64("size").Once(), []string{"--size", "1", "--size", "2"}},
			{"float", Float64("ratio").Once(), []string{"--ratio", "1", "--ratio", "2"}},
			{"duration", Duration("timeout").Once(), []string{"--timeout", "1s", "--timeout", "2s"}},
			{"bool", Bool("write", "w").Once(), []string{"--write", "-w"}},
			{"struct flag", &StringFlag{Name: "name", Once: true}, []string{"--name=a", "--name=b"}},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				_, err := run(t, tc.flag, tc.args...)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "flag --"+tc.flag.GetName()+" given more than once")
				assert.Contains(t, err.Error(), "Run 'test run --help' to see available flags.")
			})
		}
	})

	t.Run("single use is accepted", func(t *testing.T) {
		ctx, err := run(t, String("name").Once(), "--name", "a")
		assert.NoError(t, err)
		assert.Equal(t, "a", ctx.String("name"))
	})

	t.Run("environment value does not count", func(t *testing.T) {
		t.Setenv("TEST_ONCE_NAME", "env")
		ctx, err := run(t, String("name").Env("TEST_ONCE_NAME").Once(), "--name", "cli")
		assert.NoError(t, err)
		assert.Equal(t, "cli", ctx.String("name"))
	})

	t.Run("without Once the last value wins", func(t *testing.T) {
		ctx, err := run(t, String("name"), "--name", "a", "--name", "b")
		assert.NoError(t, err)
		assert.Equal(t, "b", ctx.String("name"))
	})

	t.Run("list flag ignores Once", func(t *testing.T) {
		ctx, err := run(t, onceSlice{&StringSliceFlag{Name: "tag"}}, "--tag", "a", "--tag", "b")
		assert.NoError(t, err)
		assert.Equal(t, []string{"a", "b"}, ctx.Strings("tag"))
	})

	t.Run("global flag", func(t *testing.T) {
		app := New("test").GlobalFlags(String("profile").Once())
		app.Command("run").Run(func(*Context) error { return nil })
		err := app.ExecuteArgs([]string{"--profile", "a", "run", "--profile", "b"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "flag --profile given more than once")
	})
}

// onceSlice is a slice flag that claims to be once-only.
type onceSlice struct{ *StringSliceFlag }

func (onceSlice) IsOnce() bool { return true }

func TestParseFlagsOnceAndPlaceholderTags(t *testing.T) {
	type Flags struct {
		Config string   `flag:"config,c" placeholder:"FILE" once:""`
		Max    int      `flag:"max" placeholder:"N"`
		Tags   []string `flag:"tag" placeholder:"TAG"`
		Write  bool     `flag:"write" once:""`
	}
	app := New("test")
	cmd := app.Command("run").Run(func(*Context) error { return nil })
	ParseFlags[Flags](cmd)

	assert.Equal(t, "config FILE", flagLabel(cmd.findFlag("config")))
	assert.Equal(t, "max N", flagLabel(cmd.findFlag("max")))
	assert.Equal(t, "tag TAG", flagLabel(cmd.findFlag("tag")))
	assert.Equal(t, "write", flagLabel(cmd.findFlag("write")))
	assert.True(t, isOnceFlag(cmd.findFlag("config")))
	assert.False(t, isOnceFlag(cmd.findFlag("max")))
	assert.True(t, isOnceFlag(cmd.findFlag("write")))

	err := app.ExecuteArgs([]string{"run", "-c", "a", "--config", "b"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "flag --config given more than once")
}
