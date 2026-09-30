package tui

import (
	"fmt"
	"strings"
	"testing"
)

type sourceItems struct {
	text     []string
	markdown bool
	broken   map[int]bool
	legacy   map[int]bool
	builds   int
}

func (v *sourceItems) Len() int                    { return len(v.text) }
func (v *sourceItems) Source(i int) (string, bool) { return v.text[i], !v.legacy[i] }
func (v *sourceItems) Item(i int) View {
	v.builds++
	if v.broken[i] {
		return Text("unbound source")
	}
	if v.legacy[i] {
		return Text("%s", v.text[i])
	}
	var body View = Text("%s", v.text[i]).SourceOffset(0).Wrap()
	if v.markdown {
		body = Markdown(v.text[i], nil).SourceOffset(0)
	}
	return PaddingLTRB(2, 0, 0, 0, body)
}
func newSourceViewport(t *testing.T, source string, markdown bool, width int) (*ViewportState, *sourceItems) {
	t.Helper()
	v := &sourceItems{text: []string{source}, markdown: markdown}
	s := &ViewportState{}
	renderViewport(t, s, v, width, 20, 0)
	return s, v
}
func TestSourceLogicalLinePreservesWhitespaceAndWraps(t *testing.T) {
	source := "  世界\t  " + strings.Repeat("source ", 28) + "END  "
	for _, markdown := range []bool{false, true} {
		s, _ := newSourceViewport(t, source, markdown, 30)
		s.SelectLine(2, 0)
		if got := s.SelectedText(); got != source {
			t.Fatalf("markdown=%v: want %q; got %q", markdown, source, got)
		}
	}
}
func TestSourceDragAndReversePreserveOriginalBytes(t *testing.T) {
	s, _ := newSourceViewport(t, "a  b\t c\r\nd  e  ", false, 7)
	s.BeginSelection(2, 0)
	s.ExtendSelection(6, 3)
	s.EndSelection()
	got := s.SelectedText()
	if !strings.Contains(got, "\t") || !strings.Contains(got, "\r\n") || strings.Contains(got, "assistant") {
		t.Fatalf("source bytes lost: %q", got)
	}
	s.BeginSelection(6, 3)
	s.ExtendSelection(2, 0)
	s.EndSelection()
	if reverse := s.SelectedText(); reverse != got {
		t.Fatalf("reverse %q != %q", reverse, got)
	}
}
func TestSourceMarkdownHiddenSyntaxAndEntity(t *testing.T) {
	s, _ := newSourceViewport(t, "**bold**", true, 30)
	s.SelectWord(3, 0)
	if got := s.SelectedText(); got != "bold" {
		t.Fatalf("word=%q", got)
	}
	s.SelectLine(3, 0)
	if got := s.SelectedText(); got != "**bold**" {
		t.Fatalf("line=%q", got)
	}
	s, _ = newSourceViewport(t, "[docs](url) next", true, 40)
	s.SelectWord(3, 0)
	if got := s.SelectedText(); got != "docs" {
		t.Fatalf("link word=%q", got)
	}
	s.BeginSelection(2, 0)
	s.ExtendSelection(11, 0)
	s.EndSelection()
	if got := s.SelectedText(); got != "docs](url) next" {
		t.Fatalf("link drag=%q", got)
	}
	s, _ = newSourceViewport(t, "&amp;", true, 30)
	s.BeginSelection(2, 0)
	s.ExtendSelection(3, 0)
	s.EndSelection()
	if got := s.SelectedText(); got != "&amp;" {
		t.Fatalf("entity=%q", got)
	}
}
func TestSourceSelectionSurvivesReflowAndPrefixAppend(t *testing.T) {
	s, v := newSourceViewport(t, "one two three four", false, 12)
	s.SelectLine(3, 0)
	want := s.SelectedText()
	renderViewport(t, s, v, 7, 20, 0)
	if got := s.SelectedText(); got != want {
		t.Fatalf("reflow %q", got)
	}
	v.text[0] += " more"
	s.Invalidate(0)
	renderViewport(t, s, v, 7, 20, 0)
	if got := s.SelectedText(); got != want {
		t.Fatalf("append %q", got)
	}
	v.text[0] = "replacement"
	s.Invalidate(0)
	renderViewport(t, s, v, 7, 20, 0)
	if got := s.SelectedText(); got != "" || s.HasSelection() {
		t.Fatalf("replacement kept %q", got)
	}
}
func TestSourceAppendChangedGraphemeClearsSelection(t *testing.T) {
	for _, pair := range [][2]string{{"e", "\u0301"}, {"👍", "🏽"}, {"👩", "\u200d💻"}} {
		t.Run(fmt.Sprintf("%q", pair[0]), func(t *testing.T) {
			s, v := newSourceViewport(t, pair[0], false, 20)
			s.Follow = true
			s.SelectLine(2, 0)
			if s.SelectedText() != pair[0] {
				t.Fatal("selection missing")
			}
			v.text[0] += pair[1]
			s.Invalidate(0)
			renderViewport(t, s, v, 20, 20, 0)
			if got := s.SelectedText(); got != "" || s.HasSelection() || !s.Follow {
				t.Fatalf("grapheme append retained %q follow=%v", got, s.Follow)
			}
		})
	}
}
func TestSourceIntermediateFailureAndDecoration(t *testing.T) {
	v := &sourceItems{text: []string{"first", "", "second"}}
	s := &ViewportState{Follow: true}
	renderViewport(t, s, v, 20, 20, 0)
	s.BeginSelection(2, 0)
	s.ExtendSelection(8, 2)
	s.EndSelection()
	if got := s.SelectedText(); got != "first\nsecond" {
		t.Fatalf("decoration separator: %q", got)
	}
	v.text[1] = "invalid"
	v.broken = map[int]bool{1: true}
	s.Invalidate(1)
	renderViewport(t, s, v, 20, 20, 0)
	if got := s.SelectedText(); got != "" || s.HasSelection() || !s.Follow {
		t.Fatalf("failed layout copied %q", got)
	}
}
func TestSourcePrefixesAreDecorationAndCachesAreReused(t *testing.T) {
	s, v := newSourceViewport(t, "content", false, 20)
	s.SelectLine(0, 0)
	if s.HasSelection() {
		t.Fatal("decoration selected source")
	}
	s.SelectLine(3, 0)
	for range 4 {
		renderViewport(t, s, v, 20, 20, 0)
		if s.SelectedText() != "content" {
			t.Fatal("copy changed")
		}
	}
	if v.builds != 1 {
		t.Fatalf("rebuilt unchanged item %d times", v.builds)
	}
	s.InvalidateAll()
	renderViewport(t, s, v, 20, 20, 0)
	if s.HasSelection() {
		t.Fatal("wholesale replacement kept source selection")
	}
}
func TestSourceUnicodeCompleteGraphemes(t *testing.T) {
	for _, g := range []string{"世", "e\u0301", "👩\u200d💻", "👍🏽"} {
		s, _ := newSourceViewport(t, g+" next", false, 30)
		s.BeginSelection(2, 0)
		s.ExtendSelection(3, 0)
		s.EndSelection()
		got := s.SelectedText()
		if got != g {
			t.Fatalf("%q split as %q", g, got)
		}
	}
}

