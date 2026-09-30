package tui

import (
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/deepnoodle-ai/wonton/internal/terminalinput"
	"github.com/deepnoodle-ai/wonton/terminal"
)

// managedInput owns one decoder for the runner's lifetime. Its processor can
// divert a finite pre-release stream even when the application queue is full.
// A boundary packet acknowledges both OS-read quiescence and neutral framing.
type managedInput struct {
	owner        managedInputOwner
	decoder      *terminal.KeyDecoder
	events       chan Event
	done         <-chan struct{}
	backslash    bool
	mouse        func(Event) (Event, Event)
	divert       func(Event) bool
	capturePanic func()
	mu           sync.Mutex
	cycle        *inputBoundary
	wake         chan struct{}
	pending      []Event // processor-owned; never read by the event loop
}
type managedInputOwner interface {
	io.Reader
	Pause() error
	Resume() error
	DiscardResidual() error
	Stop()
	Close() error
}
type inputBoundary struct{ ack, resume chan struct{} }
type decodedInput struct {
	event    Event
	err      error
	boundary *inputBoundary
}

func newManagedInput(owner managedInputOwner, events chan Event, done <-chan struct{}, tabWidth int, backslash bool, mouse func(Event) (Event, Event), divert func(Event) bool) *managedInput {
	decoder := terminal.NewKeyDecoder(owner)
	decoder.SetPasteTabWidth(tabWidth)
	return &managedInput{owner: owner, decoder: decoder, events: events, done: done, backslash: backslash, mouse: mouse, divert: divert, wake: make(chan struct{})}
}

func (m *managedInput) pause() error {
	m.mu.Lock()
	if m.cycle != nil {
		m.mu.Unlock()
		return errors.New("input already paused")
	}
	cycle := &inputBoundary{ack: make(chan struct{}), resume: make(chan struct{})}
	m.cycle = cycle
	close(m.wake)
	m.mu.Unlock()
	if err := m.owner.Pause(); err != nil {
		return fmt.Errorf("pause terminal reads: %w", err)
	}
	select {
	case <-cycle.ack:
		return nil
	case <-m.done:
		return errors.New("runner stopped before input acknowledgment")
	}
}
func (m *managedInput) resume() error {
	if err := m.owner.Resume(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cycle == nil {
		return errors.New("input is not paused")
	}
	cycle := m.cycle
	m.cycle = nil
	m.wake = make(chan struct{})
	close(cycle.resume)
	return nil
}

func (m *managedInput) run() {
	packets := make(chan decodedInput, 1)
	decoderDone := make(chan struct{})
	go func() {
		defer close(decoderDone)
		if m.capturePanic != nil {
			defer m.capturePanic()
		}
		for {
			event, err := m.decoder.ReadEvent()
			packet := decodedInput{event: event, err: err}
			if errors.Is(err, terminalinput.Boundary) {
				m.mu.Lock()
				packet.boundary = m.cycle
				m.mu.Unlock()
			}
			select {
			case packets <- packet:
			case <-m.done:
				return
			}
			if packet.boundary != nil {
				select {
				case <-packet.boundary.resume:
				case <-m.done:
					return
				}
				continue
			}
			if err != nil {
				return
			}
		}
	}()
	defer func() { m.owner.Stop(); <-decoderDone; _ = m.owner.Close() }()
	var held *KeyEvent
	var timer *time.Timer
	var timerC <-chan time.Time
	clearTimer := func() {
		if timer != nil {
			timer.Stop()
		}
		timerC = nil
	}
	defer clearTimer()
	emit := func(event Event) bool {
		if event == nil {
			return true
		}
		if key, ok := event.(KeyEvent); ok && key.Key == KeyUnknown && key.Rune == 0 && key.Paste == "" {
			return true
		}
		if m.mouse != nil {
			original, click := m.mouse(event)
			if click != nil && !m.forward(click) {
				return false
			}
			event = original
		}
		return m.forward(event)
	}
	flushHeld := func() bool {
		if held == nil {
			return true
		}
		event := *held
		held = nil
		clearTimer()
		return emit(event)
	}
	for {
		select {
		case <-m.done:
			return
		case <-timerC:
			if !flushHeld() {
				return
			}
		case packet := <-packets:
			if packet.event != nil {
				event := packet.event
				if held != nil {
					if key, ok := event.(KeyEvent); ok && key.Key == KeyEnter {
						held = nil
						clearTimer()
						event = KeyEvent{Key: KeyEnter, Shift: true}
					} else if !flushHeld() {
						return
					}
				}
				if key, ok := event.(KeyEvent); ok && m.backslash && key.Rune == '\\' && key.Key == KeyUnknown && key.Paste == "" {
					held = &key
					timer = time.NewTimer(100 * time.Millisecond)
					timerC = timer.C
				} else if !emit(event) {
					return
				}
			}
			if packet.boundary != nil {
				if !flushHeld() {
					return
				}
				close(packet.boundary.ack)
				select {
				case <-packet.boundary.resume:
				case <-m.done:
					return
				}
				retained := m.pending
				m.pending = nil
				for _, event := range retained {
					if !m.forward(event) {
						return
					}
				}
			} else if packet.err != nil {
				m.forward(ErrorEvent{Time: time.Now(), Err: packet.err, Cause: "input reader"})
				return
			}
		}
	}
}

func (m *managedInput) forward(event Event) bool {
	if m.divert != nil && m.divert(event) {
		return true
	}
	m.mu.Lock()
	wake := m.wake
	m.mu.Unlock()
	// Prefer diversion once release starts, even if the queue becomes writable.
	select {
	case <-wake:
		m.pending = append(m.pending, event)
		return true
	default:
	}
	select {
	case m.events <- event:
		return true
	case <-wake:
		m.pending = append(m.pending, event)
		return true
	case <-m.done:
		return false
	}
}
