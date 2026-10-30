package terminal

import (
	"errors"
	"io"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestKeyDecoder_BracketedPasteEndingWithEscape(t *testing.T) {
	// The paste content ends with ESC, so the end marker's ESC follows a
	// lone ESC. The decoder must still find the marker.
	decoder := NewKeyDecoder(strings.NewReader("\x1b[200~hello\x1b\x1b[201~q"))
	event, err := decoder.ReadEvent()
	assert.NoError(t, err)
	assert.Equal(t, event.(KeyEvent).Paste, "hello\x1b")

	event, err = decoder.ReadEvent()
	assert.NoError(t, err)
	assert.Equal(t, event.(KeyEvent).Rune, 'q')
}

func TestKeyDecoder_BracketedPasteWithEmbeddedEscapes(t *testing.T) {
	for _, content := range []string{"a\x1b[20b", "\x1b\x1b[2\x1b[201", "x\x1b[200~y", "\x1b[2\x1b[20\x1b"} {
		decoder := NewKeyDecoder(strings.NewReader("\x1b[200~" + content + "\x1b[201~z"))
		event, err := decoder.ReadEvent()
		assert.NoError(t, err)
		assert.Equal(t, event.(KeyEvent).Paste, content)
		event, err = decoder.ReadEvent()
		assert.NoError(t, err)
		assert.Equal(t, event.(KeyEvent).Rune, 'z')
	}
}

func TestKeyDecoder_EscapeSequenceAcrossBufferBoundary(t *testing.T) {
	// ESC is the last byte of the first 4096-byte buffer fill. The rest of
	// the sequence is still pending, so ESC is not a standalone key.
	input := strings.Repeat("a", 4095) + "\x1b[A"
	for _, readEvent := range []bool{true, false} {
		decoder := NewKeyDecoder(strings.NewReader(input))
		var last KeyEvent
		count := 0
		for {
			var ev KeyEvent
			var err error
			if readEvent {
				var e Event
				e, err = decoder.ReadEvent()
				if k, ok := e.(KeyEvent); ok {
					ev = k
				}
			} else {
				ev, err = decoder.ReadKeyEvent()
			}
			if err == io.EOF {
				break
			}
			assert.NoError(t, err)
			last = ev
			count++
		}
		assert.Equal(t, count, 4096)
		assert.Equal(t, last.Key, KeyArrowUp)
	}
}

func TestKeyDecoder_AltNonASCII(t *testing.T) {
	decoder := NewKeyDecoder(strings.NewReader("\x1b\xc3\xa9"))
	event, err := decoder.ReadEvent()
	assert.NoError(t, err)
	assert.Equal(t, event.(KeyEvent), KeyEvent{Rune: 'é', Alt: true})

	decoder = NewKeyDecoder(strings.NewReader("\x1b\xe4\xb8\x96"))
	key, err := decoder.ReadKeyEvent()
	assert.NoError(t, err)
	assert.Equal(t, key, KeyEvent{Rune: '世', Alt: true})
}

func TestKeyDecoder_KittyFunctionalKeys(t *testing.T) {
	tests := []struct {
		input string
		want  KeyEvent
	}{
		{"\x1b[57358u", KeyEvent{}},                                // Caps Lock
		{"\x1b[57376u", KeyEvent{}},                                // F13
		{"\x1b[57441;2u", KeyEvent{Shift: true}},                   // Left Shift
		{"\x1b[57399u", KeyEvent{Rune: '0'}},                       // KP_0
		{"\x1b[57413u", KeyEvent{Rune: '+'}},                       // KP_Add
		{"\x1b[57414u", KeyEvent{Key: KeyEnter}},                   // KP_Enter
		{"\x1b[57419u", KeyEvent{Key: KeyArrowUp}},                 // KP_Up
		{"\x1b[57426u", KeyEvent{Key: KeyDelete}},                  // KP_Delete
		{"\x1b[57417;5u", KeyEvent{Key: KeyArrowLeft, Ctrl: true}}, // Ctrl+KP_Left
	}
	for _, tt := range tests {
		decoder := NewKeyDecoder(strings.NewReader(tt.input))
		event, err := decoder.ReadEvent()
		assert.NoError(t, err)
		assert.Equal(t, event.(KeyEvent), tt.want, "input %q", tt.input)
	}
}

// chunkSource returns one chunk per Read, like a terminal that writes an
// escape sequence in pieces. WaitReady reports whether chunks remain.
type chunkSource struct {
	chunks []string
	waits  []time.Duration
}

func (c *chunkSource) Read(p []byte) (int, error) {
	if len(c.chunks) == 0 {
		return 0, io.EOF
	}
	n := copy(p, c.chunks[0])
	c.chunks = c.chunks[1:]
	return n, nil
}

func (c *chunkSource) WaitReady(timeout time.Duration) bool {
	c.waits = append(c.waits, timeout)
	return len(c.chunks) > 0
}

func TestKeyDecoder_EscapeSequenceSplitAcrossReads(t *testing.T) {
	for _, readEvent := range []bool{true, false} {
		src := &chunkSource{chunks: []string{"x\x1b", "[A"}}
		decoder := NewKeyDecoder(src)
		var keys []KeyEvent
		for {
			var ev KeyEvent
			var err error
			if readEvent {
				var e Event
				e, err = decoder.ReadEvent()
				ev, _ = e.(KeyEvent)
			} else {
				ev, err = decoder.ReadKeyEvent()
			}
			if err == io.EOF {
				break
			}
			assert.NoError(t, err)
			keys = append(keys, ev)
		}
		assert.Equal(t, keys, []KeyEvent{{Rune: 'x'}, {Key: KeyArrowUp}})
		assert.Equal(t, src.waits, []time.Duration{DefaultEscapeTimeout})
	}
}

func TestKeyDecoder_LoneEscapeAfterTimeout(t *testing.T) {
	src := &chunkSource{chunks: []string{"\x1b"}}
	decoder := NewKeyDecoder(src)
	decoder.SetEscapeTimeout(10 * time.Millisecond)
	event, err := decoder.ReadEvent()
	assert.NoError(t, err)
	assert.Equal(t, event.(KeyEvent), KeyEvent{Key: KeyEscape})
	assert.Equal(t, src.waits, []time.Duration{10 * time.Millisecond})
}

func TestKeyDecoder_EscapeTimeoutDisabled(t *testing.T) {
	src := &chunkSource{chunks: []string{"\x1b", "[A"}}
	decoder := NewKeyDecoder(src)
	decoder.SetEscapeTimeout(0)
	event, err := decoder.ReadEvent()
	assert.NoError(t, err)
	assert.Equal(t, event.(KeyEvent), KeyEvent{Key: KeyEscape})
	assert.Equal(t, len(src.waits), 0)
}

func TestKeyDecoder_MalformedSGRMouseReportIsNotEOF(t *testing.T) {
	// A mouse report missing a coordinate must not read as the end of the
	// input: the runtime stops reading for good on io.EOF.
	decoder := NewKeyDecoder(strings.NewReader("\x1b[<35;20;Ma\x1b[<0;5;6M"))
	event, err := decoder.ReadEvent()
	assert.NoError(t, err)
	assert.Equal(t, event.(KeyEvent).Key, KeyUnknown)

	event, err = decoder.ReadEvent()
	assert.NoError(t, err)
	assert.Equal(t, event.(KeyEvent).Rune, 'a')

	event, err = decoder.ReadEvent()
	assert.NoError(t, err)
	mouse := event.(MouseEvent)
	assert.Equal(t, mouse.X, 4)
	assert.Equal(t, mouse.Y, 5)

	_, err = decoder.ReadEvent()
	assert.ErrorIs(t, err, io.EOF)
}

func TestParseMouseEvent_ShortReportIsNotEOF(t *testing.T) {
	_, err := ParseMouseEvent([]byte("<35;20;M"))
	assert.Error(t, err)
	assert.False(t, errors.Is(err, io.EOF))
}

func TestKeyDecoder_TruncatedUTF8DoesNotSwallowFollowingKey(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []KeyEvent
	}{
		{
			name:  "3-byte lead interrupted after 1 byte by ASCII",
			input: "\xe2a\r",
			want:  []KeyEvent{{Key: KeyUnknown}, {Rune: 'a'}, {Key: KeyEnter}},
		},
		{
			name:  "3-byte lead interrupted after 2 bytes by ASCII",
			input: "\xe2\x82x",
			want:  []KeyEvent{{Key: KeyUnknown}, {Rune: 'x'}},
		},
		{
			name:  "4-byte lead interrupted after 3 bytes by ASCII",
			input: "\xf0\x9f\x98q",
			want:  []KeyEvent{{Key: KeyUnknown}, {Rune: 'q'}},
		},
		{
			name:  "overlong 2-byte lead 0xC0 does not swallow next byte",
			input: "\xc0z",
			want:  []KeyEvent{{Key: KeyUnknown}, {Rune: 'z'}},
		},
		{
			name:  "out-of-range 4-byte lead 0xF5 does not swallow next bytes",
			input: "\xf5abc",
			want:  []KeyEvent{{Key: KeyUnknown}, {Rune: 'a'}, {Rune: 'b'}, {Rune: 'c'}},
		},
		{
			name:  "surrogate is invalid but consumes only continuation bytes",
			input: "\xed\xa0\x80a",
			want:  []KeyEvent{{Key: KeyUnknown}, {Rune: 'a'}},
		},
		{
			name:  "typed U+FFFD is a valid rune",
			input: "\xef\xbf\xbd",
			want:  []KeyEvent{{Rune: utf8.RuneError}},
		},
		{
			name:  "Alt with truncated UTF-8 lead does not swallow next byte",
			input: "\x1b\xc3a",
			want:  []KeyEvent{{Key: KeyUnknown}, {Rune: 'a'}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, readEvent := range []bool{true, false} {
				decoder := NewKeyDecoder(strings.NewReader(tt.input))
				var got []KeyEvent
				for {
					var ev KeyEvent
					var err error
					if readEvent {
						var e Event
						e, err = decoder.ReadEvent()
						ev, _ = e.(KeyEvent)
					} else {
						ev, err = decoder.ReadKeyEvent()
					}
					if err == io.EOF {
						break
					}
					assert.NoError(t, err)
					got = append(got, ev)
				}
				assert.Equal(t, got, tt.want)
			}
		})
	}
}

