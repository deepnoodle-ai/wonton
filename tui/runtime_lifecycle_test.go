package tui

import (
	"bytes"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/deepnoodle-ai/wonton/assert"
)

// testRuntimeModel is a simple model for testing runtime lifecycle
type testRuntimeModel struct {
	executed    atomic.Value
	counter     atomic.Value
	quitOnCount int
}

func (m *testRuntimeModel) HandleEvent(event Event) []Cmd {
	switch e := event.(type) {
	case incrementEvent:
		i := m.counter.Load()
		if i == nil {
			m.counter.Store(1)
		} else {
			m.counter.Store(i.(int) + 1)
		}
		current := m.counter.Load().(int)
		if m.quitOnCount > 0 && current >= m.quitOnCount {
			return []Cmd{Quit()}
		}
	case KeyEvent:
		if e.Rune == 'q' {
			return []Cmd{Quit()}
		}
	case panicEvent:
		panic("testing panic behavior")
	}
	return nil
}

func (m *testRuntimeModel) Render(frame RenderFrame) {
	m.executed.Store(true)
	width, height := frame.Size()
	if width > 0 && height > 0 {
		frame.PrintStyled(0, 0, "test", NewStyle())
	}
}

type incrementEvent struct{}

func (e incrementEvent) Timestamp() time.Time {
	return time.Now()
}

type panicEvent struct{}

func (e panicEvent) Timestamp() time.Time {
	return time.Now()
}

// TestRuntimeQuit tests that the runtime properly quits when Quit command is returned
func TestRuntimeQuit(t *testing.T) {
	var buf bytes.Buffer
	terminal := NewTestTerminal(80, 24, &buf)

	model := &testRuntimeModel{quitOnCount: 1}
	runtime := NewRuntime(terminal, model, 30)

	// Send an increment event in a goroutine to trigger quit
	go func() {
		time.Sleep(50 * time.Millisecond)
		runtime.SendEvent(incrementEvent{})
	}()

	// Run should return without error
	err := runtime.Run()
	assert.NoError(t, err)

	// Verify the model was executed
	assert.NotNil(t, model.executed.Load())
	assert.Equal(t, 1, model.counter.Load())
}

// TestRuntimeSendEvent tests that SendEvent works correctly
func TestRuntimeSendEvent(t *testing.T) {
	var buf bytes.Buffer
	terminal := NewTestTerminal(80, 24, &buf)

	model := &testRuntimeModel{quitOnCount: 3}
	runtime := NewRuntime(terminal, model, 30)

	// Send multiple increment events
	go func() {
		time.Sleep(50 * time.Millisecond)
		runtime.SendEvent(incrementEvent{})
		runtime.SendEvent(incrementEvent{})
		runtime.SendEvent(incrementEvent{})
	}()

	err := runtime.Run()
	assert.NoError(t, err)

	// Verify all events were processed
	assert.Equal(t, 3, model.counter.Load())
}

// TestRuntimePanic tests that the runtime handles panics
// Currently, panics in HandleEvent will crash the runtime (expected behavior)
// This test documents this behavior and can be updated when panic recovery is added
func TestRuntimePanic(t *testing.T) {
	t.Skip("Panics in HandleEvent currently crash the runtime - this is expected behavior")

	// When panic handling is implemented, this test can be enabled:
	// var buf bytes.Buffer
	// terminal := NewTestTerminal(80, 24, &buf)
	//
	// model := &testRuntimeModel{}
	// runtime := NewRuntime(terminal, model, 30)
	//
	// // Send a panic event
	// go func() {
	// 	for {
	// 		time.Sleep(1 * time.Millisecond)
	// 		if model.executed.Load() != nil {
	// 			runtime.SendEvent(panicEvent{})
	// 			return
	// 		}
	// 	}
	// }()
	//
	// err := runtime.Run()
	// assert.Error(t, err, "should return error on panic")
}

// TestRuntimeMultipleQuits tests that multiple quit commands don't cause issues
func TestRuntimeMultipleQuits(t *testing.T) {
	var buf bytes.Buffer
	terminal := NewTestTerminal(80, 24, &buf)

	model := &testRuntimeModel{}
	runtime := NewRuntime(terminal, model, 30)

	// Send multiple quit events
	go func() {
		time.Sleep(50 * time.Millisecond)
		for i := 0; i < 5; i++ {
			runtime.SendEvent(KeyEvent{Rune: 'q'})
		}
	}()

	err := runtime.Run()
	assert.NoError(t, err)
}

