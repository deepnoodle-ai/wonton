# Examples

Each directory under `examples/` is a standalone Go module entry point. Run any
of them with `go run`:

```bash
go run ./examples/cli/basic --help
go run ./examples/tui/table
```

Most demos assume a real TTY for color and mouse support. Use `Ctrl+C` or the
built-in Quit instructions to exit.

## CLI highlights

- `cli/basic`: Ground-up tour of commands, groups, aliases, and help output.
- `cli/flags` and `cli/global_flags`: Patterns for struct-based flag parsing and
  global configuration.
- `env/config`: Loads typed configuration from environment variables, `.env`
  files, and JSON with the `env` package.

Run `go run ./examples/<name> --help` to inspect per-command usage.

## TUI runtime demos

- `tui/table` and `tui/text`: Workhorse primitives for grids, typography, and
  selection handling.
- `tui/markdown`: Renders markdown with syntax highlighting and adaptive layout.
- `tui/text_animation`: Demonstrates text animations (`tui.Rainbow`,
  `tui.Pulse`, `tui.Wave`, and more) applied with `.Animate()`.
- `tui/animation`: Shows `CanvasContext` for custom animations with access to
  the frame counter via `ctx.Frame()`.
- `tui/progress_spinners`: Loading spinners and tick events.
- `tui/mouse` and `tui/mouse_grid`: Pointer interaction patterns.
- `tui/file_picker`, `tui/input_forms`, `tui/checkbox`, and `tui/password`:
  Ready-made widgets (pickers, forms, toggle inputs).
- `tui/grapheme_input` and `tui/grapheme_table`: Exercise the in-tree
  `runewidth` package — ZWJ emoji, VS16 sequences, keycaps, and CJK —
  through a live input and an alignment-critical table layout.
- `tui/reflow`: Compares naive and `runewidth`-aware reflow so you can see
  where grapheme-aware width math matters.

## Workspaces

- `corral`: A terminal workspace in the style of herdr or a tmux GUI. The main
  area splits into columns and rows of live terminals, each a tmux session;
  drag the edges to resize, right-click anything for a context menu, and pick
  panes from the sidebar. Quit and run it again to pick up where you left off.
  Built on `tmux.Terminal`, `tui.Split`, `tui.Menu`, `tui.HitMap`, and
  `tui.Cells`. Requires tmux.

## Terminal techniques

- `tui/hyperlink`: Emits OSC 8 hyperlinks from a declarative view.
- `tui/counter`: Shows how to drive the TUI runtime without the CLI framework.
- `termprobe`: Writes grapheme clusters to the host terminal and compares the
  reported cursor column against `runewidth.StringWidth` to surface renderer
  disagreements.
- `gif/grapheme_eval`: Renders a corpus of grapheme clusters to GIF so you can
  eyeball how the `gif` package draws them.
- `termsession/playback`: Plays an asciinema recording with pause and speed
  controls.

## Apps

Small, complete tools built from several packages:

- `browser`: Markdown-based terminal web browser with link navigation.
- `clipstack`: Clipboard history manager.
- `crawl`: Web crawler with `fetch`, `links`, and `meta` subcommands.
- `envview`: Browser for environment variables and `.env` files.
- `gif`, `gif/shapes`, and `gif/terminal`: Animated GIF creation, including
  converting `.cast` recordings.
- `gitgif`: Renders recent git commits to an animated GIF.
- `gitscan`: Git history and diff browser.
- `htmltomd`: Converts HTML from a URL, file, or stdin to Markdown.
- `paletteer`: Terminal color palette designer.
- `sessview` and `termrec`: Record, browse, replay, and export terminal
  sessions.
- `sitecheck`: Crawls a site and checks its links.
- `sseview`: Server-Sent Events stream viewer.
- `urlx`: Watches the clipboard and converts copied URLs to Markdown.
- `webwatch`: Monitors a web page and prints diffs when it changes.

Feel free to copy these examples as starting points for your own applications.
