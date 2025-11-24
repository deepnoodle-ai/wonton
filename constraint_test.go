package gooey

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMeasureWidget_Fallback(t *testing.T) {
	btn := NewComposableButton("Click Me", nil)
	// Default button has padding/borders usually, but let's just check constraints application

	c := SizeConstraints{MaxWidth: 4, MaxHeight: 1}
	size := MeasureWidget(btn, c)

	// Even if button wants to be larger, MeasureWidget should clamp it (via ApplyConstraints default behavior)
	assert.LessOrEqual(t, size.X, 4)
	assert.LessOrEqual(t, size.Y, 1)
}

func TestWrappingLabel_Measure(t *testing.T) {
	text := "Hello World"
	wl := NewWrappingLabel(text)

	// Unconstrained
	c1 := SizeConstraints{}
	s1 := wl.Measure(c1)
	assert.Equal(t, 11, s1.X) // "Hello World" length
	assert.Equal(t, 1, s1.Y)

	// Constrained width -> should wrap
	c2 := SizeConstraints{MaxWidth: 5}
	s2 := wl.Measure(c2)
	assert.LessOrEqual(t, s2.X, 5)
	assert.Greater(t, s2.Y, 1) // Should be at least 2 lines ("Hello", "World")
}

func TestVBoxLayout_Measure(t *testing.T) {
	vb := NewVBoxLayout(0) // 0 spacing
	c := NewContainer(vb)

	l1 := NewWrappingLabel("Hello")
	l2 := NewWrappingLabel("World")
	c.AddChild(l1)
	c.AddChild(l2)

	constraints := SizeConstraints{MaxWidth: 100}
	size := c.Measure(constraints)

	// "Hello" (5) and "World" (5)
	assert.Equal(t, 5, size.X)
	assert.Equal(t, 2, size.Y) // Sum of heights (1 + 1)

	// Constrain width to force wrap
	l3 := NewWrappingLabel("Long Text Here")
	c.Clear()
	c.AddChild(l3)

	c3 := SizeConstraints{MaxWidth: 5}
	s3 := c.Measure(c3)
	assert.LessOrEqual(t, s3.X, 5)
	assert.Greater(t, s3.Y, 1) // Should wrap
}

func TestHBoxLayout_Measure(t *testing.T) {
	hb := NewHBoxLayout(0)
	c := NewContainer(hb)

	l1 := NewWrappingLabel("A")
	l2 := NewWrappingLabel("B")
	c.AddChild(l1)
	c.AddChild(l2)

	constraints := SizeConstraints{MaxWidth: 100}
	size := c.Measure(constraints)

	assert.Equal(t, 2, size.X) // 1 + 1
	assert.Equal(t, 1, size.Y) // Max height
}
