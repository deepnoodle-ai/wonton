package termtest

import (
	"fmt"
	"testing"
)

func TestRawMode_BareLineFeedKeepsColumn(t *testing.T) {
	screen := NewScreen(10, 3, WithRawMode(true))
	screen.Write([]byte("ab\ncd"))
	AssertRow(t, screen, 0, "ab")
	AssertRow(t, screen, 1, "  cd")
	AssertCursor(t, screen, 4, 1)
}

func TestRawMode_DefaultAndFalseAreCooked(t *testing.T) {
	for name, screen := range map[string]*Screen{
		"default": NewScreen(10, 3),
		"false":   NewScreen(10, 3, WithRawMode(false)),
	} {
		t.Run(name, func(t *testing.T) {
			screen.Write([]byte("ab\ncd"))
			AssertRow(t, screen, 1, "cd")
			AssertCursor(t, screen, 2, 1)
		})
	}
}

func TestRawMode_LineFeedScrollsAtBottom(t *testing.T) {
	screen := NewScreen(10, 2, WithRawMode(true))
	screen.Write([]byte("ab\ncd\nef"))
	AssertRow(t, screen, 0, "  cd")
	AssertRow(t, screen, 1, "    ef")
	AssertCursor(t, screen, 6, 1)
}

func TestRawMode_LineFeedAfterLastColumn(t *testing.T) {
	// The cursor stays on the last column with a wrap pending. The line feed
	// cancels the wrap and keeps that column.
	screen := NewScreen(4, 3, WithRawMode(true))
	screen.Write([]byte("abcd\nx"))
	AssertRow(t, screen, 0, "abcd")
	AssertRow(t, screen, 1, "   x")
}

// Every other way to move down behaves the same in both modes.
func TestRawMode_OtherControlsUnchanged(t *testing.T) {
	inputs := map[string]string{
		"CRLF": "ab\r\ncd",
		"CR":   "ab\rcd",
		"VT":   "ab\vcd",
		"FF":   "ab\fcd",
		"NEL":  "ab\x1bEcd",
		"IND":  "ab\x1bDcd",
	}
	for name, input := range inputs {
		t.Run(name, func(t *testing.T) {
			cooked := NewScreen(10, 3)
			raw := NewScreen(10, 3, WithRawMode(true))
			cooked.Write([]byte(input))
			raw.Write([]byte(input))
			if !EqualStyled(cooked, raw) {
				t.Fatalf("raw:\n%s\ncooked:\n%s", raw.Text(), cooked.Text())
			}
			cx, cy := cooked.Cursor()
			AssertCursor(t, raw, cx, cy)
		})
	}
}

func TestRawMode_ResetKeepsMode(t *testing.T) {
	screen := NewScreen(10, 3, WithRawMode(true))
	screen.Write([]byte("x\x1bc"))
	screen.Reset()
	screen.Write([]byte("ab\ncd"))
	AssertRow(t, screen, 1, "  cd")
}

func TestRawMode_CaptureHelpers(t *testing.T) {
	const output = "ab\ncd"

	capture := NewCapture(nil)
	capture.Write([]byte(output))
	AssertRow(t, capture.Screen(10, 3, WithRawMode(true)), 1, "  cd")
	AssertRow(t, capture.Screen(10, 3), 1, "cd")

	buf := NewBuffer()
	buf.WriteString(output)
	AssertRow(t, buf.Screen(10, 3, WithRawMode(true)), 1, "  cd")

	recorder := NewRecorder(10, 3, WithRawMode(true))
	recorder.Write([]byte(output))
	AssertRow(t, recorder.Screen(), 1, "  cd")
	AssertRow(t, recorder.Replay(20, 3), 1, "  cd")
	AssertRow(t, recorder.Replay(20, 3, WithRawMode(false)), 1, "cd")
}

func TestScreenOption_NilIsIgnored(t *testing.T) {
	screen := NewScreen(10, 3, nil)
	screen.Write([]byte("ab\ncd"))
	AssertRow(t, screen, 1, "cd")
}

// The same bytes show differently when the tty is in raw mode: a bare line
// feed keeps the cursor column.
func ExampleWithRawMode() {
	output := []byte("ab\ncd")

	cooked := NewScreen(10, 2)
	cooked.Write(output)

	raw := NewScreen(10, 2, WithRawMode(true))
	raw.Write(output)

	fmt.Printf("cooked: %q\n", cooked.Row(1))
	fmt.Printf("raw:    %q\n", raw.Row(1))
	x, y := raw.Cursor()
	fmt.Printf("raw cursor: %d,%d\n", x, y)
	// Output:
	// cooked: "cd"
	// raw:    "  cd"
	// raw cursor: 4,1
}
