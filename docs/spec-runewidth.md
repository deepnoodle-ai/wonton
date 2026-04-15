# Spec: `runewidth` Package

Replace `github.com/mattn/go-runewidth` and its transitive dependency
`github.com/rivo/uniseg` with a local implementation. Removes 2 third-party
dependencies.

## Motivation

The `go-runewidth` library determines terminal display widths of Unicode
characters. Wonton uses it in 17 files across `tui`, `terminal`, and `termtest`.
A local replacement can:

- Add an ASCII fast path (no allocations, no table lookups for ~95% of content)
- Use current Unicode 16.0 tables (upstream uses 15.1.0; uniseg v0.2.0 uses 13.0)
- Correctly handle modern emoji (ZWJ sequences, skin tones, flags, variation selectors)
- Expose a grapheme cluster iterator for cursor positioning
- Eliminate two transitive dependencies

## Current Usage

Only 3 functions from `go-runewidth` are used anywhere in Wonton:

| Function | Calls | Description |
|----------|-------|-------------|
| `runewidth.StringWidth(s)` | ~50 | Display width of a string |
| `runewidth.RuneWidth(r)` | ~20 | Display width of a single rune |
| `runewidth.Truncate(s, w, tail)` | ~8 | Truncate string to fit width |

No configuration APIs are used (`Condition`, `EastAsianWidth`,
`DefaultCondition`, `CreateLUT`, etc.).

### Files

- **tui**: `markdown_view.go`, `markdown_test.go`, `markdown.go`,
  `tree_view.go`, `table_view.go`, `input_view.go`, `spinner.go`,
  `progress_bar.go`, `code_view.go`, `text_input.go`, `diff_view.go`,
  `text.go`, `list.go`, `layout.go`
- **terminal**: `terminal.go`, `frame.go`
- **termtest**: `screen.go`

### Known Bug

In `tui/input_view.go:335`, `runewidth.StringWidth(string(r))` is called on a
single rune. This should be `runewidth.RuneWidth(r)` to avoid unnecessary
grapheme cluster processing overhead. Fix during migration.

## Package Location

`github.com/deepnoodle-ai/wonton/runewidth`

## Public API

```go
package runewidth

import "iter"

// RuneWidth returns the number of terminal cells needed to display rune r.
// Returns 0 for non-printable and combining characters, 1 for most characters,
// and 2 for wide characters (CJK ideographs, fullwidth forms, most emoji).
//
// This function operates on individual runes. For strings that may contain
// multi-rune grapheme clusters (emoji sequences, combining marks), use
// StringWidth instead.
func RuneWidth(r rune) int

// StringWidth returns the number of terminal cells needed to display string s.
// It correctly handles multi-rune grapheme clusters such as:
//   - Emoji ZWJ sequences (e.g., family emoji 👨‍👩‍👧‍👦)
//   - Skin tone modifiers (e.g., 👋🏽)
//   - Flag sequences (e.g., 🇺🇸)
//   - Combining characters (e.g., é composed as e + ◌́)
//   - Variation selectors (VS15 for text, VS16 for emoji presentation)
//
// Each grapheme cluster contributes the width of its base character.
func StringWidth(s string) int

// Truncate returns s truncated to at most w terminal cells, with tail appended
// if truncation occurred.
//
// The result never exceeds w cells in display width. Truncation respects
// grapheme cluster boundaries: it will not split a multi-rune cluster.
//
// Common usage:
//
//	Truncate("Hello, 世界!", 8, "…")  // "Hello, …"
//	Truncate("short", 10, "…")        // "short" (no truncation)
func Truncate(s string, w int, tail string) string

// Graphemes returns an iterator over the grapheme clusters in s, yielding each
// cluster's string value and display width.
//
// This is useful for cursor positioning and character-level text editing where
// you need to advance by grapheme cluster rather than by rune.
//
// Example:
//
//	for g, w := range runewidth.Graphemes("Hello 👨‍👩‍👧‍👦!") {
//	    fmt.Printf("%q width=%d\n", g, w)
//	}
func Graphemes(s string) iter.Seq2[string, int]
```

## Implementation Approach

### Unicode Width Tables

**Source data**: Unicode 16.0 property files:
- `EastAsianWidth.txt` — East Asian Width property
- `emoji-data.txt` — Emoji properties (Extended_Pictographic, Emoji_Presentation)

**Code generation**: A `go generate` script (`cmd/generate-runewidth/main.go`)
downloads these files and generates `tables.go` containing sorted interval
tables for:
- `doublewidth` — EAW categories W (Wide) and F (Fullwidth)
- `combining` — Combining marks (General_Category Mn/Mc/Me)
- `nonprint` — Control characters and other zero-width characters
- `emojiPresentation` — Characters with Emoji_Presentation=Yes

