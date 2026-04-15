# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/). Wonton is pre-1.0; pin your version.

## [Unreleased]

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
