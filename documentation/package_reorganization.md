# Package Reorganization Analysis

**Date:** 2025-11-22
**Status:** Proposal

## Executive Summary

The gooey root package has grown to **33 production Go files** (~17K lines of code). This document analyzes the current structure and proposes subpackage reorganization options to improve code organization, maintainability, and API clarity.

**Recommended Approach:** Three-package split (`render/`, `input/`, and root).

---

## Current State Analysis

### File Count and Size

The root package contains:
- **33 production `.go` files** (excluding tests)
- **~17,072 lines of code**
- Largest files:
  - `terminal.go` (36,830 bytes)
  - `flex_layout.go` (15,166 bytes)
  - `mouse.go` (15,846 bytes)
  - `spinner.go` (14,336 bytes)

### Functional Categories

#### 1. Core Rendering Engine (4 files)
Low-level terminal control and double-buffered rendering foundation.

- `terminal.go` - Main terminal class, buffer management, ANSI escape sequences
- `frame.go` - RenderFrame interface, SubFrame creation, atomic operations
- `style.go` - Style struct, colors, RGB support, border styles
- `metrics.go` - Performance profiling and render statistics

#### 2. Composition System (7 files)
Modern bounds-based widget hierarchy with parent-child relationships.

- `composition.go` - Core interfaces (ComposableWidget, LayoutManager)
- `container.go` - Container widget for managing children
- `composable_label.go` - Text label widget
- `composable_button.go` - Button widget
- `layout_managers.go` - VBoxLayout, HBoxLayout
- `flex_layout.go` - CSS Flexbox-style layout
- `grid_layout.go` - Grid layout system

#### 3. Traditional Components (5 files)
Legacy absolute-positioned widgets using X,Y coordinates.

- `components.go` - Button, TabCompleter, RadioGroup
- `checkbox.go` - CheckboxGroup
- `spinner.go` - Spinner, ProgressBar, MultiProgress
- `table.go` - Data table widget
- `modal.go` - Dialog/popup widget

#### 4. Input Handling (6 files)
Multiple keyboard input implementations with key decoding.

- `input.go` - Main Input struct and methods
- `input_enhanced.go` - Enhanced input variant
- `input_fixed.go` - Fixed input variant
- `input_simple.go` - Simple input variant
- `input_interactive.go` - Interactive input variant
- `key_decoder.go` - Low-level key decoding and escape sequence parsing

#### 5. Animation System (2 files)
Frame-based animation engine with configurable FPS.

- `animator.go` - Animation engine and AnimatedElement interface
- `animated_layout.go` - Layout with animation support

#### 6. Mouse Support (1 file)
Advanced mouse event handling.

- `mouse.go` - MouseEvent, MouseRegion, click tracking, modifiers

#### 7. Legacy Screen Management (2 files)
Older region-based screen coordination.

- `screen_manager.go` - Named screen regions with animations
- `layout.go` - Traditional header/footer/content layout

#### 8. Session Recording (3 files)
Record and replay terminal sessions in asciinema format.

- `recording.go` - Recording system with gzip compression
- `playback.go` - Playback controller with speed control
- `recording_privacy.go` - Privacy filter for sensitive data

#### 9. Text and Utilities (3 files)
Specialized text features and helper functions.

- `text.go` - Text wrapping utilities
- `hyperlink.go` - OSC 8 hyperlink support
- `fuzzy.go` - Fuzzy matching for search

### External Dependencies

- `github.com/mattn/go-runewidth` - Unicode display width calculations
- `golang.org/x/term` - Terminal raw mode control
- Standard library: `sync`, `time`, `io`, `os`, `strings`, `fmt`, `image`, etc.

### Internal Dependency Flow

```
Composition System (container, composable_*)
    ↓
Layout Managers (layout_managers, flex_layout, grid_layout)
    ↓
Animation System (animator, animated_layout)
    ↓
Core Rendering (terminal, frame, style)
```

Parallel systems:
```
Traditional Components (components, spinner, table, modal)
    ↓
Screen Manager / Layout (animated_layout, screen_manager)
    ↓
Input Handling (input variants, key_decoder)
    ↓
Core Rendering (terminal, frame, style)
```

---

## Key Observations

### 1. Dual Widget Systems

The codebase maintains two parallel widget architectures:

