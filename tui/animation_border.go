package tui

import "math"

// BorderAnimation defines how a border should be animated.
type BorderAnimation interface {
	GetBorderStyle(frame uint64, borderPart BorderPart, position int, length int) Style
}

// BorderPart identifies which part of the border is being rendered.
type BorderPart int

const (
	BorderPartTop BorderPart = iota
	BorderPartRight
	BorderPartBottom
	BorderPartLeft
	BorderPartTopLeft
	BorderPartTopRight
	BorderPartBottomLeft
	BorderPartBottomRight
)

// RainbowBorderAnimation creates a rainbow that cycles around the border.
type RainbowBorderAnimation struct {
	Speed    int  // How fast the rainbow moves (frames per cycle)
	Reversed bool // Direction of movement
}

// GetBorderStyle returns the style for a border position at a given frame.
func (r *RainbowBorderAnimation) GetBorderStyle(frame uint64, part BorderPart, position int, length int) Style {
	if r.Speed <= 0 {
		r.Speed = 2
	}

	// Calculate a continuous position around the border perimeter
	// This makes the rainbow flow smoothly around corners
	offset := int(frame) / r.Speed
	if r.Reversed {
		offset = -offset
	}

	colors := SmoothRainbow(length)
	colorIndex := (position + offset) % len(colors)
	if colorIndex < 0 {
		colorIndex += len(colors)
	}

	return NewStyle().WithFgRGB(colors[colorIndex])
}

// PulseBorderAnimation creates a pulsing brightness effect on the border.
type PulseBorderAnimation struct {
	Speed         int
	Color         RGB
	MinBrightness float64
	MaxBrightness float64
	Easing        Easing // Optional easing function for the pulse
}

// GetBorderStyle returns the style for a border position at a given frame.
func (p *PulseBorderAnimation) GetBorderStyle(frame uint64, part BorderPart, position int, length int) Style {
	if p.Speed <= 0 {
		p.Speed = 15
	}
	if p.MinBrightness <= 0 {
		p.MinBrightness = 0.3
	}
	if p.MaxBrightness <= 0 {
		p.MaxBrightness = 1.0
	}
	if p.Easing == nil {
		p.Easing = EaseInOutSine
	}

	// Calculate pulse with easing
	cycleLength := float64(p.Speed * 2)
	progress := math.Mod(float64(frame), cycleLength) / cycleLength
	// Make it go 0->1->0
	if progress > 0.5 {
		progress = 1.0 - progress
	}
	progress *= 2.0

	easedProgress := p.Easing(progress)
	brightness := p.MinBrightness + (p.MaxBrightness-p.MinBrightness)*easedProgress

	adjustedColor := RGB{
		R: uint8(float64(p.Color.R) * brightness),
		G: uint8(float64(p.Color.G) * brightness),
		B: uint8(float64(p.Color.B) * brightness),
	}

	return NewStyle().WithFgRGB(adjustedColor)
}

// WaveBorderAnimation creates a wave that travels around the border.
type WaveBorderAnimation struct {
	Speed     int
	Colors    []RGB
	WaveWidth int // Width of the color wave in characters
}

// GetBorderStyle returns the style for a border position at a given frame.
func (w *WaveBorderAnimation) GetBorderStyle(frame uint64, part BorderPart, position int, length int) Style {
	if w.Speed <= 0 {
		w.Speed = 3
	}
	if w.WaveWidth <= 0 {
		w.WaveWidth = 10
	}
	if len(w.Colors) == 0 {
		w.Colors = []RGB{
			NewRGB(255, 0, 100),
			NewRGB(0, 255, 100),
			NewRGB(100, 0, 255),
		}
	}

	offset := int(frame) / w.Speed
	// Calculate position in wave
	wavePos := (position + offset) % (length + w.WaveWidth)

	// Determine which color to use based on position
	colorCycle := len(w.Colors)
	colorIndex := (wavePos / w.WaveWidth) % colorCycle
	nextColorIndex := (colorIndex + 1) % colorCycle

	// Blend between colors within the wave width
	posInWave := wavePos % w.WaveWidth
	blendFactor := float64(posInWave) / float64(w.WaveWidth)

	c1 := w.Colors[colorIndex]
	c2 := w.Colors[nextColorIndex]

	blendedColor := RGB{
		R: uint8(float64(c1.R) + (float64(c2.R)-float64(c1.R))*blendFactor),
		G: uint8(float64(c1.G) + (float64(c2.G)-float64(c1.G))*blendFactor),
		B: uint8(float64(c1.B) + (float64(c2.B)-float64(c1.B))*blendFactor),
	}

	return NewStyle().WithFgRGB(blendedColor)
}

