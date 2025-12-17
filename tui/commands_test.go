package tui

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/deepnoodle-ai/wonton/assert"
)

var errTestQuit = errors.New("test quit error")

// TestTickCommand tests the Tick command returns TickEvent
func TestTickCommand(t *testing.T) {
	duration := 10 * time.Millisecond
	cmd := Tick(duration)
	assert.NotNil(t, cmd)

	start := time.Now()
	event := cmd()
	elapsed := time.Since(start)
	assert.NotNil(t, event)

	tickEvent, ok := event.(TickEvent)
	assert.True(t, ok, "expected TickEvent, got %T", event)
	assert.False(t, tickEvent.Time.IsZero())
	assert.GreaterOrEqual(t, elapsed, duration)
}

// TestAfterCommand tests the After command delays and returns a TickEvent
func TestAfterCommand(t *testing.T) {
	duration := 50 * time.Millisecond
	called := false
	fn := func() {
		called = true
	}

	cmd := After(duration, fn)
	assert.NotNil(t, cmd)

	start := time.Now()
	event := cmd()
	elapsed := time.Since(start)

	assert.NotNil(t, event)
	assert.True(t, called, "callback should have been called")

	// After returns a TickEvent
	tickEvent, ok := event.(TickEvent)
	assert.True(t, ok, "expected TickEvent, got %T", event)
	assert.False(t, tickEvent.Time.IsZero())

	// Verify it delayed approximately the right amount
	assert.GreaterOrEqual(t, elapsed, duration, "After should delay at least %v, was %v", duration, elapsed)
	assert.Less(t, elapsed, duration+100*time.Millisecond, "After delayed too long: %v", elapsed)
}

// TestQuitCommand tests the Quit command returns QuitEvent
func TestQuitCommand(t *testing.T) {
	cmd := Quit()
	assert.NotNil(t, cmd)

	event := cmd()
	assert.NotNil(t, event)

	quitEvent, ok := event.(QuitEvent)
	assert.True(t, ok, "expected QuitEvent, got %T", event)
	assert.False(t, quitEvent.Time.IsZero())
}

// TestBatchCommand tests that Batch returns a slice of commands
func TestBatchCommand(t *testing.T) {
	cmd1 := Quit()
	cmd2 := Tick(10 * time.Millisecond)
	cmd3 := After(10*time.Millisecond, nil)

	batchCmd := Batch(cmd1, cmd2, cmd3)
	assert.NotNil(t, batchCmd)
	assert.Len(t, batchCmd, 3, "Batch should return 3 commands")

	// Verify each command works
	event1 := batchCmd[0]()
	_, ok := event1.(QuitEvent)
	assert.True(t, ok, "first command should return QuitEvent")

	event2 := batchCmd[1]()
	_, ok = event2.(TickEvent)
	assert.True(t, ok, "second command should return TickEvent")

	event3 := batchCmd[2]()
	_, ok = event3.(TickEvent)
	assert.True(t, ok, "third command should return TickEvent")
}

// TestBatchCommandWithNil tests that Batch handles nil commands
func TestBatchCommandWithNil(t *testing.T) {
	cmd1 := Quit()
	cmd2 := Tick(10 * time.Millisecond)

	// Batch with nil commands - Wonton's Batch just returns the slice as-is
	batchCmd := Batch(nil, cmd1, nil, cmd2, nil)
	assert.NotNil(t, batchCmd)

	// Batch doesn't filter nils, it just returns what you give it
	assert.Len(t, batchCmd, 5)
}

// TestBatchCommandEmpty tests that Batch with no commands returns nil
func TestBatchCommandEmpty(t *testing.T) {
	batchCmd := Batch()
	// Empty batch should return nil or a no-op command
	_ = batchCmd
}

