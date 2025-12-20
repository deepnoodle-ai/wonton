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

| Package         | Description                                    |
| --------------- | ---------------------------------------------- |
| **assert**      | Test assertions with diffs                     |
| **cli**         | Commands, flags, config, middleware            |
| **clipboard**   | System clipboard read/write                    |
| **color**       | ANSI colors, RGB/HSL, gradients                |
| **crawler**     | Web crawler with caching                       |
| **env**         | Config from env vars, .env, JSON               |
| **fetch**       | HTTP page fetching                             |
| **gif**         | Animated GIF creation                          |
| **git**         | Git read operations                            |
| **htmlparse**   | HTML parsing, metadata, links                  |
| **htmltomd**    | HTML to Markdown conversion                    |
| **humanize**    | Human-readable formatting                      |
| **retry**       | Retry with backoff and jitter                  |
| **schema**      | JSON Schema types and generation for LLM tools |
| **sse**         | Server-Sent Events client                      |
| **terminal**    | Terminal control and input decoding            |
| **termsession** | Session recording (asciinema format)           |
| **termtest**    | Terminal output testing                        |
| **tui**         | Declarative TUI with layout engine             |
| **unidiff**     | Unified diff parsing                           |
| **web**         | URL utilities, binary fetch, search            |

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
