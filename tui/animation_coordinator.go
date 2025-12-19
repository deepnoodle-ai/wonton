package tui

import "sync"

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
