# Select original transcript text

A viewport can copy original source bytes through styled text, Markdown, and wrapping. Implement `tui.ViewportSourceItems` beside the ordinary item interface. `Source(i)` returns the canonical UTF-8 text and `true` for that item. Mark each source-bearing Text or Markdown leaf with `.SourceOffset(offset)`.

```go
func (items Transcript) Source(i int) (string, bool) {
    return items[i].Text, true
}

func (items Transcript) Item(i int) tui.View {
    return tui.Group(
        tui.Text("assistant: "), // decoration
        tui.Markdown(items[i].Text, nil).SourceOffset(0),
    )
}
```

The offset declares the exact beginning of the leaf's text in the canonical source. Multiple leaves may show disjoint slices of that source. Invalid bindings clear affected selections; they never fall back to copying styled cells. Unmarked children, role labels, padding, and generated Markdown borders contribute no source bytes.

Use the existing `ViewportState.HandleMouse`, `SelectedText`, and `ClearSelection` methods. Drag selection copies the exact source slice between its endpoints. Hidden Markdown syntax inside the range remains. Double-click selects a terminal word across wraps. Triple-click selects the original logical line, including indentation and trailing whitespace, excluding its line terminator. Copying a wide or combined character preserves whole graphemes.

Sources and bindings are immutable until `Invalidate(i)`. Reordering or replacing item identities requires `InvalidateAll`. Resizing preserves copied bytes and moves their highlight. Streaming appends preserve selection while its endpoints remain grapheme boundaries; an appended combining mark or ZWJ can clear selection. A failed selected layout clears the whole selection and restores Follow. Cross-item copy joins contributing items with one LF; decoration-only items add no separator.

`Selection()` still describes current rendered coordinates. Do not use those coordinates as source offsets. A source-aware selection retains byte endpoints internally. An item returning `false`, or an ordinary `ViewportItems` implementation, retains legacy cell copying. Return `"", true` for decoration-only items.

Source-aware text renders controls as inert visible replacements. The copied string remains data, including any original controls. Sanitize it before printing it as native terminal output. Applications continue to own clipboard transport and mouse-capture policy.

A runnable `ExampleViewportSourceItems` demonstrates copying unchanged source before and after reflow.
