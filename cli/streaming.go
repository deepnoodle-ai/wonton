package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/deepnoodle-ai/gooey/tui"
)

// StreamFunc is a function that yields string chunks.
type StreamFunc func(yield func(string)) error

// Stream executes a streaming function with appropriate output handling.
// In TTY mode: displays an animated spinner with streaming text.
// In pipe mode: outputs raw chunks directly.
// With --json flag: outputs newline-delimited JSON events.
func (c *Context) Stream(fn StreamFunc) error {
	if c.Bool("json") {
		return c.streamJSON(fn)
	}
	if c.Interactive() {
		return c.streamInteractive(fn)
	}
	return c.streamRaw(fn)
}

// streamRaw outputs chunks directly to stdout.
func (c *Context) streamRaw(fn StreamFunc) error {
	return fn(func(chunk string) {
		fmt.Fprint(c.stdout, chunk)
	})
}

// streamJSON outputs newline-delimited JSON events.
func (c *Context) streamJSON(fn StreamFunc) error {
	enc := json.NewEncoder(c.stdout)
	return fn(func(chunk string) {
		enc.Encode(map[string]any{
			"type":      "chunk",
			"content":   chunk,
			"timestamp": time.Now().Format(time.RFC3339Nano),
		})
	})
}

// streamInteractive displays chunks with a spinner in interactive mode.
func (c *Context) streamInteractive(fn StreamFunc) error {
	app := &streamApp{
		chunks: make([]string, 0),
	}

	var streamErr error
	done := make(chan struct{})

	go func() {
		streamErr = fn(func(chunk string) {
			app.mu.Lock()
			app.chunks = append(app.chunks, chunk)
			app.mu.Unlock()
		})
		app.mu.Lock()
		app.done = true
		app.mu.Unlock()
		close(done)
	}()

	// Run in alternate screen mode
	if err := tui.Run(app,
		tui.WithFPS(30),
		tui.WithAlternateScreen(false),
	); err != nil {
		return err
	}

	<-done
	return streamErr
}

type streamApp struct {
	mu     sync.Mutex
	chunks []string
	done   bool
	frame  uint64
}

func (a *streamApp) View() tui.View {
	a.mu.Lock()
	defer a.mu.Unlock()

	text := strings.Join(a.chunks, "")
	if a.done {
		return tui.Text("%s", text)
	}

	return tui.VStack(
		tui.Text("%s", text),
		tui.Loading(a.frame),
	)
}

func (a *streamApp) HandleEvent(event tui.Event) []tui.Cmd {
	switch e := event.(type) {
	case tui.TickEvent:
		a.frame++
		a.mu.Lock()
		done := a.done
		a.mu.Unlock()
		if done {
			return []tui.Cmd{tui.Quit()}
		}
		return []tui.Cmd{tui.Tick(100 * time.Millisecond)}
	case tui.KeyEvent:
		if e.Key == tui.KeyCtrlC {
			return []tui.Cmd{tui.Quit()}
		}
	}
	return nil
}

// Progress provides progress tracking for long-running operations.
type Progress struct {
	ctx       *Context
	message   string
	text      strings.Builder
	current   int
	total     int
	completed bool
	mu        sync.Mutex
}

// WithProgress executes a function with progress tracking.
// In TTY mode: displays an animated spinner with the message.
// In pipe mode: outputs text directly without animation.
func (c *Context) WithProgress(message string, fn func(*Progress) error) error {
	p := &Progress{
		ctx:     c,
		message: message,
	}

	if !c.Interactive() {
		// Non-interactive: just run the function
		return fn(p)
	}

	// Interactive mode: show progress UI
	app := &progressApp{
		progress: p,
	}

	var fnErr error
	done := make(chan struct{})

	go func() {
		fnErr = fn(p)
		p.mu.Lock()
		p.completed = true
		p.mu.Unlock()
		close(done)
	}()

	if err := tui.Run(app,
		tui.WithFPS(30),
		tui.WithAlternateScreen(false),
	); err != nil {
		return err
	}

	<-done
	return fnErr
}

// Append adds text to the progress output.
func (p *Progress) Append(text string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.text.WriteString(text)

	// In non-interactive mode, output immediately
	if !p.ctx.Interactive() {
		fmt.Fprint(p.ctx.stdout, text)
	}
}

// SetMessage updates the progress message.
func (p *Progress) SetMessage(msg string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.message = msg
}

// SetProgress updates the progress values.
func (p *Progress) SetProgress(current, total int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.current = current
	p.total = total
}

// Complete marks the progress as complete.
func (p *Progress) Complete() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.completed = true
}

// Write implements io.Writer for Progress.
func (p *Progress) Write(data []byte) (int, error) {
	p.Append(string(data))
	return len(data), nil
}

// Writer returns an io.Writer that appends to the progress.
func (p *Progress) Writer() io.Writer {
	return p
}

type progressApp struct {
	progress *Progress
	frame    uint64
}

func (a *progressApp) View() tui.View {
	a.progress.mu.Lock()
	defer a.progress.mu.Unlock()

	views := make([]tui.View, 0)

	// Show accumulated text
	text := a.progress.text.String()
	if text != "" {
		views = append(views, tui.Text("%s", text))
	}

	if a.progress.completed {
		return tui.VStack(views...)
	}

	// Show progress indicator
	if a.progress.total > 0 {
		views = append(views, tui.HStack(
			tui.Loading(a.frame),
			tui.Text(" %s (%d/%d)", a.progress.message, a.progress.current, a.progress.total),
		).Gap(1))
	} else {
		views = append(views, tui.HStack(
			tui.Loading(a.frame),
			tui.Text(" %s", a.progress.message),
		).Gap(1))
	}

	return tui.VStack(views...)
}

func (a *progressApp) HandleEvent(event tui.Event) []tui.Cmd {
	switch e := event.(type) {
	case tui.TickEvent:
		a.frame++
		a.progress.mu.Lock()
		done := a.progress.completed
		a.progress.mu.Unlock()
		if done {
			return []tui.Cmd{tui.Quit()}
		}
		return []tui.Cmd{tui.Tick(100 * time.Millisecond)}
	case tui.KeyEvent:
		if e.Key == tui.KeyCtrlC {
			return []tui.Cmd{tui.Quit()}
		}
	}
	return nil
}
