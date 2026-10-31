package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/deepnoodle-ai/wonton/assert"
	"github.com/deepnoodle-ai/wonton/termtest"
)

// These tests resize a termtest.Screen in the middle of a stream, as a user
// resizes the terminal window, and check what the screen shows after the
// app redraws. termtest reflows text on resize the way ghostty does.

// resizeFullScreenApp draws a title, a line with a wide character near the
// right edge, and a status bar on the last row.
type resizeFullScreenApp struct{}

func (resizeFullScreenApp) View() View {
	return Stack(
		Text("Title"),
		Text("%s中文 tail", strings.Repeat("x", 37)),
		Spacer(),
		Text("status bar"),
	)
}

// fullScreenFrame returns the screen text a fresh runtime draws for app at
// the given size.
func fullScreenFrame(t *testing.T, app Application, width, height int) string {
	t.Helper()
	screen := termtest.NewScreen(width, height)
	r := NewRuntime(NewTestTerminal(width, height, screen), app, 30)
	r.render()
	return screen.Text()
}

func TestRuntimeRedrawsAfterResize(t *testing.T) {
	sizes := []struct {
		name          string
		width, height int
	}{
		{"narrower", 39, 12},
		{"wider", 60, 12},
		{"shorter", 60, 6},
		{"taller", 60, 16},
		{"narrower and shorter", 25, 5},
	}

	app := resizeFullScreenApp{}
	screen := termtest.NewScreen(80, 12)
	term := NewTestTerminal(80, 12, screen)
	r := NewRuntime(term, app, 30)
	r.render()
	assert.Equal(t, screen.Text(), fullScreenFrame(t, app, 80, 12))

	for _, size := range sizes {
		t.Run(size.name, func(t *testing.T) {
			screen.Resize(size.width, size.height) // the user resizes the window
			term.SetSize(size.width, size.height)  // the program learns the new size
			r.processEvent(ResizeEvent{Time: time.Now(), Width: size.width, Height: size.height})
			r.render()

			assert.Equal(t, screen.Text(), fullScreenFrame(t, app, size.width, size.height))
		})
	}
}

func TestRuntimeRedrawCutsNoWideCharacter(t *testing.T) {
	// At 39 columns the wide character "中" straddles the new right edge.
	app := resizeFullScreenApp{}
	screen := termtest.NewScreen(80, 8)
	term := NewTestTerminal(80, 8, screen)
	r := NewRuntime(term, app, 30)
	r.render()

	screen.Resize(38, 8)
	term.SetSize(38, 8)
	r.processEvent(ResizeEvent{Time: time.Now(), Width: 38, Height: 8})
	r.render()

	for y := 0; y < 8; y++ {
		for x := 0; x < 38; x++ {
			cell := screen.Cell(x, y)
			if cell.Width == 2 {
				assert.True(t, x+1 < 38 && screen.Cell(x+1, y).Continuation,
					"wide character at (%d,%d) has its second half", x, y)
			}
			if cell.Continuation {
				assert.True(t, x > 0 && screen.Cell(x-1, y).Width == 2,
					"continuation at (%d,%d) has its first half", x, y)
			}
		}
	}
	assert.Equal(t, screen.Text(), fullScreenFrame(t, app, 38, 8))
}

// resizeInlineApp shows a live region with one line wider than the narrow
// size and one line shorter than it.
type resizeInlineApp struct{}

func (resizeInlineApp) LiveView() View {
	return Stack(
		Text("status: 0123456789abcdefghij"),
		Text("short"),
	)
}

func (resizeInlineApp) HandleEvent(Event) []Cmd { return nil }

var resizeHistory = []string{"history one", "history two", "history three"}

// newResizeInline starts an InlineApp on screen with three scrollback lines
// and its live region drawn.
func newResizeInline(t *testing.T, screen *termtest.Screen, width int) *InlineApp {
	t.Helper()
	runner := NewInlineApp(WithInlineOutput(screen), WithInlineWidth(width))
	runner.app = resizeInlineApp{}
	runner.render()
	for _, line := range resizeHistory {
		runner.Print(Text("%s", line))
	}
	return runner
}

// resizeInline resizes the window, then sends the app the new size and lets
// it redraw.
func resizeInline(runner *InlineApp, screen *termtest.Screen, width, height int) {
	screen.Resize(width, height)
	runner.processEvent(ResizeEvent{Time: time.Now(), Width: width, Height: height})
	runner.render()
}

// liveRows returns the rows a fresh live region draws for app at width.
func liveRows(t *testing.T, width int) []string {
	t.Helper()
	screen := termtest.NewScreen(width, 10)
	lp := NewLivePrinter(WithWidth(width), WithOutput(screen))
	assert.NoError(t, lp.Update(resizeInlineApp{}.LiveView()))
	return nonBlankRows(screen)
}

