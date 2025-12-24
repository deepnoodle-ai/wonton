package tui

// AnimatedElement represents any element that can be animated
type AnimatedElement interface {
	Update(frame uint64)
	Draw(frame RenderFrame)
	Position() (x, y int)
	Dimensions() (width, height int)
}

// TextAnimation defines how text should be animated
type TextAnimation interface {
	GetStyle(frame uint64, charIndex int, totalChars int) Style
}

// RainbowAnimation creates a moving rainbow effect
type RainbowAnimation struct {
	Speed    int // How fast the rainbow moves (frames per cycle)
	Length   int // How many characters the rainbow spans
	Reversed bool
}

// Rainbow creates a new rainbow animation with the specified speed.
// Speed controls how fast the rainbow moves (lower = faster).
//
// Example:
//
//	Text("Hello").Animate(Rainbow(3))
//	Text("Hello").Animate(Rainbow(3).Reverse())
func Rainbow(speed int) *RainbowAnimation {
	return &RainbowAnimation{Speed: speed}
}

// Reverse sets the rainbow to move in the opposite direction.
func (r *RainbowAnimation) Reverse() *RainbowAnimation {
	r.Reversed = true
	return r
}

// WithLength sets how many characters the rainbow spans.
// If not set, defaults to the text length.
func (r *RainbowAnimation) WithLength(length int) *RainbowAnimation {
	r.Length = length
	return r
}

// GetStyle returns the style for a character at a given frame
func (r *RainbowAnimation) GetStyle(frame uint64, charIndex int, totalChars int) Style {
	if r.Length <= 0 {
		r.Length = totalChars
	}
	if r.Speed <= 0 {
		r.Speed = 3
	}

	// Get rainbow colors and apply directly
	colors := SmoothRainbow(r.Length)
	offset := int(frame) / r.Speed
	if r.Reversed {
		offset = -offset
	}
	rainbowPos := (charIndex + offset) % len(colors)
	if rainbowPos < 0 {
		rainbowPos += len(colors)
	}
	rgb := colors[rainbowPos]
	return NewStyle().WithFgRGB(rgb)
}

// WaveAnimation creates a wave-like color effect that flows across text
type WaveAnimation struct {
	Speed     int
	Amplitude float64
	Colors    []RGB
}

// Wave creates a new wave animation with the specified speed and colors.
// Speed controls how fast the wave moves (lower = faster).
// Colors are the colors to cycle through; defaults to magenta/green/purple if none provided.
//
// Example:
//
//	Text("Hello").Animate(Wave(5))
//	Text("Hello").Animate(Wave(5, NewRGB(255, 0, 0), NewRGB(0, 0, 255)))
func Wave(speed int, colors ...RGB) *WaveAnimation {
	return &WaveAnimation{Speed: speed, Colors: colors}
}

// WithAmplitude sets the wave amplitude.
func (w *WaveAnimation) WithAmplitude(amplitude float64) *WaveAnimation {
	w.Amplitude = amplitude
	return w
}

// GetStyle returns the style for a character at a given frame
func (w *WaveAnimation) GetStyle(frame uint64, charIndex int, totalChars int) Style {
	if w.Speed <= 0 {
		w.Speed = 12 // Default to gentle speed
	}
	if w.Amplitude <= 0 {
		w.Amplitude = 1.0
	}
	if len(w.Colors) == 0 {
		w.Colors = []RGB{
			NewRGB(255, 0, 100),
			NewRGB(0, 255, 100),
			NewRGB(100, 0, 255),
		}
	}

	// Create a wave that flows across the text
	// Each character is offset in the wave based on its position
	numColors := len(w.Colors)
	waveOffset := int(frame) / w.Speed
	colorIndex := (charIndex + waveOffset) % numColors
	if colorIndex < 0 {
		colorIndex += numColors
	}

	// Blend between adjacent colors for smoother transitions
	nextColorIndex := (colorIndex + 1) % numColors
	// Calculate blend factor based on sub-frame position
	blendPhase := float64(int(frame)%w.Speed) / float64(w.Speed)

	c1 := w.Colors[colorIndex]
	c2 := w.Colors[nextColorIndex]

	// Linear interpolation between colors
	r := uint8(float64(c1.R) + (float64(c2.R)-float64(c1.R))*blendPhase)
	g := uint8(float64(c1.G) + (float64(c2.G)-float64(c1.G))*blendPhase)
	b := uint8(float64(c1.B) + (float64(c2.B)-float64(c1.B))*blendPhase)

	return NewStyle().WithFgRGB(NewRGB(r, g, b))
}

