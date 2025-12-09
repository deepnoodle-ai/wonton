package cli

import (
	"bytes"
	"testing"

	"github.com/deepnoodle-ai/wonton/require"
)

func newTestContext(flags map[string]any, args ...string) *Context {
	if flags == nil {
		flags = make(map[string]any)
	}
	return &Context{
		flags:      flags,
		positional: args,
		stdout:     &bytes.Buffer{},
		stderr:     &bytes.Buffer{},
	}
}

func TestContextIntParsesString(t *testing.T) {
	ctx := newTestContext(map[string]any{
		"count": "42",
	})

	require.Equal(t, 42, ctx.Int("count"))
}

func TestContextFloat64ParsesVariousTypes(t *testing.T) {
	ctx := newTestContext(map[string]any{
		"fromInt":   5,
		"fromInt64": int64(10),
		"fromStr":   "3.14",
	})

	require.Equal(t, 5.0, ctx.Float64("fromInt"))
	require.Equal(t, 10.0, ctx.Float64("fromInt64"))
	require.InDelta(t, 3.14, ctx.Float64("fromStr"), 0.0001)
}

func TestContextBoolParsesStrings(t *testing.T) {
	ctx := newTestContext(map[string]any{
		"truthy": "yes",
		"falsey": "0",
	})

	require.True(t, ctx.Bool("truthy"))
	require.False(t, ctx.Bool("falsey"))
	require.False(t, ctx.Bool("missing"))
}

func TestContextIsSet(t *testing.T) {
	ctx := newTestContext(map[string]any{
		"flag": true,
	})

	require.True(t, ctx.IsSet("flag"))
	require.False(t, ctx.IsSet("other"))
}

func TestContextArgs(t *testing.T) {
	ctx := newTestContext(nil, "first", "second")

	require.Equal(t, 2, ctx.NArg())
	require.Equal(t, "first", ctx.Arg(0))
	require.Equal(t, "second", ctx.Arg(1))
	require.Equal(t, "", ctx.Arg(2))
}
