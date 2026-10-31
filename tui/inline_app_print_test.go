package tui

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
	"github.com/deepnoodle-ai/wonton/termtest"
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

// liveRowsApp draws a live region of a set number of rows.
type liveRowsApp struct{ rows int }

func (a *liveRowsApp) LiveView() View {
	views := make([]View, a.rows)
	for i := range views {
		views[i] = Text("live %d", i).Fg(ColorCyan)
	}
	return Stack(views...)
}

// agentLiveApp is the live region of the PRD's measurement: a spinner
// line, an input field between two dividers, and a hint.
type agentLiveApp struct{ input string }

func (a *agentLiveApp) LiveView() View {
	return Stack(
		Text("⠋ Thinking… (12s · 3.4k tokens · esc to interrupt)").Fg(ColorCyan),
		Divider(),
		InputField(&a.input).ID("in").Label("> "),
		Divider(),
		Text("  ? for shortcuts").Fg(ColorBrightBlack),
	)
}

// screenWriter sends output to a termtest screen and keeps the last bytes.
type screenWriter struct {
	screen *termtest.Screen
	log    bytes.Buffer
	fail   bool // fail the next write that opens synchronized output
}

func (w *screenWriter) Write(p []byte) (int, error) {
	if w.fail && bytes.HasPrefix(p, []byte("\x1b[?2026h")) {
		w.fail = false
		n, _ := w.screen.Write(p[:len(p)/2])
		w.log.Write(p[:n])
		return n, errors.New("write failed")
	}
	w.log.Write(p)
	return w.screen.Write(p)
}

// newInsertRig draws an InlineApp's live region on a width x height
// screen whose cursor starts at row startRow, as a shell prompt leaves it.
// With insert false the runner does not know the screen height, so Print
// always clears and redraws the live region: the path Print used before
// the insert.
func newInsertRig(t *testing.T, width, height, startRow int, app InlineApplication, insert bool) (*InlineApp, *screenWriter) {
	t.Helper()
	screen := termtest.NewScreen(width, height)
	for y := 0; y < startRow; y++ {
		fmt.Fprintf(screen, "$ shell %d\r\n", y)
	}
	out := &screenWriter{screen: screen}
	runner := NewInlineApp(WithInlineOutput(out), WithInlineWidth(width))
	runner.app = app
	runner.live.config.Output = out
	runner.live.SetMaxHeight(0)
	if insert {
		runner.live.SetMaxHeight(height)
	}
	runner.running = true
	runner.render()
	return runner, out
}

func screenCells(s *termtest.Screen) [][]termtest.Cell {
	w, h := s.Size()
	cells := make([][]termtest.Cell, h)
	for y := range cells {
		cells[y] = make([]termtest.Cell, w)
		for x := range cells[y] {
			cells[y][x] = s.Cell(x, y)
		}
	}
	return cells
}

// usedInsert reports whether output holds an insert above the live region.
func usedInsert(output string) bool {
	return regexp.MustCompile(`\x1b\[\d+L`).MatchString(output)
}

// testView returns the i-th view of a stream: styled one-row lines, some
// two-row views with a background, and some empty views.
func testView(i int) View {
	colors := []Color{ColorRed, ColorGreen, ColorBlue, ColorYellow}
	switch i % 7 {
	case 3:
		return Stack(Text("line %d a", i).Bg(colors[i%4]), Text("line %d b", i))
	case 5:
		return Text("")
	}
	return Text("line %d", i).Fg(colors[i%4])
}

// In the PRD's measurement (5-line live region, width 100), one Print of a
// 46-byte line costs the line plus 34 bytes and does not redraw the live
// region.
func TestInlinePrintInsertBytes(t *testing.T) {
	const line = "log line 1: compiled package ./foo/bar in 12ms"
	measure := func(insert bool) (int, string) {
		runner, out := newInsertRig(t, 100, 30, 29, &agentLiveApp{}, insert)
		runner.render()
		runner.Printf("warm up")
		out.log.Reset()
		runner.Printf(line)
		assert.Equal(t, usedInsert(out.log.String()), insert)
		return out.log.Len(), out.log.String()
	}
	before, _ := measure(false)
	after, written := measure(true)
	t.Logf("bytes per Print: %d redrawing the live region, %d inserting above it", before, after)
	assert.Equal(t, after-len(line), 34)
	assert.False(t, strings.Contains(written, "Thinking"), "the live region is not redrawn")
}

