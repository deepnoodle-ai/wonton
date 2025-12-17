package tui

import (
	"unicode/utf8"
)

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
	Speed       int // How often sparkles update
	BaseColor   RGB
	SparkColor  RGB
	Density     int // Higher = more sparkles (1-10 recommended)
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

// AnimatedText represents text with color animation
type AnimatedText struct {
	x, y         int
	text         string
	animation    TextAnimation
	currentFrame uint64
}

// NewAnimatedText creates a new animated text element
func NewAnimatedText(x, y int, text string, animation TextAnimation) *AnimatedText {
	return &AnimatedText{
		x:         x,
		y:         y,
		text:      text,
		animation: animation,
	}
}

// Update updates the animation frame
func (at *AnimatedText) Update(frame uint64) {
	at.currentFrame = frame
}

// Draw renders the animated text
func (at *AnimatedText) Draw(frame RenderFrame) {
	runes := []rune(at.text)
	totalChars := len(runes)

	for i, r := range runes {
		currentX := at.x + i
		var style Style
		if at.animation != nil {
			// Use the animation's GetStyle method - consolidated rendering
			style = at.animation.GetStyle(at.currentFrame, i, totalChars)
		} else {
			style = NewStyle()
		}
		frame.SetCell(currentX, at.y, r, style)
	}
}

// Position returns the element's position
func (at *AnimatedText) Position() (x, y int) {
	return at.x, at.y
}

// Dimensions returns the element's dimensions
func (at *AnimatedText) Dimensions() (width, height int) {
	return utf8.RuneCountInString(at.text), 1
}

// SetText updates the text content
func (at *AnimatedText) SetText(text string) {
	at.text = text
}

// SetPosition updates the position
func (at *AnimatedText) SetPosition(x, y int) {
	at.x = x
	at.y = y
}

// AnimatedMultiLine represents multiple lines of animated content
type AnimatedMultiLine struct {
	x, y         int
	width        int
	lines        []string
	animations   []TextAnimation
	currentFrame uint64
}

// NewAnimatedMultiLine creates a new multi-line animated element
func NewAnimatedMultiLine(x, y, width int) *AnimatedMultiLine {
	return &AnimatedMultiLine{
		x:          x,
		y:          y,
		width:      width,
		lines:      make([]string, 0),
		animations: make([]TextAnimation, 0),
	}
}

// SetLine sets the content and animation for a specific line
func (aml *AnimatedMultiLine) SetLine(index int, text string, animation TextAnimation) {
	// Ensure slices are large enough
	for len(aml.lines) <= index {
		aml.lines = append(aml.lines, "")
		aml.animations = append(aml.animations, nil)
	}

	aml.lines[index] = text
	aml.animations[index] = animation
}

// AddLine adds a new line with animation
func (aml *AnimatedMultiLine) AddLine(text string, animation TextAnimation) {
	aml.lines = append(aml.lines, text)
	aml.animations = append(aml.animations, animation)
}

// ClearLines removes all lines
func (aml *AnimatedMultiLine) ClearLines() {
	aml.lines = aml.lines[:0]
	aml.animations = aml.animations[:0]
}

// Update updates the animation frame
func (aml *AnimatedMultiLine) Update(frame uint64) {
	aml.currentFrame = frame
}

// Draw renders all animated lines
func (aml *AnimatedMultiLine) Draw(frame RenderFrame) {
	frameW, _ := frame.Size()

	for i, line := range aml.lines {
		// Clear line first
		frame.FillStyled(0, aml.y+i, frameW, 1, ' ', NewStyle())

		if i < len(aml.animations) && aml.animations[i] != nil {
			// Draw animated line
			runes := []rune(line)
			totalChars := len(runes)

			for j, r := range runes {
				if j >= aml.width {
					break // Respect width limit
				}

				currentX := aml.x + j
				// Use the animation's GetStyle method - consolidated rendering
				style := aml.animations[i].GetStyle(aml.currentFrame, j, totalChars)
				frame.SetCell(currentX, aml.y+i, r, style)
			}
		} else {
			// Draw static line
			frame.PrintStyled(aml.x, aml.y+i, line, NewStyle())
		}
	}
}

// Position returns the element's position
func (aml *AnimatedMultiLine) Position() (x, y int) {
	return aml.x, aml.y
}

// Dimensions returns the element's dimensions
func (aml *AnimatedMultiLine) Dimensions() (width, height int) {
	return aml.width, len(aml.lines)
}

// SetPosition updates the element's position
func (aml *AnimatedMultiLine) SetPosition(x, y int) {
	aml.x = x
	aml.y = y
}

// SetWidth updates the element's width
func (aml *AnimatedMultiLine) SetWidth(width int) {
	aml.width = width
}

// AnimatedStatusBar represents an animated status bar
type AnimatedStatusBar struct {
	x, y         int
	width        int
	items        []*AnimatedStatusItem
	separator    string
	background   RGB
	currentFrame uint64
}

