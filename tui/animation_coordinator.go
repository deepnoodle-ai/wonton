package tui

import (
	"image"
	"sync"
)

// GlobalAnimationCoordinator manages cross-view animations and global effects.
// It allows views to opt into synchronized animations and global effects.
var GlobalAnimationCoordinator = NewAnimationCoordinator()

// AnimationCoordinator coordinates animations across multiple views.
type AnimationCoordinator struct {
	effects      map[string]*GlobalEffect
	subscribers  map[string][]AnimationSubscriber
	currentFrame uint64
	mu           sync.RWMutex
}

// NewAnimationCoordinator creates a new animation coordinator.
func NewAnimationCoordinator() *AnimationCoordinator {
	return &AnimationCoordinator{
		effects:     make(map[string]*GlobalEffect),
		subscribers: make(map[string][]AnimationSubscriber),
	}
}

// Update updates all global effects and notifies subscribers.
func (ac *AnimationCoordinator) Update(frame uint64) {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	ac.currentFrame = frame
	for _, effect := range ac.effects {
		effect.Update(frame)
	}
}

// RegisterEffect registers a global effect with a unique name.
func (ac *AnimationCoordinator) RegisterEffect(name string, effect *GlobalEffect) {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	ac.effects[name] = effect
}

// GetEffect retrieves a registered global effect.
func (ac *AnimationCoordinator) GetEffect(name string) *GlobalEffect {
	ac.mu.RLock()
	defer ac.mu.RUnlock()
	return ac.effects[name]
}

// Subscribe allows a view to subscribe to a global effect.
func (ac *AnimationCoordinator) Subscribe(effectName string, subscriber AnimationSubscriber) {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	ac.subscribers[effectName] = append(ac.subscribers[effectName], subscriber)
}

// Unsubscribe removes a subscriber from a global effect.
func (ac *AnimationCoordinator) Unsubscribe(effectName string, subscriber AnimationSubscriber) {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	subs := ac.subscribers[effectName]
	for i, sub := range subs {
		if sub == subscriber {
			ac.subscribers[effectName] = append(subs[:i], subs[i+1:]...)
			break
		}
	}
}

// GetSubscribers returns all subscribers for a given effect.
func (ac *AnimationCoordinator) GetSubscribers(effectName string) []AnimationSubscriber {
	ac.mu.RLock()
	defer ac.mu.RUnlock()
	return ac.subscribers[effectName]
}

// CurrentFrame returns the current frame counter.
func (ac *AnimationCoordinator) CurrentFrame() uint64 {
	ac.mu.RLock()
	defer ac.mu.RUnlock()
	return ac.currentFrame
}

// AnimationSubscriber represents a view that can subscribe to global effects.
type AnimationSubscriber interface {
	OnEffectUpdate(effectName string, value float64)
}

// GlobalEffect represents a global animation effect that views can subscribe to.
type GlobalEffect struct {
	animation     *Animation
	effectType    GlobalEffectType
	currentValue  float64
	colorValue    RGB
	easingFunc    Easing
	waveAmplitude float64
	waveFrequency float64
}

// GlobalEffectType defines the type of global effect.
type GlobalEffectType int

const (
	GlobalEffectBrightness GlobalEffectType = iota
	GlobalEffectColorShift
	GlobalEffectWave
	GlobalEffectPulse
	GlobalEffectShake
)

// NewGlobalEffect creates a new global effect.
func NewGlobalEffect(effectType GlobalEffectType, duration uint64) *GlobalEffect {
	return &GlobalEffect{
		animation:  NewAnimation(duration).WithLoop(true),
		effectType: effectType,
		easingFunc: EaseInOutSine,
	}
}

// WithEasing sets the easing function for the effect.
func (ge *GlobalEffect) WithEasing(easing Easing) *GlobalEffect {
	ge.animation = ge.animation.WithEasing(easing)
	return ge
}

// WithWaveParams sets wave parameters for wave-based effects.
func (ge *GlobalEffect) WithWaveParams(amplitude, frequency float64) *GlobalEffect {
	ge.waveAmplitude = amplitude
	ge.waveFrequency = frequency
	return ge
}

// WithColorValue sets a color value for color-based effects.
func (ge *GlobalEffect) WithColorValue(color RGB) *GlobalEffect {
	ge.colorValue = color
	return ge
}

// Start starts the global effect.
func (ge *GlobalEffect) Start(frame uint64) {
	ge.animation.Start(frame)
}

// Update updates the global effect.
func (ge *GlobalEffect) Update(frame uint64) {
	ge.animation.Update(frame)
	ge.currentValue = ge.animation.Value()
}

// Value returns the current effect value.
func (ge *GlobalEffect) Value() float64 {
	return ge.currentValue
}

// ColorValue returns the current color value (for color effects).
func (ge *GlobalEffect) ColorValue() RGB {
	return ge.colorValue
}

// Type returns the effect type.
func (ge *GlobalEffect) Type() GlobalEffectType {
	return ge.effectType
}

// coordinatedView wraps a view to participate in global animations.
type coordinatedView struct {
	inner         View
	effectName    string
	coordinator   *AnimationCoordinator
	transformFunc func(View, float64) View
}

// CoordinatedView wraps a view to participate in a global animation effect.
// transformFunc is called with the current effect value to transform the view.
func CoordinatedView(inner View, effectName string, transformFunc func(View, float64) View) *coordinatedView {
	return &coordinatedView{
		inner:         inner,
		effectName:    effectName,
		coordinator:   GlobalAnimationCoordinator,
		transformFunc: transformFunc,
	}
}

