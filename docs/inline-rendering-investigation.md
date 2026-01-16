# Inline Rendering Investigation: Scroll Regions and Cursor Position Queries

## Problem Statement

When using `InlineApp` with a live view region at the bottom of the terminal, shrinking the terminal vertically causes **duplicate/junk content to appear in scrollback**. This happens when the live view is taller than the terminal can accommodate.

### Symptoms
- User shrinks terminal window
- Live content that no longer fits gets pushed to scrollback by the terminal
- This creates duplicate lines in the scrollback history
- The effect is cosmetically jarring and confusing

## How Codex CLI Handles This

Codex CLI (OpenAI's terminal tool) handles inline rendering more robustly. After investigation, their approach involves:

1. **Cursor Position Queries**: Before rendering, they query the terminal for the current cursor position using `ESC[6n` (Device Status Report). The terminal responds with `ESC[row;colR`.

2. **Scroll Regions (DECSTBM)**: They set up scroll regions using `ESC[top;bottomr` to constrain where content can scroll. This prevents content from being pushed into the scrollback buffer during resize.

3. **Timing**: Critically, Codex queries cursor position **before** polling for input events. This avoids a race condition where the cursor position response could be consumed by the input reader.

### Why Codex Can Do This

Codex appears to use a synchronous or carefully coordinated approach where:
- They pause/control the event stream
- Query cursor position
- Set up scroll region
- Then resume event processing

## What We Tried

### Attempt 1: Cursor Position Query Infrastructure

Added the ability to parse cursor position responses and query cursor position from `InlineApp`:

**Changes made:**
- Added `CursorPositionEvent` type to `terminal/event.go`
- Modified `decoder.go` to detect cursor position responses (`ESC[row;colR`) and return them as `CursorPositionEvent`
- Added `cursorPosChan` channel to `InlineApp` for routing cursor position responses
- Modified `inputReader` to route `CursorPositionEvent` to the dedicated channel instead of the main event channel
- Added `QueryCursorPosition()` method to `InlineApp`

**Result:** This infrastructure works correctly. Cursor position can be queried and responses are properly routed.

### Attempt 2: Scroll Regions in LivePrinter

Tried to use scroll regions to control what goes to scrollback:

**Approach:**
1. On first render, query cursor position to get starting row
2. Set scroll region from content start row to terminal bottom: `ESC[row;bottomr`
3. Use absolute cursor positioning (`ESC[row;colH`) instead of relative
4. Update scroll region on terminal resize
5. Reset scroll region on `Stop()` and `Clear()`

**Result:** **This broke normal rendering.** The live view jumped to the top of the screen instead of staying at the bottom. The scroll region constrained rendering to the wrong area.

**Why it failed:**
- Scroll regions in terminals define a "scrolling region" where content scrolls within that region
- Setting a scroll region affects where the cursor can move and where content renders
- For full-screen applications (like vim), this works because they control the entire screen
- For inline applications that coexist with scrollback, scroll regions conflict with normal terminal behavior
- The absolute positioning (`ESC[row;colH`) combined with scroll regions caused content to render at the scroll region's top, not at the current terminal position

### Attempt 3: Revert to Relative Positioning

Reverted scroll region changes and kept only the cursor query infrastructure (unused):

**Current state:**
- LivePrinter uses relative cursor positioning (`ESC[nA` to move up n lines)
- No scroll regions
- Cursor query infrastructure exists but is not wired up

**Result:** Normal rendering works correctly, but the original scrollback-junk-on-resize issue remains.

## The Fundamental Challenge

The core issue is a **mismatch between inline rendering and terminal behavior**:

1. **Terminal's natural behavior**: When content at the cursor position can't fit (e.g., terminal shrinks), the terminal scrolls content up into scrollback to make room. This is automatic and not controllable via escape sequences without scroll regions.

2. **Scroll regions**: The only way to prevent content from going to scrollback is to use scroll regions. But scroll regions are designed for full-screen applications that own the entire terminal, not for inline applications.

3. **Race condition with input**: Cursor position queries return their response through stdin. In Go, we have a goroutine continuously reading stdin for keyboard input. This creates a race where the cursor position response could be consumed by the input reader. We solved this by having the decoder recognize cursor position responses and route them to a separate channel.

4. **Inline vs Full-Screen**: Codex may be operating in a more "full-screen-like" mode even though it appears inline, or they may have found a specific combination of escape sequences that works. Their exact implementation isn't public.

## Comparison: Wonton vs Codex CLI

| Aspect | Wonton (current) | Codex CLI |
|--------|------------------|-----------|
| Cursor position queries | Infrastructure exists, unused | Used actively |
| Scroll regions | Not used (broke rendering) | Likely used |
| Cursor positioning | Relative (`ESC[nA`) | Likely absolute |
| Scrollback on resize | Junk appears | Clean behavior |
| Architecture | Async input reader goroutine | Unknown (possibly sync) |

## Possible Future Approaches

### 1. Accept the Limitation
The scrollback junk on aggressive resize may be acceptable. Most users don't rapidly resize terminals, and the live view recovers correctly after resize completes.

### 2. Full Clear on Resize
When a resize event is detected, completely clear and redraw the live region instead of trying to preserve it. This might cause a flash but would prevent junk.

### 3. Investigate Alternate Screen Buffer
Using the alternate screen buffer (`ESC[?1049h`) would prevent all scrollback contamination, but it would also hide all previous scrollback content, which defeats the purpose of inline mode.

### 4. Hybrid Approach
Detect when live content height exceeds terminal height and switch to a degraded mode (e.g., truncate content, show "..." indicator) rather than letting the terminal push content to scrollback.

### 5. Study Codex More Closely
Run Codex CLI under `script` or terminal recording to capture exact escape sequences used and understand their approach better.

## Files Modified

- `terminal/event.go` - Added `CursorPositionEvent`
- `terminal/decoder.go` - Parse cursor position responses
- `tui/inline_app.go` - Cursor query channel and method
- `tui/print.go` - Attempted scroll regions (reverted)

## Conclusion

The cursor position query infrastructure is in place and working, but the scroll region approach to prevent scrollback contamination failed because scroll regions don't work well with inline rendering. The fundamental issue is that terminals automatically push content to scrollback when there isn't room, and the only escape sequence mechanism to prevent this (scroll regions) is incompatible with inline mode.

Further investigation into Codex CLI's exact approach, or acceptance of the limitation, may be the path forward.
