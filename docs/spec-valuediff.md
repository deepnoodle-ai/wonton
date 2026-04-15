# Spec: `valuediff` Package

Replace `github.com/google/go-cmp` with a local implementation. Removes 1
third-party dependency and makes the `assert` package free of external deps.

## Motivation

The `go-cmp` library provides deep value comparison with human-readable diffs.
Wonton uses it exclusively in the `assert` package (2 files, 5 call sites). A
local replacement can:

- Compare unexported fields by default (go-cmp panics unless you opt in)
- Compare errors via `errors.Is` by default (no need for `cmpopts.EquateErrors()`)
- Produce ANSI-colored output natively (no post-processing)
- Generate deterministic output (go-cmp intentionally randomizes whitespace)
- Offer a simpler option system for the common cases
- Eliminate the dependency entirely

## Current Usage

All usage is in `assert/assert.go` and `assert/assert_test.go`:

| Call Site | Function | Purpose |
|-----------|----------|---------|
| `Equal()` | `cmp.Diff(want, got, defaultOpts...)` | Diff string on failure |
| `EqualOpts()` | `cmp.Diff(want, got, opts...)` | Diff with caller options |
| `NotEqual()` | `cmp.Equal(got, want)` | Boolean equality check |
| `contains()` x2 | `cmp.Equal(elem, needle)` | Element-wise equality |

**Default options configured:**
- `cmpopts.EquateErrors()` — compare errors via `errors.Is`
- `cmp.Exporter(func(reflect.Type) bool { return true })` — export all
  unexported fields

**How the diff is consumed:** `formatDiff()` in assert splits the diff string on
`\n` and colorizes lines starting with `-` (red) or `+` (green).

**Important design detail:** `Equal()` uses `reflect.DeepEqual` for the
equality check and only calls `cmp.Diff` on the failure path. The happy path
never touches go-cmp.

## Package Location

`github.com/deepnoodle-ai/wonton/valuediff`

Rationale: The diff/equality logic is independently useful outside test
assertions (debugging, logging, config comparison). Clean dependency direction:
`assert` imports `valuediff`, not the other way around.

## Public API

```go
package valuediff

// Diff compares two values and returns a human-readable description of any
// differences. Returns an empty string if the values are deeply equal.
//
// By default, unexported struct fields are compared (matching reflect.DeepEqual
// behavior) and errors are compared using errors.Is semantics.
//
// Example:
//
//     diff := valuediff.Diff(got, want)
//     if diff != "" {
//         t.Errorf("mismatch (-want +got):\n%s", diff)
//     }
func Diff(x, y any, opts ...Option) string

// Equal reports whether x and y are deeply equal.
//
// This is equivalent to checking whether Diff returns an empty string, but more
// efficient because it short-circuits without building a diff string.
//
// Unexported struct fields are compared by default.
func Equal(x, y any, opts ...Option) bool

// Format returns a human-readable string representation of a Go value.
// Useful for printing values in test failure messages.
func Format(v any) string

// Option configures comparison behavior.
type Option func(*config)

// IgnoreFields returns an Option that ignores the named struct fields during
// comparison. Field names are matched exactly.
//
// Example:
//
//     diff := valuediff.Diff(got, want, valuediff.IgnoreFields("CreatedAt", "UpdatedAt"))
func IgnoreFields(names ...string) Option

// IgnoreUnexported returns an Option that skips unexported struct fields.
// By default, unexported fields ARE compared. Use this to opt out.
func IgnoreUnexported() Option

// EquateEmpty returns an Option that treats nil and empty slices/maps as equal.
// By default, nil != empty (matching reflect.DeepEqual).
func EquateEmpty() Option

// CompareFunc returns an Option that uses a custom comparison function for
// values of type T. The function should return true if the values are equal.
//
// Example:
//
//     diff := valuediff.Diff(got, want, valuediff.CompareFunc(func(a, b time.Time) bool {
//         return a.Equal(b)
//     }))
func CompareFunc[T any](fn func(T, T) bool) Option

// WithColor returns an Option that enables ANSI color in diff output.
// By default, color is disabled. The assert package enables it when stderr
// is a terminal.
func WithColor(enabled bool) Option

// MaxDepth returns an Option that limits comparison depth. Values nested deeper
// than depth are compared using reflect.DeepEqual without generating detailed
// diffs. Default is 32.
func MaxDepth(depth int) Option
```

## Diff Output Format

Structured pseudo-Go syntax. Deterministic (no random whitespace).

**Simple scalar mismatch:**
```
int(
-   42
+   43
)
```

**Struct diff:**
```
User{
    Name: "Alice",
-   Age:  30,
+   Age:  31,
    Email: "alice@example.com",
}
```

**Slice diff:**
```
[]string{
    "apple",
-   "banana",
+   "blueberry",
    "cherry",
}
```

**Map diff:**
```
map[string]int{
    "a": 1,
-   "b": 2,
+   "b": 3,
    "c": 4,
}
```

**Nested struct:**
```
Config{
    Server: ServerConfig{
        Host: "localhost",
-       Port: 8080,
+       Port: 9090,
    },
    Debug: false,
}
```

**Long identical sequences (elided):**
```
[]int{
    ... // 50 identical elements
    51,
-   52,
+   99,
    53,
    ... // 47 identical elements
}
```

**Cycle detection:**
```
*Node{
    Value: 1,
    Next: &<cycle: *Node>,
}
```

**With color enabled:**
- Lines starting with `-` rendered in red
- Lines starting with `+` rendered in green
- Context lines uncolored
- Color applied inline during generation (no post-processing)

## Implementation Approach

### Value Traversal

Recursive `walk(x, y reflect.Value, path string, depth int)` that:

1. Handles nil/invalid values
2. Checks type equality (different types are always unequal)
3. Checks for `Equal(T) bool` method (e.g., `time.Time`)
4. Checks for custom `CompareFunc` options
5. Dispatches by `reflect.Kind`:
   - **Bool, Int\*, Uint\*, Float\*, Complex\*, String**: direct `==`
   - **Struct**: iterate fields, recurse per field, respecting `IgnoreFields`
     and `IgnoreUnexported`
   - **Slice/Array**: Myers diff for insertions/deletions/modifications (full
     O(ND) for len <= 100, simplified prefix/suffix strip for larger)
   - **Map**: collect all keys from both, sort deterministically, compare per key
   - **Ptr**: dereference and recurse (with cycle detection)
   - **Interface**: extract underlying value and recurse
   - **Func**: nil vs non-nil only
   - **Chan**: pointer identity
6. Falls back to `reflect.DeepEqual` at max depth

### Unexported Field Access

Use `unsafe.Pointer` to read unexported struct fields:

```go
func readUnexportedField(v reflect.Value, f reflect.StructField) reflect.Value {
    return reflect.NewAt(f.Type, unsafe.Pointer(v.UnsafeAddr()+f.Offset)).Elem()
}
```

For non-addressable values, allocate a heap copy first.

### Cycle Detection

Maintain `map[visitKey]bool` where `visitKey` is `{ptr uintptr, typ
reflect.Type}`. Check before entering pointer/map/slice elements. Emit
`<cycle>` and return if already visited.

### Diff Building

```go
type diffLine struct {
    mode   diffMode // ' ', '-', '+'
    indent int
    text   string
}

type diffBuilder struct {
    lines []diffLine
    color bool
}
```

The `String()` method renders lines with prefixes and optional ANSI color.

### Error Comparison

By default, when both values implement `error`, use `errors.Is` semantics. This
replaces `cmpopts.EquateErrors()`.

### `Equal()` Fast Path

Use `reflect.DeepEqual` as a fast path (optimized in the runtime). Only invoke
the diff walker when `Diff()` is called. This matches the current assert
pattern.

## Edge Cases

| Case | Behavior |
|------|----------|
| Unexported fields | Compared by default (opt out with `IgnoreUnexported()`) |
| nil vs empty slice | Not equal by default (opt in with `EquateEmpty()`) |
| nil vs empty map | Same as slices |
| nil interface | Equal only to nil interface |
| Typed nil | `(*int)(nil)` == `(*int)(nil)`; `(*int)(nil)` != nil interface |
| Cycles | Detected, reported as `<cycle>` |
| `time.Time` | Compared via `Equal()` method |
| Errors | Compared via `errors.Is` by default |
| Functions | nil equality only; two non-nil functions are never equal |
| Channels | Pointer identity |
| Maps with NaN keys | Skipped |
| Large slices (>1000) | Prefix/suffix stripped, middle truncated |
| Deep nesting | Falls back to `reflect.DeepEqual` at `MaxDepth` (default 32) |

## Testing Strategy

1. **Golden tests** — deterministic output compared against expected strings
2. **Roundtrip property** — `Equal(x, y)` is true iff `Diff(x, y) == ""`
3. **Parity with reflect.DeepEqual** — for simple cases without options,
   `Equal(x, y)` matches `reflect.DeepEqual(x, y)`
4. **Edge case coverage** — explicit tests for every row in the table above
5. **Fuzz testing** — random struct/slice/map values, verify no panics
6. **Benchmarks** — compare against go-cmp and reflect.DeepEqual

## Migration

### Changes to `assert/assert.go`

**Imports:**
```go
// Remove
gocmp "github.com/google/go-cmp/cmp"
"github.com/google/go-cmp/cmp/cmpopts"

// Add
"github.com/deepnoodle-ai/wonton/valuediff"
```

**Remove `defaultOpts`** — valuediff handles unexported fields and error
comparison by default.

**`Equal()`** — replace `gocmp.Diff(want, got, defaultOpts...)` with
`valuediff.Diff(want, got, valuediff.WithColor(colorEnabled))`.

**`EqualOpts()`** — signature changes from `...gocmp.Option` to
`...valuediff.Option`. Only used in assert's own tests.

**`NotEqual()`** — replace `gocmp.Equal(got, want)` with
`valuediff.Equal(got, want)`.

**`contains()`** — replace `gocmp.Equal(...)` with `valuediff.Equal(...)`.
Improves correctness: the current code panics on unexported fields since it
passes no options.

**Remove `formatDiff()`** — colorization moves into valuediff via `WithColor`.

### Impact

- `assert.Equal` (3064 call sites across 108 files): **zero changes**
- `assert.EqualOpts` (only in assert's own tests): signature changes
- Remove `github.com/google/go-cmp` from `go.mod`

## File Structure

```
valuediff/
    valuediff.go       # Diff(), Equal(), Format(), Option types
    walk.go            # Recursive reflect-based value walker
    format.go          # diffBuilder, line formatting, ANSI color
    myers.go           # Myers diff algorithm for slices
    export.go          # Unexported field access via unsafe
    valuediff_test.go  # Core functionality tests
    format_test.go     # Output formatting tests
    edge_cases_test.go # Edge case scenarios
```

## Phased Implementation

1. **Core** — `Equal()` and `Diff()` for scalars, structs, slices, maps;
   unexported field access; deterministic output; basic tests
2. **Options and edge cases** — `IgnoreFields`, `IgnoreUnexported`,
   `EquateEmpty`, `CompareFunc`, cycle detection, `errors.Is`, `WithColor`,
   `MaxDepth`
3. **Migration** — update `assert`, remove go-cmp, run full test suite
4. **Polish** — `Format()`, Myers diff, benchmarks, docs
