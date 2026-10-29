package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
	"github.com/deepnoodle-ai/wonton/tmux"
	"github.com/deepnoodle-ai/wonton/tui"
)

// Saves made while one is being written are stored in order, and only the
// newest of those waiting is written at all.
func TestSaverKeepsTheNewestLayout(t *testing.T) {
	var (
		mu      sync.Mutex
		written []string
	)
	writing, release := make(chan struct{}), make(chan struct{})
	s := newSaver(func(data string) {
		if data == "first" {
			close(writing)
			<-release // hold the first write while more saves arrive
		}
		mu.Lock()
		written = append(written, data)
		mu.Unlock()
	})

	s.save("first")
	<-writing
	s.save("second")
	s.save("third")
	close(release)
	s.wait()

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, written, []string{"first", "third"})
}

// Restoring attaches to running sessions without resizing them: the first
// frame gives each its size in the layout.
func TestRestoreKeepsSessionSizes(t *testing.T) {
	server := &tmux.Server{Socket: fmt.Sprintf("corral-test-%d", os.Getpid()), Config: "/dev/null"}
	if !server.Available() {
		t.Skip("tmux is not installed")
	}
	ctx := context.Background()
	t.Cleanup(func() { _ = server.Kill(ctx) })
	assert.NoError(t, server.NewSession(ctx, tmux.SessionOptions{Name: "p1", Command: "cat", Width: 120, Height: 40}))

	a := newApp(server)
	assert.NoError(t, a.restore())
	t.Cleanup(func() {
		for _, p := range a.panes {
			_ = p.term.Detach()
		}
	})
	assert.Equal(t, len(a.panes), 1)
	size, err := server.Run(ctx, "display-message", "-p", "-t", "=p1:", "#{window_width}x#{window_height}")
	assert.NoError(t, err)
	assert.Equal(t, strings.TrimSpace(size), "120x40")
}

// Closing the last space replaces it. When no pane can open in the new one,
// an empty space takes its place rather than none.
func TestCloseLastSpaceKeepsASpace(t *testing.T) {
	a := newApp(&tmux.Server{Binary: "/nonexistent/tmux"}) // no pane can open
	sp := a.newSpaceValue(t.TempDir())
	a.spaces = []*space{sp}

	a.closeSpace(sp)
	a.saver.wait()
	assert.Equal(t, len(a.spaces), 1)
	assert.NotEqual(t, a.space(), sp)
	assert.Equal(t, len(a.space().split.Leaves()), 0)
}

// A pane can exit while a gap's menu is open. Its actions then leave the
// layout alone instead of acting on whatever took the gap's place.
func TestGapMenuSkipsAGapThatIsGone(t *testing.T) {
	a := newApp(&tmux.Server{Binary: "/nonexistent/tmux"}) // saves go nowhere
	sp := a.newSpaceValue(t.TempDir())
	sp.split.Append("a", tui.SplitColumns)
	sp.split.Append("b", tui.SplitColumns)
	sp.split.Split("b", "c", tui.SplitRows) // a | (b over c)
	rows := sp.split.Root.Children[1]
	items := a.gapMenu(sp, rows, 0)

	sp.split.Remove("c") // the rows branch becomes the leaf b
	for _, it := range items[:3] {
		it.Action()
	}
	assert.Equal(t, sp.split.Leaves(), []string{"a", "b"})
	assert.Equal(t, a.status, "that gap is gone")

	// A gap that is still there is acted on.
	a.status = ""
	items = a.gapMenu(sp, sp.split.Root, 0)
	items[1].Action() // Swap sides
	a.saver.wait()
	assert.Equal(t, sp.split.Leaves(), []string{"b", "a"})
	assert.Equal(t, a.status, "")
}

// Choosing Zoom on one pane while another is zoomed zooms the chosen pane.
func TestZoomMenuMovesZoomToThePane(t *testing.T) {
	server := &tmux.Server{Socket: fmt.Sprintf("corral-zoom-%d", os.Getpid()), Config: "/dev/null"}
	if !server.Available() {
		t.Skip("tmux is not installed")
	}
	t.Cleanup(func() { _ = server.Kill(context.Background()) })
	a := newApp(server)
	sp := a.newSpaceValue(t.TempDir())
	a.spaces = []*space{sp}
	var panes []*pane
	for range 2 {
		p, err := a.openPane(sp, sp.dir, 20, 5)
		assert.NoError(t, err)
		sp.split.Append(p.id, tui.SplitColumns)
		panes = append(panes, p)
	}
	t.Cleanup(a.saver.wait)

	choose := func(p *pane, label string) {
		t.Helper()
		for _, it := range a.paneMenu(p, true) {
			if it.Label == label {
				it.Action()
				return
			}
		}
		t.Fatalf("no %q item", label)
	}
	a.toggleZoom(panes[0].id)
	choose(panes[1], "Zoom")
	assert.Equal(t, sp.zoom, panes[1].id)
	assert.Equal(t, sp.focus, panes[1].id)
	choose(panes[1], "Unzoom")
	assert.Equal(t, sp.zoom, "")
}
