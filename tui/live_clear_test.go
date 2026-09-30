package tui

import (
	"errors"
	"io"
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
