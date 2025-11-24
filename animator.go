package gooey

import (
	"errors"
	"sync"
	"time"
	"unicode/utf8"
)

// Animator manages multiple animated elements and provides a render loop.
//
// The Animator runs a dedicated goroutine that updates and renders all registered
// AnimatedElements at a configurable frame rate (default 30 FPS).
//
// Thread Safety: All public methods are thread-safe. The animator uses internal
// synchronization to protect element lists and state.
//
// Lifecycle:
//  1. Create with NewAnimator(terminal, fps)
//  2. Add elements with AddElement()
//  3. Call Start() to begin animation loop
//  4. Call Stop() to pause, or Close() to shutdown completely
//
// Example:
//
//	anim := NewAnimator(term, 60) // 60 FPS
//	anim.AddElement(myAnimatedText)
//	anim.Start()
//	defer anim.Close()
type Animator struct {
	terminal     *Terminal
	elements     []AnimatedElement
	mu           sync.RWMutex
	active       bool
	ticker       *time.Ticker
	stopChan     chan struct{}
	frameCounter uint64
	fps          int
	wg           sync.WaitGroup
}

// AnimatedElement represents any element that can be animated
type AnimatedElement interface {
	Update(frame uint64)
	Draw(frame RenderFrame)
	Position() (x, y int)
	Dimensions() (width, height int)
}

// NewAnimator creates a new animation engine
func NewAnimator(terminal *Terminal, fps int) *Animator {
	if fps <= 0 {
		fps = 30
	}
	return &Animator{
		terminal: terminal,
		elements: make([]AnimatedElement, 0),
		stopChan: make(chan struct{}),
		fps:      fps,
	}
}

// AddElement adds an animated element
func (a *Animator) AddElement(element AnimatedElement) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.elements = append(a.elements, element)
}

// RemoveElement removes an animated element
func (a *Animator) RemoveElement(element AnimatedElement) {
	a.mu.Lock()
	defer a.mu.Unlock()
	for i, el := range a.elements {
		if el == element {
			a.elements = append(a.elements[:i], a.elements[i+1:]...)
			break
		}
	}
}

// Start begins the animation loop.
//
// Errors:
//   - Returns ErrAlreadyActive if animator is already running
func (a *Animator) Start() error {
	a.mu.Lock()
	if a.active {
		a.mu.Unlock()
		return ErrAlreadyActive
	}
	a.active = true
	a.stopChan = make(chan struct{}) // Re-initialize stop channel

	// Reset ticker if needed or ensure it's running
	if a.ticker != nil {
		a.ticker.Stop() // Stop old one just in case
	}
	interval := time.Second / time.Duration(a.fps)
	a.ticker = time.NewTicker(interval)

	a.wg.Add(1)
	a.mu.Unlock()

	go a.animate()
	return nil
}

// Stop stops the animation loop
func (a *Animator) Stop() {
	a.mu.Lock()
	if !a.active {
		a.mu.Unlock()
		return
	}
	a.active = false
	close(a.stopChan)
	if a.ticker != nil {
		a.ticker.Stop()
	}
	a.mu.Unlock()

	a.wg.Wait() // Wait for animation loop to finish
}

// Close stops the animation loop and releases all resources.
// It implements the Component lifecycle pattern.
// After Close() is called, the Animator should not be reused.
func (a *Animator) Close() error {
	a.Stop()

	// Clear elements to release references
	a.mu.Lock()
	a.elements = nil
	a.mu.Unlock()

	return nil
}

func (a *Animator) animate() {
	defer a.wg.Done()
	for {
		select {
		case <-a.stopChan:
			return
		case <-a.ticker.C:
			a.mu.RLock()
			elements := make([]AnimatedElement, len(a.elements))
			copy(elements, a.elements)
			frameCounter := a.frameCounter
			a.mu.RUnlock()

			// Update all elements
			for _, element := range elements {
				element.Update(frameCounter)
			}

			// Begin atomic frame
			frame, err := a.terminal.BeginFrame()
			if err != nil {
				if errors.Is(err, ErrClosed) {
					return
				}
				continue
			}

			// Draw all elements
			for _, element := range elements {
				element.Draw(frame)
			}

			// End frame (flushes)
			if err := a.terminal.EndFrame(frame); err != nil {
				if errors.Is(err, ErrClosed) {
					return
				}
				continue
			}

			a.mu.Lock()
			a.frameCounter++
			a.mu.Unlock()
		}
	}
}

