package tui

import (
	"bytes"
	"strings"
	"sync"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

// unlockedInlineApp keeps state without locks, as InlineApplication promises
// it may: HandleEvent and LiveView never run concurrently.
type unlockedInlineApp struct {
	count int
	value string
}

func (a *unlockedInlineApp) LiveView() View {
	return Stack(
		Text("count %d", a.count),
		Clickable("[ok]", func() {}),
		InputField(&a.value).ID("name").Label("Name:"),
	)
}

func (a *unlockedInlineApp) HandleEvent(event Event) []Cmd {
	if _, ok := event.(incrementEvent); ok {
		a.count++
	}
	return nil
}

func newPrintTestRunner(buf *bytes.Buffer, app InlineApplication) *InlineApp {
	runner := NewInlineApp(WithInlineOutput(buf), WithInlineWidth(40))
	runner.app = app
	runner.live.config.Output = buf
	return runner
}

// Print from a Cmd goroutine must not call LiveView while the event loop is
// inside HandleEvent. Run with -race.
func TestInlineApp_PrintFromCmdDoesNotRaceHandleEvent(t *testing.T) {
	var buf bytes.Buffer
	app := &unlockedInlineApp{}
	runner := newPrintTestRunner(&buf, app)
	runner.render()

	var wg sync.WaitGroup
	wg.Add(2)
	go func() { // event loop
		defer wg.Done()
		for i := 0; i < 200; i++ {
			runner.processEvent(incrementEvent{})
			runner.render()
		}
	}()
	go func() { // Cmd goroutine
		defer wg.Done()
		for i := 0; i < 200; i++ {
			runner.Printf("line %d", i)
		}
	}()
	wg.Wait()
}

// Print redraws the live region as the event loop last drew it: focus styling
// stays and interactive regions are not registered again.
func TestInlineApp_PrintKeepsLiveFrame(t *testing.T) {
	var buf bytes.Buffer
	app := &unlockedInlineApp{}
	runner := newPrintTestRunner(&buf, app)
	runner.render()
	assert.Equal(t, "name", runner.focusMgr.GetFocusedID())
	runner.render() // second frame draws the field as focused
	frame := append([]string(nil), runner.live.lastLines...)
	regions := len(runner.reg.interactive.regions)

	buf.Reset()
	runner.Printf("hello")
	runner.PrintRaw([]byte("raw\r\n"))
	runner.ClearScrollback()

	assert.Equal(t, frame, runner.live.lastLines)
	assert.Equal(t, regions, len(runner.reg.interactive.regions))
	for _, line := range frame {
		assert.Contains(t, buf.String(), line)
	}
	assert.True(t, strings.Contains(buf.String(), "hello"))
}
