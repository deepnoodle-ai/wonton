# Test output that goes to a terminal in raw mode
Status: Draft
PRD PR: #204  Implementation PR: #TBD  Updated: 2026-10-31

## Problem

`termtest.Screen` always treats a bare LF (`\n`) as CR+LF. That is what a
terminal shows when the tty is in cooked mode, because the tty driver adds
the CR (the ONLCR output setting). In raw mode the tty sends the bytes
unchanged. A bare LF then moves the cursor down one row and keeps the
column.

wonton writes most interactive output in raw mode: `InlineApp`, the
full-screen runtime, and `LivePrinter` or `Print` with `WithRawMode(true)`.
The code adds a CR by hand at each place that needs one. termtest cannot
see a missing CR, so a test of raw-mode output passes when the CR is
missing. A real terminal then shows text that steps to the right, or an
old line that is only partly erased.

Evidence, from a comparison of termtest with libghostty-vt (ghostty with
ONLCR off is the raw-mode reference):

- `ab\ncd` in raw mode: ghostty puts `cd` at column 2 of row 1. termtest
  puts it at column 0.
- A `LivePrinter` with `WithRawMode(true)` drew six frames. The content
  matched in every frame. In two frames the final cursor column differed
  from termtest. termtest cannot express the raw-mode result at all.
- No wonton test checks raw-mode output under raw-mode rules today.

If we do nothing, a refactor that drops one `\r` from a raw-mode path
passes CI and ships.

## Context

What termtest has today:

- `NewScreen(width, height)` is the only constructor. It takes no
  options. `Capture.Screen`, `Buffer.Screen`, `NewRecorder`, and
  `Recorder.Replay` each call `NewScreen(width, height)`.
- `writeCluster` in `termtest/screen.go` sends both `"\n"` and `"\r\n"` to
  one case that sets the column to 0 and calls `lineFeed`. VT and FF
  already move down and keep the column, which is correct in both modes.
- There is no option type in termtest. The tui package uses functional
  options (`PrintOption`, `tui.WithRawMode(enabled bool)`).

What the raw-mode producers do:

- `LivePrinter.writeFrame` writes a bare `\n` between lines in both
  modes. Each changed line starts with `\r\033[2K`, so content is correct.
  The shrink path writes `\n\r` before `ESC[0J`, so the erase starts at
  column 0. Both CRs matter only in raw mode.
- `LivePrinter.Stop` writes a bare `\n` (`fmt.Fprintln`) in both modes.
- `InlineApp.Print` uses `WithRawMode(true)` and writes `\r\n`. Its
  cleanup writes `\r\n`. Unit tests drive it without a tty through
  `newPrintTestRunner` in `tui/inline_app_print_test.go`.
- The full-screen renderer in `terminal/terminal.go` positions the cursor
  with CUP. Its unbuffered print path (`printUnbufferedLocked`) sends
  each newline as `\r\n` on purpose, with a comment about raw mode.

How comparable tools do the job:

- libghostty-vt and xterm do not translate LF. ONLCR is a tty driver
  setting, not a terminal setting. A test harness that feeds bytes to the
  terminal emulator directly gets raw-mode behavior.
- Python `pyte` treats LF as a pure line feed by default. LNM mode
  (`CSI 20 h`) makes LF also return the carriage.
- `vt10x` (Go) treats LF as a pure line feed. Callers who feed it cooked
  output must add the CR.

So termtest is the outlier. Its default is right for its main use, which
is `Sprint` output and plain CLI output with `\n` line endings. Raw mode
must be an option, not a new default.

What test authors do now: they assert on the raw bytes, such as
`strings.Contains(buf.String(), "\r\n")`, or they do not test raw-mode
output.

## Users and job

Primary user: a wonton maintainer or app author who tests code that
writes to a terminal in raw mode. The job: make the test see what a real
raw-mode terminal shows, so a missing CR fails the test.

Secondary user: a test author with cooked output. This user must see no
change.

## Concepts

- **Raw mode**: the tty sends the program's bytes to the terminal
  unchanged. A bare LF moves the cursor down one row and keeps the
  column. Only CR moves the cursor to column 0. It is the same term as
  `tui.WithRawMode` and the raw mode that `InlineApp` turns on.
- **Cooked mode**: the default. The tty adds a CR before each LF, so a
  bare LF also moves to column 0. termtest models this today and keeps
  it as the default.

## Use cases

### UC-1: Model a raw-mode terminal

```go
screen := termtest.NewScreen(80, 24, termtest.WithRawMode(true))
screen.Write([]byte("ab\ncd"))
termtest.AssertRow(t, screen, 1, "  cd")
termtest.AssertCursor(t, screen, 4, 1)
```