// AnimatedText represents text with color animation
type AnimatedText struct {
	x, y         int
	text         string
	animation    TextAnimation
	mu           sync.RWMutex
	currentFrame uint64
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
		r.Speed = 60
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

// WaveAnimation creates a wave-like color effect
type WaveAnimation struct {
	Speed     int
	Amplitude float64
	Colors    []RGB
}

// GetStyle returns the style for a character at a given frame
func (w *WaveAnimation) GetStyle(frame uint64, charIndex int, totalChars int) Style {
	if w.Speed <= 0 {
		w.Speed = 30
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

	// Calculate wave position
	waveTime := float64(frame) / float64(w.Speed)
	_ = float64(charIndex) / float64(totalChars) * 6.28 // 2*PI (position for potential use)
	wave := (1.0 + w.Amplitude*(0.5+0.5*waveTime)) * 0.5

	// Select color based on wave
	colorIndex := int(wave * float64(len(w.Colors)))
	if colorIndex >= len(w.Colors) {
		colorIndex = len(w.Colors) - 1
	}

	return NewStyle().WithForeground(ColorDefault) // We'll apply RGB directly
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
		p.Speed = 60
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
	at.mu.Lock()
	defer at.mu.Unlock()
	at.currentFrame = frame
}

// Draw renders the animated text
func (at *AnimatedText) Draw(frame RenderFrame) {
	at.mu.RLock()
	defer at.mu.RUnlock()

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
	at.mu.RLock()
	defer at.mu.RUnlock()
	return at.x, at.y
}

// Dimensions returns the element's dimensions
func (at *AnimatedText) Dimensions() (width, height int) {
	at.mu.RLock()
	defer at.mu.RUnlock()
	return utf8.RuneCountInString(at.text), 1
}

// SetText updates the text content
func (at *AnimatedText) SetText(text string) {
	at.mu.Lock()
	defer at.mu.Unlock()
	at.text = text
}

// SetPosition updates the position
func (at *AnimatedText) SetPosition(x, y int) {
	at.mu.Lock()
	defer at.mu.Unlock()
	at.x = x
	at.y = y
}

// AnimatedMultiLine represents multiple lines of animated content
type AnimatedMultiLine struct {
	x, y         int
	width        int
	lines        []string
	animations   []TextAnimation
	mu           sync.RWMutex
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
	aml.mu.Lock()
	defer aml.mu.Unlock()

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
	aml.mu.Lock()
	defer aml.mu.Unlock()
	aml.lines = append(aml.lines, text)
	aml.animations = append(aml.animations, animation)
}

// ClearLines removes all lines
func (aml *AnimatedMultiLine) ClearLines() {
	aml.mu.Lock()
	defer aml.mu.Unlock()
	aml.lines = aml.lines[:0]
	aml.animations = aml.animations[:0]
}

// Update updates the animation frame
func (aml *AnimatedMultiLine) Update(frame uint64) {
	aml.mu.Lock()
	defer aml.mu.Unlock()
	aml.currentFrame = frame
}

// Draw renders all animated lines
func (aml *AnimatedMultiLine) Draw(frame RenderFrame) {
	aml.mu.RLock()
	defer aml.mu.RUnlock()

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
	aml.mu.RLock()
	defer aml.mu.RUnlock()
	return aml.x, aml.y
}

// Dimensions returns the element's dimensions
func (aml *AnimatedMultiLine) Dimensions() (width, height int) {
	aml.mu.RLock()
	defer aml.mu.RUnlock()
	return aml.width, len(aml.lines)
}

// SetPosition updates the element's position
func (aml *AnimatedMultiLine) SetPosition(x, y int) {
	aml.mu.Lock()
	defer aml.mu.Unlock()
	aml.x = x
	aml.y = y
}

// SetWidth updates the element's width
func (aml *AnimatedMultiLine) SetWidth(width int) {
	aml.mu.Lock()
	defer aml.mu.Unlock()
	aml.width = width
}
