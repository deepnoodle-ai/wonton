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
		lp.Update(linesView(frame))
	}
	return lp, &buf
}

func linesView(lines []string) View {
	views := make([]View, len(lines))
	for i, line := range lines {
		views[i] = Text("%s", line)
	}
	return Stack(views...)
}

func assertRows(t *testing.T, screen *termtest.Screen, rows ...string) {
	t.Helper()
	for y, want := range rows {
		termtest.AssertRow(t, screen, y, want)
	}
}

func assertCursorRow(t *testing.T, screen *termtest.Screen, want int) {
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
	assertRows(t, screen, "$ prompt", "one", "two longer", "three", "")
	assertCursorRow(t, screen, 3)
}

func TestRawMode_LivePrinterChangesSomeLines(t *testing.T) {
	_, buf := rawLiveFrames(
		[]string{"x 1", "y 1", "z", "w 1"},
		[]string{"x 2", "y 2", "z", "w 1"},
	)
	screen := rawScreen(buf, 20, 6)
	assertRows(t, screen, "$ prompt", "x 2", "y 2", "z", "w 1", "")
	assertCursorRow(t, screen, 4)
}

func TestRawMode_LivePrinterShrinks(t *testing.T) {
	_, buf := rawLiveFrames(
		[]string{"first", "second", "third line"},
		[]string{"first", "2nd"},
	)
	screen := rawScreen(buf, 20, 6)
	assertRows(t, screen, "$ prompt", "first", "2nd", "", "")
	assertCursorRow(t, screen, 2)
}

func TestRawMode_LivePrinterClear(t *testing.T) {
	lp, buf := rawLiveFrames([]string{"one", "two"})
	lp.Clear()
	screen := rawScreen(buf, 20, 4)
	assertRows(t, screen, "$ prompt", "", "")
	termtest.AssertCursor(t, screen, 0, 1)
}

func TestRawMode_LivePrinterStop(t *testing.T) {
	lp, buf := rawLiveFrames([]string{"one", "two"})
	lp.Stop()
	screen := rawScreen(buf, 20, 5)
	assertRows(t, screen, "$ prompt", "one", "two", "")
	termtest.AssertCursor(t, screen, 0, 3)
}

func TestRawMode_SprintScreen(t *testing.T) {
	screen := SprintScreen(linesView([]string{"alpha", "beta", "gamma"}), WithWidth(20), WithRawMode(true))
	assertRows(t, screen, "alpha", "beta", "gamma")
}

func TestRawMode_Fprintln(t *testing.T) {
	var buf bytes.Buffer
	for range 2 {
		if err := Fprintln(&buf, Text("Hi"), WithWidth(20), WithRawMode(true), WithColor(false)); err != nil {
			t.Fatal(err)
		}
	}
	screen := rawScreen(&buf, 20, 4)
	assertRows(t, screen, "Hi", "Hi", "")
	termtest.AssertCursor(t, screen, 0, 2)
}

// rawLinesApp is an InlineApplication whose live region shows lines.
type rawLinesApp struct{ lines []string }

func (a *rawLinesApp) LiveView() View                { return linesView(a.lines) }
func (a *rawLinesApp) HandleEvent(event Event) []Cmd { return nil }

func TestRawMode_InlineAppPrint(t *testing.T) {
	var buf bytes.Buffer
	runner := newPrintTestRunner(&buf, &rawLinesApp{lines: []string{"live 1", "live 2"}})
	runner.render()
	runner.Print(linesView([]string{"printed a", "printed b"}))
	runner.Print(Text("printed c"))

	screen := rawScreen(&buf, 40, 8)
	assertRows(t, screen, "printed a", "printed b", "printed c", "live 1", "live 2", "")
	assertCursorRow(t, screen, 4)
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
	assertRows(t, screen, "live 1", "live 22", "next")
}
