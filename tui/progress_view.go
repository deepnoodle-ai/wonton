package tui

import (
	"fmt"
	"math"
)

// progressView displays a progress bar (declarative view)
type progressView struct {
	current      int
	total        int
	width        int
	filledChar   rune
	emptyChar    rune
	style        Style
	emptyStyle   Style
	showPercent  bool
	showFraction bool
	label        string
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
		emptyStyle:  NewStyle().WithForeground(ColorBrightBlack),
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

// EmptyFg sets the foreground color for the empty portion of the bar.
func (p *progressView) EmptyFg(c Color) *progressView {
	p.emptyStyle = p.emptyStyle.WithForeground(c)
	return p
}

// EmptyStyle sets the complete style for the empty portion of the bar.
func (p *progressView) EmptyStyle(s Style) *progressView {
	p.emptyStyle = s
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

// Shimmer returns an animated progress bar with a shimmer highlight effect.
// Speed controls how fast the shimmer moves (lower = faster).
// HighlightColor is the color of the shimmer highlight.
func (p *progressView) Shimmer(highlightColor RGB, speed int) *animatedProgressView {
	return &animatedProgressView{
		base:           p,
		shimmer:        true,
		shimmerSpeed:   speed,
		highlightColor: highlightColor,
	}
}

// Pulse returns an animated progress bar with a pulsing brightness effect.
// Color is the base color, speed controls the pulse rate (lower = faster).
func (p *progressView) Pulse(color RGB, speed int) *animatedProgressView {
	return &animatedProgressView{
		base:       p,
		pulse:      true,
		pulseSpeed: speed,
		pulseColor: color,
	}
}

// animatedProgressView displays an animated progress bar
type animatedProgressView struct {
	base           *progressView
	shimmer        bool
	shimmerSpeed   int
	highlightColor RGB
	pulse          bool
	pulseSpeed     int
	pulseColor     RGB
}

func (a *animatedProgressView) size(maxWidth, maxHeight int) (int, int) {
	return a.base.size(maxWidth, maxHeight)
}

func (a *animatedProgressView) render(ctx *RenderContext) {
	width, height := ctx.Size()
	if width == 0 || height == 0 {
		return
	}

	p := a.base
	x := 0

	// Draw label
	if p.label != "" {
		ctx.PrintStyled(x, 0, p.label, p.style)
		labelW, _ := MeasureText(p.label)
		x += labelW + 1
	}

	// Calculate available width for bar
	barWidth := p.width
	availableWidth := width - x
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
		ctx.SetCell(x+i, 0, p.emptyChar, p.emptyStyle)
	}

	frame := ctx.Frame()

	// Draw filled portion with animation
	for i := 0; i < fillWidth; i++ {
		style := p.style

		if a.shimmer && fillWidth > 0 {
			// Calculate shimmer position (moves across the bar)
			shimmerPos := int(frame/uint64(a.shimmerSpeed)) % (fillWidth + 3)
			distance := shimmerPos - i
			if distance < 0 {
				distance = -distance
			}

			// Apply shimmer highlight when close to shimmer position
			if distance <= 2 {
				intensity := 1.0 - float64(distance)/3.0
				baseColor := p.style.FgRGB
				r := uint8(float64(baseColor.R) + float64(a.highlightColor.R-baseColor.R)*intensity)
				g := uint8(float64(baseColor.G) + float64(a.highlightColor.G-baseColor.G)*intensity)
				b := uint8(float64(baseColor.B) + float64(a.highlightColor.B-baseColor.B)*intensity)
				style = style.WithFgRGB(NewRGB(r, g, b))
			}
		}

		if a.pulse {
			// Calculate pulse brightness using sine wave (oscillates over time)
			pulsePhase := float64(frame) / float64(a.pulseSpeed)
			// Sine wave oscillates between -1 and 1, so scale to 0.3-1.0 range
			brightness := 0.65 + 0.35*math.Sin(pulsePhase)
			r := uint8(float64(a.pulseColor.R) * brightness)
			g := uint8(float64(a.pulseColor.G) * brightness)
			b := uint8(float64(a.pulseColor.B) * brightness)
			style = style.WithFgRGB(NewRGB(r, g, b))
		}

		ctx.SetCell(x+i, 0, p.filledChar, style)
	}
	x += barWidth

	// Draw percentage or fraction
	if p.showPercent && p.total > 0 {
		percent := (p.current * 100) / p.total
		text := fmt.Sprintf(" %3d%%", percent)
		ctx.PrintStyled(x, 0, text, p.style)
	} else if p.showFraction {
		text := fmt.Sprintf(" %d/%d", p.current, p.total)
		ctx.PrintStyled(x, 0, text, p.style)
	}
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

func (p *progressView) render(ctx *RenderContext) {
	width, height := ctx.Size()
	if width == 0 || height == 0 {
		return
	}

	x := 0

	// Draw label
	if p.label != "" {
		ctx.PrintStyled(x, 0, p.label, p.style)
		labelW, _ := MeasureText(p.label)
		x += labelW + 1
	}

	// Calculate available width for bar
	barWidth := p.width
	availableWidth := width - x
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
		ctx.SetCell(x+i, 0, p.emptyChar, p.emptyStyle)
	}

	// Draw filled portion
	for i := 0; i < fillWidth; i++ {
		ctx.SetCell(x+i, 0, p.filledChar, p.style)
	}
	x += barWidth

	// Draw percentage or fraction
	if p.showPercent && p.total > 0 {
		percent := (p.current * 100) / p.total
		text := fmt.Sprintf(" %3d%%", percent)
		ctx.PrintStyled(x, 0, text, p.style)
	} else if p.showFraction {
		text := fmt.Sprintf(" %d/%d", p.current, p.total)
		ctx.PrintStyled(x, 0, text, p.style)
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

func (s *loadingView) render(ctx *RenderContext) {
	width, height := ctx.Size()
	if width == 0 || height == 0 || len(s.charset) == 0 {
		return
	}

	// Calculate which character to show
	idx := int(s.frame/uint64(s.speed)) % len(s.charset)
	char := s.charset[idx]

	// Draw spinner
	ctx.PrintStyled(0, 0, char, s.style)

	// Draw label
	if s.label != "" {
		charW, _ := MeasureText(char)
		ctx.PrintStyled(charW+1, 0, s.label, s.style)
	}
}
