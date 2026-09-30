package tui

import (
	"bytes"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type heldStopHandler struct {
	entered, release chan struct{}
	calls, renders   atomic.Int32
}

func (a *heldStopHandler) HandleEvent(Event) []Cmd {
	if a.calls.Add(1) == 1 {
		close(a.entered)
		<-a.release
	}
	return nil
}
func (a *heldStopHandler) View() View     { a.renders.Add(1); return Text("running") }
func (a *heldStopHandler) LiveView() View { return a.View() }

// A stop request must not queue behind a synchronous handler or queued events.
// Cleanup may start only after the handler relinquishes terminal ownership.
func TestStopReturnsWithHeldHandlerAndFullEventQueue(t *testing.T) {
	for _, kind := range []string{"fullscreen", "inline"} {
		t.Run(kind, func(t *testing.T) {
			app := &heldStopHandler{entered: make(chan struct{}), release: make(chan struct{})}
			var stop func()
			var loop func()
			var events chan Event
			var done <-chan struct{}
			if kind == "fullscreen" {
				r := NewRuntime(NewTestTerminal(80, 24, &bytes.Buffer{}), app, 30)
				r.ticker = time.NewTicker(time.Hour)
				defer r.ticker.Stop()
				stop, loop, events, done = r.Stop, r.eventLoop, r.events, r.done
			} else {
				r := NewInlineApp(WithInlineOutput(&bytes.Buffer{}))
				r.app = app
				stop, loop, events, done = r.Stop, r.eventLoop, r.events, r.done
			}
			events <- KeyEvent{Rune: 'x'}
			loopDone := make(chan struct{})
			go func() { loop(); close(loopDone) }()
			<-app.entered
			for range cap(events) {
				events <- KeyEvent{Rune: 'y'}
			}
			stopped := make(chan struct{})
			go func() {
				var group sync.WaitGroup
				for range 16 {
					group.Go(stop)
				}
				group.Wait()
				close(stopped)
			}()
			select {
			case <-stopped:
			case <-time.After(time.Second):
				close(app.release)
				t.Fatal("Stop waited for full event queue")
			}
			select {
			case <-done:
				close(app.release)
				t.Fatal("shutdown started while handler retained ownership")
			default:
			}
			close(app.release)
			select {
			case <-loopDone:
			case <-time.After(time.Second):
				t.Fatal("event loop did not settle after handler return")
			}
			if app.calls.Load() != 1 || app.renders.Load() != 0 {
				t.Fatalf("shutdown dispatched %d handlers and %d repaints", app.calls.Load(), app.renders.Load())
			}
			stop()
		})
	}
}