**Modern (Composition-based):**
- Bounds-relative positioning (`image.Rectangle`)
- Parent-child relationships
- Layout managers for automatic positioning
- Files: `container.go`, `composable_*.go`, layout managers

**Legacy (Absolute-positioned):**
- X, Y coordinate fields
- Direct screen positioning
- Screen manager regions
- Files: `components.go`, `spinner.go`, `modal.go`, `screen_manager.go`

This duality creates some conceptual overhead for users choosing which system to use.

### 2. Input Implementation Fragmentation

Five different input implementation files exist:
- `input.go` (main)
- `input_enhanced.go`
- `input_fixed.go`
- `input_simple.go`
- `input_interactive.go`

This suggests either:
- Multiple iterations over time
- Different use cases requiring different approaches
- Potential consolidation opportunity

### 3. Clean Separation of Concerns

Despite the large number of files, concerns are well-separated:
- Rendering is isolated in core files
- Animation has dedicated files
- Input handling is separate from rendering
- Mouse support is self-contained
- No circular dependencies detected

### 4. Size Distribution

Most files are reasonably sized (4-15KB), with `terminal.go` being the clear outlier at 36KB. This suggests `terminal.go` might benefit from internal refactoring, but that's separate from package organization.

---

## Reorganization Options

### Option 1: Three Packages (Recommended)

```
gooey/
├── render/          # Core rendering primitives
│   ├── terminal.go
│   ├── frame.go
│   ├── style.go
│   └── metrics.go
│
├── input/           # All input handling
│   ├── input.go
│   ├── input_enhanced.go
│   ├── input_fixed.go
│   ├── input_simple.go
│   ├── input_interactive.go
│   └── key_decoder.go
│
└── [root remains]   # Composition, components, animation, mouse, etc.
    ├── composition.go
    ├── container.go
    ├── composable_*.go
    ├── layout_managers.go
    ├── flex_layout.go
    ├── grid_layout.go
    ├── components.go
    ├── spinner.go
    ├── table.go
    ├── modal.go
    ├── checkbox.go
    ├── animator.go
    ├── animated_layout.go
    ├── layout.go
    ├── screen_manager.go
    ├── mouse.go
    ├── recording.go
    ├── playback.go
    ├── recording_privacy.go
    ├── text.go
    ├── hyperlink.go
    └── fuzzy.go
```

#### Benefits

- **Clear separation of abstraction levels**: `render/` is low-level primitives, `input/` handles user input, root provides high-level widgets
- **Consolidates fragmented input system**: All 6 input-related files in one place
- **Simple user-facing API**: Users still do `tui.NewContainer()`, not `widget.NewContainer()`
- **Minimal breaking changes**: Most user code only imports `gooey`
- **Clear import hierarchy**: widgets → input → render

#### Tradeoffs

- Both widget systems remain in root (composition + traditional components)
- Doesn't fully address the conceptual split between modern and legacy approaches

#### Migration Impact

- **Low**: Most users import only the root package
- Common pattern: `import "github.com/myzie/gooey"` continues to work
- Advanced users accessing `Style` or `Terminal` directly would need: `import "github.com/myzie/gooey/render"`

---

### Option 2: Two Packages (Simpler)

```
gooey/
├── core/            # Foundation
│   ├── terminal.go
│   ├── frame.go
│   ├── style.go
│   └── metrics.go
│
└── [root remains]   # Everything else
```

#### Benefits

- **Minimal disruption**: Single new package
- **Clear primitive/feature split**: Core is what you build with, root is what you build
- **Simple mental model**: Core = rarely imported directly by users

#### Tradeoffs

- Doesn't address input fragmentation
- Doesn't clarify dual widget systems
- Less organizational benefit overall

#### Migration Impact

- **Very Low**: Only affects users directly accessing `Terminal` or `Style`
- Could provide type aliases in root: `type Style = core.Style`

---

### Option 3: Four Packages (Most Organized)

```
gooey/
├── core/            # Rendering primitives
│   ├── terminal.go
│   ├── frame.go
│   ├── style.go
│   └── metrics.go
│
├── input/           # Input handling
│   ├── input.go (+ variants)
│   └── key_decoder.go
│
├── widget/          # Modern composition system
│   ├── composition.go
│   ├── container.go
│   ├── composable_*.go
│   ├── layout_managers.go
│   ├── flex_layout.go
│   └── grid_layout.go
│
└── component/       # Legacy traditional components
    ├── components.go
    ├── spinner.go
    ├── table.go
    ├── modal.go
    └── checkbox.go
```