func TestKeyDecoder_MalformedCSIDoesNotSwallowNonASCII(t *testing.T) {
	// A non-ASCII byte cannot be part of a CSI sequence. It ends the
	// malformed sequence and is decoded as the start of the next key.
	for _, input := range []string{"\x1b[中x", "\x1b[1;中x"} {
		for _, readEvent := range []bool{true, false} {
			decoder := NewKeyDecoder(strings.NewReader(input))
			var got []KeyEvent
			for {
				var ev KeyEvent
				var err error
				if readEvent {
					var e Event
					e, err = decoder.ReadEvent()
					ev, _ = e.(KeyEvent)
				} else {
					ev, err = decoder.ReadKeyEvent()
				}
				if err == io.EOF {
					break
				}
				assert.NoError(t, err)
				got = append(got, ev)
			}
			assert.Equal(t, got, []KeyEvent{{Key: KeyUnknown}, {Rune: '中'}, {Rune: 'x'}})
		}
	}
}

func TestKeyDecoder_LegacyMouseInterruptedByControlByte(t *testing.T) {
	for _, readEvent := range []bool{true, false} {
		// 1. \x1b[M\r -> interrupted at button byte by CR
		// 2. \x1b[M!\x03 -> interrupted at X coordinate by Ctrl+C
		// 3. \x1b[M \x00#a -> xterm sends NUL for a column past 223: the
		//    whole 3-byte report (' ', 0x00, '#') is one KeyUnknown, and '#'
		//    does not leak as a keystroke before 'a'.
		// 4. \x1b[M !\x00b -> the same for a row past 223.
		decoder := NewKeyDecoder(strings.NewReader("\x1b[M\r\x1b[M!\x03\x1b[M \x00#a\x1b[M !\x00b"))

		readNext := func() KeyEvent {
			if readEvent {
				e, err := decoder.ReadEvent()
				assert.NoError(t, err)
				return e.(KeyEvent)
			}
			ev, err := decoder.ReadKeyEvent()
			assert.NoError(t, err)
			return ev
		}

		assert.Equal(t, readNext(), KeyEvent{Key: KeyUnknown})
		assert.Equal(t, readNext(), KeyEvent{Key: KeyEnter})
		assert.Equal(t, readNext(), KeyEvent{Key: KeyUnknown})
		assert.Equal(t, readNext(), KeyEvent{Key: KeyCtrlC, Ctrl: true})
		assert.Equal(t, readNext(), KeyEvent{Key: KeyUnknown})
		assert.Equal(t, readNext(), KeyEvent{Rune: 'a'})
		assert.Equal(t, readNext(), KeyEvent{Key: KeyUnknown})
		assert.Equal(t, readNext(), KeyEvent{Rune: 'b'})
	}

	_, err := ParseMouseEvent([]byte{0, 33, 33})
	assert.Error(t, err)
}

// A terminal reply that arrives after DetectKittyProtocol gives up is left
// for the input reader. It must decode as KeyUnknown, not as typed keys.
func TestKeyDecoder_LateProbeReplyIsNotKeys(t *testing.T) {
	decoder := NewKeyDecoder(strings.NewReader("\x1b[?1u\x1b[?62;22ca"))
	for _, want := range []KeyEvent{{Key: KeyUnknown}, {Key: KeyUnknown}, {Rune: 'a'}} {
		ev, err := decoder.ReadEvent()
		assert.NoError(t, err)
		assert.Equal(t, ev.(KeyEvent), want)
	}
}
