# Wonton

[![Go Reference](https://pkg.go.dev/badge/github.com/deepnoodle-ai/wonton.svg)](https://pkg.go.dev/github.com/deepnoodle-ai/wonton)
[![Go Report Card](https://goreportcard.com/badge/github.com/deepnoodle-ai/wonton)](https://goreportcard.com/report/github.com/deepnoodle-ai/wonton)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

A tasty Go toolkit for CLI tools and terminal UIs. Especially suited for
building agentic tools like Claude Code.

```bash
go get github.com/deepnoodle-ai/wonton@latest
```

## What's Inside

The `cli` and `tui` packages form the core of Wonton, providing an ergonomic API
for building polished command line tools quickly, whether you're writing the
code yourself or working with AI coding agents.

Dependencies are minimal: mostly the standard library and `golang.org/x/...`.
The packages follow Go idioms, ship with thorough documentation, and include
examples throughout.

Pick the packages you need. Wonton provides 20+ packages that you can adopt
incrementally or all at once.

## Who This Is For

Wonton works well if you're:

- Building interactive Go CLIs where UX matters
- Building agentic tools like Claude Code, Gemini CLI, or Codex
- Working with HTML, markdown, source code, or diffs
- Using AI coding agents and want them to generate correct code
- Keeping dependencies minimal for supply chain security

## Why One Module?

Each package has a single responsibility, but they're versioned and released
together as one Go module.

**Integration over isolation.** Packages are designed to work together and
provide a solid foundation. No compatibility matrix. No dependency conflicts.

**Built for AI-assisted development.** Idiomatic APIs, thorough documentation,
and examples throughout help AI coding agents generate correct code.

**Minimal dependencies.** Most functionality is implemented directly, keeping
your dependency graph small. Fewer dependencies means less supply chain risk.

## Packages

| Package                                | Description                            |
| -------------------------------------- | -------------------------------------- |
| [assert](./assert/README.md)           | Test assertions with diffs             |
| [cli](./cli/README.md)                 | Commands, flags, config, middleware    |
| [clipboard](./clipboard/README.md)     | System clipboard read/write            |
| [color](./color/README.md)             | ANSI colors, RGB/HSL, gradients        |
| [crawler](./crawler/README.md)         | Web crawler with caching               |
| [env](./env/README.md)                 | Config from env vars, .env, JSON       |
| [fetch](./fetch/README.md)             | HTTP fetching with HTML to markdown    |
| [gif](./gif/README.md)                 | Animated GIF creation                  |
| [git](./git/README.md)                 | Read-only Git operations               |
| [htmlparse](./htmlparse/README.md)     | HTML parsing, metadata, links          |
| [htmltomd](./htmltomd/README.md)       | HTML to Markdown conversion            |
| [humanize](./humanize/README.md)       | Human-readable formatting              |
| [retry](./retry/README.md)             | Retry with backoff and jitter          |
| [sse](./sse/README.md)                 | Server-Sent Events client              |
| [terminal](./terminal/README.md)       | Terminal control and input decoding    |
| [termsession](./termsession/README.md) | Session recording (asciinema format)   |
| [termtest](./termtest/README.md)       | Terminal output testing                |
| [tui](./tui/README.md)                 | Declarative TUI with layout engine     |
| [unidiff](./unidiff/README.md)         | Unified diff parsing                   |
| [web](./web/README.md)                 | URL manipulation, media type detection |

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

Go downloads the module once but only compiles what you import.

**What are the external dependencies?**

Minimal. Check `go.mod` since that is the source of truth.

**Is this production-ready?**

We use it in production. APIs may evolve in minor ways before v1.0. Pin your version.

**Why "Wonton"?**

Like its namesake: a delicious bundle of savory ingredients that you can drop into a larger recipe.

## Contributing

Pull requests for bug fixes, ergonomic improvements, documentation, and tests
are welcome. We don't anticipate adding new packages, so please don't assume
a PR for one will be merged.

A few things we value:

- **Backwards compatibility.** Avoid breaking changes to existing APIs.
- **Ergonomics.** The APIs and codebase should be intuitive for both humans and AI agents.
- **Test coverage.** New code should include tests; improvements to existing
  coverage are appreciated.
- **When in doubt, reach out.** Open an issue to discuss before investing time
  in a large change.

Run tests before submitting:

```bash
go test ./...
```

See individual package READMEs for package-specific notes.

## License

[Apache License 2.0](LICENSE)