func nonBlankRows(screen *termtest.Screen) []string {
	rows := strings.Split(strings.TrimSuffix(screen.Text(), "\n"), "\n")
	for len(rows) > 0 && rows[len(rows)-1] == "" {
		rows = rows[:len(rows)-1]
	}
	return rows
}

func TestInlineAppRedrawsLiveRegionOnceAfterResize(t *testing.T) {
	screen := termtest.NewScreen(40, 12)
	runner := newResizeInline(t, screen, 40)
	want := append(append([]string{}, resizeHistory...), liveRows(t, 40)...)
	assert.Equal(t, nonBlankRows(screen), want)

	t.Run("narrower", func(t *testing.T) {
		resizeInline(runner, screen, 20, 12)
		want := append(append([]string{}, resizeHistory...), liveRows(t, 20)...)
		// The scrollback line directly above the region is intact, and no
		// old region line stays on screen, reflowed or not.
		assert.Equal(t, nonBlankRows(screen), want)
	})

	t.Run("wider again", func(t *testing.T) {
		resizeInline(runner, screen, 40, 12)
		want := append(append([]string{}, resizeHistory...), liveRows(t, 40)...)
		assert.Equal(t, nonBlankRows(screen), want)
	})
}

func TestInlineAppCoalescedResizes(t *testing.T) {
	// The user narrows and widens the window before the app redraws once.
	coalesced := termtest.NewScreen(40, 12)
	runner := newResizeInline(t, coalesced, 40)
	coalesced.Resize(20, 12)
	runner.processEvent(ResizeEvent{Time: time.Now(), Width: 20, Height: 12})
	resizeInline(runner, coalesced, 30, 12)

	single := termtest.NewScreen(40, 12)
	resizeInline(newResizeInline(t, single, 40), single, 30, 12)

	assert.Equal(t, coalesced.Text(), single.Text())
	want := append(append([]string{}, resizeHistory...), liveRows(t, 30)...)
	assert.Equal(t, nonBlankRows(coalesced), want)
}

func TestLivePrinterRedrawsOnceAfterNarrowing(t *testing.T) {
	cases := []struct {
		name          string
		from, to      int
		first, second string
	}{
		{"long first line", 40, 20, "progress: 0123456789abcdefghij", "ok"},
		// The last line ends exactly at the new edge. The cursor after it
		// wraps to the next row on reflow.
		{"last line at the new edge", 40, 20, "first", strings.Repeat("0123456789", 2)},
		// The last line filled the old width, so the cursor waited on its
		// last cell with a pending wrap.
		{"last line filled the old width", 80, 20, "first", strings.Repeat("0123456789", 4)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			screen := termtest.NewScreen(tc.from, 10)
			screen.Write([]byte("above\r\n"))
			lp := NewLivePrinter(WithWidth(tc.from), WithOutput(screen))
			view := Stack(Text("%s", tc.first), Text("%s", tc.second))
			assert.NoError(t, lp.Update(view))

			screen.Resize(tc.to, 10)
			lp.SetWidth(tc.to)
			assert.NoError(t, lp.Update(view))

			want := append([]string{"above"}, liveRowsOf(t, view, tc.to)...)
			assert.Equal(t, nonBlankRows(screen), want)
		})
	}
}

func TestLivePrinterClearsWholeRegionAfterNarrowing(t *testing.T) {
	screen := termtest.NewScreen(40, 10)
	screen.Write([]byte("above\r\n"))
	lp := NewLivePrinter(WithWidth(40), WithOutput(screen))
	assert.NoError(t, lp.Update(Stack(Text("first"), Text("%s", strings.Repeat("0123456789", 2)))))

	screen.Resize(20, 10)
	lp.SetWidth(20)
	lp.Clear()

	assert.Equal(t, nonBlankRows(screen), []string{"above"})
}

func TestLivePrinterCountsStyledTrailingRun(t *testing.T) {
	// The first line is "ok" and a 28-cell background-colored run of
	// spaces. The run is printed cells, so it reflows with the text and
	// counts toward the line's rows.
	view := Stack(Text("ok%s", strings.Repeat(" ", 28)).Bg(ColorBlue), Text("tail"))
	screen := termtest.NewScreen(40, 10)
	screen.Write([]byte("above\r\n"))
	lp := NewLivePrinter(WithWidth(40), WithOutput(screen))
	assert.NoError(t, lp.Update(view))
	assert.NotEqual(t, screen.Cell(29, 1).Style.Background, termtest.Color{})

	screen.Resize(20, 10)
	lp.SetWidth(20)
	assert.NoError(t, lp.Update(view))

	want := append([]string{"above"}, liveRowsOf(t, view, 20)...)
	assert.Equal(t, nonBlankRows(screen), want)
}

