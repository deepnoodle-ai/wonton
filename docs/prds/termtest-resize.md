# Test what a terminal shows after the user resizes it

Status: Draft\
PRD PR: [#205](https://github.com/deepnoodle-ai/wonton/pull/205) · Implementation PR: [#213](https://github.com/deepnoodle-ai/wonton/pull/213) · Updated: 2026-10-31

## Problem

Users resize terminal windows while wonton apps run. They drag a window
edge, split a pane, or rotate a laptop display. The full-screen runtime
and `InlineApp` both receive a `ResizeEvent` and redraw. No wonton test
can check what the user sees after that redraw, because a
`termtest.Screen` has a fixed size.

The gap hides a real bug. We sent an `InlineApp`-style stream to
libghostty-vt: one scrollback line and a two-line live region at 20
columns. We resized ghostty to 10 columns and sent the redraw that
`LivePrinter` writes today. Ghostty shows the first live line twice:

```
before (20 cols)       after resize + redraw (10 cols)
scroll                 scroll
0123456789abcdef       0123456789     <- old live line, reflowed
status line here       abcdef         <- old live line, reflowed
                       0123456789     <- new live region
                       status
```

Ghostty reflowed each old live line onto two rows. `LivePrinter` moved
the cursor up `lastHeight-1` rows, so it redrew over the middle of the
old region and left the top half on screen. Every narrowing of the
window leaves more stale lines in the user's scrollback. If we do
nothing, resize bugs like this one ship, and tests cannot prevent their
return.

## Context

### wonton today

- `termtest.Screen` (`termtest/screen.go`) fixes its size in
  `NewScreen`. It does not record which rows wrapped because text
  overflowed (soft wrap) and which rows ended at a CR LF (hard break).
- The full-screen runtime (`tui/runtime.go`) gets the size from
  `Terminal.Size`. `Terminal.RefreshSize` reads the tty size after
  SIGWINCH, resizes the buffers, forces a full redraw, and calls the
  `OnResize` callbacks. The runtime then queues a `ResizeEvent`. A test
  terminal (`terminal.NewTestTerminal`) has no tty, so `RefreshSize`
  does nothing and no test can change its size.
- `InlineApp` (`tui/inline_app.go`) handles `ResizeEvent` with
  `LivePrinter.SetWidth` and `SetMaxHeight`. `LivePrinter`
  (`tui/print.go`) starts each frame, each `Clear`, and each
  scrollback print with "cursor up `lastHeight-1`, CR". That count
  assumes each old line still occupies one row.
- The existing `InlineApp` resize tests (`TestInlineApp_ProcessResizeEvent`
  and others) check the stored width. They do not check the screen.
- `tmux.Terminal.Resize(width, height)` resizes a live tmux pane. It is
  the name precedent in wonton.

### How terminals resize

We ran 28 resize cases through libghostty-vt (commit 9bfa8ee) with
`ghostty_terminal_resize`. Ghostty, iTerm2, kitty, WezTerm, Alacritty,
Windows Terminal, xterm.js, and tmux reflow the primary screen. xterm
and pyte truncate at the right edge and do not reflow. Ghostty results:

- **Narrower**: a logical line (a hard-broken line and its soft-wrapped
  continuation rows) rewraps at the new width. A hard-broken line that
  is now too long also wraps.
- **Wider**: soft-wrapped rows join again. Hard-broken rows stay apart.
- **Content of a row**: printed cells count, printed spaces too. Erased
  cells with a non-default background count. Erased cells with the
  default background at the end of a row do not count.
- **Wide characters**: a wide character that does not fit at the end of
  a row moves to the next row whole. The row ends with a blank cell.
- **Cursor**: it stays on the same character. A cursor to the right of
  the row's content wraps with the content (column 8 on a 3-character
  line at width 5 moves to row +1, column 3). Pending wrap stays set.
- **Saved cursor** (ESC 7, CSI s): ghostty clamps it to the new size.
  It does not reflow.
- **Fewer rows**: ghostty first removes blank rows below the cursor.
  Then it moves rows from the top into scrollback. A cursor whose row
  moves into scrollback goes to row 0.
- **More rows**: ghostty pulls rows back from scrollback. With no
  scrollback, it adds blank rows at the bottom.
- **Alternate screen, and wraparound mode off (`CSI ?7 l`)**: ghostty
  truncates at the right edge and does not reflow. A wide character
  whose second half falls off the edge becomes blank.

## Users and job

Primary user: a wonton maintainer who tests resize handling in the
full-screen runtime, `InlineApp`, and `LivePrinter`. The job: "show me
what a real terminal displays after the user resizes the window and my
app redraws, so I can catch broken or duplicated lines before release."

Secondary user: a developer who builds an app on wonton and tests it
with termtest.

## Concepts

- **Resize**: the terminal window changes size between two writes. The
  test calls `Screen.Resize`, then writes the app's redraw.
- **Soft-wrapped row**: a row that ended because text reached the right
  edge and continued on the next row. A row that ended at a line feed is
  hard-broken.
- **Reflow**: on resize, the primary screen rewraps soft-wrapped text at
  the new width, as ghostty does. Hard breaks stay. The alternate screen
  truncates instead.

