# Test full-screen apps that leave the alternate screen

Status: Draft\
PRD PR: [#207](https://github.com/deepnoodle-ai/wonton/pull/207) · Implementation PR: TBD · Updated: 2026-10-31

## Problem

A test author tests a full-screen `tui` app. The app runs on the alternate
screen, as `tui.Run` does by default (`tui/run.go:202`). When the app quits,
a real terminal shows the shell screen again: the prompt and the output from
before the app started. `Runtime.Suspend` does the same thing for a short
time, and then returns to the app frame (`tui/runtime_suspend.go:61-100`).
`Runtime.Handoff` does it too, around a child process such as an editor
(`terminal/handoff_state.go:74`, `:119`, `:126`).

`termtest.Screen` ignores the switch. It has one screen. A test that writes
an app's output into a `Screen`, quits the app, and reads `Text()` sees the
app's last frame, not the restored shell screen. The test cannot check that
the app gives the screen back. It cannot check that suspend shows the shell
screen and that resume redraws the app.

Evidence (shared findings, 2026-10-31): the stream
`primary ESC[?1049h ESC[2J ALT ESC[?1049l` shows `primary` in ghostty and
`ALT` in termtest. This is the only alternate-screen gap the harness found,
and it hides a whole class of tui behavior from tests. Today the suspend
tests only check that the bytes `ESC[?1049l` and `ESC[?1049h` occur in the
output (`tui/runtime_suspend_test.go:37-38`), not what the user sees.

If we do nothing, exit and suspend bugs (a lost prompt, a blank screen after
resume, a frame left on the shell screen) stay invisible to wonton's tests
and to the tests of apps built on wonton.

## Context

### Our code today

- `termtest/ansi.go:327` drops every `CSI h` and `CSI l`. No private mode is
  tracked.
- `Screen` holds one cell grid, one cursor, one saved cursor (`ESC 7`/`ESC 8`
  and `CSI s`/`CSI u` share it), and the current style
  (`termtest/screen.go:133-145`). `ESC c` calls `Reset()`.
- `terminal.Terminal` writes `ESC[?1049h` and `ESC[?1049l` only
  (`terminal/terminal.go:1071`, `:1293`). It never writes `?47` or `?1047`.
  `Close` leaves the alternate screen (`terminal/terminal.go:2513`).
  `Terminal.IsAlternateScreen()` reports the mode on the writer side.
- `tui.Run` always builds its terminal with `NewTerminal()`
  (`tui/run.go:194`), on the process's stdout. A test cannot point `tui.Run`
  at a `Screen`. A test can do what `tui.Run` does by hand: `NewRuntime` on
  a `NewTestTerminal` that writes into the `Screen`, a scripted
  `InputSource`, `Run()`, then `Terminal.Close()`. The existing input tests
  use this pattern (`tui/input_simulation_test.go`).
- `Runtime.Handoff` works only on a real stdin and stdout on one terminal
  (`tui/handoff.go:79`). With a test terminal it returns
  `ErrHandoffUnsupported`. Its tests run a child under a PTY
  (`tui/handoff_pty_unix_test.go`) and check the escape sequences in the
  captured output.
- Tests that feed `?1049` into a `Screen` today: one,
  `TestANSISetResetMode` (`termtest/ansi_test.go:877`). It writes `Hello`,
  then `?1049h`, then `?1049l`, and expects `Hello` on row 0. With the new
  behavior it still passes: the primary screen is restored. No test feeds
  `?47` or `?1047`. No `tui` test sends a `Terminal` with the alternate
  screen into a `Screen`; the suspend tests write to a `bytes.Buffer`. No
  testdata file contains these sequences. So we expect zero changed results.
  The implementation confirms this with the full test run and the vtdiff
  corpus of 1,002 recorded streams.

### Reference behavior: ghostty probes

I ran 43 probes through libghostty-vt (commit 9bfa8ee, 10x3 screen). The
probes are in the notes copy of vtdiff (`altprobe.go`). They agree with the
xterm control-sequence documentation where it is specific. Results:

| Sequence | On set (`h`) | On reset (`l`) |
|---|---|---|
| `?1049` | Save cursor (as DECSC, into the current screen's slot, so the primary slot in normal use), switch to alternate, clear it. The cursor stays where it was. | Switch to primary, restore cursor from the primary slot (as DECRC: position, style, pending wrap). Alternate content is discarded. |
| `?1047` | Switch to alternate. No clear on entry. Cursor carries over. | Clear the alternate screen, then switch to primary. Cursor carries over. |
| `?47` | Switch to alternate. No clear. Cursor carries over. | Switch to primary. No clear: the alternate content stays for the next `?47h`. Cursor carries over. |

Edge cases from the same probes:

- `?1049h` while already on the alternate screen saves the cursor again and
  clears the alternate screen again.
- `?1049l` while on the primary screen still restores the saved cursor.
- `?1047l` on the primary screen clears nothing.
- `ESC 7` and `ESC 8` (DECSC and DECRC) save and restore the pending wrap
  too. `CSI s` and `CSI u` save and restore the same state as DECSC and
  DECRC, style and pending wrap included. termtest does neither today; this
  track adds both, because `?1049` uses the same save and the two save
  commands share one slot.
- Mixed pairs work per the table: `?1049h` then `?47l` switches back without
  a cursor restore. `?47h` then `?1049l` switches back and restores the
  primary slot.
- Each screen has its own saved cursor. `ESC 7` on the alternate screen does
  not change the primary screen's slot. The alternate slot starts empty, so
  `ESC 8` there goes home.
- The cursor position, the current style, and the pending-wrap state are
  shared by the two screens. A `?47` round trip keeps a pending wrap. A
  `?1047l` loses it, because the clear runs first.
- `ESC c` (RIS) returns to the primary screen and clears it.
- Multiple parameters work: `CSI ? 25 ; 1049 h`.
- Scrolling on the alternate screen does not touch the primary screen.

### Comparable libraries

- **xterm.js**: `terminal.buffer.active` is the visible buffer;
  `buffer.normal` and `buffer.alternate` expose each one, and
  `buffer.active.type` says which is active. Tests read the active buffer
  by default.
- **hinshun/vt10x** (Go): swaps line buffers on 1049/1047/47 and exposes the
  mode as the `ModeAltScreen` bit of `Mode()`. Reads return the visible
  screen.
- **pyte** (Python): no alternate screen, so it has the same gap as
  termtest today.

(From the libraries' public source and docs, not run in this track.)

Every library that models it returns the visible screen from its normal
read methods and offers a mode query. Only xterm.js exposes the hidden
buffer, because it also draws scrollback and selection.

## Users and job

Primary user: a Go developer who writes tests for a full-screen `tui` app,
inside wonton or in an app built on it. The job: prove that the app shows
the right frame while it runs, and gives the user's screen back when it
quits or suspends. Until `tui.Run` accepts a test terminal (see Not in
scope), an app author builds the `Runtime` by hand as UC-1 shows.

Secondary user: wonton maintainers, who use termtest with the vtdiff
harness to check that termtest matches a real terminal.

## Concepts

- **Alternate screen**: a second screen that a full-screen app draws on.
  When the app leaves it, the terminal shows the primary screen again,
  unchanged. Same term as `terminal.Terminal.EnableAlternateScreen`.
- **Primary screen**: the normal screen, with the shell prompt and earlier
  output. termtest has used only this screen until now.

`Screen` stays one object. It shows one of the two screens at a time, like a
terminal window.

## Use cases

### UC-1: The shell prompt comes back after the app quits

A test runs a real app the way `tui.Run` does, with a `Screen` in place of
the user's terminal:

```go
screen := termtest.NewScreen(80, 24)
screen.Write([]byte("$ myapp\r\n"))
term := tui.NewTestTerminal(80, 24, screen)
term.EnableAlternateScreen()
term.HideCursor()
rt := tui.NewRuntime(term, app, 30)
rt.SetInputSource(keys) // scripted: waits for the frame, then sends 'q'
err := rt.Run()         // the app renders "Inbox (3)", then quits
term.Close()            // the cleanup that tui.Run defers; writes ESC[?1049l
termtest.AssertRow(t, screen, 0, "$ myapp")
termtest.AssertNotContains(t, screen, "Inbox (3)")
termtest.AssertCursor(t, screen, 0, 1)
```

Acceptance:
- [ ] After `?1049l`, `Text()`, `Row()`, `Cell()`, and `Contains()` show the
      primary content written before `?1049h`.
- [ ] After `?1049l`, `Cursor()` is the position saved at `?1049h`.
- [ ] `screen.IsAlternateScreen()` is `false` after `?1049l`.
- [ ] A `tui` test runs a `Runtime` with scripted input to a quit, closes
      the terminal, and passes these checks.

### UC-2: Check the app frame while it runs

```go
term.EnableAlternateScreen()
// ... render ...
assert.True(t, screen.IsAlternateScreen())
termtest.AssertRow(t, screen, 0, "Inbox (3)")
termtest.AssertScreen(t, screen) // snapshot of the app frame
```

Acceptance:
- [ ] While the alternate screen is active, every read method, every
      `Assert*` and `Require*` helper, and `AssertScreen` snapshots show the
      alternate screen only. No primary content shows through.
- [ ] `Equal` and `EqualStyled` compare the visible screens.

### UC-3: Suspend shows the shell screen, resume redraws the app

A `tui` test runs a `Runtime` as in UC-1. A key makes the app call
`Suspend`. The callback writes `transcript line` to the same writer, as a
real callback writes to the user's terminal. Then the app quits and the
test closes the terminal.

Acceptance:
- [ ] Inside the callback: `IsAlternateScreen()` is `false`, the prompt
      written before the app started is visible, and no app frame text is
      visible. The callback's line is visible after it writes it.
- [ ] After `Suspend` returns: `IsAlternateScreen()` is `true`, the full app
      frame is visible (the forced redraw at `tui/runtime_suspend.go:119`
      works), and the callback's line is not visible.
- [ ] After the final exit: the prompt and the callback's line are both
      visible, and no app frame text is visible.

## Requirements

- R-1: `Screen` processes `CSI ? 1049 h/l`, `CSI ? 1047 h/l`, and
  `CSI ? 47 h/l` as the table and edge cases in Context describe.
- R-2: Each parameter of a private-mode sequence is processed, so
  `CSI ? 25 ; 1049 h` switches screens. Other private modes stay ignored.
  `CSI 47 h` without `?` is an ANSI mode, not the alternate screen; it is
  ignored.
- R-3: All existing read and write methods (`Text`, `Row`, `Cell`,
  `CellGlyph`, `Contains`, `Cursor`, `SetCell`, `SetCursor`, `Clear`,
  `WriteString`, and the rest) act on the visible screen.
- R-4: New method `func (s *Screen) IsAlternateScreen() bool`. It reports
  whether the alternate screen is visible. The zero state, after
  `NewScreen` and after `Reset`, is `false`.
- R-5: `Reset()` and `ESC c` return to the primary screen, clear both
  screens, and clear both saved cursors.
- R-6: Each screen has its own saved cursor for `ESC 7`/`ESC 8` and
  `CSI s`/`CSI u`. `?1049h` saves into the slot of the visible screen,
  which is the primary screen in normal use. `?1049l` restores from the
  primary screen's slot.
- R-7: The vtdiff probe set for these sequences matches ghostty: text,
  cursor, and style per cell.
- R-8: `tui` gets tests for UC-1 (exit restores the primary screen) and
  UC-3 (suspend and resume). Each runs a real `Runtime` with scripted input
  and a `Screen` as the terminal's writer.
- R-10: The handoff PTY test replays its captured output into a `Screen`
  and checks that every scenario ends on the primary screen, with the
  child's echoed input visible.
- R-11: A benchmark writes a full-screen frame stream to a `Screen` that
  never switches.
- R-9: An `Example` test shows `IsAlternateScreen` and the restored prompt.
  The termtest README and package doc list alternate-screen support.

## Not in scope

- A way to run `tui.Run` against an in-memory terminal or a `Screen`. It is
  a tui API decision, and the termtest change has value without it:
  [#206](https://github.com/deepnoodle-ai/wonton/issues/206).
- A method to read the hidden screen. See Decisions.
- `?1048` (save and restore cursor alone). wonton never sends it; no user
  asked.
- Scrollback for either screen. termtest has no scrollback today.
- Tracking other private modes (cursor visibility, mouse, bracketed paste).
  They do not change what the screen shows.
- Interaction with scroll regions and resize. Those are separate tracks.
  If one of them merges first, this track adds probes for margins or resize
  across a switch while it rebases.

## Decisions

- **Every read shows the visible screen.** A real terminal shows one
  screen, and the test author asserts what the user sees. Rejected: reads
  that always show the primary screen (hides the app frame), and a combined
  view (no terminal does that).
- **One new method, `IsAlternateScreen() bool`.** The name matches
  `terminal.Terminal.IsAlternateScreen`, so a test reads the same word on
  both sides. Rejected: a general `Mode(n int) bool` query. No second
  private mode needs it, and it exposes numbers instead of a concept.
- **No method to read the hidden screen.** The user's job ends with a
  switch back: after exit or inside a suspend callback, the primary screen
  is the visible one. A hidden-screen read tests something the user never
  sees. It is additive, so we can add it if a real test needs it. Rejected:
  an xterm.js-style `Primary() *Screen` view.
- **Ghostty is the reference where xterm's documentation is not specific.**
  The brief names ghostty, the harness already runs it, and the probes agree
  with xterm's documented behavior. Rejected: guessing from documentation.
- **Saved cursor per screen, and `?1049h` uses the visible screen's slot.**
  This is what ghostty does (probes "DECSC per screen", "1049 save shares
  DECSC slot", and "1049h from alt via 47"). An app that uses `ESC 7` on the alternate screen must not move the
  shell's cursor on exit.
- **No option to turn the support off.** The old behavior was wrong, and an
  option keeps the wrong behavior alive for nobody. A test that read the
  last frame after exit was reading something no user sees; it must assert
  before exit instead. Rejected: a `NewScreen` option for the old behavior.
- **The handoff check reuses the PTY test.** Handoff refuses a test
  terminal, so a `Screen`-only handoff test is not possible without a PTY.
  The PTY test already captures every byte; replaying it into a `Screen`
  costs a few lines. Rejected: a new handoff test harness.
- **No `ScreenOption` for this track.** The raw-mode track adds the shared
  `termtest.ScreenOption` type in `termtest/options.go`. This track needs no
  option (see the decision above), so it adds none and does not create a
  second option type.
- **No tech spec.** The change is local to termtest: a second grid and a
  per-screen saved cursor, behind one dispatch point. New code goes in
  `termtest/altscreen.go`, with small edits to `ansi.go` and `screen.go`, as
  the shared track rules ask.
- **A tui bug found by the new tests is fixed here only if the fix is
  small**, per the brief. A larger one is recorded and sent to the user.

## Success

- The UC-1 and UC-3 tests exist in `tui` and pass.
- The vtdiff alternate-screen probes match ghostty.
- No existing test changes its result. The 1,002-stream vtdiff corpus,
  the renderer comparison, and the fuzz run match as before.
- Must not get worse: `Write` throughput on screens that never switch, as
  the R-11 benchmark measures before and after (within run-to-run noise),
  and zero panics in the fuzz run.

## Risks and open questions

- **A user test depends on the old behavior.** A test that captures the
  full output of a `tui.Run` session and asserts the final frame now sees
  the primary screen. We reduce this by naming it under "Risk and rollout"
  in the PR, with the fix: assert before exit. Not blocking.
- **Merge conflicts with the scroll-regions and resize tracks.** Both touch
  the cell grid. New logic stays in a new file, and the per-screen state is
  one small struct, so a rebase is local. Not blocking.
- **The two screens must stay the same size.** termtest has no resize
  today. When the resize track lands, both screens must resize. Recorded
  for that rebase. Not blocking.
