//go:build darwin || linux

package tui

import (
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/creack/pty"
	"github.com/deepnoodle-ai/wonton/internal/terminalinput"
	"golang.org/x/term"
)

func TestManagedInputBoundaryRetainsOwnedEventsAndExcludesChild(t *testing.T) {
	for _, tc := range []struct {
		name, suffix string
		tail         KeyEvent
		haveTail     bool
	}{
		{name: "idle"},
		{name: "utf8", suffix: "\xe2\x82"},
		{name: "csi", suffix: "\x1b[1;"},
		{name: "ss3", suffix: "\x1bO"},
		{name: "escape", suffix: "\x1b", tail: KeyEvent{Key: KeyEscape}, haveTail: true},
		{name: "paste", suffix: "\x1b[200~p\r\nq\x1b[20", tail: KeyEvent{Paste: "p\nq"}, haveTail: true},
		{name: "backslash", suffix: "\\", tail: KeyEvent{Rune: '\\'}, haveTail: true},
		{name: "backslash_enter", suffix: "\\\r", tail: KeyEvent{Key: KeyEnter, Shift: true}, haveTail: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			master, slave, err := pty.Open()
			if err != nil {
				t.Fatal(err)
			}
			defer master.Close()
			defer slave.Close()
			state, err := term.MakeRaw(int(slave.Fd()))
			if err != nil {
				t.Fatal(err)
			}
			defer term.Restore(int(slave.Fd()), state)
			owner, err := terminalinput.New(slave)
			if err != nil {
				t.Fatal(err)
			}
			events := make(chan Event, 1)
			done := make(chan struct{})
			settled := make(chan struct{})
			input := newManagedInput(owner, events, done, 0, true, nil, nil)
			go func() { defer close(settled); input.run() }()
			defer func() {
				close(done)
				select {
				case <-settled:
				case <-time.After(time.Second):
					t.Error("managed reader did not stop")
				}
			}()
			if _, err = master.Write([]byte("abc" + tc.suffix)); err != nil {
				t.Fatal(err)
			}
			first := nextManagedEvent(t, events)
			if first.(KeyEvent).Rune != 'a' {
				t.Fatalf("first=%+v", first)
			}
			paused := make(chan error, 1)
			go func() { paused <- input.pause() }()
			select {
			case err = <-paused:
				if err != nil {
					t.Fatal(err)
				}
			case <-time.After(time.Second):
				t.Fatal("pause blocked behind the full application queue")
			}
			// The same terminal remains open, with no application read outstanding.
			if _, err = master.Write([]byte("child\n")); err != nil {
				t.Fatal(err)
			}
			childRead := make(chan string, 1)
			go func() {
				bytes := make([]byte, 6)
				_, err := io.ReadFull(slave, bytes)
				childRead <- fmt.Sprintf("%s|%v", bytes, err)
			}()
			select {
			case child := <-childRead:
				if child != "child\n|<nil>" {
					t.Fatalf("child read=%q", child)
				}
			case <-time.After(time.Second):
				t.Fatal("application captured child input")
			}
			if _, err = master.Write([]byte("RESIDUAL")); err != nil {
				t.Fatal(err)
			}
			if err = owner.DiscardResidual(); err != nil {
				t.Fatal(err)
			}
			if err = input.resume(); err != nil {
				t.Fatal(err)
			}
			for _, want := range []rune{'b', 'c'} {
				if got := nextManagedEvent(t, events).(KeyEvent); got.Rune != want {
					t.Fatalf("retained=%+v want=%c", got, want)
				}
			}
			if tc.haveTail {
				if got := nextManagedEvent(t, events).(KeyEvent); got.Key != tc.tail.Key || got.Rune != tc.tail.Rune || got.Paste != tc.tail.Paste || got.Shift != tc.tail.Shift {
					t.Fatalf("boundary tail=%+v want=%+v", got, tc.tail)
				}
			}
			if _, err = master.Write([]byte("x\r\x03")); err != nil {
				t.Fatal(err)
			}
			wants := []KeyEvent{{Rune: 'x'}, {Key: KeyEnter}, {Key: KeyCtrlC}}
			for _, want := range wants {
				got, ok := nextManagedEvent(t, events).(KeyEvent)
				if !ok || got.Key != want.Key || got.Rune != want.Rune {
					t.Fatalf("resumed=%+v want=%+v", got, want)
				}
			}
		})
	}
}
func nextManagedEvent(t *testing.T, events <-chan Event) Event {
	t.Helper()
	select {
	case event := <-events:
		return event
	case <-time.After(time.Second):
		t.Fatal("missing managed input event")
		return nil
	}
}
