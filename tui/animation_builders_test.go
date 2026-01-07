package tui

import (
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestAnimationBuilder(t *testing.T) {
	t.Run("Sequence", func(t *testing.T) {
		builder := NewAnimationBuilder()
		res := builder.
			Sequence().
			Add(10, EaseLinear).
			Add(10, EaseInQuad).
			Build()

		seq, ok := res.(*AnimationSequence)
		assert.True(t, ok, "Sequence build result should be *AnimationSequence")
		assert.NotNil(t, seq, "Sequence result should not be nil")
	})

	t.Run("Parallel", func(t *testing.T) {
		builder := NewAnimationBuilder()
		res := builder.
			Parallel().
			Add(10, EaseLinear).
			Add(10, EaseInQuad).
			Build()

		group, ok := res.(*AnimationGroup)
		assert.True(t, ok, "Parallel build result should be *AnimationGroup")
		assert.NotNil(t, group, "Parallel result should not be nil")
	})

	t.Run("Stagger", func(t *testing.T) {
		builder := NewAnimationBuilder()
		res := builder.
			Stagger().
			Add(10, EaseLinear).
			Add(10, EaseInQuad).
			Build()

		seq, ok := res.(*AnimationSequence)
		assert.True(t, ok, "Stagger build result should be *AnimationSequence")
		assert.NotNil(t, seq, "Stagger result should not be nil")
	})

	t.Run("AddAnimation", func(t *testing.T) {
		anim := NewAnimation(5)
		builder := NewAnimationBuilder()
		res := builder.AddAnimation(anim).Build()

		seq, ok := res.(*AnimationSequence)
		assert.True(t, ok, "Expected *AnimationSequence")
		assert.NotNil(t, seq, "Result should not be nil")
	})
}

// Mock implementations for ChainedAnimation test
type mockTextAnim struct{}

func (m *mockTextAnim) GetStyle(frame uint64, charIndex int, totalChars int) Style { return NewStyle() }

type mockBorderAnim struct{}

func (m *mockBorderAnim) GetBorderStyle(frame uint64, borderPart BorderPart, position int, length int) Style {
	return NewStyle()
}

func TestChainedAnimation(t *testing.T) {
	chain := NewChainedAnimation()

	// Step 1: Text animation for 10 frames
	chain.AddTextAnimation(&mockTextAnim{}, 10)

	// Step 2: Border animation for 20 frames
	chain.AddBorderAnimation(&mockBorderAnim{}, 20)

	// Step 3: Transform for 5 frames
	transform := func(v View) View { return v }
	chain.AddTransform(transform, 5)

	// Total duration: 35 frames

	// Frame 5 -> Should be Step 1
	step1 := chain.GetCurrentStep(5)
	assert.NotNil(t, step1, "Frame 5 should return step")
	assert.NotNil(t, step1.textAnim, "Frame 5 should have text animation")
	assert.Nil(t, step1.borderAnim, "Frame 5 should not have border animation")

	// Frame 15 -> Should be Step 2 (10 + 5)
	step2 := chain.GetCurrentStep(15)
	assert.NotNil(t, step2, "Frame 15 should return step")
	assert.NotNil(t, step2.borderAnim, "Frame 15 should have border animation")

	// Frame 32 -> Should be Step 3 (10 + 20 + 2)
	step3 := chain.GetCurrentStep(32)
	assert.NotNil(t, step3, "Frame 32 should return step")
	assert.NotNil(t, step3.transform, "Frame 32 should have transform")

	// Frame 40 -> Should return last step (Step 3)
	step4 := chain.GetCurrentStep(40)
	assert.NotNil(t, step4, "Frame 40 should return last step")
	assert.NotNil(t, step4.transform, "Frame 40 should have transform")
}

func TestAnimationPresets(t *testing.T) {
	t.Run("Standard Presets", func(t *testing.T) {
		presets := []func(uint64) *Animation{
			PresetFadeIn,
			PresetFadeOut,
			PresetPulse,
			PresetBounce,
			PresetElastic,
			PresetSlideIn,
			PresetAttention,
		}

		for _, p := range presets {
			anim := p(10)
			assert.NotNil(t, anim, "Preset returned nil")
		}
	})

	t.Run("PresetPulse Looping", func(t *testing.T) {
		anim := PresetPulse(10)
		anim.Start(0)
		anim.Update(15) // Past duration
		assert.True(t, anim.IsRunning(), "PresetPulse should loop")
	})

	t.Run("PresetFadeIn NoLoop", func(t *testing.T) {
		anim := PresetFadeIn(10)
		anim.Start(0)
		anim.Update(11) // Past duration + 1
		assert.False(t, anim.IsRunning(), "PresetFadeIn should not loop")
	})
}

func TestBorderAnimationPresets(t *testing.T) {
	c1 := NewRGB(255, 0, 0)
	c2 := NewRGB(0, 0, 255)

	t.Run("Pulsing", func(t *testing.T) {
		anim := PresetPulsingBorder(c1, 10)
		assert.NotNil(t, anim, "PresetPulsingBorder returned nil")
	})

	t.Run("Rainbow", func(t *testing.T) {
		anim := PresetRainbowBorder(10, false)
		assert.NotNil(t, anim, "PresetRainbowBorder returned nil")
	})

	t.Run("Marquee", func(t *testing.T) {
		anim := PresetMarqueeBorder(c1, c2, 10, 5)
		assert.NotNil(t, anim, "PresetMarqueeBorder returned nil")
	})

	t.Run("Fire", func(t *testing.T) {
		anim := PresetFireBorder(10)
		assert.NotNil(t, anim, "PresetFireBorder returned nil")
	})

	// Test the Presets struct wrappers
	assert.NotNil(t, BorderAnimationPresets.Pulsing(c1, 10), "BorderAnimationPresets.Pulsing returned nil")
	assert.NotNil(t, AnimationPresets.FadeIn(10), "AnimationPresets.FadeIn returned nil")
}
