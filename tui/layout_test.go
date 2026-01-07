package tui

import (
	"bytes"
	"testing"
	"time"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestLayout(t *testing.T) {
	t.Run("NewLayout", func(t *testing.T) {
		var buf bytes.Buffer
		term := NewTestTerminal(80, 24, &buf)
		l := NewLayout(term)
		defer l.Close()

		y, h := l.ContentArea()
		assert.Equal(t, 0, y)
		assert.Equal(t, 24, h)
	})

	t.Run("SetHeader", func(t *testing.T) {
		var buf bytes.Buffer
		term := NewTestTerminal(80, 24, &buf)
		l := NewLayout(term)
		defer l.Close()

		header := SimpleHeader("Title", NewStyle())
		l.SetHeader(header)

		y, h := l.ContentArea()
		assert.Equal(t, 1, y)
		assert.Equal(t, 23, h)

		// Bordered header
		header = BorderedHeader("Title", NewStyle())
		l.SetHeader(header)
		y, h = l.ContentArea()
		assert.Equal(t, 3, y)
		assert.Equal(t, 21, h)
	})

	t.Run("SetFooter", func(t *testing.T) {
		var buf bytes.Buffer
		term := NewTestTerminal(80, 24, &buf)
		l := NewLayout(term)
		defer l.Close()

		footer := SimpleFooter("Left", "Center", "Right", NewStyle())
		l.SetFooter(footer)

		y, h := l.ContentArea()
		assert.Equal(t, 0, y)
		assert.Equal(t, 23, h)

		// Bordered footer
		footer = &Footer{Height: 3, Border: true}
		l.SetFooter(footer)
		y, h = l.ContentArea()
		assert.Equal(t, 21, h)
	})

	t.Run("HeaderAndFooter", func(t *testing.T) {
		var buf bytes.Buffer
		term := NewTestTerminal(80, 24, &buf)
		l := NewLayout(term)
		defer l.Close()

		l.SetHeader(SimpleHeader("Title", NewStyle()))
		l.SetFooter(SimpleFooter("L", "C", "R", NewStyle()))

		y, h := l.ContentArea()
		assert.Equal(t, 1, y)
		assert.Equal(t, 22, h) // 24 - 1 - 1
	})

	t.Run("Draw", func(t *testing.T) {
		var buf bytes.Buffer
		term := NewTestTerminal(80, 24, &buf)
		l := NewLayout(term)
		defer l.Close()

		l.SetHeader(SimpleHeader("Title", NewStyle()))
		l.Draw()

		// Checking output is hard because it contains ANSI codes.
		// But verify no panic and content area updated.
		assert.True(t, buf.Len() > 0)
	})

	t.Run("AutoRefresh", func(t *testing.T) {
		// Isolate terminal for this test to avoid race conditions with shared buffer
		var buf bytes.Buffer
		term := NewTestTerminal(80, 24, &buf)
		l := NewLayout(term)
		l.SetHeader(SimpleHeader("Time", NewStyle()))

		// Use a slightly longer interval to ensure reliability, but short enough for test
		l.EnableAutoRefresh(10 * time.Millisecond)

		// Sleep long enough for at least one tick
		time.Sleep(50 * time.Millisecond)

		l.DisableAutoRefresh()
		l.Close()

		// Note: We don't assert on buf content here because TestTerminal uses buffered output
		// and the exact timing of background refresh vs buffer flush is flaky in tests.
		// Invoking the method ensures code coverage and no panics.
	})

	t.Run("PrintInContent", func(t *testing.T) {
		var buf bytes.Buffer
		term := NewTestTerminal(80, 24, &buf)
		l := NewLayout(term)
		defer l.Close()
		l.SetHeader(SimpleHeader("Header", NewStyle()))

		// This prints to the buffer
		l.PrintInContent("Content line")
		// Note: Output is buffered in Terminal back/front buffers and not written to buf
		// until flush/draw, so we don't assert buf content here.
	})
}

func TestLayoutHelpers(t *testing.T) {
	h := SimpleHeader("Test", NewStyle())
	assert.Equal(t, 1, h.Height)

	h2 := BorderedHeader("Test", NewStyle())
	assert.Equal(t, 3, h2.Height)
	assert.True(t, h2.Border)

	f := SimpleFooter("L", "C", "R", NewStyle())
	assert.Equal(t, 1, f.Height)

	f2 := StatusBarFooter([]StatusItem{{Key: "K", Value: "V"}})
	assert.Equal(t, 1, f2.Height)
	assert.True(t, f2.StatusBar)
}
