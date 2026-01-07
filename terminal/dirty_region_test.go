package terminal

import (
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestDirtyRegionEmptyAndClear(t *testing.T) {
	var dr DirtyRegion
	assert.True(t, dr.Empty())

	dr.Mark(2, 3)
	assert.True(t, !dr.Empty())

	dr.Clear()
	assert.True(t, dr.Empty())
	assert.Equal(t, dr.MinX, 0)
	assert.Equal(t, dr.MinY, 0)
	assert.Equal(t, dr.MaxX, 0)
	assert.Equal(t, dr.MaxY, 0)
}

func TestDirtyRegionMarkExpandsBounds(t *testing.T) {
	var dr DirtyRegion
	dr.Mark(4, 5)
	assert.Equal(t, dr.MinX, 4)
	assert.Equal(t, dr.MaxX, 4)
	assert.Equal(t, dr.MinY, 5)
	assert.Equal(t, dr.MaxY, 5)

	dr.Mark(2, 8)
	assert.Equal(t, dr.MinX, 2)
	assert.Equal(t, dr.MaxX, 4)
	assert.Equal(t, dr.MinY, 5)
	assert.Equal(t, dr.MaxY, 8)

	dr.Mark(9, 3)
	assert.Equal(t, dr.MinX, 2)
	assert.Equal(t, dr.MaxX, 9)
	assert.Equal(t, dr.MinY, 3)
	assert.Equal(t, dr.MaxY, 8)
}

func TestDirtyRegionMarkRect(t *testing.T) {
	var dr DirtyRegion
	dr.MarkRect(2, 3, 4, 2) // covers x=2..5, y=3..4
	assert.Equal(t, dr.MinX, 2)
	assert.Equal(t, dr.MaxX, 5)
	assert.Equal(t, dr.MinY, 3)
	assert.Equal(t, dr.MaxY, 4)

	dr.MarkRect(0, 1, 2, 6) // expands to x=0..5, y=1..6
	assert.Equal(t, dr.MinX, 0)
	assert.Equal(t, dr.MaxX, 5)
	assert.Equal(t, dr.MinY, 1)
	assert.Equal(t, dr.MaxY, 6)
}

func TestDirtyRegionMarkRectIgnoresEmpty(t *testing.T) {
	var dr DirtyRegion
	dr.MarkRect(1, 1, 0, 3)
	assert.True(t, dr.Empty())

	dr.MarkRect(1, 1, 3, 0)
	assert.True(t, dr.Empty())
}