// TestRuntimeConcurrentEvents tests that events can be sent concurrently
func TestRuntimeConcurrentEvents(t *testing.T) {
	var buf bytes.Buffer
	terminal := NewTestTerminal(80, 24, &buf)

	model := &testRuntimeModel{quitOnCount: 10}
	runtime := NewRuntime(terminal, model, 30)

	// Send events from multiple goroutines
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			time.Sleep(50 * time.Millisecond)
			runtime.SendEvent(incrementEvent{})
			runtime.SendEvent(incrementEvent{})
		}()
	}

	// Wait for all goroutines to send their events
	go func() {
		wg.Wait()
	}()

	err := runtime.Run()
	assert.NoError(t, err)

	// Verify we got all events
	assert.Equal(t, 10, model.counter.Load())
}

// TestRuntimeEventOrdering tests that events are processed in order
func TestRuntimeEventOrdering(t *testing.T) {
	var buf bytes.Buffer
	terminal := NewTestTerminal(80, 24, &buf)

	type orderTracker struct {
		executed atomic.Value
		order    []int
		mu       sync.Mutex
	}

	tracker := &orderTracker{
		order: make([]int, 0),
	}

	app := &struct {
		tracker *orderTracker
	}{tracker: tracker}

	handleEvent := func(event Event) []Cmd {
		if e, ok := event.(orderedEvent); ok {
			app.tracker.mu.Lock()
			app.tracker.order = append(app.tracker.order, e.num)
			if e.num >= 5 {
				app.tracker.mu.Unlock()
				return []Cmd{Quit()}
			}
			app.tracker.mu.Unlock()
		}
		return nil
	}

	render := func(frame RenderFrame) {
		app.tracker.executed.Store(true)
	}

	// Create a simple application using anonymous struct
	model := &simpleApp{
		handleFunc: handleEvent,
		renderFunc: render,
	}

	runtime := NewRuntime(terminal, model, 30)

	// Send events in order
	go func() {
		time.Sleep(50 * time.Millisecond)
		for i := 1; i <= 5; i++ {
			runtime.SendEvent(orderedEvent{num: i})
		}
	}()

	err := runtime.Run()
	assert.NoError(t, err)

	// Verify events were processed in order
	tracker.mu.Lock()
	assert.Equal(t, []int{1, 2, 3, 4, 5}, tracker.order)
	tracker.mu.Unlock()
}

type orderedEvent struct {
	num int
}

func (e orderedEvent) Timestamp() time.Time {
	return time.Now()
}

type simpleApp struct {
	handleFunc func(Event) []Cmd
	renderFunc func(RenderFrame)
}

func (a *simpleApp) HandleEvent(event Event) []Cmd {
	return a.handleFunc(event)
}

func (a *simpleApp) Render(frame RenderFrame) {
	a.renderFunc(frame)
}

// TestRuntimeFPS tests that the runtime respects FPS setting
func TestRuntimeFPS(t *testing.T) {
	var buf bytes.Buffer
	terminal := NewTestTerminal(80, 24, &buf)

	type fpsTracker struct {
		executed  atomic.Value
		tickCount atomic.Value
	}

	tracker := &fpsTracker{}

	app := &simpleApp{
		handleFunc: func(event Event) []Cmd {
			if _, ok := event.(TickEvent); ok {
				count := tracker.tickCount.Load()
				if count == nil {
					tracker.tickCount.Store(1)
				} else {
					tracker.tickCount.Store(count.(int) + 1)
				}
				if count != nil && count.(int) >= 5 {
					return []Cmd{Quit()}
				}
			}
			return nil
		},
		renderFunc: func(frame RenderFrame) {
			tracker.executed.Store(true)
		},
	}

	runtime := NewRuntime(terminal, app, 60)

	start := time.Now()
	err := runtime.Run()
	elapsed := time.Since(start)

	assert.NoError(t, err)

	// With 60 FPS and 5 ticks, should take roughly 5/60 = 83ms
	// Allow for some variance
	assert.True(t, elapsed < 200*time.Millisecond, "Runtime took too long: %v", elapsed)
}
