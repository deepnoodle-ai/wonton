# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/). Wonton is pre-1.0; pin your version.

## [Unreleased]

### Fixed

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

### Added

- `unidiff.File` gained `IsNew`, `IsDelete`, and `IsRename` fields, detected
  from git metadata lines (`new file mode`, `deleted file mode`,
  `rename from`/`rename to`) and `/dev/null` paths. Pure renames without
  hunks now parse with both paths populated. `GIT binary patch` sections are
  detected as binary.
- `sse.HTTPError` gained a `Body` field with up to 8KB of the error response
  body, which is included in `Error()` when present.

### Changed

- `web.NormalizeURL` lowercases the host for stable comparisons and
  deduplication.
- `sse.Client` validates the response Content-Type with `mime.ParseMediaType`
  instead of a prefix check.
- `crawler` honors robots.txt `Crawl-delay` (per worker, when larger than
  `RequestDelay`), skips robots.txt checks for cache hits, `Workers`
  defaults to 1, and parse failures are now counted as failed (not
  succeeded) in crawl stats.
- `crawler.Crawler.Stop` is now safe to call concurrently with `Crawl`.
- `clipboard` errors now include stderr output from the underlying
  clipboard utility.
- `assert` colored diff output now respects `NO_COLOR`/`FORCE_COLOR`
  (via `color.ShouldColorize`) instead of only checking for a terminal.

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
