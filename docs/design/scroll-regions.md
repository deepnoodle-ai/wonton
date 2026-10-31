# Scroll regions in termtest, and InlineApp.Print without a redraw

**Status:** Draft\
**Author:** Curtis Myzie (with Claude)\
**Date:** 2026-10-31 (part 2 revised 2026-11-01)\
**Workflow:** Spec, then build in parallel. The termtest part has no open design question and is built first; the tui part waits for review of this spec.\
**PRD:** [docs/prds/scroll-regions.md](../prds/scroll-regions.md)

## Context

`InlineApp.Print` clears the live region, prints, and redraws it: 855 bytes
for a 46-byte line under a 5-line live region. Print can instead insert the
rows above the live region with cursor moves relative to it (line feeds,
cursor up, insert line), so the live region is not redrawn. termtest cannot
model scroll regions, BCE, or scrollback today, so neither the insert nor
other inline output can be tested against what a real terminal shows. The
PRD has the user problem, the insert's steps, and the decisions, including
the rejected scroll-region design. This spec covers how to build it.

## Goals

- termtest matches ghostty on new rows of `TestGhosttyProbes`
  (`termtest/ghostty_probes_test.go`) for BCE, DECSTBM, IND/RI/NEL at
  margins, SU/SD, IL/DL in a region, origin mode, C0 in CSI, and scrollback
  text. The expected values come from ghostty through the vtdiff harness,
  which links libghostty with cgo and stays outside the repo
  (`vtdiff dump <families>` prints the rows).
- Bytes per Print in the PRD's measurement scenario: 95 (from 855).
- Every printed line reaches scrollback once, in order, with a stable window
  and while it resizes during output.

## Non-goals

- Scroll regions in the full-screen renderer, `PrintRaw`, standalone
  `LivePrinter`, DECSLRM. See the PRD.
- Alternate screen, raw mode, and resize in termtest (other tracks).

## Proposal

### Part 1: termtest

**State.** Four new fields on `Screen`, added in one block in `screen.go`:

```go
top, bottom int      // scroll region, 0-based inclusive; 0 and height-1 when unset
originMode  bool     // DECOM, CSI ?6 h/l
savedOrigin bool     // saved by DECSC with savedX/savedY/savedStyle
scrollback  []string // rows that left the top of a top-anchored region, oldest first
```

`NewScreen` and `Reset` set `bottom = height-1`. `Reset` (RIS) clears the
region, origin mode, and scrollback, as ghostty does (the RIS rows of
`TestGhosttyProbes`).

**BCE.** One helper, `s.eraseCell()`, returns a blank cell whose style has
only `Background: s.style.Background`. Every fill site uses it: ED/EL
(the `Clear*` methods), ECH, DCH and ICH (`rewriteRow` fills with it), IL,
DL, and each new row from a scroll. `rowGlyphs` keeps treating only
`blankCell()` as empty, so an erased cell with a background shifts with DCH
and ICH like any other cell; the cells that DCH and ICH open take the
current background. Direct calls to the public `Clear*`
methods also use the current background; on a fresh screen that is the
default, so existing callers see no change.

**Region motion.** New file `termtest/region.go`:

- `setRegion(top, bottom int)` from `CSI t;b r`: 1-based input, 0 or missing
  means the edge, ignore if `top >= bottom` or out of range. Then cursor
  home (region top in origin mode), `wrapPending = false`.
- `index()`: LF/IND/NEL/autowrap. At `bottom` scroll the region up by one;
  below `bottom` move down unless on the last screen row; otherwise move
  down. `lineFeed()` in `screen.go` becomes a one-line call to `index()`.
- `reverseIndex()`: RI. At `top` scroll the region down; else move up unless
  on row 0.
- `scrollRegionUp(n)` / `scrollRegionDown(n)`: shift rows `top..bottom`, fill
  new rows with erase cells. `scrollRegionUp` appends each departing row's
  `Row` text to `scrollback` when `top == 0`. Called by `index` (n=1) and SU.
  SD and RI never add scrollback.
- `insertLines(n)` / `deleteLines(n)`: no-op when the cursor is outside
  `top..bottom`; act on `cursorY..bottom`; cursor to column 0 (xterm). These
  replace the bodies in `ansi.go`; DL never adds scrollback.
