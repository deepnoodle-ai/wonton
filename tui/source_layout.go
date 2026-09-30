package tui

import (
	"cmp"
	"image"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/deepnoodle-ai/wonton/runewidth"
)

// ViewportSourceItems optionally supplies the original text of a viewport item.
// Source-bearing Text and Markdown leaves must declare their SourceOffset.
// Sources and bindings are immutable until Invalidate(i). Reordering items
// requires InvalidateAll. false preserves legacy rendered-cell selection;
// true with an empty source denotes decoration-only content.
type ViewportSourceItems interface {
	ViewportItems
	Source(i int) (source string, ok bool)
}

type sourceToken struct {
	lo, hi     int // byte interval in displayed segment
	start, end int // byte interval in canonical source
}
type sourceCell struct {
	x, y, width, start, end int
	text                    string
}
type sourceWordToken struct {
	start, end int
	text       string
}
type sourceBinding struct{ start, end int }
type sourceLayout struct {
	source     string
	cells      map[image.Point]sourceCell
	bindings   []sourceBinding
	words      []sourceWordToken
	valid      bool
	bounds     image.Rectangle
	boundaries []int
}
type sourceEndpoint struct {
	item, offset int
	source       string
}

func newSourceLayout(source string, width, height int) *sourceLayout {
	l := &sourceLayout{source: source, valid: utf8.ValidString(source), cells: make(map[image.Point]sourceCell), bounds: image.Rect(0, 0, width, height), boundaries: []int{0}}
	n := 0
	for g := range runewidth.Graphemes(source) {
		n += len(g)
		l.boundaries = append(l.boundaries, n)
	}
	return l
}
func (l *sourceLayout) bind(content string, offset int) {
	end := offset + len(content)
	if offset < 0 || end < offset || end > len(l.source) || l.source[offset:end] != content || !l.boundary(offset) || !l.boundary(end) {
		l.valid = false
		return
	}
	for _, b := range l.bindings {
		if offset < b.end && end > b.start {
			l.valid = false
			return
		}
	}
	l.bindings = append(l.bindings, sourceBinding{offset, end})
}
func (l *sourceLayout) complete() bool { return l.valid && (l.source == "" || len(l.bindings) > 0) }
func sourceBoundary(source string, offset int) bool {
	if offset == 0 || offset == len(source) {
		return true
	}
	if offset < 0 || offset > len(source) {
		return false
	}
	n := 0
	for g := range runewidth.Graphemes(source) {
		n += len(g)
		if n == offset {
			return true
		}
		if n > offset {
			return false
		}
	}
	return false
}
func sourceLine(source string, at int) (int, int) {
	at = min(max(at, 0), len(source))
	start := strings.LastIndex(source[:at], "\n") + 1
	end := len(source)
	if n := strings.IndexByte(source[at:], '\n'); n >= 0 {
		end = at + n
	}
	if end > start && source[end-1] == '\r' {
		end--
	}
	return start, end
}

// originalSegment carries positions before wrapping, decoding, or styling.
func originalSegment(text string, start int, style Style) StyledSegment {
	seg := StyledSegment{Text: text, Style: style}
	n := 0
	for g := range runewidth.Graphemes(text) {
		seg.source = append(seg.source, sourceToken{lo: n, hi: n + len(g), start: start + n, end: start + n + len(g)})
		n += len(g)
	}
	return seg
}
func segmentSlice(seg StyledSegment, lo, hi int) StyledSegment {
	out := seg
	out.Text = seg.Text[lo:hi]
	out.source = nil
	first, _ := slices.BinarySearchFunc(seg.source, lo, func(t sourceToken, n int) int { return cmp.Compare(t.lo, n) })
	if first > 0 && seg.source[first-1].hi > lo {
		first--
	}
	for _, t := range seg.source[first:] {
		if t.lo >= hi {
			break
		}
		if t.hi > lo {
			t.lo, t.hi = max(t.lo, lo)-lo, min(t.hi, hi)-lo
			out.source = append(out.source, t)
		}
	}
	return out
}
func appendSegment(a, b StyledSegment) StyledSegment {
	n := len(a.Text)
	a.Text += b.Text

	for _, t := range b.source {
		t.lo += n
		t.hi += n
		a.source = append(a.source, t)
	}
	return a
}

