# Print above a live region without redrawing it

Status: Draft\
PRD PR: [#210](https://github.com/deepnoodle-ai/wonton/pull/210) · Implementation PR: [#215](https://github.com/deepnoodle-ai/wonton/pull/215) · Updated: 2026-11-01

## Problem

A developer builds an inline CLI on wonton, such as a coding agent. Output
scrolls up while a live region (spinner, status, input line) stays at the
bottom. Each `InlineApp.Print` clears the live region, prints, and draws the
whole live region again (`tui/inline_app.go:1019`).

- Cost: with a 5-line live region at width 100, one `Printf` of a 46-byte
  log line writes 855 bytes. 500 lines write 428 KB instead of about 40 KB.
  The cost grows with the height of the live region. Over SSH this is slow.
- Flicker: Print is wrapped in synchronized output (mode 2026). Terminals
  that ignore mode 2026, such as macOS Terminal.app and xterm, can show the
  cleared live region between the clear and the redraw.

Test authors cannot test a better approach. `termtest` ignores scroll
regions (`CSI t;b r`), has no background color erase (BCE), keeps no
scrollback, and mishandles a C0 control inside a CSI sequence. A test cannot check that printed lines reach scrollback in
order, which is the main risk of any change here.

If we do nothing, wonton CLIs keep paying a full live-region redraw per
line, and termtest stays unable to verify output that real terminals and
other libraries already depend on.
## Context

Comparable libraries (details in the track's discovery notes):

- **Bubble Tea v2** (`InsertAbove` in its renderer) adds lines above an
  inline view with line feeds, cursor moves, and insert line, without a
  scroll region.
- **OpenAI Codex CLI** (inline viewport) sets a scroll region from row 1 to
  the row above its viewport and writes each line at the region bottom. It
  needs the viewport's absolute row, so it asks the terminal for the cursor
  position, and it falls back to a full redraw in Windows Terminal, which
  discards rows that leave a partial region.
- **Ratatui** offers the scroll-region technique behind an opt-in
  `scrolling-regions` feature.
- **Ink** and **prompt_toolkit** erase and redraw, as wonton does today.
- **Ghostty, xterm, tmux, and Windows Terminal** all move a row into
  scrollback when a line feed scrolls the whole screen.

Wonton's live region uses only cursor moves relative to itself and does not
know its absolute row. We first built the scroll-region design (see
Decisions); it lost lines when the window was resized during output.

## Users and job

Primary user: a Go developer who builds an inline agentic CLI with
`InlineApp` and prints many lines with `Print` or `Printf`. Their job:
stream output into scrollback while the live region stays still and
correct. We optimize for them.

Secondary user: a developer who tests terminal output with `termtest` and
needs it to behave like a real terminal.

## Concepts

The tui user meets no new concept. `Print` keeps its signature and its
result on screen and in scrollback.

The termtest user meets terms they already know from terminals:

- **Scroll region**: the rows between a top and bottom margin, set with
  `CSI t;b r`. Line feeds and scrolls move only these rows.
- **Scrollback**: rows that scrolled off the top of the screen, oldest
  first. New in termtest as `Screen.Scrollback()`.

## How Print adds rows

From the live region's last row, where the cursor rests, Print writes, in
one synchronized-output bracket:

1. `ESC[0m` and n line feeds. Where the live region sits at the screen
   bottom, they scroll the whole screen, so the top rows go into
   scrollback in order. Where blank rows lie below it, the first line feeds
   only move the cursor down.
2. A carriage return and `CSI n+L-1 A` (L is the live region's height).
   Either way, this reaches the live region's top row.
3. `CSI n L` (insert line), which pushes the live region back down into the
   rows that step 1 opened.
4. The n rows, then `CSI L B` back to the live region's last row.

Every move is relative to the cursor. No step depends on the screen height
or on the live region's absolute row, so a resize that lands while the
bytes travel cannot make Print scroll the wrong rows.

## Use cases

### UC-1: An agent CLI streams 500 lines under a live region

The developer calls `runner.Printf("log line %d", i)` 500 times while a
spinner and input line show at the bottom.

Acceptance:
- [ ] Each Print writes no bytes into the live region rows. With a live
      region of fewer than 10 rows, a one-row Print writes the row plus 34
      bytes of control sequences, plus 3 bytes for each row after the first
      (a line feed and the CRLF between rows). The synchronized-output
      bracket is 16 of the 34.
- [ ] In the measurement scenario above (5-line live region, width 100),
      bytes per Print drop from 855 to 100 or fewer.
- [ ] A termtest test runs 500 Prints with the insert and 500 with today's
      redraw. The visible screen is equal on both, cell by cell, with
      styles. `Screen.Scrollback()` is equal on both. Every printed line is
      in scrollback or on screen, once, in order.
- [ ] In the first screenful (blank rows below the live region), after a
      live region height change, and for a view taller than the free rows,
      the insert gives the same screen and scrollback as the redraw.

### UC-2: The user resizes the window during output

Acceptance:
- [ ] The example streams 500 lines while the window shrinks from 30 to 20
      rows (one row each 30 ms), and while it alternates between 20 and 32
      rows 20 times. All 500 lines are in scrollback, in order, and no live
      region row is in scrollback. Checked in tmux and in an emulator
      harness that resizes libghostty-vt and the pty together. termtest
      cannot resize yet, so this check stays outside the test suite.

### UC-3: A write fails in the middle of a Print

Acceptance:
- [ ] Print still ends synchronized output (`CSI ? 2026 l`), so the
      terminal does not keep holding its screen. Print leaves no mode set.

### UC-4: A test asserts scroll-region output

```go
s := termtest.NewScreen(10, 5)
s.Write([]byte("1\r\n2\r\n3\r\n4\r\n5"))
s.Write([]byte("\x1b[2;4r"))     // region rows 2-4, cursor goes home
s.Write([]byte("\x1b[4;1H\nX"))  // LF at region bottom scrolls rows 2-4
// rows: "1", "3", "4", "X", "5"; Scrollback() is empty (region not at row 1)
```

Acceptance:
- [ ] Each new behavior in R-1 to R-6 has a row in `TestGhosttyProbes`
      (`termtest/ghostty_probes_test.go`), and termtest matches every row:
      text, cursor, background per cell, and scrollback text. The expected
      values in each row come from ghostty, through the vtdiff harness. The
      harness links libghostty with cgo, so it stays outside this repo; the
      test needs neither the harness nor cgo.
- [ ] Existing termtest, tui, and cli tests and snapshots pass. Any snapshot
      that changes is explained in the PR as what a real terminal shows.

## Requirements

termtest (lands first, as its own stacked PR):

- R-1 BCE: ED, EL, ECH, DCH, ICH, IL, DL, SU, SD, and every scroll (LF,
  IND, NEL, RI, autowrap) fill new cells with the current background color
  and no other attribute, as xterm and ghostty do.
- R-2 Regions: `CSI t;b r` sets the region and moves the cursor home.
  Missing or zero parameters mean the screen edges. A bottom past the
  screen means the last row. A region with top >= bottom is ignored. RIS
  resets the region.
- R-3 Region-aware motion: LF, IND, NEL, and autowrap at the region bottom
  scroll the region up. RI at the region top scrolls it down. `CSI n S` and
  `CSI n T` scroll the region. IL and DL do nothing when the cursor is
  outside the region and stop at its bottom. Inside the region they move
  the cursor to column 0. CUU and CUD stop at a margin
  when the cursor starts inside the region. A cursor below the region moves
  down to the last screen row and does not scroll.
- R-4 Origin mode `CSI ?6 h/l`: CUP, VPA, and the home position are relative
  to the region top, and the cursor cannot leave the region. Setting or
  resetting the mode moves the cursor home. DECSC/DECRC save and restore it.
- R-5 C0 inside CSI: CR, LF, VT, FF, BS, and HT inside a CSI sequence run
  before the sequence itself, in the order they came. They run when the
  sequence ends or is cancelled (CAN, SUB, ESC). BEL is ignored. When the
  sequence ends, the screen is the same as in xterm and ghostty, which run
  each control when it arrives. The difference shows only while a sequence
  is incomplete: if the stream stops inside a sequence, its controls have
  not run yet. This is deliberate. termtest buffers an incomplete sequence
  and parses it again when the next write arrives, so running controls on
  arrival would run them twice; that needs a byte-by-byte parser, which is
  a larger change than this track.
- R-6 Scrollback: `Screen.Scrollback() []string` returns rows that scrolled
  off the top of the screen, oldest first, as `Row` text. A scroll (LF,
  IND, NEL, autowrap, or SU) of a region that starts at row 1 adds the
  rows that leave it, as in ghostty. DL and a region that starts lower drop
  them. `ED 3` empties the scrollback and leaves the screen as it is, as in
  ghostty and xterm.

tui:

- R-7 `Print` and `Printf` add rows as "How Print adds rows" describes,
  when the live region is on screen and leaves at least 2 free rows
  (screen height minus live region height). They add at most half the free
  rows per insert and repeat the steps for the rest, so a screen that
  shrank and has not yet said so still has room.
- R-8 Otherwise, `Print` clears and redraws the live region, as today.
  `PrintRaw` and `ClearScrollback` always do.
- R-9 The insert keeps the synchronized-output bracket and resets text
  attributes before the line feeds, so the rows it opens have the default
  background. Each rendered row already ends with a reset.
- R-10 If a write fails during an insert, Print sends `CSI ? 2026 l`.
- R-11 `Run` reads the screen height again when it starts.
- R-12 A runnable example, `examples/tui/inline_stream`, streams lines under
  a spinner and input line.

## Not in scope

- Scroll regions in the full-screen renderer (ScrollView or viewport with
  SU/SD). It needs scroll detection in the diff renderer, and Bubble Tea
  limits it to the alternate screen. Deferred to a GitHub issue.
- `PrintRaw` through the insert. Raw bytes can hold any sequence, so the
  row count is unknown.
- Standalone `LivePrinter`. Deferred.
- Left and right margins (DECSLRM, `?69`) in termtest. Wonton does not use
  them.
- Alternate screen, raw mode, and resize in termtest. Other tracks own them.

## Decisions

- Land termtest first as its own stacked PR. The tui change depends on it
  for tests, and it has value alone. Rejected: one PR for both, which is
  harder to review.
- Insert rows above the live region with relative moves (line feeds,
  cursor up, insert line). It keeps every line when the window resizes
  during output, writes fewer bytes, and needs no cursor-position query, no
  tracking of the live region's row, no size check, and no list of
  terminals to avoid. Rejected: a scroll region from row 1 to the row above
  the live region, as Codex does. A scroll region takes absolute rows, so
  the runtime must know the live region's row (a cursor position query and
  rules to keep it current) and the exact screen height. During a resize,
  output already on its way uses the old height: in an emulator harness
  (libghostty-vt and a pty resized together) it kept 499 and 498 of 500
  lines; in tmux, which passes a new size to the program only about every
  250 ms during a drag, it kept 320 and 418 of 500. The relative insert kept
  500 of 500 in all of these. It also needed a fallback for Windows
  Terminal and GNU screen, which drop rows that leave a partial region.
- Cost of the insert: within one write the live region moves up n rows and
  back down. On a terminal that ignores synchronized output, a repaint in
  the middle of the write can show it shifted by a row for one frame.
  Today's redraw shows it blank in the same case; the scroll region did not
  move it. Accepted: correctness on resize matters more.
- Insert at most half the free rows at a time. An insert of n rows is
  correct while n plus the live region height fits the screen; the margin
  covers a screen that shrank before the runtime heard of it. Rejected: one
  insert for any view, which a shrink in flight could break.
- No opt-out variable. The insert uses only line feeds, cursor up and down,
  and insert line, which every VT102-compatible terminal supports, and a
  whole-screen line feed keeps scrollback everywhere we know of. Rejected:
  `WONTON_NO_SCROLL_REGION`, which only the scroll region needed.
- termtest records scrollback always, with no Screen option. Tests are
  small, and an option would add a knob for no user. If one is needed
  later, it uses the shared `termtest.ScreenOption` type from the raw-mode
  track; this track adds no second option type.
- BCE fills only the background, not bold, underline, or reverse. This
  matches xterm and ghostty.

## Success

- Bytes per Print in the measurement scenario drop from 855 to 80, and stop
  growing with live-region height.
- 500 printed lines are in scrollback in order, with a stable window and
  while it resizes, in tmux and the emulator harness. Manual check in
  Ghostty.app and macOS Terminal.app; the PR holds the transcript.
- Must not get worse: scrollback content and order, the live region after
  resize and hand-off, terminal state after exit. Existing tests and
  snapshots stay correct. The vtdiff probe, render, and fuzz runs stay
  clean.

## Risks and open questions

- A terminal that ignores synchronized output can show the live region one
  row higher for a frame during a Print (see Decisions). Not blocking.
- Output that the application writes to the terminal outside the runtime
  (for example `fmt.Println` while `Run` is active) moves the cursor away
  from the live region, as it does today. Not blocking.
- Four tracks edit termtest. The private-mode dispatch in `parseCSI` is
  shared with the alt-screen and raw-mode tracks. The tech spec gives the
  file layout. Not blocking.
