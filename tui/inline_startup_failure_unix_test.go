//go:build darwin || linux

package tui

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"

	"github.com/creack/pty"
)

type startupFailureOutput struct {
	scenario     string
	text         strings.Builder
	failed       bool
	flushFailure bool
}

var startupWriteFailure = errors.New("startup output failed")

func (w *startupFailureOutput) Write(data []byte) (int, error) {
	text := string(data)
	target := "\x1b[>1u"
	if w.scenario == "paste" {
		target = "\x1b[?2004h"
	}
	if w.scenario == "mouse" {
		target = "\x1b[?1000h\x1b[?1006h"
	}
	if !w.failed && text == target && w.scenario != "flush" {
		w.failed = true
		n := 0
		switch w.scenario {
		case "short":
			n = len(data) - 1
		case "full-error":
			n = len(data)
		}
		w.text.Write(data[:n])
		return n, startupWriteFailure
	}
	w.text.Write(data)
	if text == target && w.scenario == "flush" {
		w.flushFailure = true
	}
	return len(data), nil
}
func (w *startupFailureOutput) Flush() error {
	if w.flushFailure {
		w.flushFailure = false
		w.failed = true
		return startupWriteFailure
	}
	return nil
}
func TestInlineStartupFailurePTYHelper(t *testing.T) {
	scenario := os.Getenv("WONTON_INLINE_STARTUP_FAILURE")
	if scenario == "" {
		t.Skip("subprocess helper")
	}
	report := os.NewFile(3, "report")
	defer report.Close()
	before, err := handoffTestAttributes(int(os.Stdin.Fd()))
	if err != nil {
		t.Fatal(err)
	}
	out := &startupFailureOutput{scenario: scenario}
	app := NewInlineApp(WithInlineOutput(out), WithInlineBracketedPaste(true), WithInlineKittyKeyboard(true), WithInlineMouseTracking(true))
	runErr := app.Run(&handoffPTYApplication{})
	after, err := handoffTestAttributes(int(os.Stdin.Fd()))
	if err != nil {
		t.Fatal(err)
	}
	if !errors.Is(runErr, startupWriteFailure) || !reflect.DeepEqual(before, after) || app.running || app.kittyEnabled || app.kittyFraming {
		t.Fatalf("startup=%v original=%#v final=%#v modes=%v/%v", runErr, before, after, app.kittyEnabled, app.kittyFraming)
	}
	assertHandoffKeyboardStacks(t, out.text.String())
	// Cleanup is also safe before watcher setup and after a previous cleanup.
	if err := app.cleanup(); err != nil {
		t.Fatal(err)
	}
	if err := json.NewEncoder(report).Encode("restored"); err != nil {
		t.Fatal(err)
	}
}
func TestInlineStartupOutputFailuresRestoreTerminal(t *testing.T) {
	for _, scenario := range []string{"paste", "zero", "short", "full-error", "flush", "mouse"} {
		t.Run(scenario, func(t *testing.T) {
			readReport, writeReport, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			defer readReport.Close()
			cmd := exec.Command(os.Args[0], "-test.run=^TestInlineStartupFailurePTYHelper$", "-test.timeout=10s")
			cmd.Env = append(os.Environ(), "WONTON_INLINE_STARTUP_FAILURE="+scenario)
			cmd.ExtraFiles = []*os.File{writeReport}
			master, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: 24, Cols: 80})
			writeReport.Close()
			if err != nil {
				t.Fatal(err)
			}
			defer master.Close()
			defer func() {
				if cmd.ProcessState == nil {
					_ = cmd.Process.Kill()
					_ = cmd.Wait()
				}
			}()
			go io.Copy(io.Discard, master)
			var report string
			if err := json.NewDecoder(readReport).Decode(&report); err != nil {
				t.Fatal(err)
			}
			if report != "restored" {
				t.Fatal(report)
			}
			if err := cmd.Wait(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
