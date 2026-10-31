# Print above a live region without redrawing it

Status: Draft\
PRD PR: [#210](https://github.com/deepnoodle-ai/wonton/pull/210) · Implementation PR: #TBD · Updated: 2026-10-31

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
- **Windows Terminal** discards rows that leave a partial region (Codex
  #39619). SSH does not forward `WT_SESSION`, so a remote program cannot
  detect it.

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
result on screen and in scrollback. An end user who sees lost scrollback can
set `WONTON_NO_SCROLL_REGION=1` (see R-11). It is a support escape hatch,
not an API.

The termtest user meets terms they already know from terminals:

- **Scroll region**: the rows between a top and bottom margin, set with
  `CSI t;b r`. Line feeds and scrolls move only these rows.
- **Scrollback**: rows that scrolled off the top of the screen, oldest
  first. New in termtest as `Screen.Scrollback()`.

## Where the live region is

Print can use a scroll region only when it knows where the live region is.
The runtime keeps one value, the **bottom gap**: the number of screen rows
below the live region's last row. It has three states:

| State | Meaning | Print uses |
|-------|---------|------------|
| At bottom | gap is 0 | the scroll-region path (if R-8 allows it) |
| Not at bottom | gap is a known number above 0 | today's path, then counts |
| Unknown | the runtime cannot know the gap | today's path |

The runtime asks the terminal for the cursor position (a cursor position
report, CPR) right after it draws the live region. The cursor is then on
the live region's last row, so the gap is the screen height minus the
reported row.

Events that set the state to **unknown** and send a new query:

| Event | Why counting cannot cover it |
|-------|------------------------------|
| Start of `Run` | The shell left the cursor at an unknown row. |
| Resize | Terminals differ: they reflow, pull rows from scrollback, or add rows. |
| `Handoff` return | The child can print any number of lines or clear the screen. |
| `ClearScrollback` | Terminals differ in how `ED 2` and `ED 3` treat the screen. |
| `PrintRaw` | Raw bytes can move the cursor anywhere. |
| A write error on the output | The terminal may have received part of a sequence. |

Events that keep the state **known by counting**:

| Event | New gap |
|-------|---------|
| Print on today's path, n rows | gap minus n, not below 0 |
| Print on the scroll-region path | 0 (unchanged) |
| Live region grows by k rows | gap minus k, not below 0 |
| Live region shrinks by k rows | gap plus k |
| Live region redraw with the same height | unchanged |

A reply counts only if it answers the newest query. The runtime moves the
cursor to a column that tags each query before it asks, and accepts a reply
only with the newest tag. Events that occur after the query and before the
reply are counted onto the reply's value. A terminal that never replies
leaves the state unknown, so Print stays on today's path.

Output that the application writes to the terminal outside the runtime
(for example `fmt.Println` while `Run` is active) is outside this contract,
as it is today.

## Use cases

### UC-1: An agent CLI streams 500 lines under a live region

The developer calls `runner.Printf("log line %d", i)` 500 times while a
spinner and input line show at the bottom.

Acceptance:
- [ ] In the at-bottom state, each Print writes no bytes into the live
      region rows. It writes the printed content plus at most 40 bytes of
      control sequences.
- [ ] In the measurement scenario above (5-line live region, width 100),
      bytes per Print drop from 856 to 100 or fewer.
- [ ] A termtest test runs 500 Prints on each path. The visible screen is
      equal on both paths, cell by cell, with styles: the live region and
      the newest printed rows. `Screen.Scrollback()` is equal on both paths
      as text: same lines, same order. Each printed row is visible right
      after its Print, so the test checks its styles on screen at that
      moment.
- [ ] In the not-at-bottom state (the first screenful), Print uses today's
      path and counts. When the gap reaches 0, the next Print uses the
      scroll-region path. A test checks the switch.

### UC-2: The live region moves for a reason Print did not cause

Acceptance:
- [ ] One test for each event in the "unknown" table: after the event,
      Print uses today's path until a reply to the new query arrives. Then
      Print uses the scroll-region path again, and the screen is correct.
- [ ] One test for each event in the "counting" table: the gap after the
      event matches the table, checked against the cursor row in termtest.
- [ ] A reply to an older query (for example, one sent before a resize)
      changes nothing. A legacy Shift+F3 key (`CSI 1;2 R`) never matches a tag
      and is not taken as a reply.
- [ ] With no reply at all, every Print uses today's path, and output is
      the same as today.

### UC-3: A write fails in the middle of a scroll-region Print

Acceptance:
- [ ] A test output fails after the scroll region is set and before it is
      reset. After `Run` returns, the terminal has no scroll region, and the
      cursor is where it was before the failed Print. The state is unknown.
- [ ] One short check: after `Run` returns normally, after a panic, and
      during a `Handoff` callback, the termtest screen has no scroll region.

### UC-4: A test asserts scroll-region output

```go
s := termtest.NewScreen(10, 5)
s.Write([]byte("1\r\n2\r\n3\r\n4\r\n5"))
s.Write([]byte("\x1b[2;4r"))     // region rows 2-4, cursor goes home
s.Write([]byte("\x1b[4;1H\nX"))  // LF at region bottom scrolls rows 2-4
// rows: "1", "3", "4", "X", "5"; Scrollback() is empty (region not at row 1)
```

Acceptance:
- [ ] Each new behavior in R-1 to R-6 has a probe in the vtdiff harness, and
      termtest matches ghostty on every probe (text, cursor, and style per
      cell, plus scrollback text where the harness can read it).
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
  off the top of the screen, oldest first, as `Row` text. A scroll (LF,
  IND, NEL, autowrap, or SU) of a region that starts at row 1 adds the
  rows that leave it, as in ghostty. DL and a region that starts lower drop
  them. `ED 3` empties the scrollback and leaves the screen as it is, as in
  ghostty and xterm.

tui:

- R-7 The runtime keeps the bottom gap as "Where the live region is"
  defines, with the events and rules in its tables.
- R-8 In the at-bottom state, when at least 2 rows sit above the live
  region, `Print` and `Printf` write through a scroll region from row 1 to
  the row above the live region, with a line feed per printed row. The
  region is set and reset inside the same write, and the cursor returns to
  where it was.
- R-9 In every other case, `Print` uses today's path. `PrintRaw` and
  `ClearScrollback` always use today's path.
- R-10 Print never waits for a reply. A reply does not reach `HandleEvent`.
- R-11 Print always uses today's path, and the runtime sends no query, when
  `WT_SESSION` is set (Windows Terminal), `STY` is set (GNU screen), or
  `WONTON_NO_SCROLL_REGION` is set to any value other than empty or `0`.
  The tui package docs name the variable.
- R-12 The new path keeps the synchronized-output bracket and resets text
  attributes before each line feed, so scrolled-in rows have the default
  background.
- R-13 If a write fails while a scroll region may be set, cleanup resets the
  region and keeps the cursor in place.
- R-14 A runnable example, `examples/tui/inline_stream`, streams lines under
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
- Re-query after every event that counting cannot cover, and count the rest
  (tables above). A wrong gap would scroll the wrong rows on every Print
  until the next query, so the rule is: when in doubt, unknown. Rejected:
  query only at start and on resize.
- Tag each query with a cursor column and accept only the newest tag. CPR
  has no ID, and a late reply or a Shift+F3 key can look like a reply.
  Rejected: count replies in order, which a lost reply breaks for the rest
  of the session. Rejected: DECXCPR (`CSI ?6n`), which some terminals do not
  answer.
- Use the scroll path only when the live region sits at the screen bottom.
  Today's path already moves it there during the first screenful. Rejected:
  push the live region down with RI as Codex does, which adds a second code
  path for the first screenful only.
- Scroll with line feeds, never `CSI S`. Line feeds keep scrollback in every
  terminal Codex tested; `CSI S` loses rows in xterm.js and QTermWidget.
- Fall back by a short denylist (Windows Terminal, GNU screen) plus an
  environment variable. Losing scrollback is worse than a redraw. The
  variable covers Windows Terminal over SSH, which no program can detect.
  Rejected: an allowlist, which would exclude Terminal.app and xterm, the
  terminals without mode 2026 that gain the most. Rejected: a Go option,
  which would expose "scroll region" to every developer, while the person
  who sees the problem is the end user.
- Name the variable `WONTON_NO_SCROLL_REGION`. Wonton's only own variable,
  `TERMTEST_UPDATE`, is for developers; this one is for end users, who know
  the library name better than a package name. `NO_` follows `NO_COLOR`.
- termtest records scrollback always, with no Screen option. Tests are
  small, and an option would add a knob for no user. If one is needed
  later, it uses the shared `termtest.ScreenOption` type from the raw-mode
  track; this track adds no second option type.
- BCE fills only the background, not bold, underline, or reverse. This
  matches xterm and ghostty.

## Success

- Bytes per Print in the measurement scenario drop from 856 to 100 or
  fewer, and stop growing with live-region height.
- Manual check on ghostty, macOS Terminal.app, and tmux: 500 printed lines
  are in scrollback in order, and the live region does not flicker. The PR
  holds the transcript.
- Must not get worse: scrollback content and order, the live region after
  resize and hand-off, terminal state after exit. Existing tests and
  snapshots stay correct. The vtdiff probe, render, and fuzz runs stay
  clean.

## Risks and open questions

- Windows Terminal over SSH gets the new path and can lose scrollback with
  no error. We cannot test Windows Terminal here. Mitigation: the
  `WONTON_NO_SCROLL_REGION` variable, documented in the tui package docs.
  Verification is tracked in [#208](https://github.com/deepnoodle-ai/wonton/issues/208). Not blocking.
- Another terminal could discard rows from a top-anchored region. Codex
  ships this path to a large user base with the same fallbacks, and we
  verify three terminals manually. Not blocking.
- The gap is wrong if output reaches the terminal outside the runtime. The
  tables above list every runtime event; anything else is outside the
  contract today as well. Not blocking.
- Four tracks edit termtest. The private-mode dispatch in `parseCSI` is
  shared with the alt-screen and raw-mode tracks. The tech spec gives the
  file layout. Not blocking.
