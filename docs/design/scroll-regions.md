# Scroll regions in termtest and InlineApp.Print

**Status:** Draft\
**Author:** Curtis Myzie (with Claude)\
**Date:** 2026-10-31\
**Workflow:** Spec, then build in parallel. The termtest part has no open design question and is built first; the tui part waits for review of this spec.\
**PRD:** [docs/prds/scroll-regions.md](../prds/scroll-regions.md)

## Context

`InlineApp.Print` clears the live region, prints, and redraws it: 856 bytes
for a 46-byte line under a 5-line live region. A scroll region from row 1
to the row above the live region lets Print add lines without touching the
live region, as Codex does. termtest cannot model scroll regions, BCE, or
scrollback today, so neither change can be tested. The PRD has the user
problem, the event tables for the live region's position, and the
decisions. This spec covers how to build it.

## Goals

- termtest matches ghostty on new rows of `TestGhosttyProbes`
  (`termtest/ghostty_probes_test.go`) for BCE, DECSTBM, IND/RI/NEL at
  margins, SU/SD, IL/DL in a region, origin mode, C0 in CSI, and scrollback
  text. The expected values come from ghostty through the vtdiff harness,
  which links libghostty with cgo and stays outside the repo
  (`vtdiff dump <families>` prints the rows).
- Bytes per Print in the PRD's measurement scenario: 100 or fewer (from 856).
- The live region's bottom gap follows the PRD tables exactly; each table
  row has a test.
- No exit path leaves a scroll region set, including a write that fails in
  the middle of a scroll-region Print.

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

**Bottom gap state.** In `InlineApp`, guarded by `r.mu`:

```go
type liveGap struct {
	known   bool
	gap     int     // rows below the live region's last row, when known
	height  int     // terminal rows the gap is measured against
	tag     int     // tag of the newest query; 0 = none waiting
	pending []gapOp // ops since the newest query, replayed onto its reply
}
type gapOp struct{ kind gapOpKind; n int } // printed n rows, grew n, shrank n
```

`invalidate()` sets `known = false`, clears `pending`, and marks that a query
is needed. `apply(op)` updates `gap` when known, and appends to `pending`
when a query waits. `pending` has a cap of 256; past it, `invalidate()`.

**Query.** Queries are allowed only while `Run` reads input: from after
`initializeManagedInput` in `Run` until `Run` stops, and not during a
`Handoff` callback. Before `Run`, an event that invalidates leaves the state
unknown and sends nothing; a reply that no reader reads would reach the
shell as typed text. After a live-region draw (end of `renderLiveLocked`
and `restoreLiveLocked`), if queries are allowed, a query is needed, the
scroll path is allowed (R-11), and the width is at least 28: pick the next
tag, cycling through `1..width-20`, then write `\r ESC[<19+tag>C ESC[6n \r`
inside the same output-gate section. The range is 8 tags at width 28 and 80
at width 100, so an old reply matches a reused tag only when it arrives
after that many newer queries. The cursor is hidden and the next live write
starts with `\r`, so moving the column is invisible. Column `20+tag`
(1-based) is the tag. It cannot match legacy modified F3 (`CSI 1;m R`, m at
most 16).

**Reply.** New exported event in `terminal/decoder.go`:

```go
// CursorPositionEvent is a cursor position report: the terminal's answer
// to "ESC [ 6 n". X and Y are 0-based cells, like MouseEvent.
type CursorPositionEvent struct {
	X, Y int
	Time time.Time
}
```

`decodeCSIKey` returns it for `CSI row ; col R` with two numeric
parameters. Today that sequence decodes to `KeyUnknown`, so no key is lost.
`InlineApp.processEvent` handles it before focus routing and never passes it
to `HandleEvent`: if `X+1 == 20+tag` for the newest tag, set
`gap = height - (Y+1)`, replay `pending`, `known = true`, `tag = 0`.
Otherwise drop it. `tui` adds no alias for the type, because no tui
application receives it.

**Events.** Each row of the PRD tables maps to one call site:

| Event | Call site | Action |
|-------|-----------|--------|
| Run start | after first `render()` in `Run` | `invalidate()` |
| Resize | `processEvent` ResizeEvent branch; also set `height` | `invalidate()` |
| Handoff return | after repaint in `InlineApp.Handoff` | `invalidate()` |
| ClearScrollback | `ClearScrollback` | `invalidate()` |
| PrintRaw | `PrintRaw` | `invalidate()` |
| Write error | `checkedLiveOutput` error in Print/render | `invalidate()` |
| Print, today's path | `Print`, n = rendered view height | `apply(printed n)` |
| Print, scroll path | `Print` | none |
| Live height change | `LivePrinter.writeFrame` reports old and new height | `apply(grew/shrank)` |

`LivePrinter` gets an unexported callback field for the height change, set
by `InlineApp`; standalone use is unchanged. The terminal height comes from
`term.GetSize` at start and from each `ResizeEvent`.

