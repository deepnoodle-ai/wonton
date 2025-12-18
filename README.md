# Wonton

[![Go Reference](https://pkg.go.dev/badge/github.com/deepnoodle-ai/wonton.svg)](https://pkg.go.dev/github.com/deepnoodle-ai/wonton)
[![Go Report Card](https://goreportcard.com/badge/github.com/deepnoodle-ai/wonton)](https://goreportcard.com/report/github.com/deepnoodle-ai/wonton)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

A savory toolkit for building CLI tools and terminal UIs in Go. Ideal for
_building_ agentic CLI tools like Claude Code that need rich terminal interfaces.

```bash
go get github.com/deepnoodle-ai/wonton@latest
```

## What's Inside

Wonton provides 18+ packages in a single Go module. Many of them can be used
independently, but they are also designed to work well together.

The `cli` and `tui` packages are the most important packages in Wonton. These
packages provide an ergonomic API for humans (and your coding agents) to rapidly
build command line tools that surprise and delight your users.

You can pick and choose the packages you need, or adopt all of Wonton. Keep in
mind that thanks to Go's compilation process, only the packages you import will
be included in your binary.

While Wonton provides a significant amount of functionality, it was designed to
have a minimal set of external dependencies beyond the standard library and the
extended standard library (`golang.org/x/...`).

In order to make Wonton a productive library for you and your coding agents,
there is an emphasis on providing packages that follow Go idioms and patterns
and providing comprehensive documentation and examples.

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

## Why One Module?

Each Wonton package is focused on a single responsibility. But collectively
the lot of them are versioned and released together in one Go module.

**Integration over isolation.** Every package is designed to work with the
others and is meant to provide a good foundation to build on. No compatibility
matrix, no dependency conflicts.

**Built for AI-assisted development.** Idiomatic APIs, thorough documentation,
and extensive examples across all packages mean AI coding agents can generate
correct code more reliably.

**Minimal dependencies.** Most functionality is implemented directly, keeping
your dependency graph and audit surface delighfully small. This is one building
block of minimizing supply chain complexity and risk.

## Who This Is For

Wonton works well if you're:

- Building CLI tools or terminal UIs in Go where user experience is important
- Creating agentic CLIs that work with any of: HTML, markdown, source code, or diffs
- Concerned about supply chain complexity and want to be selective with dependencies
- Looking to make your own AI-assisted development faster and more reliable

## Who This Isn't For

Consider alternatives if you would prefer to assemble your own stack or if you
want to use the more established CLI libraries in the Go ecosystem. For what
it's worth, though, you can adopt Wonton incrementally. It's not all or nothing.

## Packages

### Terminal UI

| Package                          | Description                         |
| -------------------------------- | ----------------------------------- |
| [tui](./tui/README.md)           | Declarative TUI with layout engine  |
| [terminal](./terminal/README.md) | Terminal control and input decoding |
| [color](./color/README.md)       | ANSI colors, RGB/HSL, gradients     |

### CLI Framework

| Package                | Description                         |
| ---------------------- | ----------------------------------- |
| [cli](./cli/README.md) | Commands, flags, config, middleware |
| [env](./env/README.md) | Config from env vars, .env, JSON    |

### Web & Networking

| Package                            | Description                   |
| ---------------------------------- | ----------------------------- |
| [fetch](./fetch/README.md)         | HTTP page fetching            |
| [crawler](./crawler/README.md)     | Web crawler with caching      |
| [htmlparse](./htmlparse/README.md) | HTML parsing, metadata, links |
| [htmltomd](./htmltomd/README.md)   | HTML to Markdown conversion   |
| [sse](./sse/README.md)             | Server-Sent Events client     |
| [web](./web/README.md)             | URL manipulation, media types |

### Utilities

| Package                            | Description                   |
| ---------------------------------- | ----------------------------- |
| [assert](./assert/README.md)       | Test assertions with diffs    |
| [clipboard](./clipboard/README.md) | System clipboard read/write   |
| [git](./git/README.md)             | Git read operations           |
| [humanize](./humanize/README.md)   | Human-readable formatting     |
| [retry](./retry/README.md)         | Retry with backoff and jitter |
| [unidiff](./unidiff/README.md)     | Unified diff parsing          |

### Testing & Recording

| Package                                | Description                          |
| -------------------------------------- | ------------------------------------ |
| [termtest](./termtest/README.md)       | Terminal output testing              |
| [termsession](./termsession/README.md) | Session recording (asciinema format) |
| [gif](./gif/README.md)                 | Animated GIF creation                |

## Serving Suggestions

Every folder under `examples/` is a standalone `main` package you can run directly:

| Category | Example                                | Description                     |
| -------- | -------------------------------------- | ------------------------------- |
| CLI      | `go run ./examples/cli/basic`          | Minimal command-line app        |
| CLI      | `go run ./examples/cli/flags`          | Flag types, defaults, and enums |
| TUI      | `go run ./examples/tui/text_animation` | Animated text with flex layout  |
| TUI      | `go run ./examples/tui/input_forms`    | Text input and form handling    |
| Web      | `go run ./examples/sitecheck`          | Link checker with live TUI      |
| Web      | `go run ./examples/webwatch`           | Page monitor with change diffs  |

See [examples/README.md](examples/README.md) for the full list.

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

We use it in production. The APIs are may evolve in minor ways before v1.0. Pin
your version.

**Why "Wonton"?**

Like its namesake: a delicious bundle of savory ingredients that you can drop 
into a larger recipe.

## Contributing

Pull requests welcome. Run tests before submitting:

```bash
go test ./...
```

See individual package READMEs for package-specific testing notes.

## License

[Apache License 2.0](LICENSE)
