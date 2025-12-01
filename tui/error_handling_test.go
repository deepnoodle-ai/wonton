package tui

import (
	"bytes"
	"errors"
	"image"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
	"github.com/deepnoodle-ai/wonton/require"
)

// TestError_Types verifies all error types are properly defined
func TestError_Types(t *testing.T) {
	assert.NotNil(t, ErrOutOfBounds)
	assert.NotNil(t, ErrNotInRawMode)
	assert.NotNil(t, ErrClosed)
	assert.NotNil(t, ErrInvalidFrame)
	assert.NotNil(t, ErrAlreadyActive)
}

// TestError_Messages verifies error messages are descriptive
func TestError_Messages(t *testing.T) {
	assert.Contains(t, ErrOutOfBounds.Error(), "bounds")
	assert.Contains(t, ErrNotInRawMode.Error(), "raw mode")
	assert.Contains(t, ErrClosed.Error(), "closed")
	assert.Contains(t, ErrInvalidFrame.Error(), "frame")
	assert.Contains(t, ErrAlreadyActive.Error(), "active")
}

// TestSetCell_OutOfBounds verifies SetCell returns proper errors
func TestSetCell_OutOfBounds(t *testing.T) {
	term := NewTestTerminal(80, 24, &bytes.Buffer{})
	frame, _ := term.BeginFrame()

	tests := []struct {
		name string
		x, y int
	}{
		{"negative x", -1, 10},
		{"negative y", 10, -1},
		{"x too large", 100, 10},
		{"y too large", 10, 100},
		{"both negative", -5, -5},
		{"both too large", 200, 200},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := frame.SetCell(tt.x, tt.y, 'X', NewStyle())
			assert.Error(t, err)
			assert.True(t, errors.Is(err, ErrOutOfBounds), "Expected ErrOutOfBounds, got %v", err)
		})
	}

	term.EndFrame(frame)
}

// TestSetCell_BoundaryValues verifies edge cases work correctly
func TestSetCell_BoundaryValues(t *testing.T) {
	term := NewTestTerminal(80, 24, &bytes.Buffer{})
	frame, _ := term.BeginFrame()

	tests := []struct {
		name string
		x, y int
	}{
		{"origin", 0, 0},
		{"max x", 79, 0},
		{"max y", 0, 23},
		{"bottom right", 79, 23},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := frame.SetCell(tt.x, tt.y, 'X', NewStyle())
			assert.NoError(t, err)
		})
	}

	term.EndFrame(frame)
}

// TestSubFrame_BoundsClipping verifies subframes clip properly
func TestSubFrame_BoundsClipping(t *testing.T) {
	term := NewTestTerminal(80, 24, &bytes.Buffer{})
	frame, _ := term.BeginFrame()

	// Create subframe that extends beyond terminal
	subFrame := frame.SubFrame(image.Rect(70, 20, 100, 30))

	// Should be clipped to terminal bounds
	w, h := subFrame.Size()
	assert.LessOrEqual(t, w, 10, "Width should be clipped")
	assert.LessOrEqual(t, h, 4, "Height should be clipped")

	term.EndFrame(frame)
}

// TestSubFrame_EmptyIntersection verifies empty subframes
func TestSubFrame_EmptyIntersection(t *testing.T) {
	term := NewTestTerminal(80, 24, &bytes.Buffer{})
	frame, _ := term.BeginFrame()

	// Create subframe completely outside terminal
	subFrame := frame.SubFrame(image.Rect(100, 100, 110, 110))

	w, h := subFrame.Size()
	assert.Equal(t, 0, w, "Empty subframe should have 0 width")
	assert.Equal(t, 0, h, "Empty subframe should have 0 height")

	// Writing to empty subframe should not panic
	require.NotPanics(t, func() {
		subFrame.SetCell(0, 0, 'X', NewStyle())
	})

	term.EndFrame(frame)
}

// TestFillStyled_PartiallyOutOfBounds verifies clipping behavior
func TestFillStyled_PartiallyOutOfBounds(t *testing.T) {
	term := NewTestTerminal(80, 24, &bytes.Buffer{})
	frame, _ := term.BeginFrame()

	// Fill that extends beyond bounds
	err := frame.FillStyled(70, 20, 20, 10, '#', NewStyle())

	// Should clip gracefully (some implementations might return error, others clip silently)
	// The key is it shouldn't panic
	_ = err // Error handling is implementation-dependent

	term.EndFrame(frame)
}

// TestPrintStyled_OutOfBounds verifies out of bounds printing
func TestPrintStyled_OutOfBounds(t *testing.T) {
	term := NewTestTerminal(80, 24, &bytes.Buffer{})
	frame, _ := term.BeginFrame()

	// Print completely out of bounds
	err := frame.PrintStyled(-10, -10, "Test", NewStyle())
	_ = err // Implementation-dependent

	// Print starting out of bounds
	err = frame.PrintStyled(100, 100, "Test", NewStyle())
	_ = err // Implementation-dependent

	// Should not panic
	term.EndFrame(frame)
}

