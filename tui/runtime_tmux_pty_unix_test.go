//go:build darwin || linux

package tui

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/creack/pty"
	"golang.org/x/term"
)

type tmuxKeysApplication struct {
	report *os.File
	ready  bool
}

func (a *tmuxKeysApplication) View() View {
	if !a.ready {
		a.ready = true
		fmt.Fprintln(a.report, "ready")
	}
	return Text("tmux keys")
}

func (a *tmuxKeysApplication) HandleEvent(event Event) []Cmd {
	key, ok := event.(KeyEvent)
	if !ok {
		return nil
	}
	if key.Key == KeyCtrlC {
		fmt.Fprintln(a.report, "quit")
		return []Cmd{Quit()}
	}
	fmt.Fprintf(a.report, "key=%d shift=%v\n", key.Key, key.Shift)
	return nil
}

func TestRuntimeTmuxPTYHelper(t *testing.T) {
	if os.Getenv("WONTON_TMUX_PTY_HELPER") == "" {
		t.Skip("subprocess helper")
	}
	// Set here, not by the parent: TestMain clears them in every process.
	t.Setenv("TMUX", "/tmp/tmux-test/default,1,0")
	t.Setenv("TERM", "tmux-256color")
	report := os.NewFile(3, "report")
	defer report.Close()
	terminal, err := NewTerminal()
	if err != nil {
		t.Fatal(err)
	}
	defer terminal.Close()
	terminal.EnableAlternateScreen()
	// No SetKittyKeyboard: inside tmux the runtime turns enhanced keys on
	// by itself, since tmux is never probed.
	if err := NewRuntime(terminal, &tmuxKeysApplication{report: report}, 30).Run(); err != nil {
		t.Fatal(err)
	}
}

// Inside tmux, a fullscreen runtime asks for modifyOtherKeys, reads
// Shift+Enter in both forms tmux sends, and resets the mode on exit.
func TestRuntimeEnablesTmuxExtendedKeys(t *testing.T) {
	readReport, writeReport, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer readReport.Close()
	cmd := exec.Command(os.Args[0], "-test.run=^TestRuntimeTmuxPTYHelper$", "-test.timeout=10s")
	cmd.Env = append(os.Environ(), "WONTON_TMUX_PTY_HELPER=1")
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
	var output bytes.Buffer
	outputDone := make(chan struct{})
	go func() { defer close(outputDone); _, _ = io.Copy(&output, master) }()

	lines := make(chan string, 16)
	go func() {
		scanner := bufio.NewScanner(readReport)
		for scanner.Scan() {
			lines <- scanner.Text()
		}
		close(lines)
	}()
	next := func() string {
		t.Helper()
		select {
		case line := <-lines:
			return line
		case <-time.After(5 * time.Second):
			t.Fatal("timed out waiting for the helper")
			return ""
		}
	}

	if line := next(); line != "ready" {
		t.Fatalf("first report = %q", line)
	}
	enter := fmt.Sprintf("key=%d", KeyEnter)
	for _, tc := range []struct{ input, want string }{
		{"\x1b[27;2;13~", enter + " shift=true"}, // tmux 3.5+ default format
		{"\r", enter + " shift=false"},
		{"\x1b[13;2u", enter + " shift=true"}, // tmux 3.2-3.4, or csi-u format
		{"\x03", "quit"},
	} {
		if _, err := master.Write([]byte(tc.input)); err != nil {
			t.Fatal(err)
		}
		if got := next(); got != tc.want {
			t.Fatalf("input %q reported %q, want %q", tc.input, got, tc.want)
		}
	}
	if err := cmd.Wait(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-outputDone:
	case <-time.After(time.Second):
		t.Fatal("terminal output did not settle")
	}

	out := output.String()
	on := strings.Index(out, "\x1b[>1u\x1b[>4;1m")
	off := strings.LastIndex(out, "\x1b[<u\x1b[>4m")
	if on < 0 || off < on || strings.LastIndex(out, "\x1b[>4;1m") > off {
		t.Fatalf("modifyOtherKeys not enabled then reset: %q", out)
	}
}

