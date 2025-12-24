package tui

import (
	"fmt"
	"strings"

	"github.com/mattn/go-runewidth"
)

// ProgressBar represents a progress bar widget
//
// Deprecated: Use Progress() for declarative progress bar.
type ProgressBar struct {
	Total       int
	Current     int
	Width       int
	Style       Style
	FillChar    string
	EmptyChar   string
	ShowPercent bool
	ShowNumbers bool
	Message     string
}

// NewProgressBar creates a new progress bar widget
//
// Deprecated: Use Progress() for declarative progress bar.
func NewProgressBar(total int) *ProgressBar {
	return &ProgressBar{
		Total:       total,
		Width:       40,
		Style:       NewStyle().WithForeground(ColorCyan),
		FillChar:    "█",
		EmptyChar:   "░",
		ShowPercent: true,
		ShowNumbers: false,
	}
}

// WithWidth sets the progress bar width
func (p *ProgressBar) WithWidth(width int) *ProgressBar {
	p.Width = width
	return p
}

// WithStyle sets the progress bar style
func (p *ProgressBar) WithStyle(style Style) *ProgressBar {
	p.Style = style
	return p
}

// WithChars sets the fill and empty characters
func (p *ProgressBar) WithChars(fill, empty string) *ProgressBar {
	p.FillChar = fill
	p.EmptyChar = empty
	return p
}

// SetProgress sets the current progress
func (p *ProgressBar) SetProgress(current int) {
	p.Current = current
	if p.Current > p.Total {
		p.Current = p.Total
	}
}

// Draw renders the progress bar at the specified position
func (p *ProgressBar) Draw(frame RenderFrame, x, y int) {
	// Calculate fill
	percent := 0.0
	if p.Total > 0 {
		percent = float64(p.Current) / float64(p.Total)
	}
	filled := int(percent * float64(p.Width))

	// Build bar
	currentX := x

	frame.PrintStyled(currentX, y, "[", NewStyle())
	currentX++

	// Fill part
	if filled > 0 {
		fillStr := strings.Repeat(p.FillChar, filled)
		frame.PrintStyled(currentX, y, fillStr, p.Style)
		currentX += runewidth.StringWidth(fillStr)
	}

	// Empty part
	emptyWidth := p.Width - filled
	if emptyWidth > 0 {
		emptyStr := strings.Repeat(p.EmptyChar, emptyWidth)
		frame.PrintStyled(currentX, y, emptyStr, NewStyle())
		currentX += runewidth.StringWidth(emptyStr)
	}

	frame.PrintStyled(currentX, y, "]", NewStyle())
	currentX++

	// Add percentage
	if p.ShowPercent {
		pctStr := fmt.Sprintf(" %3.0f%%", percent*100)
		frame.PrintStyled(currentX, y, pctStr, NewStyle())
		currentX += len(pctStr)
	}

	// Add numbers
	if p.ShowNumbers {
		numStr := fmt.Sprintf(" (%d/%d)", p.Current, p.Total)
		frame.PrintStyled(currentX, y, numStr, NewStyle())
		currentX += len(numStr)
	}

	// Add message
	if p.Message != "" {
		frame.PrintStyled(currentX, y, " "+p.Message, NewStyle())
	}
}
