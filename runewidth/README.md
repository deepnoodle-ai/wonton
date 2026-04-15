# runewidth

Measure the display width of Unicode text and split it into grapheme clusters. In-tree, so wonton doesn't carry an external grapheme segmenter. Unicode 17.0.0, UAX#29 conformant.

## Why in-tree

Every terminal package in wonton needs to know how wide a character will render before it paints. The ecosystem options each had a tradeoff we didn't want: `go-runewidth` pulls in an external segmenter and trails the Unicode version, and `uniseg` allocates on width queries. This package folds both jobs into one and gets the hard cases right: ZWJ emoji, skin tones, flags, keycaps, VS16, Indic conjuncts, em dashes.

For the detailed comparison and benchmarks see [../docs/runewidth-improvements.md](../docs/runewidth-improvements.md).

## Usage

```go
import "github.com/deepnoodle-ai/wonton/runewidth"

// Display width in terminal cells.
w := runewidth.StringWidth("hello 👩‍⚕️") // 9

// Iterate grapheme clusters and their widths.
for cluster, width := range runewidth.Graphemes("a👍🏽b") {
    fmt.Println(cluster, width)
}

// Fit text into a column budget.
left, _ := runewidth.FitLeft("こんにちは世界", 6) // "こんに"

// Truncate with an ellipsis when it overflows.
s := runewidth.Truncate("a long line of text", 10, "…") // "a long li…"
```

## API

| Function | What it does |
|----------|--------------|
| `RuneWidth(r) int` | Width of a single rune. Use `StringWidth` for anything that might contain combining marks or ZWJ sequences. |
| `StringWidth(s) int` | Display width of a string. Zero allocations. |
| `Graphemes(s)` | `iter.Seq2[string, int]` over clusters and their widths. Allocation-free. |
| `Truncate(s, w, tail)` | Cut `s` to width `w`, appending `tail` if it had to truncate. |
| `Fit(s, w, tail)` | Like `Truncate`, but also returns the resulting width. |
| `FitLeft(s, w)` / `FitRight(s, w)` | Longest prefix / suffix fitting in `w` cells. `FitRight` is the one you want for right-aligned columns. |
| `WidthIndex(s) []int` | Byte offset → visual column. Useful for editor cursor positioning. Allocates one slice the length of `s`. |
| `SetTerminalCompatMode(bool)` | Opt in to width tweaks that match what real terminals draw (em dashes as 1 cell, keycaps as 2). Off by default. |
| `UnicodeVersion` | `"17.0.0"`. |

## Related

- [terminal](../terminal) — input decoding and terminal control
- [tui](../tui) — uses this for layout and wrap math
- [termtest](../termtest) — uses this for grid measurement
