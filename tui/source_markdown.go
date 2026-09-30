package tui

import (
	"html"
	"strings"

	"github.com/deepnoodle-ai/wonton/runewidth"
	"github.com/yuin/goldmark/text"
)

// SourceOffset declares the position of this Markdown source in the canonical
// ViewportSourceItems.Source. Surrounding unmarked views are decorations.
func (m *MarkdownView) SourceOffset(offset int) *MarkdownView {
	m.sourceMarked = true
	m.sourceOffset = offset
	m.renderer.sourceMarked = true
	m.rendered = nil
	return m
}

func sourceSegment(source []byte, segment text.Segment, style Style, decode, marked bool) StyledSegment {
	// Goldmark padding and EOF newlines are presentation, not original bytes.
	raw := string(source[segment.Start:segment.Stop])
	seg := StyledSegment{Style: style}
	var shownText strings.Builder
	if segment.Padding > 0 {
		shownText.WriteString(strings.Repeat(" ", segment.Padding))
	}
	for at := 0; at < len(raw); {
		size := 0
		shown := ""
		if decode && raw[at] == '\\' && at+1 < len(raw) && strings.ContainsRune("!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~", rune(raw[at+1])) {
			shown = raw[at+1 : at+2]
			size = 2
		} else if decode && raw[at] == '&' {
			if end := strings.IndexByte(raw[at:min(len(raw), at+33)], ';'); end >= 0 && end <= 32 {
				token := raw[at : at+end+1]
				decoded := html.UnescapeString(token)
				if decoded != token {
					shown = decoded
					size = len(token)
				}
			}
		}
		if size == 0 {
			for g := range runewidth.Graphemes(raw[at:]) {
				shown = g
				size = len(g)
				break
			}
		}
		if shown != "\n" && shown != "\r\n" && shown != "\t" {
			for _, r := range shown {
				if sourceUnsafeRune(r) {
					shown = "�"
					break
				}
			}
		}
		start := shownText.Len()
		shownText.WriteString(shown)
		if marked {
			seg.source = append(seg.source, sourceToken{start, shownText.Len(), segment.Start + at, segment.Start + at + size})
		}
		at += size
	}
	seg.Text = shownText.String()
	if segment.ForceNewline && !strings.HasSuffix(seg.Text, "\n") {
		seg.Text += "\n"
	}
	return seg
}

// Literal layout preserves code whitespace while wrapping styled graphemes.
func wrapLiteralSegments(segments []StyledSegment, width int) [][]StyledSegment {
	var rows [][]StyledSegment
	var row []StyledSegment
	col := 0
	for _, seg := range segments {
		at := 0
		for g, w := range runewidth.Graphemes(seg.Text) {
			part := segmentSlice(seg, at, at+len(g))
			at += len(g)
			if g == "\n" || g == "\r\n" {
				rows = append(rows, row)
				row = nil
				col = 0
				continue
			}
			if g == "\t" {
				part.Text = "    "
				w = 4
				for i := range part.source {
					part.source[i].lo = 0
					part.source[i].hi = 4
				}
			}
			if width > 0 && col+w > width && col > 0 {
				rows = append(rows, row)
				row = nil
				col = 0
			}
			row = append(row, part)
			col += w
		}
	}
	if len(row) > 0 {
		rows = append(rows, row)
	}
	return rows
}

func (mr *MarkdownRenderer) renderSourceCode(lines *text.Segments, language string, ctx *renderContext) {
	code := StyledSegment{Style: mr.Theme.CodeBlockStyle}
	for i := 0; i < lines.Len(); i++ {
		line := lines.At(i)
		code = appendSegment(code, sourceSegment(ctx.source, line, mr.Theme.CodeBlockStyle, false, true))
	}
	ctx.result.words = append(ctx.result.words, segmentWords([]StyledSegment{code})...)
	var styled []StyledSegment
	if language != "" {
		highlighted := mr.highlightCode(code.Text, language)
		at := 0
	highlight:
		for _, row := range highlighted {
			for _, seg := range row {
				end := at + len(seg.Text)
				if end > len(code.Text) || code.Text[at:end] != seg.Text {
					styled = nil
					break highlight
				}
				part := segmentSlice(code, at, end)
				part.Style = seg.Style
				styled = append(styled, part)
				at = end
			}
			if strings.HasPrefix(code.Text[at:], "\r\n") {
				newline := segmentSlice(code, at, at+2)
				newline.Text = "\n"
				for i := range newline.source {
					newline.source[i].lo, newline.source[i].hi = 0, 1
				}
				styled = append(styled, newline)
				at += 2
			} else if at < len(code.Text) && code.Text[at] == '\n' {
				styled = append(styled, segmentSlice(code, at, at+1))
				at++
			}
		}
		if at != len(code.Text) {
			styled = nil
		}
	}
	if styled == nil {
		styled = []StyledSegment{code}
	}
	for _, row := range splitSegmentsAtNewlines(styled) {
		ctx.result.Lines = append(ctx.result.Lines, StyledLine{Segments: row, Indent: ctx.indent + mr.TabWidth})
	}
	ctx.result.Lines = append(ctx.result.Lines, StyledLine{})
}