Acceptance:
- [ ] `termtest.WithRawMode(true)` exists and returns a `ScreenOption`.
- [ ] With it, a bare LF moves down one row and keeps the column. At the
      last row it scrolls, as today.
- [ ] `\r\n`, CR, VT, FF, NEL (`ESC E`), and IND (`ESC D`) behave the
      same in both modes.
- [ ] A `Screen` made without the option, or with `WithRawMode(false)`,
      behaves exactly as today. No existing test or snapshot changes.
- [ ] `Screen.Reset` (and `ESC c`) keeps the mode. Raw mode is a tty
      setting, and a terminal reset does not change the tty.
- [ ] The doc comment says what changes: a bare LF keeps the cursor
      column.
- [ ] An `ExampleWithRawMode` test shows cooked and raw results for the
      same bytes, with `// Output:`.

### UC-2: Capture and replay raw-mode output

```go
capture := termtest.NewCapture(nil)
app.Run(capture)
screen := capture.Screen(80, 24, termtest.WithRawMode(true))
```

Acceptance:
- [ ] `Capture.Screen`, `Buffer.Screen`, `NewRecorder`, and
      `Recorder.Replay` accept `...ScreenOption` and pass the options to
      the new screen.
- [ ] `Recorder.Replay` applies the options given to `NewRecorder`
      first, then its own options. A replay of a raw-mode recorder stays
      in raw mode unless the call passes `WithRawMode(false)`.
- [ ] `tui.SprintScreen` builds its screen with
      `termtest.WithRawMode(cfg.RawMode)`, so the screen matches the
      line endings that the print options chose.
- [ ] A test calls `SprintScreen` with `WithRawMode(true)` on a
      multi-line view. Each line starts at column 0. The test fails if
      `renderToANSI` drops the CR from its raw-mode line ending.

### UC-3: Catch a missing CR in tui raw-mode output

A maintainer removes a `\r` from a raw-mode path in tui or terminal. A
test fails.

Acceptance:
- [ ] `LivePrinter` output on a raw-mode screen, with the default
      `LivePrinter` options, as `InlineApp` uses it: tests cover a frame
      that grows, a frame that shrinks, a frame where only some lines
      change, and `Clear`. Each test checks every row and the cursor row.
- [ ] `LivePrinter.Stop` leaves the cursor at column 0 of the row below
      the region on a raw-mode screen (see Decisions; this is a fix).
- [ ] `InlineApp`: a raw-mode test covers `Print` of a multi-line view
      with a live region, and the cleanup when the app exits. Scrollback
      text and the redrawn live region are on the correct rows at column
      0.
- [ ] Full-screen output: a raw-mode test covers the multi-line
      unbuffered print path in `terminal`.
- [ ] CR inventory. Each CR below is covered as the table says. For each
      covered CR, the implementation PR shows that its removal makes the
      named test fail.

| CR | Where | Covered by |
| --- | --- | --- |
| `\r` in each `\r\033[2K` | `LivePrinter.writeFrame` | grow and partial-change tests |
| `\r` in `\n\r` before `ESC[0J` | `LivePrinter.writeFrame`, shrink | shrink test |
| `\r` before `ESC[0J` | `LivePrinter.Clear` | Clear test |
| `\r` in `\r\n` (new) | `LivePrinter.Stop` | Stop test |
| `\r` in the raw `\r\n` | `renderToANSI` (`Print`, `Sprint`) | `SprintScreen` and `InlineApp.Print` tests |
| `\r` in the raw `\r\n` | `printlnWith` (`Println`) | `Fprintln` test |
| `\r` in `\r\n` | `InlineApp` cleanup | cleanup test |
| `\r` in `\r\n` | `terminal` `printUnbufferedLocked` | unbuffered print test |
| `\r` after `ESC[nA` | `LivePrinter.beginFrame` | none: redundant |
| `\r` in `\r\n` after content | `InlineApp.Print` | none: redundant |

  The two redundant CRs cannot make any test fail. After `beginFrame`,
  each line that `writeFrame` draws starts with its own `\r`, and the
  shrink path has its own. After `InlineApp.Print`, the live region
  redraw starts with `\r\033[2K`. Both CRs stay, as defense against a
  later change.
- [ ] The vtdiff harness (`./vtdiff live` and the `lf` probe), run with
      termtest in raw mode against ghostty without ONLCR, shows no
      content diff and no cursor diff.

## Requirements

- R-1: `termtest.ScreenOption` is a new exported type for options to a
  new `Screen`.
