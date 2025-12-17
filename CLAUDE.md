# CLAUDE.md

This file provides guidance to Claude Code when working with this codebase.

## Project Overview

Wonton is a curated collection of Go packages designed for rapid application
development. It provides a solid foundation of utilities, terminal UI components,
and CLI building blocks that work well together.

### Why Wonton?

- **Rapid development**: Integrates functionality common to modern CLI applications
  and AI agent development, so you can focus on application logic.
- **Optimized for AI-assisted development**: Designed to be easily consumed by AI
  coding agents like Claude Code. Extensive examples, thorough documentation, and
  consistent APIs make it easy for agents to write correct code.
- **Build agentic CLIs**: Ideal for building agentic command-line tools like
  Claude Code—combining rich terminal UIs with utilities AI agents need.
- **Minimal dependencies**: Where practical, functionality is implemented directly
  rather than importing external packages. Consolidating common functionality into
  a single, well-maintained module reduces supply chain complexity and audit surface.

## Packages

| Package         | Purpose                                                      |
| --------------- | ------------------------------------------------------------ |
| **assert**      | Test assertions with detailed diffs                          |
| **cli**         | CLI framework with commands, flags, config, and middleware   |
| **clipboard**   | Cross-platform system clipboard read/write                   |
| **color**       | ANSI color types, RGB/HSL conversion, gradients              |
| **crawler**     | Web crawler with pluggable fetchers, parsers, and caching    |
| **env**         | Config from environment variables, .env files, and JSON      |
| **fetch**       | HTTP page fetching with configurable options                 |
| **gif**         | Animated GIF creation with drawing primitives                |
| **git**         | Wrapper for common git read operations                       |
| **htmlparse**   | HTML parsing, metadata extraction, link discovery            |
| **htmltomd**    | HTML to Markdown conversion, ideal for LLM consumption       |
| **humanize**    | Human-readable formatting (bytes, durations, numbers)        |
| **retry**       | Retry with exponential backoff, jitter, and callbacks        |
| **sse**         | Server-Sent Events parser and reconnecting client            |
| **terminal**    | Low-level terminal control, input decoding, styling          |
| **termsession** | Terminal session recording/playback (asciinema v2 format)    |
| **termtest**    | Terminal output testing with ANSI parsing and snapshots      |
| **tui**         | Declarative TUI library with layout engine and components    |
| **unidiff**     | Unified diff parsing for display and analysis                |
| **web**         | URL manipulation, text normalization, media type detection   |

## Development Commands

```bash
# Run all tests
go test ./...

# Run tests for a specific package
go test ./tui/...

# Run example
go run ./examples/cli/basic

# Coverage
make cover-text
```