// A Print of n rows (in one insert) costs 34 control bytes plus 3 for each
// row after the first: a line feed and the CRLF between rows.
func TestInlinePrintInsertControlBytes(t *testing.T) {
	for _, rows := range []int{1, 2, 5} {
		views := make([]View, rows)
		for i := range views {
			views[i] = Text("row %d", i).Fg(ColorGreen)
		}
		var rendered bytes.Buffer
		Fprint(&rendered, Stack(views...), WithWidth(100), WithRawMode(true), WithColor(true))
		content := rendered.Len() - 2*(rows-1)

		runner, out := newInsertRig(t, 100, 30, 29, &liveRowsApp{rows: 5}, true)
		out.log.Reset()
		runner.Print(Stack(views...))
		assert.Equal(t, out.log.Len()-content, 34+3*(rows-1), "%d rows", rows)
	}
}

// 500 Prints, at the screen bottom, leave the same screen (cell by cell,
// with styles) and the same scrollback on both paths. Every line is in
// scrollback or on screen, once, in order.
func TestInlinePrintInsertKeepsEveryLine(t *testing.T) {
	inserted, insertOut := newInsertRig(t, 40, 12, 11, &liveRowsApp{rows: 3}, true)
	redrawn, redrawOut := newInsertRig(t, 40, 12, 11, &liveRowsApp{rows: 3}, false)

	inserts := 0
	for i := 0; i < 500; i++ {
		insertOut.log.Reset()
		inserted.Print(testView(i))
		if usedInsert(insertOut.log.String()) {
			inserts++
		}
		redrawn.Print(testView(i))
		if i%50 == 0 {
			assert.Equal(t, screenCells(insertOut.screen), screenCells(redrawOut.screen), "screen after Print %d", i)
		}
	}
	assert.Equal(t, inserts, 500)
	assert.Equal(t, screenCells(insertOut.screen), screenCells(redrawOut.screen))
	assert.Equal(t, insertOut.screen.Scrollback(), redrawOut.screen.Scrollback())

	all := append(insertOut.screen.Scrollback(), strings.Split(insertOut.screen.Text(), "\n")...)
	var got []int
	for _, row := range all {
		var n int
		if _, err := fmt.Sscanf(row, "line %d", &n); err == nil && !strings.HasSuffix(row, " b") {
			got = append(got, n)
		}
	}
	var want []int
	for i := 0; i < 500; i++ {
		if i%7 != 5 {
			want = append(want, i)
		}
	}
	assert.Equal(t, got, want, "every line once, in order")
	assert.Equal(t, insertOut.screen.Row(9), "live 0")
	assert.Equal(t, insertOut.screen.Row(11), "live 2")
}

// In the first screenful the live region has blank rows below it. The
// insert moves it down into them, as the redraw does, and the shell
// output above stays in place and then scrolls into scrollback in order.
func TestInlinePrintInsertFirstScreenful(t *testing.T) {
	inserted, insertOut := newInsertRig(t, 40, 12, 2, &liveRowsApp{rows: 3}, true)
	redrawn, redrawOut := newInsertRig(t, 40, 12, 2, &liveRowsApp{rows: 3}, false)
	for i := 0; i < 20; i++ {
		inserted.Print(testView(i))
		redrawn.Print(testView(i))
		assert.Equal(t, screenCells(insertOut.screen), screenCells(redrawOut.screen), "screen after Print %d", i)
		assert.Equal(t, insertOut.screen.Scrollback(), redrawOut.screen.Scrollback(), "scrollback after Print %d", i)
	}
	assert.Equal(t, insertOut.screen.Scrollback()[0], "$ shell 0")
}

