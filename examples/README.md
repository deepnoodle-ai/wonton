# Examples

Each directory under `examples/` is a standalone Go module entry point. Run any
of them with `go run`:

```bash
go run ./examples/cli_basic --help
go run ./examples/table_demo
```

Most demos assume a real TTY for color and mouse support. Use `Ctrl+C` or the
built-in Quit instructions to exit.

## CLI highlights

- `cli_basic`: Ground-up tour of commands, groups, aliases, and help output.
- `cli_interactive`: Shows progressive interactivity with separate handlers for
  non-TTY vs TTY sessions, plus selection dialogs.
- `cli_streaming`: Demonstrates streaming output, JSON event mode, and progress
  spinners.
- `cli_wizard`: Multi-step configuration wizard that mixes prompts, validation,
  and summaries.
- `cli_todo`: Hybrid CLI/TUI task manager with keyboard navigation and data
  persistence simulation.
- `cli_flags` and `cli_global_flags`: Patterns for struct-based flag parsing and
  global configuration.

Run `go run ./examples/<name> --help` to inspect per-command usage.

## TUI runtime demos

- `table_demo` and `text_demo`: Workhorse primitives for grids, typography, and
  selection handling.
- `markdown_demo`: Renders markdown with syntax highlighting and adaptive layout.
- `runtime_http`: Fetches GitHub data asynchronously and streams updates into
  the UI.
- `animation_demo`: Demonstrates the fluent animation API (`.Rainbow()`, `.Pulse()`,
  `.Wave()`) for text views.
- `declarative_animation` and `runtime_animation`: Show `CanvasContext` for custom
  animations with access to the frame counter via `ctx.Frame()`.
- `progress_spinners` and `flicker_free`: Loading spinners, tick events, and
  rendering optimizations.
- `mouse_demo` and `mouse_grid`: Pointer interaction patterns.
- `file_picker_demo`, `input_forms`, `checkbox_demo`, and `password_demo`:
  Ready-made widgets (pickers, forms, toggle inputs).

## Terminal techniques

- `hyperlink_demo`: Emits OSC 8 hyperlinks using the `terminal` package.
- `runtime_animation` and `runtime_counter`: Show how to drive the TUI runtime
  without the CLI framework.
- `slog_demo`: Uses the colorized `slog` handler to inspect structured logs.

Feel free to copy these examples as starting points for your own applications.
