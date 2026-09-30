package tui

import (
	"strings"
	"unicode"

	"github.com/deepnoodle-ai/wonton/runewidth"
)

// plainLogicalRows is the common word-layout path used by WrapText and Text.
// Provenance travels with each original field before whitespace is collapsed.
func plainLogicalRows(text string, offset, width int, wrap, safe bool) [][]StyledSegment {
	var rawRows [][]StyledSegment
	if safe {
		rawRows = sourceTextLines(text, offset, 0, false)
	} else {
		for _, line := range strings.Split(text, "\n") {
			rawRows = append(rawRows, []StyledSegment{{Text: line}})
		}
	}
	var out [][]StyledSegment
	for _, raw := range rawRows {
		seg := StyledSegment{}
		for _, part := range raw {
			seg = appendSegment(seg, part)
		}
		if !wrap || width <= 0 || runewidth.StringWidth(seg.Text) <= width {
			out = append(out, []StyledSegment{seg})
			continue
		}
		var row []StyledSegment
		col, start := 0, -1
		emit := func(end int) {
			if start < 0 {
				return
			}
			word := segmentSlice(seg, start, end)
			w := runewidth.StringWidth(word.Text)
			space := 0
			if col > 0 {
				space = 1
			}
			if col+space+w > width && col > 0 {
				out = append(out, row)
				row = nil
				col = 0
			}
			if col > 0 {
				row = append(row, StyledSegment{Text: " "})
				col++
			}
			row = append(row, word)
			col += w
			start = -1
		}
		for at, r := range seg.Text {
			if unicode.IsSpace(r) {
				emit(at)
			} else if start < 0 {
				start = at
			}
		}
		emit(len(seg.Text))
		out = append(out, row)
	}
	return out
}
func plainPhysicalRows(rows [][]StyledSegment, width int, wrap bool, align Alignment) [][]StyledSegment {
	var out [][]StyledSegment
	for _, row := range rows {
		n := 0
		for _, seg := range row {
			n += runewidth.StringWidth(seg.Text)
		}
		if align != AlignLeft && width > n {
			left := width - n
			if align == AlignCenter {
				left /= 2
			}
			right := width - n - left
			row = append([]StyledSegment{{Text: strings.Repeat(" ", left)}}, row...)
			row = append(row, StyledSegment{Text: strings.Repeat(" ", right)})
		}
		if wrap && width > 0 && n > width {
			wrapped := wrapLiteralSegments(row, width)
			if len(wrapped) == 0 {
				wrapped = [][]StyledSegment{nil}
			}
			out = append(out, wrapped...)
		} else {
			out = append(out, row)
		}
	}
	return out
}

type textRowsCache struct {
	width, offset int
	wrap, marked  bool
	align         Alignment
	valid         bool
	rows          [][]StyledSegment
}

func (t *TextView) textRows(width int) [][]StyledSegment {
	for _, c := range t.rowsCache {
		if c.valid && c.width == width && c.offset == t.sourceOffset &&
			c.wrap == t.wrap && c.marked == t.sourceMarked && c.align == t.align {
			return c.rows
		}
	}
	rows := plainLogicalRows(t.content, t.sourceOffset, width, t.wrap, t.sourceMarked)
	rows = plainPhysicalRows(rows, width, t.wrap, t.align)
	t.rowsCache[1] = t.rowsCache[0]
	t.rowsCache[0] = textRowsCache{width, t.sourceOffset, t.wrap, t.sourceMarked, t.align, true, rows}
	return rows
}
