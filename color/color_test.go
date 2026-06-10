package color_test

import (
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
	"github.com/deepnoodle-ai/wonton/color"
)

// forceColor enables color output for the duration of the test, restoring
// the prior state on exit. Tests run with piped output, so Enabled would
// otherwise default to false.
func forceColor(t *testing.T) {
	t.Helper()
	original := color.Enabled
	color.Enabled = true
	t.Cleanup(func() { color.Enabled = original })
}

// disableColor disables color output for the duration of the test.
func disableColor(t *testing.T) {
	t.Helper()
	original := color.Enabled
	color.Enabled = false
	t.Cleanup(func() { color.Enabled = original })
}

func TestColor_ForegroundCode_AllColors(t *testing.T) {
	tests := []struct {
		c        color.Color
		expected string
	}{
		{color.Black, "30"},
		{color.Red, "31"},
		{color.Green, "32"},
		{color.Yellow, "33"},
		{color.Blue, "34"},
		{color.Magenta, "35"},
		{color.Cyan, "36"},
		{color.White, "37"},
		{color.BrightBlack, "90"},
		{color.BrightRed, "91"},
		{color.BrightGreen, "92"},
		{color.BrightYellow, "93"},
		{color.BrightBlue, "94"},
		{color.BrightMagenta, "95"},
		{color.BrightCyan, "96"},
		{color.BrightWhite, "97"},
		{color.Default, "39"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.c.ForegroundCode())
		})
	}
}

func TestColor_BackgroundCode_AllColors(t *testing.T) {
	tests := []struct {
		c        color.Color
		expected string
	}{
		{color.Black, "40"},
		{color.Red, "41"},
		{color.Green, "42"},
		{color.Yellow, "43"},
		{color.Blue, "44"},
		{color.Magenta, "45"},
		{color.Cyan, "46"},
		{color.White, "47"},
		{color.BrightBlack, "100"},
		{color.BrightRed, "101"},
		{color.BrightGreen, "102"},
		{color.BrightYellow, "103"},
		{color.BrightBlue, "104"},
		{color.BrightMagenta, "105"},
		{color.BrightCyan, "106"},
		{color.BrightWhite, "107"},
		{color.Default, "49"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.c.BackgroundCode())
		})
	}
}

func TestColor_ExtendedPalette(t *testing.T) {
	tests := []struct {
		n          uint8
		expectedFg string
		expectedBg string
	}{
		{16, "38;5;16", "48;5;16"},
		{196, "38;5;196", "48;5;196"}, // bright red in 256 palette
		{232, "38;5;232", "48;5;232"}, // grayscale start
		{255, "38;5;255", "48;5;255"}, // grayscale end
	}

	for _, tt := range tests {
		c := color.Palette(tt.n)
		assert.Equal(t, tt.expectedFg, c.ForegroundCode())
		assert.Equal(t, tt.expectedBg, c.BackgroundCode())
	}
}

func TestColor_ForegroundSeq(t *testing.T) {
	assert.Equal(t, "\033[31m", color.Red.ForegroundSeq())
	assert.Equal(t, "\033[92m", color.BrightGreen.ForegroundSeq())
	assert.Equal(t, "", color.Default.ForegroundSeq())
}

func TestColor_BackgroundSeq(t *testing.T) {
	assert.Equal(t, "\033[41m", color.Red.BackgroundSeq())
	assert.Equal(t, "\033[102m", color.BrightGreen.BackgroundSeq())
	assert.Equal(t, "", color.Default.BackgroundSeq())
}

func TestColor_ForegroundSeqDim(t *testing.T) {
	assert.Equal(t, "\033[2;31m", color.Red.ForegroundSeqDim())
	assert.Equal(t, "\033[2m", color.Default.ForegroundSeqDim())
}

func TestColor_ForegroundSeqBold(t *testing.T) {
	assert.Equal(t, "\033[1;31m", color.Red.ForegroundSeqBold())
	assert.Equal(t, "\033[1m", color.Default.ForegroundSeqBold())
}

func TestColor_SeqsIgnoreEnabled(t *testing.T) {
	disableColor(t)
	assert.Equal(t, "\033[31m", color.Red.ForegroundSeq())
	assert.Equal(t, "\033[41m", color.Red.BackgroundSeq())
}

func TestColor_Apply(t *testing.T) {
	forceColor(t)
	assert.Equal(t, "\033[31mError\033[0m", color.Red.Apply("Error"))
	assert.Equal(t, "plain", color.Default.Apply("plain"))
}

func TestColor_ApplyBg(t *testing.T) {
	forceColor(t)
	assert.Equal(t, "\033[41m ERROR \033[0m", color.Red.ApplyBg(" ERROR "))
	assert.Equal(t, "plain", color.Default.ApplyBg("plain"))
}

func TestColor_ApplyDim(t *testing.T) {
	forceColor(t)
	assert.Equal(t, "\033[2;37mfaint\033[0m", color.White.ApplyDim("faint"))
	assert.Equal(t, "\033[2mfaint\033[0m", color.Default.ApplyDim("faint"))
}

func TestColor_ApplyBold(t *testing.T) {
	forceColor(t)
	assert.Equal(t, "\033[1;31mFAILED\033[0m", color.Red.ApplyBold("FAILED"))
	assert.Equal(t, "\033[1mloud\033[0m", color.Default.ApplyBold("loud"))
}

func TestColor_Apply_RespectsEnabled(t *testing.T) {
	disableColor(t)
	assert.Equal(t, "Error", color.Red.Apply("Error"))
	assert.Equal(t, " ERROR ", color.Red.ApplyBg(" ERROR "))
	assert.Equal(t, "faint", color.White.ApplyDim("faint"))
	assert.Equal(t, "FAILED", color.Red.ApplyBold("FAILED"))
	assert.Equal(t, "x: 1", color.Red.Sprintf("x: %d", 1))
	assert.Equal(t, "ab", color.Red.Sprint("a", "b"))
}

func TestColor_Sprintf(t *testing.T) {
	forceColor(t)
	assert.Equal(t, "\033[31mFound 3 errors\033[0m", color.Red.Sprintf("Found %d errors", 3))
}

func TestColor_Sprint(t *testing.T) {
	forceColor(t)
	assert.Equal(t, "\033[32mok: 2 items\033[0m", color.Green.Sprint("ok: ", 2, " items"))
}

func TestApplyBold(t *testing.T) {
	forceColor(t)
	assert.Equal(t, "\033[1mImportant\033[0m", color.ApplyBold("Important"))

	disableColor(t)
	assert.Equal(t, "Important", color.ApplyBold("Important"))
}

func TestApplyDim(t *testing.T) {
	forceColor(t)
	assert.Equal(t, "\033[2maside\033[0m", color.ApplyDim("aside"))

	disableColor(t)
	assert.Equal(t, "aside", color.ApplyDim("aside"))
}