// SlideAnimation creates a highlight that slides across the text
type SlideAnimation struct {
	Speed          int
	BaseColor      RGB
	HighlightColor RGB
	Width          int  // Width of the highlight in characters
	Reverse        bool // True = right to left, false = left to right
}

// Slide creates a new slide animation with a highlight that moves across text.
// Speed controls how fast the highlight moves (lower = faster).
//
// Example:
//
//	Text("Loading").Animate(Slide(2, gray, white))
//	Text("Loading").Animate(Slide(2, gray, white).Reversed())
func Slide(speed int, baseColor, highlightColor RGB) *SlideAnimation {
	return &SlideAnimation{
		Speed:          speed,
		BaseColor:      baseColor,
		HighlightColor: highlightColor,
	}
}

// Reversed sets the slide to move from right to left.
func (s *SlideAnimation) Reversed() *SlideAnimation {
	s.Reverse = true
	return s
}

// WithWidth sets the width of the highlight in characters.
func (s *SlideAnimation) WithWidth(width int) *SlideAnimation {
	s.Width = width
	return s
}

// GetStyle returns the style for a character at a given frame
func (s *SlideAnimation) GetStyle(frame uint64, charIndex int, totalChars int) Style {
	if s.Speed <= 0 {
		s.Speed = 2
	}
	if s.Width <= 0 {
		s.Width = 3
	}

	// Calculate highlight position (slides across text)
	cycleLength := totalChars + s.Width*2 // Extra space for highlight to fully enter/exit
	highlightPos := int(frame/uint64(s.Speed)) % cycleLength

	if s.Reverse {
		highlightPos = cycleLength - 1 - highlightPos
	}

	// Adjust position to account for highlight entering from off-screen
	highlightPos = highlightPos - s.Width

	// Calculate distance from highlight center
	distance := charIndex - highlightPos
	if distance < 0 {
		distance = -distance
	}

	// Apply highlight if within range
	if distance <= s.Width {
		// Smooth falloff from center
		intensity := 1.0 - float64(distance)/float64(s.Width+1)
		r := uint8(float64(s.BaseColor.R) + float64(s.HighlightColor.R-s.BaseColor.R)*intensity)
		g := uint8(float64(s.BaseColor.G) + float64(s.HighlightColor.G-s.BaseColor.G)*intensity)
		b := uint8(float64(s.BaseColor.B) + float64(s.HighlightColor.B-s.BaseColor.B)*intensity)
		return NewStyle().WithFgRGB(NewRGB(r, g, b))
	}

	return NewStyle().WithFgRGB(s.BaseColor)
}

// SparkleAnimation creates a twinkling star-like effect where random characters briefly brighten
type SparkleAnimation struct {
	Speed      int // How often sparkles update
	BaseColor  RGB
	SparkColor RGB
	Density    int // Higher = more sparkles (1-10 recommended)
}

// Sparkle creates a new sparkle animation with twinkling star-like effects.
// Speed controls animation timing (lower = faster).
//
// Example:
//
//	Text("Stars").Animate(Sparkle(3, gray, white))
//	Text("Stars").Animate(Sparkle(3, gray, white).WithDensity(5))
func Sparkle(speed int, baseColor, sparkColor RGB) *SparkleAnimation {
	return &SparkleAnimation{
		Speed:      speed,
		BaseColor:  baseColor,
		SparkColor: sparkColor,
		Density:    3,
	}
}

// WithDensity sets how many sparkles appear (1-10 recommended, higher = more sparkles).
func (s *SparkleAnimation) WithDensity(density int) *SparkleAnimation {
	s.Density = density
	return s
}