// A view taller than the free rows goes in several inserts, with the same
// result as the redraw.
func TestInlinePrintInsertTallView(t *testing.T) {
	inserted, insertOut := newInsertRig(t, 40, 10, 9, &liveRowsApp{rows: 3}, true)
	redrawn, redrawOut := newInsertRig(t, 40, 10, 9, &liveRowsApp{rows: 3}, false)
	views := make([]View, 25)
	for i := range views {
		views[i] = Text("tall %d", i)
	}
	insertOut.log.Reset()
	inserted.Print(Stack(views...))
	redrawn.Print(Stack(views...))
	inserts := regexp.MustCompile(`\x1b\[\d+L`).FindAllString(insertOut.log.String(), -1)
	assert.Equal(t, len(inserts), 9, "25 rows in inserts of 3 rows (half of 7 free rows)")
	assert.Equal(t, screenCells(insertOut.screen), screenCells(redrawOut.screen))
	assert.Equal(t, insertOut.screen.Scrollback(), redrawOut.screen.Scrollback())
}

// When the live region grows and shrinks between Prints, the insert still
// matches the redraw.
func TestInlinePrintInsertLiveRegionChangesHeight(t *testing.T) {
	insertApp, redrawApp := &liveRowsApp{rows: 2}, &liveRowsApp{rows: 2}
	inserted, insertOut := newInsertRig(t, 40, 12, 6, insertApp, true)
	redrawn, redrawOut := newInsertRig(t, 40, 12, 6, redrawApp, false)
	for i, rows := range []int{4, 1, 6, 3, 3, 2, 5} {
		insertApp.rows, redrawApp.rows = rows, rows
		inserted.render()
		redrawn.render()
		inserted.Print(testView(i))
		redrawn.Print(testView(i))
		assert.Equal(t, screenCells(insertOut.screen), screenCells(redrawOut.screen), "after Print %d", i)
		assert.Equal(t, insertOut.screen.Scrollback(), redrawOut.screen.Scrollback(), "after Print %d", i)
	}
}

// A live region that leaves less than two free rows is redrawn instead.
func TestInlinePrintInsertNeedsRoom(t *testing.T) {
	runner, out := newInsertRig(t, 40, 6, 5, &liveRowsApp{rows: 5}, true)
	out.log.Reset()
	runner.Printf("no room")
	assert.False(t, usedInsert(out.log.String()))
	assert.True(t, out.screen.Contains("no room"))
}

// A Print before the live region is drawn uses the redraw path.
func TestInlinePrintInsertNeedsLiveRegion(t *testing.T) {
	var buf bytes.Buffer
	runner := NewInlineApp(WithInlineOutput(&buf), WithInlineWidth(40))
	runner.app = &liveRowsApp{rows: 2}
	runner.live.config.Output = &buf
	runner.live.SetMaxHeight(24)
	runner.Printf("before run")
	assert.False(t, usedInsert(buf.String()))
	assert.True(t, strings.Contains(buf.String(), "before run"))
}

// A write that fails part way still ends synchronized output, so the
// terminal does not keep holding its screen.
func TestInlinePrintInsertWriteFails(t *testing.T) {
	runner, out := newInsertRig(t, 40, 10, 9, &liveRowsApp{rows: 2}, true)
	out.fail = true
	out.log.Reset()
	runner.Printf("lost")
	assert.True(t, strings.HasSuffix(out.log.String(), "\x1b[?2026l"))
}

func TestInsertChunk(t *testing.T) {
	assert.Equal(t, insertChunk(30, 5), 12)
	assert.Equal(t, insertChunk(10, 8), 1)
	assert.Equal(t, insertChunk(10, 9), 0, "one free row")
	assert.Equal(t, insertChunk(0, 3), 0, "unknown height")
	assert.Equal(t, insertChunk(10, 0), 0, "no live region")
}
