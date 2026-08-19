package cli

import (
	"strings"
	"testing"
	"time"

	"github.com/deepnoodle-ai/wonton/assert"
)

// Env var values must be converted to the flag's type, so typed accessors
// like Ints and Duration work whether the value came from the command line
// or the environment.
func TestEnvVarTypedConversion(t *testing.T) {
	t.Setenv("TEST_COUNT", "42")
	t.Setenv("TEST_VERBOSE", "yes")
	t.Setenv("TEST_TIMEOUT", "90s")
	t.Setenv("TEST_RATE", "0.25")
	t.Setenv("TEST_TAGS", "alpha, beta,gamma")
	t.Setenv("TEST_PORTS", "80, 443,8080")

	var (
		count   int
		verbose bool
		timeout time.Duration
		rate    float64
		tags    []string
		ports   []int
	)

	app := New("test").Description("Test")
	app.Command("run").
		Description("Run").
		Flags(
			Int("count", "c").Env("TEST_COUNT"),
			Bool("verbose", "v").Env("TEST_VERBOSE"),
			Duration("timeout", "t").Env("TEST_TIMEOUT"),
			Float64("rate", "r").Env("TEST_RATE"),
			Strings("tags", "").Env("TEST_TAGS"),
			Ints("ports", "p").Env("TEST_PORTS"),
		).
		Run(func(ctx *Context) error {
			count = ctx.Int("count")
			verbose = ctx.Bool("verbose")
			timeout = ctx.Duration("timeout")
			rate = ctx.Float64("rate")
			tags = ctx.Strings("tags")
			ports = ctx.Ints("ports")
			return nil
		})

	err := app.ExecuteArgs([]string{"run"})
	assert.NoError(t, err)
	assert.Equal(t, 42, count)
	assert.True(t, verbose)
	assert.Equal(t, 90*time.Second, timeout)
	assert.InDelta(t, 0.25, rate, 0.0001)
	assert.Equal(t, []string{"alpha", "beta", "gamma"}, tags)
	assert.Equal(t, []int{80, 443, 8080}, ports)
}

func TestEnvVarInvalidValueErrors(t *testing.T) {
	t.Setenv("TEST_COUNT", "not-a-number")

	app := New("test").Description("Test")
	app.Command("run").
		Description("Run").
		Flags(Int("count", "c").Env("TEST_COUNT")).
		Run(func(ctx *Context) error { return nil })

	err := app.ExecuteArgs([]string{"run"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "TEST_COUNT")
}

func TestContextDurationGetter(t *testing.T) {
	var fromDefault, fromFlag time.Duration

	app := New("test").Description("Test")
	app.Command("run").
		Description("Run").
		Flags(
			Duration("timeout", "t").Default(30*time.Second),
			Duration("interval", "i"),
		).
		Run(func(ctx *Context) error {
			fromDefault = ctx.Duration("timeout")
			fromFlag = ctx.Duration("interval")
			return nil
		})

	err := app.ExecuteArgs([]string{"run", "--interval", "1h15m"})
	assert.NoError(t, err)
	assert.Equal(t, 30*time.Second, fromDefault)
	assert.Equal(t, 75*time.Minute, fromFlag)
}

func TestParseIntRejectsTrailingGarbage(t *testing.T) {
	_, err := parseInt("12abc")
	assert.Error(t, err)

	n, err := parseInt(" 42 ")
	assert.NoError(t, err)
	assert.Equal(t, 42, n)
}

// Zero-valued defaults should be omitted from help text for every flag type,
// including typed zeros like time.Duration(0) and empty slices.
func TestWriteFlagHelpOmitsZeroDefaults(t *testing.T) {
	var sb strings.Builder
	writeFlagHelp(&sb, Duration("timeout", "t").Help("Timeout"), 10)
	writeFlagHelp(&sb, Strings("tags", "").Help("Tags"), 10)
	writeFlagHelp(&sb, Ints("ports", "p").Help("Ports"), 10)
	writeFlagHelp(&sb, Float64("rate", "r").Help("Rate"), 10)
	assert.False(t, strings.Contains(sb.String(), "default"), "zero defaults should be hidden, got: %s", sb.String())

	sb.Reset()
	writeFlagHelp(&sb, Duration("timeout", "t").Default(30*time.Second).Help("Timeout"), 10)
	assert.Contains(t, sb.String(), "(default: 30s)")
}
