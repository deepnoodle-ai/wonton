# Give an interactive child sole terminal ownership

Status: Draft\
PRD PR: Not opened · Implementation PR: Not opened · Updated: 2026-09-30

## Problem and context

A Go developer needs to open a terminal editor from a fullscreen or inline Wonton application, then continue editing the same draft. Dashi's composer requires this workflow. Today, `Runtime.Suspend` retains stdin and sends decoded events to its callback. `InlineApp` has no suspension API. Both runtimes leave a nested decoder goroutine reading input. Launching an editor through either path races for keystrokes and can deliver editor input to the application after return.

[Bubble Tea](https://github.com/charmbracelet/bubbletea/blob/main/tea.go) cancels and waits for its input loop before releasing the terminal. [tcell](https://github.com/gdamore/tcell/blob/main/screen.go) suspends input/output and restores startup terminal settings. Wonton needs the same ownership guarantee, with separate operation and restoration results. Keep `Suspend` for decoded-key transcript viewing.

## Users and concepts

The primary user is a developer using `Runtime` or `InlineApp`. Their terminal user opens an interactive child and returns to the existing application. A **handoff** temporarily transfers terminal input and output to a synchronous callback. The callback starts and waits for its child. Wonton retains application state and restores terminal ownership before returning.

## Use cases and acceptance

### UC-1: Open an editor and return

Call the matching method from `HandleEvent`, between frames:

```go
operationErr, restoreErr := runner.Handoff(func() error {
    cmd := exec.Command("vi", draftPath)
    cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
    return cmd.Run()
})
// Handle restoreErr first: unsafe restoration ends the runner.
// Handle operationErr without replacing the application's draft.
```

Acceptance:

- [ ] Both runners export `Handoff(fn func() error) (operationErr, restoreErr error)`. A nil callback is a no-op. Callers use the event-loop goroutine. Nested handoffs, including overlap with `Runtime.Suspend`, reject without calling the callback.
- [ ] The callback starts only after Wonton acknowledges that no terminal read is outstanding. The child receives normal terminal input/output. Application input dispatch and rendering remain suspended throughout the callback.
- [ ] Wonton retains the same decoder, its partial/buffered bytes, and already-owned events in order. Unread terminal bytes at release join application-owned storage and cannot reach the child. Input after release belongs to the child, including the launch interval.
- [ ] On callback return, Wonton discards residual child input before resuming its reader. Child input cannot complete an application escape, paste, or UTF-8 sequence. Pre-handoff application input returns exactly once.
- [ ] Successful child exit, cancellation, and launch/nonzero errors restore prior raw mode, alternate-screen mode, mouse mode, bracketed paste, enhanced keyboard, and cursor visibility. A full repaint uses the current terminal size. Inline scrollback persists and its live region returns without ghost rows.
- [ ] Inline `Print`, `Printf`, and `PrintRaw` cannot write or redraw during the callback. Concurrent calls wait until restoration completes, then print normally. The callback must not call these methods.

### UC-2: Refuse an unsafe handoff

Call handoff with unsupported input, or encounter a release/restoration failure.

Acceptance:

- [ ] Initial support covers managed terminal stdin on macOS and Linux. Custom `InputSource`/inline readers, redirected input, Windows, and other unqualified platforms return `ErrHandoffUnsupported` before the callback or terminal mutation. Existing custom-input use remains supported outside handoff.
- [ ] A runner that is not running returns `ErrHandoffNotRunning`. Reentry returns `ErrHandoffReentrant`. Errors name the failed phase and preserve underlying causes for `errors.Is`/`errors.As`.
- [ ] Qualification/release failures and the callback result use `operationErr`. Restoration failures use `restoreErr` independently, including when the callback also fails. Failed release attempts restoration after any mutation and never start the callback.
- [ ] Restoration, input-resume, or repaint failure stops input admission and rendering. `Run` returns that failure after best-effort terminal cleanup. The runner never continues interactively with uncertain ownership.
- [ ] A callback panic attempts restoration before the existing runtime panic path propagates it. Stopping during the callback waits for callback return and restores a plain terminal without restarting application input.

### UC-3: Use Ctrl-D as forward delete

- [ ] Focused `InputField` consumes Ctrl-D for nonempty text, using existing Delete behavior: one forward grapheme or existing atomic special segment. At the end it consumes the key without changing text.
- [ ] Empty input returns Ctrl-D to the application for its exit policy. `OnKey` retains precedence. Deletion updates the binding and change callback once; a no-op does not report a text change. Fullscreen and inline routing agree.

## Decisions and boundaries

Use a callback API to pair release and restoration, rather than public Pause/Resume methods that callers can leave unbalanced. Keep one internal pauseable reader shared by both runners. Reject another stdin reader and decoder replacement because either loses sole ownership or buffered application input. Define release as the ownership boundary, rather than child process startup, so launch latency has one explicit owner.

Qualify macOS/Linux first. Reject generic-reader cancellation claims without a low-level acknowledgment. Keep separate errors because child failure can be recoverable while restoration failure is fatal. Reuse existing Delete semantics and hooks rather than adding a composer widget.

Editor selection, temporary files, draft replacement, subprocess cancellation, background execution, configurable keymaps, public input-owner interfaces, and a general input framework are outside this track. The [technical spec](../design/terminal-handoff.md) defines the reader and restoration contract.

## Success and risk

Success requires public `Example*` coverage, repeated real PTY handoffs in both runners, and an interactive editor recording at 80×24. A simulated editor alone cannot prove keyboard ownership. Preserve `Suspend`, custom-reader, Unicode editing, paste, keyboard, mouse, panic, and shutdown behavior. Independent implementation and acceptance reviews must exercise the real child workflow.

The main risk is pausing while a decoder holds a partial sequence or an OS read is active. PTY tests must prove acknowledged quiescence and both input boundaries before release. No product decision remains open; platform qualification is an implementation gate.
