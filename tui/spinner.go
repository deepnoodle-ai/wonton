package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/mattn/go-runewidth"
)

// Spinner represents an animated loading spinner widget
type Spinner struct {
	frames       []string
	interval     time.Duration // Time per frame
	style        Style
	message      string
	currentFrame int
	lastUpdate   time.Time
}

// SpinnerStyle defines different spinner animations
type SpinnerStyle struct {
	Frames   []string
	Interval time.Duration
}

// Predefined spinner styles
var (
	SpinnerDots = SpinnerStyle{
		Frames:   []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		Interval: 80 * time.Millisecond,
	}

	SpinnerLine = SpinnerStyle{
		Frames:   []string{"-", "\\", "|", "/"},
		Interval: 100 * time.Millisecond,
	}

	SpinnerArrows = SpinnerStyle{
		Frames:   []string{"←", "↖", "↑", "↗", "→", "↘", "↓", "↙"},
		Interval: 100 * time.Millisecond,
	}

	SpinnerCircle = SpinnerStyle{
		Frames:   []string{"◐", "◓", "◑", "◒"},
		Interval: 120 * time.Millisecond,
	}

	SpinnerSquare = SpinnerStyle{
		Frames:   []string{"◰", "◳", "◲", "◱"},
		Interval: 100 * time.Millisecond,
	}

	SpinnerBounce = SpinnerStyle{
		Frames:   []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"},
		Interval: 80 * time.Millisecond,
	}

	SpinnerBar = SpinnerStyle{
		Frames: []string{
			"[    ]",
			"[=   ]",
			"[==  ]",
			"[=== ]",
			"[====]",
			"[ ===]",
			"[  ==]",
			"[   =]",
		},
		Interval: 80 * time.Millisecond,
	}

	SpinnerStars = SpinnerStyle{
		Frames: []string{
			"✶",
			"✸",
			"✹",
			"✺",
			"✹",
			"✷",
		},
		Interval: 120 * time.Millisecond,
	}

	SpinnerStarField = SpinnerStyle{
		Frames: []string{
			"·   ",
			"*·  ",
			"✦*· ",
			"✧✦*·",
			"·✧✦*",
			" ·✧✦",
			"  ·✧",
			"   ·",
		},
		Interval: 100 * time.Millisecond,
	}

	SpinnerAsterisk = SpinnerStyle{
		Frames: []string{
			"*",
			"⁎",
			"✱",
			"✲",
			"✳",
			"✴",
			"✵",
			"✶",
			"✷",
			"✸",
			"✹",
			"✺",
		},
		Interval: 100 * time.Millisecond,
	}

	SpinnerSparkle = SpinnerStyle{
		Frames: []string{
			"✨",
			"💫",
			"⭐",
			"🌟",
			"✨",
			"💫",
		},
		Interval: 150 * time.Millisecond,
	}
)

// NewSpinner creates a new spinner widget
func NewSpinner(style SpinnerStyle) *Spinner {
	return &Spinner{
		frames:     style.Frames,
		interval:   style.Interval,
		style:      NewStyle(),
		lastUpdate: time.Now(),
	}
}

// WithStyle sets the spinner style
func (s *Spinner) WithStyle(style Style) *Spinner {
	s.style = style
	return s
}

// WithMessage sets the spinner message
func (s *Spinner) WithMessage(message string) *Spinner {
	s.message = message
	return s
}

// Update advances the spinner animation based on elapsed time.
// Call this from HandleEvent on TickEvent.
func (s *Spinner) Update(now time.Time) {
	if now.Sub(s.lastUpdate) >= s.interval {
		s.currentFrame = (s.currentFrame + 1) % len(s.frames)
		s.lastUpdate = now
	}
}

// Draw renders the spinner at the specified position
func (s *Spinner) Draw(frame RenderFrame, x, y int) {
	// Draw spinner frame
	frame.PrintStyled(x, y, s.frames[s.currentFrame], s.style)

	// Draw message if present
	if s.message != "" {
		// Calculate offset based on spinner width
		offset := runewidth.StringWidth(s.frames[s.currentFrame]) + 1
		frame.PrintStyled(x+offset, y, s.message, NewStyle())
	}
}

// ProgressBar represents a progress bar widget
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