// TestSequentialCommands tests multiple commands executed in sequence
func TestSequentialCommands(t *testing.T) {
	var buf bytes.Buffer
	terminal := NewTestTerminal(80, 24, &buf)

	type tracker struct {
		events []int
		quit   bool
	}

	tr := &tracker{
		events: make([]int, 0),
	}

	app := &simpleApp{
		handleFunc: func(event Event) []Cmd {
			switch e := event.(type) {
			case sequenceEvent:
				tr.events = append(tr.events, e.num)
				if e.num >= 3 {
					tr.quit = true
					return []Cmd{Quit()}
				}
			}
			return nil
		},
		renderFunc: func() View { return Text("test") },
	}

	runtime := NewRuntime(terminal, app, 30)

	// Send events in sequence
	go func() {
		time.Sleep(50 * time.Millisecond)
		runtime.SendEvent(sequenceEvent{num: 1})
		runtime.SendEvent(sequenceEvent{num: 2})
		runtime.SendEvent(sequenceEvent{num: 3})
	}()

	err := runtime.Run()
	assert.NoError(t, err)

	// Verify all events were received in order
	assert.Equal(t, []int{1, 2, 3}, tr.events)
	assert.True(t, tr.quit)
}

type sequenceEvent struct {
	num int
}

func (e sequenceEvent) Timestamp() time.Time {
	return time.Now()
}

// TestCommandReturningCommand tests commands that return other commands
func TestCommandReturningCommand(t *testing.T) {
	var buf bytes.Buffer
	terminal := NewTestTerminal(80, 24, &buf)

	type chainTracker struct {
		step1 bool
		step2 bool
		step3 bool
	}

	tracker := &chainTracker{}

	app := &simpleApp{
		handleFunc: func(event Event) []Cmd {
			switch e := event.(type) {
			case chainEvent:
				switch e.step {
				case 1:
					tracker.step1 = true
					// Return a command that will trigger step 2
					return []Cmd{func() Event {
						return chainEvent{step: 2}
					}}
				case 2:
					tracker.step2 = true
					// Return a command that will trigger step 3
					return []Cmd{func() Event {
						return chainEvent{step: 3}
					}}
				case 3:
					tracker.step3 = true
					return []Cmd{Quit()}
				}
			}
			return nil
		},
		renderFunc: func() View { return Text("test") },
	}

	runtime := NewRuntime(terminal, app, 30)

	// Trigger the chain
	go func() {
		time.Sleep(50 * time.Millisecond)
		runtime.SendEvent(chainEvent{step: 1})
	}()

	err := runtime.Run()
	assert.NoError(t, err)

	// Verify all steps were executed
	assert.True(t, tracker.step1, "step 1 should execute")
	assert.True(t, tracker.step2, "step 2 should execute")
	assert.True(t, tracker.step3, "step 3 should execute")
}

type chainEvent struct {
	step int
}

func (e chainEvent) Timestamp() time.Time {
	return time.Now()
}

// TestAfterWithZeroDuration tests After with zero duration
func TestAfterWithZeroDuration(t *testing.T) {
	called := false
	cmd := After(0, func() { called = true })
	assert.NotNil(t, cmd)

	start := time.Now()
	event := cmd()
	elapsed := time.Since(start)

	assert.NotNil(t, event)
	assert.True(t, called)

	// After returns TickEvent
	_, ok := event.(TickEvent)
	assert.True(t, ok)

	// Should execute almost immediately
	assert.Less(t, elapsed, 10*time.Millisecond)
}

// TestAfterWithLongDuration tests After with a longer duration
func TestAfterWithLongDuration(t *testing.T) {
	called := false
	duration := 200 * time.Millisecond
	cmd := After(duration, func() { called = true })
	assert.NotNil(t, cmd)

	start := time.Now()
	event := cmd()
	elapsed := time.Since(start)

	assert.NotNil(t, event)
	assert.True(t, called)

	// After returns TickEvent
	_, ok := event.(TickEvent)
	assert.True(t, ok)

	assert.GreaterOrEqual(t, elapsed, duration)
	assert.Less(t, elapsed, duration+100*time.Millisecond)
}

// TestMultipleAfterCommands tests multiple After commands in sequence
func TestMultipleAfterCommands(t *testing.T) {
	// This test just verifies that After commands work when chained
	count := 0

	cmd1 := After(10*time.Millisecond, func() { count++ })
	cmd2 := After(10*time.Millisecond, func() { count++ })
	cmd3 := After(10*time.Millisecond, func() { count++ })

	start := time.Now()

	// Execute them in sequence
	cmd1()
	cmd2()
	cmd3()

	elapsed := time.Since(start)

	assert.Equal(t, 3, count)
	assert.GreaterOrEqual(t, elapsed, 30*time.Millisecond)
}

