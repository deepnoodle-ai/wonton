package color_test

import (
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
	"github.com/deepnoodle-ai/wonton/color"
)

func TestHex_Valid(t *testing.T) {
	tests := []struct {
		in       string
		expected color.RGB
	}{
		{"#ff8800", color.NewRGB(255, 136, 0)},
		{"ff8800", color.NewRGB(255, 136, 0)},
		{"#FF8800", color.NewRGB(255, 136, 0)},
		{"#f80", color.NewRGB(255, 136, 0)},
		{"f80", color.NewRGB(255, 136, 0)},
		{"#000000", color.NewRGB(0, 0, 0)},
		{"#ffffff", color.NewRGB(255, 255, 255)},
		{"#5f87ff", color.NewRGB(95, 135, 255)},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			rgb, err := color.Hex(tt.in)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, rgb)
		})
	}
}

func TestHex_Invalid(t *testing.T) {
	for _, in := range []string{"", "#", "#ff", "#ffff", "#fffff00", "#gggggg", "#ff 800", "+f8800f"} {
		t.Run(in, func(t *testing.T) {
			_, err := color.Hex(in)
			assert.Error(t, err)
		})
	}
}

func TestMustHex(t *testing.T) {
	assert.Equal(t, color.NewRGB(255, 136, 0), color.MustHex("#ff8800"))
	assert.Panics(t, func() { color.MustHex("nope") })
}

func TestRGB_Hex(t *testing.T) {
	assert.Equal(t, "#ff8800", color.NewRGB(255, 136, 0).Hex())
	assert.Equal(t, "#000000", color.NewRGB(0, 0, 0).Hex())
	assert.Equal(t, "#0a0b0c", color.NewRGB(10, 11, 12).Hex())
}

func TestHex_RoundTrip(t *testing.T) {
	original := color.NewRGB(95, 135, 255)
	parsed, err := color.Hex(original.Hex())
	assert.NoError(t, err)
	assert.Equal(t, original, parsed)
}

func TestRGB_ForegroundSeq(t *testing.T) {
	rgb := color.NewRGB(255, 0, 127)
	assert.Equal(t, "\033[38;2;255;0;127m", rgb.ForegroundSeq())
}

func TestRGB_BackgroundSeq(t *testing.T) {
	rgb := color.NewRGB(127, 0, 255)
	assert.Equal(t, "\033[48;2;127;0;255m", rgb.BackgroundSeq())
}

func TestRGB_SeqsIgnoreEnabled(t *testing.T) {
	disableColor(t)
	rgb := color.NewRGB(255, 0, 127)
	assert.Equal(t, "\033[38;2;255;0;127m", rgb.ForegroundSeq())
	assert.Equal(t, "\033[48;2;127;0;255m", color.NewRGB(127, 0, 255).BackgroundSeq())
}

func TestRGB_Apply(t *testing.T) {
	forceColor(t)
	rgb := color.NewRGB(255, 128, 0)
	assert.Equal(t, "\033[38;2;255;128;0mTest\033[0m", rgb.Apply("Test"))
}

func TestRGB_ApplyBg(t *testing.T) {
	forceColor(t)
	rgb := color.NewRGB(0, 128, 255)
	assert.Equal(t, "\033[48;2;0;128;255mTest\033[0m", rgb.ApplyBg("Test"))
}

func TestRGB_Apply_RespectsEnabled(t *testing.T) {
	disableColor(t)
	rgb := color.NewRGB(255, 128, 0)
	assert.Equal(t, "Test", rgb.Apply("Test"))
	assert.Equal(t, "Test", rgb.ApplyBg("Test"))
	assert.Equal(t, "x: 1", rgb.Sprintf("x: %d", 1))
	assert.Equal(t, "ab", rgb.Sprint("a", "b"))
}

func TestRGB_Sprintf(t *testing.T) {
	forceColor(t)
	rgb := color.NewRGB(255, 128, 0)
	assert.Equal(t, "\033[38;2;255;128;0mx: 1\033[0m", rgb.Sprintf("x: %d", 1))
}

func TestRGB_Lerp(t *testing.T) {
	start := color.NewRGB(255, 0, 0)
	end := color.NewRGB(0, 0, 255)

	assert.Equal(t, start, start.Lerp(end, 0))
	assert.Equal(t, end, start.Lerp(end, 1))
	assert.Equal(t, color.NewRGB(128, 0, 128), start.Lerp(end, 0.5))

	// t is clamped to [0, 1]
	assert.Equal(t, start, start.Lerp(end, -1))
	assert.Equal(t, end, start.Lerp(end, 2))
}
