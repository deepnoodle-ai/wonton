package tui

import (
	"image"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

// MockLayoutWidget is a simple widget for testing layouts
type MockLayoutWidget struct {
	*BaseWidget
	drawCalled bool
}

func NewMockLayoutWidget(width, height int) *MockLayoutWidget {
	bw := NewBaseWidget()
	w := &MockLayoutWidget{
		BaseWidget: &bw,
	}
	w.SetPreferredSize(image.Point{X: width, Y: height})
	w.SetMinSize(image.Point{X: width / 2, Y: height / 2})
	return w
}

func (m *MockLayoutWidget) Draw(frame RenderFrame) {
	m.drawCalled = true
}

func (m *MockLayoutWidget) HandleKey(event KeyEvent) bool {
	return false
}

// Ensure MockLayoutWidget implements ComposableWidget
var _ ComposableWidget = &MockLayoutWidget{}

func TestVBoxLayout(t *testing.T) {
	t.Run("Basic Stacking", func(t *testing.T) {
		layout := NewVBoxLayout(10) // 10px spacing

		w1 := NewMockLayoutWidget(100, 20)
		w2 := NewMockLayoutWidget(100, 30)
		w3 := NewMockLayoutWidget(100, 40)

		children := []ComposableWidget{w1, w2, w3}
		container := image.Rect(0, 0, 200, 200)

		layout.Layout(container, children)

		b1 := w1.GetBounds()
		assert.Equal(t, 0, b1.Min.Y)
		assert.Equal(t, 20, b1.Max.Y)

		b2 := w2.GetBounds()
		expectedY2 := 20 + 10 // height1 + spacing
		assert.Equal(t, expectedY2, b2.Min.Y)
		assert.Equal(t, expectedY2+30, b2.Max.Y)

		b3 := w3.GetBounds()
		expectedY3 := expectedY2 + 30 + 10 // y2 + height2 + spacing
		assert.Equal(t, expectedY3, b3.Min.Y)
		assert.Equal(t, expectedY3+40, b3.Max.Y)
	})

	t.Run("Alignment Center", func(t *testing.T) {
		layout := NewVBoxLayout(0).WithAlignment(LayoutAlignCenter)
		w := NewMockLayoutWidget(50, 20)
		container := image.Rect(0, 0, 200, 100)

		layout.Layout(container, []ComposableWidget{w})

		b := w.GetBounds()
		expectedX := (200 - 50) / 2 // 75
		assert.Equal(t, expectedX, b.Min.X)
	})

	t.Run("Alignment End", func(t *testing.T) {
		layout := NewVBoxLayout(0).WithAlignment(LayoutAlignEnd)
		w := NewMockLayoutWidget(50, 20)
		container := image.Rect(0, 0, 200, 100)

		layout.Layout(container, []ComposableWidget{w})

		b := w.GetBounds()
		expectedX := 200 - 50 // 150
		assert.Equal(t, expectedX, b.Min.X)
	})

	t.Run("Alignment Stretch", func(t *testing.T) {
		layout := NewVBoxLayout(0).WithAlignment(LayoutAlignStretch)
		w := NewMockLayoutWidget(50, 20)
		container := image.Rect(0, 0, 200, 100)

		layout.Layout(container, []ComposableWidget{w})

		b := w.GetBounds()
		assert.Equal(t, 200, b.Dx())
	})

	t.Run("CalculateSizes", func(t *testing.T) {
		layout := NewVBoxLayout(10)
		w1 := NewMockLayoutWidget(100, 20)
		w2 := NewMockLayoutWidget(120, 30) // Wider

		children := []ComposableWidget{w1, w2}

		pref := layout.CalculatePreferredSize(children)
		// Width: Max(100, 120) = 120
		// Height: 20 + 10 + 30 = 60
		assert.Equal(t, 120, pref.X)
		assert.Equal(t, 60, pref.Y)

		min := layout.CalculateMinSize(children)
		// Min sizes are half: 50,20 and 60,30 (height 20/2=10, 30/2=15)
		// w1 min: 50, 10
		// w2 min: 60, 15
		// Width: Max(50, 60) = 60
		// Height: 10 + 10 + 15 = 35
		assert.Equal(t, 60, min.X)
		assert.Equal(t, 35, min.Y)
	})

	t.Run("Distribution", func(t *testing.T) {
		layout := NewVBoxLayout(0).WithDistribute(true)
		w1 := NewMockLayoutWidget(100, 20)
		w2 := NewMockLayoutWidget(100, 20)

		// Set Grow params
		lp1 := w1.GetLayoutParams()
		lp1.Grow = 1
		w1.SetLayoutParams(lp1)

		lp2 := w2.GetLayoutParams()
		lp2.Grow = 1
		w2.SetLayoutParams(lp2)

		container := image.Rect(0, 0, 100, 100) // 100 height available
		// Required height: 20 + 20 = 40. Extra = 60.
		// Distributed: 30 each. Total height per widget = 20 + 30 = 50.

		layout.Layout(container, []ComposableWidget{w1, w2})

		assert.Equal(t, 50, w1.GetBounds().Dy())
		assert.Equal(t, 50, w2.GetBounds().Dy())
	})
}

func TestHBoxLayout(t *testing.T) {
	t.Run("Basic Stacking", func(t *testing.T) {
		layout := NewHBoxLayout(10) // 10px spacing

		w1 := NewMockLayoutWidget(20, 100)
		w2 := NewMockLayoutWidget(30, 100)
		w3 := NewMockLayoutWidget(40, 100)

		children := []ComposableWidget{w1, w2, w3}
		container := image.Rect(0, 0, 200, 200)

		layout.Layout(container, children)

		b1 := w1.GetBounds()
		assert.Equal(t, 0, b1.Min.X)
		assert.Equal(t, 20, b1.Max.X)

		b2 := w2.GetBounds()
		expectedX2 := 20 + 10
		assert.Equal(t, expectedX2, b2.Min.X)
		assert.Equal(t, expectedX2+30, b2.Max.X)
	})

	t.Run("Alignment Center", func(t *testing.T) {
		layout := NewHBoxLayout(0).WithAlignment(LayoutAlignCenter)
		w := NewMockLayoutWidget(20, 50)
		container := image.Rect(0, 0, 100, 200)

		layout.Layout(container, []ComposableWidget{w})

		b := w.GetBounds()
		expectedY := (200 - 50) / 2 // 75
		assert.Equal(t, expectedY, b.Min.Y)
	})

	t.Run("CalculateSizes", func(t *testing.T) {
		layout := NewHBoxLayout(10)
		w1 := NewMockLayoutWidget(20, 100)
		w2 := NewMockLayoutWidget(30, 120) // Taller

		children := []ComposableWidget{w1, w2}

		pref := layout.CalculatePreferredSize(children)
		// Width: 20 + 10 + 30 = 60
		// Height: Max(100, 120) = 120
		assert.Equal(t, 60, pref.X)
		assert.Equal(t, 120, pref.Y)
	})
}
