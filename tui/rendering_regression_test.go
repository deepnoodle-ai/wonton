package tui

import (
	"bytes"
	"math"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/deepnoodle-ai/wonton/assert"
	"github.com/deepnoodle-ai/wonton/runewidth"
)

// BUG-046: FuzzyMatch must compare runes, not bytes.
func TestFuzzyMatch_MultiByte(t *testing.T) {
	assert.False(t, FuzzyMatch("é", "à©"))
	assert.True(t, FuzzyMatch("é", "café"))
	assert.True(t, FuzzyMatch("cf", "Café"))
}

// BUG-046: a long diff line must clip, not wrap onto the next row.
func TestDiffView_LongLineClips(t *testing.T) {
	diffText := "--- a/f.txt\n+++ b/f.txt\n@@ -1,2 +1,2 @@\n-" + strings.Repeat("x", 60) + "\n \n"
	scrollY := 0
	view, err := DiffViewFromText(diffText, "", &scrollY)
	assert.NoError(t, err)
	term := renderOnTerminal(view, 40, 10)
	for y := range 10 {
		row := ""
		for x := range 40 {
			row += string(term.GetCell(x, y).Char)
		}
		if strings.Contains(row, "xxx") {
			next := ""
			for x := range 40 {
				next += string(term.GetCell(x, y+1).Char)
			}
			assert.False(t, strings.Contains(next, "x"), "wrapped onto next row: %q", next)
			return
		}
	}
	t.Fatal("diff line not rendered")
}

// BUG-046: table rows wider than the container must clip, not wrap.
func TestTableView_OverflowClips(t *testing.T) {
	sel := 0
	cols := []TableColumn{{Title: "A", MinWidth: 15}, {Title: "B", MinWidth: 15}}
	view := Table(cols, &sel).Rows([][]string{{"aaaaaaaaaaaaaaa", "bbbbbbbbbbbbbbb"}})
	term := renderOnTerminal(view, 20, 6)
	for y := range 6 {
		assert.True(t, term.GetCell(0, y).Char != 'b', "row %d starts with wrapped text", y)
	}
}

// renderOnTerminal renders view at frame 0 onto a w x h test terminal.
func renderOnTerminal(view View, w, h int) *Terminal {
	term := NewTestTerminal(w, h, &bytes.Buffer{})
	frame, _ := term.BeginFrame()
	view.size(w, h)
	view.render(NewRenderContext(frame, 0))
	term.EndFrame(frame)
	return term
}

// BUG-004: Progress(...).Shimmer(...) must render without an RGB foreground.
func TestProgress_ShimmerRendersWithoutRGBForeground(t *testing.T) {
	view := Progress(50, 100).Width(10).Shimmer(NewRGB(255, 255, 255), 3)
	assert.NotPanics(t, func() {
		screen := SprintScreen(view, WithWidth(20))
		assert.Equal(t, "█", screen.CellGlyph(0, 0))
	})
}

// BUG-004: a highlight darker than the base color must not wrap around.
func TestProgress_ShimmerBlendsDarkerHighlight(t *testing.T) {
	base := Progress(50, 100).Width(10)
	base.style = base.style.WithFgRGB(NewRGB(200, 200, 200))
	term := renderOnTerminal(base.Shimmer(NewRGB(0, 0, 0), 3), 20, 1)
	// Frame 0 puts the shimmer centre on cell 0: full highlight.
	fg := term.GetCell(0, 0).Style.FgRGB
	assert.NotNil(t, fg)
	assert.Equal(t, NewRGB(0, 0, 0), *fg)
	fg = term.GetCell(1, 0).Style.FgRGB
	assert.NotNil(t, fg)
	assert.True(t, fg.R < 200, "channel must move toward the darker highlight, got %d", fg.R)
}

