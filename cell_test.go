package gooey

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCell_DefaultValues(t *testing.T) {
	cell := Cell{}
	assert.Equal(t, rune(0), cell.Char)
	assert.Equal(t, 0, cell.Width)
	assert.False(t, cell.Continuation)
}

func TestCell_NormalCharacter(t *testing.T) {
	style := NewStyle().WithForeground(ColorRed)
	cell := Cell{
		Char:         'A',
		Style:        style,
		Width:        1,
		Continuation: false,
	}

	assert.Equal(t, 'A', cell.Char)
	assert.Equal(t, 1, cell.Width)
	assert.False(t, cell.Continuation)
	assert.Equal(t, ColorRed, cell.Style.Foreground)
}

func TestCell_WideCharacter(t *testing.T) {
	style := NewStyle()
	cell := Cell{
		Char:         'Ａ', // Full-width A
		Style:        style,
		Width:        2,
		Continuation: false,
	}

	assert.Equal(t, 'Ａ', cell.Char)
	assert.Equal(t, 2, cell.Width)
	assert.False(t, cell.Continuation)
}

func TestCell_ContinuationCell(t *testing.T) {
	cell := Cell{
		Char:         0,
		Style:        NewStyle(),
		Width:        0,
		Continuation: true,
	}

	assert.Equal(t, rune(0), cell.Char)
	assert.Equal(t, 0, cell.Width)
	assert.True(t, cell.Continuation)
}

func TestDirtyRegion_Empty(t *testing.T) {
	dr := &DirtyRegion{}
	assert.True(t, dr.Empty())
	assert.False(t, dr.dirty)
}

func TestDirtyRegion_Clear(t *testing.T) {
	dr := &DirtyRegion{
		MinX:  10,
		MinY:  5,
		MaxX:  20,
		MaxY:  15,
		dirty: true,
	}

	dr.Clear()

	assert.False(t, dr.dirty)
	assert.Equal(t, 0, dr.MinX)
	assert.Equal(t, 0, dr.MinY)
	assert.Equal(t, 0, dr.MaxX)
	assert.Equal(t, 0, dr.MaxY)
	assert.True(t, dr.Empty())
}

func TestDirtyRegion_Mark_FirstCell(t *testing.T) {
	dr := &DirtyRegion{}
	dr.Mark(10, 5)

	assert.True(t, dr.dirty)
	assert.Equal(t, 10, dr.MinX)
	assert.Equal(t, 5, dr.MinY)
	assert.Equal(t, 10, dr.MaxX)
	assert.Equal(t, 5, dr.MaxY)
}

func TestDirtyRegion_Mark_ExpandRight(t *testing.T) {
	dr := &DirtyRegion{}
	dr.Mark(10, 5)
	dr.Mark(15, 5)

	assert.Equal(t, 10, dr.MinX)
	assert.Equal(t, 5, dr.MinY)
	assert.Equal(t, 15, dr.MaxX)
	assert.Equal(t, 5, dr.MaxY)
}

func TestDirtyRegion_Mark_ExpandLeft(t *testing.T) {
	dr := &DirtyRegion{}
	dr.Mark(10, 5)
	dr.Mark(5, 5)

	assert.Equal(t, 5, dr.MinX)
	assert.Equal(t, 5, dr.MinY)
	assert.Equal(t, 10, dr.MaxX)
	assert.Equal(t, 5, dr.MaxY)
}

func TestDirtyRegion_Mark_ExpandDown(t *testing.T) {
	dr := &DirtyRegion{}
	dr.Mark(10, 5)
	dr.Mark(10, 10)

	assert.Equal(t, 10, dr.MinX)
	assert.Equal(t, 5, dr.MinY)
	assert.Equal(t, 10, dr.MaxX)
	assert.Equal(t, 10, dr.MaxY)
}

func TestDirtyRegion_Mark_ExpandUp(t *testing.T) {
	dr := &DirtyRegion{}
	dr.Mark(10, 5)
	dr.Mark(10, 2)

	assert.Equal(t, 10, dr.MinX)
	assert.Equal(t, 2, dr.MinY)
	assert.Equal(t, 10, dr.MaxX)
	assert.Equal(t, 5, dr.MaxY)
}

