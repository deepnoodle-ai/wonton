# World-Class TUI Features Analysis

This document analyzes advanced features that would elevate Gooey to world-class
status, comparing it to the best TUI libraries (bubbletea, tview, charm tools)
and modern CLI applications (lazygit, k9s, bottom).

**Status Legend:**
- 🟢 **Easy** - Straightforward implementation, leverages existing foundation
- 🟡 **Medium** - Requires new subsystems but no architectural changes
- 🔴 **Hard** - Needs significant refactoring or complex implementation
- ⚡ **High Impact** - Major user-facing improvement
- 🔧 **Foundation** - Enables other features

---

## Table of Contents

1. [Layout & Rendering](#layout--rendering)
2. [Input & Interaction](#input--interaction)
3. [Text & Styling](#text--styling)
4. [Widgets & Components](#widgets--components)
5. [Concurrency & Async](#concurrency--async)
6. [Accessibility & UX](#accessibility--ux)
7. [Theming & Customization](#theming--customization)
8. [Integration & Environment](#integration--environment)
9. [Advanced & Rare Features](#advanced--rare-features)

---

## Layout & Rendering

### 1. True Dynamic Reflow + Constraints System (Flutter/Flexbox/CSS Grid)

**Difficulty:** 🔴 Hard | **Impact:** ⚡🔧 High Impact + Foundation | **Status:** ✅ **IMPLEMENTED** (Nov 2025)

**Current State:**
- ✅ **Complete Constraint-Based System Implemented!**
- ✅ `Measurable` interface allows widgets to size themselves based on parent constraints
- ✅ `ConstraintLayoutManager` propagates constraints (min/max width/height) down the tree
- ✅ `VBoxLayout`, `HBoxLayout`, and `FlexLayout` support constraints
- ✅ `WrappingLabel` demonstrates automatic text reflow
- ✅ **Documentation:** See `documentation/constraints_and_reflow.md`
- ✅ **Demo:** See `examples/reflow_demo/main.go`

**Implementation Details:**
1. **Two-Pass Layout Engine:**
   - **Measure Pass:** Parent calls `Measure(constraints)` on children. Children return desired size.
   - **Layout Pass:** Parent calls `SetBounds` to position children based on measured size.
2. **Constraint Propagation:**
   - Layout managers reduce available space (accounting for padding/margins) and pass it to children.
   - Widgets like `WrappingLabel` use `MaxWidth` to determine where to wrap text.
3. **Backward Compatibility:**
   - `MeasureWidget` helper automatically falls back to `GetPreferredSize` for legacy widgets.

**Future Enhancements:**
- `GridLayout` with constraint support (currently legacy)
- `StackLayout` for z-ordering (needed for overlays/modals)

---

### 2. Fractional/Weighted Layout (0.3 width, 30% height, etc.)

**Difficulty:** 🟡 Medium | **Impact:** ⚡ High Impact | **Status:** ✅ **IMPLEMENTED** (Nov 2025)

**Current State:**
- ✅ **FlexLayout Implemented!** (See `flex_layout.go`)
- ✅ Supports `FlexGrow` and `FlexShrink` for weighted sizing
- ✅ Supports `FlexWrap` for automatic wrapping of items
- ✅ Integration with new Constraint System ensures correct sizing

**Implementation Details:**
- `FlexLayout` uses the standard CSS Flexbox algorithm.
- `FlexGrow: 1` distributes remaining space equally.
- `FlexGrow: 2` takes double the space of `FlexGrow: 1`.
- Works seamlessly with `VBoxLayout` (which also supports `Grow` factor in `LayoutParams`).

---

### 3. Split-Pane Container with Draggable Dividers

**Difficulty:** 🟡 Medium | **Impact:** ⚡ High Impact

**Current State:**
- Grid can create pane-like layouts but dividers are static
- Mouse support exists (`mouse.go`)
- No drag-and-drop infrastructure

**How to Implement:**
1. **SplitPane Component:**
   ```go
   type SplitPane struct {
       Orientation  Orientation // Horizontal, Vertical
       Split        float64     // 0.0 to 1.0
       MinSplit     float64
       MaxSplit     float64
       Resizable    bool
       ShowDivider  bool
       Child1       Widget
       Child2       Widget
       dragging     bool
   }
   ```

2. **Interaction:**
   - Render divider line (│ or ─ characters with distinct style)
   - Mouse hover detection - change cursor or divider style
   - Click + drag to resize - update `Split` value
   - Keyboard support - focus divider, use arrow keys to resize
   - Touchpad gesture support (if terminal supports)

3. **Animation:**
   - Smooth split changes (interpolate over frames)
   - Momentum for touchpad fling gestures

**Easy Aspects:**
- Two-child layout is simple
- Existing mouse infrastructure (MouseRegion, MouseHandler)
- SubFrame makes rendering child bounds trivial

**Hard Aspects:**
- Drag state management (press, move, release across frames)
- Cursor changes (many terminals don't support custom cursors)
- Smooth animation conflicts with keyboard-driven UIs
- Keyboard accessibility (how to focus/move divider without mouse?)
- Nested splits (VSCode-style 4-way split)

**Things to Consider:**
- Save/restore split positions (user preferences)
- Snap-to-size behavior (e.g., snap to 50/50 when near middle)
- Collapse behavior (drag to edge to hide pane)
- Multiple dividers for 3+ panes
- Terminal resizes - maintain percentage or absolute sizes?

**Overlap:**
- **Fractional layout** - same underlying math
- **Mouse support** - uses existing infrastructure
- **Focus management** - divider should be focusable
- **Layout managers** - generalized by FlexLayout

---

### 4. Virtualized Rendering (Lists/Tables)

**Difficulty:** 🟡 Medium | **Impact:** ⚡ High Impact | **Status:** 🔓 **UNBLOCKED** (Nov 2025)

**Current State:**
- Table component renders all rows
- No viewport/scrolling abstraction
- **New Layout System:** The `Measurable` interface allows measuring items before rendering, which is critical for virtualization.

**How to Implement:**
1. **Virtual Viewport:**
   ```go
   type VirtualList struct {
       ItemCount    int
       ItemHeight   int // Fixed height per item (or callback)
       ViewportY    int
       ViewportH    int
       RenderItem   func(index int, frame RenderFrame)
       scrollOffset int
   }
   ```

2. **Rendering:**
   - Calculate visible range: `start = scrollOffset / itemHeight`, `end = (scrollOffset + viewportH) / itemHeight`
   - Render only visible items: `for i := start; i <= end; i++ { renderItem(i) }`
   - Draw scrollbar indicator

3. **Scrolling:**
   - Mouse wheel events
   - Keyboard (up/down arrows, page up/down, home/end)
   - Smooth scrolling with pixel offsets (if terminal supports)

4. **Variable Height (Unlocked by Reflow):**
   - Use `Measure(constraints)` to calculate height of items without rendering them.
   - Maintain cumulative height array: `heights[i] = sum(itemHeight[0..i])`
   - Binary search to find visible range
   - Cache measured heights

**Easy Aspects:**
- Fixed-height case is trivial math
- Existing dirty regions prevent full redraws
- Keyboard scrolling is straightforward

**Hard Aspects:**
- Variable-height items (text wrapping, expanded tree nodes)
- Smooth scrolling in character-based terminals (pixel scrolling not universal)
- Scroll position persistence during data updates (keep item X in view)
- Bidirectional virtualization (wide tables with horizontal scroll)
- Estimating total scrollbar size without measuring all items

**Things to Consider:**
- Overscan (render extra rows above/below for smooth scrolling)
- Scroll inertia/momentum for touchpad
- Search integration (jump to item, highlight in viewport)
- Selection spanning beyond viewport
- Infinite scrolling / lazy loading integration

**Overlap:**
- **Tree view with lazy loading** - same virtualization tech
- **Table component** - immediate beneficiary
- **Scrollbars** - need visual indicator
- **Mouse wheel** - primary interaction

**Recommendation:** Start with fixed-height list, expand to tables, then variable-height.

---

### 5. Off-Screen Buffering + Damage Region Tracking

**Difficulty:** 🟢 Easy | **Impact:** 🔧 Foundation | **Status:** ✅ **PROFILING COMPLETE**

**Current State:**
- ✅ **Already implemented!** Double-buffering exists
- ✅ DirtyRegion tracks modified cells
- ✅ Flush only updates changed regions
- ✅ **Profiling/Metrics system implemented!** (Jan 2025)
  - Comprehensive metrics tracking: frames, cells, ANSI codes, bytes, timing, FPS
  - Thread-safe with minimal overhead (<100% when enabled, zero when disabled)
  - API: `EnableMetrics()`, `GetMetrics()`, `ResetMetrics()`
  - Formatted output via `String()` and `Compact()`
  - See: `metrics.go`, `documentation/metrics.md`, `examples/metrics_demo/`

**Future Enhancements (Deferred):**
1. **Finer-Grained Tracking:**
   - Per-widget dirty flags (currently tracks global dirty region)
   - Z-order optimization (don't redraw widgets under opaque overlays)

2. **Dirty Rectangles (Multiple Regions):**
   - Current: Single bounding box of all changes
   - Enhanced: Track multiple disjoint rectangles
   - Skip rendering gaps between dirty regions
   - **Note:** Defer until profiling data shows this is needed

**Easy Aspects:**
- Infrastructure already exists
- ✅ Metrics implementation was straightforward as predicted
- Per-widget flags are simple bookkeeping

**Hard Aspects:**
- Multiple dirty regions increase complexity (diminishing returns?)
- ✅ Profiling achieved minimal overhead via conditional checks
- Z-order culling requires widget depth tracking

**Things to Consider:**
- Is multi-region tracking worth complexity? (metrics can now answer this!)
- ✅ Metrics exposed via API with both detailed and compact output
- Integration with layout system (whole subtree dirty vs leaf-only)

**Implementation Notes:**
- Followed recommendation: Profiling first ✅, multi-region deferred ⏸️
- Metrics disabled by default for zero overhead
- Thread-safe using RWMutex
- Comprehensive test coverage including performance overhead tests

---

## Input & Interaction

### 6. Full Mouse Support (Click, Drag, Wheel, Hover)

**Difficulty:** 🟡 Medium | **Impact:** ⚡ High Impact

**Current State:**
- ✅ `mouse.go` exists with `MouseHandler` and rectangular `MouseRegion` hit-testing
- ✅ Basic click handling reaches widgets
- ⚠️ Drag/hover only partially covered; no unified capture/enter/leave lifecycle
- ⚠️ Mouse wheel parsing/dispatch not verified; acceleration not present

**How to Implement:**
1. **Unified Event Model + Normalization:**
   - Standardize payload: `Type (Press, Release, Move, Drag, Wheel, Enter, Leave, Click, DoubleClick)`, `Button (Left/Right/Middle)`, `X/Y`, `Modifiers`, `DeltaX/DeltaY` for wheel.
   - Normalize coordinates to screen space, then transform to widget-local coordinates when dispatching.
   - Map protocol-specific wheel units (SGR 1006/1016 vs legacy) into a consistent small integer delta; capture horizontal wheel where available.

2. **Region Tracking & Dispatch Order:**
   - Maintain active `MouseRegion` list with z-order and hit-testing against bounds (absolute for legacy widgets, relative for composable ones).
   - Resolve target region on every move; deliver Enter/Leave when target changes.
   - Allow pointer capture: on `Press`, mark capturing region so subsequent `Move/Drag/Release` events stay routed even if cursor leaves bounds (critical for splitters/drag handles).

3. **Hover + Cursor State Machine:**
   - Track `lastHoveredRegion` and fire `Enter/Leave` plus optional `HoverMove` for tooltips or hover styles.
   - Provide per-region hover debounce/threshold to avoid flicker when moving diagonally across tight controls.
   - Optional cursor hints: expose `CursorStyle` (pointer, resize-ew, resize-ns) even if some terminals ignore it.

4. **Drag Lifecycle:**
   - Emit `DragStart` on press; stash origin + button + modifiers.
   - On move while pressed, emit `Drag` with deltas and absolute positions; support multi-button drags (middle-button panning).
   - Emit `DragEnd` on release regardless of cursor location (requires capture).
   - Handle cancel (Esc) to abort drag and send `DragCancel`.

5. **Wheel + Momentum:**
   - Parse wheel up/down (and left/right if available) as signed `DeltaY/DeltaX`.
   - Add optional acceleration: if multiple wheel events arrive within short intervals, scale delta; apply inertia decay (synthetic smaller events) for smooth scrolling.
   - Let components opt into consuming wheel (lists/tables) or bubbling to parent scroll containers.

6. **Double/Triple Click Detection:**
   - Track last click time/button/position; if within threshold, emit `DoubleClick`/`TripleClick` instead of separate single clicks.
   - Allow per-widget thresholds to match terminal feel; ensure single-click still fires after double-click timeout (or expose explicit `ClickCount`).

7. **Testing & Diagnostics:**
   - Add debug mode to log parsed events and routing decisions; useful for terminal quirks.
   - Write table/list/split-pane harness tests that simulate sequences: enter → press → drag out of bounds → release, wheel bursts, rapid hover transitions.

**Easy Aspects:**
- Existing `MouseHandler`/`MouseRegion` give hit-testing and callback plumbing.
- SGR event formats are documented; terminal enable/disable lives beside other init codes.
- Drag/hover logic is stateful bookkeeping rather than heavy rendering changes.

**Hard Aspects:**
- Terminal variance: some terminals drop hover/move events unless specific modes are enabled; wheel precision differs.
- Ensuring capture works across nested containers with mixed absolute vs bounds-based coordinates.
- Momentum/acceleration tuning so lists feel natural but not jumpy.
- Avoiding hover/drag thrash in dense UIs while keeping latency low.

**Things to Consider:**
- Focus policy (hover-focus vs click-focus) and how mouse events interact with keyboard-driven focus ring.
- Right-click context menus and long-press as fallback; configurable per-widget.
- Accessibility: keep full keyboard parity and let users disable momentum/hover effects.
- Gesture extensions (pinch/trackpad) likely out of scope; prefer wheel + drag abstractions.

**Overlap:**
- **Split-pane dragging** and **column resizing** rely on capture + drag delta.
- **Scrolling** components (lists/tables/modals) depend on wheel normalization and momentum.
- **Tooltips** and **hover highlights** need Enter/Leave/Move fidelity.
- **Context menus** and **multi-click selection** reuse click counting and button differentiation.

---

### 7. SGR 1006/1016 Mouse Protocol with Auto-Detection

**Difficulty:** 🟢 Easy | **Impact:** 🔧 Foundation

**Current State:**
- Unknown if auto-detection exists
- Likely uses basic mouse mode

**How to Implement:**
1. **Protocol Progression:**
   - Send escape codes to enable modes in order of preference:
     - `\033[?1003h` - All mouse events
     - `\033[?1006h` - SGR extended mode (coordinates > 255)
     - `\033[?1016h` - SGR pixel mode (if supported)
   - Query terminal capabilities via XTVERSION or terminfo

2. **Fallback Chain:**
   - Try SGR 1006 first (most common modern)
   - Fall back to legacy X10/X11 if not supported
   - Detect by parsing response sequences

3. **Coordinate Handling:**
   - SGR 1006: supports coordinates beyond 255
   - Legacy: limited to 223 columns
   - Handle both formats in parser

**Easy Aspects:**
- Escape codes are well-documented
- Fallback is simple (just send different codes)
- Existing input parsing infrastructure

**Hard Aspects:**
- Reliable terminal capability detection
- Handling terminals that lie about support
- Testing across many terminals (iTerm2, WezTerm, xterm, Windows Terminal, etc.)

**Things to Consider:**
- Store detected capabilities for session
- User override (force mode via env var)
- Logging/debugging which mode is active

**Overlap:**
- **Mouse support** - foundational for all mouse features
- **Environment detection** - same capability detection system

---

### 8. Bracketed Paste Mode

**Difficulty:** 🟢 Easy | **Impact:** ⚡ High Impact | **Status:** ✅ **IMPLEMENTED** (Jan 2025)

**Current State:**
- ✅ **Complete bracketed paste mode implementation!**
- ✅ Terminal methods: `EnableBracketedPaste()`, `DisableBracketedPaste()`
- ✅ KeyDecoder parses `\033[200~` (start) and `\033[201~` (end) sequences
- ✅ KeyEvent includes `Paste` field for paste content
- ✅ Input.Read() automatically handles paste events
- ✅ Comprehensive test coverage (11 test cases)
- ✅ Full documentation with examples
- ✅ Security benefits: prevents accidental command execution
- ✅ Graceful degradation in unsupported terminals
- See: `terminal.go`, `key_decoder.go`, `input.go`, `documentation/bracketed_paste.md`, `examples/bracketed_paste_demo/`

**How to Implement:**
1. **Enable Mode:**
   - Send `\033[?2004h` on terminal init
   - Send `\033[?2004l` on terminal restore

2. **Parse Paste Events:**
   - Detect `\033[200~` (paste start)
   - Collect all text until `\033[201~` (paste end)
   - Emit as single `PasteEvent` with full content

3. **Input Field Integration:**
   - Insert pasted text as single operation (undo-able)
   - Prevent interpreting newlines as Enter key
   - Sanitize pasted content (strip ANSI codes, etc.)

**Easy Aspects:**
- Simple escape codes
- Clear start/end delimiters
- Supported by all modern terminals

**Hard Aspects:**
- Handling malicious paste (ANSI codes, control chars)
- Large pastes (buffer limits, performance)
- Paste during non-input modes (ignore? queue?)

**Things to Consider:**
- Paste confirmation for large/multiline pastes
- Syntax highlighting pasted code
- Paste into password fields (security)

**Overlap:**
- **Input fields** - primary beneficiary
- **Security** - paste attack prevention

**Recommendation:** Implement immediately - high value, minimal effort.

---

### 9. IME Support (East Asian Input)

**Difficulty:** 🔴 Hard | **Impact:** ⚡ High Impact

**Current State:**
- Unlikely to be supported
- Wide character display works (go-runewidth) but input likely broken

**How to Implement:**
1. **Composition Feedback:**
   - Terminals send pre-composition characters
   - Display with underline/different style
   - Show candidate window (if terminal supports)

2. **Event Handling:**
   ```go
   type IMEEvent struct {
       Composing   bool
       Composition string
       Candidates  []string
       Cursor      int
   }
   ```

3. **Terminal Integration:**
   - Some terminals (iTerm2) have native IME support
   - Others require application-level handling
   - Query capabilities via escape codes

**Easy Aspects:**
- Display is already wide-char aware
- Basic composition events are parsable

**Hard Aspects:**
- Terminal support is inconsistent (iTerm2 yes, many others no)
- Testing without native IME knowledge
- Candidate window positioning (may be terminal-controlled)
- Complex input methods (Pinyin, Cangjie, etc.) have different flows

**Things to Consider:**
- Graceful degradation (fall back to basic input)
- Testing matrix (Chinese, Japanese, Korean)
- Right-to-left text (Arabic, Hebrew) - different challenge
- Emoji picker (similar composition UX)

**Overlap:**
- **Text rendering** - already handles wide chars
- **Input fields** - integration point
- **Environment detection** - capability checking

**Recommendation:** Defer until user demand - complex and niche.

---

### 10. Secure Password Input (iTerm2, VS Code Terminal)

**Difficulty:** 🟢 Easy | **Impact:** 🔧 Foundation | **Status:** ✅ **IMPLEMENTED** (Nov 2025)

**Current State:**
- ✅ **Complete secure password input system!**
- ✅ `PasswordInput` component with iTerm2 secure mode support
- ✅ `SecureString` type with automatic memory zeroing
- ✅ Masked character display option (asterisks or custom)
- ✅ No-echo mode using term.ReadPassword
- ✅ Clipboard disable support
- ✅ Paste confirmation option
- ✅ Max length enforcement
- ✅ **Documentation:** See `documentation/password_input.md`
- ✅ **Demo:** See `examples/password_demo/`
- ✅ Comprehensive test coverage

**How to Implement:**
1. **Secure Input Mode:**
   - iTerm2: Send `\033]1337;SetUserVar=PasswordInput=1\007`
   - VS Code: Uses different protocol (check docs)
   - Generic: No echo + clipboard disable

2. **Password Field:**
   ```go
   type PasswordInput struct {
       Placeholder string
       Value       []byte  // Store as bytes, not string
       MaxLength   int
       secureMode  bool
   }
   ```

3. **Security Measures:**
   - Disable clipboard copying while focused
   - Clear value on lose focus (optional)
   - Zero memory on component destroy
   - Disable paste (or require confirmation)

**Easy Aspects:**
- iTerm2 protocol is simple
- Hiding characters is trivial

**Hard Aspects:**
- Cross-terminal compatibility
- Detecting secure input capability
- Memory security (prevent swap, clear on GC)
- Paste handling (allow? sanitize?)

**Things to Consider:**
- Password strength indicator
- Show/hide toggle (eye icon)
- Autocomplete from password managers (browser integration)
- Timeout-based auto-hide

**Overlap:**
- **Input fields** - specialized variant
- **Bracketed paste** - may want to disable

---

### 11. Inline Images (OSC 1337 / Sixel / Kitty)

**Difficulty:** 🔴 Hard | **Impact:** ⚡ High Impact

**Current State:**
- No image support

**How to Implement:**
1. **Protocol Detection:**
   - Query terminal for supported graphics protocols
   - Priority: Kitty > iTerm2 inline > Sixel > ASCII art fallback

2. **iTerm2 Inline Images (OSC 1337):**
   - Encode image as base64
   - Send: `\033]1337;File=inline=1;width=Npx;height=Npx:[base64]\007`
   - Terminal renders inline with text

3. **Kitty Graphics Protocol:**
   - More efficient than base64
   - Supports animation, z-index, scrolling
   - Chunked transmission for large images

4. **Sixel:**
   - Older protocol, wide support (xterm, mlterm)
   - Limited to 256 colors (palette-based)
   - Encode image as escape sequences

5. **Fallback:**
   - ASCII art conversion (use library like `go-term-img`)
   - Unicode block characters (▀▄█▌▐░▒▓)
   - Braille characters for higher resolution

**Easy Aspects:**
- iTerm2 protocol is well-documented
- Base64 encoding is standard library
- Image decoding (PNG, JPEG) via `image` package

**Hard Aspects:**
- Protocol compatibility matrix is complex
- Image positioning (absolute vs inline)
- Handling terminal scrollback (images disappear)
- Resize behavior (scale vs crop)
- Performance (large images, animation)
- Cross-platform testing (many terminals)

**Things to Consider:**
- Image caching (avoid re-encoding)
- Lazy loading (thumbnail -> full res)
- Animation (GIF, video frames)
- Clickable images (image maps)
- Accessibility (alt text)

**Overlap:**
- **Environment detection** - capability checking
- **Virtualized rendering** - scrolling images
- **Rich content** - complements markdown rendering

**Recommendation:** Start with iTerm2 + ASCII fallback, expand based on user needs.

---

## Text & Styling

### 12. Full 24-bit Truecolor + Dynamic Downgrade

**Difficulty:** 🟡 Medium | **Impact:** ⚡ High Impact

**Current State:**
- ✅ RGB colors supported via `FgRGB`, `BgRGB`
- ❓ Unknown if downgrade implemented

**How to Implement:**
1. **Color Capability Detection:**
   - Check `$COLORTERM` env var (truecolor, 24bit)
   - Check `$TERM` (xterm-256color, etc.)
   - Query terminal via `\033[c` or terminfo
   - Store as `ColorDepth` enum (TrueColor, Color256, Color16, Monochrome)

2. **Downgrade Functions:**
   - **TrueColor -> 256 color:**
     - Use 6x6x6 color cube: `16 + 36*r + 6*g + b` where r,g,b ∈ [0,5]
     - Or use grayscale ramp for gray colors
   - **256 -> 16 color:**
     - Map to nearest ANSI color (NN distance in RGB space)
   - **16 -> 2 color:**
     - Threshold on luminance: `0.299*R + 0.587*G + 0.114*B`

3. **Palette Mapping:**
   ```go
   type ColorMapper struct {
       depth  ColorDepth
       cache  map[RGB]string // Memoize conversions
   }

   func (cm *ColorMapper) ToANSI(color RGB) string {
       switch cm.depth {
       case TrueColor: return fmt.Sprintf("\033[38;2;%d;%d;%dm", r, g, b)
       case Color256:  return fmt.Sprintf("\033[38;5;%dm", cm.rgbTo256(color))
       case Color16:   return fmt.Sprintf("\033[%dm", cm.rgbTo16(color))
       case Monochrome: return "" // Use bold/dim instead
       }
   }
   ```

**Easy Aspects:**
- RGB to 256 color cube is well-defined math
- ANSI color codes are standard
- Caching prevents recalculation

**Hard Aspects:**
- Optimal color mapping (perceptual distance vs RGB distance)
- Handling theme colors (some terminals remap ANSI colors)
- Dithering for smooth gradients in low color modes
- Testing across all color depths

**Things to Consider:**
- User override (force color depth)
- Graceful degradation (bold/italic as color substitute)
- Theme-aware mapping (dark vs light backgrounds)
- Color blindness modes (deuteranopia, protanopia, tritanopia)

**Overlap:**
- **Environment detection** - same capability system
- **Theming** - color variables need downgrade
- **Contrast checking** - needs color conversion

---

### 13. Color Contrast Checker (WCAG AA/AAA)

**Difficulty:** 🟢 Easy | **Impact:** 🔧 Foundation

**Current State:**
- No contrast checking

**How to Implement:**
1. **Contrast Ratio Calculation:**
   ```go
   func ContrastRatio(c1, c2 RGB) float64 {
       l1 := RelativeLuminance(c1)
       l2 := RelativeLuminance(c2)
       lighter := max(l1, l2)
       darker := min(l1, l2)
       return (lighter + 0.05) / (darker + 0.05)
   }

   func RelativeLuminance(c RGB) float64 {
       // Convert RGB to linear space, then apply formula
       // https://www.w3.org/WAI/GL/wiki/Relative_luminance
   }
   ```

2. **WCAG Levels:**
   - AA large text: 3:1
   - AA normal text: 4.5:1
   - AAA large text: 4.5:1
   - AAA normal text: 7:1

3. **Auto-Correction:**
   - If contrast too low, darken/lighten one color
   - Binary search for minimum adjustment
   - Preserve hue (only adjust brightness)

**Easy Aspects:**
- Formula is standard (WCAG 2.1)
- Pure math, no I/O
- Fast enough to run on every render

**Hard Aspects:**
- Auto-correction may not preserve brand colors
- Hue-preserving adjustment is subjective
- Some color combinations can't meet AAA (pure yellow on white)

**Things to Consider:**
- Lint mode (warn but don't change)
- Accessibility profiles (AA vs AAA)
- Custom palettes (ensure all pairs have good contrast)
- Testing with actual color-blind users

**Overlap:**
- **Theming** - validate themes meet standards
- **Color downgrade** - verify low-color modes stay accessible
- **High-contrast mode** - extreme version of auto-correction

---

### 14. Hyperlink Support (OSC 8)

**Difficulty:** 🟢 Easy | **Impact:** ⚡ High Impact | **Status:** ✅ **IMPLEMENTED** (Jan 2025)

**Current State:**
- ✅ **Complete OSC 8 hyperlink support!**
- ✅ Hyperlink type with URL, text, and style
- ✅ PrintHyperlink and PrintHyperlinkFallback methods
- ✅ Automatic URL validation
- ✅ Default blue underlined styling with customization
- ✅ Graceful fallback for unsupported terminals
- ✅ Comprehensive test coverage
- ✅ Full documentation and examples
- See: `hyperlink.go`, `documentation/hyperlinks.md`, `examples/hyperlink_demo/`

**How to Implement:**
1. **OSC 8 Protocol:**
   - Start link: `\033]8;;URL\033\\`
   - Link text: normal text output
   - End link: `\033]8;;\033\\`
   - Example: `\033]8;;https://example.com\033\\Click Here\033]8;;\033\\`

2. **API:**
   ```go
   type Hyperlink struct {
       URL   string
       Text  string
       Style Style
   }

   func (frame RenderFrame) PrintHyperlink(x, y int, link Hyperlink)
   ```

3. **Feature Detection:**
   - Query terminal support (not widely implemented)
   - Graceful degradation: render as `Text (URL)` if unsupported

4. **Interaction:**
   - Terminal handles click (opens browser)
   - Application doesn't need to handle events
   - Optional: detect hover for style change

**Easy Aspects:**
- Simple escape codes
- Terminal does all the work
- No event handling needed

**Hard Aspects:**
- Support is inconsistent (iTerm2 yes, many others no)
- No way to know if link was clicked
- Security concerns (phishing via misleading text)
- Rendering in unsupported terminals (fallback UX)

**Things to Consider:**
- Link validation (ensure URL is valid)
- Relative vs absolute URLs
- Fragment links (jump to section in TUI? probably not)
- Styling (underline, color, hover effect if terminal supports)

**Overlap:**
- **Markdown rendering** - links in formatted text
- **Environment detection** - capability checking
- **Rich text** - links are inline content

**Recommendation:** Implement with clear fallback - widely requested feature.

---

### 15. Grapheme Cluster-Aware Text (Emoji, Zalgo, etc.)

**Difficulty:** 🔴 Hard | **Impact:** 🔧 Foundation

**Current State:**
- ✅ go-runewidth handles wide characters
- ❌ Likely doesn't handle complex grapheme clusters

**How to Implement:**
1. **Use Unicode Segmentation:**
   - Integrate `github.com/rivo/uniseg` (grapheme cluster library)
   - Replace rune iteration with grapheme iteration
   - Example: "👨‍👩‍👧‍👦" is 1 grapheme, 7 runes

2. **Width Calculation:**
   - East Asian Width property (wide, narrow, neutral)
   - Emoji presentation (text vs emoji style)
   - Zero-width joiners (ZWJ) and combiners
   - Handle modifier sequences (skin tone, gender)

3. **Wrapping & Truncation:**
   - Wrap at grapheme boundaries (not rune or byte)
   - Truncate with ellipsis at grapheme boundary
   - Handle breaking spaces vs non-breaking

4. **Rendering:**
   - Some terminals render emoji as 2 cells, others 1
   - Detect via test sequence: render emoji, query cursor position
   - Pad with spaces if needed

**Easy Aspects:**
- Libraries handle the hard Unicode logic
- Modern terminals generally support emoji

**Hard Aspects:**
- Terminal behavior is inconsistent (emoji width varies)
- Combining marks can stack infinitely (Zalgo text)
- Performance (grapheme iteration slower than runes)
- Right-to-left text (bidirectional algorithm)
- Font availability (missing emoji render as ▯)

**Things to Consider:**
- Emoji picker widget (searchable, categorized)
- Emoji rendering mode (text style vs colored)
- Terminal feature detection (does it support emoji?)
- Graceful degradation (text descriptions, :shortcodes:)

**Overlap:**
- **Text rendering** - foundational change
- **Input fields** - cursor positioning with graphemes
- **Markdown** - emoji in formatted text

**Recommendation:** Use uniseg for correctness, add terminal-specific emoji width detection.

---

### 16. Rich Text / Markdown Renderer

**Difficulty:** 🟡 Medium | **Impact:** ⚡ High Impact

**Current State:**
- No rich text support
- Manual styling via ANSI codes

**How to Implement:**
1. **Markdown Parsing:**
   - Use `github.com/yuin/goldmark` (extensible, CommonMark compliant)
   - Parse to AST (Abstract Syntax Tree)

2. **Rendering:**
   - Walk AST, convert to styled text
   - **Bold:** ANSI bold code
   - **Italic:** ANSI italic (not universal)
   - **Code:** Different color + monospace hint (no actual font change)
   - **Headings:** Larger text (bold + underline), or color change
   - **Lists:** Render bullets/numbers with indentation
   - **Links:** Use OSC 8 hyperlinks (or show as footnotes)
   - **Images:** Inline image protocol (or [IMAGE] placeholder)
   - **Code blocks:** Syntax highlighting

3. **Syntax Highlighting:**
   - Use `github.com/alecthomas/chroma` (same as charm/glamour)
   - Convert tokens to colored text
   - Theme-aware (match terminal color scheme)

4. **Layout:**
   - Word wrapping (grapheme-aware)
   - Table rendering (align columns, borders)
   - Block quotes (indentation, left border)
   - Horizontal rules (unicode line characters)

**Easy Aspects:**
- Libraries handle parsing
- ANSI styling is straightforward
- Similar to web rendering (familiar model)

**Hard Aspects:**
- Table alignment with variable-width fonts (terminals are monospace, but CJK chars are wide)
- Image fallbacks (ASCII art? placeholders?)
- Nested formatting (bold+italic+code)
- Preserving semantic structure (screen readers)
- Performance for large documents

**Things to Consider:**
- Live preview (edit markdown, see rendered)
- Custom extensions (GitHub-flavored markdown, task lists)
- Themeable (allow user color schemes)
- Export rendered output (e.g., to HTML)
- Pager integration (less-like navigation)

**Overlap:**
- **Hyperlinks** - markdown links
- **Images** - markdown images
- **Syntax highlighting** - code blocks
- **Pager** - viewing long documents

**Recommendation:** High value, use `goldmark` + `chroma` like charm/glow does.

---

### 17. Animated Text Effects (Typewriter, Shimmer, Wave, Rainbow)

**Difficulty:** 🟢 Easy | **Impact:** ⚡ High Impact

**Current State:**
- ✅ Rainbow, Pulse, Wave animations exist!
- ✅ TextAnimation interface with GetStyle

**How to Enhance:**
1. **Additional Effects:**
   - **Typewriter:** Reveal text character-by-character
     ```go
     type TypewriterAnimation struct {
         Speed       int // Chars per second
         revealed    int
         lastFrame   uint64
     }
     ```
   - **Shimmer:** Traveling highlight (like loading shimmer)
   - **Blink:** Classic terminal blink (but modern)
   - **Glitch:** Random color/position shifts (matrix effect)
   - **Fade In/Out:** Gradually change alpha (or brightness)

2. **Easing Functions:**
   - Linear, EaseIn, EaseOut, EaseInOut
   - Bounce, Elastic, Back
   - Apply to color transitions, positions, etc.

3. **Composable Animations:**
   - Chain animations: typewriter -> rainbow -> pulse
   - Combine effects: wave + shimmer

**Easy Aspects:**
- Infrastructure exists
- Frame counter makes timing easy
- Per-character styling is flexible

**Hard Aspects:**
- Easing function math (borrow from web animation libraries)
- Composing animations without conflicts
- Performance (many animated elements)
- Pausing/resuming animations

**Things to Consider:**
- Accessibility (disable animations on request)
- Battery usage (animation FPS throttling)
- Distraction (overuse hurts UX)
- Animation presets (named themes)

**Overlap:**
- **Theming** - animations are part of theme
- **Accessibility** - respect prefers-reduced-motion
- **FPS control** - existing animator system

**Recommendation:** Add typewriter and shimmer - most commonly requested.

---

## Widgets & Components

### 18. File Picker (Mouse, Fuzzy Search, Icons)

**Difficulty:** 🟡 Medium | **Impact:** ⚡ High Impact

**Current State:**
- No file picker widget

**How to Implement:**
1. **File Tree Display:**
   - Use Tree widget (if exists) or build custom
   - Render directories with expand/collapse
   - Icons via unicode glyphs or Nerd Fonts

2. **Fuzzy Search:**
   - Use `github.com/sahilm/fuzzy` or `github.com/junegunn/fzf` algorithm
   - Filter files as user types
   - Highlight matching characters
   - Score and sort by relevance

3. **Interactions:**
   - Mouse: click to select, double-click to open/expand
   - Keyboard: arrows to navigate, Enter to select, / to search
   - Multi-select: Space to toggle, Shift+arrows for range

4. **Preview Pane:**
   - Show file contents (text files)
   - Image preview (if inline images supported)
   - File metadata (size, modified date, permissions)

**Easy Aspects:**
- File system traversal is standard library
- Fuzzy search libraries exist
- Mouse events are supported

**Hard Aspects:**
- Large directories (virtualization needed)
- Symlinks, permissions, hidden files
- Preview for binary files (hex dump? file command output?)
- Cross-platform paths (Windows vs Unix)
- Performance (caching, lazy loading)

**Things to Consider:**
- .gitignore support (respect exclusion rules)
- Recent files (frecency algorithm)
- Bookmarks (quick jump to common dirs)
- Create new file/folder within picker
- Drag-and-drop (if terminal supports)

**Overlap:**
- **Tree view** - file tree is specific case
- **Fuzzy search** - reusable component
- **Virtualized rendering** - for large dirs
- **Icons** - Nerd Fonts or emoji

---

### 19. Tree View (Lazy Loading, Async Expansion)

**Difficulty:** 🟡 Medium | **Impact:** ⚡ High Impact

**Current State:**
- No tree widget

**How to Implement:**
1. **Tree Structure:**
   ```go
   type TreeNode struct {
       Label      string
       Icon       string
       Expanded   bool
       Children   []*TreeNode
       LazyLoad   func() ([]*TreeNode, error)
       loading    bool
   }

   type TreeView struct {
       Root         *TreeNode
       Selected     *TreeNode
       OnSelect     func(*TreeNode)
       virtualized  bool
   }
   ```

2. **Rendering:**
   - Indent levels with `│ ├─ └─` characters
   - Expand/collapse icons: `▶ ▼` or `+ -`
   - Highlight selected node
   - Virtualize for large trees

3. **Lazy Loading:**
   - Node has `LazyLoad` callback
   - On expand, show loading indicator
   - Run callback in goroutine
   - Update UI when children arrive

4. **Interactions:**
   - Arrow keys: up/down to navigate, left/right to collapse/expand
   - Mouse: click to select, click icon to toggle
   - Type-ahead: jump to node starting with typed letter
   - Search: fuzzy search across all nodes

**Easy Aspects:**
- Recursive rendering is straightforward
- Lazy loading prevents initial slowness
- Existing virtualization can be reused

**Hard Aspects:**
- Performance for deep/wide trees (virtualization essential)
- Async loading (coordinating goroutines with UI updates)
- Selection during tree mutation (node might move/disappear)
- Scrolling (keep selected node visible)
- Indentation with wide characters

**Things to Consider:**
- Expand all / collapse all
- Filter/search (hide non-matching subtrees)
- Drag-and-drop to reorder (if mouse supports)
- Checkboxes for multi-select
- Custom icons per node type

**Overlap:**
- **File picker** - uses tree for directories
- **Virtualized rendering** - essential for performance
- **Async components** - lazy loading pattern
- **Fuzzy search** - search integration

---

### 20. Advanced Progress Bars (ETA, Speed, Sparklines, Multi-Segment)

**Difficulty:** 🟡 Medium | **Impact:** ⚡ High Impact

**Current State:**
- ✅ Basic ProgressBar exists
- ❓ Unknown features

**How to Enhance:**
1. **Multi-Segment Progress:**
   ```go
   type SegmentedProgress struct {
       Segments []ProgressSegment
       Width    int
   }

   type ProgressSegment struct {
       Value   float64
       Color   RGB
       Label   string
   }
   ```
   - Render like: `[███▓▓▒▒░░] 45%` (different colors for segments)

2. **ETA Calculation:**
   - Track progress over time: `samples []ProgressSample`
   - Calculate velocity: `dProgress / dTime`
   - Extrapolate: `timeRemaining = (1.0 - progress) / velocity`
   - Smooth with moving average

3. **Speed Display:**
   - Track bytes/items processed
   - Format with units: "15.3 MB/s", "1.2k items/s"
   - Show smoothed rate (avoid jitter)

4. **Sparkline:**
   - Mini-graph of speed over time
   - Use Unicode block chars: `▁▂▃▄▅▆▇█`
   - Fit in ~20 columns

5. **Layout:**
   ```
   [████████████░░░░░░░░] 45% | 15.3 MB/s | ETA 2m 15s
   Speed: ▁▂▃▄▅▆▇█▇▆▅▄▃▂
   ```

**Easy Aspects:**
- ETA math is straightforward
- Sparklines are just scaled values
- Multi-segment is colored fill

**Hard Aspects:**
- Smoothing jittery rates
- Handling variable-speed operations (fast start, slow end)
- Layout in narrow terminals
- Unicode block characters (width/alignment)
- Stalled progress detection

**Things to Consider:**
- Indeterminate progress (no known total)
- Circular progress (for narrow spaces)
- Percentage vs absolute counts
- Cancel button integration
- Nested progress (parent + child tasks)

**Overlap:**
- **MultiProgress** - likely already exists
- **Sparklines** - reusable widget
- **Status bar** - shows progress

---

### 21. Data Table (Auto-Sizing, Sorting, Fixed Columns, Excel-Like)

**Difficulty:** 🔴 Hard | **Impact:** ⚡ High Impact

**Current State:**
- ✅ Basic Table exists
- ❓ Unknown features

**How to Enhance:**
1. **Auto-Sizing:**
   - Measure max content width per column
   - Distribute available space proportionally
   - Min/max column widths
   - Weighted columns (some grow more than others)

2. **Sorting:**
   - Click column header to sort
   - Multiple sort keys (Shift+click)
   - Sort indicators: `▲▼`
   - Custom sort functions per column type

3. **Fixed Columns:**
   - Left-freeze: first N columns stay visible on horizontal scroll
   - Right-freeze: last M columns (rare but useful)
   - Render in separate SubFrames with clipping

4. **Excel-Like Selection:**
   - Cell selection (highlight single cell)
   - Row selection (highlight entire row)
   - Column selection (highlight entire column)
   - Range selection (drag or Shift+arrows)
   - Copy to clipboard (CSV or TSV format)

5. **Virtualization:**
   - Render only visible rows and columns
   - Smooth scrolling
   - Scroll bars with position indicators

**Easy Aspects:**
- Sorting is standard library
- Cell rendering is existing infrastructure
- SubFrame for frozen columns

**Hard Aspects:**
- Auto-sizing with wrapping text
- Performance for large tables (100k+ rows)
- Selection rendering (highlighting across scrolling)
- Clipboard interaction (OSC 52)
- Fixed columns with horizontal scroll (coordinate math)

**Things to Consider:**
- CSV import/export
- Filtering (show rows matching criteria)
- Grouping (collapse rows by category)
- Inline editing (click to edit cell)
- Resizable columns (drag header border)
- Conditional formatting (color cells by value)

**Overlap:**
- **Virtualized rendering** - essential for large tables
- **Mouse support** - click to select, drag to resize
- **Clipboard** - copy selected cells
- **Sorting** - reusable for lists

**Recommendation:** Start with sorting + virtualization, defer frozen columns and inline editing.

---

### 22. Floating Windows / Modals / Popovers / Context Menus

**Difficulty:** 🟡 Medium | **Impact:** ⚡ High Impact

**Current State:**
- ✅ Modal exists
- ❓ Unknown if z-ordering supported

**How to Implement:**
1. **Z-Index System:**
   - Assign depth to each widget
   - Render in order: low to high
   - Higher z-index occludes lower

2. **Floating Container:**
   ```go
   type FloatingWidget struct {
       Content  Widget
       X, Y     int     // Absolute or relative
       Width, Height int
       ZIndex   int
       Shadow   bool    // Drop shadow effect
       Closable bool    // Show X button
   }
   ```

3. **Modal Behavior:**
   - Dim background (overlay with semi-transparent fill)
   - Block input to widgets below
   - Escape or click outside to close

4. **Popover:**
   - Attach to parent widget (relative positioning)
   - Auto-position to stay on screen (flip if near edge)
   - Arrow pointing to parent

5. **Context Menu:**
   - Right-click or keyboard shortcut to open
   - Render at mouse position
   - Dismiss on click away or Escape

**Easy Aspects:**
- Z-ordering is simple sorting
- SubFrame handles clipping
- Input blocking is flag check

**Hard Aspects:**
- Shadow rendering (pseudo-transparency with darker colors)
- Auto-positioning (collision detection with screen edges)
- Focus management (tab order across layers)
- Animation (slide in, fade in)
- Nested modals (modal opens another modal)

**Things to Consider:**
- Accessibility (trap focus in modal, announce to screen readers)
- Multiple modals (stack or queue?)
- Portal behavior (render outside parent bounds)
- Resize handling (re-center on terminal resize)

**Overlap:**
- **Modal** - specific type of floating window
- **Focus management** - layer-aware tabbing
- **Animation** - slide/fade effects

---

### 23. Toast / Notification System

**Difficulty:** 🟢 Easy | **Impact:** ⚡ High Impact

**Current State:**
- No notification system

**How to Implement:**
1. **Toast Manager:**
   ```go
   type ToastManager struct {
       toasts    []Toast
       maxToasts int // Limit visible toasts
       position  ToastPosition // TopRight, BottomRight, etc.
   }

   type Toast struct {
       Message  string
       Level    ToastLevel // Info, Success, Warning, Error
       Duration time.Duration
       Icon     string
       created  time.Time
   }
   ```

2. **Rendering:**
   - Stack toasts in corner
   - Newest on top or bottom (configurable)
   - Fade in/out animation
   - Auto-dismiss after duration
   - Hover to pause dismiss timer (if mouse supported)

3. **Interactions:**
   - Click X to dismiss
   - Click toast for details (optional)
   - Swipe to dismiss (if terminal supports gestures)

**Easy Aspects:**
- Simple queue management
- Rendering is just styled text in fixed position
- Auto-dismiss is timer-based

**Hard Aspects:**
- Stacking animation (smooth slide)
- Graceful handling when too many toasts
- Persistent toasts (don't auto-dismiss)
- Click handler in corner (doesn't block main UI)

**Things to Consider:**
- Max visible toasts (overflow handling)
- History view (show dismissed toasts)
- Sound effects (OSC 777 bell)
- Progress toast (shows progress bar)
- Action buttons (e.g., "Undo" for delete notification)

**Overlap:**
- **Floating windows** - toasts are positioned floating elements
- **Animation** - slide and fade
- **Sound** - notification bells

**Recommendation:** Implement - very common UX pattern, relatively simple.

---

### 24. Pager Widget (Like less)

**Difficulty:** 🟡 Medium | **Impact:** ⚡ High Impact

**Current State:**
- No pager widget

**How to Implement:**
1. **Pager Component:**
   ```go
   type Pager struct {
       Content       []string // Lines of text
       ScrollOffset  int
       SearchTerm    string
       SearchMatches []int // Line numbers with matches
       CurrentMatch  int
       WrapLines     bool
       LineNumbers   bool
   }
   ```

2. **Features:**
   - Scroll: j/k, up/down arrows, page up/down, home/end
   - Search: `/` to search, `n` next match, `N` previous
   - Jump: `g` to top, `G` to bottom, `50G` to line 50
   - Highlight search matches
   - Follow mode (tail -f behavior)

3. **Rendering:**
   - Virtualized (only render visible lines)
   - Line numbers in gutter
   - Scrollbar indicator
   - Status line: "Line 50/1000 (5%) search term (3 matches)"

**Easy Aspects:**
- Text scrolling is straightforward
- Search is string matching (or regex)
- Virtualization reuses existing patterns

**Hard Aspects:**
- Regex search (performance on large files)
- Word wrapping (line becomes multiple display lines)
- ANSI code preservation (colored logs)
- Follow mode (watch file for changes)
- Binary file handling (hex view?)

**Things to Consider:**
- Syntax highlighting (code files)
- Markdown rendering (pretty-printed docs)
- Column alignment (CSV/TSV viewing)
- Horizontal scroll (wide lines)
- Bookmarks (mark line, jump to mark)

**Overlap:**
- **Virtualized rendering** - essential for large files
- **Markdown renderer** - pretty docs
- **Syntax highlighting** - code files
- **Search** - reusable fuzzy/regex search

---

## Concurrency & Async

### 25. First-Class Async Widget Support

**Difficulty:** 🟡 Medium | **Impact:** 🔧 Foundation

**Current State:**
- Animator runs goroutines
- Unclear how widgets integrate with async

**How to Implement:**
1. **Async Widget Interface:**
   ```go
   type AsyncWidget interface {
       Widget
       Start(ctx context.Context) error
       Stop() error
       Update(data interface{}) // Receive async updates
   }
   ```

2. **Widget Lifecycle:**
   - `Start()` spawns goroutine(s)
   - Goroutine sends updates to channel
   - Main loop receives and calls `Update()`
   - `Stop()` cancels context and waits for cleanup

3. **Example - Live CPU Monitor:**
   ```go
   type CPUMonitor struct {
       usage    float64
       updateCh chan float64
       stopCh   chan struct{}
   }

   func (c *CPUMonitor) Start(ctx context.Context) error {
       go func() {
           ticker := time.NewTicker(time.Second)
           for {
               select {
               case <-ticker.C:
                   c.updateCh <- readCPU()
               case <-ctx.Done():
                   return
               }
           }
       }()
   }
   ```

4. **Integration:**
   - Screen manages widget lifecycle
   - Event loop receives from all widget channels (select)
   - Batches updates (avoid redraw per channel message)

**Easy Aspects:**
- Goroutines and channels are Go primitives
- Context for cancellation is standard
- Widget interface extension is straightforward

**Hard Aspects:**
- Coordinating many goroutines (resource limits)
- Avoiding redraw storms (rate limiting)
- Error handling in background tasks
- Testing async behavior (race detector, timeouts)

**Things to Consider:**
- Backpressure (widget produces faster than UI can consume)
- Pause/resume (stop updates when widget not visible)
- Resource cleanup (ensure goroutines exit)
- Debugging (goroutine leaks, deadlocks)

**Overlap:**
- **Task runner** - manages background tasks
- **Animator** - already async, can be generalized
- **Live data widgets** - CPU, network, logs

---

### 26. Task Runner with Progress

**Difficulty:** 🟡 Medium | **Impact:** ⚡ High Impact

**Current State:**
- No task management system

**How to Implement:**
1. **Task API:**
   ```go
   type Task struct {
       Name     string
       Run      func(ctx context.Context, progress Progress) error
       OnDone   func(result interface{}, err error)
   }

   type Progress interface {
       SetTotal(total int64)
       SetCurrent(current int64)
       SetStatus(status string)
   }

   type TaskRunner struct {
       tasks   []*Task
       results map[*Task]TaskResult
       ui      *TaskUI // Progress bars, spinners, etc.
   }
   ```

2. **Execution:**
   - Sequential or parallel (worker pool)
   - Progress reporting via channels
   - Cancellation via context
   - Result aggregation

3. **UI Integration:**
   - Show spinner for active task
   - Progress bar if task reports total
   - Update status message
   - Show results (success/failure)

**Easy Aspects:**
- Worker pool pattern is well-known
- Progress reporting is channel-based
- Existing progress widgets

**Hard Aspects:**
- Dependency graphs (task B depends on task A)
- Error handling (continue vs abort)
- Retry logic (exponential backoff)
- Resource limits (max concurrent tasks)

**Things to Consider:**
- Streaming output (show command stdout)
- Log aggregation (collect logs from all tasks)
- Time estimates (predict remaining time)
- Pause/resume/cancel individual tasks

**Overlap:**
- **Async widgets** - tasks are async operations
- **Progress bars** - visual feedback
- **MultiProgress** - show many tasks

---

### 27. Automatic Redraw Rate Limiting (60 FPS Max)

**Difficulty:** 🟢 Easy | **Impact:** 🔧 Foundation

**Current State:**
- ✅ ScreenManager has `minDrawInterval` (50ms = 20 FPS max)
- ✅ Animator uses ticker for FPS control

**How to Enhance:**
1. **Adaptive FPS:**
   - Detect terminal speed (measure time to render frame)
   - If slow, reduce FPS automatically
   - If fast, increase up to 60 FPS

2. **Frame Skipping:**
   - If update takes longer than frame time, skip next frame
   - Prevents backlog of pending renders

3. **VSync-Like Behavior:**
   - Align renders with terminal refresh (if detectable)
   - Probably not practical for terminals

**Easy Aspects:**
- Ticker-based throttling already exists
- Measuring render time is simple

**Hard Aspects:**
- Detecting terminal refresh rate (not exposed)
- Balancing smoothness vs CPU usage
- Different FPS for different widgets (priority system)

**Things to Consider:**
- Battery usage (lower FPS on battery)
- Focus state (reduce FPS when not focused - but can't detect)
- User preference (let user set max FPS)

**Recommendation:** Current system is good, add adaptive FPS as optional enhancement.

---

## Accessibility & UX

### 28. Screen Reader Support (Terminal Accessibility Protocol)

**Difficulty:** 🔴 Hard | **Impact:** ⚡ High Impact

**Current State:**
- No accessibility features

**How to Implement:**
1. **Protocols:**
   - iTerm2: Custom escape codes for semantic regions
   - WezTerm: Accessibility API (proprietary)
   - Generic: ANSI positioning as fallback (screen readers use)

2. **Semantic Markup:**
   - Mark regions as heading, button, input, etc.
   - Announce changes (e.g., "Loading complete")
   - Describe visuals (alt text for images)

3. **Focus Management:**
   - Clear focus ring (visible indicator)
   - Tab order (logical navigation)
   - Skip links (jump to main content)

4. **Announcements:**
   - OSC codes for live regions (if supported)
   - Fallback: status line at bottom

**Easy Aspects:**
- Focus ring is visual styling
- Tab order is widget tree traversal
- Some terminals support OSC announcements

**Hard Aspects:**
- Terminal support is spotty
- Testing without screen reader knowledge
- Balancing visual vs accessible (sometimes conflict)
- Describing complex visuals (charts, graphs)

**Things to Consider:**
- Keyboard-only navigation (no mouse required)
- High-contrast mode (separate concern)
- Reduced motion (disable animations)
- Screen reader testing (NVDA, JAWS, VoiceOver)

**Overlap:**
- **Focus management** - visible focus rings
- **Keyboard navigation** - full keyboard control
- **Announcements** - toast notifications

**Recommendation:** Defer until user request - complex and specialized.

---

### 29. Focus Ring / Visible Keyboard Focus

**Difficulty:** 🟢 Easy | **Impact:** ⚡ High Impact

**Current State:**
- Unknown if focus indicators exist

**How to Implement:**
1. **Focus State:**
   ```go
   type FocusManager struct {
       widgets      []Widget
       focusedIndex int
   }

   func (fm *FocusManager) Next()
   func (fm *FocusManager) Previous()
   func (fm *FocusManager) SetFocus(widget Widget)
   ```

2. **Visual Indicator:**
   - Border change (e.g., different color or thickness)
   - Background highlight
   - Prefix/suffix markers (e.g., `> Button <`)

3. **Navigation:**
   - Tab to next widget
   - Shift+Tab to previous
   - Arrow keys within widget
   - Mouse click to focus

**Easy Aspects:**
- Simple state tracking
- Visual styling is straightforward
- Tab order is widget list

**Hard Aspects:**
- Focus within complex widgets (table cell, tree node)
- Modal focus trapping (can't tab out)
- Skipping non-focusable widgets
- Programmatic focus (auto-focus on error)

**Things to Consider:**
- Focus restoration (remember focus on page change)
- Initial focus (which widget on startup)
- Disabled widgets (skip in tab order)
- Visibility (scroll to keep focused widget visible)

**Overlap:**
- **Keyboard navigation** - uses focus system
- **Accessibility** - screen readers need focus info
- **Modal** - focus trapping

**Recommendation:** Implement early - critical for keyboard users.

---

### 30. High-Contrast and Color-Blind Themes

**Difficulty:** 🟢 Easy | **Impact:** 🔧 Foundation

**Current State:**
- Theming unclear

**How to Implement:**
1. **Theme System:**
   ```go
   type Theme struct {
       Name       string
       Background RGB
       Foreground RGB
       Primary    RGB
       Secondary  RGB
       // ... more semantic colors
       HighContrast bool
   }
   ```

2. **Color-Blind Modes:**
   - Deuteranopia (green-blind): avoid red-green
   - Protanopia (red-blind): avoid red-green
   - Tritanopia (blue-blind): avoid blue-yellow
   - Use color simulation library to test

3. **High-Contrast:**
   - Ensure all colors meet WCAG AAA (7:1 ratio)
   - Increase border thickness
   - Remove subtle shading

**Easy Aspects:**
- Palette swapping is simple
- Contrast checking is math (covered earlier)

**Hard Aspects:**
- Designing accessible palettes (requires expertise)
- Supporting user-uploaded themes (validation)
- Testing with actual color-blind users

**Things to Consider:**
- System preference detection (macOS high-contrast mode)
- User override (manual theme selection)
- Previewing themes (live switch)

**Overlap:**
- **Theming** - part of theme system
- **Contrast checking** - validation

---

### 31. Sound Effects (OSC 777 Bell)

**Difficulty:** 🟢 Easy | **Impact:** Low

**Current State:**
- No sound support

**How to Implement:**
1. **Bell API:**
   - Standard bell: `\007` (BEL character)
   - OSC 777: `\033]777;notify;Title;Message\007` (iTerm2)
   - Play sound file: Some terminals support OSC for audio

2. **Sound Manager:**
   ```go
   type SoundManager struct {
       enabled bool
       volume  float64
   }

   func (sm *SoundManager) Beep()
   func (sm *SoundManager) Notify(title, message string)
   func (sm *SoundManager) PlaySound(name string) // If terminal supports
   ```

**Easy Aspects:**
- Escape codes are simple
- User can disable easily

**Hard Aspects:**
- Terminal support varies wildly
- Custom sounds rarely supported
- Volume control (not exposed)
- Annoying if overused

**Things to Consider:**
- Accessibility (some users need sound cues)
- User preference (mute option)
- Frequency limits (don't spam beeps)

**Recommendation:** Low priority - niche use case.

---

## Theming & Customization

### 32. CSS-Like Theming System

**Difficulty:** 🔴 Hard | **Impact:** 🔧 Foundation

**Current State:**
- Manual styling per widget

**How to Implement:**
1. **Style Rules:**
   ```go
   type Stylesheet struct {
       rules []StyleRule
   }

   type StyleRule struct {
       Selector Selector // e.g., "Button.primary", "#myButton"
       Style    Style
   }

   type Selector struct {
       Type  string // Widget type
       ID    string
       Class []string
   }
   ```

2. **Selectors:**
   - Type: `Button { foreground: blue }`
   - Class: `.primary { bold: true }`
   - ID: `#submitButton { background: green }`
   - Pseudo: `:hover { background: lightblue }`
   - Combinators: `Modal Button { ... }` (descendants)

3. **Cascading:**
   - Specificity: ID > Class > Type
   - Merge styles (more specific overrides)
   - Inheritance (children inherit parent styles)

4. **Variables:**
   - Define: `--primary-color: blue`
   - Use: `foreground: var(--primary-color)`
   - Scope: global or per-component

**Easy Aspects:**
- Basic selector matching is straightforward
- Variable substitution is string replacement

**Hard Aspects:**
- Full CSS specificity algorithm is complex
- Pseudo-classes (:hover, :focus, :active)
- Inheritance vs override semantics
- Performance (style resolution per widget per frame)

**Things to Consider:**
- Parsing format (CSS syntax? YAML? Go structs?)
- Hot-reload (edit theme, see changes live)
- Validation (catch typos, invalid colors)
- Editor support (autocomplete, linting)

**Overlap:**
- **Theming** - this IS the theme system
- **Variables** - part of this system
- **Dark/light mode** - theme variants

**Recommendation:** Start simple (named themes with palettes), defer CSS-like complexity.

---

### 33. Built-In Beautiful Themes (Dracula, Nord, etc.)

**Difficulty:** 🟢 Easy | **Impact:** ⚡ High Impact

**Current State:**
- Unknown if themes exist

**How to Implement:**
1. **Theme Definitions:**
   - Port popular themes:
     - Dracula (purple/pink)
     - Nord (blue/white)
     - OneDark (Atom)
     - Solarized (light/dark)
     - Gruvbox (warm)
     - Catppuccin (pastel)
   - Define as color palettes

2. **Application:**
   - Map semantic colors: `Primary -> Dracula.Purple`
   - Update all widgets
   - Save user preference

**Easy Aspects:**
- Color values are just RGB constants
- Switching is palette swap
- Many themes have published hex codes

**Hard Aspects:**
- Adapting web themes to terminal constraints
- Ensuring accessibility (contrast)
- Handling user terminal theme conflicts

**Things to Consider:**
- Light vs dark variants
- User customization (tweak colors)
- Preview before applying
- Sync with editor theme (if detectable)

**Overlap:**
- **Theming system** - uses this infrastructure
- **Color downgrade** - themes must work in 256-color

**Recommendation:** High value for relatively low effort - bundle 5-10 themes.

---

### 34. Animated Theme Transitions

**Difficulty:** 🟡 Medium | **Impact:** Low

**Current State:**
- Instant theme change (assumed)

**How to Implement:**
1. **Interpolation:**
   - Lerp between old and new colors
   - Animate over N frames (e.g., 30 frames = 1 second at 30 FPS)
   - Update all widget styles each frame

2. **Easing:**
   - Use easing function (ease-in-out is pleasant)
   - Apply to color interpolation

**Easy Aspects:**
- Color interpolation is simple math
- Animation infrastructure exists

**Hard Aspects:**
- Performance (updating all widgets every frame)
- Distraction (is animation worth it?)
- Interruption (user switches theme mid-animation)

**Things to Consider:**
- User preference (instant vs animated)
- Reduced motion (accessibility)

**Recommendation:** Low priority - flashy but low utility.

---

## Integration & Environment

### 35. Auto-Detect JetBrains/VS Code Terminal

**Difficulty:** 🟢 Easy | **Impact:** 🔧 Foundation

**Current State:**
- Unknown

**How to Implement:**
1. **Environment Variables:**
   - VS Code: `$TERM_PROGRAM = vscode`
   - JetBrains: `$TERMINAL_EMULATOR = JetBrains-*`
   - iTerm2: `$TERM_PROGRAM = iTerm.app`

2. **Adjustments:**
   - VS Code: May not support images, limit mouse
   - JetBrains: Similar to VS Code
   - SSH: Disable features (detect via `$SSH_CONNECTION`)

3. **Feature Flags:**
   ```go
   type TerminalCapabilities struct {
       InlineImages bool
       TrueColor    bool
       Mouse        bool
       HyperlinkOSC bool
   }
   ```

**Easy Aspects:**
- Env vars are easy to check
- Feature flags are simple booleans

**Hard Aspects:**
- Keeping up with new terminals
- False positives (user sets env vars)
- Graceful degradation

**Things to Consider:**
- User override (force enable/disable)
- Logging detected environment (debugging)

**Overlap:**
- **Color downgrade** - uses capabilities
- **Image support** - checks if supported

---

### 36. Clipboard Integration (OSC 52 + Fallback)

**Difficulty:** 🟡 Medium | **Impact:** ⚡ High Impact

**Current State:**
- No clipboard support

**How to Implement:**
1. **OSC 52 (Universal):**
   - Copy: `\033]52;c;BASE64\007`
   - Works in most modern terminals (even over SSH)
   - Supported: iTerm2, WezTerm, tmux, etc.

2. **Fallback Commands:**
   - macOS: `pbcopy` / `pbpaste`
   - Linux (X11): `xclip` / `xsel`
   - Linux (Wayland): `wl-copy` / `wl-paste`
   - Windows: `clip.exe` / PowerShell

3. **API:**
   ```go
   type Clipboard interface {
       Copy(text string) error
       Paste() (string, error)
   }
   ```

**Easy Aspects:**
- OSC 52 is simple escape code
- Shelling out to commands is straightforward

**Hard Aspects:**
- OSC 52 disabled in some terminals (security)
- Detecting which fallback command exists
- Binary clipboard (images, not just text)
- Permissions (clipboard access may require approval)

**Things to Consider:**
- Size limits (OSC 52 has max length in some terminals)
- Security (malicious paste)
- User notification (show "Copied!" toast)

**Overlap:**
- **Table** - copy selected cells
- **Text fields** - Ctrl+C/V
- **Security** - sanitize pasted content

---

### 37. Auto-Detect SSH and Downgrade Features

**Difficulty:** 🟢 Easy | **Impact:** 🔧 Foundation

**Current State:**
- Unknown

**How to Implement:**
1. **SSH Detection:**
   - Check `$SSH_CONNECTION` env var
   - Check `$SSH_TTY`
   - Check if stdin is socket: `stat /proc/self/fd/0`

2. **Downgrades:**
   - Disable inline images (bandwidth)
   - Reduce animation FPS (latency)
   - Limit mouse events (may not forward)
   - Use OSC 52 for clipboard (works over SSH)

3. **User Override:**
   - Allow forcing features on/off
   - Detect fast SSH (LAN vs WAN)

**Easy Aspects:**
- Env var check is trivial
- Feature flags already defined

**Hard Aspects:**
- Detecting connection speed (LAN SSH is fast)
- mosh vs ssh (different capabilities)
- Nested SSH (ssh -> ssh -> app)

**Overlap:**
- **Environment detection** - same system
- **Performance** - adaptive based on connection

---

### 38. Zero-Config Live Reload for Development

**Difficulty:** 🟡 Medium | **Impact:** 🔧 Foundation (Dev Only)

**Current State:**
- Manual restart required

**How to Implement:**
1. **File Watcher:**
   - Use `github.com/fsnotify/fsnotify`
   - Watch `*.go` files in project
   - Debounce changes (wait 100ms after last change)

2. **Rebuild & Restart:**
   - Run `go build` on change
   - If success, kill old process, start new
   - Preserve terminal state (or clear)

3. **State Preservation:**
   - Serialize app state to file
   - Restore on restart
   - Or just restart clean (simpler)

**Easy Aspects:**
- File watching libraries exist
- Running go build is simple
- Killing/restarting is standard

**Hard Aspects:**
- Preserving UI state (cursor position, scroll, etc.)
- Handling compile errors gracefully
- Race conditions (user input during restart)

**Things to Consider:**
- Env var to enable (e.g., `GOOEY_DEV=1`)
- Logging (show rebuild messages)
- Integration with go run (vs compiled binary)

**Recommendation:** Nice dev experience, medium effort.

---

## Advanced & Rare Features

### 39. Session Recording / Playback (Embedded Asciinema)

**Difficulty:** 🔴 Hard | **Impact:** Low (Niche)

**Current State:**
- No recording

**How to Implement:**
1. **Recording:**
   - Capture all terminal output (ANSI codes)
   - Capture input events (keyboard, mouse)
   - Timestamp each event
   - Save as asciinema v2 format (JSON)

2. **Playback:**
   - Parse recording file
   - Replay events with timing
   - Controls: play, pause, seek, speed

3. **Integration:**
   ```go
   func (t *Terminal) StartRecording(filename string)
   func (t *Terminal) StopRecording()
   func PlayRecording(filename string, terminal *Terminal)
   ```

**Easy Aspects:**
- Format is well-documented (asciinema)
- Capturing output is just logging

**Hard Aspects:**
- File size (long sessions are large)
- Compression (gzip recordings)
- Synchronization (timing replay correctly)
- Editing recordings (cut, trim, annotate)

**Things to Consider:**
- Privacy (recordings may contain secrets)
- Sharing (upload to asciinema.org)
- Embedding (show recording in docs)

**Recommendation:** Niche, defer unless user requests.

---

### 40. Remote Control Mode (Control TUI over SSH/WebSocket)

**Difficulty:** 🔴 Hard | **Impact:** Low (Niche)

**Current State:**
- Local only

**How to Implement:**
1. **Server Mode:**
   - TUI runs as server
   - Accept connections (WebSocket or raw TCP)
   - Send screen updates to client
   - Receive input from client

2. **Protocol:**
   - JSON messages: `{ type: "render", data: [...cells] }`
   - Input: `{ type: "key", key: "a" }`
   - Similar to VNC/RDP but text-based

3. **Multi-Client:**
   - Broadcast updates to all clients
   - Arbitrate input (first-come or round-robin)

**Easy Aspects:**
- WebSocket libraries exist
- Screen state is already buffered

**Hard Aspects:**
- Latency (remote rendering feels slow)
- Security (authentication, encryption)
- Bandwidth (full screen updates are large)
- Synchronization (clients may be out of sync)

**Things to Consider:**
- Use case (why not just SSH?)
- Web-based client (HTML5 terminal)
- Collaboration (multiple users editing)

**Recommendation:** Very niche, only if specific need arises.

---

### 41. Built-in TUIs for Common Tasks

**Difficulty:** Varies | **Impact:** ⚡ High Impact (If Implemented)

**Examples:**
- **Git Log Browser:** Navigate commits, view diffs, search
- **Process Explorer:** List processes, CPU/memory usage, kill
- **Log Tail:** `tail -f` with highlighting, filtering
- **JSON Explorer:** Expand/collapse, search, pretty-print
- **HTTP Client:** Make requests, view responses (Postman-like)

**How to Implement:**
Each is a separate application built with Gooey:
- Use widgets: Tree, Table, TextArea, etc.
- Custom logic for git/process/JSON parsing
- Ship as examples or separate binaries

**Easy Aspects:**
- Demonstrates library capabilities
- Reuses existing widgets

**Hard Aspects:**
- Each is a substantial project
- Maintenance burden
- May diverge from core library focus

**Things to Consider:**
- Ship as examples (not main library)
- Community contributions (accept PRs)
- Inspiration from existing tools (lazygit, btop)

**Recommendation:** Build 1-2 as flagship examples, leave rest to community.

---

## Priority Matrix

| Feature                  | Difficulty | Impact | Priority               |
| ------------------------ | ---------- | ------ | ---------------------- |
| **Layout & Rendering**   |
| Flex/Grid Layout         | 🔴 Hard     | ⚡🔧     | **HIGH**               |
| Fractional Layout        | 🟡 Medium   | ⚡      | **HIGH**               |
| Split-Pane               | 🟡 Medium   | ⚡      | **MEDIUM**             |
| Virtualized Lists        | 🟡 Medium   | ⚡      | **HIGH**               |
| Dirty Region (Enhanced)  | 🟢 Easy     | 🔧      | ✅ **DONE** (Profiling) |
| **Input & Interaction**  |
| Full Mouse Support       | 🟡 Medium   | ⚡      | **HIGH**               |
| Mouse Protocol Detection | 🟢 Easy     | 🔧      | **MEDIUM**             |
| Bracketed Paste          | 🟢 Easy     | ⚡      | ✅ **DONE** (Jan 2025)  |
| IME Support              | 🔴 Hard     | ⚡      | **LOW**                |
| Secure Password Input    | 🟢 Easy     | 🔧      | ✅ **DONE** (Nov 2025)  |
| Inline Images            | 🔴 Hard     | ⚡      | **MEDIUM**             |
| **Text & Styling**       |
| Color Downgrade          | 🟡 Medium   | ⚡      | **MEDIUM**             |
| Contrast Checker         | 🟢 Easy     | 🔧      | **HIGH**               |
| Hyperlinks (OSC 8)       | 🟢 Easy     | ⚡      | ✅ **DONE** (Jan 2025)  |
| Grapheme Clusters        | 🔴 Hard     | 🔧      | **MEDIUM**             |
| Markdown Renderer        | 🟡 Medium   | ⚡      | **HIGH**               |
| Animated Text (Enhanced) | 🟢 Easy     | ⚡      | **LOW**                |
| **Widgets**              |
| File Picker              | 🟡 Medium   | ⚡      | **MEDIUM**             |
| Tree View                | 🟡 Medium   | ⚡      | **HIGH**               |
| Advanced Progress        | 🟡 Medium   | ⚡      | **MEDIUM**             |
| Advanced Table           | 🔴 Hard     | ⚡      | **HIGH**               |
| Floating Windows         | 🟡 Medium   | ⚡      | **MEDIUM**             |
| Toast Notifications      | 🟢 Easy     | ⚡      | **HIGH**               |
| Pager Widget             | 🟡 Medium   | ⚡      | **MEDIUM**             |
| **Concurrency**          |
| Async Widgets            | 🟡 Medium   | 🔧      | **MEDIUM**             |
| Task Runner              | 🟡 Medium   | ⚡      | **MEDIUM**             |
| FPS Rate Limiting        | 🟢 Easy     | 🔧      | **LOW** (Exists)       |
| **Accessibility**        |
| Screen Reader            | 🔴 Hard     | ⚡      | **LOW**                |
| Focus Indicators         | 🟢 Easy     | ⚡      | **HIGH**               |
| High-Contrast Themes     | 🟢 Easy     | 🔧      | **HIGH**               |
| Sound Effects            | 🟢 Easy     | Low    | **LOW**                |
| **Theming**              |
| CSS-Like System          | 🔴 Hard     | 🔧      | **LOW** (Start Simple) |
| Built-In Themes          | 🟢 Easy     | ⚡      | **HIGH**               |
| Animated Transitions     | 🟡 Medium   | Low    | **LOW**                |
| **Integration**          |
| Environment Detection    | 🟢 Easy     | 🔧      | **HIGH**               |
| Clipboard (OSC 52)       | 🟡 Medium   | ⚡      | **HIGH**               |
| SSH Detection            | 🟢 Easy     | 🔧      | **MEDIUM**             |
| Live Reload              | 🟡 Medium   | 🔧      | **LOW** (Dev Tool)     |
| **Advanced**             |
| Session Recording        | 🔴 Hard     | Low    | **LOW**                |
| Remote Control           | 🔴 Hard     | Low    | **LOW**                |
| Example TUIs             | Varies     | ⚡      | **MEDIUM**             |

---

## Recommended Roadmap

### Phase 1: Foundation (0-3 months)
**Goal:** Core improvements that enable other features

1. ✅ **Bracketed Paste** - Immediate win (Completed Jan 2025)
2. ✅ **Focus Indicators** - Essential UX
3. ✅ **Toast Notifications** - Common pattern
4. ✅ **Hyperlink Support (OSC 8)** - Easy + high value (Completed Jan 2025)
5. ✅ **Performance Metrics/Profiling** - Developer visibility (Completed Jan 2025)
6. 🔧 **Environment Detection** - Enables smart defaults
7. 🔧 **Contrast Checker** - Accessibility foundation
8. 🔧 **Clipboard (OSC 52)** - Highly requested

### Phase 2: Layout & Widgets (3-6 months)
**Goal:** Make complex UIs possible

1. 🎯 **Virtualized Lists** - Performance for large data
2. 🎯 **Tree View** - Many use cases (files, data, navigation)
3. 🎯 **Flex/Box Layout** - Modern layout engine
4. 🎯 **Advanced Table** - Sorting, scrolling, selection
5. 🎯 **Markdown Renderer** - Documentation, help text

### Phase 3: Polish (6-9 months)
**Goal:** World-class UX

1. 🎨 **Built-In Themes** - Beautiful out of box
2. 🎨 **High-Contrast Mode** - Accessibility
3. 🎨 **Split-Pane** - Power user favorite
4. 🎨 **File Picker** - Common need
5. 🎨 **Pager Widget** - Viewing logs, docs

### Phase 4: Advanced (9-12 months)
**Goal:** Differentiation from competitors

1. 🚀 **Inline Images** - Unique feature
2. 🚀 **Async Widgets** - Live data
3. 🚀 **Task Runner** - Background operations
4. 🚀 **Example TUIs** - Showcase capabilities
5. 🚀 **Grapheme Clusters** - True Unicode support

### Defer / Community-Driven
- IME Support (complex, niche)
- Screen Reader (needs specialized expertise)
- CSS-Like Theming (complex, start with simple themes)
- Session Recording (niche)
- Remote Control (very niche)

---

## Overlap Map

Features that share implementation:

**Layout Engine:**
- Flex/Grid Layout
- Fractional Layout
- Split-Pane
- Container Widgets

**Search & Filter:**
- File Picker (fuzzy search)
- Tree View (search)
- Pager (regex search)
- Table (column filtering)

**Virtualization:**
- Lists
- Tables
- Trees
- Pager (large files)

**Mouse System:**
- Full mouse events
- Drag-and-drop
- Split-pane resize
- Click-to-focus

**Environment Detection:**
- Color downgrade
- Image support
- Hyperlink support
- SSH detection
- Clipboard method

**Rich Content:**
- Markdown renderer
- Syntax highlighting
- Hyperlinks
- Inline images

**Accessibility:**
- Focus indicators
- High-contrast themes
- Screen reader
- Reduced motion