## Use cases

### UC-1: Redraw a full-screen frame after the window narrows

The terminal window changes first, then the program learns the new size.
The test calls the two methods in that order, with the same size:

```go
screen := termtest.NewScreen(80, 24)
term := terminal.NewTestTerminal(80, 24, screen)
rt := tui.NewRuntime(term, app, 60)
// ... the runtime renders the first frame into screen ...

screen.Resize(40, 24)  // 1. the user narrows the window
term.SetSize(40, 24)   // 2. the program learns the new size
// ... the runtime handles the ResizeEvent and redraws ...

termtest.AssertRow(t, screen, 0, "Title")
```

The test runs on whichever screen the runtime writes to. Today
`tui.NewRuntime` does not switch to the alternate screen, so the test
uses the primary screen, which reflows. When the alternate screen lands
(see Decisions), the same test passes on it, which truncates. The
runtime forces a full redraw after a resize, so the criteria check only
the final frame, which is the same under both models.

Acceptance:
- [ ] `Screen.Resize(width, height int)` exists. It returns nothing.
- [ ] After `Resize`, `Size` returns the new size, and `Text`, `Row`,
      `Cell`, and snapshots use it.
- [ ] A full-screen runtime test narrows and widens the screen, and
      shrinks and grows its height. After each redraw, every row equals
      the frame the app draws at the new size. No row from the old frame
      stays visible.
- [ ] A full-screen test cuts a wide character at the new right edge.
      After the redraw, the final frame shows no half character.
- [ ] A full-screen test on the alternate screen narrows the screen.
      Before the redraw, each old row is truncated at the new right edge,
      not reflowed, and a wide character cut at the edge is blank (R-12).
      After the redraw, every row equals the frame the app draws at the
      new size.

### UC-2: Narrow and widen while an `InlineApp` live region shows

A test prints three scrollback lines and shows a live region at 40
columns. One region line is 30 cells wide. One region line is shorter
than 20 cells. It calls `screen.Resize(20, 24)` and sends the app a
`ResizeEvent`. The app redraws.

Acceptance:
- [ ] The live region appears once. No line of the old region stays on
      screen, reflowed or not.
- [ ] The scrollback lines above the region show, reflowed to 20
      columns, in their original order. The scrollback line directly
      above the region is intact.
- [ ] A narrow and a widen before one redraw (coalesced resizes) give
      the same result as a single resize to the final size.
- [ ] The same test widens back to 40 columns. The region appears once.
- [ ] A `LivePrinter` test covers the same narrowing without
      `InlineApp`.

### UC-3: Learn the resize rules from the docs

A developer reads the `Resize` doc comment and the `ExampleScreen_Resize`
test.

Acceptance:
- [ ] The doc comment states reflow, the cursor rule, the wide-character
      rule, and that rows pushed off the top are lost.
- [ ] `ExampleScreen_Resize` narrows a screen with a soft-wrapped line
      and prints the result. CI runs it.
- [ ] The package doc and `termtest/README.md` list resize under the
      supported features.

## Requirements

- R-1: While the primary screen is active, `Resize` reflows the screen.
  Its results equal ghostty's on the rules in "How terminals resize",
  except R-6. R-12 covers the alternate screen.
- R-2: termtest records, for each row, whether it is soft-wrapped. A
  row becomes soft-wrapped when a printed character wraps to the next
  row, including a wide character that does not fit. The flag moves
  with the row when the screen scrolls or inserts or deletes rows. The
  implementation verifies on probes when ghostty clears the flag (for
  example on erase or cursor movement into the row), and follows it.
- R-3: The current style, pending escape bytes, and the deferred-wrap
  state stay across `Resize`. The saved cursor clamps to the new size.
- R-4: Width and height under 1 clamp to 1, as in `NewScreen`. A resize
  to the current size changes nothing.
- R-5: A resize that changes both width and height reflows first, then
  fits the rows to the new height.
- R-6: termtest has no scrollback. Rows that leave the top are lost. A
  later resize to more rows adds blank rows at the bottom. Ghostty would
  bring the lost rows back.
- R-7: `terminal.(*Terminal).SetSize(width, height int)` sets the size
  the program uses. It resizes the buffers, forces a full redraw, and
  calls the `OnResize` callbacks, as `RefreshSize` does after SIGWINCH.
  It shares that code with `RefreshSize`. Its doc says what it does not
  do: it does not resize the terminal window, and on a tty the next
  SIGWINCH replaces the size. Its doc shows the test order: the window
  (`termtest.Screen.Resize`) changes first, then `SetSize`, with the
  same size. `terminal` does not import `termtest`.
- R-10: `LivePrinter` finds the top of its previous frame by counting
  rows, on every frame: each old line occupies
  `max(1, ceil(lineWidth/currentWidth))` rows, where `lineWidth` is the
  line's display width as ghostty measures row content (printed cells
  and styled trailing cells count, default trailing blanks do not). It
  moves up the sum less one. `Clear` and the scrollback print path use
  the same count.