- R-2: `termtest.WithRawMode(enabled bool) ScreenOption` sets raw mode.
- R-3: `NewScreen(width, height int, opts ...ScreenOption)`. Calls with
  two arguments compile and behave as today.
- R-4: In raw mode, `Write` and `WriteString` treat a bare `\n` as a pure
  line feed. All other bytes behave as in cooked mode.
- R-5: The termtest package doc and README name the option and say when
  to use it.

## Not in scope

- A setter to switch modes in the middle of a stream. Each test today
  writes in one mode. Add it when a test needs it.
- Other tty output settings (OCRNL, ONOCR, tab expansion). wonton does
  not use them.
- LNM mode (`CSI 20 h`). wonton does not send it. ghostty and xterm
  support it, but it is a terminal mode, not a tty setting.
- `renderToANSILive` in `tui/print.go`. No code calls it. Removing it is
  a separate cleanup.
- Tests of the full-screen tui runtime's diff renderer bytes. It moves
  the cursor with CUP, not LF. The renderer-bytes-test track owns tests
  that feed live renderer bytes to termtest.

## Decisions

- **Name the option `WithRawMode(enabled bool)`.** It is the same name
  and signature as `tui.WithRawMode`, so a test can pass one bool to
  both. Rejected: `RawMode()` with no argument, which cannot take a
  table-driven bool. Rejected: `WithONLCR(false)`, which uses a tty term
  that most test authors do not know.
- **Add variadic options to `NewScreen` and the four capture helpers.**
  One constructor stays the entry point, and later options fit the same
  shape. Rejected: `NewRawScreen`, which needs a raw twin of each capture
  helper. Rejected: a `SetRawMode` method, which lets a test change the
  mode after bytes arrived and gives a result that no terminal shows.
- **`Recorder.Replay` reuses the recorder's options.** A replay shows the
  same bytes at a new size, so it must keep the mode by default. Options
  passed to `Replay` come last and can override. Rejected: a replay with
  only the options passed to `Replay`, which silently drops raw mode.
- **Keep cooked mode as the default.** Most termtest input is `Sprint`
  output and CLI output with `\n` endings. A raw default would break
  those tests and every snapshot.
- **The cursor column at the end of a `LivePrinter` frame is not a bug.**
  ghostty showed it: when the last lines do not change, raw mode leaves
  the cursor at the end of the last changed line, not column 0. The
  cursor is hidden during the live region. The next `Update`, `Clear`,
  and `InlineApp.Print` all start with `\r`. Cooked mode also has no
  fixed column, because a changed last line leaves the cursor at its
  end. Tests assert the cursor row and the content, not this column.
  Rejected: add a `\r` at the end of each frame. It costs bytes on every
  frame and fixes nothing that a user sees.
- **`LivePrinter.Stop` always writes `\r\n`.** Today it writes a bare
  `\n`, so in raw mode the cursor lands below the region at the old
  column. The next write then starts in the middle of the line.
  `LivePrinter` ignores `RawMode` everywhere else, and `InlineApp` makes
  its printer without `WithRawMode` but writes to a raw tty. In cooked
  mode `\r\n` gives the same result. Rejected: write `\r\n` only with
  `WithRawMode(true)`, because users would need a flag with no other
  effect. Rejected: defer, because the fix is one line.
- **Test `LivePrinter` with its default options.** That is how
  `InlineApp` uses it. Rejected: test with `WithRawMode(true)`, which no
  wonton caller uses for `LivePrinter`.
- **This track owns the shared option type.** `termtest/options.go`
  holds only `ScreenOption`, the code that applies options, and
  `WithRawMode`. Other termtest tracks add their options to the same
  type. The edit to `termtest/screen.go` is a field, the constructor
  signature, and a split of one `case`. Rejected: each track adds its
  own type and the first to merge wins, because two open PRs could add
  two types.
- **No tech spec.** The change is one option and one branch in
  `writeCluster`. The PRD names every touched API.

## Success

- A test can model a raw-mode terminal with one option.
- Removing any covered CR in the UC-3 inventory fails a named test. The
  two redundant CRs are the only CRs with no test.
- No existing termtest, tui, or cli test or snapshot changes.

## Risks and open questions

- Adding a variadic parameter to `NewScreen`, `NewRecorder`,
  `Capture.Screen`, `Buffer.Screen`, and `Recorder.Replay` breaks code
  that stores them as a function value with the old type. This is rare
  in test code. The PR states it under "Risk and rollout". Not blocking.
- Other tracks (resize, alt screen, scroll regions) also edit
  `termtest/screen.go`. They use `ScreenOption` from
  `termtest/options.go`, as the coordinator agreed. Not blocking.
