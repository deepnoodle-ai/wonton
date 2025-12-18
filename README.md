# Wonton 🥟

[![Go Reference](https://pkg.go.dev/badge/github.com/deepnoodle-ai/wonton.svg)](https://pkg.go.dev/github.com/deepnoodle-ai/wonton)
[![Go Report Card](https://goreportcard.com/badge/github.com/deepnoodle-ai/wonton)](https://goreportcard.com/report/github.com/deepnoodle-ai/wonton)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

A savory toolkit for building CLI tools and terminal UIs in Go. Ideal for
agentic CLI tools like Claude Code, Gemini CLI, or Codex that need rich terminal
interfaces, Markdown rendering, syntax highlighting, diffs, and HTML parsing.

```bash
go get github.com/deepnoodle-ai/wonton@latest
```

## What's Inside

Wonton provides 18 packages in a single module—CLI framework, TUI components,
Markdown rendering, syntax highlighting, diff display, HTML parsing, and
utilities. Most packages are implemented from scratch in pure Go. A few build on
[Goldmark](https://github.com/yuin/goldmark) for Markdown and
[Chroma](https://github.com/alecthomas/chroma) for highlighting. Import just the
packages you need, or use the whole thing.

```go
package main

import (
    "github.com/deepnoodle-ai/wonton/cli"
    "github.com/deepnoodle-ai/wonton/color"
)

func main() {
    app := cli.New("greet").Description("A friendly greeter")

    app.Command("hello").
        Description("Say hello").
        Args("name?").
        Run(func(ctx *cli.Context) error {
            name := ctx.Arg(0)
            if name == "" {
                name = "World"
            }
            ctx.Println(color.Green("Hello, %s!", name))
            return nil
        })

    app.Execute()
}
```

## Why One Wrapper?

Each Wonton package is small and focused—the Go way. But they're versioned and
released together in one module, which simplifies dependency management when you
need several of them.

**Integration over isolation.** Every package is designed to work with the others.
The TUI components, CLI framework, and utilities share consistent patterns and are
versioned together. No compatibility matrix, no dependency conflicts.

**Built for AI-assisted development.** Consistent APIs, thorough documentation, and
extensive examples across all packages mean AI coding tools generate correct code
more reliably. Predictable patterns reduce hallucinations.

**Pure Go, no CGO.** The dependency footprint is minimal. Most functionality is
implemented directly, keeping your dependency graph flat and your audit surface
small.

**Focused, not expansive.** Each package covers core functionality well, with
enough flexibility to handle common variations. We're not trying to be comprehensive—
just reliably useful.

This isn't the right choice for every project. If you only need one small utility,
a focused single-purpose package may suit you better. But if your project needs
three or more of these capabilities, Wonton saves you from wiring them together.

## Packages

### Terminal UI

| Package                          | Description                                                  | Use Cases                                      |
| -------------------------------- | ------------------------------------------------------------ | ---------------------------------------------- |
| [tui](./tui/README.md)           | Declarative TUI library with layout engine and components    | Dashboards, interactive tools, rich CLI output |
| [terminal](./terminal/README.md) | Low-level terminal control, input decoding, double buffering | Custom renderers, raw terminal access          |
| [color](./color/README.md)       | ANSI color types, RGB/HSL conversion, gradients              | Syntax highlighting, themed output             |

### CLI Framework

| Package                | Description                                            | Use Cases                    |
| ---------------------- | ------------------------------------------------------ | ---------------------------- |
| [cli](./cli/README.md) | Commands, flags, config, middleware, shell completions | Any command-line application |
| [env](./env/README.md) | Config from environment variables, .env files, JSON    | Application configuration    |

### Web & Networking

| Package                            | Description                                           | Use Cases                              |
| ---------------------------------- | ----------------------------------------------------- | -------------------------------------- |
| [fetch](./fetch/README.md)         | HTTP page fetching with configurable options          | Web scraping, API clients              |
| [crawler](./crawler/README.md)     | Web crawler with pluggable fetchers, parsers, caching | Site indexing, link analysis           |
| [htmlparse](./htmlparse/README.md) | HTML parsing, metadata extraction, link discovery     | Content extraction, SEO tools          |
| [htmltomd](./htmltomd/README.md)   | HTML to Markdown conversion                           | LLM context preparation, documentation |
| [sse](./sse/README.md)             | Server-Sent Events parser and reconnecting client     | Streaming APIs, real-time updates      |
| [web](./web/README.md)             | URL manipulation, text normalization, media types     | URL processing, content detection      |

### Utilities

| Package                            | Description                                   | Use Cases                         |
| ---------------------------------- | --------------------------------------------- | --------------------------------- |
| [assert](./assert/README.md)       | Test assertions with detailed diffs           | Unit testing                      |
| [clipboard](./clipboard/README.md) | Cross-platform clipboard read/write           | Copy/paste integration            |
| [git](./git/README.md)             | Wrapper for common git operations             | Repository tooling                |
| [humanize](./humanize/README.md)   | Human-readable bytes, durations, numbers      | User-facing output                |
| [retry](./retry/README.md)         | Exponential backoff with jitter and callbacks | Resilient network calls           |
| [unidiff](./unidiff/README.md)     | Unified diff parsing                          | Code review tools, patch analysis |

### Testing & Recording

| Package                                | Description                                   | Use Cases                    |
| -------------------------------------- | --------------------------------------------- | ---------------------------- |
| [termtest](./termtest/README.md)       | Terminal output testing with ANSI parsing     | TUI testing, snapshot tests  |
| [termsession](./termsession/README.md) | Session recording/playback (asciinema v2)     | Demos, documentation         |
| [gif](./gif/README.md)                 | Animated GIF creation with drawing primitives | Demo generation, visual docs |

## Serving Suggestions

Every folder under `examples/` is a standalone `main` package you can run directly:

| Category | Example                                | Description                        |
| -------- | -------------------------------------- | ---------------------------------- |
| CLI      | `go run ./examples/cli/basic`          | Minimal command-line app           |
| CLI      | `go run ./examples/cli/flags`          | Flag types, defaults, and enums    |
| TUI      | `go run ./examples/tui/text_animation` | Animated text with flex layout     |
| TUI      | `go run ./examples/tui/input_forms`    | Text input and form handling       |
| Web      | `go run ./examples/sitecheck`          | Link checker with live TUI         |
| Web      | `go run ./examples/webwatch`           | Page monitor with change diffs     |

See [examples/README.md](examples/README.md) for the full list.

## Who This Is For

Wonton works well if you're:

- Building CLI tools or terminal UIs in Go
- Creating agentic CLIs that need web fetching, HTML parsing, or streaming
- Prototyping quickly and want batteries included
- Tired of wiring together a dozen small dependencies

## Who This Isn't For

Consider alternatives if you:

- Need only one specific capability—a single-purpose library may be simpler
- Require absolute minimal binary size (though Go only compiles what you import)
- Prefer curating your own stack from individual packages

## Requirements

- Go 1.21 or later
- Pure Go—no CGO required

## FAQ

**Can I import just one package?**

Yes, and this is expected:

```go
import "github.com/deepnoodle-ai/wonton/htmltomd"
```

Go downloads the module once but only compiles what you import. Your binary
includes only the packages you use.

**What are the external dependencies?**

Minimal. Check `go.mod` for the current list. Most functionality is pure Go.

**Is this production-ready?**

We use it in production. The APIs are stable but may evolve before v1.0. Pin
your version.

**Why "Wonton"?**

Like its namesake: thin wrapper, savory filling, and improves whatever you
drop it into.

## Contributing

Pull requests welcome. Run tests before submitting:

```bash
go test ./...
```

See individual package READMEs for package-specific testing notes.

## License

[Apache License 2.0](LICENSE)