// BUG-004: a zero speed must not divide by zero.
func TestProgress_AnimatedZeroSpeedDoesNotPanic(t *testing.T) {
	assert.NotPanics(t, func() {
		SprintScreen(Progress(50, 100).Width(10).Shimmer(NewRGB(255, 255, 255), 0), WithWidth(20))
	})
}

// BUG-044: color interpolation must not wrap when a channel decreases.
func TestAnimations_InterpolateDecreasingChannels(t *testing.T) {
	base := NewRGB(100, 100, 100)
	high := NewRGB(255, 0, 0)

	assertBetween := func(t *testing.T, s Style) {
		t.Helper()
		assert.NotNil(t, s.FgRGB)
		assert.True(t, s.FgRGB.G <= 100, "G wrapped: %d", s.FgRGB.G)
		assert.True(t, s.FgRGB.B <= 100, "B wrapped: %d", s.FgRGB.B)
		assert.True(t, s.FgRGB.R >= 100, "R decreased: %d", s.FgRGB.R)
	}

	slide := Slide(1, base, high)
	for i := range 10 {
		assertBetween(t, slide.GetStyle(uint64(i), 0, 5))
	}
	sparkle := Sparkle(1, base, high)
	for i := range 40 {
		assertBetween(t, sparkle.GetStyle(uint64(i), 0, 5))
	}
	sb := &SparkleBorderAnimation{Speed: 1, BaseColor: base, SparkColor: high, Density: 5}
	for i := range 40 {
		assertBetween(t, sb.GetBorderStyle(uint64(i), BorderPartTop, 0, 10))
	}
	ch := &CornerHighlightAnimation{Speed: 1, BaseColor: base, HighlightColor: high}
	for i := range 40 {
		assertBetween(t, ch.GetBorderStyle(uint64(i), BorderPartTop, 1, 10))
	}
}

// BUG-044: GlitchAnimation's brighter flash must saturate, not wrap.
func TestGlitchAnimation_BrightFlashSaturates(t *testing.T) {
	g := Glitch(1, NewRGB(0, 0, 0), NewRGB(250, 250, 250))
	g.intensity = 10
	for frame := range 200 {
		for i := range 10 {
			s := g.GetStyle(uint64(frame), i, 10)
			if s.FgRGB == nil {
				continue
			}
			c := *s.FgRGB
			if c == NewRGB(0, 0, 0) {
				continue
			}
			assert.True(t, c.G >= 250, "glitch G wrapped to %d", c.G)
		}
	}
}

// BUG-044: LerpRGB must clamp easings that overshoot [0, 1].
func TestAnimation_LerpRGBClampsOvershoot(t *testing.T) {
	anim := NewAnimation(10)
	anim.value = 1.2
	got := anim.LerpRGB(NewRGB(0, 0, 0), NewRGB(250, 250, 250))
	assert.Equal(t, NewRGB(255, 255, 255), got)
	anim.value = -0.2
	got = anim.LerpRGB(NewRGB(10, 10, 10), NewRGB(250, 250, 250))
	assert.Equal(t, NewRGB(0, 0, 0), got)
}

// BUG-024: animated text must keep wide characters intact.
func TestAnimatedTextView_WideCharacters(t *testing.T) {
	screen := SprintScreen(Text("世界A").Animate(Rainbow(3)), WithWidth(10))
	assert.Equal(t, "世界A", screen.Row(0))
}

func TestAnimatedText_DrawWideCharacters(t *testing.T) {
	term := NewTestTerminal(20, 3, &bytes.Buffer{})
	frame, _ := term.BeginFrame()
	NewAnimatedText(0, 0, "世界A", Rainbow(3)).Draw(frame)
	ml := NewAnimatedMultiLine(0, 1, 20)
	ml.AddLine("世界A", Rainbow(3))
	ml.Draw(frame)
	term.EndFrame(frame)
	for y := range 2 {
		assert.Equal(t, '世', term.GetCell(0, y).Char, "row %d", y)
		assert.Equal(t, '界', term.GetCell(2, y).Char, "row %d", y)
		assert.Equal(t, 'A', term.GetCell(4, y).Char, "row %d", y)
	}
}

