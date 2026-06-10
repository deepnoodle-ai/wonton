package terminal

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/deepnoodle-ai/wonton/assert"
)

// --- CSI sequence consumption (decoder must never leave sequence bytes
// behind to be misread as typed keys) ---

func TestKeyDecoder_MultiDigitModifier(t *testing.T) {
	// Kitty protocol with a lock-modifier bit set: ESC[97;65u (CapsLock + a).
	// The two-digit modifier must be consumed fully; "x" follows.
	input := []byte("\x1b[97;65ux")
	decoder := NewKeyDecoder(bytes.NewReader(input))

	event, err := decoder.ReadKeyEvent()
	assert.NoError(t, err)
	assert.Equal(t, 'a', event.Rune)

	event, err = decoder.ReadKeyEvent()
	assert.NoError(t, err)
	assert.Equal(t, 'x', event.Rune)
}

func TestKeyDecoder_CursorPositionReportConsumed(t *testing.T) {
	// A DSR cursor position report (ESC[24;80R) is not a key, but its bytes
	// must be fully consumed so the trailing "R" isn't reported as typed.
	input := []byte("\x1b[24;80Rq")
	decoder := NewKeyDecoder(bytes.NewReader(input))

	event, err := decoder.ReadKeyEvent()
	assert.NoError(t, err)
	assert.Equal(t, KeyUnknown, event.Key)

	event, err = decoder.ReadKeyEvent()
	assert.NoError(t, err)
	assert.Equal(t, 'q', event.Rune)
}

func TestKeyDecoder_ModifyOtherKeys(t *testing.T) {
	// xterm modifyOtherKeys: ESC[27;5;13~ is Ctrl+Enter
	input := []byte("\x1b[27;5;13~")
	decoder := NewKeyDecoder(bytes.NewReader(input))

	event, err := decoder.ReadKeyEvent()
	assert.NoError(t, err)
	assert.Equal(t, KeyEnter, event.Key)
	assert.True(t, event.Ctrl)
}

func TestKeyDecoder_ModifiedHomeEnd(t *testing.T) {
	tests := []struct {
		name  string
		input string
		key   Key
		ctrl  bool
		shift bool
	}{
		{"Ctrl+Home", "\x1b[1;5H", KeyHome, true, false},
		{"Ctrl+End", "\x1b[1;5F", KeyEnd, true, false},
		{"Shift+Home", "\x1b[1;2H", KeyHome, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decoder := NewKeyDecoder(strings.NewReader(tt.input))
			event, err := decoder.ReadKeyEvent()
			assert.NoError(t, err)
			assert.Equal(t, tt.key, event.Key)
			assert.Equal(t, tt.ctrl, event.Ctrl)
			assert.Equal(t, tt.shift, event.Shift)
		})
	}
}

func TestKeyDecoder_ModifiedFunctionKey(t *testing.T) {
	// F5 with Ctrl: ESC[15;5~
	decoder := NewKeyDecoder(strings.NewReader("\x1b[15;5~"))
	event, err := decoder.ReadKeyEvent()
	assert.NoError(t, err)
	assert.Equal(t, KeyF5, event.Key)
	assert.True(t, event.Ctrl)
}

func TestKeyDecoder_CtrlDelete(t *testing.T) {
	decoder := NewKeyDecoder(strings.NewReader("\x1b[3;5~"))
	event, err := decoder.ReadKeyEvent()
	assert.NoError(t, err)
	assert.Equal(t, KeyDelete, event.Key)
	assert.True(t, event.Ctrl)
}

func TestKeyDecoder_SGRMouseOnKeyAPIStaysInSync(t *testing.T) {
	// A mouse report arriving on the key-only API must be consumed fully so
	// the following keystroke is decoded correctly.
	input := []byte("\x1b[<0;10;5Ma")
	decoder := NewKeyDecoder(bytes.NewReader(input))

	event, err := decoder.ReadKeyEvent()
	assert.NoError(t, err)
	assert.Equal(t, KeyUnknown, event.Key)

	event, err = decoder.ReadKeyEvent()
	assert.NoError(t, err)
	assert.Equal(t, 'a', event.Rune)
}

