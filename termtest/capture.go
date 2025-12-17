package termtest

import (
	"bytes"
	"io"
)

// Capture is an io.Writer that captures terminal output for testing.
// It records all written bytes while optionally forwarding them to another writer.
// Use Screen() to convert the captured output into a virtual terminal screen.
//
// Example:
//
//	capture := termtest.NewCapture(nil)
//	app.WriteTo(capture)  // App writes terminal output
//	screen := capture.Screen(80, 24)
//	termtest.AssertContains(t, screen, "Success")
type Capture struct {
	writer io.Writer
	buffer bytes.Buffer
}

// NewCapture creates a new Capture that records all writes.
// If writer is non-nil, writes are also forwarded to it.
// Pass nil to only capture without forwarding.
//
// Example with forwarding to stdout:
//
//	capture := termtest.NewCapture(os.Stdout)  // See and capture output
//	app.Run(capture)
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

// Screen creates a new Screen with the given dimensions and processes all
// captured output through it. This interprets ANSI sequences and returns
// the resulting screen state.
//
// You can call this multiple times with different dimensions to see how
// the output would look on different terminal sizes.
func (c *Capture) Screen(width, height int) *Screen {
	screen := NewScreen(width, height)
	screen.Write(c.Bytes())
	return screen
}

// Recorder captures terminal output as a sequence of write events.
// Unlike Capture, it maintains the order of writes, allowing you to
// replay them or inspect individual events. Useful for debugging complex
// terminal output or creating test recordings.
type Recorder struct {
	events []RecordedEvent
	screen *Screen
}

// RecordedEvent represents a single write operation to the terminal.
// Each event captures the exact bytes written.
type RecordedEvent struct {
	Data []byte // The bytes written in this event
}

// NewRecorder creates a new Recorder with an internal screen of the given size.
// All writes are recorded as events and also processed through the screen.
//
// Example:
//
//	recorder := termtest.NewRecorder(80, 24)
//	app.Run(recorder)
//	for i, event := range recorder.Events() {
//	    fmt.Printf("Event %d: %q\n", i, event.Data)
//	}
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

// Replay creates a new screen with the given dimensions and replays all
// recorded events to it. This lets you see how the same output would look
// on a different terminal size.
func (r *Recorder) Replay(width, height int) *Screen {
	screen := NewScreen(width, height)
	for _, event := range r.events {
		screen.Write(event.Data)
	}
	return screen
}

// Buffer is a convenience wrapper around bytes.Buffer that adds a Screen() method.
// Use this when you want simple buffering with easy conversion to a terminal screen.
type Buffer struct {
	bytes.Buffer
}

// NewBuffer creates a new empty Buffer.
func NewBuffer() *Buffer {
	return &Buffer{}
}

// Screen creates a new Screen and processes all buffered content through it.
func (b *Buffer) Screen(width, height int) *Screen {
	screen := NewScreen(width, height)
	screen.Write(b.Bytes())
	return screen
}
