package tui

import (
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestProgress_Basic(t *testing.T) {
	p := Progress(50, 100)
	assert.Equal(t, 50, p.current)
	assert.Equal(t, 100, p.total)
	assert.Equal(t, 20, p.width)
	assert.Equal(t, '█', p.filledChar)
	assert.Equal(t, '░', p.emptyChar)
	assert.Equal(t, true, p.showPercent)
}

func TestProgress_Width(t *testing.T) {
	p := Progress(50, 100).Width(30)
	assert.Equal(t, 30, p.width)
}

func TestProgress_FilledChar(t *testing.T) {
	p := Progress(50, 100).FilledChar('▓')
	assert.Equal(t, '▓', p.filledChar)
}

func TestProgress_EmptyChar(t *testing.T) {
	p := Progress(50, 100).EmptyChar('·')
	assert.Equal(t, '·', p.emptyChar)
	assert.Equal(t, "", p.emptyPattern) // should clear pattern
}

func TestProgress_EmptyPattern(t *testing.T) {
	p := Progress(50, 100).EmptyPattern("·-")
	assert.Equal(t, "·-", p.emptyPattern)

	// Setting empty char should clear pattern
	p.EmptyChar('░')
	assert.Equal(t, '░', p.emptyChar)
	assert.Equal(t, "", p.emptyPattern)
}

func TestProgress_ShowPercent(t *testing.T) {
	p := Progress(50, 100).ShowPercent()
	assert.Equal(t, true, p.showPercent)
	assert.Equal(t, false, p.showFraction)
}

func TestProgress_HidePercent(t *testing.T) {
	p := Progress(50, 100).HidePercent()
	assert.Equal(t, false, p.showPercent)
}

func TestProgress_ShowFraction(t *testing.T) {
	p := Progress(50, 100).ShowFraction()
	assert.Equal(t, true, p.showFraction)
	assert.Equal(t, false, p.showPercent)
}

func TestProgress_PercentStyle(t *testing.T) {
	customStyle := NewStyle().WithForeground(ColorCyan)
	p := Progress(50, 100).PercentStyle(customStyle)
	assert.NotNil(t, p.percentStyle)
	assert.Equal(t, ColorCyan, p.percentStyle.Foreground)
}

func TestProgress_PercentFg(t *testing.T) {
	p := Progress(50, 100).PercentFg(ColorMagenta)
	assert.NotNil(t, p.percentStyle)
	assert.Equal(t, ColorMagenta, p.percentStyle.Foreground)
}

func TestProgress_Label(t *testing.T) {
	p := Progress(50, 100).Label("Downloading:")
	assert.Equal(t, "Downloading:", p.label)
}

func TestProgress_Shimmer(t *testing.T) {
	p := Progress(50, 100).Shimmer(NewRGB(255, 255, 255), 10)
	assert.NotNil(t, p)
	assert.Equal(t, true, p.shimmer)
	assert.Equal(t, 10, p.shimmerSpeed)
}

func TestProgress_Pulse(t *testing.T) {
	p := Progress(50, 100).Pulse(NewRGB(0, 255, 0), 15)
	assert.NotNil(t, p)
	assert.Equal(t, true, p.pulse)
	assert.Equal(t, 15, p.pulseSpeed)
}

func TestLoading_Basic(t *testing.T) {
	l := Loading(0)
	assert.Equal(t, uint64(0), l.frame)
	assert.Equal(t, 4, l.speed)
	assert.NotNil(t, l.charset)
	assert.Equal(t, SpinnerDots.Frames, l.charset)
}

func TestLoading_CharSet(t *testing.T) {
	custom := []string{"a", "b", "c"}
	l := Loading(0).CharSet(custom)
	assert.Equal(t, custom, l.charset)
}

func TestLoading_Speed(t *testing.T) {
	l := Loading(0).Speed(10)
	assert.Equal(t, 10, l.speed)
}

func TestLoading_Label(t *testing.T) {
	l := Loading(0).Label("Loading...")
	assert.Equal(t, "Loading...", l.label)
}