func TestDirtyRegion_Mark_ExpandAllDirections(t *testing.T) {
	dr := &DirtyRegion{}
	dr.Mark(10, 10)
	dr.Mark(5, 5)
	dr.Mark(15, 15)
	dr.Mark(8, 12)

	assert.Equal(t, 5, dr.MinX)
	assert.Equal(t, 5, dr.MinY)
	assert.Equal(t, 15, dr.MaxX)
	assert.Equal(t, 15, dr.MaxY)
}

func TestDirtyRegion_MarkRect_FirstRect(t *testing.T) {
	dr := &DirtyRegion{}
	dr.MarkRect(10, 5, 20, 10)

	assert.True(t, dr.dirty)
	assert.Equal(t, 10, dr.MinX)
	assert.Equal(t, 5, dr.MinY)
	assert.Equal(t, 29, dr.MaxX) // 10 + 20 - 1
	assert.Equal(t, 14, dr.MaxY) // 5 + 10 - 1
}

func TestDirtyRegion_MarkRect_ExpandRect(t *testing.T) {
	dr := &DirtyRegion{}
	dr.MarkRect(10, 10, 5, 5)
	dr.MarkRect(20, 20, 5, 5)

	assert.Equal(t, 10, dr.MinX)
	assert.Equal(t, 10, dr.MinY)
	assert.Equal(t, 24, dr.MaxX) // 20 + 5 - 1
	assert.Equal(t, 24, dr.MaxY) // 20 + 5 - 1
}

func TestDirtyRegion_MarkRect_ZeroWidth(t *testing.T) {
	dr := &DirtyRegion{}
	dr.MarkRect(10, 10, 0, 5)

	assert.False(t, dr.dirty, "MarkRect with zero width should not mark dirty")
}

func TestDirtyRegion_MarkRect_ZeroHeight(t *testing.T) {
	dr := &DirtyRegion{}
	dr.MarkRect(10, 10, 5, 0)

	assert.False(t, dr.dirty, "MarkRect with zero height should not mark dirty")
}

func TestDirtyRegion_MarkRect_NegativeWidth(t *testing.T) {
	dr := &DirtyRegion{}
	dr.MarkRect(10, 10, -5, 5)

	assert.False(t, dr.dirty, "MarkRect with negative width should not mark dirty")
}

func TestDirtyRegion_MarkRect_NegativeHeight(t *testing.T) {
	dr := &DirtyRegion{}
	dr.MarkRect(10, 10, 5, -5)

	assert.False(t, dr.dirty, "MarkRect with negative height should not mark dirty")
}

func TestDirtyRegion_MarkRect_SingleCell(t *testing.T) {
	dr := &DirtyRegion{}
	dr.MarkRect(10, 10, 1, 1)

	assert.True(t, dr.dirty)
	assert.Equal(t, 10, dr.MinX)
	assert.Equal(t, 10, dr.MinY)
	assert.Equal(t, 10, dr.MaxX)
	assert.Equal(t, 10, dr.MaxY)
}

func TestDirtyRegion_MixMarkAndMarkRect(t *testing.T) {
	dr := &DirtyRegion{}
	dr.Mark(10, 10)
	dr.MarkRect(5, 5, 3, 3)

	assert.Equal(t, 5, dr.MinX)
	assert.Equal(t, 5, dr.MinY)
	assert.Equal(t, 10, dr.MaxX)
	assert.Equal(t, 10, dr.MaxY)
}

func TestDirtyRegion_MarkRect_OverlapAndExpand(t *testing.T) {
	dr := &DirtyRegion{}
	dr.MarkRect(10, 10, 10, 10) // (10,10) to (19,19)
	dr.MarkRect(15, 15, 10, 10) // (15,15) to (24,24)

	assert.Equal(t, 10, dr.MinX)
	assert.Equal(t, 10, dr.MinY)
	assert.Equal(t, 24, dr.MaxX)
	assert.Equal(t, 24, dr.MaxY)
}

func TestDirtyRegion_Bounds_AfterClear(t *testing.T) {
	dr := &DirtyRegion{}
	dr.Mark(10, 10)
	dr.Clear()
	dr.Mark(5, 5)

	assert.Equal(t, 5, dr.MinX)
	assert.Equal(t, 5, dr.MinY)
	assert.Equal(t, 5, dr.MaxX)
	assert.Equal(t, 5, dr.MaxY)
}
