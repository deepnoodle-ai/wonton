# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/). Wonton is pre-1.0, so minor changes and API evolution are expected between `v0.0.x` releases. Pin your version.

## [Unreleased]

### Added

- **`runewidth` package.** Unicode 17.0.0, full UAX#29 grapheme cluster
  segmentation (including GB9c Indic conjunct breaks), and correct widths
  for ZWJ emoji, skin tones, flags, keycap sequences, VS16-forced emoji,
  and the two/three-em dashes. `StringWidth`, `Graphemes`, `Truncate`,
  `Fit`, `FitLeft`, `FitRight`, and `WidthIndex`. `SetTerminalCompatMode`
  opts into width tweaks that match what real terminals draw. See
  [docs/runewidth-improvements.md](docs/runewidth-improvements.md) for
  the comparison against `go-runewidth` and `uniseg`.
- **`pty` package.** In-tree pseudo-terminal allocation for Linux and
  macOS. `Open`, `Start`, `StartWithAttrs`, `Resize`, `GetSize`,
  `InheritSize`. `PTY.Close` is safe from multiple goroutines; the master
  pointer is stored atomically so `Close` racing with other operations is
  defined behavior.
- **Package READMEs** for `pty` and `runewidth` to match the rest of the
  packages in the module.
- **CLI group formatting controls.** `Group.FlatRouting`, a new `-h` flag
  override, and `PrintRaw` on the help output.
- **Examples**: `tui/grapheme_input`, `tui/grapheme_table`, `tui/reflow`
  (grapheme-aware reflow demo), `termprobe` (compares the host terminal's
  cursor math against `runewidth`), and `gif/grapheme_eval`.
- **Cross-platform CI matrix.** `test` runs on Linux, macOS, and Windows;
  `cross-compile` covers linux/386, linux/arm64, and windows/amd64.

### Changed

- **Dropped `github.com/mattn/go-runewidth` and `github.com/rivo/uniseg`**
  from `go.mod`. The in-tree `runewidth` package replaces both.
- **`tui/text_input`**: password mask is now sized by display cell width
  rather than rune count, so mask cells line up with the grapheme-based
  cursor math used elsewhere in the input.
- **CLI help output**: group formatting fixes and a new expand control.

### Fixed

- **`pty.PTY.Close` race.** Two goroutines calling `Close` no longer race
  on the master field. Only the first call closes the fd.
- **Wrap math for oversized grapheme clusters.** A cluster wider than the
  available line width no longer produces a blank leading line in
  `TextInput`.

## [0.0.29] - 2026-03-19

### Added

- `cli.Group.FlatRouting` for flat command routing under a group.
- `PrintRaw` helper on CLI help output.

### Fixed

- `-h` flag override behavior.

## [0.0.28] - 2026-03-19

### Changed

- CLI help: insertion-ordering preserved, dual usage rendering, improved
  examples layout.

## [0.0.27] - 2026-03-12

### Changed

- Removed the default 80-character max width from markdown rendering.
  Output now fills the available terminal width.

## [0.0.26] - 2026-02-06

### Added

- `llms.txt` with full package documentation for LLM consumers.

### Fixed

- UTF-8 handling in the `gif` emulator.
- Edge cases in `gif` for invalid terminal dimensions.

## [0.0.25] - 2026-01-26

### Changed

- `assert.Equal` now uses `reflect.DeepEqual` for structural comparison.

## [0.0.24] - 2026-01-06

### Fixed

- Inline-app resize bug in the TUI runtime.

### Added

- Expanded test coverage for input and inline rendering paths.

## [0.0.23] - 2026-01-05

### Fixed

- Assorted example fixes.

## [0.0.22] - 2026-01-05

### Changed

- Broad design pass on the `tui` package: consistency improvements across
  layout, inputs, and rendering.

## Earlier

Releases before `v0.0.22` are available as git tags. See `git log` for the
full history.
