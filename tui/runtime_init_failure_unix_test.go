//go:build darwin || linux

package tui

import (
	"encoding/json"
	"errors"
	"github.com/creack/pty"
	"golang.org/x/sys/unix"
	"io"
	"os"
	"os/exec"
	"reflect"
	"testing"
)

func TestRuntimeInputInitializationFailurePTYHelper(t *testing.T) {
	if os.Getenv("WONTON_INIT_FAILURE_HELPER") != "1" {
		t.Skip("subprocess helper")
	}
	report := os.NewFile(3, "report")
	defer report.Close()
	before, err := handoffTestAttributes(int(os.Stdin.Fd()))
	if err != nil {
		t.Fatal(err)
	}
	terminal, err := NewTerminal()
	if err != nil {
		t.Fatal(err)
	}
	defer terminal.Close()
	runtime := NewRuntime(terminal, &handoffPTYApplication{}, 30)
	runtime.SetKittyKeyboard(true)
	var limit unix.Rlimit
	if err := unix.Getrlimit(unix.RLIMIT_NOFILE, &limit); err != nil {
		t.Fatal(err)
	}
	exhausted := limit
	exhausted.Cur = 0
	if err := unix.Setrlimit(unix.RLIMIT_NOFILE, &exhausted); err != nil {
		t.Fatal(err)
	}
	runErr := runtime.Run() // Wake-pipe creation must fail after raw and Kitty startup.
	restoreErr := unix.Setrlimit(unix.RLIMIT_NOFILE, &limit)
	after, attrErr := handoffTestAttributes(int(os.Stdin.Fd()))
	if restoreErr != nil || attrErr != nil {
		t.Fatalf("restore limit=%v attributes=%v", restoreErr, attrErr)
	}
	if !errors.Is(runErr, unix.EMFILE) {
		t.Fatalf("setup error=%v", runErr)
	}
	if terminal.IsKittyProtocolEnabled() || terminal.IsRawMode() || runtime.running {
		t.Fatal("failed startup retained enabled runtime modes")
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("original=%#v after=%#v", before, after)
	}
	if err := json.NewEncoder(report).Encode("restored"); err != nil {
		t.Fatal(err)
	}
}

func TestRuntimeInputInitializationFailureRestoresTerminal(t *testing.T) {
	readReport, writeReport, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer readReport.Close()
	cmd := exec.Command(os.Args[0], "-test.run=^TestRuntimeInputInitializationFailurePTYHelper$", "-test.timeout=10s")
	cmd.Env = append(os.Environ(), "WONTON_INIT_FAILURE_HELPER=1")
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
}
