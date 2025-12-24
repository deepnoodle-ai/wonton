package tui

import (
	"time"

	"github.com/mattn/go-runewidth"
)

// Spinner represents an animated loading spinner widget
//
// Deprecated: Use Loading() for declarative spinner.
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
//
// Deprecated: Use Loading() for declarative spinner.
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

// Fg sets the foreground color for the spinner.
func (s *Spinner) Fg(c Color) *Spinner {
	s.style = s.style.WithForeground(c)
	return s
}

// Bg sets the background color for the spinner.
func (s *Spinner) Bg(c Color) *Spinner {
	s.style = s.style.WithBackground(c)
	return s
}

// Bold makes the spinner bold.
func (s *Spinner) Bold() *Spinner {
	s.style = s.style.WithBold()
	return s
}

// Dim makes the spinner dimmed.
func (s *Spinner) Dim() *Spinner {
	s.style = s.style.WithDim()
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
