//go:build darwin || linux

package tui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"

	"github.com/creack/pty"
)

type panicLiveViewApp struct{}

func (panicLiveViewApp) LiveView() View { panic("first render failed") }

func TestInlineFirstRenderPanicPTYHelper(t *testing.T) {
	scenario := os.Getenv("WONTON_INLINE_RENDER_PANIC")
	if scenario == "" {
		t.Skip("subprocess helper")
	}
	report := os.NewFile(3, "report")
	defer report.Close()
	fail := func(format string, args ...any) {
		_ = json.NewEncoder(report).Encode(fmt.Sprintf(format, args...))
		t.Fatalf(format, args...)
	}
	before, err := handoffTestAttributes(int(os.Stdin.Fd()))
	if err != nil {
		fail("attributes: %v", err)
	}
	var out bytes.Buffer
	opts := []InlineOption{WithInlineMouseTracking(true), WithInlineBracketedPaste(true)}
	if scenario == "buffer" {
		opts = append(opts, WithInlineOutput(&out))
	} // "tty" keeps stdout, so Run also sets up managed terminal input.
	app := NewInlineApp(opts...)
	recovered := func() (rec any) {
		defer func() { rec = recover() }()
		_ = app.Run(panicLiveViewApp{})
		return nil
	}()
	after, err := handoffTestAttributes(int(os.Stdin.Fd()))
	if err != nil {
		fail("attributes: %v", err)
	}
	if recovered == nil || !strings.Contains(fmt.Sprint(recovered), "first render failed") {
		fail("Run did not re-panic with the render panic: %v", recovered)
	}
	if !reflect.DeepEqual(before, after) {
		fail("terminal left in raw mode: before=%#v after=%#v", before, after)
	}
	if app.running {
		fail("InlineApp still marked running")
	}
	for _, seq := range []string{"\x1b[?1000l", "\x1b[?2004l"} {
		if scenario == "buffer" && !strings.Contains(out.String(), seq) {
			fail("cleanup did not write %q: %q", seq, out.String())
		}
	}
	_ = json.NewEncoder(report).Encode("restored")
}

// A panic in the first LiveView call restores the terminal before Run
// re-panics, like a panic on any later frame.
func TestInlineFirstRenderPanicRestoresTerminal(t *testing.T) {
	for _, scenario := range []string{"buffer", "tty"} {
		t.Run(scenario, func(t *testing.T) { runInlineFirstRenderPanic(t, scenario) })
	}
}

func runInlineFirstRenderPanic(t *testing.T, scenario string) {
	readReport, writeReport, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer readReport.Close()
	cmd := exec.Command(os.Args[0], "-test.run=^TestInlineFirstRenderPanicPTYHelper$", "-test.timeout=10s")
	cmd.Env = append(os.Environ(), "WONTON_INLINE_RENDER_PANIC="+scenario)
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
		t.Fatalf("helper sent no report: %v", err)
	}
	if report != "restored" {
		t.Fatal(report)
	}
	_ = cmd.Wait()
}