Additional considerations:
- Where do `animator.go`, `mouse.go` go? (Both widget systems use them)
- Where do `recording.go`, `playback.go` go? (Cross-cutting)
- Could add `recording/` package for session management

#### Benefits

- **Cleanest separation**: Every major concern has its own package
- **Makes dual systems explicit**: `widget/` (modern) vs `component/` (legacy)
- **Future-proof**: New users naturally discover `widget/` as the modern approach
- **Maximum organization**: Each package has clear purpose

#### Tradeoffs

- **More breaking changes**: Users must import multiple packages
- **Cross-cutting concerns unclear**: Where do animation and mouse live?
- **More complex**: Higher cognitive overhead for simple use cases
- **API verbosity**: `widget.NewContainer()` instead of `tui.NewContainer()`

#### Migration Impact

- **High**: Most user code needs updated imports
- Would require careful re-export strategy in root package
- Could provide transition aliases for major version bump

---

### Option 4: Additional Specialty Package (Recording)

Any of the above options could add:

```
recording/
├── recording.go
├── playback.go
└── recording_privacy.go
```

#### Benefits

- Session recording is a specialized, optional feature
- Self-contained with minimal dependencies on rest of codebase
- Clear opt-in for users who need it

#### Tradeoffs

- Only 3 files, might be overkill
- Currently has no dependencies beyond core rendering
- Could just stay in root

---

## Recommendation: Option 1 (Three Packages)

### Rationale

**`render/` package** (4 files)
- Isolates the low-level rendering foundation
- Most users won't need to import this directly
- Clear abstraction boundary
- Easy to test in isolation

**`input/` package** (6 files)
- Desperately needed: 5 input variants are confusing in root
- Natural grouping: all keyboard/input handling together
- Makes it obvious where to look for input features
- Could eventually consolidate the 5 variants into one public API

**Root package** (23 files)
- Keeps high-level API simple and friendly
- Both widget systems accessible as `tui.NewContainer()` or `tui.NewButton()`
- Animation, mouse, recording, utilities all stay together
- Avoids forcing users to choose between `widget/` and `component/`

### Package Purpose Summary

| Package | Purpose | User Visibility | File Count |
|---------|---------|-----------------|------------|
| `render/` | Terminal control, frame rendering, styling | Low - internal use | 4 |
| `input/` | Keyboard input, key decoding | Medium - direct use when needed | 6 |
| `gooey/` (root) | Widgets, layouts, components, features | High - primary API | 23 |

### Import Patterns After Reorganization

**Simple usage (90% of users):**
```go
import "github.com/myzie/gooey"

term, _ := tui.NewTerminal()
container := tui.NewContainer(tui.NewVBoxLayout(2))
```

**Advanced usage (users needing low-level control):**
```go
import (
    "github.com/myzie/gooey"
    "github.com/myzie/gooey/render"
    "github.com/myzie/gooey/input"
)

term, _ := tui.NewTerminal()
style := render.NewStyle().WithFgRGB(render.RGB{255, 0, 0})
decoder := input.NewKeyDecoder(os.Stdin)
```

**Could provide convenience re-exports in root:**
```go
// In gooey/exports.go
type Style = render.Style
type RenderFrame = render.RenderFrame
type KeyEvent = input.KeyEvent

var NewStyle = render.NewStyle
```

This allows most users to continue using `tui.Style` without importing `render/`.

---

## Implementation Plan

### Phase 1: Preparation
1. Ensure all tests pass
2. Document current public API surface
3. Create feature branch for reorganization

### Phase 2: Create New Packages
1. Create `render/` directory
   - Move: `terminal.go`, `frame.go`, `style.go`, `metrics.go`
   - Update package declarations to `package render`
2. Create `input/` directory
   - Move: `input.go`, `input_*.go`, `key_decoder.go`
   - Update package declarations to `package input`

### Phase 3: Update Imports
1. Update all files in root package to import `render` and `input`
2. Update all example files
3. Update test files

### Phase 4: Re-exports (Optional)
1. Create `gooey/exports.go` with type aliases for commonly-used types
2. This maintains backward compatibility for most users

