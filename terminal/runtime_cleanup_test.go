package terminal

import (
	"errors"
	"golang.org/x/term"
	"io"
	"testing"
)

type cleanupFailWriter struct{ err error }

func (w cleanupFailWriter) Write([]byte) (int, error) { return 0, w.err }
func TestRuntimeCleanupReportsProtocolAndAttributeFailures(t *testing.T) {
	writeErr := errors.New("protocol output failed")
	terminal := &Terminal{fd: -1, out: cleanupFailWriter{writeErr}, kittyEnabled: true, rawMode: true, oldState: &term.State{}}
	err := terminal.cleanupRuntime(true, true)
	if !errors.Is(err, writeErr) || len(err.(interface{ Unwrap() []error }).Unwrap()) != 2 {
		t.Fatalf("joined cleanup errors=%v", err)
	}
	if !terminal.kittyEnabled || !terminal.rawMode {
		t.Fatal("failed cleanup claimed restored modes")
	}
	terminal.out = io.Discard
	if err := terminal.cleanupRuntime(false, true); err != nil || terminal.kittyEnabled {
		t.Fatalf("retry cleanup=%v", err)
	}
}

type cleanupFlushWriter struct {
	output []byte
	err    error
}

func (w *cleanupFlushWriter) Write(data []byte) (int, error) {
	w.output = append(w.output, data...)
	return len(data), nil
}
func (w *cleanupFlushWriter) Flush() error { return w.err }
func TestRuntimeCleanupDoesNotRepeatCompletedKeyboardPopAfterFlushFailure(t *testing.T) {
	out := &cleanupFlushWriter{err: errors.New("flush failed")}
	terminal := &Terminal{out: out, kittyEnabled: true}
	if err := terminal.cleanupRuntime(false, true); !errors.Is(err, out.err) {
		t.Fatal(err)
	}
	if err := terminal.cleanupRuntime(false, true); err != nil {
		t.Fatal(err)
	}
	if string(out.output) != "\x1b[<u" {
		t.Fatalf("repeated keyboard pop=%q", out.output)
	}
}
