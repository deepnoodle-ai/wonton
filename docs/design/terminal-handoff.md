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

Introduce a private managed terminal reader in `tui`, reused by both input loops. It wraps qualified terminal stdin beneath the existing decoder. It owns low-level reads, pause/resume/stop acknowledgment, and retained application bytes. Keep custom input paths unchanged, but reject their handoff before mutation. Initially qualify Darwin and Linux terminal descriptors; unsupported build targets provide the rejection path.

The managed reader uses readiness waiting with a control wakeup and reads that cannot remain blocked after a pause request. The implementation must prove its cancellation at the OS-read level, including an idle terminal. A flag checked before blocking `Read`, closing global stdin, abandoning a reader goroutine, and setting an unsupported deadline do not meet this contract. It must not leave fd flags or terminal settings changed for the child.

Pause waits until the owner acknowledges no outstanding OS read and promises not to begin another. A decoder waiting for bytes stays blocked above that owner without receiving a synthetic EOF/cancellation error. Keep its identity, buffer, and partial parse stack. Also retain event-forwarding state, including pending backslash+Enter handling and already-read events. Resume continues those same states rather than constructing a decoder or replaying decoded events as bytes.

With reads paused, capture unread kernel input into application-owned storage before release. Do not decode or discard it during handoff. The release boundary follows this capture and precedes callback invocation. Input after release belongs to the child, including launch delay. On callback return, discard the remaining child-era kernel input while Wonton's reader stays paused. Restore modes before opening application admission. Resume delivers retained bytes and events in their original order exactly once; new application input follows them. An incomplete pre-handoff sequence can use retained application bytes, never residual child bytes.

Managed stop also wakes and acknowledges the low-level read loop. Repeated handoffs cannot accumulate competing reader goroutines. Keep this mechanism private and limited to terminal ownership; do not extend `InputSource` with a speculative cancellation interface.

## Transition and restoration

The event loop owns the transition `running → releasing → child → restoring → running`, or `failed/stopped`. Command results, resize notifications, and application events may queue, but do not dispatch during handoff. Preserve pre-owned event order. Coalesce the latest size for the restoration repaint. Prevent partial frames and app output from reaching the child.

1. Check running state, input qualification, and mutual exclusion with handoff/Suspend. Acquire the output exclusion gate. Snapshot actual modes and prior terminal attributes.
2. Pause and acknowledge input reads. Retain queued input and capture unread terminal bytes. If either step fails, skip the callback.
3. Clear the inline live region or leave fullscreen alternate screen. Disable active mouse, bracketed paste, and enhanced keyboard modes. Show the cursor and restore pre-application terminal attributes. Flush output before release.
4. Invoke the callback once. A defer restores on normal return or panic. Callback errors do not bypass restoration.
5. While input remains paused, discard residual child bytes. Restore exact prior raw/protocol/screen/cursor settings, remeasure size, invalidate fullscreen cells or inline live layout, and fully repaint.
6. Resume the owner and release the output gate only after successful restoration. If stopping, perform plain-terminal cleanup without resuming. If restoring fails, mark failed and unblock waiting print calls without letting them write.

Capture and check errors from raw attributes, input boundaries, protocol writes, output flush, repaint, and reader resume. Existing terminal mode helpers that discard writer errors cannot establish success for handoff; use a checked internal path. Attempt remaining cleanup even after one step fails and join restoration causes. Track the modes Wonton actually enabled, including inline configuration, instead of enabling every feature after return.

Inline `Print`/`Printf`/`PrintRaw` use the same output gate as handoff, so concurrent calls wait and then render once. Avoid holding a state mutex needed by pause acknowledgments or stop. A waiting print returns without writing if restoration fails or shutdown wins. The callback uses terminal output directly and must never wait on a runner print method. Preserve inline scrollback and recreate only its cleared live region.

Keep `Runtime.Suspend`'s decoded-event contract and drop policy. Share only transition exclusion and terminal bookkeeping needed to prevent overlap. Handoff is the API for children that read terminal bytes directly.

## Ctrl-D and compatibility

Map nonempty Ctrl-D to `textInput`'s existing forward Delete path. Reuse grapheme boundaries, atomic paste/newline segment handling, completion cancellation, and binding synchronization. Empty Ctrl-D remains unhandled after `OnKey`, so the application's existing exit path decides. Fire `OnChange` only when text changes, including the existing focused-field binding path. No new input widget or exit policy.

The APIs are additive. Existing custom readers, `Suspend`, and defaults keep their behavior outside handoff. Applications own editor argv, files, draft updates, and operation-error presentation. They must treat `restoreErr` as terminal failure. Document the qualified platform/input matrix in `tui/README.md`, with an example that handles both results.

## Alternatives, costs, and verification

Bubble Tea's cancel/reinitialize approach is simpler, but replacing Wonton's decoder loses buffered and partial input. A public release/resume pair supports more integrations but exposes unbalanced lifecycle states. A generic cancellable-reader API adds permanent surface before a second caller exists. The shared private owner and callback keep the public contract small. The cost is low-level platform code, boundary buffering, and a synchronous pause during child execution. Input typed after release but before child startup belongs to the child; input still pending when the callback returns is discarded.

Use controlled reader tests for pause acknowledgment, release failure, restoration failure, panic, stop, ordering, and output exclusion. Preserve existing custom-source and `Suspend` tests. Focused-field tests cover ASCII, combining graphemes, ZWJ emoji, CJK, multiline/paste segments, end no-op, empty bubbling, and OnKey precedence in both runners.

Real PTY tests on macOS/Linux must cover both runners with idle blocked reads, buffered bursts, incomplete escape/UTF-8/paste input, pre-release unread bytes, child input, residual child bytes, callback errors, resize, and repeated handoffs. Assert child-exclusive stdin, normal child terminal attributes, exact app input restoration, mode sequences, repaint, and clean shutdown. Add public `ExampleRuntime_Handoff` and `ExampleInlineApp_Handoff` tests. Capture a real interactive editor recording at 80×24 for save, cancellation, and failure return; scripted/noninteractive callbacks alone do not satisfy acceptance. Run formatter, vet, build, package tests, and race tests before implementation review. Dashi integration follows release qualification and a pinned Wonton version.
