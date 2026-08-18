# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/). Wonton is pre-1.0; pin your version.

## [Unreleased]

### Changed

- **Breaking**: `fetch.DefaultHTTPClient` and `fetch.DefaultDownloadClient` are
  now built by the new `httpguard` package. They refuse to connect to
  loopback, private, link-local, and other non-public addresses, and they
  ignore `HTTP_PROXY` / `HTTPS_PROXY` (a proxy would choose the destination
  itself, where the guard cannot see it). Redirects are still followed up to
  the standard limit of 10, with every hop address-validated. `crawler`
  inherits this through whatever fetcher it is given.

  Fetching or downloading from `localhost` or an intranet host now requires an
  explicit client — `HTTPFetcherOptions.Client`, `DownloadOptions.Client`, or
  reassigning the package variable:

  ```go
  fetch.DefaultHTTPClient = &http.Client{Timeout: fetch.DefaultTimeout}
  ```

### Added

- New `strs` package: `FirstNonEmpty` / `FirstNonBlank` / `FirstNonBlankTrim`
  for layered fallbacks, and `Dedupe` / `DedupeNonBlank` for order-preserving
  deduplication.
- New `ptr` package: generic pointer helpers (`To`, `Deref`, `Or`,
  `IfNotZero`, `DerefSlice`, `DerefMap`, `MapIfNotEmpty`, `SliceIfNotEmpty`)
  for optional and generated-client fields.
- New `httpguard` package: an `http.Client` for URLs your program did not
  choose. It validates every address a hostname resolves to, dials a validated
  address rather than the hostname (closing the DNS-rebinding hole), refuses
  redirects unless `WithMaxRedirects` enables a bounded number of guarded HTTPS
  ones, and ignores ambient proxy environment variables. `WithHTTPRedirects`
  additionally allows plain-HTTP hops (still address-validated) for callers
  fetching public pages, and `ValidatePublicIP` is exported for callers that
  want the same check elsewhere.
- New `thumbnail` package: preview images for files — downscaled PNG/JPEG/GIF/
  WebP rasters, and synthetic cards for text, code, and unknown types. Bad
  input never errors; it degrades to a typed card. Pure Go, no new module
  dependencies (stdlib plus the existing `golang.org/x/image`).

## [0.0.37] - 2026-06-29

### Fixed

- `cli`: help and shell completion output now render commands and groups in
  stable registration order instead of depending on Go map iteration.
- `cli`: group help now sizes the command-name column to the longest visible
  command so long command names keep descriptions aligned.

## [0.0.36] - 2026-06-10

