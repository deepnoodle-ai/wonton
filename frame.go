package gooey

import (
	"strings"

	"github.com/mattn/go-runewidth"
)

// BorderStyle defines the characters used for drawing borders
type BorderStyle struct {
	TopLeft     string
	TopRight    string
	BottomLeft  string
	BottomRight string
	Horizontal  string
	Vertical    string
	Cross       string
	TopJoin     string
	BottomJoin  string
	LeftJoin    string
	RightJoin   string
}

// Predefined border styles
var (
	// SingleBorder uses single-line box drawing characters
	SingleBorder = BorderStyle{
		TopLeft:     "┌",
		TopRight:    "┐",
		BottomLeft:  "└",
		BottomRight: "┘",
		Horizontal:  "─",
		Vertical:    "│",
		Cross:       "┼",
		TopJoin:     "┬",
		BottomJoin:  "┴",
		LeftJoin:    "├",
		RightJoin:   "┤",
	}

	// DoubleBorder uses double-line box drawing characters
	DoubleBorder = BorderStyle{
		TopLeft:     "╔",
		TopRight:    "╗",
		BottomLeft:  "╚",
		BottomRight: "╝",
		Horizontal:  "═",
		Vertical:    "║",
		Cross:       "╬",
		TopJoin:     "╦",
		BottomJoin:  "╩",
		LeftJoin:    "╠",
		RightJoin:   "╣",
	}

	// RoundedBorder uses rounded corners
	RoundedBorder = BorderStyle{
		TopLeft:     "╭",
		TopRight:    "╮",
		BottomLeft:  "╰",
		BottomRight: "╯",
		Horizontal:  "─",
		Vertical:    "│",
		Cross:       "┼",
		TopJoin:     "┬",
		BottomJoin:  "┴",
		LeftJoin:    "├",
		RightJoin:   "┤",
	}

	// ThickBorder uses thick box drawing characters
	ThickBorder = BorderStyle{
		TopLeft:     "┏",
		TopRight:    "┓",
		BottomLeft:  "┗",
		BottomRight: "┛",
		Horizontal:  "━",
		Vertical:    "┃",
		Cross:       "╋",
		TopJoin:     "┳",
		BottomJoin:  "┻",
		LeftJoin:    "┣",
		RightJoin:   "┫",
	}

	// ASCIIBorder uses ASCII characters for compatibility
	ASCIIBorder = BorderStyle{
		TopLeft:     "+",
		TopRight:    "+",
		BottomLeft:  "+",
		BottomRight: "+",
		Horizontal:  "-",
		Vertical:    "|",
		Cross:       "+",
		TopJoin:     "+",
		BottomJoin:  "+",
		LeftJoin:    "+",
		RightJoin:   "+",
	}
)

// Frame represents a bordered frame
type Frame struct {
	X           int
	Y           int
	Width       int
	Height      int
	Border      BorderStyle
	BorderStyle Style
	Title       string
	TitleStyle  Style
	TitleAlign  Alignment
}

// Alignment represents text alignment
type Alignment int

const (
	AlignLeft Alignment = iota
	AlignCenter
	AlignRight
)

// NewFrame creates a new frame with default border
func NewFrame(x, y, width, height int) *Frame {
	return &Frame{
		X:           x,
		Y:           y,
		Width:       width,
		Height:      height,
		Border:      SingleBorder,
		BorderStyle: NewStyle(),
		TitleStyle:  NewStyle(),
		TitleAlign:  AlignLeft,
	}
}

// WithBorderStyle sets the border style
func (f *Frame) WithBorderStyle(style BorderStyle) *Frame {
	f.Border = style
	return f
}

// WithColor sets the border color
func (f *Frame) WithColor(style Style) *Frame {
	f.BorderStyle = style
	return f
}

// WithTitle sets the frame title
func (f *Frame) WithTitle(title string, align Alignment) *Frame {
	f.Title = title
	f.TitleAlign = align
	return f
}

// WithTitleStyle sets the title style
func (f *Frame) WithTitleStyle(style Style) *Frame {
	f.TitleStyle = style
	return f
}

// Draw renders the frame to the terminal
// Updated to use RenderFrame
func (f *Frame) Draw(t RenderFrame) {
	if f.Width < 2 || f.Height < 2 {
		return
	}

	// Draw top border
	f.drawTopBorder(t)

	// Draw sides
	for i := 1; i < f.Height-1; i++ {
		t.PrintStyled(f.X, f.Y+i, f.Border.Vertical, f.BorderStyle)
		t.PrintStyled(f.X+f.Width-1, f.Y+i, f.Border.Vertical, f.BorderStyle)
	}

	// Draw bottom border
	f.drawBottomBorder(t)
}