// sourceTextLines preserves whitespace and splits only at grapheme boundaries.
// Tabs have a fixed four-column presentation; the source tab remains one token.
func sourceTextLines(text string, offset, width int, wrap bool) [][]StyledSegment {
	var rows [][]StyledSegment
	var shownRow strings.Builder
	var tokens []sourceToken
	col, n := 0, 0
	flush := func() {
		rows = append(rows, []StyledSegment{{Text: shownRow.String(), source: tokens}})
		shownRow = strings.Builder{}
		tokens = nil
		col = 0
	}
	for g, w := range runewidth.Graphemes(text) {
		start := n
		n += len(g)
		if g == "\n" || g == "\r\n" {
			flush()
			continue
		}
		shown := g
		if g == "\t" {
			shown = "    "
			w = 4
		} else {
			for _, r := range g {
				if sourceUnsafeRune(r) {
					shown, w = "�", 1
					break
				}
			}
		}
		if wrap && width > 0 && col+w > width && col > 0 {
			flush()
		}
		lo := shownRow.Len()
		shownRow.WriteString(shown)
		tokens = append(tokens, sourceToken{lo, shownRow.Len(), offset + start, offset + n})
		col += w
	}
	flush()
	return rows
}
func (c *RenderContext) clearSource(x, y, width, height int) {
	if c.source == nil {
		return
	}
	r := image.Rect(x, y, x+width, y+height).Add(c.sourceOrigin)
	for yy := max(r.Min.Y, 0); yy < min(r.Max.Y, c.source.bounds.Max.Y); yy++ {
		for xx := max(r.Min.X, 0); xx < min(r.Max.X, c.source.bounds.Max.X); xx++ {
			point := image.Pt(xx, yy)
			if cell, ok := c.source.cells[point]; ok {
				for col := cell.x; col < cell.x+cell.width; col++ {
					delete(c.source.cells, image.Pt(col, yy))
				}
			}
		}
	}
}
func (c *RenderContext) recordSegment(x, y int, seg StyledSegment, base int) {
	if c.source == nil {
		return
	}
	n, col := 0, x
	width, height := c.Size()
	for g, w := range runewidth.Graphemes(seg.Text) {
		if y >= 0 && y < height && col >= 0 && col+w <= width {
			for _, token := range seg.source {
				if token.lo <= n && token.hi >= n+len(g) {
					lo, hi := token.start+base, token.end+base
					if lo < 0 || hi > len(c.source.source) || lo >= hi {
						c.source.valid = false
						continue
					}
					lo, hi = c.source.cluster(lo, hi)
					point := image.Pt(col, y).Add(c.sourceOrigin)
					if point.In(c.source.bounds) && point.X+w <= c.source.bounds.Max.X {
						cell := sourceCell{point.X, point.Y, max(w, 1), lo, hi, g}
						for col := point.X; col < point.X+max(w, 1); col++ {
							c.source.cells[image.Pt(col, point.Y)] = cell
						}
					}
					break
				}
			}
		}
		n += len(g)
		col += w
	}
}
func (l *sourceLayout) hit(x, y int, clamp bool) (int, bool) {
	for _, cell := range l.cells {
		if cell.y == y && x >= cell.x && x < cell.x+cell.width {
			if clamp && x > cell.x {
				return cell.end, true
			}
			return cell.start, true
		}
	}
	if !clamp || len(l.cells) == 0 {
		return 0, false
	}
	var before, after *sourceCell
	for _, value := range l.cells {
		t := value

		if t.y < y || t.y == y && t.x < x {
			if before == nil || t.y > before.y || t.y == before.y && t.x > before.x {
				before = &t
			}
		} else if after == nil || t.y < after.y || t.y == after.y && t.x < after.x {
			after = &t
		}
	}
	if before != nil {
		return before.end, true
	}
	return after.start, true
}
func (l *sourceLayout) point(offset int, end bool) (line, col int) {
	var first, last *sourceCell
	for _, value := range l.cells {
		t := value

		if t.start <= offset && offset < t.end {
			return t.y, t.x
		}
		if t.start >= offset && (first == nil || t.start < first.start) {
			first = &t
		}
		if t.end <= offset && (last == nil || t.end > last.end) {
			last = &t
		}
	}
	if end && last != nil {
		return last.y, last.x + last.width
	}
	if first != nil {
		return first.y, first.x
	}
	if last != nil {
		return last.y, last.x + last.width
	}
	return 0, 0
}
func (s *ViewportState) sourceLayout(item int) (*sourceLayout, bool) {
	items, ok := s.items.(ViewportSourceItems)
	if !ok {
		return nil, false
	}
	e := s.entry(item)
	if e.sourceValid {
		return e.source, e.source != nil
	}
	source, aware := items.Source(item)
	if !aware {
		e.sourceValid = true
		return nil, false
	}
	view := s.viewOf(item)
	height := s.heightOf(item)
	l := newSourceLayout(source, s.width, height)
	if view != nil && s.width > 0 && height > 0 {
		term := NewTestTerminal(s.width, height, &strings.Builder{})
		frame, err := term.BeginFrame()
		if err == nil {
			ctx := NewRenderContext(frame, 0)
			ctx.source = l
			view.render(ctx)
			term.EndFrame(frame)
		} else {
			l.valid = false
		}
	}
	collectSourceBindings(view, l)
	if !l.complete() {
		l.valid = false
	}
	e = s.entry(item)
	e.source, e.sourceValid = l, true
	return l, true
}
func (s *ViewportState) sourceHit(p SelectionPoint, clamp, continuationEnd bool) (*sourceEndpoint, bool) {
	l, aware := s.sourceLayout(p.Item)
	if !aware {
		return nil, true
	}
	if !l.valid {
		return nil, false
	}
	if clamp && len(l.cells) == 0 {
		offset := len(l.source)
		if p.Item < s.selAnchor.Item {
			offset = 0
		}
		return &sourceEndpoint{item: p.Item, offset: offset, source: l.source}, true
	}
	offset, ok := l.hit(p.Col, p.Line, clamp)
	if ok {
		for _, cell := range l.cells {
			if cell.y != p.Line || p.Col < cell.x || p.Col >= cell.x+cell.width {
				continue
			}
			continuation := p.Col > cell.x
			for _, previous := range l.cells {
				if previous.start == cell.start && previous.end == cell.end &&
					(previous.y < cell.y || previous.y == cell.y && previous.x < cell.x) {
					continuation = true
					break
				}
			}
			if continuation {
				offset = cell.start
				if continuationEnd {
					offset = cell.end
				}
			}
			break
		}
	}
	if !ok {
		return nil, false
	}
	return &sourceEndpoint{item: p.Item, offset: offset, source: l.source}, true
}
func (s *ViewportState) validateSourceSelection() bool {
	for _, p := range []*sourceEndpoint{s.sourceAnchor, s.sourceCursor} {
		if p == nil {
			continue
		}
		l, aware := s.sourceLayout(p.item)
		if !aware || !l.valid || !strings.HasPrefix(l.source, p.source) || !sourceBoundary(l.source, p.offset) {
			s.ClearSelection()
			return false
		}
	}
	start, end := s.selAnchor.Item, s.selCursor.Item
	if start > end {
		start, end = end, start
	}
	for i := start; i <= end; i++ {
		l, aware := s.sourceLayout(i)
		if _, selected := s.sourceSnapshots[i]; selected && !aware {
			s.ClearSelection()
			return false
		}
		if aware {
			old, selected := s.sourceSnapshots[i]
			if !selected || !l.valid || selected && !strings.HasPrefix(l.source, old) || selected && i > start && i < end && !l.boundary(len(old)) {
				s.ClearSelection()
				return false
			}
		}
	}
	return true
}
func (s *ViewportState) sourceSelectionPoint(p SelectionPoint, source *sourceEndpoint, end bool) SelectionPoint {
	if source == nil {
		return p
	}
	l, _ := s.sourceLayout(source.item)
	line, col := l.point(source.offset, end)
	return SelectionPoint{source.item, line, col}
}
func (s *ViewportState) sourceOrdered() (start, end *sourceEndpoint) {
	start, end = s.sourceAnchor, s.sourceCursor
	if start != nil && end != nil && (start.item > end.item || start.item == end.item && start.offset > end.offset) {
		start, end = end, start
	}
	return start, end
}
func (s *ViewportState) sourceSelectRun(p SelectionPoint, line bool) bool {
	l, aware := s.sourceLayout(p.Item)
	if !aware {
		return false
	}
	if !l.valid {
		s.ClearSelection()
		return true
	}
	offset, ok := l.hit(p.Col, p.Line, false)
	if !ok {
		return true
	}
	lo, hi := sourceLine(l.source, offset)
	if !line {
		// Grow over mapped original graphemes only. Hidden Markdown syntax is
		// outside a word; it remains inside drag slices between endpoints.
		type word struct {
			start, end int
			text       string
		}
		var words []word
		seen := map[int]bool{}
		for _, token := range l.words {
			if token.start < lo || token.end > hi || seen[token.start] {
				continue
			}
			seen[token.start] = true
			words = append(words, word{token.start, token.end, token.text})
		}
		// Layout cell order can differ from source order (tables and wrapping).
		slices.SortFunc(words, func(a, b word) int { return cmp.Compare(a.start, b.start) })
		at := -1
		for i, w := range words {
			if w.start <= offset && offset < w.end {
				at = i
				break
			}
		}
		if at < 0 || !isWordRune(words[at].text) {
			return true
		}
		first, last := at, at
		for first > 0 && isWordRune(words[first-1].text) {
			first--
		}
		for last+1 < len(words) && isWordRune(words[last+1].text) {
			last++
		}
		lo, hi = words[first].start, words[last].end
	}
	if lo == hi {
		return true
	}
	s.suspendFollow()
	s.selAnchor, s.selCursor = p, p
	s.sourceAnchor = &sourceEndpoint{p.Item, lo, l.source}
	s.sourceCursor = &sourceEndpoint{p.Item, hi, l.source}
	s.selecting = false
	s.hasSelection = true
	s.captureSourceSelection()
	return true
}