**Lookup**: Binary search on sorted interval tables. The tables are small enough
that O(log n) is fast and avoids the 557KB memory cost of a full lookup table.

### RuneWidth Logic

```
1. r < 0x20       → 0 (C0 controls)
2. r < 0x7F       → 1 (printable ASCII — fast path)
3. r < 0xA0       → 0 (C1 controls, DEL)
4. r == 0xAD      → 0 (soft hyphen)
5. r < 0x300      → 1 (Latin-1 Supplement through Latin Extended-B)
6. inTable(combining) → 0
7. inTable(nonprint)  → 0
8. inTable(doublewidth) → 2
9. default → 1
```

Branch order is optimized for ASCII-heavy terminal content.

### StringWidth — Two-Phase Approach

**Phase 1: ASCII fast path.** Scan bytes. If every byte is `>= 0x20` and
`< 0x80`, the width equals `len(s)`. No allocations, no grapheme processing.

**Phase 2: Grapheme-aware path.** For strings with non-ASCII bytes, iterate by
grapheme cluster. Rather than importing a full UAX#29 library, implement a
focused grapheme iterator handling the cases that affect width:

1. **Combining marks** (Mn, Mc, Me): Don't start a new cluster
2. **ZWJ** (U+200D): Joins preceding and following into one cluster
3. **Variation selectors** (U+FE00-FE0F): Attach to preceding character; VS16
   forces emoji width (2)
4. **Regional indicators** (U+1F1E6-U+1F1FF): Pairs form a single flag cluster
   of width 2
5. **Skin tone modifiers** (U+1F3FB-U+1F3FF): Attach to preceding emoji
6. **Keycap sequences**: Digit + U+FE0F + U+20E3 forms a single cluster
7. **Enclosing marks** (U+20DD-U+20E3): Attach to preceding character

Width of a cluster = width of first non-zero-width rune, except:
- Cluster contains VS16 and base is emoji-capable → width 2
- Cluster is a regional indicator pair → width 2

### Truncate — Single Pass

1. Iterate by grapheme cluster (reusing the same iterator)
2. Track accumulated width
3. When the next cluster would exceed `w - StringWidth(tail)`, stop
4. Return accumulated string + tail

This avoids the double-pass that go-runewidth performs.

### Ambiguous-Width Characters

Default: treat EAW category A as **width 1**, matching go-runewidth's default
and modern terminal emulators (iTerm2, Alacritty, WezTerm, Ghostty, Windows
Terminal). No `Condition` or EastAsian mode is implemented initially. If needed
later, add via functional options.

## Testing Strategy

1. **Exhaustive rune width tests** — all categories: ASCII, Latin, CJK, emoji,
   combining, control, private use
2. **Grapheme cluster tests** — ZWJ family (`👨‍👩‍👧‍👦` → 2), skin tone (`👋🏽` → 2),
   flag (`🇺🇸` → 2), combining (`e\u0301` → 1), keycap (`#\uFE0F\u20E3` → 2),
   mixed (`"Hello 世界 👋🏽!"` → 16)
3. **Truncation tests** — grapheme-boundary-aware, correct tail, no-truncation
   pass-through, edge cases (width 0, width 1, empty string)
4. **ASCII fast path benchmark** — verify measurably faster than go-runewidth
5. **Compatibility tests** — compare against go-runewidth for a large corpus;
   document intentional differences
6. **Fuzz tests** — random Unicode strings verifying invariants:
   `StringWidth(s) >= 0`, `StringWidth(Truncate(s, w, "")) <= w`

## Migration

**Import change** in all 17 files:
```go
// Before
import "github.com/mattn/go-runewidth"

// After
import "github.com/deepnoodle-ai/wonton/runewidth"
```

**No code changes** required — `RuneWidth`, `StringWidth`, and `Truncate` have
identical signatures.

**Bug fix**: In `tui/input_view.go:335`, change
`runewidth.StringWidth(string(r))` to `runewidth.RuneWidth(r)`.

**Optional follow-up**: Refactor rune-by-rune loops in `text_input.go` to use
`Graphemes()` for correct multi-rune emoji cursor movement.

**Dependency removal**: Remove `github.com/mattn/go-runewidth` and
`github.com/rivo/uniseg` from `go.mod`.

**Zero new dependencies**: Pure Go, no external imports. Unicode tables are
generated at development time.

## File Structure

```
runewidth/
    runewidth.go       # RuneWidth, StringWidth, Truncate, Graphemes
    grapheme.go        # Grapheme cluster iterator implementation
    tables.go          # Generated Unicode width tables
    generate.go        # go:generate script for tables
    runewidth_test.go  # Tests and benchmarks
```