// BUG-024 and BUG-027: the status bar keeps wide values and lazily sizes Rainbow.
func TestAnimatedStatusBar_RainbowWideValue(t *testing.T) {
	term := NewTestTerminal(40, 1, &bytes.Buffer{})
	bar := NewAnimatedStatusBar(0, 0, 40)
	bar.AddItem("S", "世界", "", Rainbow(3), NewStyle())
	frame, _ := term.BeginFrame()
	assert.NotPanics(t, func() { bar.Draw(frame) })
	term.EndFrame(frame)
	// "S: " occupies columns 0-2.
	assert.Equal(t, '世', term.GetCell(3, 0).Char)
	assert.Equal(t, '界', term.GetCell(5, 0).Char)
}

// BUG-027: a narrow FilePicker with a current path must not panic.
func TestFilePicker_NarrowWithPath(t *testing.T) {
	filter := ""
	sel := 0
	items := []ListItem{{Label: "a"}}
	for w := 1; w <= 12; w++ {
		assert.NotPanics(t, func() {
			renderOnTerminal(FilePicker(items, &filter, &sel).CurrentPath("/tmp/some/deep/path"), w, 6)
		}, "width %d", w)
	}
}

// BUG-027: shrinking a table with many narrow columns must not panic.
func TestMarkdown_TableShrinkManyNarrowColumns(t *testing.T) {
	md := "| " + strings.Repeat("w", 200) + " | a | b | c | d | e |\n|---|---|---|---|---|---|\n| x | 1 | 2 | 3 | 4 | 5 |\n"
	mr := NewMarkdownRenderer().WithMaxWidth(80)
	var out *RenderedMarkdown
	assert.NotPanics(t, func() {
		var err error
		out, err = mr.Render(md)
		assert.NoError(t, err)
	})
	for _, line := range strings.Split(renderToPlainText(out), "\n") {
		assert.True(t, runewidth.StringWidth(line) <= 80, "line too wide: %q", line)
	}
}

// BUG-027: a horizontal rule deeper than the width must not panic.
func TestMarkdown_HorizontalRuleNarrowerThanIndent(t *testing.T) {
	mr := NewMarkdownRenderer().WithMaxWidth(2)
	assert.NotPanics(t, func() {
		_, err := mr.Render("> > > ---\n")
		assert.NoError(t, err)
	})
}

// BUG-027: a 1x1 animated border must not divide by zero.
func TestAnimatedBordered_OneByOne(t *testing.T) {
	for _, anim := range []BorderAnimation{PresetRainbowBorder(2, false), &GradientBorderAnimation{}} {
		assert.NotPanics(t, func() {
			renderOnTerminal(AnimatedBordered(Text("x"), anim), 1, 1)
		})
	}
}

// BUG-040: CodeView truncation must not split or overflow grapheme clusters.
func TestTruncateToWidth_GraphemeClusters(t *testing.T) {
	assert.Equal(t, "", truncateToWidth("❤️", 1))
	assert.Equal(t, "👨‍👩‍👧", truncateToWidth("👨‍👩‍👧b", 2))
	assert.Equal(t, "ab", truncateToWidth("ab世", 3))
}

func TestCodeView_TruncatesClusterAtEdge(t *testing.T) {
	term := renderOnTerminal(Code("a👨‍👩‍👧b", "text").LineNumbers(false), 3, 1)
	assert.Equal(t, 'a', term.GetCell(0, 0).Char)
	assert.Equal(t, '👨', term.GetCell(1, 0).Char)
}

// BUG-040: tab stops must use cluster widths, like the rest of CodeView.
func TestCodeView_ExpandTabsAfterCluster(t *testing.T) {
	c := Code("", "text").TabWidth(4)
	// "❤️" is 2 columns wide, so the tab advances 2 more columns.
	assert.Equal(t, "❤️  x", c.expandTabs("❤️\tx", 0))
}

