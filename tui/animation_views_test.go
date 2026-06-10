package tui

import (
	"bytes"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

// recordingBorderAnim records the position passed for each border part.
type recordingBorderAnim struct {
	positions map[BorderPart][]int
	length    int
}

func (r *recordingBorderAnim) GetBorderStyle(frame uint64, part BorderPart, position int, length int) Style {
	if r.positions == nil {
		r.positions = make(map[BorderPart][]int)
	}
	r.positions[part] = append(r.positions[part], position)
	r.length = length
	return NewStyle()
}

// Positions must run clockwise around the full perimeter with no gaps so that
// position-based animations (rainbow, gradient, wave) flow smoothly through
// the corners.
func TestAnimatedBordered_PerimeterPositions(t *testing.T) {
	const w, h = 6, 4
	perimeter := (w-2)*2 + (h-2)*2 + 4

	var buf bytes.Buffer
	terminal := NewTestTerminal(w, h, &buf)
	frame, err := terminal.BeginFrame()
	assert.NoError(t, err)
	defer terminal.EndFrame(frame)

	anim := &recordingBorderAnim{}
	view := AnimatedBordered(Text("x"), anim)
	view.render(NewRenderContext(frame, 0))

	assert.Equal(t, perimeter, anim.length)

	// Corners sit at their true perimeter positions.
	assert.Equal(t, []int{0}, anim.positions[BorderPartTopLeft])
	assert.Equal(t, []int{w - 1}, anim.positions[BorderPartTopRight])
	assert.Equal(t, []int{w + h - 2}, anim.positions[BorderPartBottomRight])
	assert.Equal(t, []int{2*w + h - 3}, anim.positions[BorderPartBottomLeft])

	// Edges fill the slots between the corners.
	assert.Equal(t, []int{1, 2, 3, 4}, anim.positions[BorderPartTop])
	assert.Equal(t, []int{6, 7}, anim.positions[BorderPartRight])
	assert.Equal(t, []int{9, 10, 11, 12}, anim.positions[BorderPartBottom])
	assert.Equal(t, []int{14, 15}, anim.positions[BorderPartLeft])

	// Together they cover every perimeter position exactly once.
	seen := make(map[int]bool)
	for _, positions := range anim.positions {
		for _, p := range positions {
			assert.False(t, seen[p], "position %d used more than once", p)
			seen[p] = true
		}
	}
	assert.Equal(t, perimeter, len(seen))
}
