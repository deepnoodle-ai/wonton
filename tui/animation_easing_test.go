package tui

import (
	"math"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestStandardEasings(t *testing.T) {
	tests := []struct {
		name   string
		easing Easing
		input  float64
		want   float64
	}{
		// EaseLinear
		{"EaseLinear 0", EaseLinear, 0.0, 0.0},
		{"EaseLinear 0.5", EaseLinear, 0.5, 0.5},
		{"EaseLinear 1", EaseLinear, 1.0, 1.0},

		// EaseInQuad
		{"EaseInQuad 0", EaseInQuad, 0.0, 0.0},
		{"EaseInQuad 0.5", EaseInQuad, 0.5, 0.25},
		{"EaseInQuad 1", EaseInQuad, 1.0, 1.0},

		// EaseOutQuad
		{"EaseOutQuad 0", EaseOutQuad, 0.0, 0.0},
		{"EaseOutQuad 0.5", EaseOutQuad, 0.5, 0.75},
		{"EaseOutQuad 1", EaseOutQuad, 1.0, 1.0},

		// EaseInOutQuad
		{"EaseInOutQuad 0", EaseInOutQuad, 0.0, 0.0},
		{"EaseInOutQuad 0.25", EaseInOutQuad, 0.25, 0.125}, // 2 * 0.25^2 = 2 * 0.0625 = 0.125
		{"EaseInOutQuad 0.5", EaseInOutQuad, 0.5, 0.5},
		{"EaseInOutQuad 0.75", EaseInOutQuad, 0.75, 0.875}, // -1 + (4 - 1.5) * 0.75 = -1 + 2.5 * 0.75 = -1 + 1.875 = 0.875
		{"EaseInOutQuad 1", EaseInOutQuad, 1.0, 1.0},

		// EaseInCubic
		{"EaseInCubic 0", EaseInCubic, 0.0, 0.0},
		{"EaseInCubic 0.5", EaseInCubic, 0.5, 0.125},
		{"EaseInCubic 1", EaseInCubic, 1.0, 1.0},

		// EaseOutCubic
		{"EaseOutCubic 0", EaseOutCubic, 0.0, 0.0},
		{"EaseOutCubic 0.5", EaseOutCubic, 0.5, 0.875},
		{"EaseOutCubic 1", EaseOutCubic, 1.0, 1.0},

		// EaseInOutCubic
		{"EaseInOutCubic 0", EaseInOutCubic, 0.0, 0.0},
		{"EaseInOutCubic 0.5", EaseInOutCubic, 0.5, 0.5},
		{"EaseInOutCubic 1", EaseInOutCubic, 1.0, 1.0},

		// EaseInQuart
		{"EaseInQuart 0", EaseInQuart, 0.0, 0.0},
		{"EaseInQuart 0.5", EaseInQuart, 0.5, 0.0625},
		{"EaseInQuart 1", EaseInQuart, 1.0, 1.0},

		// EaseOutQuart
		{"EaseOutQuart 0", EaseOutQuart, 0.0, 0.0},
		{"EaseOutQuart 0.5", EaseOutQuart, 0.5, 0.9375},
		{"EaseOutQuart 1", EaseOutQuart, 1.0, 1.0},

		// EaseInOutQuart
		{"EaseInOutQuart 0", EaseInOutQuart, 0.0, 0.0},
		{"EaseInOutQuart 0.5", EaseInOutQuart, 0.5, 0.5},
		{"EaseInOutQuart 1", EaseInOutQuart, 1.0, 1.0},

		// EaseInSine
		{"EaseInSine 0", EaseInSine, 0.0, 0.0},
		{"EaseInSine 1", EaseInSine, 1.0, 1.0},

		// EaseOutSine
		{"EaseOutSine 0", EaseOutSine, 0.0, 0.0},
		{"EaseOutSine 1", EaseOutSine, 1.0, 1.0},

		// EaseInOutSine
		{"EaseInOutSine 0", EaseInOutSine, 0.0, 0.0},
		{"EaseInOutSine 0.5", EaseInOutSine, 0.5, 0.5},
		{"EaseInOutSine 1", EaseInOutSine, 1.0, 1.0},

		// EaseInExpo
		{"EaseInExpo 0", EaseInExpo, 0.0, 0.0},
		{"EaseInExpo 0.1", EaseInExpo, 0.1, math.Pow(2, -9)},
		{"EaseInExpo 1", EaseInExpo, 1.0, 1.0}, // 2^(10*0) = 1

		// EaseOutExpo
		{"EaseOutExpo 0", EaseOutExpo, 0.0, 0.0},
		{"EaseOutExpo 0.9", EaseOutExpo, 0.9, 1 - math.Pow(2, -9)},
		{"EaseOutExpo 1", EaseOutExpo, 1.0, 1.0},

		// EaseInOutExpo
		{"EaseInOutExpo 0", EaseInOutExpo, 0.0, 0.0},
		{"EaseInOutExpo 0.5", EaseInOutExpo, 0.5, 0.5},
		{"EaseInOutExpo 1", EaseInOutExpo, 1.0, 1.0},

		// EaseInCirc
		{"EaseInCirc 0", EaseInCirc, 0.0, 0.0},
		{"EaseInCirc 1", EaseInCirc, 1.0, 1.0},

		// EaseOutCirc
		{"EaseOutCirc 0", EaseOutCirc, 0.0, 0.0},
		{"EaseOutCirc 1", EaseOutCirc, 1.0, 1.0},

		// EaseInOutCirc
		{"EaseInOutCirc 0", EaseInOutCirc, 0.0, 0.0},
		{"EaseInOutCirc 0.5", EaseInOutCirc, 0.5, 0.5},
		{"EaseInOutCirc 1", EaseInOutCirc, 1.0, 1.0},

		// EaseInBack
		{"EaseInBack 0", EaseInBack, 0.0, 0.0},
		{"EaseInBack 1", EaseInBack, 1.0, 1.0},

		// EaseOutBack
		{"EaseOutBack 0", EaseOutBack, 0.0, 0.0},
		{"EaseOutBack 1", EaseOutBack, 1.0, 1.0},

		// EaseInOutBack
		{"EaseInOutBack 0", EaseInOutBack, 0.0, 0.0},
		{"EaseInOutBack 0.5", EaseInOutBack, 0.5, 0.5},
		{"EaseInOutBack 1", EaseInOutBack, 1.0, 1.0},

		// EaseInElastic
		{"EaseInElastic 0", EaseInElastic, 0.0, 0.0},
		{"EaseInElastic 1", EaseInElastic, 1.0, 1.0},

		// EaseOutElastic
		{"EaseOutElastic 0", EaseOutElastic, 0.0, 0.0},
		{"EaseOutElastic 1", EaseOutElastic, 1.0, 1.0},

		// EaseInOutElastic
		{"EaseInOutElastic 0", EaseInOutElastic, 0.0, 0.0},
		{"EaseInOutElastic 0.5", EaseInOutElastic, 0.5, 0.5},
		{"EaseInOutElastic 1", EaseInOutElastic, 1.0, 1.0},

		// EaseInBounce
		{"EaseInBounce 0", EaseInBounce, 0.0, 0.0},
		{"EaseInBounce 1", EaseInBounce, 1.0, 1.0},

		// EaseOutBounce
		{"EaseOutBounce 0", EaseOutBounce, 0.0, 0.0},
		{"EaseOutBounce 1", EaseOutBounce, 1.0, 1.0},

		// EaseInOutBounce
		{"EaseInOutBounce 0", EaseInOutBounce, 0.0, 0.0},
		{"EaseInOutBounce 0.5", EaseInOutBounce, 0.5, 0.5},
		{"EaseInOutBounce 1", EaseInOutBounce, 1.0, 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.easing(tt.input)
			assert.True(t, math.Abs(got-tt.want) <= 1e-6, "got %v, want %v", got, tt.want)
		})
	}
}

