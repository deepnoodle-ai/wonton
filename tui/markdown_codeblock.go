package tui

import (
	"strings"
	"unicode"

	"github.com/deepnoodle-ai/wonton/runewidth"
)

// finishCodeBlock decorates the code block lines from start onward, when the
// theme asks for a background panel or a language label, and then adds the
// blank line that follows every code block.
func (mr *MarkdownRenderer) finishCodeBlock(ctx *renderContext, start int, language string) {
	// A zero Style is an unset field, not a request for a black panel.
	fill := mr.Theme.CodeBlockStyle != (Style{}) && hasBackground(mr.Theme.CodeBlockStyle)
	label := ""
	if mr.Theme.CodeBlockShowLanguage && !strings.ContainsFunc(language, unicode.IsControl) {
		label = language
	}
	if (fill || label != "") && start < len(ctx.result.Lines) {
		mr.decorateCodeBlock(ctx.result.Lines[start:], ctx.indent, label, fill)
	}
	ctx.result.Lines = append(ctx.result.Lines, StyledLine{})
}

func hasBackground(s Style) bool {
	return s.Background != ColorDefault || s.BgRGB != nil
}

// decorateCodeBlock pads code lines to one width. With fill, the lines start
// at indent and get one column of padding on each side, all in the code block
// background. The language label, if any, is right-aligned on the first line.
func (mr *MarkdownRenderer) decorateCodeBlock(lines []StyledLine, indent int, label string, fill bool) {
	background := Style{Foreground: ColorDefault, Background: mr.Theme.CodeBlockStyle.Background, BgRGB: mr.Theme.CodeBlockStyle.BgRGB}
	left, pad := indent+mr.TabWidth, 0
	if fill {
		left, pad = indent, 1
	}

	widths := make([]int, len(lines))
	inner := 0
	for i := range lines {
		lines[i].Segments = mr.codeLineSegments(lines[i].Segments)
		widths[i] = mr.segmentsWidth(lines[i].Segments)
		inner = max(inner, widths[i])
	}
	if mr.MaxWidth > 0 {
		inner = max(inner, mr.MaxWidth-left-2*pad)
	}

	labelWidth := runewidth.StringWidth(label)
	need := widths[0] + 2 + labelWidth
	if label != "" && mr.MaxWidth <= 0 {
		inner = max(inner, need)
	}
	showLabel := label != "" && need <= inner

	labelStyle := mr.Theme.CodeBlockLanguageStyle
	if labelStyle == (Style{}) {
		labelStyle = NewStyle().WithForeground(ColorBrightBlack)
	}
	if fill {
		labelStyle = mr.mergeStyles(background, labelStyle)
	}

	for i := range lines {
		if !fill && !(showLabel && i == 0) {
			continue
		}
		var segments []StyledSegment
		if pad > 0 {
			segments = append(segments, StyledSegment{Text: strings.Repeat(" ", pad), Style: background})
		}
		for _, seg := range lines[i].Segments {
			if fill {
				// Highlighting sets the foreground; the block style fills in
				// the background and any color a token leaves unset.
				seg.Style = mr.mergeStyles(mr.Theme.CodeBlockStyle, seg.Style)
			}
			segments = append(segments, seg)
		}
		space := inner - widths[i]
		if showLabel && i == 0 {
			space -= labelWidth
		}
		if space > 0 {
			segments = append(segments, StyledSegment{Text: strings.Repeat(" ", space), Style: background})
		}
		if showLabel && i == 0 {
			segments = append(segments, StyledSegment{Text: label, Style: labelStyle})
		}
		if pad > 0 {
			segments = append(segments, StyledSegment{Text: strings.Repeat(" ", pad), Style: background})
		}
		lines[i].Segments = segments
		lines[i].Indent = left
	}
}

// codeLineSegments drops a line's trailing newline and expands tabs, so the
// line's display width can be measured and padded.
func (mr *MarkdownRenderer) codeLineSegments(segments []StyledSegment) []StyledSegment {
	out := make([]StyledSegment, 0, len(segments))
	for _, seg := range segments {
		text := strings.TrimRight(seg.Text, "\r\n")
		if len(text) < len(seg.Text) {
			seg = segmentSlice(seg, 0, len(text))
		}
		if seg.source == nil && strings.Contains(seg.Text, "\t") {
			seg.Text = strings.ReplaceAll(seg.Text, "\t", strings.Repeat(" ", max(mr.TabWidth, 1)))
		}
		if seg.Text != "" {
			out = append(out, seg)
		}
	}
	return out
}
