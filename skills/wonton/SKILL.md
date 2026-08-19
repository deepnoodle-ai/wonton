---
name: wonton
description: >-
  Build Go CLI tools, terminal UIs, and agentic command-line apps with Wonton
  (github.com/deepnoodle-ai/wonton) — the cli, tui, fetch, crawler, env, git, schema, retry and
  related packages. Use when writing, reviewing, or debugging Go code that imports wonton, or when
  deciding which wonton package solves a task.
---

# Wonton

Wonton is one Go module — `github.com/deepnoodle-ai/wonton` — holding ~26 focused packages for
CLI and terminal-UI development. One module means no compatibility matrix and no version skew.
Dependencies are mostly stdlib plus a little `golang.org/x/...`. Import only what you need; Go
compiles only what you import.

```bash
go get github.com/deepnoodle-ai/wonton@latest   # requires Go 1.25+
```

```go
import "github.com/deepnoodle-ai/wonton/cli"
```

## Pick a package

| Task | Packages |
| --- | --- |
| Commands, flags, config, shell completion | `cli`, `env` |
| Fullscreen terminal UI | `tui` |
| Streaming output plus a pinned live status region | `tui` (InlineApp) |
| Styled one-shot output with no event loop | `tui.Print`, `color` |
| Fetch pages, crawl sites, HTML to Markdown | `fetch`, `crawler`, `htmlparse`, `htmltomd` |
| Outbound HTTP to URLs you did not choose | `httpguard`, `web` |
| LLM plumbing: tool schemas, token streams, retries | `schema`, `sse`, `retry` |
| Repo state, commits, diffs | `git`, `unidiff` |
| Human-readable output, colors, terminal cell widths | `humanize`, `color`, `runewidth` |
| Fallbacks, dedupe, pointer helpers | `strs`, `ptr` |
| Raw terminal control, clipboard | `terminal`, `clipboard` |
| Session recordings, GIFs, file previews | `termsession`, `gif`, `thumbnail` |
| Tests and assertions | `assert`, `termtest`, `cli.Test` |

Each row maps to a reference file listed under [Going deeper](#going-deeper).

## Minimal CLI

```go
package main

import (
	"fmt"
	"os"

	"github.com/deepnoodle-ai/wonton/cli"
)

func main() {
	app := cli.New("greeter").Description("Greet people").Version("1.0.0")

	app.Command("greet").
		Description("Greet someone").
		Args("name?").                                  // "?" marks the arg optional
		Flags(cli.Int("times", "t").Default(1).Help("Repeat count")).
		Run(func(ctx *cli.Context) error {
			name := ctx.Arg(0)
			if name == "" {
				name = "World"
			}
			for i := 0; i < ctx.Int("times"); i++ {
				ctx.Printf("Hello, %s!\n", name)
			}
			return nil
		})

	if err := app.Execute(); err != nil {
		if cli.IsHelpRequested(err) {
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(cli.GetExitCode(err))
	}
}
```

No subcommands? Configure the root command with `app.Main().Args(...).Flags(...).Run(...)`.

## Minimal TUI

```go
package main

import (
	"log"

	"github.com/deepnoodle-ai/wonton/tui"
)

type app struct{ count int }

func (a *app) View() tui.View {
	return tui.Stack(
		tui.Text("Counter: %d", a.count).Bold().Fg(tui.ColorGreen),
		tui.Text("[+/-] change  [q] quit").Dim(),
	).Align(tui.AlignCenter).Padding(2)
}

func (a *app) HandleEvent(event tui.Event) []tui.Cmd {
	if e, ok := event.(tui.KeyEvent); ok {
		switch {
		case e.Rune == 'q' || e.Key == tui.KeyCtrlC:
			return []tui.Cmd{tui.Quit()}
		case e.Rune == '+':
			a.count++
		case e.Rune == '-':
			a.count--
		}
	}
	return nil
}

func main() {
	if err := tui.Run(&app{}); err != nil {
		log.Fatal(err)
	}
}
```

## Rules that prevent most mistakes

1. `tui.Text` takes a **format string**: write `tui.Text("%s", s)`, never `tui.Text(s)` — `go vet`
   fails on the latter.
2. Never write app state from a `Cmd` goroutine, not even a `bool`. Return an event or call
   `SendEvent`, and mutate in `HandleEvent`.
3. Never use a mutex in a TUI app. `View()`/`LiveView()` and `HandleEvent()` never run
   concurrently; locking around `Print()` deadlocks.
4. `Init() error` cannot return commands. Kick off startup work from the first `TickEvent`
   (needs `tui.WithFPS(n)`) or from a goroutine that calls `SendEvent`.
5. Arrow keys are `tui.KeyArrowUp/Down/Left/Right` — there is no `KeyUp`/`KeyDown`.
6. A focused input consumes plain letters, so `q` will not reach `HandleEvent`. Always accept
   `tui.KeyCtrlC` as quit.
7. `.Fg()` takes ANSI `tui.Color` constants; true color needs `.FgRGB(r, g, b)` or
   `.Style(tui.NewStyle().WithFgRGB(...))`.
8. `.Animate(...)`, `.Shimmer(...)` and `.Pulse(...)` return a *different type* — call them
   last in a chain, and type the variable as `tui.View` when animation is conditional.
9. `fetch`/`crawler` default clients are SSRF-guarded: loopback, private, and link-local
   addresses are refused and `HTTP_PROXY`/`HTTPS_PROXY` are ignored. Pass an explicit `Client`
   to reach `localhost` or an intranet host.
10. Return `cli.Error("msg").Hint("try this")` from handlers; at the top level check
    `cli.IsHelpRequested(err)` and exit with `cli.GetExitCode(err)`.

## Verify your work

```bash
go build ./... && go vet ./... && go test ./...
```

Inside a clone of the wonton repo, every directory under `examples/` is a runnable program:
`go run ./examples/cli/basic --help`, `go run ./examples/tui/input_forms`. They are the most
reliable source of working patterns.

## Going deeper

- [references/cli.md](references/cli.md) — commands, groups, flags, context, middleware, errors
- [references/tui.md](references/tui.md) — views, layout, events, commands, InlineApp
- [references/web.md](references/web.md) — fetch, crawler, htmlparse, htmltomd, httpguard, sse
- [references/toolkit.md](references/toolkit.md) — env, retry, schema, git, small utilities
- [references/testing.md](references/testing.md) — assert, CLI tests, golden terminal snapshots

Upstream docs: package READMEs at `<pkg>/README.md` in the repo, godoc on
[pkg.go.dev](https://pkg.go.dev/github.com/deepnoodle-ai/wonton), and the machine-readable index
[llms.txt](https://raw.githubusercontent.com/deepnoodle-ai/wonton/main/llms.txt). When a detail
here disagrees with the source, the Go source and its godoc win.
