# wonton

Wonton is a batteries-included toolkit for building terminal-first software in Go.
It combines a composable TUI engine, an opinionated CLI framework, low-level
terminal primitives, and a handful of utility packages that make building
operator tools and AI copilots enjoyable.

The repository is organized as a single Go module
(`github.com/deepnoodle-ai/wonton`) so you can `go get` exactly the packages you
need.

```bash
go get github.com/deepnoodle-ai/wonton@latest
```

## Quick start: Declarative TUI

```go
package main

import (
	"log"

	"github.com/deepnoodle-ai/wonton/tui"
)

type counter struct{ n int }

func (c *counter) View() tui.View {
	return tui.Stack(
		tui.Text("Count: %d", c.n),
		tui.Clickable("[+]", func() { c.n++ }),
		tui.Clickable("Reset", func() { c.n = 0 }),
	).Gap(1)
}

func (c *counter) HandleEvent(ev tui.Event) []tui.Cmd {
	if key, ok := ev.(tui.KeyEvent); ok && key.Rune == 'q' {
		return []tui.Cmd{tui.Quit()}
	}
	return nil
}

func main() {
	if err := tui.Run(&counter{}); err != nil {
		log.Fatal(err)
	}
}
```

Run the program with `go run .`, then press `q` or activate the Quit button to
exit.

## Packages at a glance

- `tui`: Declarative layout engine, runtime, and ready-made views (lists, tables,
  markdown, spinners, forms).
- `cli`: Command framework that automatically switches between one-shot and
  progressive (TUI) modes, supports groups, global flags, completions, and
  middleware.
- `terminal`: Low-level control over ANSI terminals with double buffering,
  hyperlink support, recording/replay, and resize/mouse helpers.
- `env`: Parse strongly typed configuration structures from environment
  variables, `.env` files, and JSON files with validation hooks.
- `retry`: Context-aware retries with exponential or custom backoff, jitter, and
  callbacks.
- `slog`: Colorized `log/slog` handler that understands structured attributes.
- `assert`: Lightweight testing helpers with detailed diffs.
- `humanize`: Helpers for formatting numbers, durations, relative times, and
  byte sizes.
- `sse`: Streaming parser and reconnecting HTTP client for Server-Sent Events.
- `examples`: Over 30 runnable demos that showcase the CLI and TUI stacks.

Browse each package directory (for example `tui/README.md`) for focused
documentation and usage notes.

## Examples

Every folder under `examples/` is a standalone `main` package. Run any example
with `go run`:

```bash
go run ./examples/cli_basic
go run ./examples/table_demo
```

The [examples/README.md](examples/README.md) file lists the demos by category
and highlights a few good starting points.

## Contributing

Pull requests are welcome—especially new examples that exercise tricky parts of
the runtime. Run `go test ./...` before submitting changes, and feel free to add
new example directories under `examples/` to showcase additional use cases.

## License

This project is licensed under the [MIT License](LICENSE).