### Phase 5: Documentation
1. Update `CLAUDE.md` with new package structure
2. Update `README.md` with import examples
3. Update `documentation/` guides
4. Create migration guide for existing users

### Phase 6: Testing
1. Run full test suite
2. Run all examples
3. Check for any missed import updates

### Phase 7: Release
1. Tag as new minor or major version (depending on breaking changes)
2. Announce reorganization with migration guide
3. Update examples in README

---

## Alternative Considerations

### Keep Everything in Root

**Pros:**
- No breaking changes
- Simple single import
- Works for many libraries this size

**Cons:**
- 33 files is getting unwieldy
- Hard to navigate for newcomers
- Doesn't address input fragmentation
- Doesn't clarify modern vs legacy widget systems

**Verdict:** Reorganization is worthwhile given current size and dual systems.

### Gradual Migration

Instead of moving files, could:
1. Create new packages with new implementations
2. Keep old files as deprecated wrappers
3. Gradually migrate over multiple releases

**Pros:**
- Zero breaking changes
- Users migrate at their own pace

**Cons:**
- Doubles the codebase temporarily
- Maintenance burden of supporting both
- Doesn't solve current organizational issues

---

## Questions for Consideration

1. **Breaking changes tolerance**: Is this a 1.x → 2.x change, or can we do minor version?
   - If re-exports are added, could be minor version
   - Clean break would be major version

2. **Input consolidation**: ✅ DONE - Input API has been rationalized to three methods:
   - `Read()` - Full-featured input with history, suggestions, cursor editing
   - `ReadPassword()` - Secure password input
   - `ReadSimple()` - Basic line reading
   - Legacy methods removed (see `documentation/input_api_migration.md`)

3. **Recording package**: Should recording/playback get its own package?
   - Probably yes if going with 4-package approach
   - Probably no for 2-3 package approach

4. **Re-export strategy**: How much to re-export from root?
   - Conservative: Only `Style`, `RenderFrame`
   - Generous: Everything users might commonly need
   - None: Clean break, explicit imports

---

## Recommendation Summary

**Implement Option 1: Three packages (`render/`, `input/`, root)**

**Next steps:**
1. Create feature branch
2. Move files to new packages
3. Update imports across codebase
4. Update examples and documentation
5. Add re-exports for `Style`, `RenderFrame`, `KeyEvent` in root
6. Full test pass
7. Tag new version with migration guide

**Expected outcome:**
- Clearer code organization
- Better discoverability for new users
- Foundation for future growth
- Minimal disruption for existing users (with re-exports)

---

## Appendix: Complete File Listing

### Proposed `render/` package (4 files)
- `terminal.go` (36,830 bytes) - Main terminal class
- `frame.go` (8,728 bytes) - RenderFrame interface
- `style.go` (9,402 bytes) - Style, Color, RGB, borders
- `metrics.go` (7,254 bytes) - Performance metrics

### Proposed `input/` package (6 files)
- `input.go` (10,868 bytes) - Main Input struct
- `input_enhanced.go` (9,745 bytes) - Enhanced variant
- `input_fixed.go` (7,681 bytes) - Fixed variant
- `input_simple.go` (4,641 bytes) - Simple variant
- `input_interactive.go` (8,821 bytes) - Interactive variant
- `key_decoder.go` (9,266 bytes) - Key decoding

### Root package (23 files)
- `composition.go` (8,233 bytes)
- `container.go` (10,516 bytes)
- `composable_label.go` (6,062 bytes)
- `composable_button.go` (4,557 bytes)
- `layout_managers.go` (10,996 bytes)
- `flex_layout.go` (15,166 bytes)
- `grid_layout.go` (8,747 bytes)
- `components.go` (10,108 bytes)
- `checkbox.go` (3,038 bytes)
- `spinner.go` (14,336 bytes)
- `table.go` (5,122 bytes)
- `modal.go` (4,520 bytes)
- `animator.go` (11,552 bytes)
- `animated_layout.go` (12,099 bytes)
- `layout.go` (13,007 bytes)
- `screen_manager.go` (7,072 bytes)
- `mouse.go` (15,846 bytes)
- `recording.go` (6,782 bytes)
- `playback.go` (7,440 bytes)
- `recording_privacy.go` (2,038 bytes)
- `text.go` (2,469 bytes)
- `hyperlink.go` (4,515 bytes)
- `fuzzy.go` (937 bytes)