// BUG-045: nil children must not crash a tree.
func TestTreeView_NilChild(t *testing.T) {
	root := NewTreeNode("root").SetExpanded(true).AddChildren(nil, NewTreeNode("leaf"))
	assert.NotPanics(t, func() {
		root.ExpandAll()
		root.CollapseAll()
		root.Expanded = true
		tree := Tree(root)
		w, h := Measure(tree, 80, 24)
		assert.Equal(t, 2, h, "width %d", w)
		renderOnTerminal(tree, 20, 5)
		assert.NotNil(t, tree.FindNode("leaf"))
		assert.Nil(t, tree.FindNode("missing"))
	})
}

// BUG-045: easings must stay finite and in range for t outside [0, 1].
func TestEasings_OutOfRangeInput(t *testing.T) {
	chain := EaseChain(EaseLinear, EaseLinear)
	assert.NotPanics(t, func() { chain(-0.5) })
	for _, ease := range []Easing{EaseInCirc, EaseOutCirc, EaseInOutCirc} {
		for _, in := range []float64{-0.5, 1.5} {
			v := ease(in)
			assert.False(t, math.IsNaN(v), "NaN for t=%v", in)
		}
	}
}

// BUG-041: a long panel title must not overwrite the top-right corner.
func TestPanel_LongTitleKeepsCorner(t *testing.T) {
	term := renderOnTerminal(Panel(Text("x")).Border(BorderSingle).Title("LongTitle").Width(10), 10, 3)
	assert.Equal(t, '┐', term.GetCell(9, 0).Char)
}

// BUG-041: a long text area title must not overwrite the top-right corner.
func TestTextArea_LongTitleKeepsCorner(t *testing.T) {
	content := "x"
	term := renderOnTerminal(TextArea(&content).Bordered().Title("A very long title").Size(10, 3), 10, 3)
	assert.Equal(t, '╮', term.GetCell(9, 0).Char)
}

// BUG-041: a multi-byte animated border title must not split a rune.
func TestAnimatedBordered_MultiByteTitle(t *testing.T) {
	term := renderOnTerminal(AnimatedBordered(Text("x"), PresetRainbowBorder(2, false)).Title("日本語タイトル"), 8, 3)
	// The title area is w-4 = 4 columns: two wide runes.
	assert.Equal(t, '日', term.GetCell(2, 0).Char)
	assert.Equal(t, '本', term.GetCell(4, 0).Char)
	assert.True(t, term.GetCell(6, 0).Char != utf8.RuneError, "title split a rune")
}

// positionBorderAnimation colors each border cell with its position, so a
// test can read the position back from the rendered cell.
type positionBorderAnimation struct{}

func (positionBorderAnimation) GetBorderStyle(frame uint64, part BorderPart, position, length int) Style {
	return NewStyle().WithFgRGB(NewRGB(uint8(position), uint8(length), 0))
}

// Corners take their place on the perimeter, so a rainbow flows through
// them instead of drawing every corner in the first color.
func TestAnimatedBordered_CornersFollowPerimeter(t *testing.T) {
	term := renderOnTerminal(AnimatedBordered(Text("x"), positionBorderAnimation{}), 5, 4)
	// Clockwise from the top-left corner.
	walk := [][2]int{
		{0, 0}, {1, 0}, {2, 0}, {3, 0}, {4, 0},
		{4, 1}, {4, 2}, {4, 3},
		{3, 3}, {2, 3}, {1, 3}, {0, 3},
		{0, 2}, {0, 1},
	}
	for i, p := range walk {
		fg := term.GetCell(p[0], p[1]).Style.FgRGB
		assert.NotNil(t, fg)
		assert.Equal(t, int(fg.R), i, "position at %v", p)
		assert.Equal(t, int(fg.G), len(walk), "length at %v", p)
	}
}