// TestDirtyRegion_EmptyOperations verifies empty dirty region behavior
func TestDirtyRegion_EmptyOperations(t *testing.T) {
	dr := &DirtyRegion{}

	assert.True(t, dr.Empty())

	// Mark with zero width/height should not make it dirty
	dr.MarkRect(10, 10, 0, 10)
	assert.True(t, dr.Empty())

	dr.MarkRect(10, 10, 10, 0)
	assert.True(t, dr.Empty())

	// Negative dimensions should not make it dirty
	dr.MarkRect(10, 10, -5, 10)
	assert.True(t, dr.Empty())
}

// TestDirtyRegion_ClearAfterMark verifies Clear works
func TestDirtyRegion_ClearAfterMark(t *testing.T) {
	dr := &DirtyRegion{}

	dr.Mark(10, 10)
	assert.False(t, dr.Empty())

	dr.Clear()
	assert.True(t, dr.Empty())
	assert.Equal(t, 0, dr.MinX)
	assert.Equal(t, 0, dr.MinY)
	assert.Equal(t, 0, dr.MaxX)
	assert.Equal(t, 0, dr.MaxY)
}

// TestAnimation_EmptyText verifies empty text handling
func TestAnimation_EmptyText(t *testing.T) {
	term := NewTestTerminal(80, 24, &bytes.Buffer{})

	elem := NewAnimatedText(10, 10, "", &RainbowAnimation{})

	frame, _ := term.BeginFrame()
	require.NotPanics(t, func() {
		elem.Draw(frame)
	})
	term.EndFrame(frame)
}

// TestAnimation_ZeroSpeedParameters verifies default handling
func TestAnimation_ZeroSpeedParameters(t *testing.T) {
	// RainbowAnimation with zero speed should use defaults
	rainbow := &RainbowAnimation{Speed: 0, Length: 0}
	style := rainbow.GetStyle(0, 0, 10)
	assert.NotNil(t, style)

	// PulseAnimation with zero parameters should use defaults
	pulse := &PulseAnimation{Speed: 0}
	style = pulse.GetStyle(0, 0, 10)
	assert.NotNil(t, style)

	// WaveAnimation with zero parameters should use defaults
	wave := &WaveAnimation{Speed: 0}
	style = wave.GetStyle(0, 0, 10)
	assert.NotNil(t, style)
}

// TestAnimatedMultiLine_EmptyLines verifies empty line handling
func TestAnimatedMultiLine_EmptyLines(t *testing.T) {
	term := NewTestTerminal(80, 24, &bytes.Buffer{})
	aml := NewAnimatedMultiLine(0, 0, 80)

	// Add empty line
	aml.AddLine("", &RainbowAnimation{})

	frame, _ := term.BeginFrame()
	require.NotPanics(t, func() {
		aml.Draw(frame)
	})
	term.EndFrame(frame)
}

// TestAnimatedMultiLine_OutOfBoundsLineIndex verifies safe line access
func TestAnimatedMultiLine_OutOfBoundsLineIndex(t *testing.T) {
	aml := NewAnimatedMultiLine(0, 0, 80)

	// Set line beyond current size - should expand safely
	require.NotPanics(t, func() {
		aml.SetLine(10, "Test", nil)
	})

	// Verify it was added
	w, h := aml.Dimensions()
	assert.Equal(t, 80, w)
	assert.GreaterOrEqual(t, h, 11)
}

// TestGradient_EdgeCases verifies gradient edge cases
func TestGradient_EdgeCases(t *testing.T) {
	start := NewRGB(255, 0, 0)
	end := NewRGB(0, 0, 255)

	// Zero steps - returns start color
	colors := Gradient(start, end, 0)
	assert.Len(t, colors, 1)
	assert.Equal(t, start, colors[0])

	// Negative steps - returns start color (steps <= 1)
	colors = Gradient(start, end, -5)
	assert.Len(t, colors, 1)
	assert.Equal(t, start, colors[0])

	// One step - returns start color
	colors = Gradient(start, end, 1)
	assert.Len(t, colors, 1)
	assert.Equal(t, start, colors[0])
}

// TestMultiGradient_EdgeCases verifies multi-gradient edge cases
func TestMultiGradient_EdgeCases(t *testing.T) {
	// Empty stops
	colors := MultiGradient([]RGB{}, 10)
	assert.Len(t, colors, 0)

	// Nil stops
	colors = MultiGradient(nil, 10)
	assert.Len(t, colors, 0)

	// Zero steps
	colors = MultiGradient([]RGB{NewRGB(255, 0, 0)}, 0)
	assert.Len(t, colors, 0)
}