// runOnTestTerminal runs a full-screen app with tui.Run on a test terminal,
// types 'q', and returns what Run wrote. rawDuringRun reports whether the
// process's stdin was in raw mode while the app drew.
func runOnTestTerminal(t *testing.T) (out string, rawDuringRun bool) {
	t.Helper()
	before, _ := term.GetState(int(os.Stdin.Fd()))
	var buf bytes.Buffer
	app := &stdinCheckApp{check: func() {
		now, _ := term.GetState(int(os.Stdin.Fd()))
		rawDuringRun = !reflect.DeepEqual(before, now)
	}}
	keys := NewTestInput()
	keys.Type("q")
	keys.Close()
	if err := Run(app, WithTerminal(NewTestTerminal(20, 3, &buf)), WithInput(keys)); err != nil {
		t.Fatal(err)
	}
	// How many frames with no change the runtime draws depends on timing.
	return emptyFrame.ReplaceAllString(buf.String(), ""), rawDuringRun
}

// emptyFrame matches a synchronized-output frame that changes no cell.
var emptyFrame = regexp.MustCompile(`\x1b\[\?2026h(\x1b\[\d+;\d+H)?(\x1b\[0m)?\x1b\[\?2026l`)

// stdinCheckApp draws one line and runs check on its first frame.
type stdinCheckApp struct{ check func() }

func (a *stdinCheckApp) View() View {
	if a.check != nil {
		a.check()
		a.check = nil
	}
	return Text("hello")
}

func TestRunOnTestTerminalPTYHelper(t *testing.T) {
	if os.Getenv("WONTON_TEST_TERMINAL_PTY_HELPER") == "" {
		t.Skip("subprocess helper")
	}
	// Set here, not by the parent: TestMain clears them in every process.
	t.Setenv("TMUX", "/tmp/tmux-test/default,1,0")
	t.Setenv("TERM", "tmux-256color")
	out, raw := runOnTestTerminal(t)
	report := os.NewFile(3, "report")
	defer report.Close()
	fmt.Fprintf(report, "%t %q\n", raw, out)
}

// With WithTerminal and a test terminal, Run writes the same bytes whether
// the test runs in a terminal inside tmux or in CI with no terminal, and it
// leaves the test's own terminal alone.
func TestRunOnTestTerminalIgnoresTheRealTerminal(t *testing.T) {
	readReport, writeReport, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer readReport.Close()
	cmd := exec.Command(os.Args[0], "-test.run=^TestRunOnTestTerminalPTYHelper$", "-test.timeout=10s")
	cmd.Env = append(os.Environ(), "WONTON_TEST_TERMINAL_PTY_HELPER=1")
	cmd.ExtraFiles = []*os.File{writeReport}
	master, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: 24, Cols: 80})
	writeReport.Close()
	if err != nil {
		t.Fatal(err)
	}
	defer master.Close()
	go func() { _, _ = io.Copy(io.Discard, master) }()

	reported := make(chan string, 1)
	go func() {
		line, _ := bufio.NewReader(readReport).ReadString('\n')
		reported <- strings.TrimSpace(line)
	}()
	var line string
	select {
	case line = <-reported:
	case <-time.After(10 * time.Second):
		_ = cmd.Process.Kill()
		t.Fatal("timed out waiting for the helper")
	}
	_ = cmd.Wait()

	var raw bool
	var inTmux string
	if _, err := fmt.Sscanf(line, "%t %q", &raw, &inTmux); err != nil {
		t.Fatalf("helper report %q: %v", line, err)
	}
	if raw {
		t.Error("Run put the test's own terminal in raw mode")
	}
	if strings.Contains(inTmux, "\x1b[>") {
		t.Errorf("Run asked a test terminal for enhanced keys: %q", inTmux)
	}
	plain, _ := runOnTestTerminal(t) // this process: TMUX cleared, maybe no tty
	if inTmux != plain {
		t.Errorf("output differs in a tty inside tmux:\n tmux: %q\nplain: %q", inTmux, plain)
	}
}
