package cli

import (
	"bytes"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

func newTestContext(flags map[string]any, args ...string) *Context {
	if flags == nil {
		flags = make(map[string]any)
	}
	// Also populate setFlags for all passed flags
	setFlags := make(map[string]bool)
	for name := range flags {
		setFlags[name] = true
	}
	return &Context{
		flags:      flags,
		setFlags:   setFlags,
		positional: args,
		stdout:     &bytes.Buffer{},
		stderr:     &bytes.Buffer{},
	}
}

func TestContextIntParsesString(t *testing.T) {
	ctx := newTestContext(map[string]any{
		"count": "42",
	})

	assert.Equal(t, 42, ctx.Int("count"))
}

func TestContextFloat64ParsesVariousTypes(t *testing.T) {
	ctx := newTestContext(map[string]any{
		"fromInt":   5,
		"fromInt64": int64(10),
		"fromStr":   "3.14",
	})

	assert.Equal(t, 5.0, ctx.Float64("fromInt"))
	assert.Equal(t, 10.0, ctx.Float64("fromInt64"))
	assert.InDelta(t, 3.14, ctx.Float64("fromStr"), 0.0001)
}

func TestContextBoolParsesStrings(t *testing.T) {
	ctx := newTestContext(map[string]any{
		"truthy": "yes",
		"falsey": "0",
	})

	assert.True(t, ctx.Bool("truthy"))
	assert.False(t, ctx.Bool("falsey"))
	assert.False(t, ctx.Bool("missing"))
}

func TestContextIsSet(t *testing.T) {
	ctx := newTestContext(map[string]any{
		"flag": true,
	})

	assert.True(t, ctx.IsSet("flag"))
	assert.False(t, ctx.IsSet("other"))
}

func TestContextArgs(t *testing.T) {
	ctx := newTestContext(nil, "first", "second")

	assert.Equal(t, 2, ctx.NArg())
	assert.Equal(t, "first", ctx.Arg(0))
	assert.Equal(t, "second", ctx.Arg(1))
	assert.Equal(t, "", ctx.Arg(2))
}

func TestSemanticOutputHelpers(t *testing.T) {
	var stdout, stderr bytes.Buffer

	// Create app with colors disabled for predictable output
	app := New("test")
	app.colorEnabled = false

	ctx := &Context{
		app:    app,
		stdout: &stdout,
		stderr: &stderr,
	}

	ctx.Success("deployed to %s", "prod")
	ctx.Info("processing %d items", 10)
	ctx.Warn("disk usage at %d%%", 85)
	ctx.Fail("connection %s", "failed")

	assert.Contains(t, stdout.String(), "deployed to prod")
	assert.Contains(t, stdout.String(), "processing 10 items")
	assert.Contains(t, stderr.String(), "disk usage at 85%")
	assert.Contains(t, stderr.String(), "connection failed")
}

func TestPromptsRequireInteractive(t *testing.T) {
	ctx := &Context{
		interactive: false,
		stdout:      &bytes.Buffer{},
		stderr:      &bytes.Buffer{},
	}

	_, err := ctx.Select("Choose:", "a", "b")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "interactive")

	_, err = ctx.Input("Name:")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "interactive")

	_, err = ctx.Confirm("Continue?")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "interactive")
}

func TestContextStrings(t *testing.T) {
	t.Run("returns slice when flag is []string", func(t *testing.T) {
		ctx := newTestContext(map[string]any{
			"tags": []string{"tag1", "tag2", "tag3"},
		})
		assert.Equal(t, []string{"tag1", "tag2", "tag3"}, ctx.Strings("tags"))
	})

	t.Run("returns single item slice when flag is string", func(t *testing.T) {
		ctx := newTestContext(map[string]any{
			"tag": "single",
		})
		assert.Equal(t, []string{"single"}, ctx.Strings("tag"))
	})

	t.Run("returns nil when flag is empty string", func(t *testing.T) {
		ctx := newTestContext(map[string]any{
			"tag": "",
		})
		assert.Nil(t, ctx.Strings("tag"))
	})

	t.Run("returns nil when flag not set", func(t *testing.T) {
		ctx := newTestContext(map[string]any{})
		assert.Nil(t, ctx.Strings("missing"))
	})
}

func TestContextInts(t *testing.T) {
	t.Run("returns slice when flag is []int", func(t *testing.T) {
		ctx := newTestContext(map[string]any{
			"ports": []int{8080, 8081, 8082},
		})
		assert.Equal(t, []int{8080, 8081, 8082}, ctx.Ints("ports"))
	})

	t.Run("returns single item slice when flag is int", func(t *testing.T) {
		ctx := newTestContext(map[string]any{
			"port": 8080,
		})
		assert.Equal(t, []int{8080}, ctx.Ints("port"))
	})

	t.Run("returns nil when flag not set", func(t *testing.T) {
		ctx := newTestContext(map[string]any{})
		assert.Nil(t, ctx.Ints("missing"))
	})
}

func TestContextStringWithNonStringTypes(t *testing.T) {
	ctx := newTestContext(map[string]any{
		"number": 123,
		"bool":   true,
	})

	assert.Equal(t, "123", ctx.String("number"))
	assert.Equal(t, "true", ctx.String("bool"))
	assert.Equal(t, "", ctx.String("missing"))
}

func TestContextIntParsesVariousTypes(t *testing.T) {
	ctx := newTestContext(map[string]any{
		"fromInt":    42,
		"fromInt64":  int64(100),
		"fromFloat":  float64(55.7),
		"fromString": "99",
		"invalid":    "not-a-number",
	})

	assert.Equal(t, 42, ctx.Int("fromInt"))
	assert.Equal(t, 100, ctx.Int("fromInt64"))
	assert.Equal(t, 55, ctx.Int("fromFloat"))
	assert.Equal(t, 99, ctx.Int("fromString"))
	assert.Equal(t, 0, ctx.Int("invalid"))
	assert.Equal(t, 0, ctx.Int("missing"))
}

func TestContextInt64ParsesVariousTypes(t *testing.T) {
	ctx := newTestContext(map[string]any{
		"fromInt":    42,
		"fromInt64":  int64(100),
		"fromFloat":  float64(55.7),
		"fromString": "99",
	})

	assert.Equal(t, int64(42), ctx.Int64("fromInt"))
	assert.Equal(t, int64(100), ctx.Int64("fromInt64"))
	assert.Equal(t, int64(55), ctx.Int64("fromFloat"))
	assert.Equal(t, int64(99), ctx.Int64("fromString"))
	assert.Equal(t, int64(0), ctx.Int64("missing"))
}

func TestContextIsSetWithNilSetFlags(t *testing.T) {
	ctx := &Context{
		flags:    map[string]any{"test": "value"},
		setFlags: nil,
	}
	assert.False(t, ctx.IsSet("test"))
}
