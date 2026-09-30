# Give an interactive child sole terminal ownership

Status: Approved\
PRD PR: [#63](https://github.com/deepnoodle-ai/wonton/pull/63) · Implementation PR: Not opened · Updated: 2026-09-30

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
    cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stdout
    return cmd.Run()
})
// Handle restoreErr first: unsafe restoration ends the runner.
// Handle operationErr without replacing the application's draft.
```

Acceptance:

- [ ] Both runners export `Handoff(fn func() error) (operationErr, restoreErr error)`. A nil callback is a no-op. Callers use the event-loop goroutine. Nested handoffs, including overlap with `Runtime.Suspend`, reject without calling the callback.
- [ ] The callback starts only after Wonton acknowledges that no terminal read is outstanding. The child receives normal terminal input/output. Application input dispatch and rendering remain suspended throughout the callback.
- [ ] Before release, Wonton captures unread terminal bytes and finalizes that finite application input stream. It retains complete events in order, exactly once. Incomplete UTF-8/control suffixes are discarded. An unterminated paste retains its complete UTF-8 payload as one paste event, excluding incomplete terminator framing.
- [ ] The same decoder releases all incomplete framing before the callback. Input after release belongs to the child, including launch delay. On return, Wonton discards residual child input and resumes in neutral parsing state. Missing UTF-8, CSI, or paste suffixes never consume resumed typing, Enter, or the application quit key.
- [ ] Successful child exit, cancellation, and launch/nonzero errors restore prior raw mode, alternate-screen mode, mouse mode, bracketed paste, enhanced keyboard, and cursor visibility. A full repaint uses the current terminal size. Inline scrollback persists and its live region returns without ghost rows.
- [ ] Inline `Print`, `Printf`, and `PrintRaw` cannot write or redraw during the callback. Concurrent calls wait until restoration completes, then print normally. The callback must not call these methods.

### UC-2: Refuse an unsafe handoff

Call handoff with unsupported input, or encounter a release/restoration failure.

Acceptance:

- [ ] Initial support requires managed stdin and runner output through `os.Stdout`, both on the same terminal, on macOS/Linux. Custom input/output, redirected streams, mismatched terminals, and unqualified platforms return `ErrHandoffUnsupported` before callback or terminal mutation. Existing custom-stream use remains supported outside handoff.
- [ ] The caller attaches the child's stdin and interactive output to that qualified terminal, as in UC-1. Handoff does not rewrite the callback's child configuration. Acceptance includes rejecting custom, redirected, and different-terminal output without starting a child.
- [ ] A runner that is not running returns `ErrHandoffNotRunning`. Reentry returns `ErrHandoffReentrant`. Errors name the failed phase and preserve underlying causes for `errors.Is`/`errors.As`.
- [ ] Qualification/release failures and the callback result use `operationErr`. Restoration failures use `restoreErr` independently, including when the callback also fails. Failed release attempts restoration after any mutation and never start the callback.
- [ ] Restoration, input-resume, or repaint failure stops input admission and rendering. `Run` returns that failure after best-effort terminal cleanup. The runner never continues interactively with uncertain ownership.
- [ ] A callback panic attempts restoration before the existing runtime panic path propagates it.
- [ ] Both runners record `Stop` requests independently of event dispatch. `Stop` returns without waiting for callback settlement, queued-event dispatch, or terminal cleanup. Handoff waits for callback return, checks the stop signal before restoring interactive modes or resuming input, and restores a plain terminal without restarting the application reader, repainting, or dispatching pending input.
- [ ] Stop-during-callback tests cover both runners with queued application events: `Stop` returns while the callback remains active, the child retains sole terminal ownership until callback return, and `Run` finishes after plain-terminal cleanup. No reader restart or application input dispatch occurs after the request.

### UC-3: Use Ctrl-D as forward delete

- [ ] Focused `InputField` consumes Ctrl-D for nonempty text, using existing Delete behavior: one forward grapheme or existing atomic special segment. At the end it consumes the key without changing text.
- [ ] Empty input returns Ctrl-D to the application for its exit policy. `OnKey` retains precedence. Deletion updates the binding and change callback once; a no-op does not report a text change. Fullscreen and inline routing agree.

## Decisions and boundaries

Use a callback API to pair release and restoration, rather than public Pause/Resume methods that callers can leave unbalanced. Keep one internal pauseable reader shared by both runners. Reject another stdin reader and decoder replacement because either loses sole ownership or buffered application input. Finalize incomplete framing at release. Reject preserving it across ownership because a child-owned suffix may never return. Define release as the boundary, so launch latency has one explicit owner.

Qualify macOS/Linux with stdin/stdout on one terminal first. Reject generic-reader cancellation and opaque output writers because their ownership cannot be checked. Keep separate errors because child failure can be recoverable while restoration failure is fatal. Reuse existing Delete semantics and hooks rather than adding a composer widget.

Editor selection, temporary files, draft replacement, subprocess cancellation, background execution, configurable keymaps, public input-owner interfaces, and a general input framework are outside this track. The [technical spec](../design/terminal-handoff.md) defines the reader and restoration contract.

## Success and risk

Success requires public `Example*` coverage, repeated real PTY handoffs in both runners, and an interactive editor recording at 80×24. A simulated editor alone cannot prove keyboard ownership. Preserve `Suspend`, custom-reader, Unicode editing, paste, keyboard, mouse, panic, and shutdown behavior. Independent implementation and acceptance reviews must exercise the real child workflow.

The main risk is pausing while a decoder holds a partial sequence or an OS read is active. PTY tests must prove acknowledged quiescence and both input boundaries before release. No product decision remains open; platform qualification is an implementation gate.