func (c *RenderContext) clearSourceText(x, y int, text string, wrap bool) {
	if c.source == nil {
		return
	}
	w, h := c.Size()
	start := x
	for g, width := range runewidth.Graphemes(text) {
		if g == "\n" || g == "\r\n" {
			y++
			x = start
			continue
		}
		if wrap && x+width > w {
			y++
			x = 0
		}
		if y >= 0 && y < h && x >= 0 && x+width <= w {
			c.clearSource(x, y, max(width, 1), 1)
		}
		x += width
	}
}

func (l *sourceLayout) boundary(offset int) bool {
	_, ok := slices.BinarySearch(l.boundaries, offset)
	return ok
}
func (l *sourceLayout) cluster(start, end int) (int, int) {
	lo, found := slices.BinarySearch(l.boundaries, start)
	if !found {
		lo--
	}
	hi, _ := slices.BinarySearch(l.boundaries, end)
	return l.boundaries[max(lo, 0)], l.boundaries[min(hi, len(l.boundaries)-1)]
}

func (s *ViewportState) captureSourceSelection() {
	if _, aware := s.items.(ViewportSourceItems); !aware {
		return
	}
	start, end := s.selAnchor.Item, s.selCursor.Item
	if start > end {
		start, end = end, start
	}
	s.sourceSnapshots = make(map[int]string)
	for i := start; i <= end; i++ {
		if l, aware := s.sourceLayout(i); aware {
			s.sourceSnapshots[i] = l.source
		}
	}
}
func (s *ViewportState) sourceBounds(item, startItem, endItem, length int) (int, int) {
	lo, hi := 0, length
	if snapshot, ok := s.sourceSnapshots[item]; ok {
		hi = len(snapshot)
	}
	a, b := s.sourceOrdered()
	if a != nil && b != nil {
		if a.item == item {
			lo = a.offset
		}
		if b.item == item {
			hi = b.offset
		}
	} else {
		for _, p := range []*sourceEndpoint{s.sourceAnchor, s.sourceCursor} {
			if p != nil && p.item == item {
				if item == startItem {
					lo = p.offset
				} else if item == endItem {
					hi = p.offset
				}
			}
		}
	}
	return lo, hi
}

func segmentWords(segments []StyledSegment) []sourceWordToken {
	var out []sourceWordToken
	for _, seg := range segments {
		for _, token := range seg.source {
			out = append(out, sourceWordToken{token.start, token.end, seg.Text[token.lo:token.hi]})
		}
	}
	return out
}
func (l *sourceLayout) addWords(words []sourceWordToken, base int) {
	for _, word := range words {
		lo, hi := word.start+base, word.end+base
		if lo < 0 || hi > len(l.source) || lo >= hi {
			l.valid = false
			continue
		}
		lo, hi = l.cluster(lo, hi)
		word.start, word.end = lo, hi
		l.words = append(l.words, word)
	}
}

// Direction and paragraph controls can make visible approval or transcript text
// misleading. Source copying retains them as data; rendering replaces them.
func sourceUnsafeRune(r rune) bool {
	return unicode.IsControl(r) || r >= 0x202a && r <= 0x202e ||
		r >= 0x2066 && r <= 0x2069 || r == 0x2028 || r == 0x2029 || r == 0x200e || r == 0x200f
}
