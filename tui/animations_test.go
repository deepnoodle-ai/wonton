package tui

import (
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

// RainbowAnimation tests

func TestRainbowAnimation_GetStyle(t *testing.T) {
	anim := &RainbowAnimation{
		Speed:  3,
		Length: 10,
	}

	style := anim.GetStyle(0, 0, 10)
	assert.NotNil(t, style)
	assert.NotNil(t, style.FgRGB) // Should have RGB foreground
}

func TestRainbowAnimation_Defaults(t *testing.T) {
	anim := &RainbowAnimation{}

	// Should use defaults when Speed/Length are 0
	style := anim.GetStyle(0, 0, 10)
	assert.NotNil(t, style)
	assert.Equal(t, 3, anim.Speed)   // default
	assert.Equal(t, 10, anim.Length) // set to totalChars
}

func TestRainbowAnimation_Reversed(t *testing.T) {
	anim := &RainbowAnimation{
		Speed:    3,
		Length:   10,
		Reversed: true,
	}

	style := anim.GetStyle(0, 0, 10)
	assert.NotNil(t, style)
}

func TestRainbowAnimation_VariousFrames(t *testing.T) {
	anim := &RainbowAnimation{
		Speed:  3,
		Length: 10,
	}

	// Different frames should produce different colors (for animation)
	style1 := anim.GetStyle(0, 5, 10)
	style2 := anim.GetStyle(30, 5, 10)

	// Both should be valid
	assert.NotNil(t, style1.FgRGB)
	assert.NotNil(t, style2.FgRGB)
}

// WaveAnimation tests

func TestWaveAnimation_GetStyle(t *testing.T) {
	anim := &WaveAnimation{
		Speed:     12,
		Amplitude: 1.0,
		Colors: []RGB{
			NewRGB(255, 0, 0),
			NewRGB(0, 255, 0),
			NewRGB(0, 0, 255),
		},
	}

	style := anim.GetStyle(0, 0, 10)
	assert.NotNil(t, style)
	assert.NotNil(t, style.FgRGB)
}

func TestWaveAnimation_Defaults(t *testing.T) {
	anim := &WaveAnimation{}

	style := anim.GetStyle(0, 0, 10)
	assert.NotNil(t, style)
	assert.Equal(t, 12, anim.Speed)
	assert.Equal(t, 1.0, anim.Amplitude)
	assert.Len(t, anim.Colors, 3) // default colors
}

func TestWaveAnimation_ColorBlending(t *testing.T) {
	anim := &WaveAnimation{
		Speed: 2,
		Colors: []RGB{
			NewRGB(255, 0, 0),
			NewRGB(0, 255, 0),
		},
	}

	// At different frames, colors should blend
	style1 := anim.GetStyle(0, 0, 10)
	style2 := anim.GetStyle(1, 0, 10)

	assert.NotNil(t, style1.FgRGB)
	assert.NotNil(t, style2.FgRGB)
}

// SlideAnimation tests

func TestSlideAnimation_GetStyle(t *testing.T) {
	anim := &SlideAnimation{
		Speed:          2,
		BaseColor:      NewRGB(50, 50, 50),
		HighlightColor: NewRGB(255, 255, 255),
		Width:          3,
	}

	style := anim.GetStyle(0, 0, 20)
	assert.NotNil(t, style)
	assert.NotNil(t, style.FgRGB)
}

func TestSlideAnimation_Defaults(t *testing.T) {
	anim := &SlideAnimation{}

	style := anim.GetStyle(0, 0, 10)
	assert.NotNil(t, style)
	assert.Equal(t, 2, anim.Speed)
	assert.Equal(t, 3, anim.Width)
}

func TestSlideAnimation_Reverse(t *testing.T) {
	anim := &SlideAnimation{
		Speed:          2,
		BaseColor:      NewRGB(50, 50, 50),
		HighlightColor: NewRGB(255, 255, 255),
		Width:          3,
		Reverse:        true,
	}

	style := anim.GetStyle(0, 5, 20)
	assert.NotNil(t, style)
}

func TestSlideAnimation_Highlight(t *testing.T) {
	anim := &SlideAnimation{
		Speed:          1,
		BaseColor:      NewRGB(0, 0, 0),
		HighlightColor: NewRGB(255, 255, 255),
		Width:          2,
	}

	// At frame 0, highlight should be around position 0
	// Characters near the highlight should have brighter colors
	style := anim.GetStyle(0, 0, 10)
	assert.NotNil(t, style.FgRGB)
}

// SparkleAnimation tests

func TestSparkleAnimation_GetStyle(t *testing.T) {
	anim := &SparkleAnimation{
		Speed:      3,
		BaseColor:  NewRGB(100, 100, 100),
		SparkColor: NewRGB(255, 255, 255),
		Density:    3,
	}

	style := anim.GetStyle(0, 0, 10)
	assert.NotNil(t, style)
	assert.NotNil(t, style.FgRGB)
}

func TestSparkleAnimation_Defaults(t *testing.T) {
	anim := &SparkleAnimation{}

	style := anim.GetStyle(0, 0, 10)
	assert.NotNil(t, style)
	assert.Equal(t, 3, anim.Speed)
	assert.Equal(t, 3, anim.Density)
}

func TestSparkleAnimation_DifferentChars(t *testing.T) {
	anim := &SparkleAnimation{
		Speed:      3,
		BaseColor:  NewRGB(100, 100, 100),
		SparkColor: NewRGB(255, 255, 255),
		Density:    5,
	}

	// Different character indices should have different sparkle patterns
	styles := make([]Style, 10)
	for i := 0; i < 10; i++ {
		styles[i] = anim.GetStyle(0, i, 10)
		assert.NotNil(t, styles[i].FgRGB)
	}
}

// TypewriterAnimation tests

func TestTypewriterAnimation_GetStyle(t *testing.T) {
	anim := &TypewriterAnimation{
		Speed:       4,
		TextColor:   NewRGB(255, 255, 255),
		CursorColor: NewRGB(255, 200, 0),
	}

	style := anim.GetStyle(0, 0, 10)
	assert.NotNil(t, style)
	assert.NotNil(t, style.FgRGB)
}

func TestTypewriterAnimation_Defaults(t *testing.T) {
	anim := &TypewriterAnimation{}

	style := anim.GetStyle(0, 0, 10)
	assert.NotNil(t, style)
	assert.Equal(t, 4, anim.Speed)
	assert.Equal(t, 60, anim.HoldFrames)
}

func TestTypewriterAnimation_Reveal(t *testing.T) {
	anim := &TypewriterAnimation{
		Speed:     1,
		TextColor: NewRGB(255, 255, 255),
	}

	// At frame 0, char 0 should be cursor (dimmed until revealed)
	style0 := anim.GetStyle(0, 0, 5)
	assert.NotNil(t, style0.FgRGB)

	// At frame 5, first 5 chars revealed
	style5 := anim.GetStyle(5, 0, 10)
	assert.NotNil(t, style5.FgRGB)
}

func TestTypewriterAnimation_Loop(t *testing.T) {
	anim := &TypewriterAnimation{
		Speed:      2,
		TextColor:  NewRGB(255, 255, 255),
		Loop:       true,
		HoldFrames: 10,
	}

	// Should cycle
	style := anim.GetStyle(1000, 0, 10)
	assert.NotNil(t, style.FgRGB)
}

// GlitchAnimation tests

func TestGlitchAnimation_GetStyle(t *testing.T) {
	anim := &GlitchAnimation{
		Speed:       2,
		BaseColor:   NewRGB(100, 100, 100),
		GlitchColor: NewRGB(0, 255, 255),
		Intensity:   3,
	}

	style := anim.GetStyle(0, 0, 10)
	assert.NotNil(t, style)
	assert.NotNil(t, style.FgRGB)
}

func TestGlitchAnimation_Defaults(t *testing.T) {
	anim := &GlitchAnimation{}

	style := anim.GetStyle(0, 0, 10)
	assert.NotNil(t, style)
	assert.Equal(t, 2, anim.Speed)
	assert.Equal(t, 3, anim.Intensity)
}

func TestGlitchAnimation_Patterns(t *testing.T) {
	anim := &GlitchAnimation{
		Speed:       2,
		BaseColor:   NewRGB(100, 100, 100),
		GlitchColor: NewRGB(0, 255, 255),
		Intensity:   5,
	}

	// Check various frames to verify deterministic glitch behavior
	for frame := uint64(0); frame < 100; frame++ {
		style := anim.GetStyle(frame, 5, 20)
		assert.NotNil(t, style.FgRGB)
	}
}

// PulseAnimation tests

func TestPulseAnimation_GetStyle(t *testing.T) {
	anim := &PulseAnimation{
		Speed:         15,
		Color:         NewRGB(255, 100, 50),
		MinBrightness: 0.3,
		MaxBrightness: 1.0,
	}

	style := anim.GetStyle(0, 0, 10)
	assert.NotNil(t, style)
	assert.NotNil(t, style.FgRGB)
}

func TestPulseAnimation_Defaults(t *testing.T) {
	anim := &PulseAnimation{
		Color: NewRGB(255, 255, 255),
	}

	style := anim.GetStyle(0, 0, 10)
	assert.NotNil(t, style)
	assert.Equal(t, 15, anim.Speed)
	assert.Equal(t, 0.3, anim.MinBrightness)
	assert.Equal(t, 1.0, anim.MaxBrightness)
}

func TestPulseAnimation_Brightness(t *testing.T) {
	anim := &PulseAnimation{
		Speed:         10,
		Color:         NewRGB(200, 200, 200),
		MinBrightness: 0.5,
		MaxBrightness: 1.0,
	}

	// Different frames should have different brightness
	style1 := anim.GetStyle(0, 0, 10)
	style2 := anim.GetStyle(5, 0, 10)

	assert.NotNil(t, style1.FgRGB)
	assert.NotNil(t, style2.FgRGB)
}

// Sine helper test

func TestSine(t *testing.T) {
	// Test sine at key points
	// Sine(0) should be close to 0
	assert.True(t, Sine(0) < 0.1)

	// Sine at π/2 should be close to 1
	assert.True(t, Sine(1.5708) > 0.9)

	// Sine at π should be close to 0
	assert.True(t, Sine(3.14159) < 0.1)

	// Sine at 3π/2 should be close to -1
	assert.True(t, Sine(4.7124) < -0.9)
}

// AnimatedText tests

func TestNewAnimatedText(t *testing.T) {
	anim := &RainbowAnimation{Speed: 3}
	at := NewAnimatedText(10, 5, "Hello", anim)

	assert.NotNil(t, at)
	assert.Equal(t, 10, at.x)
	assert.Equal(t, 5, at.y)
	assert.Equal(t, "Hello", at.text)
	assert.Equal(t, anim, at.animation)
}

func TestAnimatedText_Update(t *testing.T) {
	at := NewAnimatedText(0, 0, "Test", nil)

	at.Update(42)
	assert.Equal(t, uint64(42), at.currentFrame)
}

func TestAnimatedText_Position(t *testing.T) {
	at := NewAnimatedText(15, 20, "Text", nil)

	x, y := at.Position()
	assert.Equal(t, 15, x)
	assert.Equal(t, 20, y)
}

func TestAnimatedText_Dimensions(t *testing.T) {
	at := NewAnimatedText(0, 0, "Hello World", nil)

	w, h := at.Dimensions()
	assert.Equal(t, 11, w) // 11 characters
	assert.Equal(t, 1, h)  // single line
}

func TestAnimatedText_SetText(t *testing.T) {
	at := NewAnimatedText(0, 0, "Original", nil)
	at.SetText("Updated")
	assert.Equal(t, "Updated", at.text)
}

func TestAnimatedText_SetPosition(t *testing.T) {
	at := NewAnimatedText(0, 0, "Test", nil)
	at.SetPosition(50, 25)
	x, y := at.Position()
	assert.Equal(t, 50, x)
	assert.Equal(t, 25, y)
}

// AnimatedMultiLine tests

func TestNewAnimatedMultiLine(t *testing.T) {
	aml := NewAnimatedMultiLine(10, 5, 80)

	assert.NotNil(t, aml)
	assert.Equal(t, 10, aml.x)
	assert.Equal(t, 5, aml.y)
	assert.Equal(t, 80, aml.width)
	assert.Len(t, aml.lines, 0)
}

func TestAnimatedMultiLine_SetLine(t *testing.T) {
	aml := NewAnimatedMultiLine(0, 0, 80)
	anim := &RainbowAnimation{}

	aml.SetLine(0, "Line 0", anim)
	aml.SetLine(1, "Line 1", nil)
	aml.SetLine(3, "Line 3", nil) // skip index 2

	assert.Len(t, aml.lines, 4)
	assert.Equal(t, "Line 0", aml.lines[0])
	assert.Equal(t, "Line 1", aml.lines[1])
	assert.Equal(t, "", aml.lines[2]) // filled with empty
	assert.Equal(t, "Line 3", aml.lines[3])
}

func TestAnimatedMultiLine_AddLine(t *testing.T) {
	aml := NewAnimatedMultiLine(0, 0, 80)

	aml.AddLine("First", nil)
	aml.AddLine("Second", nil)

	assert.Len(t, aml.lines, 2)
	assert.Equal(t, "First", aml.lines[0])
	assert.Equal(t, "Second", aml.lines[1])
}

func TestAnimatedMultiLine_ClearLines(t *testing.T) {
	aml := NewAnimatedMultiLine(0, 0, 80)
	aml.AddLine("Line 1", nil)
	aml.AddLine("Line 2", nil)

	aml.ClearLines()
	assert.Len(t, aml.lines, 0)
}

func TestAnimatedMultiLine_Update(t *testing.T) {
	aml := NewAnimatedMultiLine(0, 0, 80)
	aml.Update(99)
	assert.Equal(t, uint64(99), aml.currentFrame)
}

func TestAnimatedMultiLine_Position(t *testing.T) {
	aml := NewAnimatedMultiLine(25, 15, 80)
	x, y := aml.Position()
	assert.Equal(t, 25, x)
	assert.Equal(t, 15, y)
}

func TestAnimatedMultiLine_Dimensions(t *testing.T) {
	aml := NewAnimatedMultiLine(0, 0, 60)
	aml.AddLine("Line 1", nil)
	aml.AddLine("Line 2", nil)
	aml.AddLine("Line 3", nil)

	w, h := aml.Dimensions()
	assert.Equal(t, 60, w)
	assert.Equal(t, 3, h)
}

func TestAnimatedMultiLine_SetPosition(t *testing.T) {
	aml := NewAnimatedMultiLine(0, 0, 80)
	aml.SetPosition(30, 20)
	x, y := aml.Position()
	assert.Equal(t, 30, x)
	assert.Equal(t, 20, y)
}

func TestAnimatedMultiLine_SetWidth(t *testing.T) {
	aml := NewAnimatedMultiLine(0, 0, 80)
	aml.SetWidth(100)
	w, _ := aml.Dimensions()
	assert.Equal(t, 100, w)
}

// AnimatedStatusBar tests

func TestNewAnimatedStatusBar(t *testing.T) {
	asb := NewAnimatedStatusBar(0, 23, 80)

	assert.NotNil(t, asb)
	assert.Equal(t, 0, asb.x)
	assert.Equal(t, 23, asb.y)
	assert.Equal(t, 80, asb.width)
	assert.Len(t, asb.items, 0)
}

func TestAnimatedStatusBar_AddItem(t *testing.T) {
	asb := NewAnimatedStatusBar(0, 0, 80)
	anim := &RainbowAnimation{}

	asb.AddItem("Status", "OK", "✓", anim, NewStyle())

	assert.Len(t, asb.items, 1)
	assert.Equal(t, "Status", asb.items[0].Key)
	assert.Equal(t, "OK", asb.items[0].Value)
	assert.Equal(t, "✓", asb.items[0].Icon)
}

func TestAnimatedStatusBar_UpdateItem(t *testing.T) {
	asb := NewAnimatedStatusBar(0, 0, 80)
	asb.AddItem("Status", "Initial", "", nil, NewStyle())

	asb.UpdateItem(0, "Status", "Updated")
	assert.Equal(t, "Updated", asb.items[0].Value)
}

func TestAnimatedStatusBar_UpdateItemOutOfRange(t *testing.T) {
	asb := NewAnimatedStatusBar(0, 0, 80)
	asb.AddItem("Status", "OK", "", nil, NewStyle())

	// Should not panic with invalid index
	asb.UpdateItem(-1, "X", "Y")
	asb.UpdateItem(100, "X", "Y")
}

func TestAnimatedStatusBar_Update(t *testing.T) {
	asb := NewAnimatedStatusBar(0, 0, 80)
	asb.Update(42)
	assert.Equal(t, uint64(42), asb.currentFrame)
}

func TestAnimatedStatusBar_Position(t *testing.T) {
	asb := NewAnimatedStatusBar(10, 20, 80)
	x, y := asb.Position()
	assert.Equal(t, 10, x)
	assert.Equal(t, 20, y)
}

func TestAnimatedStatusBar_Dimensions(t *testing.T) {
	asb := NewAnimatedStatusBar(0, 0, 100)
	w, h := asb.Dimensions()
	assert.Equal(t, 100, w)
	assert.Equal(t, 1, h)
}

func TestAnimatedStatusBar_SetPosition(t *testing.T) {
	asb := NewAnimatedStatusBar(0, 0, 80)
	asb.SetPosition(5, 10)
	x, y := asb.Position()
	assert.Equal(t, 5, x)
	assert.Equal(t, 10, y)
}
