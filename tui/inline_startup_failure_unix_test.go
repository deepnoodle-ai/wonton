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
	fullscreen := strings.HasPrefix(scenario, "fullscreen-")
	scenario = strings.TrimPrefix(scenario, "fullscreen-")
	out := &startupFailureOutput{scenario: scenario}
	var runErr error
	var running, keyboardEnabled, keyboardFraming bool
	if fullscreen {
		terminal := NewTestTerminal(80, 24, out)
		runtime := NewRuntime(terminal, &handoffPTYApplication{}, 30)
		runtime.SetKittyKeyboard(true)
		source := NewMockInputSource()
		source.Close()
		runtime.SetInputSource(source)
		runtime.Stop()
		runErr = runtime.Run()
		running, keyboardEnabled = runtime.running, terminal.IsKittyProtocolEnabled()
		if terminal.IsRawMode() {
			t.Fatal("Run returned in raw mode")
		}
		if (scenario == "zero" || scenario == "short") && strings.Count(out.text.String(), "\x18") != 1 {
			t.Fatalf("Run did not cancel incomplete framing: %q", out.text.String())
		}
		// Cleanup belongs to Run. No Terminal.Close call supplies a missing CAN.
	} else {
		app := NewInlineApp(WithInlineOutput(out), WithInlineBracketedPaste(true), WithInlineKittyKeyboard(true), WithInlineMouseTracking(true))
		runErr = app.Run(&handoffPTYApplication{})
		running, keyboardEnabled, keyboardFraming = app.running, app.kittyEnabled, app.kittyFraming
		if err := app.cleanup(); err != nil {
			t.Fatal(err)
		}
	}
	after, err := handoffTestAttributes(int(os.Stdin.Fd()))
	if err != nil {
		t.Fatal(err)
	}
	if (!fullscreen && !errors.Is(runErr, startupWriteFailure)) || (fullscreen && runErr != nil) || !reflect.DeepEqual(before, after) || running || keyboardEnabled || keyboardFraming {
		t.Fatalf("startup=%v original=%#v final=%#v modes=%v/%v", runErr, before, after, keyboardEnabled, keyboardFraming)
	}
	assertHandoffKeyboardStacks(t, out.text.String())
	if err := json.NewEncoder(report).Encode("restored"); err != nil {
		t.Fatal(err)
	}
}
func TestInlineStartupOutputFailuresRestoreTerminal(t *testing.T) {
	for _, scenario := range []string{"paste", "zero", "short", "full-error", "flush", "mouse", "fullscreen-zero", "fullscreen-short", "fullscreen-full-error"} {
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
