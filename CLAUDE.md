# CLAUDE.md

This file provides guidance to Claude Code when working with this codebase.

## Project Overview

Wonton is a collection of Go packages for building command-line applications. It
provides everything needed to create professional CLIs: from simple argument
parsing to rich terminal UIs with animations.

### Packages

| Package       | Import Path                                 | Purpose                                                    |
| ------------- | ------------------------------------------- | ---------------------------------------------------------- |
| **assert**    | `github.com/deepnoodle-ai/wonton/assert`    | Test assertions (fail immediately; use NonFatal for soft)  |
| **cli**       | `github.com/deepnoodle-ai/wonton/cli`       | CLI framework with commands, flags, config, and middleware |
| **clipboard** | `github.com/deepnoodle-ai/wonton/clipboard` | Cross-platform system clipboard access                     |
| **color**     | `github.com/deepnoodle-ai/wonton/color`     | ANSI color types, RGB, HSL, and gradient utilities         |
| **env**       | `github.com/deepnoodle-ai/wonton/env`       | Config loading from env vars, .env files, and JSON         |
| **htmltomd**  | `github.com/deepnoodle-ai/wonton/htmltomd`  | HTML to Markdown conversion for LLMs/AI agents             |
| **humanize**  | `github.com/deepnoodle-ai/wonton/humanize`  | Human-readable formatting for bytes, durations, numbers    |
| **retry**     | `github.com/deepnoodle-ai/wonton/retry`     | Retry logic with exponential backoff and jitter            |
| **slog**      | `github.com/deepnoodle-ai/wonton/slog`      | Colorized slog.Handler for terminal output                 |
| **sse**       | `github.com/deepnoodle-ai/wonton/sse`       | Server-Sent Events (SSE) stream parser and client          |
| **terminal**  | `github.com/deepnoodle-ai/wonton/terminal`  | Low-level terminal control, input decoding, styles         |
| **tui**       | `github.com/deepnoodle-ai/wonton/tui`       | Full TUI library with declarative views and runtime        |
| **tuisnap**   | `github.com/deepnoodle-ai/wonton/tuisnap`   | Snapshot testing utilities for TUI applications            |
| **unidiff**   | `github.com/deepnoodle-ai/wonton/unidiff`   | Unified diff parsing for display and analysis              |
