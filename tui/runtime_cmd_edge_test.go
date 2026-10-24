package tui

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/deepnoodle-ai/wonton/assert"
)

// runCmdsApp returns cmds from its first event and quits on doneEvent.
type runCmdsApp struct {
	cmds    []Cmd
	sent    bool
	initErr error
}

func (a *runCmdsApp) Init() error { return a.initErr }
func (a *runCmdsApp) View() View  { return Text("x") }
func (a *runCmdsApp) HandleEvent(event Event) []Cmd {
	if _, ok := event.(doneEvent); ok {
		return []Cmd{Quit()}
	}
	if !a.sent {
		a.sent = true
		return a.cmds
	}
	return nil
}

// runWithTimeout runs the runtime and fails the test if it does not return.
func runWithTimeout(t *testing.T, runtime *Runtime, timeout time.Duration) error {
	t.Helper()
	result := make(chan error, 1)
	go func() { result <- runtime.Run() }()
	select {
	case err := <-result:
		return err
	case <-time.After(timeout):
		runtime.Stop()
		<-result
		t.Fatal("Run did not return")
		return nil
	}
}

func TestRuntime_NilCmdIsIgnored(t *testing.T) {
	var buf bytes.Buffer
	app := &runCmdsApp{cmds: []Cmd{nil, Sequence(nil, After(time.Millisecond, "t"))}}
	runtime := NewRuntime(NewTestTerminal(80, 24, &buf), app, 30)
	go func() {
		time.Sleep(100 * time.Millisecond)
		runtime.SendEvent(doneEvent{})
	}()
	assert.NoError(t, runWithTimeout(t, runtime, 2*time.Second))
}

func TestRuntime_NestedSequenceQuit(t *testing.T) {
	var buf bytes.Buffer
	app := &runCmdsApp{cmds: []Cmd{Sequence(Sequence(Quit()))}}
	runtime := NewRuntime(NewTestTerminal(80, 24, &buf), app, 30)
	assert.NoError(t, runWithTimeout(t, runtime, 2*time.Second))
}

func TestRuntime_InitErrorDoesNotLeaveRunning(t *testing.T) {
	var buf bytes.Buffer
	initErr := errors.New("init failed")
	app := &runCmdsApp{initErr: initErr}
	runtime := NewRuntime(NewTestTerminal(80, 24, &buf), app, 30)
	err := runtime.Run()
	assert.True(t, errors.Is(err, initErr))
	assert.False(t, runtime.running)
	// A second Run reports the Init error again, not "already running".
	err = runtime.Run()
	assert.True(t, errors.Is(err, initErr))
}

func TestInlineApp_NestedBatchQuit(t *testing.T) {
	runner := NewInlineApp()
	runner.app = &testInlineApp{}
	nested := BatchEvent{Events: []Event{BatchEvent{Events: []Event{QuitEvent{}}}}}
	assert.True(t, runner.processEventWithQuitCheck(nested))
}

func TestInlineApp_NilCmdIsIgnored(t *testing.T) {
	runner := NewInlineApp()
	runner.app = &runCmdsInlineApp{cmds: []Cmd{nil}}
	runner.processEvent(incrementEvent{})
	select {
	case cmd := <-runner.cmds:
		t.Fatalf("nil Cmd was queued: %v", cmd != nil)
	default:
	}
}

type runCmdsInlineApp struct{ cmds []Cmd }

func (a *runCmdsInlineApp) LiveView() View                { return Text("x") }
func (a *runCmdsInlineApp) HandleEvent(event Event) []Cmd { return a.cmds }
