package tui

import (
	"bytes"
	"testing"
	"time"

	"github.com/deepnoodle-ai/wonton/assert"
)

// clickAt drives the runtime's press/release pair and returns the synthetic
// click it produced, or nil if it produced none.
func clickAt(r *Runtime, x, y int, at time.Time) *MouseEvent {
	r.processMouseEvent(MouseEvent{X: x, Y: y, Button: MouseButtonLeft, Type: MousePress, Time: at})
	_, click := r.processMouseEvent(MouseEvent{X: x, Y: y, Button: MouseButtonLeft, Type: MouseRelease, Time: at})
	if click == nil {
		return nil
	}
	ev := click.(MouseEvent)
	return &ev
}

func newTestRuntime(t *testing.T) *Runtime {
	t.Helper()
	var buf bytes.Buffer
	return NewRuntime(NewTestTerminal(80, 24, &buf), &testRuntimeModel{}, 30)
}

func TestRuntimeCountsRepeatedClicksOnTheSameCell(t *testing.T) {
	r := newTestRuntime(t)
	now := time.Now()

	assert.Equal(t, 1, clickAt(r, 5, 5, now).ClickCount)
	assert.Equal(t, 2, clickAt(r, 5, 5, now.Add(100*time.Millisecond)).ClickCount)
	assert.Equal(t, 3, clickAt(r, 5, 5, now.Add(200*time.Millisecond)).ClickCount)
}

func TestRuntimeClickCountResets(t *testing.T) {
	tests := []struct {
		name   string
		second func(*Runtime, time.Time) *MouseEvent
	}{
		{
			name: "a different cell starts over",
			second: func(r *Runtime, now time.Time) *MouseEvent {
				return clickAt(r, 6, 5, now.Add(50*time.Millisecond))
			},
		},
		{
			name: "too slow starts over",
			second: func(r *Runtime, now time.Time) *MouseEvent {
				return clickAt(r, 5, 5, now.Add(doubleClickThreshold+time.Millisecond))
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := newTestRuntime(t)
			now := time.Now()
			assert.Equal(t, 1, clickAt(r, 5, 5, now).ClickCount)
			assert.Equal(t, 1, tc.second(r, now).ClickCount)
		})
	}
}

func TestRuntimeReleaseElsewhereIsNotAClick(t *testing.T) {
	r := newTestRuntime(t)
	now := time.Now()

	r.processMouseEvent(MouseEvent{X: 5, Y: 5, Button: MouseButtonLeft, Type: MousePress, Time: now})
	_, click := r.processMouseEvent(MouseEvent{X: 12, Y: 9, Button: MouseButtonLeft, Type: MouseRelease, Time: now})
	assert.Nil(t, click, "a drag that ends elsewhere is not a click")
}

func TestMouseModeOptionsKeepTheMostCapableRequest(t *testing.T) {
	tests := []struct {
		name string
		opts []RunOption
		want mouseMode
	}{
		{"none", nil, mouseOff},
		{"buttons", []RunOption{WithMouseButtons(true)}, mouseButton},
		{"drag", []RunOption{WithMouseDrag(true)}, mouseDrag},
		{"hover", []RunOption{WithMouseTracking(true)}, mouseHover},
		{"drag then buttons keeps drag", []RunOption{WithMouseDrag(true), WithMouseButtons(true)}, mouseDrag},
		{"buttons then drag keeps drag", []RunOption{WithMouseButtons(true), WithMouseDrag(true)}, mouseDrag},
		{"drag then hover keeps hover", []RunOption{WithMouseDrag(true), WithMouseTracking(true)}, mouseHover},
		{"disabling the mode you set turns it off", []RunOption{WithMouseDrag(true), WithMouseDrag(false)}, mouseOff},
		{"disabling a mode you did not set changes nothing", []RunOption{WithMouseDrag(true), WithMouseButtons(false)}, mouseDrag},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := defaultRunConfig()
			for _, opt := range tc.opts {
				opt(&cfg)
			}
			assert.Equal(t, tc.want, cfg.mouseMode)
		})
	}
}

func TestSetKittyKeyboardEnablesWithoutProbing(t *testing.T) {
	// The probe never runs against a test terminal, so what this pins down is
	// the decision: asking for the protocol outright is not the same as being
	// told the terminal supports it.
	var buf bytes.Buffer
	term := NewTestTerminal(80, 24, &buf)
	r := NewRuntime(term, &testRuntimeModel{}, 30)

	assert.False(t, r.kittyKeyboard)
	r.SetKittyKeyboard(true)
	assert.True(t, r.kittyKeyboard)

	buf.Reset()
	term.EnableEnhancedKeyboard()
	assert.Equal(t, "\033[>1u", buf.String(), "the enable is unconditional")
	assert.True(t, term.IsKittyProtocolEnabled())
	assert.False(t, term.IsKittyProtocolSupported(), "enabling is not a claim of support")

	buf.Reset()
	term.DisableEnhancedKeyboard()
	assert.Equal(t, "\033[<u", buf.String(), "and it is released on the way out")
}

func TestResizeRepaintsAreThrottledButKeystrokesAreNot(t *testing.T) {
	// A window drag delivers SIGWINCH far faster than the frame rate and each
	// resize repaints every cell, so back-to-back resizes coalesce onto the
	// ticker. Anything the user typed renders straight away.
	r := newTestRuntime(t)
	r.lastRender = time.Now()
	within := r.lastRender.Add(r.frameInterval() / 2)
	after := r.lastRender.Add(2 * r.frameInterval())

	assert.True(t, r.shouldThrottleResize(true, within), "a second resize inside the frame waits")
	assert.False(t, r.shouldThrottleResize(true, after), "a frame later it repaints")
	assert.False(t, r.shouldThrottleResize(false, within), "a keystroke never waits")
}

func TestFrameIntervalMatchesTheFrameRate(t *testing.T) {
	var buf bytes.Buffer
	assert.Equal(t, time.Second/30, NewRuntime(NewTestTerminal(80, 24, &buf), &testRuntimeModel{}, 30).frameInterval())
	assert.Equal(t, time.Second/60, NewRuntime(NewTestTerminal(80, 24, &buf), &testRuntimeModel{}, 60).frameInterval())
}