func (f *Frame) drawTopBorder(t RenderFrame) {
	line := f.Border.TopLeft + strings.Repeat(f.Border.Horizontal, f.Width-2) + f.Border.TopRight

	// Insert title if present
	if f.Title != "" {
		titleWithSpaces := " " + f.Title + " "
		titleLen := runewidth.StringWidth(titleWithSpaces)

		if titleLen < f.Width-2 {
			var pos int
			switch f.TitleAlign {
			case AlignLeft:
				pos = 1 // Start right after the top-left corner
			case AlignCenter:
				pos = ((f.Width - titleLen) / 2)
				if pos < 1 {
					pos = 1
				}
			case AlignRight:
				pos = f.Width - titleLen - 1 // End before the top-right corner
				if pos < 1 {
					pos = 1
				}
			}

			lineRunes := []rune(line)
			titleRunes := []rune(titleWithSpaces)
			for i, r := range titleRunes {
				if pos+i < len(lineRunes)-1 { // Don't overwrite the top-right corner
					lineRunes[pos+i] = r
				}
			}
			line = string(lineRunes)
		}
	}

	// Apply styles
	if f.Title != "" {
		// Split the line to apply different styles
		parts := f.splitLineForTitle(line)

		// Part 0: before title (border style)
		t.PrintStyled(f.X, f.Y, parts[0], f.BorderStyle)

		// Part 1: title (title style)
		titleX := f.X + runewidth.StringWidth(parts[0])
		t.PrintStyled(titleX, f.Y, parts[1], f.TitleStyle)

		// Part 2: after title (border style)
		afterX := titleX + runewidth.StringWidth(parts[1])
		t.PrintStyled(afterX, f.Y, parts[2], f.BorderStyle)
	} else {
		t.PrintStyled(f.X, f.Y, line, f.BorderStyle)
	}
}

func (f *Frame) splitLineForTitle(line string) []string {
	if f.Title == "" {
		return []string{line, "", ""}
	}

	titleWithSpaces := " " + f.Title + " "
	lineRunes := []rune(line)
	titleRunes := []rune(titleWithSpaces)

	titleStart := -1
	for i := 0; i <= len(lineRunes)-len(titleRunes); i++ {
		match := true
		for j := 0; j < len(titleRunes); j++ {
			if lineRunes[i+j] != titleRunes[j] {
				match = false
				break
			}
		}
		if match {
			titleStart = i
			break
		}
	}

	if titleStart == -1 {
		return []string{line, "", ""}
	}

	before := string(lineRunes[:titleStart])
	title := titleWithSpaces
	after := string(lineRunes[titleStart+len(titleRunes):])

	return []string{before, title, after}
}

func (f *Frame) drawBottomBorder(t RenderFrame) {
	line := f.Border.BottomLeft + strings.Repeat(f.Border.Horizontal, f.Width-2) + f.Border.BottomRight
	t.PrintStyled(f.X, f.Y+f.Height-1, line, f.BorderStyle)
}

// Clear clears the content inside the frame (not the border)
func (f *Frame) Clear(t RenderFrame) {
	if f.Width < 2 {
		return
	}
	// Use FillStyled with spaces
	for i := 1; i < f.Height-1; i++ {
		t.FillStyled(f.X+1, f.Y+i, f.Width-2, 1, ' ', NewStyle())
	}
}

// Box is a simpler API for drawing boxes
type Box struct {
	Content     []string
	Border      BorderStyle
	BorderStyle Style
	Padding     int
}

// NewBox creates a new box with content
func NewBox(content []string) *Box {
	return &Box{
		Content:     content,
		Border:      SingleBorder,
		BorderStyle: NewStyle(),
		Padding:     1,
	}
}

// Draw renders the box at the specified position
func (b *Box) Draw(t RenderFrame, x, y int) {
	if len(b.Content) == 0 {
		return
	}

	// Calculate dimensions
	maxWidth := 0
	for _, line := range b.Content {
		if len := runewidth.StringWidth(line); len > maxWidth {
			maxWidth = len
		}
	}

	width := maxWidth + 2 + (b.Padding * 2)
	height := len(b.Content) + 2 + (b.Padding * 2)

	// Draw the complete box line by line

	// Top border
	topLine := b.Border.TopLeft + strings.Repeat(b.Border.Horizontal, width-2) + b.Border.TopRight
	t.PrintStyled(x, y, topLine, b.BorderStyle)

	// Content lines with side borders
	for i := 1; i < height-1; i++ {
		currentY := y + i

		// Left border
		t.PrintStyled(x, currentY, b.Border.Vertical, b.BorderStyle)

		// Inner content x position
		contentX := x + 1

		// Content or padding
		contentIndex := i - 1 - b.Padding

		// Padding before content
		if b.Padding > 0 {
			t.FillStyled(contentX, currentY, b.Padding, 1, ' ', NewStyle())
			contentX += b.Padding
		}

		if contentIndex >= 0 && contentIndex < len(b.Content) {
			// Content
			contentLine := b.Content[contentIndex]
			maxLineWidth := width - 2 - (b.Padding * 2)
			// Truncate to fit width properly considering character widths
			if runewidth.StringWidth(contentLine) > maxLineWidth {
				contentLine = runewidth.Truncate(contentLine, maxLineWidth, "")
			}
			t.PrintStyled(contentX, currentY, contentLine, NewStyle())

			// Padding after content
			contentWidth := runewidth.StringWidth(contentLine)
			padding := width - 2 - b.Padding - contentWidth
			if padding > 0 {
				t.FillStyled(contentX+contentWidth, currentY, padding, 1, ' ', NewStyle())
			}
		} else {
			// Just empty space for vertical padding
			t.FillStyled(contentX, currentY, width-2-(b.Padding*2), 1, ' ', NewStyle())
		}

		// Right border
		t.PrintStyled(x+width-1, currentY, b.Border.Vertical, b.BorderStyle)
	}

	// Bottom border
	bottomLine := b.Border.BottomLeft + strings.Repeat(b.Border.Horizontal, width-2) + b.Border.BottomRight
	t.PrintStyled(x, y+height-1, bottomLine, b.BorderStyle)
}