func TestKeyDecoder_KittyEventTypeSubparam(t *testing.T) {
	// Kitty can append an event type sub-parameter: ESC[13;1:1u
	decoder := NewKeyDecoder(strings.NewReader("\x1b[13;1:1u"))
	event, err := decoder.ReadKeyEvent()
	assert.NoError(t, err)
	assert.Equal(t, KeyEnter, event.Key)
}

// --- Style / Cell equality with RGB values ---

func TestStyleEqualRGBByValue(t *testing.T) {
	a := NewStyle().WithFgRGB(NewRGB(10, 20, 30)).WithBgRGB(NewRGB(1, 2, 3))
	b := NewStyle().WithFgRGB(NewRGB(10, 20, 30)).WithBgRGB(NewRGB(1, 2, 3))
	c := NewStyle().WithFgRGB(NewRGB(99, 20, 30))

	assert.True(t, a.Equal(b)) // separate pointers, same values
	assert.False(t, a.Equal(c))
	assert.False(t, a.Equal(NewStyle()))
}

func TestFlushSkipsUnchangedRGBCells(t *testing.T) {
	var out bytes.Buffer
	term := NewTestTerminal(10, 3, &out)

	style := NewStyle().WithFgRGB(NewRGB(200, 100, 50))
	term.PrintAtStyled(0, 0, "hi", style)
	term.Flush()
	firstLen := out.Len()
	if firstLen == 0 {
		t.Fatal("expected output from first flush")
	}

	// Rebuild the same content with a freshly-allocated RGB pointer; the
	// diff must treat it as unchanged and emit nothing.
	out.Reset()
	style2 := NewStyle().WithFgRGB(NewRGB(200, 100, 50))
	term.PrintAtStyled(0, 0, "hi", style2)
	term.Flush()
	if got := out.String(); strings.Contains(got, "hi") || strings.Contains(got, "38;2;200") {
		t.Errorf("unchanged RGB-styled cells were re-emitted: %q", got)
	}
}

// --- Flush retry after write error ---

type failOnceWriter struct {
	failed bool
	buf    bytes.Buffer
}

func (w *failOnceWriter) Write(p []byte) (int, error) {
	if !w.failed {
		w.failed = true
		return 0, assertError("write failed")
	}
	return w.buf.Write(p)
}

type assertError string

func (e assertError) Error() string { return string(e) }

func TestFlushRetryAfterWriteError(t *testing.T) {
	w := &failOnceWriter{}
	term := NewTestTerminal(10, 3, w)

	term.PrintAt(0, 0, "hello")
	term.Flush() // first flush fails; front buffer must stay unchanged

	term.Flush() // retry must re-emit the cells
	if !strings.Contains(w.buf.String(), "hello") {
		t.Errorf("retry flush did not re-emit cells; got %q", w.buf.String())
	}
}

// --- PrintTruncated must not reorder text at the clip edge ---

func TestPrintTruncatedNoReorderAtEdge(t *testing.T) {
	var out bytes.Buffer
	term := NewTestTerminal(5, 3, &out)

	frame, err := term.BeginFrame()
	assert.NoError(t, err)
	// 界 (width 2) doesn't fit in the last column; the following narrow 'x'
	// must NOT be drawn in its place.
	assert.NoError(t, frame.PrintTruncated(0, 0, "abcd界x", NewStyle()))
	assert.NoError(t, term.EndFrame(frame))

	assert.Equal(t, 'd', term.GetCell(3, 0).Char)
	if got := term.GetCell(4, 0).Char; got == 'x' {
		t.Errorf("truncated text was reordered: trailing 'x' drawn at clip edge")
	}
}