// The lit corner fades into the edges on both sides of it.
func TestCornerHighlight_FadesIntoBothEdges(t *testing.T) {
	base, high := NewRGB(0, 0, 0), NewRGB(250, 250, 250)
	anim := &CornerHighlightAnimation{Speed: 1, BaseColor: base, HighlightColor: high, Duration: 10}
	w, h := 14, 8
	// For each corner in turn: the corner, its neighbor on each edge, and
	// a cell in the middle of the top edge.
	cases := []struct{ corner, before, after [2]int }{
		{[2]int{0, 0}, [2]int{0, 1}, [2]int{1, 0}},
		{[2]int{w - 1, 0}, [2]int{w - 2, 0}, [2]int{w - 1, 1}},
		{[2]int{w - 1, h - 1}, [2]int{w - 1, h - 2}, [2]int{w - 2, h - 1}},
		{[2]int{0, h - 1}, [2]int{1, h - 1}, [2]int{0, h - 2}},
	}
	for i, c := range cases {
		term := NewTestTerminal(w, h, &bytes.Buffer{})
		frame, _ := term.BeginFrame()
		AnimatedBordered(Text("x"), anim).render(NewRenderContext(frame, uint64(i*10)))
		term.EndFrame(frame)
		for _, p := range [][2]int{c.corner, c.before, c.after} {
			assert.Equal(t, *term.GetCell(p[0], p[1]).Style.FgRGB, high, "corner %d, cell %v", i, p)
		}
		if i == 0 {
			// (2, 0) is one cell into the top-left fade: blended, not lit.
			fade := *term.GetCell(2, 0).Style.FgRGB
			assert.NotEqual(t, fade, high, "fade cell (2, 0) fully lit")
			assert.NotEqual(t, fade, base, "fade cell (2, 0) not lit")
		}
		assert.Equal(t, *term.GetCell(w/2, 0).Style.FgRGB, base, "corner %d, mid-edge", i)
	}
}

// Wrappers that leave a view's size alone must keep it flexible, or a Stack
// measures it unconstrained and a Scroll's full content pushes the rest off.
func TestWrappersKeepFlex(t *testing.T) {
	lines := make([]View, 50)
	for i := range lines {
		lines[i] = Text("line")
	}
	wrappers := map[string]func(View) View{
		"FocusableView": func(v View) View { return FocusableView(v, &FocusHandler{ID: "f"}) },
		"AnimatedBordered": func(v View) View {
			return AnimatedBordered(v, &RainbowBorderAnimation{})
		},
		"MinHeight": func(v View) View { return MinHeight(2, v) },
	}
	for name, wrap := range wrappers {
		scrollY := 0
		view := Stack(Text("Header"), wrap(Scroll(Stack(lines...), &scrollY)), Text("Footer"))
		screen := SprintScreen(view, WithWidth(20), WithHeight(10))
		assert.Equal(t, screen.Row(9), "Footer", name)
	}
}

func TestGroupScalesFlexibleMinimumsToFit(t *testing.T) {
	view := Group(Stack(Text("%s", strings.Repeat("x", 40))).Flex(1), Text("END"))
	screen := SprintScreen(view, WithWidth(20), WithHeight(1))
	assert.Equal(t, screen.Row(0), strings.Repeat("x", 17)+"END")
}

func TestPanelMeasuresContentInsideBorderAndWidth(t *testing.T) {
	view := Panel(Text("one two three four").Wrap()).Width(10).Border(BorderSingle)
	w, h := view.size(80, 0)
	assert.Equal(t, w, 10)
	assert.Equal(t, h, 5, "text wraps to the 8 columns inside the border")
}

func TestScrollResetsOffsetWhenContentFits(t *testing.T) {
	scrollY := 7
	SprintScreen(Scroll(Text("short"), &scrollY), WithWidth(20), WithHeight(5))
	assert.Equal(t, scrollY, 0)
}
