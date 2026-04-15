# Implementation Plan: pty, runewidth, valuediff

Date: April 12, 2026

This plan captures the refined scope, API improvements, and build order for three
new Wonton packages that replace external dependencies. The goal is to exceed
what's available in the Go ecosystem today.

## Build Order

1. **`pty`** — low risk, quick win, builds confidence
2. **`runewidth`** — highest value for the TUI layer, correctness-sensitive
3. **`valuediff`** — touches test infrastructure, benefits from shipping last

## Package 1: `pty`

**Replaces:** `github.com/creack/pty v1.1.24`
**Current usage:** 3 call sites in `termsession/session.go`

### API (refined from spec)

```go
package pty

var ErrUnsupported = errors.New("pty: unsupported platform")

type Size struct {
    Rows uint16
    Cols uint16
}

type PTY struct { /* master *os.File */ }

// io.ReadWriteCloser
func (p *PTY) Read(b []byte) (int, error)
func (p *PTY) Write(b []byte) (int, error)
func (p *PTY) Close() error          // nil-safe, idempotent

// Accessors
func (p *PTY) Fd() uintptr
func (p *PTY) File() *os.File        // NEW: exposes underlying file for poll/select

// Sizing
func (p *PTY) Resize(size Size) error
func (p *PTY) GetSize() (Size, error)
func (p *PTY) InheritSize(tty *os.File) error  // NEW: copy size from another terminal

// Allocation
func Open() (master *PTY, slave *os.File, err error)
func Start(cmd *exec.Cmd, size *Size) (*PTY, error)
func StartWithAttrs(cmd *exec.Cmd, size *Size, attrs *syscall.SysProcAttr) (*PTY, error)  // NEW
```

### Improvements over `creack/pty`

| Feature | `creack/pty` | Wonton `pty` |
|---------|-------------|--------------|
| Return type | raw `*os.File` | structured `*PTY` |
| Close safety | caller must track | nil-safe, idempotent |
| Resize API | free function with fd | method on PTY |
| Initial sizing | separate `StartWithSize` | built into `Start` |
| Size inheritance | manual | `InheritSize()` one-liner |
| Custom proc attrs | `StartWithAttrs` | `StartWithAttrs` (preserved) |
| Underlying file | it IS the file | `File()` accessor |

### Platform tiers

- **Tier 1 (tested in CI):** Linux, Darwin
- **Tier 2 (compiles, best-effort):** FreeBSD, OpenBSD, NetBSD, DragonFly
- **Tier 3 (stubs):** Windows, other — returns `ErrUnsupported`

### File structure

```
pty/
    pty.go             # Types, shared methods, docs
    pty_unix.go        # Resize, GetSize, InheritSize, Start, StartWithAttrs (shared Unix)
    pty_darwin.go      # open() for Darwin
    pty_linux.go       # open() for Linux
    pty_freebsd.go     # open() for FreeBSD
    pty_openbsd.go     # open() for OpenBSD
    pty_netbsd.go      # open() for NetBSD
    pty_dragonfly.go   # open() for DragonFly
    pty_windows.go     # stubs returning ErrUnsupported
    pty_other.go       # stubs returning ErrUnsupported
    pty_test.go        # tests
```

### Migration

1. Implement `pty` package with tests
2. Update `termsession/session.go` (3 call sites)
3. Remove `github.com/creack/pty` from `go.mod`
4. Run full test suite

### Tests

- `TestOpen` — allocate, write master→slave, read back
- `TestStart` — start `echo hello`, read output, verify exit
- `TestResize` — resize, verify `GetSize()` matches
- `TestInheritSize` — copy from a real terminal fd
- `TestStartWithAttrs` — custom `SysProcAttr` is preserved
- `TestClose_Idempotent` — double close, no panic
- `TestClose_Nil` — methods on nil `*PTY`, no panic
- `TestStart_InvalidCommand` — error, no resource leak
- `TestReadWriteCloser` — compile-time interface check
- Platform skip when `/dev/ptmx` or `/dev/ptm` unavailable

---

## Package 2: `runewidth`

**Replaces:** `github.com/mattn/go-runewidth v0.0.17` + `github.com/rivo/uniseg v0.2.0`
**Current usage:** 17 files across `tui`, `terminal`, `termtest`

### v1 API (narrowed from spec)