func TestEasingBuilders(t *testing.T) {
	t.Run("EaseChain", func(t *testing.T) {
		chain := EaseChain(EaseLinear, EaseLinear)
		assert.Equal(t, 0.0, chain(0.0))

		// EaseChain logic:
		// n=2. t=0.5. section = 1. sectionT = 0.5*2 - 1 = 0.
		// return (1 + EaseLinear(0)) / 2 = 0.5
		assert.Equal(t, 0.5, chain(0.5))

		assert.Equal(t, 1.0, chain(1.0))

		// Test empty chain fallback
		empty := EaseChain()
		assert.Equal(t, 0.5, empty(0.5))
	})

	t.Run("EaseReverse", func(t *testing.T) {
		rev := EaseReverse(EaseLinear)
		assert.Equal(t, 0.25, rev(0.25))

		// Reverse EaseInQuad (t^2) -> 1 - (1-t)^2 = 1 - (1 - 2t + t^2) = 2t - t^2 = EaseOutQuad
		revQuad := EaseReverse(EaseInQuad)
		assert.Equal(t, 0.75, revQuad(0.5))
	})

	t.Run("EaseMirror", func(t *testing.T) {
		mirror := EaseMirror(EaseLinear)
		assert.Equal(t, 0.0, mirror(0.0))
		assert.Equal(t, 1.0, mirror(0.5)) // Peak
		assert.Equal(t, 0.0, mirror(1.0))
		assert.Equal(t, 0.5, mirror(0.25))
		assert.Equal(t, 0.5, mirror(0.75))
	})

	t.Run("EaseScale", func(t *testing.T) {
		scale := EaseScale(EaseLinear, 2.0)
		assert.Equal(t, 1.0, scale(0.5))
	})

	t.Run("EaseClamp", func(t *testing.T) {
		overshoot := func(t float64) float64 { return 1.5 }
		undershoot := func(t float64) float64 { return -0.5 }

		clampedOver := EaseClamp(overshoot)
		assert.Equal(t, 1.0, clampedOver(0.5))

		clampedUnder := EaseClamp(undershoot)
		assert.Equal(t, 0.0, clampedUnder(0.5))

		normal := EaseClamp(EaseLinear)
		assert.Equal(t, 0.5, normal(0.5))
	})
}
