package cli

import (
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

// --- DSL parser ---------------------------------------------------------

func TestParseArgSpec(t *testing.T) {
	cases := []struct {
		spec     string
		name     string
		required bool
		variadic bool
	}{
		{"name", "name", true, false},
		{"name?", "name", false, false},
		{"urls...", "urls", true, true},
		{"urls?...", "urls", false, true},
		{"urls...?", "urls", false, true},
		{"file-path", "file-path", true, false},
		{"file_path", "file_path", true, false},
		{"A1", "A1", true, false},
	}
	for _, tc := range cases {
		t.Run(tc.spec, func(t *testing.T) {
			got := parseArgSpec(tc.spec)
			assert.Equal(t, tc.name, got.Name)
			assert.Equal(t, tc.required, got.Required)
			assert.Equal(t, tc.variadic, got.Variadic)
		})
	}
}

func TestParseArgSpec_PanicsOnMalformed(t *testing.T) {
	cases := []struct {
		name string
		spec string
	}{
		{"empty", ""},
		{"just question", "?"},
		{"just ellipsis", "..."},
		{"double variadic", "urls......"},
		{"double optional", "name??"},
		{"bad char", "name!"},
		{"space in name", "a name"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Panics(t, func() { parseArgSpec(tc.spec) })
		})
	}
}

func TestArgs_PanicsOnInvalidLayout(t *testing.T) {
	t.Run("variadic not last", func(t *testing.T) {
		assert.Panics(t, func() {
			New("test").Command("c").Args("files...", "tail")
		})
	})

	t.Run("required after optional", func(t *testing.T) {
		assert.Panics(t, func() {
			New("test").Command("c").Args("a?", "b")
		})
	})

	t.Run("duplicate names", func(t *testing.T) {
		assert.Panics(t, func() {
			New("test").Command("c").Args("src", "src")
		})
	})

	t.Run("two variadic slots", func(t *testing.T) {
		assert.Panics(t, func() {
			New("test").Command("c").Args("a...", "b...")
		})
	})
}

func TestArgs_LayoutChecksAcrossMultipleCalls(t *testing.T) {
	// The layout invariants must hold across the accumulated slot list,
	// not just within a single Args call.
	assert.Panics(t, func() {
		New("test").Command("c").Args("a?").Args("b")
	})
	assert.Panics(t, func() {
		New("test").Command("c").Args("a...").Args("b")
	})
}

// --- Parse-time behavior ------------------------------------------------

