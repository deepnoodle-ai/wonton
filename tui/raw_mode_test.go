package tui

import (
	"bytes"
	"testing"

	"github.com/deepnoodle-ai/wonton/termtest"
)

// These tests show tui output on a raw-mode screen (termtest.WithRawMode),
// where a bare "\n" keeps the cursor column, as it does when the tty is in
// raw mode. Each test fails if the output loses a "\r" that raw mode needs.

const rawPrompt = "$ prompt\r\n"

// rawScreen replays all output in buf on a raw-mode screen.
func rawScreen(buf *bytes.Buffer, width, height int) *termtest.Screen {
	screen := termtest.NewScreen(width, height, termtest.WithRawMode(true))
	screen.Write(buf.Bytes())
	return screen
}

// rawLiveFrames draws each frame with a LivePrinter that has its default
// options, as InlineApp makes it, and returns the output.
func rawLiveFrames(frames ...[]string) (*LivePrinter, *bytes.Buffer) {
	var buf bytes.Buffer
	buf.WriteString(rawPrompt)
	lp := NewLivePrinter(WithWidth(20), WithOutput(&buf))
	for _, frame := range frames {
		lp.Update(rawLinesView(frame))
	}
	return lp, &buf
}

func rawLinesView(lines []string) View {
	views := make([]View, len(lines))
	for i, line := range lines {
		views[i] = Text("%s", line)
	}
	return Stack(views...)
}

// rawAssertRows checks every row of screen: the first rows match rows, and
// each row after them is empty.
func rawAssertRows(t *testing.T, screen *termtest.Screen, rows ...string) {
	t.Helper()
	_, height := screen.Size()
	if len(rows) > height {
		t.Fatalf("%d rows given for a screen of height %d", len(rows), height)
	}
	for y := range height {
		want := ""
		if y < len(rows) {
			want = rows[y]
		}
		termtest.AssertRow(t, screen, y, want)
	}
}

func rawAssertCursorRow(t *testing.T, screen *termtest.Screen, want int) {
	t.Helper()
	if _, y := screen.Cursor(); y != want {
		t.Fatalf("cursor row = %d, want %d\n%s", y, want, screen.Text())
	}
}

func TestRawMode_LivePrinterGrows(t *testing.T) {
	_, buf := rawLiveFrames(
		[]string{"one"},
		[]string{"one", "two longer", "three"},
	)
	screen := rawScreen(buf, 20, 6)
	rawAssertRows(t, screen, "$ prompt", "one", "two longer", "three", "")
	rawAssertCursorRow(t, screen, 3)
}

func TestRawMode_LivePrinterChangesSomeLines(t *testing.T) {
	_, buf := rawLiveFrames(
		[]string{"x 1", "y 1", "z", "w 1"},
		[]string{"x 2", "y 2", "z", "w 1"},
	)
	screen := rawScreen(buf, 20, 6)
	rawAssertRows(t, screen, "$ prompt", "x 2", "y 2", "z", "w 1", "")
	rawAssertCursorRow(t, screen, 4)
}

func TestRawMode_LivePrinterShrinks(t *testing.T) {
	_, buf := rawLiveFrames(
		[]string{"first", "second", "third line"},
		[]string{"first", "2nd"},
	)
	screen := rawScreen(buf, 20, 6)
	rawAssertRows(t, screen, "$ prompt", "first", "2nd", "", "")
	rawAssertCursorRow(t, screen, 2)
}

func TestRawMode_LivePrinterClear(t *testing.T) {
	lp, buf := rawLiveFrames([]string{"one", "two"})
	lp.Clear()
	screen := rawScreen(buf, 20, 4)
	rawAssertRows(t, screen, "$ prompt", "", "")
	termtest.AssertCursor(t, screen, 0, 1)
}

func TestRawMode_LivePrinterStop(t *testing.T) {
	lp, buf := rawLiveFrames([]string{"one", "two"})
	lp.Stop()
	screen := rawScreen(buf, 20, 5)
	rawAssertRows(t, screen, "$ prompt", "one", "two", "")
	termtest.AssertCursor(t, screen, 0, 3)
}

func TestRawMode_SprintScreen(t *testing.T) {
	screen := SprintScreen(rawLinesView([]string{"alpha", "beta", "gamma"}), WithWidth(20), WithRawMode(true))
	rawAssertRows(t, screen, "alpha", "beta", "gamma")
}

func TestRawMode_Fprintln(t *testing.T) {
	var buf bytes.Buffer
	for range 2 {
		if err := Fprintln(&buf, Text("Hi"), WithWidth(20), WithRawMode(true), WithColor(false)); err != nil {
			t.Fatal(err)
		}
	}
	screen := rawScreen(&buf, 20, 4)
	rawAssertRows(t, screen, "Hi", "Hi", "")
	termtest.AssertCursor(t, screen, 0, 2)
}

// rawLinesApp is an InlineApplication whose live region shows lines.
type rawLinesApp struct{ lines []string }

func (a *rawLinesApp) LiveView() View                { return rawLinesView(a.lines) }
func (a *rawLinesApp) HandleEvent(event Event) []Cmd { return nil }

func TestRawMode_InlineAppPrint(t *testing.T) {
	var buf bytes.Buffer
	runner := newPrintTestRunner(&buf, &rawLinesApp{lines: []string{"live 1", "live 2"}})
	runner.render()
	runner.Print(rawLinesView([]string{"printed a", "printed b"}))
	runner.Print(Text("printed c"))

	screen := rawScreen(&buf, 40, 8)
	rawAssertRows(t, screen, "printed a", "printed b", "printed c", "live 1", "live 2", "")
	rawAssertCursorRow(t, screen, 4)
}

func TestRawMode_InlineAppCleanup(t *testing.T) {
	var buf bytes.Buffer
	runner := newPrintTestRunner(&buf, &rawLinesApp{lines: []string{"live 1", "live 22"}})
	runner.render()
	if err := runner.cleanup(); err != nil {
		t.Fatal(err)
	}
	buf.WriteString("next")

	screen := rawScreen(&buf, 40, 5)
	rawAssertRows(t, screen, "live 1", "live 22", "next")
}
