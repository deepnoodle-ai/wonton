# Terminal handoff

Status: Draft · Author: Codex · Date: 2026-09-30\
Workflow: spec, independent PRD review, then build.

## Context and contract

The [PRD](../prds/terminal-handoff.md) requires a real terminal editor to own stdin exclusively. Current `Runtime.Suspend` intentionally retains the decoder. Both runners create a nested blocking reader. `terminal.KeyDecoder` uses `bufio.Reader`; stopping event forwarding or waiting between `ReadEvent` calls cannot establish that an OS read stopped.

Add matching methods, with no new public input abstraction:

```go
func (r *Runtime) Handoff(fn func() error) (operationErr, restoreErr error)
func (r *InlineApp) Handoff(fn func() error) (operationErr, restoreErr error)
```

Call synchronously from `HandleEvent`, never a `Cmd` goroutine or renderer. The callback owns child lifecycle and waits for every terminal reader/writer it starts before returning. It cannot call runner printing, rendering, `Suspend`, or handoff methods. Document this event-loop requirement rather than adding goroutine identity inspection. Shared transition state rejects nesting and `Suspend` overlap. Nil is a no-op. Export `ErrHandoffUnsupported`, `ErrHandoffNotRunning`, and `ErrHandoffReentrant`; wrap phase-specific causes.

`operationErr` reports precondition, release, or callback failure. `restoreErr` reports restoration, reader resume, or repaint failure. Both can be nonnil. A release error skips the callback and restores any changed state. Any restoration failure marks the runner failed, stops interactive admission/rendering, and reaches `Run` after best-effort cleanup. Never emit a successful resume event after failed restoration.

## One managed input owner

Introduce a private managed terminal reader in `tui`, reused by both input loops. It wraps qualified terminal stdin beneath the existing decoder. It owns low-level reads, pause/resume/stop acknowledgment, and boundary capture. Initially qualify Darwin/Linux with managed `os.Stdin` and runner output through `os.Stdout` on the same terminal device. Reject custom input/output, redirected streams, mismatched terminals, and unsupported targets before mutation. Verify actual terminal/device identity, not only `IsTerminal`. Explicit inline `os.Stdout` is equivalent to the default; opaque writers and test terminals are unsupported.

The managed reader uses readiness waiting with a control wakeup and reads that cannot remain blocked after a pause request. The implementation must prove its cancellation at the OS-read level, including an idle terminal. A flag checked before blocking `Read`, closing global stdin, abandoning a reader goroutine, and setting an unsupported deadline do not meet this contract. It must not leave fd flags or terminal settings changed for the child.

Pause acknowledges no outstanding OS read and promises not to start another. Capture unread kernel bytes, then finalize the finite pre-release stream, including decoder buffers and parsing in progress. The decoder acknowledges neutral framing before release. Retain its identity and complete application events, never an incomplete parse stack across child ownership. An internal boundary signal may unwind parsing; it must not become application EOF or an input error.

Finalization uses only captured application bytes. It retains complete events in order and applies these rules to an incomplete suffix:

- UTF-8: retain complete scalar values; discard a trailing incomplete encoding.
- Escape/control: emit a lone ESC as Escape. Discard an incomplete CSI, SS3, or other control sequence, including its introducer. Never replay it as text.
- Bracketed paste: emit the complete UTF-8 payload as one normal paste event with existing normalization. Remove a trailing partial paste-end marker and incomplete UTF-8 encoding. Clear paste framing without waiting for its terminator. Empty payload emits nothing.
- Backslash+Enter: combine only when both events are already owned. Otherwise emit the pending backslash normally and clear its timer.

After finalization, input belongs to the child, including launch delay. On callback return, discard all residual child-era kernel bytes while reads remain paused. Restore modes, then resume the same decoder from neutral framing. Deliver retained complete events exactly once before new input. Child bytes never enter decoder buffers, complete an old sequence, or trigger discard of resumed application keys.

Managed stop also wakes and acknowledges the low-level read loop. Repeated handoffs cannot accumulate competing reader goroutines. Keep this mechanism private and limited to terminal ownership; do not extend `InputSource` with a speculative cancellation interface.

## Transition and restoration

The event loop owns the transition `running → releasing → child → restoring → running`, or `failed/stopped`. Command results, resize notifications, and application events may queue, but do not dispatch during handoff. Preserve pre-owned event order. Coalesce the latest size for the restoration repaint. Prevent partial frames and app output from reaching the child.

