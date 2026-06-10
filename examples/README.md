# Examples

Each directory under `examples/` is a standalone Go module entry point. Run any
of them with `go run`:

```bash
go run ./examples/cli/basic --help
go run ./examples/tui/table
```

Most TUI demos assume a real TTY for color and mouse support. Use `Ctrl+C` or the
built-in quit key shown on screen to exit.

## CLI

- `cli/basic`: Ground-up tour of commands, flags, args, command groups, aliases,
  hidden and deprecated commands.
- `cli/flags`: Advanced flags — struct-based flag definitions with generics, env
  fallbacks, enums, required flags, positional-arg and custom validation.
- `cli/global_flags`: Global flags shared by all commands, with middleware for
  verbose logging and config handling.

## Configuration

- `env/config`: Typed app configuration from env vars, `.env`, and JSON files with
  prefixes, nested structs, defaults, and required fields.

## GIF generation

- `gif`: CLI that converts asciinema `.cast` recordings to animated GIFs, with
  `info` and bouncing-ball `demo` subcommands.
- `gif/grapheme_eval`: Characterizes grapheme-cluster handling in the gif terminal
  screen vs. emulator, dumping cells and rendering sample emoji GIFs.
- `gif/shapes`: Basic animated GIF creation with drawing primitives (rotating
  squares, circles, pulsing ring).
- `gif/terminal`: Terminal-style GIF of a shell session typed out character by
  character on a virtual screen.

## Web / networking

- `browser`: TUI web browser that fetches pages, renders them as Markdown, and
  offers URL bar, link panel, and history navigation.
- `crawl`: Web crawler CLI with fetch/links/meta subcommands and optional
  interactive TUI crawl display.
- `htmltomd`: Converts HTML to Markdown from a URL, file, or stdin with
  configurable link/heading/code styles.
- `sitecheck`: Live-TUI website link checker: crawls a site and HEAD-checks every
  discovered URL, reporting broken links with color-coded stats.
- `sseview`: Real-time TUI viewer for Server-Sent Events streams: connects to an
  SSE endpoint and displays events with JSON pretty-printing and vim-style navigation.
- `urlx`: Clipboard URL watcher: polls the clipboard for URLs, fetches each page,
  converts to Markdown, and writes the result back to the clipboard.
- `webwatch`: Web page change monitor: periodically fetches a URL, converts to
  Markdown, and prints a colorized unified diff whenever the content changes.

## Terminal / session

- `termprobe`: Terminal cell-width probe: measures the host terminal's reported
  column width for a battery of Unicode grapheme clusters and flags disagreements
  with `runewidth.StringWidth`.
- `termrec`: Terminal session recorder and GIF exporter: record a shell session to
  asciinema format, convert to animated GIF, inspect metadata, or replay a
  recording.
- `termsession/playback`: Minimal asciinema playback demo using
  `tui.LoadRecording` and `PlaybackController`, playing a `.cast` file with
  original timing.
- `sessview`: Interactive TUI browser for asciinema `.cast` recordings: browse a
  directory, preview metadata, and replay sessions.

## Utilities

- `clipstack`: Clipboard history manager TUI that polls the system clipboard and
  lets you re-copy or delete past entries.
- `envview`: TUI browser for environment variables and `.env` files with search,
  value masking, and clipboard export.
- `gitgif`: Generates an animated GIF visualizing recent git commits with
  per-file addition/deletion bar charts.
- `gitscan`: tig-style TUI for browsing git history with commit details, diff
  view, and changed-files view.
- `paletteer`: TUI color palette designer with RGB sliders, gradient preview,
  presets, and hex/RGB/CSS/ANSI export.

## TUI — animation

- `tui/animation`: 60 FPS rainbow blocks drawn via `CanvasContext` and the
  render-frame counter.
- `tui/animation_showcase`: Gallery of border/text animation presets (rainbow,
  pulse, marquee, fire, sparkle, glitch, wave) navigable with arrow keys.
- `tui/text_animation`: All built-in text animations — Rainbow, Pulse, Sparkle,
  Typewriter, Glitch, Slide, and Wave.

## TUI — layout & display

- `tui/border`: Full-screen border that tracks terminal resize events.
- `tui/code`: Syntax-highlighted scrollable source-file viewer.
- `tui/composition`: Nested Stack/Group layouts with bordered panels and clickable
  counter buttons.
- `tui/counter`: Minimal declarative counter with keyboard shortcuts and clickable
  mouse controls.
- `tui/diff`: Unified diff rendered with `DiffView` (syntax highlighting, line
  numbers) in a scrollable bordered panel.
- `tui/hyperlink`: OSC 8 clickable hyperlinks in eight styles (inline groups,
  fallback URL display, link rows).
