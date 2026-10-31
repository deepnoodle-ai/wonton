package tui

import (
	"bytes"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/deepnoodle-ai/wonton/assert"
	"github.com/deepnoodle-ai/wonton/termtest"
)

// WithInput must reach the Runtime that Run builds.
func TestRun_WithInputFeedsRuntime(t *testing.T) {
	source := NewTestInput()
	defer source.Close()
	cfg := defaultRunConfig()
	WithInput(source)(&cfg)

	var buf bytes.Buffer
	app := &testRuntimeModel{}
	runtime := newRunRuntime(NewTestTerminal(80, 24, &buf), app, cfg)
	go func() {
		time.Sleep(50 * time.Millisecond)
		source.Send(KeyEvent{Rune: 'q'})
	}()
	assert.NoError(t, runWithTimeout(t, runtime, 2*time.Second))
}

// keyCounter draws how many keys it read. It never quits.
type keyCounter struct{ keys int }

func (a *keyCounter) View() View { return Text("keys: %d", a.keys) }

func (a *keyCounter) HandleEvent(event Event) []Cmd {
	if _, ok := event.(KeyEvent); ok {
		a.keys++
	}
	return nil
}

// runReturns runs Run in a goroutine and returns its result, or fails the
// test if Run does not return within two seconds.
func runReturns(t *testing.T, run func() error) error {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- run() }()
	select {
	case err := <-done:
		return err
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return")
		return nil
	}
}

func TestRunReturnsWhenInputEnds(t *testing.T) {
	screen := termtest.NewScreen(20, 3)
	term := NewTestTerminal(20, 3, screen)
	keys := NewTestInput()
	keys.Type("ab")
	keys.Close()

	app := &keyCounter{}
	err := runReturns(t, func() error {
		return Run(app, WithTerminal(term), WithInput(keys), WithAlternateScreen(false))
	})
	assert.NoError(t, err)
	assert.Equal(t, app.keys, 2, "the app read every key before Run returned")
	termtest.AssertRow(t, screen, 0, "keys: 2") // the last frame was drawn
}

// failingInput fails its first read.
type failingInput struct{ err error }

func (f failingInput) ReadEvent() (Event, error) { return nil, f.err }
func (failingInput) SetPasteTabWidth(int)        {}

func TestRunReturnsTheInputError(t *testing.T) {
	broken := errors.New("device gone")
	term := NewTestTerminal(20, 3, io.Discard)
	err := runReturns(t, func() error {
		return Run(&keyCounter{}, WithTerminal(term), WithInput(failingInput{broken}))
	})
	assert.ErrorIs(t, err, broken)
}

func TestRunOnATestTerminalWithoutInputDrawsOnceAndReturns(t *testing.T) {
	screen := termtest.NewScreen(20, 3)
	term := NewTestTerminal(20, 3, screen)
	app := &keyCounter{}
	err := runReturns(t, func() error {
		return Run(app, WithTerminal(term), WithAlternateScreen(false))
	})
	assert.NoError(t, err)
	termtest.AssertRow(t, screen, 0, "keys: 0") // the first frame
}

func TestTestInputReadsEventsThenEOF(t *testing.T) {
	keys := NewTestInput()
	keys.Type("a")
	keys.Send(KeyEvent{Key: KeyEnter})
	keys.Close()
	keys.Close() // closing twice does nothing

	first, err := keys.ReadEvent()
	assert.NoError(t, err)
	assert.Equal(t, first.(KeyEvent).Rune, 'a')
	second, err := keys.ReadEvent()
	assert.NoError(t, err)
	assert.Equal(t, second.(KeyEvent).Key, KeyEnter)
	_, err = keys.ReadEvent()
	assert.ErrorIs(t, err, io.EOF)

	assert.Panics(t, func() { keys.Send(KeyEvent{Rune: 'b'}) })
}