func TestLivePrinterRegionTallerThanScreen(t *testing.T) {
	// At 5 columns the old region reflows to 20 rows on a 6-row screen.
	// Its top rows leave the screen; the redraw starts at row 0.
	line := strings.Repeat("0123456789", 2) + "abcde"
	view := Stack(
		Text("1 %s", line).Truncate(),
		Text("2 %s", line).Truncate(),
		Text("3 %s", line).Truncate(),
		Text("4 %s", line).Truncate(),
	)
	screen := termtest.NewScreen(40, 6)
	screen.Write([]byte("above\r\n"))
	lp := NewLivePrinter(WithWidth(40), WithOutput(screen))
	assert.NoError(t, lp.Update(view))

	screen.Resize(5, 6)
	lp.SetWidth(5)
	assert.NoError(t, lp.Update(view))

	assert.Equal(t, nonBlankRows(screen), liveRowsOf(t, view, 5))
}

func TestInlineAppLastLineAtNewEdge(t *testing.T) {
	screen := termtest.NewScreen(40, 12)
	runner := NewInlineApp(WithInlineOutput(screen), WithInlineWidth(40))
	runner.app = edgeInlineApp{}
	runner.render()
	runner.Print(Text("history"))

	resizeInline(runner, screen, 20, 12)

	want := append([]string{"history"}, liveRowsOf(t, edgeInlineApp{}.LiveView(), 20)...)
	assert.Equal(t, nonBlankRows(screen), want)
}

// edgeInlineApp's last live line is 20 cells wide.
type edgeInlineApp struct{}

func (edgeInlineApp) LiveView() View {
	return Stack(Text("first"), Text("%s", strings.Repeat("0123456789", 2)))
}

func (edgeInlineApp) HandleEvent(Event) []Cmd { return nil }

func liveRowsOf(t *testing.T, view View, width int) []string {
	t.Helper()
	screen := termtest.NewScreen(width, 10)
	lp := NewLivePrinter(WithWidth(width), WithOutput(screen))
	assert.NoError(t, lp.Update(view))
	return nonBlankRows(screen)
}

func TestLiveLineWidthSkipsEscapes(t *testing.T) {
	assert.Equal(t, liveLineWidth("plain"), 5)
	assert.Equal(t, liveLineWidth("\x1b[1;31mred\x1b[0m"), 3)
	assert.Equal(t, liveLineWidth("\x1b]8;;https://x.test\x1b\\link\x1b]8;;\x1b\\"), 4)
	assert.Equal(t, liveLineWidth("\x1b]8;;https://x.test\alink\x1b]8;;\a中"), 6)
	assert.Equal(t, liveLineWidth("\x1b(Babc"), 3)
	assert.Equal(t, liveLineWidth("\x1b7x\x1b8"), 1)
}

// resizeRecorderApp draws resizeFullScreenApp and reports each ResizeEvent.
type resizeRecorderApp struct {
	resizeFullScreenApp
	resizes chan ResizeEvent
}

func (a resizeRecorderApp) HandleEvent(event Event) []Cmd {
	switch e := event.(type) {
	case ResizeEvent:
		a.resizes <- e
	case KeyEvent:
		if e.Rune == 'q' {
			return []Cmd{Quit()}
		}
	}
	return nil
}

func TestRunningRuntimeRedrawsAfterSetSize(t *testing.T) {
	out := &lockedScreen{screen: termtest.NewScreen(80, 12)}
	term := NewTestTerminal(80, 12, out)
	app := resizeRecorderApp{resizes: make(chan ResizeEvent, 4)}
	r := NewRuntime(term, app, 30)
	done := make(chan error, 1)
	go func() { done <- r.Run() }()

	waitResize := func(width, height int) {
		t.Helper()
		select {
		case e := <-app.resizes:
			assert.Equal(t, [2]int{e.Width, e.Height}, [2]int{width, height})
		case <-time.After(5 * time.Second):
			t.Fatalf("no ResizeEvent %dx%d", width, height)
		}
	}
	waitResize(80, 12) // the initial size

	// Only SetSize: the runtime's OnResize callback queues the ResizeEvent.
	out.read(func(s *termtest.Screen) { s.Resize(40, 12) })
	term.SetSize(40, 12)
	waitResize(40, 12)

	// The runtime redraws after the event. Exit writes its own cleanup, so
	// check the frame while the runtime still runs.
	want := fullScreenFrame(t, resizeFullScreenApp{}, 40, 12)
	got := ""
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(5 * time.Millisecond) {
		if out.read(func(s *termtest.Screen) { got = s.Text() }); got == want {
			break
		}
	}
	assert.Equal(t, got, want)

	r.SendEvent(KeyEvent{Rune: 'q'})
	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("runtime did not stop")
	}
}
