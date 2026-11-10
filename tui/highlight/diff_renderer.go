package highlight

import (
	"fmt"
	"strings"

	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/deepnoodle-ai/wonton/tui"
)

// DiffTheme defines colors and styles for diff rendering
type DiffTheme struct {
	AddedBg      tui.RGB   // Background for added lines
	AddedFg      tui.RGB   // Foreground for added lines
	RemovedBg    tui.RGB   // Background for removed lines
	RemovedFg    tui.RGB   // Foreground for removed lines
	ContextStyle tui.Style // Style for context lines
	HeaderStyle  tui.Style // Style for file headers
	HunkStyle    tui.Style // Style for hunk headers
	LineNumStyle tui.Style // Style for line numbers
	SyntaxTheme  string    // Chroma theme for syntax highlighting
}

// DefaultDiffTheme returns a default diff theme
func DefaultDiffTheme() DiffTheme {
	return DiffTheme{
		AddedBg:      tui.RGB{R: 0, G: 64, B: 0},      // Dark green background
		AddedFg:      tui.RGB{R: 100, G: 255, B: 100}, // Light green foreground
		RemovedBg:    tui.RGB{R: 64, G: 0, B: 0},      // Dark red background
		RemovedFg:    tui.RGB{R: 255, G: 100, B: 100}, // Light red foreground
		ContextStyle: tui.NewStyle().WithForeground(tui.ColorWhite),
		HeaderStyle:  tui.NewStyle().WithForeground(tui.ColorCyan).WithBold(),
		HunkStyle:    tui.NewStyle().WithForeground(tui.ColorBlue).WithBold(),
		LineNumStyle: tui.NewStyle().WithForeground(tui.ColorBrightBlack),
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
	LineNumOld string              // Old line number (formatted)
	LineNumNew string              // New line number (formatted)
	Segments   []tui.StyledSegment // Styled content segments
	BgColor    *tui.RGB            // Background color for the entire line
}

// RenderDiff renders a diff to styled output
func (dr *DiffRenderer) RenderDiff(diff *tui.Diff, language string) []RenderedDiffLine {
	if diff == nil {
		return nil
	}

	var result []RenderedDiffLine

	for _, file := range diff.Files {
		// Render file header
		headerText := fmt.Sprintf("--- %s", file.OldPath)
		result = append(result, RenderedDiffLine{
			Segments: []tui.StyledSegment{{
				Text:  headerText,
				Style: dr.Theme.HeaderStyle,
			}},
		})

		headerText = fmt.Sprintf("+++ %s", file.NewPath)
		result = append(result, RenderedDiffLine{
			Segments: []tui.StyledSegment{{
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
				Segments: []tui.StyledSegment{{
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

// expandTabs expands tab characters to spaces, using 4 columns per tab stop
// when TabWidth is not positive.
func (dr *DiffRenderer) expandTabs(s string) string {
	tabWidth := dr.TabWidth
	if tabWidth <= 0 {
		tabWidth = 4
	}
	return expandTabStops(s, 0, tabWidth)
}

func (dr *DiffRenderer) renderDiffLine(line tui.DiffLine, language string) RenderedDiffLine {
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
	var style tui.Style
	var bgColor *tui.RGB

	switch line.Type {
	case tui.DiffLineAdded:
		style = tui.NewStyle().WithFgRGB(dr.Theme.AddedFg)
		bgColor = &dr.Theme.AddedBg

	case tui.DiffLineRemoved:
		style = tui.NewStyle().WithFgRGB(dr.Theme.RemovedFg)
		bgColor = &dr.Theme.RemovedBg

	case tui.DiffLineContext:
		style = dr.Theme.ContextStyle

	default:
		style = tui.NewStyle()
	}

	rendered.BgColor = bgColor

	// Expand tabs to spaces for consistent rendering
	content := dr.expandTabs(line.Content)

	// Apply syntax highlighting if enabled and we have a language
	if dr.SyntaxHighlight && language != "" && content != "" &&
		(line.Type == tui.DiffLineAdded || line.Type == tui.DiffLineRemoved || line.Type == tui.DiffLineContext) {

		segments := dr.highlightLine(content, language)
		if segments != nil {
			// Merge diff coloring with syntax highlighting
			for i := range segments {
				// Keep syntax foreground colors but apply diff background
				if line.Type == tui.DiffLineAdded || line.Type == tui.DiffLineRemoved {
					// For diff lines, blend the colors
					segments[i].Style.BgRGB = bgColor
				}
			}
			rendered.Segments = segments
		} else {
			// Fallback to plain style
			rendered.Segments = []tui.StyledSegment{{
				Text:  content,
				Style: style,
			}}
		}
	} else {
		// No syntax highlighting
		rendered.Segments = []tui.StyledSegment{{
			Text:  content,
			Style: style,
		}}
	}

	return rendered
}

func (dr *DiffRenderer) highlightLine(content, language string) []tui.StyledSegment {
	lexer := lexers.Get(language)
	if lexer == nil {
		return nil
	}
	iterator, err := lexer.Tokenise(nil, content)
	if err != nil {
		return nil
	}
	chromaStyle := chromaTheme(dr.Theme.SyntaxTheme)

	var segments []tui.StyledSegment
	for _, token := range iterator.Tokens() {
		segments = append(segments, tui.StyledSegment{
			Text:  token.Value,
			Style: tokenStyle(chromaStyle.Get(token.Type)),
		})
	}
	return segments
}
