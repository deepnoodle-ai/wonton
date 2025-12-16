package termtest

import (
	"bytes"
	"io"
)

// Capture wraps an io.Writer and captures all output while also forwarding
// it to the underlying writer. This is useful for capturing terminal output
// during tests.
type Capture struct {
	writer io.Writer
	buffer bytes.Buffer
}

// NewCapture creates a new Capture that forwards writes to the given writer.
// If writer is nil, writes are only captured (not forwarded).
func NewCapture(writer io.Writer) *Capture {
	return &Capture{writer: writer}
}

// Write implements io.Writer.
func (c *Capture) Write(p []byte) (n int, err error) {
	c.buffer.Write(p)
	if c.writer != nil {
		return c.writer.Write(p)
	}
	return len(p), nil
}

// Bytes returns all captured bytes.
func (c *Capture) Bytes() []byte {
	return c.buffer.Bytes()
}

// String returns all captured output as a string.
func (c *Capture) String() string {
	return c.buffer.String()
}

// Reset clears the captured output.
func (c *Capture) Reset() {
	c.buffer.Reset()
}

// Screen creates a new Screen and writes all captured output to it.
func (c *Capture) Screen(width, height int) *Screen {
	screen := NewScreen(width, height)
	screen.Write(c.Bytes())
	return screen
}

// Recorder captures terminal output with timing information.
// This can be useful for debugging test failures or creating recordings.
type Recorder struct {
	events []RecordedEvent
	screen *Screen
}

// RecordedEvent represents a single write event.
type RecordedEvent struct {
	Data []byte
}

// NewRecorder creates a new Recorder with the given screen dimensions.
func NewRecorder(width, height int) *Recorder {
	return &Recorder{
		screen: NewScreen(width, height),
	}
}

// Write implements io.Writer.
func (r *Recorder) Write(p []byte) (n int, err error) {
	data := make([]byte, len(p))
	copy(data, p)
	r.events = append(r.events, RecordedEvent{Data: data})
	r.screen.Write(p)
	return len(p), nil
}

// Screen returns the current screen state.
func (r *Recorder) Screen() *Screen {
	return r.screen
}

// Events returns all recorded events.
func (r *Recorder) Events() []RecordedEvent {
	return r.events
}

// Reset clears all recorded events and resets the screen.
func (r *Recorder) Reset() {
	r.events = nil
	r.screen.Clear()
}

// Replay replays all events to a new screen and returns it.
func (r *Recorder) Replay(width, height int) *Screen {
	screen := NewScreen(width, height)
	for _, event := range r.events {
		screen.Write(event.Data)
	}
	return screen
}

// Buffer is a simple bytes.Buffer wrapper that implements io.Writer
// and provides screen conversion.
type Buffer struct {
	bytes.Buffer
}

// NewBuffer creates a new Buffer.
func NewBuffer() *Buffer {
	return &Buffer{}
}

// Screen creates a new Screen and writes all buffer contents to it.
func (b *Buffer) Screen(width, height int) *Screen {
	screen := NewScreen(width, height)
	screen.Write(b.Bytes())
	return screen
}