```go
package runewidth

// Core width functions
func RuneWidth(r rune) int
func StringWidth(s string) int

// Truncation and fitting
func Truncate(s string, w int, tail string) string
func FitLeft(s string, w int) (result string, width int)   // NEW
func FitRight(s string, w int) (result string, width int)  // NEW
```

**Not exported in v1:**
- `Graphemes()` iterator — kept internal until UAX#29 conformance is strong
- `WidthIndex()` — deferred to v2 after internal usage validates the design

### Improvements over `go-runewidth` and `uniseg`

| Feature | `go-runewidth` | `uniseg` | Wonton `runewidth` |
|---------|---------------|----------|-------------------|
| ASCII fast path | no | no | yes — `len(s)` for pure ASCII |
| Unicode version | 15.1.0 | 13.0 (v0.2) | 16.0 |
| Grapheme-aware width | no (rune-by-rune) | yes | yes (internal iterator) |
| Truncate at grapheme boundary | no (splits clusters) | n/a | yes |
| `FitLeft` / `FitRight` | no | no | yes — **new to Go ecosystem** |
| Dependencies | `uniseg` | none | none |
| Emoji ZWJ sequences | incorrect width | correct | correct |

### `FitLeft` and `FitRight`

These are what TUI code actually needs: "give me the longest prefix/suffix that
fits in N cells, and tell me how many cells it used."

```go
// FitLeft returns the longest prefix of s that fits in w terminal cells,
// along with the actual display width of the returned string. Truncation
// respects grapheme cluster boundaries.
func FitLeft(s string, w int) (result string, width int)

// FitRight returns the longest suffix of s that fits in w terminal cells,
// along with the actual display width of the returned string.
func FitRight(s string, w int) (result string, width int)
```

`Truncate(s, w, tail)` is implemented in terms of `FitLeft`.

### Implementation notes

- **ASCII fast path**: `StringWidth` scans bytes first. If all `>= 0x20` and
  `< 0x80`, return `len(s)`. Zero allocations.
- **Grapheme iterator**: internal, handles combining marks, ZWJ, variation
  selectors, regional indicators, skin tones, keycaps, enclosing marks.
- **Unicode tables**: generated from Unicode 16.0 `EastAsianWidth.txt`,
  `emoji-data.txt`, `DerivedGeneralCategory.txt`. Binary search on sorted
  interval tables. Generator is reproducible with pinned Unicode version.
- **Width model**: returns `int`, documented as "terminal cells". Not hardcoded
  to 0/1/2.
- **Ambiguous width**: EAW category A treated as width 1 (matching modern
  terminal emulators). No `Condition` API in v1.

### File structure

```
runewidth/
    runewidth.go       # RuneWidth, StringWidth, Truncate, FitLeft, FitRight
    grapheme.go        # Internal grapheme cluster iterator
    tables.go          # Generated Unicode width tables (with version comment)
    generate.go        # go:generate script
    runewidth_test.go  # Tests and benchmarks
```

### Migration

1. Implement `runewidth` package with tests and benchmarks
2. Update 17 files: import swap (signatures are identical)
3. Fix bug: `tui/input_view.go:335` — `StringWidth(string(r))` → `RuneWidth(r)`
4. Remove `go-runewidth` and `uniseg` from `go.mod`
5. Run full test suite + benchmarks vs `go-runewidth` and `uniseg`

### Tests

- Exhaustive rune width: ASCII, Latin, CJK, emoji, combining, control, private use
- Grapheme clusters: ZWJ family (width 2), skin tone (width 2), flags (width 2),
  combining (`e\u0301` = 1), keycap (width 2), mixed strings
- `FitLeft` / `FitRight`: grapheme boundaries, exact fit, oversize, empty
- `Truncate`: with tail, without, no-truncation passthrough, edge cases
- ASCII fast path benchmark: measurably faster than `go-runewidth`
- Compatibility: compare against `go-runewidth` for large corpus, document diffs
- Fuzz: random Unicode, `StringWidth(s) >= 0`, `StringWidth(Truncate(s, w, "")) <= w`

### v2 candidates (after v1 ships)

