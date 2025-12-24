package tui

import (
	"fmt"
	"strings"

	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

// DiffTheme defines colors and styles for diff rendering
type DiffTheme struct {
	AddedBg      RGB    // Background for added lines
	AddedFg      RGB    // Foreground for added lines
	RemovedBg    RGB    // Background for removed lines
	RemovedFg    RGB    // Foreground for removed lines
	ContextStyle Style  // Style for context lines
	HeaderStyle  Style  // Style for file headers
	HunkStyle    Style  // Style for hunk headers
	LineNumStyle Style  // Style for line numbers
	SyntaxTheme  string // Chroma theme for syntax highlighting
}

// DefaultDiffTheme returns a default diff theme
func DefaultDiffTheme() DiffTheme {
	return DiffTheme{
		AddedBg:      RGB{R: 0, G: 64, B: 0},      // Dark green background
		AddedFg:      RGB{R: 100, G: 255, B: 100}, // Light green foreground
		RemovedBg:    RGB{R: 64, G: 0, B: 0},      // Dark red background
		RemovedFg:    RGB{R: 255, G: 100, B: 100}, // Light red foreground
		ContextStyle: NewStyle().WithForeground(ColorWhite),
		HeaderStyle:  NewStyle().WithForeground(ColorCyan).WithBold(),
		HunkStyle:    NewStyle().WithForeground(ColorBlue).WithBold(),
		LineNumStyle: NewStyle().WithForeground(ColorBrightBlack),
		SyntaxTheme:  "monokai",
	}
}

// DiffRenderer renders diff content with syntax highlighting
type DiffRenderer struct {
	Theme           DiffTheme
	ShowLineNums    bool
	SyntaxHighlight bool
	LineNumWidth    int // Width reserved for line numbers
	TabWidth        int // Number of spaces to expand tabs to (default 4)
}

// NewDiffRenderer creates a new diff renderer with default settings
func NewDiffRenderer() *DiffRenderer {
	return &DiffRenderer{
		Theme:           DefaultDiffTheme(),
		ShowLineNums:    true,
		SyntaxHighlight: true,
		LineNumWidth:    5,
		TabWidth:        4,
	}
}

// WithTheme sets a custom theme
func (dr *DiffRenderer) WithTheme(theme DiffTheme) *DiffRenderer {
	dr.Theme = theme
	return dr
}

// RenderedDiffLine represents a single rendered diff line with styling
type RenderedDiffLine struct {
	LineNumOld string          // Old line number (formatted)
	LineNumNew string          // New line number (formatted)
	Segments   []StyledSegment // Styled content segments
	BgColor    *RGB            // Background color for the entire line
}

// RenderDiff renders a diff to styled output
func (dr *DiffRenderer) RenderDiff(diff *Diff, language string) []RenderedDiffLine {
	var result []RenderedDiffLine

	for _, file := range diff.Files {
		// Render file header
		headerText := fmt.Sprintf("--- %s", file.OldPath)
		result = append(result, RenderedDiffLine{
			Segments: []StyledSegment{{
				Text:  headerText,
				Style: dr.Theme.HeaderStyle,
			}},
		})

		headerText = fmt.Sprintf("+++ %s", file.NewPath)
		result = append(result, RenderedDiffLine{
			Segments: []StyledSegment{{
				Text:  headerText,
				Style: dr.Theme.HeaderStyle,
			}},
		})

		// Blank line after header
		result = append(result, RenderedDiffLine{})

		// Render hunks
		for _, hunk := range file.Hunks {
			// Render hunk header
			result = append(result, RenderedDiffLine{
				Segments: []StyledSegment{{
					Text:  hunk.Header,
					Style: dr.Theme.HunkStyle,
				}},
			})

			// Render hunk lines
			for _, line := range hunk.Lines {
				renderedLine := dr.renderDiffLine(line, language)
				result = append(result, renderedLine)
			}

			// Blank line after hunk
			result = append(result, RenderedDiffLine{})
		}
	}

	return result
}

// expandTabs expands tab characters to spaces
func (dr *DiffRenderer) expandTabs(s string) string {
	if !strings.Contains(s, "\t") {
		return s
	}

	var result strings.Builder
	result.Grow(len(s) * 2) // Estimate expanded size

	col := 0
	for _, ch := range s {
		if ch == '\t' {
			// Expand tab to reach next tab stop
			spaces := dr.TabWidth - (col % dr.TabWidth)
			result.WriteString(strings.Repeat(" ", spaces))
			col += spaces
		} else {
			result.WriteRune(ch)
			col++
		}
	}

	return result.String()
}

func (dr *DiffRenderer) renderDiffLine(line DiffLine, language string) RenderedDiffLine {
	rendered := RenderedDiffLine{}

	// Format line numbers
	if dr.ShowLineNums {
		if line.OldLineNum > 0 {
			rendered.LineNumOld = fmt.Sprintf("%*d", dr.LineNumWidth, line.OldLineNum)
		} else {
			rendered.LineNumOld = strings.Repeat(" ", dr.LineNumWidth)
		}

		if line.NewLineNum > 0 {
			rendered.LineNumNew = fmt.Sprintf("%*d", dr.LineNumWidth, line.NewLineNum)
		} else {
			rendered.LineNumNew = strings.Repeat(" ", dr.LineNumWidth)
		}
	}

	// Determine style and background based on line type
	var style Style
	var bgColor *RGB

	switch line.Type {
	case DiffLineAdded:
		style = NewStyle().WithFgRGB(dr.Theme.AddedFg)
		bgColor = &dr.Theme.AddedBg

	case DiffLineRemoved:
		style = NewStyle().WithFgRGB(dr.Theme.RemovedFg)
		bgColor = &dr.Theme.RemovedBg

	case DiffLineContext:
		style = dr.Theme.ContextStyle

	default:
		style = NewStyle()
	}

	rendered.BgColor = bgColor

	// Expand tabs to spaces for consistent rendering
	content := dr.expandTabs(line.Content)

	// Apply syntax highlighting if enabled and we have a language
	if dr.SyntaxHighlight && language != "" && content != "" &&
		(line.Type == DiffLineAdded || line.Type == DiffLineRemoved || line.Type == DiffLineContext) {

		segments := dr.highlightLine(content, language)
		if segments != nil {
			// Merge diff coloring with syntax highlighting
			for i := range segments {
				// Keep syntax foreground colors but apply diff background
				if line.Type == DiffLineAdded || line.Type == DiffLineRemoved {
					// For diff lines, blend the colors
					segments[i].Style.BgRGB = bgColor
				}
			}
			rendered.Segments = segments
		} else {
			// Fallback to plain style
			rendered.Segments = []StyledSegment{{
				Text:  content,
				Style: style,
			}}
		}
	} else {
		// No syntax highlighting
		rendered.Segments = []StyledSegment{{
			Text:  content,
			Style: style,
		}}
	}

	return rendered
}

func (dr *DiffRenderer) highlightLine(content, language string) []StyledSegment {
	lexer := lexers.Get(language)
	if lexer == nil {
		return nil
	}

	chromaStyle := styles.Get(dr.Theme.SyntaxTheme)
	if chromaStyle == nil {
		chromaStyle = styles.Fallback
	}

	iterator, err := lexer.Tokenise(nil, content)
	if err != nil {
		return nil
	}

	var segments []StyledSegment

	for _, token := range iterator.Tokens() {
		styleEntry := chromaStyle.Get(token.Type)
		style := NewStyle()

		if styleEntry.Colour.IsSet() {
			r, g, b := styleEntry.Colour.Red(), styleEntry.Colour.Green(), styleEntry.Colour.Blue()
			style = style.WithFgRGB(RGB{R: r, G: g, B: b})
		}

		if styleEntry.Bold == 1 {
			style = style.WithBold()
		}
		if styleEntry.Italic == 1 {
			style = style.WithItalic()
		}
		if styleEntry.Underline == 1 {
			style = style.WithUnderline()
		}

		segments = append(segments, StyledSegment{
			Text:  token.Value,
			Style: style,
		})
	}

	return segments
}