func TestSourceMarkdownStructuresRetainLogicalLines(t *testing.T) {
	cases := []struct{ source, line string }{
		{"# heading", "# heading"},
		{"- repeated repeated", "- repeated repeated"},
		{"> quoted **word**", "> quoted **word**"},
		{"`inline code`", "`inline code`"},
		{"\\*escaped*", "\\*escaped*"},
		{"```go\nvar answer = 42\n```", "var answer = 42"},
		{"```text\n" + strings.Repeat("long", 50), strings.Repeat("long", 50)},
		{"| name | value |\n| --- | --- |\n| same | same |", "| name | value |"},
	}
	for _, tc := range cases {
		t.Run(tc.line[:min(len(tc.line), 25)], func(t *testing.T) {
			s, v := newSourceViewport(t, tc.source, true, 30)
			l, aware := s.sourceLayout(0)
			if !aware || !l.valid || len(l.cells) == 0 {
				t.Fatalf("missing provenance for %q: %+v", tc.source, l)
			}
			var first sourceCell
			found := false
			for _, cell := range l.cells {
				if !found || cell.y < first.y || cell.y == first.y && cell.x < first.x {
					first = cell
					found = true
				}
			}
			s.SelectLine(first.x, first.y)
			if got := s.SelectedText(); got != tc.line {
				t.Fatalf("want %q; got %q", tc.line, got)
			}
			renderViewport(t, s, v, 12, 20, 0)
			if got := s.SelectedText(); got != tc.line {
				t.Fatalf("narrow changed line to %q", got)
			}
		})
	}
}

