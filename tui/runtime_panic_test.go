package tui

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"
)

// TestRuntimePanicInView verifies that a panic in the application's View()
// method propagates out of Run() (so callers' deferred terminal cleanup runs)
// with the original panic value and stack embedded in the message.
func TestRuntimePanicInView(t *testing.T) {
	var buf bytes.Buffer
	terminal := NewTestTerminal(80, 24, &buf)

	app := &simpleApp{
		handleFunc: func(event Event) []Cmd { return nil },
		renderFunc: func() View { panic("boom in view") },
	}

	runtime := NewRuntime(terminal, app, 30)

	defer func() {
		rec := recover()
		if rec == nil {
			t.Fatal("expected Run to re-panic after a View panic")
		}
		msg := fmt.Sprint(rec)
		if !strings.Contains(msg, "boom in view") {
			t.Errorf("re-panic message missing original value: %q", msg)
		}
		if !strings.Contains(msg, "original stack") {
			t.Errorf("re-panic message missing original stack: %q", msg)
		}
	}()

	// The initial ResizeEvent triggers a render, which panics in View().
	_ = runtime.Run()
	t.Fatal("Run returned normally; expected panic")
}

// TestRuntimePanicInHandleEvent verifies that a panic in HandleEvent
// propagates out of Run() with the original panic value.
func TestRuntimePanicInHandleEvent(t *testing.T) {
	var buf bytes.Buffer
	terminal := NewTestTerminal(80, 24, &buf)

	app := &simpleApp{
		handleFunc: func(event Event) []Cmd {
			if _, ok := event.(sequenceEvent); ok {
				panic("boom in handler")
			}
			return nil
		},
		renderFunc: func() View { return Text("ok") },
	}

	runtime := NewRuntime(terminal, app, 30)

	go func() {
		time.Sleep(20 * time.Millisecond)
		runtime.SendEvent(sequenceEvent{num: 1})
	}()

	defer func() {
		rec := recover()
		if rec == nil {
			t.Fatal("expected Run to re-panic after a HandleEvent panic")
		}
		if msg := fmt.Sprint(rec); !strings.Contains(msg, "boom in handler") {
			t.Errorf("re-panic message missing original value: %q", msg)
		}
	}()

	_ = runtime.Run()
	t.Fatal("Run returned normally; expected panic")
}

// TestRuntimePanicInCommand verifies that a panic inside an async Cmd
// goroutine shuts the runtime down and propagates out of Run().
func TestRuntimePanicInCommand(t *testing.T) {
	var buf bytes.Buffer
	terminal := NewTestTerminal(80, 24, &buf)

	app := &simpleApp{
		handleFunc: func(event Event) []Cmd {
			if _, ok := event.(sequenceEvent); ok {
				return []Cmd{func() Event { panic("boom in cmd") }}
			}
			return nil
		},
		renderFunc: func() View { return Text("ok") },
	}

	runtime := NewRuntime(terminal, app, 30)

	go func() {
		time.Sleep(20 * time.Millisecond)
		runtime.SendEvent(sequenceEvent{num: 1})
	}()

	defer func() {
		rec := recover()
		if rec == nil {
			t.Fatal("expected Run to re-panic after a Cmd panic")
		}
		if msg := fmt.Sprint(rec); !strings.Contains(msg, "boom in cmd") {
			t.Errorf("re-panic message missing original value: %q", msg)
		}
	}()

	_ = runtime.Run()
	t.Fatal("Run returned normally; expected panic")
}

// TestRuntimeNormalQuitNoPanic verifies the quit path is unaffected by the
// panic-capture machinery.
func TestRuntimeNormalQuitNoPanic(t *testing.T) {
	var buf bytes.Buffer
	terminal := NewTestTerminal(80, 24, &buf)

	app := &simpleApp{
		handleFunc: func(event Event) []Cmd {
			if _, ok := event.(sequenceEvent); ok {
				return []Cmd{Quit()}
			}
			return nil
		},
		renderFunc: func() View { return Text("ok") },
	}

	runtime := NewRuntime(terminal, app, 30)

	go func() {
		time.Sleep(20 * time.Millisecond)
		runtime.SendEvent(sequenceEvent{num: 1})
	}()

	if err := runtime.Run(); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
}
