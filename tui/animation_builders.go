package tui

// AnimationBuilder provides a fluent API for composing complex animations.
type AnimationBuilder struct {
	animations []*Animation
	sequences  []*AnimationSequence
	groups     []*AnimationGroup
	mode       AnimationMode
}

// AnimationMode determines how multiple animations are combined.
type AnimationMode int

const (
	AnimationModeSequence AnimationMode = iota // Play one after another
	AnimationModeParallel                      // Play simultaneously
	AnimationModeStagger                       // Start each with a delay
)

// NewAnimationBuilder creates a new animation builder.
func NewAnimationBuilder() *AnimationBuilder {
	return &AnimationBuilder{
		animations: make([]*Animation, 0),
		mode:       AnimationModeSequence,
	}
}

// Add adds an animation to the builder.
func (ab *AnimationBuilder) Add(duration uint64, easing Easing) *AnimationBuilder {
	anim := NewAnimation(duration).WithEasing(easing)
	ab.animations = append(ab.animations, anim)
	return ab
}

// AddAnimation adds a pre-configured animation to the builder.
func (ab *AnimationBuilder) AddAnimation(anim *Animation) *AnimationBuilder {
	ab.animations = append(ab.animations, anim)
	return ab
}

// Sequence sets the mode to sequence (animations play one after another).
func (ab *AnimationBuilder) Sequence() *AnimationBuilder {
	ab.mode = AnimationModeSequence
	return ab
}

// Parallel sets the mode to parallel (animations play simultaneously).
func (ab *AnimationBuilder) Parallel() *AnimationBuilder {
	ab.mode = AnimationModeParallel
	return ab
}

// Stagger sets the mode to staggered (each animation starts with a delay).
func (ab *AnimationBuilder) Stagger() *AnimationBuilder {
	ab.mode = AnimationModeStagger
	return ab
}

// Build constructs the final animation based on the mode.
func (ab *AnimationBuilder) Build() interface{} {
	switch ab.mode {
	case AnimationModeSequence:
		return NewAnimationSequence(ab.animations...)
	case AnimationModeParallel:
		return NewAnimationGroup(ab.animations...)
	case AnimationModeStagger:
		return ab.buildStaggered()
	default:
		return NewAnimationSequence(ab.animations...)
	}
}

func (ab *AnimationBuilder) buildStaggered() *AnimationSequence {
	// Create delayed versions of each animation
	staggerDelay := uint64(10) // frames between each start
	delayedAnims := make([]*Animation, 0, len(ab.animations))

	for i, anim := range ab.animations {
		delay := uint64(i) * staggerDelay
		delayedAnim := NewAnimation(delay + anim.duration).WithEasing(anim.easing)
		delayedAnims = append(delayedAnims, delayedAnim)
	}

	return NewAnimationSequence(delayedAnims...)
}

// ChainedAnimation allows chaining different animation types together.
type ChainedAnimation struct {
	steps []AnimationStep
}

// AnimationStep represents one step in a chained animation.
type AnimationStep struct {
	textAnim   TextAnimation
	borderAnim BorderAnimation
	transform  func(View) View
	duration   uint64
}

// NewChainedAnimation creates a new chained animation.
func NewChainedAnimation() *ChainedAnimation {
	return &ChainedAnimation{
		steps: make([]AnimationStep, 0),
	}
}

// AddTextAnimation adds a text animation step.
func (ca *ChainedAnimation) AddTextAnimation(anim TextAnimation, duration uint64) *ChainedAnimation {
	ca.steps = append(ca.steps, AnimationStep{
		textAnim: anim,
		duration: duration,
	})
	return ca
}

// AddBorderAnimation adds a border animation step.
func (ca *ChainedAnimation) AddBorderAnimation(anim BorderAnimation, duration uint64) *ChainedAnimation {
	ca.steps = append(ca.steps, AnimationStep{
		borderAnim: anim,
		duration:   duration,
	})
	return ca
}

// AddTransform adds a view transformation step.
func (ca *ChainedAnimation) AddTransform(transform func(View) View, duration uint64) *ChainedAnimation {
	ca.steps = append(ca.steps, AnimationStep{
		transform: transform,
		duration:  duration,
	})
	return ca
}