// AnimatedStatusItem represents a single status item with animation
type AnimatedStatusItem struct {
	Key       string
	Value     string
	Icon      string
	Animation TextAnimation
	Style     Style
}

// NewAnimatedStatusBar creates a new animated status bar
func NewAnimatedStatusBar(x, y, width int) *AnimatedStatusBar {
	return &AnimatedStatusBar{
		x:          x,
		y:          y,
		width:      width,
		items:      make([]*AnimatedStatusItem, 0),
		separator:  " │ ",
		background: NewRGB(40, 40, 40),
	}
}

// AddItem adds a status item with optional animation
func (asb *AnimatedStatusBar) AddItem(key, value, icon string, animation TextAnimation, style Style) {
	item := &AnimatedStatusItem{
		Key:       key,
		Value:     value,
		Icon:      icon,
		Animation: animation,
		Style:     style,
	}

	asb.items = append(asb.items, item)
}

// UpdateItem updates an existing status item
func (asb *AnimatedStatusBar) UpdateItem(index int, key, value string) {
	if index >= 0 && index < len(asb.items) {
		asb.items[index].Key = key
		asb.items[index].Value = value
	}
}

// Update updates the animation frame
func (asb *AnimatedStatusBar) Update(frame uint64) {
	asb.currentFrame = frame
}

// Draw renders the animated status bar
func (asb *AnimatedStatusBar) Draw(frame RenderFrame) {
	// Draw background
	bgStyle := NewStyle().WithBgRGB(asb.background)
	frame.FillStyled(asb.x, asb.y, asb.width, 1, ' ', bgStyle)

	currentX := asb.x
	for i, item := range asb.items {
		if currentX >= asb.x+asb.width {
			break
		}

		// Add separator if not first item
		if i > 0 {
			sepStyle := NewStyle().WithForeground(ColorBrightBlack).WithBgRGB(asb.background)
			frame.PrintStyled(currentX, asb.y, asb.separator, sepStyle)
			currentX += utf8.RuneCountInString(asb.separator)
		}

		// Draw icon if present
		if item.Icon != "" {
			iconText := item.Icon + " "
			if item.Animation != nil {
				// Apply animation to icon
				runes := []rune(iconText)
				for j, r := range runes {
					style := item.Animation.GetStyle(asb.currentFrame, j, len(runes))
					// Combine with background
					style = style.WithBgRGB(asb.background)
					frame.SetCell(currentX+j, asb.y, r, style)
				}
			} else {
				iconStyle := item.Style.WithBgRGB(asb.background)
				frame.PrintStyled(currentX, asb.y, iconText, iconStyle)
			}
			currentX += utf8.RuneCountInString(iconText)
		}

		// Draw key
		if item.Key != "" {
			keyText := item.Key + ": "
			keyStyle := item.Style
			if keyStyle.IsEmpty() {
				keyStyle = NewStyle().WithBold()
			}
			keyStyle = keyStyle.WithBgRGB(asb.background)
			frame.PrintStyled(currentX, asb.y, keyText, keyStyle)
			currentX += utf8.RuneCountInString(keyText)
		}

		// Draw value
		if item.Value != "" {
			if item.Animation != nil {
				// Apply animation to value
				runes := []rune(item.Value)
				for j, r := range runes {
					switch anim := item.Animation.(type) {
					case *RainbowAnimation:
						colors := SmoothRainbow(anim.Length)
						offset := int(asb.currentFrame) / anim.Speed
						if anim.Reversed {
							offset = -offset
						}
						rainbowPos := (j + offset) % len(colors)
						if rainbowPos < 0 {
							rainbowPos += len(colors)
						}
						rgb := colors[rainbowPos]
						// Combine with background
						style := NewStyle().WithFgRGB(rgb).WithBgRGB(asb.background)
						frame.SetCell(currentX+j, asb.y, r, style)
					default:
						style := item.Animation.GetStyle(asb.currentFrame, j, len(runes))
						style = style.WithBgRGB(asb.background)
						frame.SetCell(currentX+j, asb.y, r, style)
					}
				}
			} else {
				valueStyle := NewStyle().WithForeground(ColorWhite).WithBgRGB(asb.background)
				frame.PrintStyled(currentX, asb.y, item.Value, valueStyle)
			}
			currentX += utf8.RuneCountInString(item.Value)
		}
	}
}

// Position returns the status bar position
func (asb *AnimatedStatusBar) Position() (x, y int) {
	return asb.x, asb.y
}

// Dimensions returns the status bar dimensions
func (asb *AnimatedStatusBar) Dimensions() (width, height int) {
	return asb.width, 1
}

// SetPosition updates the status bar position
func (asb *AnimatedStatusBar) SetPosition(x, y int) {
	asb.x = x
	asb.y = y
}