// GetStyle returns the style for a character at a given frame
func (s *SparkleAnimation) GetStyle(frame uint64, charIndex int, totalChars int) Style {
	if s.Speed <= 0 {
		s.Speed = 3
	}
	if s.Density <= 0 {
		s.Density = 3
	}

	// Use a pseudo-random approach based on frame and character position
	// This creates deterministic but seemingly random sparkles
	seed := uint64(charIndex*7919 + 104729) // Prime numbers for good distribution
	sparklePhase := (frame / uint64(s.Speed)) + seed

	// Create sparkle pattern using modular arithmetic
	// Each character has its own sparkle cycle
	cycleLength := uint64(20 + (charIndex % 15)) // Vary cycle per character
	posInCycle := sparklePhase % cycleLength

	// Sparkle occurs at specific points in the cycle
	isSparkle := posInCycle < uint64(s.Density)

	if isSparkle {
		// Calculate sparkle intensity (builds up then fades)
		intensity := 1.0 - float64(posInCycle)/float64(s.Density)
		r := uint8(float64(s.BaseColor.R) + float64(s.SparkColor.R-s.BaseColor.R)*intensity)
		g := uint8(float64(s.BaseColor.G) + float64(s.SparkColor.G-s.BaseColor.G)*intensity)
		b := uint8(float64(s.BaseColor.B) + float64(s.SparkColor.B-s.BaseColor.B)*intensity)
		return NewStyle().WithFgRGB(NewRGB(r, g, b))
	}

	return NewStyle().WithFgRGB(s.BaseColor)
}

// TypewriterAnimation reveals text character by character with a blinking cursor
type TypewriterAnimation struct {
	Speed       int // Frames per character reveal
	TextColor   RGB
	CursorColor RGB
	Loop        bool // Whether to restart after fully revealed
	HoldFrames  int  // Frames to hold before looping (if Loop is true)
}

// Typewriter creates a new typewriter animation that reveals text character by character.
// Speed controls how fast characters appear (lower = faster).
//
// Example:
//
//	Text("Hello, World!").Animate(Typewriter(4, white, green))
//	Text("Hello, World!").Animate(Typewriter(4, white, green).WithLoop(true))
func Typewriter(speed int, textColor, cursorColor RGB) *TypewriterAnimation {
	return &TypewriterAnimation{
		Speed:       speed,
		TextColor:   textColor,
		CursorColor: cursorColor,
		HoldFrames:  60, // Default hold frames
	}
}

// WithLoop sets whether the animation should restart after fully revealed.
func (t *TypewriterAnimation) WithLoop(loop bool) *TypewriterAnimation {
	t.Loop = loop
	return t
}

// WithHoldFrames sets how many frames to hold before looping (if Loop is true).
func (t *TypewriterAnimation) WithHoldFrames(frames int) *TypewriterAnimation {
	t.HoldFrames = frames
	return t
}

// GetStyle returns the style for a character at a given frame
func (t *TypewriterAnimation) GetStyle(frame uint64, charIndex int, totalChars int) Style {
	if t.Speed <= 0 {
		t.Speed = 4
	}
	if t.HoldFrames <= 0 {
		t.HoldFrames = 60
	}

	// Calculate how many characters should be visible
	var revealedChars int
	if t.Loop {
		cycleLength := totalChars*t.Speed + t.HoldFrames
		posInCycle := int(frame) % cycleLength
		revealedChars = posInCycle / t.Speed
		if revealedChars > totalChars {
			revealedChars = totalChars
		}
	} else {
		revealedChars = int(frame) / t.Speed
		if revealedChars > totalChars {
			revealedChars = totalChars
		}
	}

	// Character not yet revealed - render as invisible/dim
	if charIndex >= revealedChars {
		// Cursor position - blink it
		if charIndex == revealedChars {
			// Blink cursor every 15 frames
			if (frame/15)%2 == 0 {
				return NewStyle().WithFgRGB(t.CursorColor)
			}
		}
		// Not revealed yet - very dim
		dimColor := NewRGB(t.TextColor.R/8, t.TextColor.G/8, t.TextColor.B/8)
		return NewStyle().WithFgRGB(dimColor)
	}

	// Character is revealed
	return NewStyle().WithFgRGB(t.TextColor)
}

// GlitchAnimation creates a cyberpunk-style digital glitch effect
type GlitchAnimation struct {
	Speed       int // Base speed for glitch timing
	BaseColor   RGB
	GlitchColor RGB // Color during glitch
	Intensity   int // How often glitches occur (1-10, higher = more glitches)
}

// Glitch creates a new glitch animation with a cyberpunk-style digital effect.
// Speed controls glitch timing (lower = faster).
//
// Example:
//
//	Text("SYSTEM ERROR").Animate(Glitch(2, gray, red))
//	Text("SYSTEM ERROR").Animate(Glitch(2, gray, red).WithIntensity(8))
func Glitch(speed int, baseColor, glitchColor RGB) *GlitchAnimation {
	return &GlitchAnimation{
		Speed:       speed,
		BaseColor:   baseColor,
		GlitchColor: glitchColor,
		Intensity:   3,
	}
}

