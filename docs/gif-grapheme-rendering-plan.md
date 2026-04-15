# GIF Grapheme Rendering Investigation

## Summary

The `gif` package does not currently render grapheme clusters correctly.

This is not a single bug. It is a combination of:

- missing glyph coverage in the default bundled font
- no font fallback mechanism
- no shaping/composition support for complex sequences
- rune-by-rune terminal storage and rendering instead of grapheme-by-grapheme handling

Because of that, emoji and other multi-rune clusters currently fail in two ways:

- some content is split into multiple cells internally
- many glyphs paint nothing at all, so the final GIF appears blank where the emoji should be

## High-Signal Observations

### 1. The bundled default TTF font does not draw the target glyphs

The default renderer uses embedded Inconsolata:

- [gif/font_ttf.go](../gif/font_ttf.go)

The draw path is:

- `LoadDefaultFont()`
- `FontFace.DrawChar()`
- `font.Drawer.DrawString(string(char))`

Direct probing showed:

- ASCII such as `A` and `#` paint pixels
- `❤`, `FE0F`, `20E3`, `👋`, `🏽`, `🇯`, `🇵`, and `👨` paint `0` pixels

That means the current font path cannot render the glyphs needed for:

- VS16 emoji presentation
- keycaps
- skin-tone modifiers
- regional-indicator flags
- ZWJ emoji sequences

This alone explains why many generated GIFs appear as if the grapheme content is missing.

### 2. The `gif` terminal model is rune-based, not grapheme-based

Relevant files:

- [gif/terminal.go](../gif/terminal.go)
- [gif/emulator.go](../gif/emulator.go)

Current behavior:

- `TerminalCell` stores only `Char rune`
- `TerminalScreen.WriteString()` iterates `for _, r := range s`
- `Emulator.ProcessOutput()` decodes and writes one rune at a time

Effects:

- `#️⃣` becomes `#`, `FE0F`, `20E3` in separate cells
- `🇯🇵` becomes two regional indicators in separate cells
- `👨‍👩‍👧‍👦` becomes seven separate cells

This is the same class of issue that previously existed in `terminal`, `tui`, and `termtest`.

### 3. The renderer is also rune-based

Relevant file:

- [gif/terminal.go](../gif/terminal.go)

Current behavior:

- `RenderFrame()` loops cell-by-cell
- each cell is rendered by calling `drawChar(...)`
- `drawChar(...)` accepts a single `rune`

So even if storage were fixed, the renderer still has no place to render a full grapheme cluster string.

### 4. Current evaluation output confirms the issue

Evaluation assets were added:

- [gif/grapheme_characterization_test.go](../gif/grapheme_characterization_test.go)
- [examples/gif/grapheme_eval/main.go](../examples/gif/grapheme_eval/main.go)

These were used to evaluate:

- `❤️`
- `#️⃣`
- `🇯🇵`
- `🏳️‍🌈`
- `👨‍👩‍👧‍👦`
- `👋🏽`

Observed output:

- many GIFs render only surrounding ASCII text such as `direct: <-` or `emu: <-`
- `hash_keycap` can show the base `#`, but not the full keycap cluster
- heart, skin-tone, flag, and ZWJ-family cases appear blank or effectively missing

This indicates the current problem is not merely cluster storage. It is a rendering-stack limitation.

## Why A Small Patch Is Not Enough

A minimal internal storage fix would improve correctness, but it would not solve the visible output problem shown in the generated GIFs.

If only storage is fixed:

- `gif` would hold better internal data
- the output would likely still be blank for many emoji because the current font does not paint those glyphs
- keycaps, flags, and ZWJ sequences would still need shaping/composition support

So the long-term fix must be treated as a renderer architecture improvement, not a helper-function cleanup.

## Recommendations

### Recommendation 1: Make the `gif` terminal model grapheme-aware

Update the terminal/emulator layer to store grapheme clusters as a single visible cell.

Proposed `TerminalCell` shape:

```go
type TerminalCell struct {
    Glyph        string
    Width        int
    Continuation bool
    FG           color.Color
    BG           color.Color
}
```

Required changes:

- change `TerminalScreen.WriteString()` to iterate `runewidth.Graphemes(...)`
- change `Emulator.ProcessOutput()` to operate on grapheme clusters instead of decoded runes
- mark continuation cells for wide glyphs
- move cursor by display width of the cluster rather than by rune count

Why this matters:

- it aligns `gif` with the grapheme-aware behavior already added to `terminal`, `tui`, and `termtest`
- it fixes internal correctness
- it creates the necessary substrate for a real renderer fix

### Recommendation 2: Change the renderer from rune-based to glyph-based

