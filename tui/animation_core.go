package tui

import "math"

// AnimationController coordinates multiple animations and provides timing control.
// It can be used to create synchronized animations across different views.
type AnimationController struct {
	animations map[string]*Animation
	globalTime uint64
}

// NewAnimationController creates a new animation controller.
func NewAnimationController() *AnimationController {
	return &AnimationController{
		animations: make(map[string]*Animation),
	}
}

// Update updates all managed animations.
func (ac *AnimationController) Update(frame uint64) {
	ac.globalTime = frame
	for _, anim := range ac.animations {
		anim.Update(frame)
	}
}

// Register adds an animation to the controller with a unique name.
func (ac *AnimationController) Register(name string, anim *Animation) {
	ac.animations[name] = anim
}

// Get retrieves a registered animation by name.
func (ac *AnimationController) Get(name string) *Animation {
	return ac.animations[name]
}

// GlobalTime returns the current global frame counter.
func (ac *AnimationController) GlobalTime() uint64 {
	return ac.globalTime
}

// Animation represents a time-based animation with duration, easing, and looping.
type Animation struct {
	duration   uint64  // Duration in frames
	easing     Easing  // Easing function
	loop       bool    // Whether to loop
	pingPong   bool    // Whether to reverse on loop
	startFrame uint64  // Frame when animation started
	value      float64 // Current animated value [0,1]
	running    bool
}

// NewAnimation creates a new animation with the given duration in frames.
func NewAnimation(duration uint64) *Animation {
	return &Animation{
		duration: duration,
		easing:   EaseLinear,
		loop:     false,
		pingPong: false,
		running:  false,
	}
}

// WithEasing sets the easing function for the animation.
func (a *Animation) WithEasing(easing Easing) *Animation {
	a.easing = easing
	return a
}

// WithLoop enables looping for the animation.
func (a *Animation) WithLoop(loop bool) *Animation {
	a.loop = loop
	return a
}

// WithPingPong enables ping-pong looping (reverses direction on loop).
func (a *Animation) WithPingPong(pingPong bool) *Animation {
	a.pingPong = pingPong
	a.loop = true // ping-pong requires looping
	return a
}

// Start starts or restarts the animation.
func (a *Animation) Start(frame uint64) {
	a.startFrame = frame
	a.running = true
}

// Stop stops the animation.
func (a *Animation) Stop() {
	a.running = false
}

// Update updates the animation value based on the current frame.
func (a *Animation) Update(frame uint64) {
	if !a.running {
		return
	}

	elapsed := frame - a.startFrame
	if a.duration == 0 {
		a.value = 1.0
		return
	}

	// Calculate progress [0,1]
	progress := float64(elapsed) / float64(a.duration)

	if a.loop {
		// Loop the progress
		progress = math.Mod(progress, 1.0)

		if a.pingPong {
			// Ping-pong: 0->1->0->1...
			cycle := int(float64(elapsed) / float64(a.duration))
			if cycle%2 == 1 {
				progress = 1.0 - progress
			}
		}
	} else {
		// Clamp to [0,1] if not looping
		if progress > 1.0 {
			progress = 1.0
			a.running = false
		}
	}

	// Apply easing
	a.value = a.easing(progress)
}

// Value returns the current animation value [0,1].
func (a *Animation) Value() float64 {
	return a.value
}

// IsRunning returns whether the animation is currently running.
func (a *Animation) IsRunning() bool {
	return a.running
}

// Lerp performs linear interpolation between two values using the animation value.
func (a *Animation) Lerp(from, to float64) float64 {
	return from + (to-from)*a.value
}

// LerpRGB performs linear interpolation between two RGB colors.
func (a *Animation) LerpRGB(from, to RGB) RGB {
	return RGB{
		R: uint8(a.Lerp(float64(from.R), float64(to.R))),
		G: uint8(a.Lerp(float64(from.G), float64(to.G))),
		B: uint8(a.Lerp(float64(from.B), float64(to.B))),
	}
}

// AnimationSequence represents a sequence of animations that play one after another.
type AnimationSequence struct {
	animations   []*Animation
	currentIndex int
	startFrame   uint64
}

// NewAnimationSequence creates a new animation sequence.
func NewAnimationSequence(animations ...*Animation) *AnimationSequence {
	return &AnimationSequence{
		animations:   animations,
		currentIndex: 0,
	}
}

// Update updates the current animation in the sequence.
func (as *AnimationSequence) Update(frame uint64) {
	if as.currentIndex >= len(as.animations) {
		return
	}

	current := as.animations[as.currentIndex]
	if !current.IsRunning() {
		current.Start(frame)
	}

	current.Update(frame)

	// Move to next animation when current finishes
	if !current.IsRunning() {
		as.currentIndex++
		if as.currentIndex < len(as.animations) {
			as.startFrame = frame
		}
	}
}

// Value returns the current animation value.
func (as *AnimationSequence) Value() float64 {
	if as.currentIndex >= len(as.animations) {
		return 1.0
	}
	return as.animations[as.currentIndex].Value()
}

// IsComplete returns whether all animations in the sequence have finished.
func (as *AnimationSequence) IsComplete() bool {
	return as.currentIndex >= len(as.animations)
}

// Reset resets the sequence to the beginning.
func (as *AnimationSequence) Reset() {
	as.currentIndex = 0
	for _, anim := range as.animations {
		anim.Stop()
	}
}

// AnimationGroup represents multiple animations that play simultaneously.
type AnimationGroup struct {
	animations []*Animation
	combiner   func(values []float64) float64 // How to combine values
}

// NewAnimationGroup creates a new animation group.
func NewAnimationGroup(animations ...*Animation) *AnimationGroup {
	return &AnimationGroup{
		animations: animations,
		combiner:   animationGroupAverage, // Default to average
	}
}

func animationGroupAverage(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func animationGroupMax(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	max := values[0]
	for _, v := range values {
		if v > max {
			max = v
		}
	}
	return max
}

func animationGroupMin(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	min := values[0]
	for _, v := range values {
		if v < min {
			min = v
		}
	}
	return min
}

// WithCombiner sets how to combine values from multiple animations.
func (ag *AnimationGroup) WithCombiner(combiner func(values []float64) float64) *AnimationGroup {
	ag.combiner = combiner
	return ag
}

// CombineAverage averages all animation values.
func (ag *AnimationGroup) CombineAverage() *AnimationGroup {
	ag.combiner = animationGroupAverage
	return ag
}

// CombineMax takes the maximum of all animation values.
func (ag *AnimationGroup) CombineMax() *AnimationGroup {
	ag.combiner = animationGroupMax
	return ag
}

// CombineMin takes the minimum of all animation values.
func (ag *AnimationGroup) CombineMin() *AnimationGroup {
	ag.combiner = animationGroupMin
	return ag
}

// Start starts all animations in the group.
func (ag *AnimationGroup) Start(frame uint64) {
	for _, anim := range ag.animations {
		anim.Start(frame)
	}
}

// Update updates all animations in the group.
func (ag *AnimationGroup) Update(frame uint64) {
	for _, anim := range ag.animations {
		anim.Update(frame)
	}
}

// Value returns the combined value from all animations.
func (ag *AnimationGroup) Value() float64 {
	values := make([]float64, len(ag.animations))
	for i, anim := range ag.animations {
		values[i] = anim.Value()
	}
	return ag.combiner(values)
}
