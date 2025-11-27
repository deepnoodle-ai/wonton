package gooey

import (
	"fmt"
	"image"
)

// progressView displays a progress bar (declarative view)
type progressView struct {
	current       int
	total         int
	width         int
	filledChar    rune
	emptyChar     rune
	style         Style
	bgStyle       Style
	showPercent   bool
	showFraction  bool
	label         string
}

// Progress creates a declarative progress bar view.
// current is the current progress value, total is the maximum value.
//
// Example:
//
//	Progress(50, 100).Width(30).ShowPercent()
func Progress(current, total int) *progressView {
	return &progressView{
		current:     current,
		total:       total,
		width:       20,
		filledChar:  '█',
		emptyChar:   '░',
		style:       NewStyle().WithForeground(ColorGreen),
		bgStyle:     NewStyle().WithForeground(ColorBrightBlack),
		showPercent: true,
	}
}

// Width sets the width of the progress bar (not including label/percentage).
func (p *progressView) Width(w int) *progressView {
	p.width = w
	return p
}

// FilledChar sets the character used for the filled portion.
func (p *progressView) FilledChar(c rune) *progressView {
	p.filledChar = c
	return p
}

// EmptyChar sets the character used for the empty portion.
func (p *progressView) EmptyChar(c rune) *progressView {
	p.emptyChar = c
	return p
}

// Fg sets the foreground color for the filled portion.
func (p *progressView) Fg(c Color) *progressView {
	p.style = p.style.WithForeground(c)
	return p
}

// BgFg sets the foreground color for the empty portion.
func (p *progressView) BgFg(c Color) *progressView {
	p.bgStyle = p.bgStyle.WithForeground(c)
	return p
}

// Style sets the complete style for the filled portion.
func (p *progressView) Style(s Style) *progressView {
	p.style = s
	return p
}

// ShowPercent enables percentage display after the bar.
func (p *progressView) ShowPercent() *progressView {
	p.showPercent = true
	return p
}

// HidePercent disables percentage display.
func (p *progressView) HidePercent() *progressView {
	p.showPercent = false
	return p
}

// ShowFraction shows current/total instead of percentage.
func (p *progressView) ShowFraction() *progressView {
	p.showFraction = true
	p.showPercent = false
	return p
}

// Label sets a label to display before the bar.
func (p *progressView) Label(label string) *progressView {
	p.label = label
	return p
}

func (p *progressView) size(maxWidth, maxHeight int) (int, int) {
	w := p.width
	if p.label != "" {
		labelW, _ := MeasureText(p.label)
		w += labelW + 1 // +1 for space
	}
	if p.showPercent {
		w += 5 // " 100%"
	}
	if p.showFraction {
		// Estimate fraction width
		w += len(fmt.Sprintf(" %d/%d", p.total, p.total)) + 1
	}
	if maxWidth > 0 && w > maxWidth {
		w = maxWidth
	}
	return w, 1
}

func (p *progressView) render(frame RenderFrame, bounds image.Rectangle) {
	if bounds.Empty() {
		return
	}

	subFrame := frame.SubFrame(bounds)
	x := 0

	// Draw label
	if p.label != "" {
		subFrame.PrintStyled(x, 0, p.label, p.style)
		labelW, _ := MeasureText(p.label)
		x += labelW + 1
	}

	// Calculate available width for bar
	barWidth := p.width
	availableWidth := bounds.Dx() - x
	if p.showPercent {
		availableWidth -= 5
	}
	if p.showFraction {
		availableWidth -= len(fmt.Sprintf(" %d/%d", p.total, p.total)) + 1
	}
	if barWidth > availableWidth {
		barWidth = availableWidth
	}
	if barWidth < 1 {
		barWidth = 1
	}

	// Calculate filled width
	fillWidth := 0
	if p.total > 0 {
		fillWidth = (p.current * barWidth) / p.total
		if fillWidth > barWidth {
			fillWidth = barWidth
		}
		if fillWidth < 0 {
			fillWidth = 0
		}
	}

	// Draw empty background
	for i := 0; i < barWidth; i++ {
		subFrame.SetCell(x+i, 0, p.emptyChar, p.bgStyle)
	}

	// Draw filled portion
	for i := 0; i < fillWidth; i++ {
		subFrame.SetCell(x+i, 0, p.filledChar, p.style)
	}
	x += barWidth

	// Draw percentage or fraction
	if p.showPercent && p.total > 0 {
		percent := (p.current * 100) / p.total
		text := fmt.Sprintf(" %3d%%", percent)
		subFrame.PrintStyled(x, 0, text, p.style)
	} else if p.showFraction {
		text := fmt.Sprintf(" %d/%d", p.current, p.total)
		subFrame.PrintStyled(x, 0, text, p.style)
	}
}

// loadingView displays an animated spinner (declarative view)
type loadingView struct {
	frame   uint64
	charset []string
	speed   int // frames per character change
	style   Style
	label   string
}

// Loading creates an animated loading spinner view.
// The frame parameter should come from TickEvent.Frame for animation.
//
// Example:
//
//	Loading(app.frame).Label("Loading...")
//	Loading(app.frame).CharSet(SpinnerDots.Frames)
func Loading(frame uint64) *loadingView {
	return &loadingView{
		frame:   frame,
		charset: SpinnerDots.Frames,
		speed:   4,
		style:   NewStyle(),
	}
}

// CharSet sets the character set for the spinner animation.
func (s *loadingView) CharSet(chars []string) *loadingView {
	if len(chars) > 0 {
		s.charset = chars
	}
	return s
}

// Speed sets how many frames per character change (higher = slower).
func (s *loadingView) Speed(frames int) *loadingView {
	if frames > 0 {
		s.speed = frames
	}
	return s
}

// Fg sets the foreground color.
func (s *loadingView) Fg(c Color) *loadingView {
	s.style = s.style.WithForeground(c)
	return s
}

// Style sets the complete style.
func (s *loadingView) Style(st Style) *loadingView {
	s.style = st
	return s
}

// Label sets a label to display after the spinner.
func (s *loadingView) Label(label string) *loadingView {
	s.label = label
	return s
}

func (s *loadingView) size(maxWidth, maxHeight int) (int, int) {
	w := 1 // spinner character
	if s.label != "" {
		labelW, _ := MeasureText(s.label)
		w += 1 + labelW // space + label
	}
	if maxWidth > 0 && w > maxWidth {
		w = maxWidth
	}
	return w, 1
}

func (s *loadingView) render(frame RenderFrame, bounds image.Rectangle) {
	if bounds.Empty() || len(s.charset) == 0 {
		return
	}

	subFrame := frame.SubFrame(bounds)

	// Calculate which character to show
	idx := int(s.frame/uint64(s.speed)) % len(s.charset)
	char := s.charset[idx]

	// Draw spinner
	subFrame.PrintStyled(0, 0, char, s.style)

	// Draw label
	if s.label != "" {
		charW, _ := MeasureText(char)
		subFrame.PrintStyled(charW+1, 0, s.label, s.style)
	}
}