Update the renderer so it renders the full glyph string from each lead cell instead of a single rune.

Proposed API shift:

- replace `drawChar(...)` with `drawGlyph(...)`
- have `RenderFrame()` skip continuation cells
- render the full `Glyph` string on the lead cell

Why this matters:

- without this, cluster-aware storage still gets collapsed back into rune-by-rune drawing

### Recommendation 3: Add font fallback support

The current renderer supports:

- one embedded TTF font, or
- one bitmap font

That is not enough for long-term Unicode support.

The renderer should support a font stack:

- primary monospace text font
- emoji/symbol fallback font
- optional user-provided fallback faces

Suggested design direction:

- add a renderer option for multiple TTF faces
- probe each face to see whether a glyph paints something
- use the first face that can render the glyph

Why this matters:

- the current bundled Inconsolata path cannot render the target emoji glyphs
- fallback support is required even before shaping enters the picture

### Recommendation 4: Add shaping support for complex emoji sequences

Font fallback alone is not enough for:

- keycaps
- flags
- ZWJ sequences

Those need shaping/composition support.

The current `golang.org/x/image/font` drawer is too low-level for this problem because:

- it does not provide full text shaping
- it assumes simple direct string drawing with a single face

Long-term, `gif` should move to a shaping-capable text pipeline for glyph layout.

This is the only durable way to support:

- `#️⃣`
- `🇯🇵`
- `👨‍👩‍👧‍👦`
- other extended emoji sequences

### Recommendation 5: Aim for visible monochrome support first

There are two distinct end states:

- visible monochrome/single-color emoji glyphs
- full color emoji rendering

Full color emoji is significantly harder, especially in a GIF pipeline.

The pragmatic target should be:

- first: reliable visible glyph rendering for clusters, even if monochrome
- later: investigate color emoji support if still needed

That keeps the implementation tractable while still solving the user-visible problem.

## Proposed Fix Plan

### Phase 1: Internal correctness

Scope:

- make `TerminalScreen` grapheme-aware
- make `Emulator` grapheme-aware
- add tests for cluster storage and cursor movement

Expected outcome:

- internal cell model becomes correct
- output may still remain visually poor for many emoji

Risk:

- low to moderate

### Phase 2: Glyph-based rendering

Scope:

- change renderer to draw whole glyph strings from lead cells
- skip continuation cells when painting
- update tests/examples accordingly

Expected outcome:

- renderer becomes architecture-compatible with cluster storage

Risk:

- moderate

### Phase 3: Font fallback

Scope:

- support multiple TTF faces
- introduce configurable fallback chain
- bundle or load at least one emoji-capable fallback font

Expected outcome:

- many simple non-ASCII glyphs become visible
- base emoji may begin to render

Risk:

- moderate
- package size / licensing / embedded asset choices must be considered

### Phase 4: Shaping support

Scope:

- introduce shaping-capable text layout for the GIF renderer
- shape full grapheme strings before painting

Expected outcome:

- proper support path for keycaps, flags, ZWJ sequences, and complex emoji

Risk:

- highest
- likely requires a new dependency and renderer refactor

## Proposed Tests

### Characterization tests

Keep and expand:

- [gif/grapheme_characterization_test.go](../gif/grapheme_characterization_test.go)

Purpose:

- show current and future storage behavior explicitly
- prevent regressions during the migration

### Model-level correctness tests

Add tests that verify:

- a grapheme cluster occupies one lead cell
- continuation cells are marked correctly
- cursor movement uses display width rather than rune count
- emulator output and direct screen writes behave consistently

### Rendering tests

Initial rendering tests should avoid requiring exact emoji pixels.

Prefer assertions such as:

- a glyph paints at least some pixels
- fallback face is selected when primary face paints nothing
- lead cell renders and continuation cells do not double-paint

Only later, after shaping and fallback are stable, consider stricter image-based expectations.

## Practical Conclusion

If the goal is long-term success for emoji/grapheme support in GIF output, the right path is:

1. make `gif` storage grapheme-aware
2. make rendering glyph-based
3. add font fallback
4. add shaping support for complex sequences

The current screenshots and evaluation runs strongly suggest that:

- a storage-only patch is not enough
- the default bundled font is a hard blocker for visible output
- shaping support is required for correct rendering of the most important cluster cases

## Suggested Next Implementation Slice

The smallest meaningful implementation slice is:

1. grapheme-aware `TerminalScreen`
2. grapheme-aware `Emulator`
3. glyph-based lead-cell rendering

That will not fully solve visual emoji rendering, but it will put the package on the correct architecture so font fallback and shaping can be added cleanly afterward.