- Cursor moves: CUU stops at `top` if the cursor starts at or below `top`;
  CUD stops at `bottom` if it starts at or above `bottom`. CUP/VPA add `top`
  and clamp to the region in origin mode.

`scrollUp()` and `scrollDown()` in `screen.go` and `ansi.go` are replaced
by calls to the region functions, with the full screen as the region when
none is set. So a plain screen behaves as today, except BCE and scrollback.

**Scrollback.** New file `termtest/scrollback.go` with
`func (s *Screen) Scrollback() []string` (a copy) and the append helper.
`CSI 3 J` empties scrollback and no longer clears the screen, as in xterm
and ghostty. `CSI 2 J` is unchanged. No cap: test screens are small, and
the fuzz run checks memory.

**Origin mode and private modes.** New file `termtest/modes.go`:

```go
// setPrivateMode applies one DEC private mode parameter (CSI ? n h/l).
func (s *Screen) setPrivateMode(mode int, on bool) {
	switch mode {
	case 6: // DECOM
		...
	}
}
```

`parseCSI` gets one change in its private-parameter branch: for `?` with
`h` or `l`, call `setPrivateMode` for each parameter. The alt-screen track
needs the same loop for `1049`, `1047`, and `47`. Whichever track merges
second adds its `case` lines to this switch, so the conflict is a few lines
in one function. I will tell the alt-screen lead before part 1 is pushed.

**C0 inside CSI.** In `parseCSI`, the scan for the final byte collects
C0 bytes (0x00-0x1F except ESC, CAN, SUB) instead of passing them over.
After the final byte is found, the collected controls run in order through
the same path as plain text (`writeCluster` for CR, LF, VT, FF, BS, HT; BEL
and others are ignored), and then the CSI runs with the C0 bytes removed
from its parameters. When CAN, SUB, or ESC cancels the sequence, the
collected controls run first. They run only once the sequence ends, so a
sequence split across `Write` calls never runs a control twice. The order
is correct because every collected control came before the final byte.

This differs from xterm and ghostty, which run each control on arrival. The
screen is the same once the sequence ends. If the stream stops inside a
sequence, its controls stay unrun until the next write completes or cancels
it. The deferred rule is deliberate: `Write` buffers an incomplete sequence
and parses it again from its start, so on-arrival execution needs a
byte-by-byte parser state, which is out of scope (PRD R-5).

**Files touched.**

| File | Change |
|------|--------|
| `termtest/screen.go` | field block; `lineFeed` body; `Clear*` fills; `NewScreen`/`Reset` region init |
| `termtest/ansi.go` | `ESC D/M/E`, `CSI r/S/T/A/B/H/d/L/M`, `CSI 3J`, private `h/l` call, C0 collection; remove old `scrollDown`/`insertLines`/`deleteLines` |
| `termtest/region.go` | new: region motion |
| `termtest/bce.go` | new: `eraseCell`, `isEraseCell` |
| `termtest/scrollback.go` | new: `Scrollback` |
| `termtest/modes.go` | new: `setPrivateMode` |
| `termtest/csicontrols.go` | new: run C0 controls found inside a CSI |
| `termtest/region_test.go` and others | new tests, one per probe family |

The raw-mode track changes how a bare LF moves the column; that change sits
in `writeCluster` before `index()` is called, so the two compose. The resize
track must clamp `top`/`bottom` and reset the region on resize; noted for
whichever track rebases second.

### Part 2: tui

**Print.** Render the view to a buffer first (`Fprint` with the runner's
width, raw mode, and color). Then, in `tui/inline_insert.go`:

```go
// insertChunk returns how many rows one insert may add, or 0 to redraw.
func insertChunk(screenHeight, liveHeight int) int // (H-L)/2 when H-L >= 2

// insertAboveLive builds one synchronized-output bracket with, per chunk
// of n rows: ESC[0m, n x LF, CR, CSI n+L-1 A, CSI n L, rows joined by
// CRLF, CSI L B.
func insertAboveLive(rows []string, liveHeight, chunk int) string
```

