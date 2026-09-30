package tui

import (
	"errors"
	"io"
	"strings"
	"testing"
)

type clearOutputProbe struct {
	writes, flushes int
	short           bool
}

func (w *clearOutputProbe) Write(data []byte) (int, error) {
	w.writes++
	if w.short {
		return len(data) - 1, nil
	}
	return len(data), nil
}
func (w *clearOutputProbe) Flush() error { w.flushes++; return errors.New("clear must not flush") }
func TestLiveClearChecksWritesWithoutDraining(t *testing.T) {
	out := &clearOutputProbe{}
	printer := NewLivePrinter(WithOutput(out))
	if err := printer.clearChecked(); err != nil || out.writes != 0 || out.flushes != 0 {
		t.Fatalf("empty clear: %v %#v", err, out)
	}
	printer.started = true
	printer.lastHeight = 2
	if err := printer.clearChecked(); err != nil || out.writes != 1 || out.flushes != 0 {
		t.Fatalf("clear: %v %#v", err, out)
	}
	printer.started = true
	printer.lastHeight = 1
	out.short = true
	if err := printer.clearChecked(); !errors.Is(err, io.ErrShortWrite) || !printer.started {
		t.Fatalf("short clear: %v", err)
	}
}

type cleanupOutputProbe struct {
	text     []byte
	flushErr error
}

func (w *cleanupOutputProbe) Write(data []byte) (int, error) {
	w.text = append(w.text, data...)
	return len(data), nil
}
func (w *cleanupOutputProbe) Flush() error { return w.flushErr }
func TestInlineCleanupDoesNotRepeatCompletedKeyboardPop(t *testing.T) {
	out := &cleanupOutputProbe{flushErr: errors.New("flush failed")}
	app := NewInlineApp(WithInlineOutput(out), WithInlineKittyKeyboard(true))
	app.kittyEnabled = true // Includes an attempted startup enable.
	if err := app.releaseKeyboard(); err != nil {
		t.Fatal(err)
	}
	// Run owns one resize watcher and performs final cleanup once, after a
	// handoff may already have released the keyboard protocol.
	app.setupResizeWatcher()
	if err := app.cleanup(); !errors.Is(err, out.flushErr) {
		t.Fatal(err)
	}
	if strings.Count(string(out.text), "\x1b[<u") != 1 {
		t.Fatalf("repeated keyboard pops=%q", out.text)
	}
}