// MarqueeBorderAnimation creates a moving pattern around the border.
type MarqueeBorderAnimation struct {
	Speed         int
	OnColor       RGB
	OffColor      RGB
	SegmentLength int  // Length of each on/off segment
	Reversed      bool // Direction of movement
}

// GetBorderStyle returns the style for a border position at a given frame.
func (m *MarqueeBorderAnimation) GetBorderStyle(frame uint64, part BorderPart, position int, length int) Style {
	if m.Speed <= 0 {
		m.Speed = 3
	}
	if m.SegmentLength <= 0 {
		m.SegmentLength = 3
	}

	offset := int(frame) / m.Speed
	if m.Reversed {
		offset = -offset
	}

	// Determine if this position is "on" or "off"
	adjustedPos := (position + offset) % (m.SegmentLength * 2)
	if adjustedPos < 0 {
		adjustedPos += m.SegmentLength * 2
	}

	if adjustedPos < m.SegmentLength {
		return NewStyle().WithFgRGB(m.OnColor)
	}
	return NewStyle().WithFgRGB(m.OffColor)
}

// SparkleBorderAnimation creates random sparkles around the border.
type SparkleBorderAnimation struct {
	Speed       int
	BaseColor   RGB
	SparkColor  RGB
	Density     int // Probability of sparkle (1-10)
	SparkleSize int // Size of each sparkle in characters
}

// GetBorderStyle returns the style for a border position at a given frame.
func (s *SparkleBorderAnimation) GetBorderStyle(frame uint64, part BorderPart, position int, length int) Style {
	if s.Speed <= 0 {
		s.Speed = 2
	}
	if s.Density <= 0 {
		s.Density = 3
	}
	if s.SparkleSize <= 0 {
		s.SparkleSize = 1
	}

	// Pseudo-random sparkle generation
	seed := uint64(position*7919 + 104729)
	sparklePhase := (frame / uint64(s.Speed)) + seed

	cycleLength := uint64(30 + (position % 20))
	posInCycle := sparklePhase % cycleLength

	isSparkle := posInCycle < uint64(s.Density)

	if isSparkle {
		// Calculate sparkle intensity
		intensity := 1.0 - float64(posInCycle)/float64(s.Density)

		r := uint8(float64(s.BaseColor.R) + float64(s.SparkColor.R-s.BaseColor.R)*intensity)
		g := uint8(float64(s.BaseColor.G) + float64(s.SparkColor.G-s.BaseColor.G)*intensity)
		b := uint8(float64(s.BaseColor.B) + float64(s.SparkColor.B-s.BaseColor.B)*intensity)

		return NewStyle().WithFgRGB(NewRGB(r, g, b))
	}

	return NewStyle().WithFgRGB(s.BaseColor)
}

// GradientBorderAnimation creates a gradient that rotates around the border.
type GradientBorderAnimation struct {
	Speed    int
	Colors   []RGB
	Reversed bool
}

// GetBorderStyle returns the style for a border position at a given frame.
func (g *GradientBorderAnimation) GetBorderStyle(frame uint64, part BorderPart, position int, length int) Style {
	if g.Speed <= 0 {
		g.Speed = 5
	}
	if len(g.Colors) < 2 {
		g.Colors = []RGB{
			NewRGB(255, 0, 0),
			NewRGB(0, 0, 255),
		}
	}

	offset := int(frame) / g.Speed
	if g.Reversed {
		offset = -offset
	}

	// Create a gradient across the entire border
	gradient := MultiGradient(g.Colors, length)

	colorIndex := (position + offset) % len(gradient)
	if colorIndex < 0 {
		colorIndex += len(gradient)
	}

	return NewStyle().WithFgRGB(gradient[colorIndex])
}

// CornerHighlightAnimation highlights corners in sequence.
type CornerHighlightAnimation struct {
	Speed          int
	BaseColor      RGB
	HighlightColor RGB
	Duration       int // How long each corner stays highlighted (in frames)
}

