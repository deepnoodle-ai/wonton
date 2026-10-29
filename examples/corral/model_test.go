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
