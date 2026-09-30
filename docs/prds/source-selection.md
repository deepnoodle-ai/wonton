# Copy original text from a styled transcript

Status: Approved\
PRD PR: [#61](https://github.com/deepnoodle-ai/wonton/pull/61) · Implementation PR: Not opened · Updated: 2026-09-30

## Problem and context

A terminal user selects an answer or tool output to reuse its text. Wonton's `ViewportState.SelectedText` copies rendered rows, trims trailing spaces, and inserts wrap breaks. Markdown rendering also hides syntax and generates spacing. The original text cannot be recovered reliably from those cells.

Dashi's approved [transcript PRD](https://github.com/deepnoodle-ai/dashi/pull/127) requires drag, word, and logical-line copy with Unicode and source lines. Wonton already owns Text and Goldmark Markdown layouts, mouse gestures, scrolling, and item caches. Add source provenance there rather than requiring an application to duplicate those layouts.

## Users and concepts

The primary user is a Go developer building a transcript with `Viewport` and `Markdown`. Their terminal user needs exact text despite narrow screens and streaming updates.

**Source** is the original UTF-8 text supplied for an item. **Source position** is a byte boundary in that text. **Logical line** is one source line, independent of visual wrapping. **Decoration** is a displayed cell with no source text, such as a role prefix, border, or table padding.

## Use cases and acceptance

### UC-1: Select and copy wrapped text

The developer opts an item into source selection, marks its source-bearing leaves, and uses the existing mouse and copy workflow:

```go
// items supplies Source(i) and returns views such as:
body := tui.Markdown(source, nil).SourceOffset(0)
item := tui.Padding(1, tui.Stack(
    tui.Group(tui.Text("assistant: "), body),
)) // unmarked Text remains decoration
_ = item
view := tui.Viewport(&state, items)
// Render view through the existing Wonton runtime.
state.HandleMouse(event)
selected := state.SelectedText()
// The application sends selected to its existing clipboard workflow.
```

Acceptance:

- [ ] `ViewportSourceItems.Source(i)` opts individual items into one canonical source. `TextView.SourceOffset` and `MarkdownView.SourceOffset` mark source-bearing leaves. They reuse existing layout provenance.
- [ ] Each marked leaf declares its exact contiguous source slice. Multiple leaves can represent disjoint slices, including colored diff rows. Unmarked children remain decoration. Conflicting declarations invalidate the mapping.
- [ ] Forward and backward drags return identical exact source slices. Wrapping adds no copied newline. Original tabs, repeated spaces, trailing spaces, blank lines, and CRLF remain unchanged inside the slice.
- [ ] Hits on either cell of a wide grapheme select whole graphemes. Combining sequences, emoji modifiers, ZWJ sequences, and CJK never produce partial UTF-8 or partial graphemes.
- [ ] Double-click uses Wonton's existing terminal-word policy, across visual wraps within one logical line. Triple-click selects the full logical line, excluding its terminating newline.
- [ ] A logical-line selection includes original indentation and trailing whitespace. A long clipped code line copies in full. An empty line creates no selection, as today.
- [ ] Drag auto-scroll reaches off-screen text. Selection still suspends Follow and restores its previous value when cleared.

### UC-2: Copy Markdown source through its presentation

The developer renders Markdown inside the transcript's existing composition, with a role prefix and padding. The user selects visible content.

Acceptance:

- [ ] Drag/word copy returns the exact source slice between mapped endpoints. Hidden syntax strictly inside the slice remains. Hidden syntax outside the slice does not appear.
- [ ] For `**bold**`, word selection copies `bold`. Logical-line selection copies `**bold**`. For `[docs](url) next`, selecting `docs` copies `docs`; selecting from its `d` through `next` copies `docs](url) next`.
- [ ] Escapes and entities map their displayed grapheme to the complete original token. Selection of displayed `&` from `&amp;` copies `&amp;`.
- [ ] Lists, blockquotes, headings, links, inline code, fenced code, and tables retain original offsets. Logical-line copy retains source markers, delimiters, and spacing.
- [ ] Table borders, alignment spaces, synthetic paragraph spacing, prefixes, and padding never become source bytes. A decoration-only click creates no selection. A drag through decorations clamps to the adjacent source boundary in reading order.
- [ ] Provenance follows existing padding, Stack/Group offsets, and clipping. Wrapped code and table content map to their source cells without guessed text matching.
- [ ] Empty Markdown and unfinished fences during streaming never fabricate source offsets. A failed selected source-aware layout clears the entire selection before copy or highlight and restores Follow. This includes failure in an intermediate item of a cross-item range.

### UC-3: Keep selection correct as the transcript changes

The user selects text, resizes the terminal, and receives more output.

Acceptance:

- [ ] Width changes and theme changes preserve source endpoints and copied bytes. The highlight follows the selected text in the new layout.
- [ ] Appending items preserves selection. An unchanged-prefix append preserves selection only while both endpoints remain grapheme boundaries. Combining-mark, emoji-modifier, and ZWJ appends that remove a boundary clear selection and restore Follow. Appends never silently extend the range.
- [ ] Source replacement that changes the old prefix clears an affected selection. Wholesale replacement or reordering through `InvalidateAll` clears source selection. It never copies another item's text under old endpoints.
- [ ] `Invalidate(i)`, `InvalidateAll`, width changes, and layout-setting changes invalidate matching provenance caches. Repeated unchanged frames reuse item and layout caches.
- [ ] A cross-item range copies contributions in item order, with one explicit LF between contributing items. Zero-contribution decoration items add neither bytes nor separators. Source/decoration/source copies `first\nsecond`. Source bytes and legacy extraction retain their respective behavior. Viewport gap rows add no separators.

## Requirements and decisions

- Opt-in is additive. Existing `ViewportItems`, rendered `SelectionPoint`, gestures, and copying keep their behavior when source selection is absent. Document how source endpoints relate to the existing rendered-coordinate accessor.
- A missing source capability means legacy selection. An opted-in empty source means decoration-only content, with no copied contribution. Zero-value state and no selection still return an empty string.
- The caller owns immutable source strings and leaf bindings until invalidation. Item indices identify the same logical items until `InvalidateAll`. Replacing or reordering identities requires `InvalidateAll`, even when their text is identical.
- Keep half-open source-byte endpoints at grapheme boundaries. Reflow changes coordinates, not source identity. Reject stored rendered-row endpoints for source items.
- Copy exact slices, including hidden syntax between endpoints. Reject reconstructed Markdown and normalized whitespace because either changes the user's original text.
- Produce provenance during existing layout and Goldmark traversal. Reject rendered-to-source matching, unsafe access, module-cache modification, and a second Markdown parser.
- Source means the caller-supplied text, including any sanitization the caller already applied. Rendering controls must not execute. Their visible replacement must have explicit provenance. Copied strings remain inert data and never replay unsanitized controls to native scrollback.
- Extend existing composition and caches only as needed for transcript text. The [technical spec](../design/source-selection.md) defines the public signatures and layout contract.

Clipboard transports, Dashi commands, terminal mouse-capture toggles, arbitrary Canvas provenance, editing, and a general source-change transaction system are outside this track.

## Success and risks

Success requires byte-for-byte fixture assertions for every use case, compatibility tests, a runnable public API example, and terminal evidence at narrow and wide sizes. Verify that unchanged frames do not repeatedly parse or lay out the transcript. An independent implementation reviewer and taste reviewer must exercise the gestures and highlight behavior.

Markdown can hide or reorder presentation details. The main risk is an ambiguous source mapping. Pin the rules with repeated words, entities, links, tables, prefixes, long code, and streamed unfinished syntax before publishing the API.