// GetCurrentStep returns the current animation step based on frame.
func (ca *ChainedAnimation) GetCurrentStep(frame uint64) *AnimationStep {
	if len(ca.steps) == 0 {
		return nil
	}

	elapsed := frame
	for _, step := range ca.steps {
		if elapsed < step.duration {
			return &step
		}
		elapsed -= step.duration
	}

	// Return last step if we've exceeded total duration
	return &ca.steps[len(ca.steps)-1]
}

// Preset animation combinations

// PresetPulsingBorder creates a pulsing border animation preset.
func PresetPulsingBorder(color RGB, speed int) BorderAnimation {
	return &PulseBorderAnimation{
		Speed:         speed,
		Color:         color,
		MinBrightness: 0.4,
		MaxBrightness: 1.0,
		Easing:        EaseInOutSine,
	}
}

// PresetRainbowBorder creates a rainbow border animation preset.
func PresetRainbowBorder(speed int, reversed bool) BorderAnimation {
	return &RainbowBorderAnimation{
		Speed:    speed,
		Reversed: reversed,
	}
}

// PresetMarqueeBorder creates a marquee border animation preset.
func PresetMarqueeBorder(onColor, offColor RGB, speed, segmentLength int) BorderAnimation {
	return &MarqueeBorderAnimation{
		Speed:         speed,
		OnColor:       onColor,
		OffColor:      offColor,
		SegmentLength: segmentLength,
	}
}

// PresetFireBorder creates a fire border animation preset.
func PresetFireBorder(speed int) BorderAnimation {
	return &FireBorderAnimation{
		Speed: speed,
		CoolColors: []RGB{
			NewRGB(200, 50, 0),
			NewRGB(180, 40, 0),
		},
		HotColors: []RGB{
			NewRGB(255, 150, 0),
			NewRGB(255, 200, 100),
		},
	}
}

// PresetFadeIn creates a fade-in animation.
func PresetFadeIn(duration uint64) *Animation {
	return NewAnimation(duration).WithEasing(EaseInQuad)
}

// PresetFadeOut creates a fade-out animation.
func PresetFadeOut(duration uint64) *Animation {
	return NewAnimation(duration).WithEasing(EaseOutQuad)
}

// PresetPulse creates a pulsing animation.
func PresetPulse(duration uint64) *Animation {
	return NewAnimation(duration).
		WithEasing(EaseInOutSine).
		WithLoop(true).
		WithPingPong(true)
}

// PresetBounce creates a bouncing animation.
func PresetBounce(duration uint64) *Animation {
	return NewAnimation(duration).
		WithEasing(EaseOutBounce)
}

// PresetElastic creates an elastic animation.
func PresetElastic(duration uint64) *Animation {
	return NewAnimation(duration).
		WithEasing(EaseOutElastic)
}

// PresetSlideIn creates a slide-in animation.
func PresetSlideIn(duration uint64) *Animation {
	return NewAnimation(duration).
		WithEasing(EaseOutCubic)
}

// PresetAttention creates an attention-grabbing animation.
func PresetAttention(duration uint64) *Animation {
	return NewAnimation(duration).
		WithEasing(EaseInOutBack).
		WithLoop(true).
		WithPingPong(true)
}

// AnimationPresets provides common animation presets.
var AnimationPresets = struct {
	FadeIn    func(uint64) *Animation
	FadeOut   func(uint64) *Animation
	Pulse     func(uint64) *Animation
	Bounce    func(uint64) *Animation
	Elastic   func(uint64) *Animation
	SlideIn   func(uint64) *Animation
	Attention func(uint64) *Animation
}{
	FadeIn:    PresetFadeIn,
	FadeOut:   PresetFadeOut,
	Pulse:     PresetPulse,
	Bounce:    PresetBounce,
	Elastic:   PresetElastic,
	SlideIn:   PresetSlideIn,
	Attention: PresetAttention,
}

// BorderAnimationPresets provides common border animation presets.
var BorderAnimationPresets = struct {
	Pulsing func(RGB, int) BorderAnimation
	Rainbow func(int, bool) BorderAnimation
	Marquee func(RGB, RGB, int, int) BorderAnimation
	Fire    func(int) BorderAnimation
}{
	Pulsing: PresetPulsingBorder,
	Rainbow: PresetRainbowBorder,
	Marquee: PresetMarqueeBorder,
	Fire:    PresetFireBorder,
}