1. Check running state, input qualification, and mutual exclusion with handoff/Suspend. Acquire the output exclusion gate. Snapshot actual modes and prior terminal attributes.
2. Pause and acknowledge input reads. Capture unread bytes, finalize incomplete framing, and retain complete events. Await decoder acknowledgment. If any step fails, skip the callback.
3. Clear the inline live region or leave fullscreen alternate screen. Disable active mouse, bracketed paste, and enhanced keyboard modes. Show the cursor and restore pre-application terminal attributes. Flush output before release.
4. Invoke the callback once. A defer restores on normal return or panic. Callback errors do not bypass restoration.
5. While input remains paused, discard residual child bytes. Restore exact prior raw/protocol/screen/cursor settings, remeasure size, invalidate fullscreen cells or inline live layout, and fully repaint.
6. Resume the owner and release the output gate only after successful restoration. If stopping, perform plain-terminal cleanup without resuming. If restoring fails, mark failed and unblock waiting print calls without letting them write.

Capture and check errors from raw attributes, input boundaries, protocol writes, output flush, repaint, and reader resume. Existing terminal mode helpers that discard writer errors cannot establish success for handoff; use a checked internal path. Attempt remaining cleanup even after one step fails and join restoration causes. Track the modes Wonton actually enabled, including inline configuration, instead of enabling every feature after return.

Inline `Print`/`Printf`/`PrintRaw` use the same output gate as handoff, so concurrent calls wait and then render once. Avoid holding a state mutex needed by pause acknowledgments or stop. A waiting print returns without writing if restoration fails or shutdown wins. The callback uses terminal output directly and must never wait on a runner print method. Preserve inline scrollback and recreate only its cleared live region.

Keep `Runtime.Suspend`'s decoded-event contract and drop policy. Share only transition exclusion and terminal bookkeeping needed to prevent overlap. Handoff is the API for children that read terminal bytes directly.

## Ctrl-D and compatibility

Map nonempty Ctrl-D to `textInput`'s existing forward Delete path. Reuse grapheme boundaries, atomic paste/newline segment handling, completion cancellation, and binding synchronization. Empty Ctrl-D remains unhandled after `OnKey`, so the application's existing exit path decides. Fire `OnChange` only when text changes, including the existing focused-field binding path. No new input widget or exit policy.

The APIs are additive. Existing custom streams and `Suspend` keep their behavior outside handoff. Callers connect child stdin and interactive output to the qualified stdin/stdout terminal; the callback API cannot validate arbitrary subprocess configuration. Applications own argv, files, drafts, and operation-error presentation. Treat `restoreErr` as terminal failure. Document the platform/input/output matrix in `tui/README.md` with an example handling both results.

## Alternatives, costs, and verification

Replacing the decoder loses buffered input; retaining incomplete framing can capture unrelated future keys. Boundary finalization retains complete events and paste text but discards an incomplete protocol suffix. Public release/resume methods expose unbalanced states; a generic reader API adds permanent surface. The private owner and callback cost platform code, boundary processing, and a synchronous pause. Launch-delay input belongs to the child; input pending on callback return is discarded.

Use controlled reader tests for pause acknowledgment, release failure, restoration failure, panic, stop, ordering, and output exclusion. Preserve existing custom-source and `Suspend` tests. Focused-field tests cover ASCII, combining graphemes, ZWJ emoji, CJK, multiline/paste segments, end no-op, empty bubbling, and OnKey precedence in both runners.

Real PTY tests on macOS/Linux cover both runners: idle reads, buffered bursts, pre-release unread bytes, child/residual input, callback errors, resize, and repeated handoffs. Incomplete UTF-8, CSI, and bracketed-paste cases supply their suffix only during child ownership, or never. Assert retained complete events/paste text, discarded framing, then immediate ordinary typing, Enter, and application quit. Also reject redirected/custom/different-terminal output before callback. Assert exclusive child stdin, normal attributes, restored modes, repaint, and clean shutdown. Add public `ExampleRuntime_Handoff` and `ExampleInlineApp_Handoff`. Record a real editor at 80×24 for save, cancellation, and failure return; scripted callbacks alone cannot close acceptance. Run formatter, vet, build, tests, and race tests before review. Dashi follows release qualification and a pinned version.
