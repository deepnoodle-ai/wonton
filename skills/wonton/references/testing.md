# testing — assertions, CLI tests, terminal snapshots

Packages: `assert` (general assertions), `cli` test helpers, `tui` render helpers, `termtest`
(virtual screens and golden files).

## assert

Thin wrappers over `testing.TB` that print a colored diff on failure. They call `t.Errorf` and
keep going, so a test can report several problems at once.

```go
import "github.com/deepnoodle-ai/wonton/assert"

func TestParse(t *testing.T) {
	got, err := Parse("input")
	assert.NoError(t, err)
	assert.Equal(t, got.Name, "expected")            // (t, got, want)
	assert.Len(t, got.Items, 3)
	assert.Contains(t, got.Body, "substring")
	assert.True(t, got.Valid, "record %d should be valid", got.ID)
}
```

Available: `Equal`, `EqualOpts` (takes `go-cmp` options), `NotEqual`, `NoError`, `Error`,
`ErrorIs`, `ErrorAs`, `ErrorContains`, `Nil`, `NotNil`, `True`, `False`, `Contains`,
`NotContains`, `Len`, `Empty`, `NotEmpty`, `Panics`, `NotPanics`, `Greater`, `GreaterOrEqual`,
`Less`, `LessOrEqual`, `InDelta`, `Regexp`. Every one accepts trailing `msgAndArgs ...any`.

`assert.EqualOpts` is the escape hatch for unexported fields or float tolerance:

```go
assert.EqualOpts(t, got, want, cmpopts.IgnoreUnexported(Config{}))
```

## Testing a CLI

`app.Test` runs the app with captured I/O and returns a result you can inspect.

```go
func TestGreet(t *testing.T) {
	app := newApp()
	result := app.Test(t, cli.TestArgs("greet", "Alice", "--times", "2"))

	if !result.Success() {
		t.Fatalf("command failed: %v", result.Err)
	}
	if !result.Contains("Hello, Alice!") {
		t.Errorf("unexpected output: %s", result.Stdout)
	}
}
```

For full control, wire the streams yourself and call `ExecuteArgs`:

```go
var stdout, stderr bytes.Buffer
app.SetStdout(&stdout)
app.SetStderr(&stderr)
app.SetStdin(strings.NewReader("yes\n"))
err := app.ExecuteArgs([]string{"deploy", "prod"})
```

`app.ForceInteractive(true)` makes TTY-gated paths (`ctx.Interactive()`, prompts) testable without
a real terminal.

**Write handlers against `ctx.Printf` / `ctx.Stdout()`, never `fmt.Printf` or `os.Stdout`** —
otherwise the output escapes capture and the test cannot see it.

## Testing views

`tui.SprintScreen` renders a view into a virtual screen at fixed dimensions; `termtest` asserts on
it. No terminal, no event loop, deterministic output.

```go
func TestGolden_Summary(t *testing.T) {
	view := tui.Stack(
		tui.Text("Header").Bold(),
		tui.Divider(),
		tui.Text("%d items", 3),
	)

	screen := tui.SprintScreen(view, tui.WithWidth(30), tui.WithHeight(10))

	termtest.AssertContains(t, screen, "3 items")
	termtest.AssertRow(t, screen, 0, "Header")
	termtest.AssertScreen(t, screen)                 // compares against a golden file
}
```

`tui.Sprint(view, tui.WithWidth(80))` returns the rendered string when you only need text.

`termtest` assertions: `AssertContains`, `AssertNotContains`, `AssertRow`, `AssertRowContains`,
`AssertRowPrefix`, `AssertTextEqual`, `AssertCursor`, `AssertCell`, `AssertCellGlyph`,
`AssertCellStyle`, `AssertCellBold`, `AssertEmpty`, `AssertEqual`, plus `RequireContains` and
`RequireRow` which stop the test. Snapshot helpers: `AssertScreen`, `AssertScreenNamed`,
`AssertText`, `AssertTextNamed`.

`termtest.NewCapture(w)`, `NewRecorder(w, h)`, and `NewBuffer()` feed raw ANSI output into a
`Screen` when you are testing something that writes escape codes directly.

## Golden files

Snapshots live in `testdata/snapshots/<TestName>.snap` next to the test. Create or refresh them
with the update flag, then read the diff before committing:

```bash
go test ./tui -run TestGolden -update       # flag comes after the package path
go test ./tui -run TestGolden               # verify
```

Snapshots are a contract: an unexplained snapshot change is a rendering regression, so review the
diff rather than regenerating on autopilot.

## Testing async app logic

Event handling is a pure function of state — call it directly, no runtime needed:

```go
app := &myApp{}
cmds := app.HandleEvent(tui.KeyEvent{Rune: 'j'})
assert.Equal(t, app.cursor, 1)
assert.Len(t, cmds, 0)

// A Cmd is just a func() Event — run it and inspect the result.
event := app.load()()
loaded, ok := event.(loadedEvent)
assert.True(t, ok)
```

For network-dependent code, `fetch.NewMockFetcher()` substitutes for `HTTPFetcher`.

## Gotchas

- `assert.Equal(t, got, want)` — got first. Reversing it inverts every failure message.
- Assertions do not stop the test; use `t.Fatal` yourself after a check that makes the rest
  meaningless, or the `Require*` helpers in `termtest`.
- Fix the width in `SprintScreen`; a view rendered at the ambient terminal width is not
  reproducible across machines.
- Use `runewidth.StringWidth` rather than `len()` when asserting on column alignment.
