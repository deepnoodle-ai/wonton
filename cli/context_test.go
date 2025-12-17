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
