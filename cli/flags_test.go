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