`tui` input enhancements for REPL-style apps (#42).

### Added

- `tui`: `OnKey(fn)` hook on `Input`, `InputField`, and `TextArea` — the app
  sees every key before built-in handling; return `true` to consume.
- `tui`: `History(items)` on `Input`/`InputField` — shell-style ↑/↓ recall
  with draft preservation; multiline recalls only at the first/last line.
- `tui`: `OnComplete(fn)` on `Input`/`InputField` — Tab completion with
  in-buffer cycling (Tab/Shift+Tab to cycle, Esc to restore).
- New `examples/tui/input_repl` REPL example.

### Changed

- `tui`: multiline ↑/↓ bubble to the app at the first/last visual line, and
  Tab is offered to the focused element before focus-cycling.

## [0.0.35] - 2026-06-10

A large correctness and API-hardening release spanning nearly every package:
a multi-package bug-hunting pass (#28–#32) plus a `tui` runtime/refactor
overhaul (#33–#37), the `color` rework (#38), and the `web`/`fetch`/`crawler`
refactor (#39) and `cli` review (#40). It includes **breaking** API changes in
`color`, `web`/`fetch`, `cli`, and `tui` — see **Changed** and **Removed**.

### Added

- `cli` gained `Context.Duration` (a typed getter for `Duration` flags), an
  `Int64Flag` type and `Int64()` builder, and `CommandError.Wrap(err)` /
  `Unwrap()` for standard error wrapping.
- `color`: `Hex`, `MustHex`, and `RGB.Hex` for parsing/formatting `#rrggbb` /
  `#rgb`; `RGB.HSL()` (the inverse of `HSLToRGB`); `RGB.Lerp`; `Color.ApplyBold`
  / `ForegroundSeqBold` (parity with `ApplyDim`); and `ApplyGradient(text,
  stops...)`, which colors a string per grapheme cluster (emoji-safe via
  `runewidth`).
- `fetch.Download` downloads a URL to a file (replacing `web`'s binary
  fetcher), with a unified `fetch.Error{StatusCode, URL, Err}` and
  `fetch.IsRetryable()` for use with `retry.WithRetryIf`.
- `web.NormalizeURL` gained `KeepQuery()` / `KeepHTTP()` options, and the media
  helpers expose `BinaryExtensions` (an `ExtensionSet`).
- `tui`: shared list navigation across all list-shaped views (`SelectList`,
  `FilterableList`, `CheckboxList`, `RadioList`, `Table`, `Tree`) — arrows,
  PageUp/PageDown, Home/End, and vi-style `j`/`k`/`g`/`G`. Tagged timer events
  via `After(d, tag)` → `TimerEvent{Tag}`, handled on the event loop.
  Opt-in backslash+Enter → Shift+Enter synthesis via `WithBackslashEnter(true)`
  / `InlineAppConfig.BackslashEnter`.
- `unidiff.File` gained `IsNew`, `IsDelete`, and `IsRename` fields, detected
  from git metadata lines (`new file mode`, `deleted file mode`,
  `rename from`/`rename to`) and `/dev/null` paths. Pure renames without
  hunks now parse with both paths populated. `GIT binary patch` sections are
  detected as binary.
- `sse.HTTPError` gained a `Body` field with up to 8KB of the error response
  body, which is included in `Error()` when present.
- `htmltomd` honors the ordered-list `start` attribute and preserves image
  `title` attributes (`[alt](src "title")`).
- `terminal` now decodes modified Home/End, modifiers on tilde-keys (Ctrl+Delete,
  Ctrl+F5), xterm modifyOtherKeys, and Kitty sub-parameters / high-bitfield
  modifiers.
- `termtest.Screen.Reset()` performs a full RIS reset (backs `ESC c` and
  `Recorder.Reset`).
- `termsession` records and emits asciicast v2 `"r"` resize events on
  `UpdateSize` (previously dropped).

### Changed

- **`color` (breaking):** the high-level helpers (`Apply`, `ApplyBg`,
  `ApplyDim`, `ApplyBold`, `Sprint`, `Sprintf`, `ApplyGradient`) now respect
  `color.Enabled` — auto-initialized at startup from `FORCE_COLOR` /
  `CLICOLOR_FORCE` (force on), `NO_COLOR` / `CLICOLOR=0` (force off), then stdout
  TTY detection — so output is plain text when piped or redirected. Low-level
  `*Seq` functions stay unconditional for `terminal`/`tui` frame rendering.
  `RGB.Apply(text, bool)` is split into `Apply(text)` / `ApplyBg(text)` to
  mirror `Color`; `NoColor` is renamed to `Default`; and all gradient functions
  uniformly return an empty slice for `steps <= 0`.
- **`web` (breaking):** now a pure, I/O-free utilities package. `ResolveLink` is
  redesigned per RFC 3986 and returns `*url.URL`; `NormalizeURL` fixes bare
  `host:port` inputs and lowercases the host; media helpers are renamed
  `IsMediaURL` / `IsBinaryURL`; and `SearchOutput.Items` is now `[]SearchItem`
  (was `[]*SearchItem`).
- **`cli` (breaking):** flag-builder constructors (`String`, `Bool`, `Int`,
  `Float64`, `Float`, `Duration`, `Strings`, `Ints`) take the `short` name as a
  variadic — callers no longer pass `""` when there is no short form.
  `ParseFlags[T]` now returns `*Command` (for chaining) instead of a `*T` zero
  value; `middleware.Confirm` is renamed to `ConfirmBefore` (avoiding the
  collision with `ctx.Confirm`). Env-var flag values are converted to the flag's
  type at parse time, so typed getters like `ctx.Ints` work for env-provided
  values and invalid values error instead of silently becoming zero. The `After`
  middleware returns both the handler and after-fn errors via `errors.Join`, and
  `--color`/`--no-color`/`SetColorEnabled` mirror into `color.Enabled`.
- **`env` (breaking):** acronym runs in field names now map to a single segment
  (`APIKey` → `API_KEY`, `DB` → `DB`, `HTTPServer` → `HTTP_SERVER`), affecting
  `WithUseFieldName()` and default nested-struct prefixes. `.env` parsing splits
  on whichever of `=` / `:` appears first, strips `export` followed by any
  whitespace, skips lines with unclosed quotes, and passes the raw value bytes
  to `[]byte` fields.
- **`tui` (breaking):** keys consumed by a focused widget no longer reach the
  app's `HandleEvent` — **Ctrl+C is always delivered** so apps keep a reliable
  quit shortcut. The five package-global view registries are now per-`Runtime` /
  per-`InlineApp`, so two apps in one process no longer share input/click state.
  `After(d, func())` is replaced by `After(d, tag)`, and the `Tick(d)` command is
  removed (use `After`; frame ticks are unchanged).
- `crawler` de-forks its private `normalizeURL`/`resolveLink` onto
  `web.NormalizeURL`/`web.ResolveLink`; honors robots.txt `Crawl-delay` (per
  worker, when larger than `RequestDelay`); skips robots.txt checks on cache
  hits; defaults `Workers` to 1; counts parse failures as failed (not
  succeeded); and `Stop` is now safe to call concurrently with `Crawl`.
- `sse.Client` validates the response Content-Type with `mime.ParseMediaType`
  instead of a prefix check.
- `clipboard` errors now include stderr output from the underlying clipboard
  utility.
- `assert` colored diff output now respects `NO_COLOR`/`FORCE_COLOR`
  (via `color.ShouldColorize`) instead of only checking for a terminal.
- `termtest` (behavior): the cursor now reports `width-1` after filling a line
  (deferred wrap, matching xterm); `CSI 2J`/`3J` no longer home the cursor; DCH
  never erases left of the cursor; SGR colon sub-parameters are scoped to their
  parameter per ECMA-48/T.416; and `Diff` is rewritten with LCS line alignment
  (real context lines, no cascade of false changes after an inserted line).

### Removed

- `tui`: the imperative `Widget`/`ComposableWidget` and
  `Layout`/`Header`/`Footer`/`StatusItem` systems, plus the standalone
  `FlexLayout`, size-constraint, and `List` widgets (−3,125 lines; nothing in
  the live View system or examples used them). The internal `TextInput` engine
  is unexported to `textInput`. The `Tick(d)` command (use `After`).
- `color`: `Colorize`, `ColorizeIf`, and `ColorizeRGB` (use `Apply`); the
  deprecated `RGB.Foreground`/`Background`; `IsTerminal` (use `tty.IsTerminal`
  or `ShouldColorize`); and the `Escape` constant is unexported.
- `cli`: `Command.Aliases`/`Group.Aliases` (the variadic `Alias` already covers
  them), plus dead internals (`App.findCommand`, `Command.looksLikeFlag`).
- `web`: `SortURLs`, `EndsWithPunctuation`, `BinaryFetcher`, and `FetchError`
  (use `fetch.Download` / `fetch.Error`).

### Fixed

- `retry.ExponentialBackoff` applied `MaxBackoff` only after casting to
  `time.Duration`, so around attempt 40 the float product overflowed negative
  and, with infinite retries, degraded into a hot loop hammering the failing
  service; the cap is now applied in float space. `LinearBackoff` gains an
  equivalent overflow guard, and nil `Config.Timer`/`DelayFunc` now use the
  documented fallbacks instead of panicking.
- `schema.Generate` now matches `encoding/json`: `[]byte` generates a base64
  string (not an int array), `json.RawMessage` an unconstrained schema, tagged
  embeds nest instead of flattening, and embedded non-structs are named after
  the type.
- `htmlparse.Transform` escapes text nodes, closing an injection vector where
  source like `&lt;script&gt;` round-tripped into a live `<script>` tag.
  `Metadata` also reads `property="twitter:*"` cards, legacy `http-equiv`
  charset declarations, and space-separated `rel` token lists.
- `htmltomd`: inline code spans and code fences now pick backtick runs that
  don't collide with the content, table cells escape literal `|`, and language
  detection handles multi-class / `lang-` prefixes.
- `cli`: typed flag validators actually run (`Validate` previously always
  returned nil); `parseInt` rejects `12abc` (was accepted as `12`); and help no
  longer prints `(default: [])` / `(default: 0s)` / `(default: 0)` for
  zero-valued slice/duration/float defaults.
- `env` sentinel parse errors now carry real messages instead of printing
  "parse error: parse error".
- `tui`: a panic in `View`/`HandleEvent`/a `Cmd` now restores the terminal and
  re-panics with the original stack (previously it left the terminal in raw mode
  on the alternate screen and ate the trace). `FlexLayout` wrapping (`FlexWrapOn`
  did nothing) and grow/shrink remainder loss are fixed; the
  `PlaybackController` recursive `RLock` deadlock, `PasswordInput` non-ASCII
  corruption, and `listView.ItemHeight(0)` panic are fixed. Tab is no longer
  swallowed when there are no focusables; list views scroll-follow on
  End/`G`/PageDown so the cursor stays visible; vi keys ignore Alt/Ctrl chords;
  and PageUp/PageDown uses the real viewport height.
- `terminal`: the CSI input decoder now reads full multi-parameter sequences, so
  cursor-position reports, modifyOtherKeys, and Kitty modifiers no longer leak
  phantom keystrokes into the app; SGR/X10 mouse release events carry the
  released button; `Style`/`Cell` compare RGB by value (truecolor cells no longer
  defeat double-buffering); `EndFrame` no longer deadlocks on an invalid frame;
  and the resize-watcher data race is fixed.
- `termsession`: `LoadCast` no longer infinite-loops at 100% CPU on a truncated
  or corrupt cast file; `Wait` returns an error instead of deadlocking when
  `Start` failed or was never called; pause gaps are removed from the timeline;
  and `RecordInput` honors `RedactSecrets`.
- `termtest`: C0 controls (backspace, etc.) are no longer rendered as glyphs;
  escape sequences and UTF-8 runes split across `Write` calls are buffered; and
  erasing a straddled wide glyph clears both halves.
- `crawler` no longer rewrites `mailto:` (and other non-http) links into
  crawlable `https://` URLs.
- `unidiff.Parse` now uses hunk line counts from `@@` headers, so changed
  lines that resemble file headers (e.g., a removed line starting with `--`,
  which appears as `--- ...` in the diff) are no longer misparsed as file
  headers.
- `unidiff.Parse` now parses plain `diff -u` output (no `diff --git` lines),
  which previously produced an empty result, and strips tab-separated
  timestamps from `---`/`+++` headers.
- `unidiff.Parse` accepts lines longer than 64KB (e.g., minified code).
- `humanize.Duration` no longer returns an empty string for sub-millisecond
  durations (now formats µs/ns) and no longer renders "1000ms" for values
  that round up to a second.
- `humanize.DurationShort` formats sub-microsecond durations as nanoseconds
  instead of "0µs".
- `humanize.Bytes`/`BytesSI` no longer display values like "1024.0 KiB" just
  below a unit boundary; they roll over to the next unit.
- `humanize.Number`/`NumberWithSeparator` no longer return a double negative
  sign for `math.MinInt64`.
- `humanize.Truncate`/`TruncateWithSuffix` no longer panic for negative
  maxLen; non-positive maxLen returns an empty string.
- `tty.IsTerminal` returns false for a nil file instead of panicking.
- `sse.Reader` strips a leading UTF-8 BOM per the SSE specification, and the
  `retry` field now requires ASCII digits only (signed values are ignored).
- `color.Gradient` no longer produces wrong intermediate colors when a channel
  decreases from start to end (unsigned underflow). Interpolation now rounds
  instead of truncating.
- `color.HSLToRGB` wraps hue values of any magnitude (e.g. 730° ≡ 10°,
  negative hues) and clamps saturation/lightness to [0, 1].
- `color.MultiGradient` no longer panics for `steps <= 1`.
- `assert.Equal` no longer fails with an empty diff when `reflect.DeepEqual`
  and go-cmp disagree (e.g. `time.Time` values that represent the same
  instant now compare equal).
- `assert.NotEqual` no longer panics on structs with unexported fields.
- `assert.InDelta` now fails on NaN arguments instead of silently passing.
- `assert.Regexp` reports a test failure for invalid patterns instead of
  panicking; `assert.Len` does the same for types without a length.
- `crawler.Crawler` is now reusable: a second `Crawl` call no longer panics
  on a closed queue, and `KnownURLs` (previously ignored) are now skipped.
- `crawler` fixed a race where a crawl could terminate before all queued
  URLs were processed, and a race where the same URL could be enqueued twice.
- `crawler.Options.MaxURLs` is now enforced strictly against the number of
  URLs admitted to the queue (previously it could over-admit while pages
  were in flight).
- `crawler` robots.txt handling: the most specific (longest) matching rule
  now wins per the standard (previously any Allow overrode every Disallow),
  and wildcard rules are anchored at the start of the path.
- `crawler` pages served from cache now have their links re-extracted, so
  link following keeps working on cache hits.
- `clipboard.Read` no longer trims a trailing newline from clipboard
  contents on macOS and Linux/X11, so `Write` → `Read` round-trips exactly.
- `clipboard` on Windows uses `Get-Clipboard -Raw`, preserving line endings
  in multiline content.

## [0.0.34] - 2026-05-12

### Fixed

- `schema.Generate` detects recursive and mutually recursive struct types
  instead of overflowing the goroutine stack at schema-construction time;
  a cyclic type now yields a permissive placeholder schema where it recurses.

## [0.0.33] - 2026-04-28

### Fixed

- `tui.Table` rendering no longer hangs when proportional column shrinking
  rounds all shrink shares to zero at constrained widths.
- CLI help now wraps long usage, description, flag, argument, example, and group
  text instead of clipping, including clean metadata-only flag rendering.

### Removed

- `pty` package. `termsession` depends on `github.com/creack/pty` directly.

## [0.0.32] - 2026-04-23

### Changed

- `cli` positional args are now strict. Extras beyond declared slots are
  rejected at parse time with `unexpected argument: <name>`, matching the
  default behavior of Cobra, urfave, clap, and click. Commands that want to
  accept arbitrary trailing positionals must opt in with the variadic DSL.

### Added

- `cli.Command.Args` DSL grew two variadic forms: `"name..."` (one or
  more) and `"name?..."` / `"name...?"` (zero or more). The last slot
  may be variadic. Invalid layouts — variadic not last, required slot
  following an optional one, duplicate names — panic at registration.

### Removed

- `cli.Command.ArgsRange`, `ExactArgs`, and `NoArgs`. All three are now
  derivable from the `Args` DSL:
  - `NoArgs()` → don't call `Args()`.
  - `ExactArgs(n)` → `Args("a", "b", ...)` with `n` required names.
  - `ArgsRange(min, max)` → required names up to `min`, optional names up
    to `max`, or a trailing variadic for unbounded.

## [0.0.31] - 2026-04-18

- Added `cli.Group.Alias` for alternate group names (singular/plural variants).

## [0.0.30] - 2026-04-15

### Added

- `runewidth` package: Unicode 17.0.0, UAX#29 grapheme segmentation, correct widths for ZWJ emoji, keycaps, VS16, and em dashes. See [docs/runewidth-improvements.md](docs/runewidth-improvements.md).
- `pty` package: in-tree pseudo-terminal allocation for Linux and macOS.
- READMEs for `pty` and `runewidth`, plus a top-level `CHANGELOG.md`.
- Examples: `tui/grapheme_input`, `tui/grapheme_table`, `tui/reflow`, `termprobe`, `gif/grapheme_eval`.
- CI test matrix across Linux, macOS, and Windows plus cross-compile coverage.

### Changed

- Dropped `go-runewidth` and `uniseg` dependencies. The in-tree `runewidth` replaces both.
- `tui/text_input`: password mask sized by display cell width, not rune count.

### Fixed

- `pty.PTY.Close` race between concurrent callers.
- Oversized grapheme cluster produced a blank leading line in `TextInput` wrap math.

## [0.0.29] - 2026-03-19

- Added `cli.Group.FlatRouting` and `PrintRaw`. Fixed `-h` override.

## [0.0.28] - 2026-03-19

- CLI help: insertion ordering, dual usage, better examples layout.

## [0.0.27] - 2026-03-12

- Removed the default 80-column cap on markdown rendering.

## [0.0.26] - 2026-02-06

- Added `llms.txt`. Fixed UTF-8 handling and invalid-dimension edge cases in `gif`.

## [0.0.25] - 2026-01-26

- `assert.Equal` uses `reflect.DeepEqual`.

## [0.0.24] - 2026-01-06

- Fixed inline-app resize bug. Expanded test coverage.

## [0.0.23] - 2026-01-05

- Example fixes.

## [0.0.22] - 2026-01-05

- Design pass across the `tui` package.

## Earlier

See git tags for history before `v0.0.22`.