// --- EndFrame with an invalid frame must not deadlock the terminal ---

func TestEndFrameInvalidFrameRecoverable(t *testing.T) {
	var out bytes.Buffer
	term := NewTestTerminal(10, 3, &out)

	_, err := term.BeginFrame()
	assert.NoError(t, err)

	assert.Equal(t, ErrInvalidFrame, term.EndFrame(nil))

	// The terminal must remain usable: BeginFrame would deadlock here if
	// EndFrame had kept the lock.
	done := make(chan struct{})
	go func() {
		frame, err := term.BeginFrame()
		if err == nil {
			term.EndFrame(frame)
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("terminal deadlocked after EndFrame with invalid frame")
	}
}

func TestEndFrameTwiceReturnsError(t *testing.T) {
	var out bytes.Buffer
	term := NewTestTerminal(10, 3, &out)

	frame, err := term.BeginFrame()
	assert.NoError(t, err)
	assert.NoError(t, term.EndFrame(frame))
	assert.Equal(t, ErrInvalidFrame, term.EndFrame(frame))
}

// --- Fill clipping ---

func TestFillClipsToBounds(t *testing.T) {
	var out bytes.Buffer
	term := NewTestTerminal(5, 3, &out)

	// Extends one column past the right edge: must fill the in-bounds part
	// instead of silently doing nothing.
	term.Fill(3, 0, 3, 1, '#')

	assert.Equal(t, '#', term.GetCell(3, 0).Char)
	assert.Equal(t, '#', term.GetCell(4, 0).Char)
}

// --- OnResize unregister safety ---

func TestOnResizeUnregisterAfterClear(t *testing.T) {
	var out bytes.Buffer
	term := NewTestTerminal(10, 3, &out)

	unregister1 := term.OnResize(func(w, h int) {})
	term.ClearResizeCallbacks()

	called := false
	term.OnResize(func(w, h int) { called = true })

	// The stale unregister function must not remove the new callback.
	unregister1()

	term.callbackMu.RLock()
	n := len(term.resizeCallbacks)
	term.callbackMu.RUnlock()
	assert.Equal(t, 1, n)
	_ = called
}

// --- Recorder idle clamp accumulates across events ---

func TestRecorderIdleClampCumulative(t *testing.T) {
	r := &Recorder{
		startTime:     time.Now(),
		lastEventTime: time.Now(),
		idleTimeLimit: 0.05,
	}

	// Simulate three events with a long idle gap between the first two by
	// computing timestamps directly.
	base := r.startTime
	t1 := r.eventTimeLocked(base.Add(10 * time.Millisecond))
	t2 := r.eventTimeLocked(base.Add(1010 * time.Millisecond)) // 1s gap, clamped to 50ms
	t3 := r.eventTimeLocked(base.Add(1020 * time.Millisecond)) // 10ms later

	gap12 := t2 - t1
	gap23 := t3 - t2
	if gap12 > 0.06 {
		t.Errorf("idle gap not clamped: %.3fs", gap12)
	}
	if gap23 > 0.02 {
		t.Errorf("clamped idle time reappeared before the next event: gap = %.3fs", gap23)
	}
}

// --- Frame title with wide characters ---

func TestFrameTitleWideCharsKeepsBorderWidth(t *testing.T) {
	var out bytes.Buffer
	term := NewTestTerminal(20, 5, &out)

	f := NewFrame(0, 0, 12, 3).WithTitle("日本", AlignLeft)
	frame, err := term.BeginFrame()
	assert.NoError(t, err)
	f.Draw(frame)
	assert.NoError(t, term.EndFrame(frame))

	// The top-right corner must land exactly at column Width-1.
	corner := term.GetCell(11, 0)
	assert.Equal(t, []rune(f.Border.TopRight)[0], corner.Char)
	// Nothing may spill past the frame edge.
	assert.Equal(t, ' ', term.GetCell(12, 0).Char)
}