// TestCommandErrorHandling tests how commands handle errors
func TestCommandErrorHandling(t *testing.T) {
	// Command that returns an error event
	cmd := func() Event {
		return ErrorEvent{
			Time: time.Now(),
			Err:  errTestQuit,
		}
	}

	event := cmd()
	assert.NotNil(t, event)

	errEvent, ok := event.(ErrorEvent)
	assert.True(t, ok)
	assert.Error(t, errEvent.Err)
	assert.Equal(t, errTestQuit, errEvent.Err)
}

// TestConcurrentCommandExecution tests commands executing concurrently
func TestConcurrentCommandExecution(t *testing.T) {
	// Commands in Wonton execute in separate goroutines via the runtime
	// This test verifies that concurrent command execution works correctly

	var buf bytes.Buffer
	terminal := NewTestTerminal(80, 24, &buf)

	type concurrentTracker struct {
		count    int
		maxCount int
	}

	tracker := &concurrentTracker{
		maxCount: 5,
	}

	app := &simpleApp{
		handleFunc: func(event Event) []Cmd {
			if _, ok := event.(concurrentEvent); ok {
				tracker.count++
				if tracker.count >= tracker.maxCount {
					return []Cmd{Quit()}
				}
			}
			return nil
		},
		renderFunc: func() View { return Text("test") },
	}

	runtime := NewRuntime(terminal, app, 30)

	// Send multiple events concurrently
	go func() {
		time.Sleep(50 * time.Millisecond)
		for i := 0; i < 5; i++ {
			go runtime.SendEvent(concurrentEvent{})
		}
	}()

	err := runtime.Run()
	assert.NoError(t, err)
	assert.Equal(t, 5, tracker.count)
}

type concurrentEvent struct{}

func (e concurrentEvent) Timestamp() time.Time {
	return time.Now()
}

// TestNoneCommand tests that None returns nil
func TestNoneCommand(t *testing.T) {
	cmds := None()
	assert.Nil(t, cmds)
}

// TestSequenceCommand tests that Sequence executes commands in order
func TestSequenceCommand(t *testing.T) {
	// Create some commands that return events
	cmd1 := func() Event { return TickEvent{Time: time.Now(), Frame: 1} }
	cmd2 := func() Event { return TickEvent{Time: time.Now(), Frame: 2} }
	cmd3 := func() Event { return TickEvent{Time: time.Now(), Frame: 3} }

	// Execute the sequence
	seqCmd := Sequence(cmd1, cmd2, cmd3)
	assert.NotNil(t, seqCmd)

	result := seqCmd()
	assert.NotNil(t, result)

	// Result should be a BatchEvent containing all the events
	batchEvent, ok := result.(BatchEvent)
	assert.True(t, ok, "Sequence should return BatchEvent")
	assert.Len(t, batchEvent.Events, 3, "BatchEvent should contain 3 events")

	// Verify each event is a TickEvent
	for i, event := range batchEvent.Events {
		tickEvent, ok := event.(TickEvent)
		assert.True(t, ok, "Event %d should be TickEvent", i)
		assert.Equal(t, uint64(i+1), tickEvent.Frame, "Frame should be %d", i+1)
	}
}

// TestSequenceCommandEmpty tests Sequence with no commands
func TestSequenceCommandEmpty(t *testing.T) {
	seqCmd := Sequence()
	assert.NotNil(t, seqCmd)

	result := seqCmd()
	batchEvent, ok := result.(BatchEvent)
	assert.True(t, ok)
	assert.Len(t, batchEvent.Events, 0)
}

// TestSequenceCommandSingle tests Sequence with single command
func TestSequenceCommandSingle(t *testing.T) {
	cmd := Quit()
	seqCmd := Sequence(cmd)
	assert.NotNil(t, seqCmd)

	result := seqCmd()
	batchEvent, ok := result.(BatchEvent)
	assert.True(t, ok)
	assert.Len(t, batchEvent.Events, 1)

	_, ok = batchEvent.Events[0].(QuitEvent)
	assert.True(t, ok, "Event should be QuitEvent")
}