- `tui/markdown`: Full-screen Markdown viewer with syntax-highlighted code, tables,
  and keyboard scrolling.
- `tui/metrics`: Performance metrics display using manual Terminal/Runtime setup
  and `Terminal.GetMetrics()`.
- `tui/print`: Inline view printing with `tui.Print`, `LivePrinter`, and
  `tui.Live` — styled output, borders, and in-place progress updates without a
  full TUI.
- `tui/table`: Declarative Table view with keyboard navigation, uppercase headers,
  max column width, selection color inversion, and an OnSelect callback.
- `tui/text`: Text wrapping, truncation, and left/center/right alignment in a 2×2
  grid of flexed colored cells.

## TUI — input & forms

- `tui/bracketed_paste`: Multi-line text editor showing atomic paste detection via
  bracketed paste mode, with paste history.
- `tui/checkbox`: Multi-select `CheckboxList` with cursor navigation and live
  selection summary.
- `tui/file_picker`: Directory browser using `FilePicker` with type-to-filter,
  hidden-file toggle, and selection callbacks.
- `tui/input_display`: Minimal declarative single-input form: `tui.Input` with
  live greeting via `tui.IfElse`.
- `tui/input_forms`: Multi-field form with Tab/Shift+Tab navigation, field
  validation, and a submit button.
- `tui/input_styles`: `InputField` style gallery: bordered, horizontal-bar,
  prompt, and blinking-bar cursor variants.
- `tui/password`: Password entry using `tui.Input` with mask character in a
  declarative form.
- `tui/paste_placeholder`: Input field paste-placeholder mode that collapses
  multi-line pastes into a `[pasted N lines]` token, with a TextArea content
  viewer.
- `tui/prompt_choice`: Claude Code-style PromptChoice confirmation widget with
  numbered options, arrow navigation, and an inline free-text option.
- `tui/prompt_wizard`: Multi-step setup wizard built on PromptChoice with
  per-step options, Esc-to-go-back, progress bar, and a summary screen.
- `tui/textarea_features`: TextArea display options — line numbers, left-border
  style, and current-line highlighting.

## TUI — lists & navigation

- `tui/list`: `FilterableList` with type-to-filter, multi-select markers, and a
  live selection info panel.

## TUI — progress

- `tui/progress_enhanced`: Progress bar customization — empty-region patterns,
  percentage/fraction labels, and per-element colors.
- `tui/progress_spinners`: Multiple simultaneous spinners and progress bars driven
  by TickEvents, auto-quitting after all simulated tasks finish.

## TUI — mouse

- `tui/mouse`: Clickable buttons and a scroll area navigable by mouse wheel and
  arrow keys.
- `tui/mouse_grid`: 5×5 color grid where clicking a cell cycles through five
  colors via `ColorGrid` built-in click handling.

## TUI — inline / scrollback

- `tui/claude`: Claude Code-style chat UI with fixed bottom input, scrollable
  history, and command history recall.
- `tui/inline_chat`: InlineApp chat UI: async response commands, scrollback
  history, and live typing indicator.
- `tui/inline_counter`: Minimal InlineApp: live counter updated by +/- keys,
  printing to scrollback.
- `tui/inline_dive_like`: Stress-test of InlineApp live region: spinner,
  tool-call rows, todo list, dialog, and variable-height footer.
- `tui/inline_input`: `tui.Prompt` with history navigation, `@file` autocomplete,
  multi-line entry, and `LivePrinter` streaming response.
- `tui/inline_test`: InlineApp test harness: phased live views that grow/shrink,
  batch scrollback printing, and auto-progress commands.
- `tui/scrollback_demo`: Chat-style scrollback + live region pattern using raw
  input, `tui.Print` for history, and `LivePrinter` for the editable input area.
- `tui/scrollback_simple`: Minimal scrollback + live region — keypresses print
  views into history while two ticker-driven simulated tasks update the live region.

## TUI — async commands

- `tui/tui_command`: Async `tui.Cmd` pattern — non-blocking HTTP fetches of GitHub
  profiles delivering custom events back to `HandleEvent`.

## TUI — grapheme / Unicode

- `tui/grapheme_input`: Grapheme-cluster-aware text input with live
  byte/rune/grapheme/width counters (ZWJ emoji, flags, VS16).
- `tui/grapheme_table`: Table whose columns stay aligned across CJK/emoji/flag
  content via runewidth-backed measurement.
- `tui/reflow`: Animated container reflow stress-testing grapheme-cluster width
  handling (emoji ZWJ, flags, keycaps, Indic, CJK) with runewidth compat mode.

---

Feel free to copy these examples as starting points for your own applications.
