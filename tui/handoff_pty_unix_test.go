//go:build darwin || linux

package tui

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/creack/pty"
	"golang.org/x/sys/unix"
)

type handoffPTYMessage struct {
	Phase string
	Text  string
	Error string
}
type handoffPTYApplication struct {
	report     *os.File
	mu         sync.Mutex
	fullscreen *Runtime
	inline     *InlineApp
	before     *unix.Termios
	ready      bool
	text       string
}

func (a *handoffPTYApplication) send(phase, text string, err error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	message := handoffPTYMessage{Phase: phase, Text: text}
	if err != nil {
		message.Error = err.Error()
	}
	_ = json.NewEncoder(a.report).Encode(message)
}
func (a *handoffPTYApplication) View() View {
	if !a.ready {
		a.ready = true
		a.send("ready", "", nil)
	}
	return Text("Handoff fixture %s", a.text)
}
func (a *handoffPTYApplication) LiveView() View { return a.View() }
func (a *handoffPTYApplication) HandleEvent(event Event) []Cmd {
	if !a.ready {
		a.ready = true
		a.send("ready", "", nil)
	}
	key, ok := event.(KeyEvent)
	if !ok {
		return nil
	}
	switch key.Key {
	case KeyCtrlG:
		active, err := handoffTestAttributes(int(os.Stdin.Fd()))
		if err != nil {
			panic(err)
		}
		callback := func() error {
			plain, err := handoffTestAttributes(int(os.Stdin.Fd()))
			if err != nil {
				return err
			}
			if !reflect.DeepEqual(plain, a.before) {
				return fmt.Errorf("child attributes=%#v original=%#v", plain, a.before)
			}
			a.send("released", "", nil)
			data := make([]byte, 6)
			if _, err := io.ReadFull(os.Stdin, data); err != nil {
				return err
			}
			a.send("child-read", string(data), nil)
			return nil
		}
		var operationErr, restoreErr error
		if a.fullscreen != nil {
			operationErr, restoreErr = a.fullscreen.Handoff(callback)
		} else {
			operationErr, restoreErr = a.inline.Handoff(callback)
		}
		restored, err := handoffTestAttributes(int(os.Stdin.Fd()))
		if err != nil {
			panic(err)
		}
		if !reflect.DeepEqual(restored, active) {
			restoreErr = fmt.Errorf("active attributes were not restored: %v", restoreErr)
		}
		a.send("restored", "", errorsJoinForTest(operationErr, restoreErr))
	case KeyCtrlC:
		a.send("keys", a.text, nil)
		return []Cmd{Quit()}
	default:
		if key.Rune != 0 {
			a.text += string(key.Rune)
		}
	}
	return nil
}
func errorsJoinForTest(a, b error) error {
	if a != nil {
		return a
	}
	return b
}

func TestHandoffPTYHelper(t *testing.T) {
	mode := os.Getenv("WONTON_HANDOFF_PTY_HELPER")
	if mode == "" {
		t.Skip("subprocess helper")
	}
	report := os.NewFile(3, "report")
	defer report.Close()
	original, err := handoffTestAttributes(int(os.Stdin.Fd()))
	if err != nil {
		t.Fatal(err)
	}
	app := &handoffPTYApplication{report: report, before: original}
	if mode == "fullscreen" {
		terminal, err := NewTerminal()
		if err != nil {
			t.Fatal(err)
		}
		defer terminal.Close()
		terminal.EnableAlternateScreen()
		terminal.HideCursor()
		terminal.EnableMouseDrag()
		terminal.EnableBracketedPaste()
		app.fullscreen = NewRuntime(terminal, app, 30)
		app.fullscreen.SetKittyKeyboard(true)
		app.fullscreen.SetBackslashEnter(true)
		err = app.fullscreen.Run()
	} else {
		app.inline = NewInlineApp(WithInlineBracketedPaste(true), WithInlineKittyKeyboard(true), WithInlineMouseTracking(true), WithInlineBackslashEnter(true))
		err = app.inline.Run(app)
	}
	app.send("run-done", "", err)
	if err != nil {
		t.Fatal(err)
	}
}

func TestHandoffBothRunnersOwnChildInputAndRestore(t *testing.T) {
	for _, mode := range []string{"fullscreen", "inline"} {
		t.Run(mode, func(t *testing.T) {
			readReport, writeReport, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			defer readReport.Close()
			cmd := exec.Command(os.Args[0], "-test.run=^TestHandoffPTYHelper$", "-test.timeout=10s")
			cmd.Env = append(os.Environ(), "WONTON_HANDOFF_PTY_HELPER="+mode)
			cmd.ExtraFiles = []*os.File{writeReport}
			master, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: 24, Cols: 80})
			if err != nil {
				writeReport.Close()
				t.Fatal(err)
			}
			writeReport.Close()
			defer master.Close()
			defer func() {
				if cmd.ProcessState == nil {
					_ = cmd.Process.Kill()
					_ = cmd.Wait()
				}
			}()
			go io.Copy(io.Discard, master)
			messages := make(chan handoffPTYMessage, 16)
			go func() {
				scanner := bufio.NewScanner(readReport)
				for scanner.Scan() {
					var message handoffPTYMessage
					if json.Unmarshal(scanner.Bytes(), &message) == nil {
						messages <- message
					}
				}
				close(messages)
			}()
			wait := func(phase string) handoffPTYMessage {
				t.Helper()
				select {
				case message, ok := <-messages:
					if !ok || message.Phase != phase || message.Error != "" {
						t.Fatalf("expected %s got %+v", phase, message)
					}
					return message
				case <-time.After(5 * time.Second):
					t.Fatalf("waiting for %s", phase)
					return handoffPTYMessage{}
				}
			}
			wait("ready")
			if _, err = master.Write([]byte("\x07abc\xe2\x82")); err != nil {
				t.Fatal(err)
			}
			wait("released")
			if _, err = master.Write([]byte("child\nRESIDUAL\n")); err != nil {
				t.Fatal(err)
			}
			if child := wait("child-read"); child.Text != "child\n" {
				t.Fatalf("child input=%q", child.Text)
			}
			wait("restored")
			if _, err = master.Write([]byte("x\r\x03")); err != nil {
				t.Fatal(err)
			}
			if keys := wait("keys"); keys.Text != "abcx" {
				t.Fatalf("application keys=%q", keys.Text)
			}
			wait("run-done")
			if err = cmd.Wait(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestHandoffUnsupportedAndNotRunningDoNotInvokeCallback(t *testing.T) {
	app := &testRuntimeModel{}
	terminal := NewTestTerminal(80, 24, io.Discard)
	runtime := NewRuntime(terminal, app, 30)
	callback := func() error { t.Fatal("unsupported callback invoked"); return nil }
	if operation, restore := runtime.Handoff(nil); operation != nil || restore != nil {
		t.Fatal(operation, restore)
	}
	if operation, restore := runtime.Handoff(callback); operation != ErrHandoffNotRunning || restore != nil {
		t.Fatal(operation, restore)
	}
	runtime.running = true
	if operation, restore := runtime.Handoff(callback); operation != ErrHandoffUnsupported || restore != nil {
		t.Fatal(operation, restore)
	}
	inline := NewInlineApp(WithInlineOutput(&strings.Builder{}))
	inline.running = true
	if operation, restore := inline.Handoff(callback); operation != ErrHandoffUnsupported || restore != nil {
		t.Fatal(operation, restore)
	}
}
