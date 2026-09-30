//go:build darwin || linux

package tui

import (
	"bufio"
	"encoding/json"
	"errors"
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
	scenario   string
	control    *os.File
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
			if a.scenario == "full-queue" {
				events := a.inlineEvents()
			fillQueue:
				for {
					select {
					case events <- TickEvent{}:
					default:
						break fillQueue
					}
				}
			}
			a.send("released", "", nil)
			if a.scenario == "stop" {
				var signal [1]byte
				if _, err := io.ReadFull(a.control, signal[:]); err != nil {
					return err
				}
				events := a.inlineEvents()
				for len(events) < cap(events) {
					events <- KeyEvent{Rune: 'q'}
				}
				if a.fullscreen != nil {
					a.fullscreen.Stop()
				} else {
					a.inline.Stop()
				}
				afterStop, err := handoffTestAttributes(int(os.Stdin.Fd()))
				if err != nil || !reflect.DeepEqual(afterStop, a.before) {
					return fmt.Errorf("stop changed child attributes: %v", err)
				}
				a.send("stopped", "", nil)
			}
			data := make([]byte, 6)
			if _, err := io.ReadFull(os.Stdin, data); err != nil {
				return err
			}
			a.send("child-read", string(data), nil)
			if a.scenario == "error" {
				return errors.New("fixture child failed")
			}
			if a.scenario == "panic" {
				panic("fixture child panic")
			}
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
		expected := active
		if a.scenario == "stop" {
			expected = a.before
		}
		if !reflect.DeepEqual(restored, expected) {
			restoreErr = fmt.Errorf("active attributes were not restored: %v", restoreErr)
		}
		a.send("restored", "", errors.Join(operationErr, restoreErr))
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
func (a *handoffPTYApplication) inlineEvents() chan Event {
	if a.fullscreen != nil {
		return a.fullscreen.events
	}
	return a.inline.events
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
	app := &handoffPTYApplication{report: report, before: original, scenario: os.Getenv("WONTON_HANDOFF_SCENARIO"), control: os.NewFile(4, "control")}
	defer app.control.Close()
	defer func() {
		if value := recover(); value != nil {
			plain, err := handoffTestAttributes(int(os.Stdin.Fd()))
			if err == nil && !reflect.DeepEqual(plain, original) {
				err = errors.New("panic cleanup left changed attributes")
			}
			app.send("run-panic", fmt.Sprint(value), err)
		}
	}()
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
	app.send("run-done", app.text, err)
	if err != nil {
		t.Fatal(err)
	}
}

func TestHandoffBothRunnersOwnChildInputAndRestore(t *testing.T) {
	for _, mode := range []string{"fullscreen", "inline"} {
		for _, scenario := range []string{"normal", "full-queue", "stop", "error", "panic"} {
			t.Run(mode+"/"+scenario, func(t *testing.T) {
				readReport, writeReport, err := os.Pipe()
				if err != nil {
					t.Fatal(err)
				}
				defer readReport.Close()
				cmd := exec.Command(os.Args[0], "-test.run=^TestHandoffPTYHelper$", "-test.timeout=10s")
				controlRead, controlWrite, err := os.Pipe()
				if err != nil {
					t.Fatal(err)
				}
				defer controlWrite.Close()
				cmd.Env = append(os.Environ(), "WONTON_HANDOFF_PTY_HELPER="+mode, "WONTON_HANDOFF_SCENARIO="+scenario)
				cmd.ExtraFiles = []*os.File{writeReport, controlRead}
				master, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: 24, Cols: 80})
				if err != nil {
					writeReport.Close()
					t.Fatal(err)
				}
				writeReport.Close()
				controlRead.Close()
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
						if !ok || message.Phase != phase || (message.Error != "" && !(scenario == "error" && phase == "restored" && message.Error == "fixture child failed")) {
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
				if err := pty.Setsize(master, &pty.Winsize{Rows: 30, Cols: 100}); err != nil {
					t.Fatal(err)
				}
				if scenario == "stop" {
					if _, err := controlWrite.Write([]byte{1}); err != nil {
						t.Fatal(err)
					}
					wait("stopped")
				}
				if _, err = master.Write([]byte("child\nRESIDUAL\n")); err != nil {
					t.Fatal(err)
				}
				if child := wait("child-read"); child.Text != "child\n" {
					t.Fatalf("child input=%q", child.Text)
				}
				if scenario == "panic" {
					if message := wait("run-panic"); !strings.Contains(message.Text, "fixture child panic") {
						t.Fatalf("panic=%q", message.Text)
					}
					if err := cmd.Wait(); err != nil {
						t.Fatal(err)
					}
					return
				}
				wait("restored")
				if scenario == "stop" {
					if message := wait("run-done"); message.Text != "" {
						t.Fatalf("pending events dispatched after stop: %q", message.Text)
					}
					if err := cmd.Wait(); err != nil {
						t.Fatal(err)
					}
					return
				}
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
