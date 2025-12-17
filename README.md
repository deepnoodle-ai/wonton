# Wonton

Wonton is a curated collection of Go packages for rapid application development.
It provides a solid foundation of utilities, terminal UI components, and CLI
building blocks that work well together.

```bash
go get github.com/deepnoodle-ai/wonton@latest
```

## Why Wonton?

**Rapid development on a strong foundation.** Wonton integrates functionality
common to modern CLI applications and AI agent development, so you can focus on
your application logic instead of stitching together dependencies.

**Optimized for AI-assisted development.** Wonton is designed to be easily
consumed by AI coding agents like Claude Code. Extensive examples, thorough
documentation, and consistent APIs across all packages make it easy for agents
to write correct code on the first try.

**Build agentic CLIs.** Wonton is ideal for building your own agentic command-line
tools like Claude Code—combining rich terminal UIs with the utilities AI agents
need: HTML-to-Markdown conversion, SSE streaming, retry logic, and more.

**Minimal dependencies.** Where practical, functionality is implemented directly
rather than pulling in external packages. Consolidating common functionality into
a single, well-maintained module reduces supply chain complexity and audit surface.

## Packages

### Terminal UI

| Package      | Description                                                       |
| ------------ | ----------------------------------------------------------------- |
| **tui**      | Declarative TUI library with layout engine, views, and components |
| **terminal** | Low-level terminal control, input decoding, double buffering      |
| **color**    | ANSI color types, RGB/HSL conversion, gradients                   |

### CLI Framework

| Package | Description                                                       |
| ------- | ----------------------------------------------------------------- |
| **cli** | Command framework with flags, config, middleware, and completions |
| **env** | Config loading from environment variables, .env files, and JSON   |

### Web & Networking

| Package       | Description                                                |
| ------------- | ---------------------------------------------------------- |
| **fetch**     | HTTP page fetching with configurable options               |
| **crawler**   | Web crawler with pluggable fetchers, parsers, and caching  |
| **htmlparse** | HTML parsing, metadata extraction, link discovery          |
| **htmltomd**  | HTML to Markdown conversion, ideal for LLM consumption     |
| **sse**       | Server-Sent Events parser and reconnecting HTTP client     |
| **web**       | URL manipulation, text normalization, media type detection |

### Utilities

| Package       | Description                                             |
| ------------- | ------------------------------------------------------- |
| **assert**    | Test assertions with detailed diffs                     |
| **clipboard** | Cross-platform system clipboard read/write              |
| **git**       | Wrapper for common git read operations                  |
| **humanize**  | Human-readable formatting for bytes, durations, numbers |
| **retry**     | Retry with exponential backoff, jitter, and callbacks   |
| **unidiff**   | Unified diff parsing for display and analysis           |

### Testing & Recording

| Package         | Description                                               |
| --------------- | --------------------------------------------------------- |
| **termtest**    | Terminal output testing with ANSI parsing and snapshots   |
| **termsession** | Terminal session recording/playback (asciinema v2 format) |
| **gif**         | Animated GIF creation with drawing primitives             |

Browse each package directory (e.g., `tui/README.md`) for detailed documentation.

## Examples

Every folder under `examples/` is a standalone `main` package:

```bash
go run ./examples/cli/basic
```

See [examples/README.md](examples/README.md) for the full list organized by category.

## Contributing

Pull requests are welcome. Run `go test ./...` before submitting changes.

## License

[Apache License 2.0](LICENSE)