type disjointSourceItems struct {
	source string
	bad    bool
}

func (d disjointSourceItems) Len() int                  { return 1 }
func (d disjointSourceItems) Source(int) (string, bool) { return d.source, true }
func (d disjointSourceItems) Item(int) View {
	offset := 6
	if d.bad {
		offset = 0
	}
	return Stack(Text("first").SourceOffset(0), Group(Text("! "), Text("second").SourceOffset(offset))).Gap(0)
}
func TestSourceDisjointLeavesAndFailedDeclarations(t *testing.T) {
	items := disjointSourceItems{source: "first\nsecond"}
	s := &ViewportState{Follow: true}
	renderViewport(t, s, items, 30, 20, 0)
	s.BeginSelection(0, 0)
	s.ExtendSelection(8, 1)
	s.EndSelection()
	if got := s.SelectedText(); got != "first\nsecond" {
		t.Fatalf("disjoint leaves: %q", got)
	}
	items.bad = true
	s.Invalidate(0)
	renderViewport(t, s, items, 30, 20, 0)
	if got := s.SelectedText(); got != "" || s.HasSelection() {
		t.Fatalf("invalid binding copied %q", got)
	}
}
func TestSourceMixedLegacySelection(t *testing.T) {
	v := &sourceItems{text: []string{"source  ", "legacy  ", "last"}, legacy: map[int]bool{1: true}}
	s := &ViewportState{}
	renderViewport(t, s, v, 20, 20, 0)
	s.BeginSelection(2, 0)
	s.ExtendSelection(6, 2)
	s.EndSelection()
	if got := s.SelectedText(); got != "source  \nlegacy\nlast" {
		t.Fatalf("mixed copy %q", got)
	}
}
func TestSourceControlPresentationIsInert(t *testing.T) {
	for _, md := range []bool{false, true} {
		s, _ := newSourceViewport(t, "before\x1b[31mafter", md, 40)
		l, _ := s.sourceLayout(0)
		if !l.valid {
			t.Fatal("control source lost provenance")
		}
		s.SelectLine(3, 0)
		if got := s.SelectedText(); got != "before\x1b[31mafter" {
			t.Fatalf("inert source bytes changed: %q", got)
		}
	}
}

func TestSourceIntermediateAppendDoesNotExtendCopiedRange(t *testing.T) {
	v := &sourceItems{text: []string{"first", "middle", "last"}}
	s := &ViewportState{}
	renderViewport(t, s, v, 30, 20, 0)
	s.BeginSelection(2, 0)
	s.ExtendSelection(6, 2)
	s.EndSelection()
	want := s.SelectedText()
	v.text[1] += " appended"
	s.Invalidate(1)
	renderViewport(t, s, v, 30, 20, 0)
	if got := s.SelectedText(); got != want {
		t.Fatalf("intermediate append extended copy: %q => %q", want, got)
	}
}
func TestSourceWordSelectionCrossesVisualWraps(t *testing.T) {
	source := strings.Repeat("a", 50) + " trailing"
	s, _ := newSourceViewport(t, source, false, 12)
	s.SelectWord(3, 2)
	if got := s.SelectedText(); got != strings.Repeat("a", 50) {
		t.Fatalf("wrapped word %q", got)
	}
}
func TestSourceDragAutoScrollAndHighlightAfterResize(t *testing.T) {
	v := &sourceItems{text: []string{strings.Repeat("row 世界\n", 60)}}
	s := &ViewportState{Follow: true}
	renderViewport(t, s, v, 20, 5, 0)
	s.ScrollToTop()
	renderViewport(t, s, v, 20, 5, 0)
	s.BeginSelection(2, 0)
	s.ExtendSelection(8, 6)
	for range 15 {
		s.DragAutoScroll()
		renderViewport(t, s, v, 20, 5, 0)
	}
	s.EndSelection()
	got := s.SelectedText()
	if strings.Count(got, "\n") < 10 {
		t.Fatalf("auto-scroll did not grow source selection: %q", got)
	}
	screen := renderViewport(t, s, v, 12, 5, 0)
	if s.SelectedText() != got || len(reversedRuns(screen, 12, 5)) == 0 {
		t.Fatal("resize lost source or highlight")
	}
}