func TestArgs_RejectsExtrasByDefault(t *testing.T) {
	app := New("test")
	app.Command("pair").
		Args("left", "right").
		Run(func(ctx *Context) error { return nil })

	err := app.ExecuteArgs([]string{"pair", "a", "b", "c"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected argument: c")
}

func TestArgs_NoSlotsRejectsPositionals(t *testing.T) {
	// Commands that don't declare Args accept no positionals.
	app := New("test")
	app.Command("status").Run(func(ctx *Context) error { return nil })

	err := app.ExecuteArgs([]string{"status", "oops"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected argument: oops")
}

func TestArgs_MissingRequired(t *testing.T) {
	app := New("test")
	app.Command("pair").
		Args("left", "right").
		Run(func(ctx *Context) error { return nil })

	err := app.ExecuteArgs([]string{"pair", "a"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing required argument: right")
}

func TestArgs_OptionalFillsFromDefault(t *testing.T) {
	app := New("test")
	var got []string
	app.Command("copy").
		Args("src", "dst?").
		Run(func(ctx *Context) error {
			got = ctx.Args()
			return nil
		})

	err := app.ExecuteArgs([]string{"copy", "a"})
	assert.NoError(t, err)
	assert.Equal(t, []string{"a"}, got)
}

// --- Variadic slots -----------------------------------------------------

func TestArgs_VariadicRequiredOneOrMore(t *testing.T) {
	app := New("test")
	var got []string
	app.Command("crawl").
		Args("urls...").
		Run(func(ctx *Context) error {
			got = ctx.Args()
			return nil
		})

	// Zero values → error, since variadic without "?" means one or more.
	err := app.ExecuteArgs([]string{"crawl"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing required argument: urls")

	// Single value.
	err = app.ExecuteArgs([]string{"crawl", "https://a"})
	assert.NoError(t, err)
	assert.Equal(t, []string{"https://a"}, got)

	// Many values.
	got = nil
	err = app.ExecuteArgs([]string{"crawl", "https://a", "https://b", "https://c"})
	assert.NoError(t, err)
	assert.Equal(t, []string{"https://a", "https://b", "https://c"}, got)
}

func TestArgs_VariadicOptionalZeroOrMore(t *testing.T) {
	app := New("test")
	var got []string
	app.Command("touch").
		Args("files?...").
		Run(func(ctx *Context) error {
			got = ctx.Args()
			return nil
		})

	// Zero is fine.
	err := app.ExecuteArgs([]string{"touch"})
	assert.NoError(t, err)
	assert.Empty(t, got)

	// Many still fine.
	err = app.ExecuteArgs([]string{"touch", "a", "b"})
	assert.NoError(t, err)
	assert.Equal(t, []string{"a", "b"}, got)
}

func TestArgs_FixedThenVariadic(t *testing.T) {
	app := New("test")
	var src string
	var dsts []string
	app.Command("copy").
		Args("src", "dsts...").
		Run(func(ctx *Context) error {
			src = ctx.Arg(0)
			dsts = ctx.Args()[1:]
			return nil
		})

	err := app.ExecuteArgs([]string{"copy", "source.txt", "a", "b", "c"})
	assert.NoError(t, err)
	assert.Equal(t, "source.txt", src)
	assert.Equal(t, []string{"a", "b", "c"}, dsts)

	// Missing variadic → error because "dsts..." is required.
	err = app.ExecuteArgs([]string{"copy", "source.txt"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing required argument: dsts")
}

func TestArgs_VariadicAlternateSyntax(t *testing.T) {
	// "name?..." and "name...?" produce identical slots.
	a := parseArgSpec("urls?...")
	b := parseArgSpec("urls...?")
	assert.Equal(t, a.Name, b.Name)
	assert.Equal(t, a.Required, b.Required)
	assert.Equal(t, a.Variadic, b.Variadic)
}

// --- App root + Group surfaces ------------------------------------------

func TestArgs_AppRootDSL(t *testing.T) {
	app := New("test")
	var got []string
	app.Main().
		Args("paths?...").
		Run(func(ctx *Context) error {
			got = ctx.Args()
			return nil
		})

	err := app.ExecuteArgs([]string{})
	assert.NoError(t, err)
	assert.Empty(t, got)

	err = app.ExecuteArgs([]string{"a", "b"})
	assert.NoError(t, err)
	assert.Equal(t, []string{"a", "b"}, got)
}

func TestArgs_GroupDSL(t *testing.T) {
	app := New("test")
	var got []string
	app.Group("files").
		Args("path").
		Run(func(ctx *Context) error {
			got = ctx.Args()
			return nil
		})

	// Missing required.
	err := app.ExecuteArgs([]string{"files"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing required argument: path")

	// OK with the single required arg.
	err = app.ExecuteArgs([]string{"files", "/tmp"})
	assert.NoError(t, err)
	assert.Equal(t, []string{"/tmp"}, got)

	// Extras rejected.
	err = app.ExecuteArgs([]string{"files", "/tmp", "/var"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected argument: /var")
}

// --- AddArg validation --------------------------------------------------

func TestAddArg_ValidatesLayout(t *testing.T) {
	assert.Panics(t, func() {
		New("test").Command("c").
			AddArg(&Arg{Name: "a", Required: false}).
			AddArg(&Arg{Name: "b", Required: true})
	})

	assert.Panics(t, func() {
		New("test").Command("c").
			AddArg(&Arg{Name: "v", Required: true, Variadic: true}).
			AddArg(&Arg{Name: "tail", Required: true})
	})
}

// --- Help / usage rendering --------------------------------------------

func TestUsage_VariadicRequired(t *testing.T) {
	app := New("myapp")
	cmd := app.Command("crawl").Args("urls...")
	usage := buildUsageString(cmd)
	assert.Contains(t, usage, "<urls...>")
}

func TestUsage_VariadicOptional(t *testing.T) {
	app := New("myapp")
	cmd := app.Command("touch").Args("files?...")
	usage := buildUsageString(cmd)
	assert.Contains(t, usage, "[files...]")
}

func TestUsage_MixedSlots(t *testing.T) {
	app := New("myapp")
	cmd := app.Command("copy").Args("src", "dst?", "extra?...")
	usage := buildUsageString(cmd)
	assert.Contains(t, usage, "<src>")
	assert.Contains(t, usage, "[dst]")
	assert.Contains(t, usage, "[extra...]")
}

func TestUsage_RootVariadic(t *testing.T) {
	app := New("myapp")
	app.Main().Args("paths?...").Run(func(ctx *Context) error { return nil })
	usage := app.buildRootUsageString()
	assert.Contains(t, usage, "[paths...]")
}

func TestArgHintLabel(t *testing.T) {
	cases := []struct {
		arg  *Arg
		want string
	}{
		{&Arg{Required: true}, ""},
		{&Arg{Required: false}, " (optional)"},
		{&Arg{Required: true, Variadic: true}, " (one or more)"},
		{&Arg{Required: false, Variadic: true}, " (zero or more)"},
	}
	for _, tc := range cases {
		got := argHintLabel(tc.arg)
		assert.Equal(t, tc.want, got)
	}
}
