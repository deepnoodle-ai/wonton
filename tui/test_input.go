package tui

import (
	"io"
	"sync"
	"time"
)

// TestInput is an InputSource for tests: the test sends the events that a
// user would type. Give it to Run with WithInput, or to a Runtime with
// SetInputSource. It is safe to use from any goroutine.
//
// Events wait in a queue until the runtime reads them, so a test can send
// them before or after Run starts. Close ends the input. The runtime reads
// the events already sent, then gets io.EOF, and Run returns:
//
//	keys := tui.NewTestInput()
//	keys.Type("hello")
//	keys.Send(tui.KeyEvent{Key: tui.KeyEnter})
//	keys.Close()
//	err := tui.Run(app, tui.WithTerminal(term), tui.WithInput(keys))
type TestInput struct {
	mu     sync.Mutex
	ready  sync.Cond
	queue  []Event
	closed bool
}

// NewTestInput returns an empty TestInput.
func NewTestInput() *TestInput {
	in := &TestInput{}
	in.ready.L = &in.mu
	return in
}

// Send queues an event, such as a KeyEvent or a MouseEvent.
// Send panics after Close, as a send on a closed channel does.
func (in *TestInput) Send(events ...Event) {
	in.mu.Lock()
	defer in.mu.Unlock()
	if in.closed {
		panic("tui: TestInput.Send after Close")
	}
	in.queue = append(in.queue, events...)
	in.ready.Broadcast()
}

// Type queues one KeyEvent for each rune of s, as if the user typed s. For
// keys with no rune, such as Enter or the arrows, use Send with a KeyEvent
// that sets Key.
func (in *TestInput) Type(s string) {
	now := time.Now()
	events := make([]Event, 0, len(s))
	for _, r := range s {
		events = append(events, KeyEvent{Rune: r, Time: now})
	}
	in.Send(events...)
}

// Close ends the input. ReadEvent returns the events already sent, then
// io.EOF. Closing twice does nothing.
func (in *TestInput) Close() {
	in.mu.Lock()
	defer in.mu.Unlock()
	in.closed = true
	in.ready.Broadcast()
}

// ReadEvent returns the next event. It waits for one to be sent, and returns
// io.EOF once the input is closed and every event has been read.
func (in *TestInput) ReadEvent() (Event, error) {
	in.mu.Lock()
	defer in.mu.Unlock()
	for len(in.queue) == 0 && !in.closed {
		in.ready.Wait()
	}
	if len(in.queue) == 0 {
		return nil, io.EOF
	}
	event := in.queue[0]
	in.queue = in.queue[1:]
	return event, nil
}

// SetPasteTabWidth does nothing: a test sends pasted text as it wants it read.
func (in *TestInput) SetPasteTabWidth(int) {}