**Print.** Render the view to a buffer first, so a panic in a view cannot
happen with a region set. Both paths render once, with the options today's
path uses: `Fprint(&b, view, WithWidth(r.config.Width), WithRawMode(true),
WithColor(true))`. Raw mode ends each row with `\r\n`, so the scroll path
splits the buffer on `\r\n` and gets the same rows today's path prints.
Then, if `known && gap == 0 && height - liveHeight >= 2` and R-11 allows:

```
ESC[?2026h  ESC7  ESC[1;<height-liveHeight>r  ESC[<height-liveHeight>;1H  ESC[0m
  for each rendered row:  \r\n <row>
ESC[r  ESC8  ESC[?2026l
```

DECSTBM homes the cursor, so DECSC/DECRC keep the live region's cursor.
One `ESC[0m` before the first LF gives the first scrolled-in row the
default background (BCE). Each rendered row with a style already ends with
`ESC[0m`, so the later LFs also scroll in default-background rows. Overhead:
43 bytes for one row (16 of them the 2026 bracket) plus 2 per extra row,
with a 5-line live region on a 30-row screen. Otherwise, today's path, then
`apply(printed n)`.

**Failure.** Set `regionMaybeSet = true` before the write and clear it after
a complete write. If the write fails, `invalidate()`, then:

- If the write failed before `ESC7` went out (fewer bytes than
  `ESC[?2026h ESC7`), no region is set and the saved cursor is not Print's.
  Write `ESC[?2026l` only and clear the flag.
- Otherwise write `ESC[r ESC8 ESC[?2026l` at once. `ESC[r` resets the region
  and homes the cursor; `ESC8` then returns it to the cursor that this
  Print saved with `ESC7`, on the live region's last row. Never write `ESC7`
  in a reset: it would overwrite Print's saved cursor with the mid-print
  cursor, and `ESC8` would restore that (PRD UC-3 would fail). Clear the
  flag when this write succeeds.

While the flag is still set, every later write to the output (live draw,
query, Print on either path, `cleanup()`, the Handoff release path) starts
with `ESC[r ESC8` and clears the flag when that write succeeds. No other
output writes `ESC7` while the flag is set, so the saved cursor stays
Print's, and the live renderer's idea of the cursor (the state before the
failed Print) is still right. The reset is idempotent, so a repeat after a
partial write is safe. A scroll region never survives past one Print
otherwise, so normal exit, panic, and Handoff need no extra bytes.

**Opt-out.** `NewInlineApp` reads `WT_SESSION`, `STY`, and
`WONTON_NO_SCROLL_REGION` once. When any blocks the path, no query is sent
and Print always uses today's path. Documented in the `InlineApp` doc
comment and `tui/doc.go`.

## Alternatives considered

- **Count replies in order instead of tagging.** CPR replies come in query
  order, so the last reply answers the newest query. But one lost reply
  (dropped by a multiplexer, or read by a Handoff child) shifts the count
  for the rest of the session, and a pre-resize reply looks valid. The tag
  costs 8 bytes per query and needs no ordering assumption.
- **Synchronous query before the input reader starts** (as ratatui and
  `DetectTerminalBackground` do). It works at start, but resize and
  Handoff need a query while the reader runs, so the async path is needed
  anyway. One path is simpler than two.
- **termtest region code inside `ansi.go` and `screen.go`.** Smaller diff in
  total, but four tracks edit those files at once. New files keep each
  track's conflicts to a few call sites.
- **Codex-style RI push when the live region is not at the bottom.** It
  makes the first screenful use the scroll path too, but adds a second
  region sequence for a short phase. Rejected in the PRD.

## Tradeoffs and consequences

- termtest output changes for any stream that erases with a background
  color set, or sends `CSI 3 J` alone. Both now match real terminals. The
  PR lists any snapshot that changes.
- A new exported type, `terminal.CursorPositionEvent`. A full-screen
  `Runtime` app that sends its own `ESC[6n` now receives it in `HandleEvent`
  instead of a `KeyUnknown` key.
- The gap tracking is a small state machine with nine inputs. It is the
  main source of risk; the event table and one test per row contain it.
- Each query moves the cursor column of a hidden cursor and costs about 12
  bytes, once per start, resize, Handoff, ClearScrollback, or PrintRaw.

## Rollout

No flag; the path turns on by itself where it is safe. Part 1 (termtest)
merges first as its own PR stacked on the PRD branch. Part 2 follows with
the example `examples/tui/inline_stream` and a manual transcript from
ghostty, Terminal.app, and tmux. `WONTON_NO_SCROLL_REGION=1` is the
fallback if a terminal loses scrollback; [#208](https://github.com/deepnoodle-ai/wonton/issues/208)
tracks Windows Terminal over SSH.

## Open questions

- tmux: does it forward the CPR reply in order with key input when the
  outer terminal is slow? Leaning yes (tmux answers CPR itself from its own
  grid). Checked in the manual tmux run before part 2 merges.