func (cv *coordinatedView) size(maxWidth, maxHeight int) (int, int) {
	return cv.inner.size(maxWidth, maxHeight)
}

func (cv *coordinatedView) render(ctx *RenderContext) {
	// Get the global effect value
	effect := cv.coordinator.GetEffect(cv.effectName)
	if effect == nil {
		// No effect registered, just render normally
		cv.inner.render(ctx)
		return
	}

	// Apply transformation based on effect value
	transformedView := cv.transformFunc(cv.inner, effect.Value())
	transformedView.render(ctx)
}

// Predefined transform functions for common effects

// BrightnessTransform creates a transform that adjusts brightness based on effect value.
func BrightnessTransform(minBright, maxBright float64) func(View, float64) View {
	return func(inner View, value float64) View {
		brightness := minBright + (maxBright-minBright)*value
		anim := NewAnimation(1).WithEasing(EaseLinear)
		anim.Start(0)
		anim.value = value
		return Brightness(inner, anim, brightness, brightness)
	}
}

// FadeTransform creates a transform that fades in/out based on effect value.
func FadeTransform(minOpacity, maxOpacity float64) func(View, float64) View {
	return func(inner View, value float64) View {
		opacity := minOpacity + (maxOpacity-minOpacity)*value
		anim := NewAnimation(1).WithEasing(EaseLinear)
		anim.Start(0)
		anim.value = value
		return Fade(inner, anim, opacity, opacity)
	}
}

// globalWaveView applies a wave effect across the entire screen.
type globalWaveView struct {
	inner     View
	amplitude float64
	frequency float64
	speed     int
	direction WaveDirection
}

// WaveDirection specifies the direction of wave propagation.
type WaveDirection int

const (
	WaveHorizontal WaveDirection = iota
	WaveVertical
	WaveDiagonal
	WaveRadial
)

// GlobalWave wraps a view with a global wave effect.
func GlobalWave(inner View, amplitude, frequency float64, speed int, direction WaveDirection) *globalWaveView {
	return &globalWaveView{
		inner:     inner,
		amplitude: amplitude,
		frequency: frequency,
		speed:     speed,
		direction: direction,
	}
}

func (gw *globalWaveView) size(maxWidth, maxHeight int) (int, int) {
	return gw.inner.size(maxWidth, maxHeight)
}

func (gw *globalWaveView) render(ctx *RenderContext) {
	wrappedFrame := &waveFrame{
		inner:     ctx.RenderFrame(),
		frame:     ctx.Frame(),
		amplitude: gw.amplitude,
		frequency: gw.frequency,
		speed:     gw.speed,
		direction: gw.direction,
	}

	wrappedCtx := ctx.WithFrame(wrappedFrame)
	gw.inner.render(wrappedCtx)
}

type waveFrame struct {
	inner     RenderFrame
	frame     uint64
	amplitude float64
	frequency float64
	speed     int
	direction WaveDirection
}

func (w *waveFrame) SetCell(x, y int, char rune, style Style) error {
	// Calculate wave offset based on position and direction
	var waveInput float64
	switch w.direction {
	case WaveHorizontal:
		waveInput = float64(x)
	case WaveVertical:
		waveInput = float64(y)
	case WaveDiagonal:
		waveInput = float64(x + y)
	case WaveRadial:
		width, height := w.inner.Size()
		centerX, centerY := width/2, height/2
		dx, dy := float64(x-centerX), float64(y-centerY)
		waveInput = dx*dx + dy*dy // Distance from center
	}

	// Apply wave function
	timeOffset := float64(w.frame) / float64(w.speed)
	waveValue := w.amplitude * Sine((waveInput*w.frequency+timeOffset)*0.5)

	// Modulate brightness based on wave
	brightness := 1.0 + waveValue

	if style.FgRGB != nil && (style.FgRGB.R != 0 || style.FgRGB.G != 0 || style.FgRGB.B != 0) {
		style = style.WithFgRGB(RGB{
			R: uint8(float64(style.FgRGB.R) * brightness),
			G: uint8(float64(style.FgRGB.G) * brightness),
			B: uint8(float64(style.FgRGB.B) * brightness),
		})
	}

	return w.inner.SetCell(x, y, char, style)
}

func (w *waveFrame) Size() (int, int)                                             { return w.inner.Size() }
func (w *waveFrame) PrintStyled(x, y int, text string, style Style) error        { return w.inner.PrintStyled(x, y, text, style) }
func (w *waveFrame) PrintTruncated(x, y int, text string, style Style) error     { return w.inner.PrintTruncated(x, y, text, style) }
func (w *waveFrame) FillStyled(x, y, width, h int, char rune, style Style) error { return w.inner.FillStyled(x, y, width, h, char, style) }
func (w *waveFrame) Fill(char rune, style Style) error                           { return w.inner.Fill(char, style) }
func (w *waveFrame) PrintHyperlink(x, y int, link Hyperlink) error               { return w.inner.PrintHyperlink(x, y, link) }
func (w *waveFrame) PrintHyperlinkFallback(x, y int, link Hyperlink) error       { return w.inner.PrintHyperlinkFallback(x, y, link) }
func (w *waveFrame) GetBounds() image.Rectangle                                  { return w.inner.GetBounds() }
func (w *waveFrame) SubFrame(bounds image.Rectangle) RenderFrame {
	return &waveFrame{
		inner:     w.inner.SubFrame(bounds),
		frame:     w.frame,
		amplitude: w.amplitude,
		frequency: w.frequency,
		speed:     w.speed,
		direction: w.direction,
	}
}