`printAboveLiveLocked` reads the terminal's size (`screenSize`, one ioctl
on the output). It inserts when the live region is on screen
(`live.started`) and drawn at the runner's width, the terminal is at least
as wide as the runner's width and as wide as when the live frame was drawn,
the width has not changed for 500 ms, and
`insertChunk(height, live.lastHeight) > 0`. Under tmux or zellij (which
report a resize late), if Print inserted after the last frame and within
300 ms when a width change arrives, `LivePrinter.keepRowsAbove` makes the next erase of
the old frame cover only its own height, so inserts that landed among its
rewrapped rows stay. The rows go out with
autowrap off (`CSI ? 7 l` after `CSI ? 2026 h`, `CSI ? 7 h` before
`CSI ? 2026 l`), so a narrowing that lands while the bytes travel cuts a
row's end instead of wrapping it over the next row.
Otherwise Print runs today's clear-print-redraw path with the same rendered
bytes. The live printer's state does not change: its frame is still on
screen, one or more rows lower or at the same rows.

Why each step is correct for any gap g (the blank rows below the live
region's last row), with the live region at rows T..B on a screen of H rows:

- n line feeds from B: the first min(n, g) move the cursor down; the rest
  scroll the whole screen up by s = max(0, n-g), sending its top rows into
  scrollback. The cursor ends at B+n-s; the live region at T-s..B-s.
- The distance from the cursor to the live region's top is
  (B+n-s) - (T-s) = n+L-1, whatever g is. So CR and `CSI n+L-1 A` reach it.
- `CSI n L` pushes the live region down n rows. Its new last row is
  B-s+n = min(B+n, H): it fits, so no live row is lost. Only blank rows
  fall off the bottom.
- The rows fill the n inserted lines; `CSI L B` returns to the live region's
  last row.
- The one condition: s must not exceed T-1 (the live region must not scroll
  off the top), which holds while n <= H-L. Chunks of (H-L)/2 keep a margin
  for a screen that shrank and has not told the runtime yet.

**Size.** `terminalSize(output)` reads the width and height from the
terminal the runner writes to. `NewInlineApp` uses it for the default width
and the live region's cap, and `Run` reads both again when it starts
(`refreshScreenSize`). `ResizeEvent` and Handoff's repaint update them as
before. After every `ResizeEvent` (and `LivePrinter.SetWidth` with a new
width) the next frame writes every row, because Print no longer redraws the
live region and a terminal may change rows around the cursor on a resize.

**Failure.** If the write fails, Print writes `ESC[?7h ESC[?2026l`, so
autowrap is on again and the terminal does not keep holding its screen.

**Files.** `tui/inline_insert.go` (new), `tui/inline_app.go` (Print and one
call in Run), tests in `tui/inline_app_print_test.go`, the example
`examples/tui/inline_stream`, and `tui/README.md`.

## Alternatives considered

- **A scroll region from row 1 to the row above the live region** (as
  Codex). Built first and rejected. It needs the live region's absolute row
  (a cursor position query, tagged replies, and rules for every event that
  moves the live region) and the exact screen height. Output already on its
  way when the window resizes uses the old height and scrolls the wrong
  rows: 499 and 498 of 500 lines kept in an emulator harness that resizes
  libghostty-vt and the pty together, 320 and 418 of 500 in tmux, which
  passes a new size to the program only about every 250 ms during a drag.
  It also needed a fallback for Windows Terminal and GNU screen. The
  relative insert kept 500 of 500 in every case.
- **One insert for any view.** Simpler, but a view taller than the free rows
  (or a shrink in flight) would scroll live region rows into scrollback.
- **termtest region code inside `ansi.go` and `screen.go`.** Smaller diff in
  total, but four tracks edit those files at once. New files keep each
  track's conflicts to a few call sites.

## Tradeoffs and consequences

- termtest output changes for any stream that erases with a background
  color set, or sends `CSI 3 J` alone. Both now match real terminals. The
  PR lists any snapshot that changes.
- The insert moves the live region up and back within one write. A
  terminal that ignores synchronized output can show it one row higher for
  a frame; today's redraw shows it blank in the same case.
- Print relies on the cursor resting on the live region's last row, as
  today's path already does. Output written to the terminal outside the
  runtime breaks both paths alike.

## Rollout

No flag. Part 1 (termtest) merges first. Part 2 follows with the example
`examples/tui/inline_stream`, the tmux and harness results, and a manual
check in Ghostty.app and Terminal.app.
