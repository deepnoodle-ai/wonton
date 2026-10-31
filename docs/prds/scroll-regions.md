# Print above a live region without redrawing it

Status: Draft\
PRD PR: #TBD · Implementation PR: #TBD · Updated: 2026-10-31

## Problem

A developer builds an inline CLI on wonton, such as a coding agent. Output
scrolls up while a live region (spinner, status, input line) stays at the
bottom. Each `InlineApp.Print` clears the live region, prints, and draws the
whole live region again (`tui/inline_app.go:1019`).

- Cost: with a 5-line live region at width 100, one `Printf` of a 46-byte
  log line writes 856 bytes. 500 lines write 428 KB instead of about 40 KB.
  The cost grows with the height of the live region. Over SSH this is slow.
- Flicker: Print is wrapped in synchronized output (mode 2026). Terminals
  that ignore mode 2026, such as macOS Terminal.app and xterm, can show the
  cleared live region between the clear and the redraw.

Test authors cannot test a better approach. `termtest` ignores scroll
regions (`CSI t;b r`), has no background color erase (BCE), keeps no
scrollback, and mishandles a C0 control inside a CSI sequence
(findings.md). A test cannot check that printed lines reach scrollback in
order, which is the main risk of any change here.

If we do nothing, wonton CLIs keep paying a full live-region redraw per
line, and termtest stays unable to verify output that real terminals and
other libraries already depend on.

## Context

