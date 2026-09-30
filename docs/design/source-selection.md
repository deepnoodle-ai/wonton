# Source selection through existing layouts

Status: In Review · Author: Codex · Date: 2026-09-30\
Workflow: Spec → root checkpoint → build. Requirements: [source-selection PRD](../prds/source-selection.md).

## Context and scope

Viewport selection currently addresses rendered rows and copies an off-screen cell render. Text wrapping and Markdown presentation discard source offsets. Carry offsets through those same layouts so copy uses original bytes while hit-testing and highlight use current cells.

This adds source selection to Text/Markdown transcript items and their existing composition. It does not add another parser, public source-map framework, editing transactions, or clipboard transport.

## Public contract

```go
type ViewportSourceItems interface {
    ViewportItems
    Source(i int) (source string, ok bool)
}

func (t *TextView) SourceOffset(offset int) *TextView
func (m *MarkdownView) SourceOffset(offset int) *MarkdownView
```

`Source(i)` with `ok == false` retains legacy cell selection. With `ok == true`, it supplies the item's canonical immutable UTF-8 source. An empty source marks decoration-only content. Existing `ViewportItems` implementations need no change.

`SourceOffset` marks that leaf's content as a contiguous slice starting at an absolute byte offset in the item source. Zero is a valid explicit declaration. An unmarked leaf is decoration. Multiple disjoint leaves can represent one source, such as separately styled diff rows. Validate content equality at the declared offset. Do not search for matching strings. Reject invalid bounds, conflicting overlapping declarations, invalid UTF-8, or inconsistent mappings.

The caller preserves item identity at each index until `InvalidateAll`. Source strings and leaf bindings are immutable while cached. Changing content or bindings requires `Invalidate(i)` and a new view. Appending items needs no invalidation. Reordering or replacing item identities requires `InvalidateAll`, even when source bytes match.

Retain `SelectionPoint` and `Selection()` as rendered coordinates. For source endpoints, project the selected source span onto the current mapped cells. These coordinates describe presentation and must not become copy offsets. No new exported source-selection accessor is needed for this workflow. `HasSelection`, `ClearSelection`, `HandleMouse`, and `SelectedText` remain the application entry points.

## Provenance and layout

An opted-in item owns a private provenance collector during its off-screen selection render. Pass it through `RenderContext` and `SubContext`. Existing padding, Stack, Group, and clipping apply their normal coordinate transforms. Unmarked leaves and synthesized decorations contribute no source cells.

The collector records emitted cell coordinates and their half-open source intervals. A wide grapheme's continuation cell shares its interval. Keep a validated source binding even when clipping leaves no visible tokens. Explicit empty-source decoration is distinct from failed provenance. Reject a nonempty source item with no valid declared layout.

Text layout carries offsets through word wrapping, newline splitting, alignment, and sanitized control representations. Collapsed whitespace remains present in the canonical source slice. Generated spacing stays unmapped. Rendering never emits executable source controls. Copy returns inert text and does not write native scrollback.

Markdown continues to use Goldmark. Add private source metadata to `StyledSegment` and preserve it through merges, wrapping, code highlighting, table layout, and clipping. Derive offsets from AST source segments during traversal. Decoded escapes and entities map to their complete original tokens. Hidden syntax contributes no cells but remains between selected source endpoints. Synthetic markers, borders, indentation, and padding remain unmapped.

Record enough source-grapheme and logical-line boundaries to support token transformations and clipped text. Do not calculate original offsets by counting rendered characters. Source order controls range ordering even where wrapped table cells change visual order.

## Selection, copy, and failure

Store private endpoint variants: legacy item/row/column or source item/byte offset. Source ranges are half-open and ordered by item, then byte offset. A hit maps through the owning cell token and snaps to a complete source grapheme or transformed token. Both cells of a wide grapheme behave consistently.

Starting a selection on decoration creates no selection. During a drag, unmapped spacing clamps to the adjacent mapped boundary in rendered reading order. Gaps between items retain the existing end-of-previous-item behavior. Highlight mapped cells whose source intervals intersect the selection. Do not highlight decoration as source.

Word selection uses the existing terminal-word predicate over mapped logical content, continuing through visual wraps. Logical-line selection uses the source line containing the hit. It includes original indentation, syntax, and trailing whitespace, excludes its LF or CRLF terminator, and can exceed clipping. Empty logical lines retain today's no-selection behavior.

`SelectedText` validates every selected source-aware item's binding and layout before slicing. Intermediate source items contribute their complete source. Endpoint items contribute their bounded slices. Legacy items retain their existing extraction. Skip zero-length decoration contributions, then join contributing items with one LF. Do not trim source contributions or copy visual gaps.

If any selected source-aware item's binding or layout is invalid, clear the whole selection before copy or highlight. Restore Follow through `ClearSelection` and return no copied text. Never silently omit a failed intermediate item or fall back to its rendered text. No selected item may reuse stale provenance.

## Cache and streaming lifecycle

Keep provenance with each viewport entry's existing view, measurement, and rendered-line caches. Width, theme/layout settings, and `Invalidate(i)` discard matching maps. Unchanged frames reuse them. Source selection retains its prior immutable source snapshot while an affected entry rebuilds.

After rebuild, preserve endpoints only if the new source equals the old source or has the old source as its exact prefix. Recheck stored endpoints against the new grapheme boundaries. Appending U+0301 to selected `e`, an emoji modifier, or a ZWJ continuation can remove a boundary. Clear selection in that case instead of extending it. An unchanged byte range may survive Markdown reflow only after the new mapping validates.

Non-prefix replacements clear affected selection. `InvalidateAll` clears source selection because index identities can change. Width/theme changes preserve source bytes and project the highlight through fresh layout. Any failed selected rebuild takes the same no-copy/no-highlight clearing path.

## Alternatives, costs, and delivery

Rendered-row joining is simpler, but cannot preserve hidden syntax or original spacing. A separate source renderer would duplicate wrapping and drift from presentation. Exporting a general source-map API would create more public machinery than this consumer needs. Use private metadata and two leaf declarations instead.

The cost is mapping metadata proportional to laid-out text and explicit invalidation by callers. Conservative clearing can dismiss a selection during Unicode streaming. It avoids broken graphemes and misleading highlights. Complex Markdown transformations need fixtures before the API publishes.

Ship additively with public example tests and updated viewport/Markdown documentation. Test exact slices, reverse drag, auto-scroll, Unicode, entities, logical lines, long clipped code, repeated text, links/tables, composed prefixes, disjoint leaves, empty/failed layouts, mixed legacy items, source/decoration/source, reflow, cache reuse, and all three grapheme-changing append cases. Preserve the existing selection suite. Capture terminal evidence and obtain independent implementation/taste review before the implementation PR.

There are no unresolved behavioral decisions. Root's checkpoint must confirm the public signatures and fail-closed composition contract before implementation.