- R-11: On the first frame after a width change, `LivePrinter` writes
  erase-below (`CSI J`) once the cursor is at the top of the old frame,
  and rewrites every line (no line diffing for that frame).
- R-12: While the alternate screen is active, `Resize` truncates each
  row at the new right edge and does not reflow, as ghostty does. A wide
  character whose second half falls off the edge becomes blank. Widening
  adds blank cells on the right. Height changes follow ghostty, verified
  on probes. This requirement depends on the alternate screen from the
  termtest-alt-screen track (#212) and lands after it merges (see
  Decisions).
- R-8: The new resize probes in our copy of the vtdiff harness all
  match ghostty. The existing 100 probes, the 1,002 recorded streams,
  and the fuzz run still match as before.
- R-9: Existing snapshots do not change.

## Not in scope

- Scrollback in termtest. It is a separate concept with its own API.
- A truncate mode that models xterm. One model keeps tests simple. We
  add a mode if a test needs to prove xterm behavior.
- Wraparound mode (`CSI ?7 l`). termtest does not track it today.
- Testing xterm behavior. termtest models only reflow, so no test
  checks what xterm shows after a resize.
- Resize in `Capture` and `Recorder`. They build a screen from a fixed
  size at the end.

## Decisions

- **Reflow, with ghostty as the reference.** Ghostty is the reference in
  our harness, and nearly every modern terminal reflows. Tests then show
  what most users see. The `InlineApp` bug above only shows under
  reflow. Rejected: xterm-style truncation. It is simpler, but it would
  hide that bug.
- **Name it `Resize(width, height int)` with no return value.** It
  matches `tmux.Terminal.Resize` and `NewScreen(width, height)`. A
  resize cannot fail after clamping. Rejected: `SetSize`, which reads
  as a plain setter and hides reflow.
- **No `ScreenOption`.** Resize is a method a test calls during a
  stream, not a setting made at construction. If a later need arises,
  it uses the shared `termtest.ScreenOption` type that the
  termtest-raw-mode track adds in `termtest/options.go`. This track does
  not add a second option type.
- **Clamp the saved cursor; do not reflow it.** Ghostty clamps it.
- **Lose rows that leave the top.** termtest has no scrollback, and
  adding one is a new concept. Tests that grow the height after rows
  left the top differ from ghostty. The doc comment says so.
- **This track owns truncation on resize while the alternate screen is
  active.** Merge order: termtest-alt-screen merges first. This track
  rebases on it and makes `Resize` truncate the alternate screen at the
  right edge, without reflow, as ghostty does (R-12). A wide character
  whose second half falls off the edge becomes blank. Probes verify it.
- **Add `terminal.(*Terminal).SetSize`.** The full-screen runtime gets
  its size only from the terminal. A test terminal has no tty, so
  without this method no test can drive the runtime's resize path.
  Rejected: an internal test hook. Go cannot reach unexported terminal
  code from `tui` tests without an export. Rejected: a helper that
  resizes a screen and a terminal together. It couples `terminal` to
  `termtest`.
- **Fix the `LivePrinter` reflow bug in this track (R-10, R-11).** Count
  rows per old line by that line's own width, on every frame. A count
  by the old frame width moves up too far for short lines and erases a
  scrollback line above the region. A count only after a width decrease
  misses coalesced resizes: the runtime throttles them, and a user can
  narrow and widen before one redraw.
- **Optimize for reflowing terminals (ghostty, iTerm2, kitty, WezTerm,
  Terminal.app).** In xterm, which does not reflow, the cursor moves up
  k rows too far, where k is the number of extra rows the count adds.
  R-11 then erases from there down, so the region is drawn once, and no
  stale region rows stay. The cost: k scrollback rows directly above the
  region are overwritten. termtest models only reflow, so no test checks
  this xterm result. The implementation PR states it under "Risk and
  rollout".
- **No tech spec.** The behavior rules above are the design. The one
  internal choice, where each row's soft-wrap flag lives, is local to
  termtest and is covered by R-2 and the probes.

## Success

- A `tui` test fails on today's `LivePrinter` resize redraw and passes
  after the fix, including the short-line and scrollback-line checks.
- The new resize probes match ghostty.
- No existing termtest, `tui`, or `cli` test or snapshot changes.

## Risks and open questions

- **Merge conflicts on row moves.** The soft-wrap flag must move with
  rows in scroll, IL, DL, and the scroll regions that the scroll-regions
  track adds. Mitigation: one helper moves a row and its flag. New logic
  goes in `termtest/resize.go`. Edits to `screen.go` and `ansi.go` stay
  small. We rebase after each termtest track merges and add probes for
  wrapped rows inside scroll regions.
- **Row content needs more than the cell value.** A printed space and an
  erased cell are the same `Cell` today. Reflow must tell them apart
  (see "Content of a row"). Mitigation: track the content extent of
  each row in private state. Probes cover printed trailing spaces and
  background-colored erases. The BCE change in the scroll-regions track
  adds background-colored erases; we rebase on it and rerun the probes.
- **Ghostty details we have not probed.** For example, when an erase
  clears the soft-wrap flag. The probes decide each case. None blocks
  the build.