Comparable libraries (details in the track's discovery notes):

- **OpenAI Codex CLI** (inline viewport) does exactly this. It sets a
  scroll region from row 1 to the row above the viewport, writes `\r\n` and
  each line at the region bottom, then resets the region. The viewport is
  never redrawn. It uses line feeds, not `CSI S`, because `CSI S` loses rows
  in xterm.js and QTermWidget. It falls back to a full redraw in Windows
  Terminal, which discards rows that leave a partial region.
- **Ratatui** offers the same technique behind an opt-in `scrolling-regions`
  feature, with `CSI S`.
- **Bubble Tea v2**, **Ink**, and **prompt_toolkit** erase and redraw, as
  wonton does today. Bubble Tea uses scroll regions only in the full-screen
  alternate-screen renderer, and not on Windows.
- **Ghostty and xterm** move a row that leaves a region into scrollback only
  when the region starts at row 1. tmux does the same.

A scroll region uses absolute rows. Wonton's live region uses only relative
cursor moves and does not know its absolute row. Wonton never asks the
terminal for the cursor position today.

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

## Use cases

### UC-1: An agent CLI streams 500 lines under a live region

The developer calls `runner.Printf("log line %d", i)` 500 times while a
spinner and input line show at the bottom.

Acceptance:
- [ ] When the live region's bottom is on the last screen row, each Print
      writes no bytes into the live region rows. It writes the printed
      content plus at most 40 bytes of control sequences.
- [ ] In the measurement scenario above (5-line live region, width 100),
      bytes per Print drop from 856 to 100 or fewer.
- [ ] The screen and the scrollback after 500 Prints are equal to what
      today's path produces: same lines, same order, same styles, live region
      intact. A termtest test checks this for both paths.
- [ ] Before the live region reaches the screen bottom (the first
      screenful), Print uses today's path. The result on screen is the same.

### UC-2: The user resizes the window during output

Acceptance:
- [ ] After a resize, Print uses today's path until wonton knows the live
      region's row again. No Print writes a scroll region based on the old
      size.
- [ ] A test resizes, prints, and checks that the live region and the
      printed lines are correct.

### UC-3: The app exits, panics, or hands off the terminal

Acceptance:
- [ ] After `Run` returns (normal quit, error, or panic), and while a
      `Handoff` callback runs, the terminal has no scroll region set. Tests
      check this with termtest on each path.
- [ ] If a write fails partway through a scroll-region Print, cleanup
      resets the region and keeps the cursor where it was.

### UC-4: A test asserts scroll-region output

```go
s := termtest.NewScreen(10, 5)
s.Write([]byte("1\r\n2\r\n3\r\n4\r\n5"))
s.Write([]byte("\x1b[2;4r"))     // region rows 2-4, cursor goes home
s.Write([]byte("\x1b[4;1H\nX"))  // LF at region bottom scrolls rows 2-4
// rows: "1", "3", "4", "X", "5"; Scrollback() is empty (region not at row 1)
```

Acceptance:
- [ ] Each new behavior in R-1 to R-5 has a probe in the vtdiff harness, and
      termtest matches ghostty on every probe (text, cursor, and style per
      cell, plus scrollback where the harness can read it).
- [ ] Existing termtest, tui, and cli tests and snapshots pass. Any snapshot
      that changes is explained in the PR as what a real terminal shows.

## Requirements

termtest (lands first, as its own stacked PR):

- R-1 BCE: ED, EL, ECH, DCH, ICH, IL, DL, SU, SD, and every scroll (LF,
  IND, NEL, RI, autowrap) fill new cells with the current background color
  and no other attribute, as xterm and ghostty do.
- R-2 Regions: `CSI t;b r` sets the region and moves the cursor home.
  Missing or zero parameters mean the screen edges. An invalid region
  (top >= bottom, or out of range) is ignored. RIS resets the region.
- R-3 Region-aware motion: LF, IND, NEL, and autowrap at the region bottom
  scroll the region up. RI at the region top scrolls it down. `CSI n S` and
  `CSI n T` scroll the region. IL and DL do nothing when the cursor is
  outside the region and stop at its bottom. CUU and CUD stop at a margin
  when the cursor starts inside the region. A cursor below the region moves
  down to the last screen row and does not scroll.
- R-4 Origin mode `CSI ?6 h/l`: CUP, VPA, and the home position are relative
  to the region top, and the cursor cannot leave the region. Setting or
  resetting the mode moves the cursor home. DECSC/DECRC save and restore it.
- R-5 C0 inside CSI: CR, LF, VT, FF, BS, and HT inside a CSI sequence run
  when they arrive, and the sequence continues, as in xterm. BEL is ignored.
- R-6 Scrollback: `Screen.Scrollback() []string` returns rows that scrolled
  off the top of the screen, oldest first, as `Row` text. Only LF, IND,
  NEL, or autowrap at the bottom of a region that starts at row 1 adds a
  row, as in ghostty. SU, DL, and a region that starts lower drop the row.
  `ED 3` empties it.

tui:

- R-7 When the runtime knows that the live region's bottom is on the last
  screen row, and at least 2 rows sit above the live region, `Print` and
  `Printf` write through a scroll region from row 1 to the row above the
  live region, with a line feed per printed row. The region is set and reset
  inside the same write, and the cursor returns to where it was.
- R-8 In every other case, `Print` uses today's path. `PrintRaw` and
  `ClearScrollback` always use today's path.
- R-9 The runtime learns the live region's row by asking the terminal for
  the cursor position at start and after each resize. Print never waits for
  the answer. The answer does not reach `HandleEvent`.
- R-10 In Windows Terminal (`WT_SESSION` set) and GNU screen (`STY` set),
  Print always uses today's path.
- R-11 The new path keeps the synchronized-output bracket and resets text
  attributes before each line feed, so scrolled-in rows have the default
  background.
- R-12 A runnable example, `examples/tui/inline_stream`, streams lines under
  a spinner and input line.

## Not in scope

- Scroll regions in the full-screen renderer (ScrollView or viewport with
  SU/SD). It needs scroll detection in the diff renderer, and Bubble Tea
  limits it to the alternate screen. Deferred to a GitHub issue.
- `PrintRaw` through a scroll region. Raw bytes can hold any sequence, so
  the row count is unknown.
- Standalone `LivePrinter`. It has no input reader to receive the cursor
  position. Deferred.
- Left and right margins (DECSLRM, `?69`) in termtest. Wonton does not use
  them.
- A persistent scroll region (a status bar that stays while the app writes
  freely). It needs a restore on every exit path and breaks child processes.
- Alternate screen, raw mode, and resize in termtest. Other tracks own them.

## Decisions

- Land termtest first as its own stacked PR. The tui change depends on it
  for tests, and it has value alone. Rejected: one PR for both, which is
  harder to review.
- Set and reset the region inside one write. Then no exit, panic, resize,
  or hand-off path has a region to restore. Rejected: keep a region set
  between writes, as some status-bar UIs do.
- Learn the absolute row from a cursor position report, and use today's
  path until it arrives. Ratatui and Codex do the same. Rejected: move the
  live region to the screen bottom at start, which leaves a visible blank
  gap. Rejected: block Print on a query, which stalls output.
- Use the scroll path only when the live region sits at the screen bottom.
  Today's path already moves it there during the first screenful. Rejected:
  push the live region down with RI as Codex does, which adds a second code
  path for the first screenful only.
- Scroll with line feeds, never `CSI S`. Line feeds keep scrollback in every
  terminal Codex tested; `CSI S` loses rows in xterm.js and QTermWidget.
- Fall back by a short denylist (Windows Terminal, GNU screen). Losing
  scrollback is worse than a redraw. Rejected: an allowlist, which would
  exclude Terminal.app and xterm, the terminals without mode 2026 that gain
  the most.
- Add no option to turn the new path off. The output is the same, so a
  caller has no reason to choose. Rejected: `WithInlineScrollRegion`, which
  would expose the term "scroll region" to every user.
- termtest records scrollback always, with no Screen option. Tests are
  small, and an option would add a knob for no user. If one is needed
  later, it uses the shared `termtest.ScreenOption` type from the raw-mode
  track; this track adds no second option type.
- A cursor position reply is recognized only while a query is waiting.
  Legacy Shift+F3 (`CSI 1;2 R`) has the same bytes as a reply at row 1,
  column 2. Rejected: DECXCPR (`CSI ?6n`), which some terminals do not
  answer.
- BCE fills only the background, not bold, underline, or reverse. This
  matches xterm and ghostty.

## Success

- Bytes per Print in the measurement scenario drop from 856 to 100 or
  fewer, and stop growing with live-region height.
- Manual check on ghostty, macOS Terminal.app, and tmux: 500 printed lines
  are in scrollback in order, and the live region does not flicker. The PR
  holds the transcript.
- Must not get worse: scrollback content and order, the live region after
  resize, terminal state after exit. Existing tests and snapshots stay
  correct. The vtdiff probe, render, and fuzz runs stay clean.

## Risks and open questions

- A terminal outside the denylist could discard rows from a top-anchored
  region, and the user loses history. Codex ships this path to a large user
  base with the same two fallbacks, and we verify three terminals manually.
  Not blocking.
- Tracking the live region's row can drift (for example, after the user
  types into the shell during startup). The answer after each resize
  corrects it, and a wrong row only risks a misplaced region for one Print.
  The tech spec defines the tracking rules. Not blocking.
- Four tracks edit termtest. The private-mode dispatch in `parseCSI` is
  shared with the alt-screen and raw-mode tracks. New logic goes in new
  files, and edits to `ansi.go` and `screen.go` stay small. Not blocking.
