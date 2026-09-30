package terminal

import (
	"errors"
	"testing"

	"github.com/deepnoodle-ai/wonton/internal/terminalinput"
)

type ownershipBoundaryReader struct{ bytes []byte }

func (r *ownershipBoundaryReader) Read(dst []byte) (int, error) {
	if len(r.bytes) == 0 {
		return 0, terminalinput.Boundary
	}
	n := copy(dst, r.bytes)
	r.bytes = r.bytes[n:]
	return n, nil
}

func TestDecoderOwnershipBoundaryFinalizesIncompleteFraming(t *testing.T) {
	for _, tc := range []struct {
		name, input, paste string
		escape             bool
	}{
		{name: "utf8", input: "\xe2\x82"},
		{name: "csi", input: "\x1b[12;"},
		{name: "ss3", input: "\x1bO"},
		{name: "escape", input: "\x1b", escape: true},
		{name: "paste", input: "\x1b[200~one\r\ntwo\t界", paste: "one\ntwo  界"},
		{name: "paste partial end", input: "\x1b[200~one\x1b[201", paste: "one"},
		{name: "paste partial utf8", input: "\x1b[200~one\xe2\x82", paste: "one"},
		{name: "empty paste", input: "\x1b[200~"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reader := &ownershipBoundaryReader{bytes: []byte(tc.input)}
			decoder := NewKeyDecoder(reader)
			decoder.SetPasteTabWidth(2)
			var gotPaste string
			escapes := 0
			for {
				event, err := decoder.ReadEvent()
				if key, ok := event.(KeyEvent); ok {
					gotPaste += key.Paste
					if key.Key == KeyEscape {
						escapes++
					}
					if key.Rune != 0 {
						t.Fatalf("incomplete framing became text: %+v", key)
					}
				}
				if errors.Is(err, terminalinput.Boundary) {
					break
				}
				if err != nil {
					t.Fatal(err)
				}
			}
			if gotPaste != tc.paste || (escapes == 1) != tc.escape {
				t.Fatalf("retained paste=%q escapes=%d", gotPaste, escapes)
			}
			// Keep the same decoder. Child suffix bytes never enter this reader. A
			// missing old suffix cannot consume the first application typing/Enter/quit.
			reader.bytes = []byte("x\r\x03")
			for _, expected := range []KeyEvent{{Rune: 'x'}, {Key: KeyEnter}, {Key: KeyCtrlC, Ctrl: true}} {
				event, err := decoder.ReadEvent()
				if err != nil || event.(KeyEvent) != expected {
					t.Fatalf("resumed event=%+v err=%v want=%+v", event, err, expected)
				}
			}
		})
	}
}
