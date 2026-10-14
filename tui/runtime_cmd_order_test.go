package tui

import (
	"bytes"
	"testing"
	"time"

	"github.com/deepnoodle-ai/wonton/assert"
)

// cmdOrderApp has three input fields. A cmdsEvent makes HandleEvent return
// the commands it carries, so a test can queue commands and then a key.
type cmdOrderApp struct {
	a, b, c  string
	received []rune
}

type cmdsEvent struct{ cmds []Cmd }

func (cmdsEvent) Timestamp() time.Time { return time.Now() }

func (app *cmdOrderApp) View() View {
	return Stack(
		InputField(&app.a).ID("a"),
		InputField(&app.b).ID("b"),
		InputField(&app.c).ID("c"),
	)
}

func (app *cmdOrderApp) HandleEvent(event Event) []Cmd {
	switch e := event.(type) {
	case cmdsEvent:
		return e.cmds
	case KeyEvent:
		app.received = append(app.received, e.Rune)
	case doneEvent:
		return []Cmd{Quit()}
	}
	return nil
}

// runCmdOrderApp sends cmds and then the key 'x' back to back, so the key is
// queued before anything the commands produce could arrive asynchronously.
func runCmdOrderApp(t *testing.T, cmds ...Cmd) *cmdOrderApp {
	t.Helper()
	var buf bytes.Buffer
	app := &cmdOrderApp{}
	runtime := NewRuntime(NewTestTerminal(80, 24, &buf), app, 30)

	go func() {
		// Let the first render register the fields.
		time.Sleep(50 * time.Millisecond)
		runtime.SendEvent(cmdsEvent{cmds: cmds})
		runtime.SendEvent(KeyEvent{Rune: 'x', Key: KeyUnknown})
		time.Sleep(50 * time.Millisecond)
		runtime.SendEvent(doneEvent{})
	}()

	if err := runtime.Run(); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	return app
}

func TestRuntimeFocusAppliesBeforeNextKey(t *testing.T) {
	app := runCmdOrderApp(t, Focus("b"))
	assert.Equal(t, "", app.a)
	assert.Equal(t, "x", app.b)
}

func TestRuntimeFocusNextAndPrevApplyBeforeNextKey(t *testing.T) {
	app := runCmdOrderApp(t, FocusNext())
	assert.Equal(t, "x", app.b)

	app = runCmdOrderApp(t, FocusPrev())
	assert.Equal(t, "x", app.c)
}

func TestRuntimeFocusCommandsApplyInOrder(t *testing.T) {
	// If the commands ran concurrently, FocusNext could land before Focus
	// and leave "b" focused. In order, focus moves to "b" and then to "c".
	app := runCmdOrderApp(t, Focus("b"), FocusNext())
	assert.Equal(t, "x", app.c)
}

func TestRuntimeQuitStopsBeforeNextKey(t *testing.T) {
	var buf bytes.Buffer
	app := &cmdOrderApp{}
	runtime := NewRuntime(NewTestTerminal(80, 24, &buf), app, 30)

	go func() {
		time.Sleep(50 * time.Millisecond)
		runtime.SendEvent(cmdsEvent{cmds: []Cmd{Quit()}})
		runtime.SendEvent(KeyEvent{Rune: 'x', Key: KeyUnknown})
	}()

	if err := runtime.Run(); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	assert.Equal(t, "", app.a)
}

// keyRecorder is a focusable that consumes and records every key.
type keyRecorder struct {
	testFocusable
	keys []rune
}

func (k *keyRecorder) HandleKeyEvent(e KeyEvent) bool {
	k.keys = append(k.keys, e.Rune)
	return true
}

func TestInlineAppFocusAppliesBeforeNextKey(t *testing.T) {
	runner := NewInlineApp()
	runner.app = &cmdOrderInlineApp{}
	a := &keyRecorder{testFocusable: *newTestFocusable("a", 0, 0)}
	b := &keyRecorder{testFocusable: *newTestFocusable("b", 0, 1)}
	runner.focusMgr.Register(a)
	runner.focusMgr.Register(b)

	runner.processEvent(cmdsEvent{cmds: []Cmd{Focus("b")}})
	runner.processEvent(KeyEvent{Rune: 'x', Key: KeyUnknown})

	assert.Equal(t, 0, len(a.keys))
	assert.Equal(t, []rune{'x'}, b.keys)
}

func TestInlineAppQuitStopsImmediately(t *testing.T) {
	runner := NewInlineApp()
	runner.app = &cmdOrderInlineApp{}

	runner.processEvent(cmdsEvent{cmds: []Cmd{Quit()}})

	assert.True(t, runner.stopping())
	assert.True(t, runner.processEventWithQuitCheck(KeyEvent{Rune: 'x'}))
}

type cmdOrderInlineApp struct{}

func (cmdOrderInlineApp) LiveView() View { return Text("") }

func (cmdOrderInlineApp) HandleEvent(event Event) []Cmd {
	if e, ok := event.(cmdsEvent); ok {
		return e.cmds
	}
	return nil
}

func TestIsRuntimeCmd(t *testing.T) {
	// Focus and the others are inlined into this function; each must still
	// be recognized, since a copied closure would not be.
	assert.True(t, isRuntimeCmd(Quit()))
	assert.True(t, isRuntimeCmd(Focus("a")))
	assert.True(t, isRuntimeCmd(Focus("b")))
	assert.True(t, isRuntimeCmd(FocusNext()))
	assert.True(t, isRuntimeCmd(FocusPrev()))

	// Commands that do work, and custom commands that return the same
	// events, still run asynchronously.
	assert.False(t, isRuntimeCmd(nil))
	assert.False(t, isRuntimeCmd(After(time.Second, "t")))
	assert.False(t, isRuntimeCmd(Sequence(Focus("a"))))
	assert.False(t, isRuntimeCmd(func() Event { return FocusSetEvent{ID: "a"} }))
}
