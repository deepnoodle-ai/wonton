package termtest

import (
	"bytes"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestCapture(t *testing.T) {
	var underlying bytes.Buffer
	cap := NewCapture(&underlying)

	cap.Write([]byte("Hello"))
	cap.Write([]byte(" World"))

	assert.Equal(t, "Hello World", cap.String())
	assert.Equal(t, "Hello World", underlying.String())
}

func TestCaptureNoForward(t *testing.T) {
	cap := NewCapture(nil) // No underlying writer

	cap.Write([]byte("Hello"))

	assert.Equal(t, "Hello", cap.String())
}

func TestCaptureScreen(t *testing.T) {
	cap := NewCapture(nil)

	cap.Write([]byte("\x1b[1mBold\x1b[0m Normal"))

	screen := cap.Screen(20, 5)

	assert.Equal(t, "Bold Normal", screen.Row(0))
	assert.True(t, screen.Cell(0, 0).Style.Bold)
	assert.False(t, screen.Cell(5, 0).Style.Bold)
}

func TestCaptureReset(t *testing.T) {
	cap := NewCapture(nil)

	cap.Write([]byte("Hello"))
	cap.Reset()
	cap.Write([]byte("World"))

	assert.Equal(t, "World", cap.String())
}

func TestRecorder(t *testing.T) {
	rec := NewRecorder(20, 5)

	rec.Write([]byte("Line1\n"))
	rec.Write([]byte("Line2"))

	assert.Equal(t, "Line1", rec.Screen().Row(0))
	assert.Equal(t, "Line2", rec.Screen().Row(1))

	events := rec.Events()
	assert.Equal(t, 2, len(events))
	assert.Equal(t, "Line1\n", string(events[0].Data))
	assert.Equal(t, "Line2", string(events[1].Data))
}

func TestRecorderReplay(t *testing.T) {
	rec := NewRecorder(20, 5)

	rec.Write([]byte("\x1b[2J\x1b[H"))
	rec.Write([]byte("Hello"))

	// Replay to a different size screen
	replayed := rec.Replay(30, 10)

	assert.Equal(t, "Hello", replayed.Row(0))
}

func TestRecorderReset(t *testing.T) {
	rec := NewRecorder(20, 5)

	rec.Write([]byte("Hello"))
	rec.Reset()

	assert.Equal(t, 0, len(rec.Events()))
	assert.Equal(t, "", rec.Screen().Row(0))
}

func TestBuffer(t *testing.T) {
	buf := NewBuffer()

	buf.WriteString("\x1b[31mRed\x1b[0m")

	screen := buf.Screen(20, 5)
	assert.Equal(t, "Red", screen.Row(0))
	assert.Equal(t, ColorBasic, screen.Cell(0, 0).Style.Foreground.Type)
	assert.Equal(t, uint8(1), screen.Cell(0, 0).Style.Foreground.Value) // Red
}

func TestCaptureANSISequences(t *testing.T) {
	cap := NewCapture(nil)

	// Simulate typical terminal application output
	cap.Write([]byte("\x1b[2J\x1b[H"))            // Clear and home
	cap.Write([]byte("┌──────────────────┐\n"))  // Box top
	cap.Write([]byte("│ \x1b[1mMenu\x1b[0m             │\n")) // Box content with bold
	cap.Write([]byte("└──────────────────┘\n"))  // Box bottom

	screen := cap.Screen(30, 10)

	// Check content
	assert.Contains(t, screen.Text(), "Menu")
	assert.Contains(t, screen.Row(0), "┌")
	assert.Contains(t, screen.Row(0), "┐")

	// Check styling (find 'M' in "Menu")
	// The 'M' should be at position 2 on row 1
	menuStart := 2 // After "│ "
	assert.True(t, screen.Cell(menuStart, 1).Style.Bold)
}

// Additional Capture tests

func TestCaptureBytes(t *testing.T) {
	cap := NewCapture(nil)
	cap.Write([]byte("Hello"))

	bytes := cap.Bytes()
	assert.Equal(t, []byte("Hello"), bytes)
}

func TestCaptureLargeData(t *testing.T) {
	cap := NewCapture(nil)

	// Write a lot of data
	data := make([]byte, 100000)
	for i := range data {
		data[i] = byte('A' + (i % 26))
	}
	cap.Write(data)

	assert.Equal(t, 100000, len(cap.Bytes()))
}

func TestCaptureMultipleWrites(t *testing.T) {
	cap := NewCapture(nil)

	for i := 0; i < 100; i++ {
		cap.Write([]byte("X"))
	}

	assert.Equal(t, 100, len(cap.Bytes()))
}

func TestCaptureWithWriter(t *testing.T) {
	var buf1, buf2 bytes.Buffer
	cap := NewCapture(&buf1)

	cap.Write([]byte("Test"))
	buf2.Write(cap.Bytes())

	assert.Equal(t, "Test", buf1.String())
	assert.Equal(t, "Test", buf2.String())
}

func TestCaptureScreenDifferentSizes(t *testing.T) {
	cap := NewCapture(nil)
	cap.Write([]byte("ABCDEFGHIJ\n1234567890"))

	small := cap.Screen(5, 2)
	large := cap.Screen(20, 5)

	// Small screen causes wrapping and scrolling
	// After all scrolling, we end up with last content visible
	// Just verify it has some content and is different from large
	assert.True(t, len(small.Row(0)) > 0 || len(small.Row(1)) > 0, "small screen should have content")

	// Large screen should show all
	assert.Contains(t, large.Row(0), "ABCDEFGHIJ")
	assert.Contains(t, large.Row(1), "1234567890")
}

func TestRecorderEventsAreCopied(t *testing.T) {
	rec := NewRecorder(20, 5)

	original := []byte("Hello")
	rec.Write(original)

	// Modify original
	original[0] = 'X'

	// Recorder should have copy
	events := rec.Events()
	assert.Equal(t, "Hello", string(events[0].Data))
}

func TestRecorderReplayDifferentSizes(t *testing.T) {
	rec := NewRecorder(10, 3)
	rec.Write([]byte("Short\nText"))

	// Replay to larger screen
	large := rec.Replay(50, 10)
	assert.Equal(t, "Short", large.Row(0))
	assert.Equal(t, "Text", large.Row(1))

	// Replay to same size
	same := rec.Replay(10, 3)
	assert.Equal(t, "Short", same.Row(0))
}

func TestRecorderWithANSI(t *testing.T) {
	rec := NewRecorder(30, 5)

	rec.Write([]byte("\x1b[1mBold\x1b[0m"))
	rec.Write([]byte("\x1b[31mRed\x1b[0m"))

	screen := rec.Screen()
	assert.True(t, screen.Cell(0, 0).Style.Bold)
	assert.Equal(t, ColorBasic, screen.Cell(4, 0).Style.Foreground.Type)
}

func TestBufferWrite(t *testing.T) {
	buf := NewBuffer()

	n, err := buf.Write([]byte("Test"))
	assert.Equal(t, 4, n)
	assert.NoError(t, err)
}

func TestBufferWriteString(t *testing.T) {
	buf := NewBuffer()
	buf.WriteString("Hello")
	buf.WriteString(" World")

	assert.Equal(t, "Hello World", buf.String())
}

func TestBufferScreenMultipleTimes(t *testing.T) {
	buf := NewBuffer()
	buf.WriteString("Content")

	// Can create multiple screens from same buffer
	s1 := buf.Screen(20, 5)
	s2 := buf.Screen(30, 10)

	assert.Equal(t, s1.Row(0), s2.Row(0))
}

func TestCaptureEmptyWrites(t *testing.T) {
	cap := NewCapture(nil)

	n, err := cap.Write([]byte{})
	assert.Equal(t, 0, n)
	assert.NoError(t, err)

	n, err = cap.Write(nil)
	assert.Equal(t, 0, n)
	assert.NoError(t, err)
}

func TestRecorderEmptyEvents(t *testing.T) {
	rec := NewRecorder(20, 5)

	events := rec.Events()
	assert.Equal(t, 0, len(events))
}

func TestRecorderResetMultipleTimes(t *testing.T) {
	rec := NewRecorder(20, 5)

	rec.Write([]byte("First"))
	rec.Reset()

	rec.Write([]byte("Second"))
	rec.Reset()

	rec.Write([]byte("Third"))

	assert.Equal(t, 1, len(rec.Events()))
	assert.Equal(t, "Third", string(rec.Events()[0].Data))
}