// GetBorderStyle returns the style for a border position at a given frame.
func (c *CornerHighlightAnimation) GetBorderStyle(frame uint64, part BorderPart, position int, length int) Style {
	if c.Speed <= 0 {
		c.Speed = 1
	}
	if c.Duration <= 0 {
		c.Duration = 30
	}

	// Cycle through corners: TL -> TR -> BR -> BL
	cycle := (int(frame) / c.Speed) / c.Duration
	activeCorner := cycle % 4

	isHighlighted := false
	switch part {
	case BorderPartTopLeft:
		isHighlighted = activeCorner == 0
	case BorderPartTopRight:
		isHighlighted = activeCorner == 1
	case BorderPartBottomRight:
		isHighlighted = activeCorner == 2
	case BorderPartBottomLeft:
		isHighlighted = activeCorner == 3
	}

	// Also highlight adjacent border sections
	if !isHighlighted && (part == BorderPartTop || part == BorderPartRight ||
		part == BorderPartBottom || part == BorderPartLeft) {
		// Fade highlight into adjacent borders
		fadeWidth := 5
		if position < fadeWidth {
			// Near start of this border section
			switch part {
			case BorderPartTop:
				if activeCorner == 0 || activeCorner == 3 { // TL or BL (wraps around)
					intensity := 1.0 - float64(position)/float64(fadeWidth)
					return c.blendColors(c.BaseColor, c.HighlightColor, intensity)
				}
			case BorderPartRight:
				if activeCorner == 1 { // TR
					intensity := 1.0 - float64(position)/float64(fadeWidth)
					return c.blendColors(c.BaseColor, c.HighlightColor, intensity)
				}
			case BorderPartBottom:
				if activeCorner == 2 { // BR
					intensity := 1.0 - float64(position)/float64(fadeWidth)
					return c.blendColors(c.BaseColor, c.HighlightColor, intensity)
				}
			case BorderPartLeft:
				if activeCorner == 3 { // BL
					intensity := 1.0 - float64(position)/float64(fadeWidth)
					return c.blendColors(c.BaseColor, c.HighlightColor, intensity)
				}
			}
		}
	}

	if isHighlighted {
		return NewStyle().WithFgRGB(c.HighlightColor)
	}
	return NewStyle().WithFgRGB(c.BaseColor)
}

func (c *CornerHighlightAnimation) blendColors(from, to RGB, intensity float64) Style {
	r := uint8(float64(from.R) + float64(to.R-from.R)*intensity)
	g := uint8(float64(from.G) + float64(to.G-from.G)*intensity)
	b := uint8(float64(from.B) + float64(to.B-from.B)*intensity)
	return NewStyle().WithFgRGB(NewRGB(r, g, b))
}

// FireBorderAnimation creates a flickering fire effect around the border.
type FireBorderAnimation struct {
	Speed      int
	CoolColors []RGB // Cooler fire colors (base)
	HotColors  []RGB // Hotter fire colors (peaks)
}

// GetBorderStyle returns the style for a border position at a given frame.
func (f *FireBorderAnimation) GetBorderStyle(frame uint64, part BorderPart, position int, length int) Style {
	if f.Speed <= 0 {
		f.Speed = 1
	}
	if len(f.CoolColors) == 0 {
		f.CoolColors = []RGB{NewRGB(200, 50, 0), NewRGB(150, 30, 0)}
	}
	if len(f.HotColors) == 0 {
		f.HotColors = []RGB{NewRGB(255, 150, 0), NewRGB(255, 200, 100)}
	}

	// Pseudo-random flicker
	seed := uint64(position*6547 + 32771)
	flicker := (frame/uint64(f.Speed) + seed) * 2654435761 % 100

	// Determine intensity based on flicker
	intensity := float64(flicker) / 100.0

	// Choose color based on intensity
	var color RGB
	if intensity > 0.7 {
		// Hot flash
		idx := int(intensity*10) % len(f.HotColors)
		color = f.HotColors[idx]
	} else {
		// Cool/normal
		idx := int(intensity*10) % len(f.CoolColors)
		color = f.CoolColors[idx]
	}

	return NewStyle().WithFgRGB(color)
}