- Public `Graphemes()` iterator (after UAX#29 conformance is proven)
- `WidthIndex(s string) []int` for cursor positioning
- Refactor `tui/text_input.go` rune loops to use grapheme iteration

---

## Package 3: `valuediff`

**Replaces:** `github.com/google/go-cmp v0.7.0`
**Current usage:** `assert/assert.go` and `assert/assert_test.go` (5 call sites)

### API (refined from spec)

```go
package valuediff

// Core
func Diff(x, y any, opts ...Option) string
func Equal(x, y any, opts ...Option) bool
func Format(v any) string

// Options
type Option func(*config)

func IgnoreFields[T any](names ...string) Option   // NEW: type-scoped (generics)
func IgnoreUnexported() Option
func EquateEmpty() Option
func CompareFunc[T any](fn func(T, T) bool) Option
func WithColor(enabled bool) Option
func MaxDepth(depth int) Option
```

### Improvements over `go-cmp`

| Feature | `go-cmp` | Wonton `valuediff` |
|---------|---------|-------------------|
| Unexported fields | panics by default | compared by default |
| Error comparison | requires `cmpopts.EquateErrors()` | `errors.Is` by default |
| Colored output | not supported | native via `WithColor` |
| Deterministic output | intentionally randomizes whitespace | fully deterministic |
| Field ignoring | `cmpopts.IgnoreFields(User{}, "Name")` | `IgnoreFields[User]("Name")` — type-safe |
| NaN map keys | undefined | emitted with marker, not skipped |
| `Format()` | not provided | colorized pretty-print of any value |
| Slice diffs | wall of text | smart: inline for scalars, per-field for structs |

### Key design decisions

1. **Type-scoped `IgnoreFields`** — Uses generics instead of empty-value trick.
   Safer than both `go-cmp` and the original spec's unscoped version.

2. **`Equal()` fast path** — Uses `reflect.DeepEqual` first (matching current
   `assert.Equal` behavior). Only builds diff string on failure path.

3. **NaN map keys** — Emitted as `NaN(0x...)` with hex representation.
   Deterministic, not silent.

4. **`EqualOpts` breaking change** — Accepted. Only used in `assert`'s own tests.
   Signature changes from `...cmp.Option` to `...valuediff.Option`.

5. **Color default** — `valuediff` itself defaults to no color. `assert` enables
   it when stderr is a TTY.

### Diff output format

Structured pseudo-Go syntax. Deterministic. Scannable.

```
User{
    Name: "Alice",
-   Age:  30,
+   Age:  31,
}
```

Smart slice diffing:
```
// Simple types: inline
[]string{
    "apple",
-   "banana",
+   "blueberry",
}

// Structs: per-field
[]User{
    {Name: "Alice", Age: 30},
    {
        Name: "Bob",
-       Age:  25,
+       Age:  26,
    },
}
```

### File structure

```
valuediff/
    valuediff.go       # Diff(), Equal(), Format(), Option types
    walk.go            # Recursive value walker
    format.go          # diffBuilder, line formatting, ANSI color
    myers.go           # Myers diff for slices
    export.go          # Unexported field access
    valuediff_test.go  # Core tests
    format_test.go     # Output formatting tests
    edge_cases_test.go # Edge cases
```

### Migration

1. Implement `valuediff` package with tests
2. Update `assert/assert.go`:
   - Replace `gocmp.Diff` → `valuediff.Diff`
   - Replace `gocmp.Equal` → `valuediff.Equal`
   - Remove `defaultOpts` (sane defaults built in)
   - Remove `formatDiff()` (color handled by `WithColor`)
   - Change `EqualOpts` signature to `...valuediff.Option`
3. Remove `github.com/google/go-cmp` from `go.mod`
4. Run full test suite (3064 `assert.Equal` call sites, zero changes needed)

### Tests

- Golden tests: deterministic output compared against expected strings
- Roundtrip: `Equal(x, y)` iff `Diff(x, y) == ""`
- Parity: matches `reflect.DeepEqual` for simple cases without options
- Edge cases: nil, typed nil, cycles, time.Time, errors, functions, channels, NaN
- Fuzz: random struct/slice/map values, no panics
- Benchmarks: vs `go-cmp` and `reflect.DeepEqual`

---

## Dependencies removed

After all three packages ship:

| Dependency | Removed |
|-----------|---------|
| `github.com/creack/pty v1.1.24` | by `pty` |
| `github.com/mattn/go-runewidth v0.0.17` | by `runewidth` |
| `github.com/rivo/uniseg v0.2.0` | by `runewidth` |
| `github.com/google/go-cmp v0.7.0` | by `valuediff` |

Net: **-4 dependencies, +0 new external dependencies.**