// WithIntensity sets how often glitches occur (1-10, higher = more glitches).
func (g *GlitchAnimation) WithIntensity(intensity int) *GlitchAnimation {
	g.Intensity = intensity
	return g
}

// GetStyle returns the style for a character at a given frame
func (g *GlitchAnimation) GetStyle(frame uint64, charIndex int, totalChars int) Style {
	if g.Speed <= 0 {
		g.Speed = 2
	}
	if g.Intensity <= 0 {
		g.Intensity = 3
	}

	// Create pseudo-random glitch pattern
	seed := uint64(charIndex*6547 + 32771)
	glitchPhase := (frame / uint64(g.Speed)) + seed

	// Multiple layers of glitch patterns for more organic feel
	pattern1 := (glitchPhase * 7) % 100
	pattern2 := (glitchPhase * 13) % 67
	pattern3 := ((frame + uint64(charIndex*3)) / 3) % 23

	// Combine patterns to determine if glitching
	isGlitch := pattern1 < uint64(g.Intensity*2) ||
		(pattern2 < uint64(g.Intensity) && pattern3 < 5)

	if isGlitch {
		// Vary the glitch color slightly for more visual interest
		variation := int(glitchPhase % 3)
		var r, gVal, b uint8
		switch variation {
		case 0:
			// Primary glitch color
			r, gVal, b = g.GlitchColor.R, g.GlitchColor.G, g.GlitchColor.B
		case 1:
			// Shifted toward cyan
			r = g.GlitchColor.R / 2
			gVal = g.GlitchColor.G
			b = g.GlitchColor.B
		case 2:
			// Brighter flash
			r = min(255, g.GlitchColor.R+50)
			gVal = min(255, g.GlitchColor.G+50)
			b = min(255, g.GlitchColor.B+50)
		}
		return NewStyle().WithFgRGB(NewRGB(r, gVal, b))
	}

	return NewStyle().WithFgRGB(g.BaseColor)
}

// PulseAnimation creates a pulsing brightness effect
type PulseAnimation struct {
	Speed         int
	Color         RGB
	MinBrightness float64
	MaxBrightness float64
}

// Pulse creates a new pulse animation with a pulsing brightness effect.
// Color is the base color, speed controls the pulse rate (lower = faster).
//
// Example:
//
//	Text("Active").Animate(Pulse(green, 10))
//	Text("Active").Animate(Pulse(green, 10).Brightness(0.3, 1.0))
func Pulse(color RGB, speed int) *PulseAnimation {
	return &PulseAnimation{
		Color: color,
		Speed: speed,
	}
}

// Brightness sets the minimum and maximum brightness levels (0.0 to 1.0).
func (p *PulseAnimation) Brightness(minBrightness, maxBrightness float64) *PulseAnimation {
	p.MinBrightness = minBrightness
	p.MaxBrightness = maxBrightness
	return p
}

// GetStyle returns the style for a character at a given frame
func (p *PulseAnimation) GetStyle(frame uint64, charIndex int, totalChars int) Style {
	if p.Speed <= 0 {
		p.Speed = 15
	}
	if p.MinBrightness <= 0 {
		p.MinBrightness = 0.3
	}
	if p.MaxBrightness <= 0 {
		p.MaxBrightness = 1.0
	}

	// Calculate pulse
	pulseTime := float64(frame) / float64(p.Speed)
	brightness := p.MinBrightness + (p.MaxBrightness-p.MinBrightness)*(0.5+0.5*Sine(pulseTime))

	// Apply brightness to color
	adjustedColor := RGB{
		R: uint8(float64(p.Color.R) * brightness),
		G: uint8(float64(p.Color.G) * brightness),
		B: uint8(float64(p.Color.B) * brightness),
	}

	return NewStyle().WithFgRGB(adjustedColor)
}

// Sine helper for pulse calculations
func Sine(x float64) float64 {
	// Simple sine approximation
	x = x - float64(int(x/6.28318))*6.28318 // Normalize to 0-2π
	if x < 3.14159 {
		return 4 * x * (3.14159 - x) / (3.14159 * 3.14159)
	}
	x = x - 3.14159
	return -4 * x * (3.14159 - x) / (3.14159 * 3.14159)
}
