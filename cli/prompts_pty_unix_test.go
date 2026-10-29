//go:build darwin || linux

package cli

import (
	"bytes"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/creack/pty"
	"github.com/deepnoodle-ai/wonton/termtest"
)

// TestInputSubmitsOnPTY runs ctx.Input on a real PTY, types a line, and ends
// it the two ways a terminal can: CR is Enter, and LF (Ctrl-J, or a terminal
// that sends LF for Enter) decodes as Shift+Enter. Both must submit.
func TestInputSubmitsOnPTY(t *testing.T) {
	for _, tc := range []struct{ name, enter string }{
		{"CR", "\r"},
		{"LF", "\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			master, slave, err := pty.Open()
			if err != nil {
				t.Skipf("pty unavailable: %v", err)
			}
			defer master.Close()
			defer slave.Close()
			if err := pty.Setsize(slave, &pty.Winsize{Rows: 24, Cols: 80}); err != nil {
				t.Fatal(err)
			}

			oldStdin, oldStdout := os.Stdin, os.Stdout
			os.Stdin, os.Stdout = slave, slave
			defer func() { os.Stdin, os.Stdout = oldStdin, oldStdout }()

			// Collect everything the prompt draws, and answer the startup
			// device attributes query the way a real terminal does.
			var mu sync.Mutex
			var out bytes.Buffer
			go func() {
				buf := make([]byte, 4096)
				answered := false
				for {
					n, err := master.Read(buf)
					mu.Lock()
					out.Write(buf[:n])
					if !answered && strings.Contains(out.String(), "\x1b[c") {
						answered = true
						master.WriteString("\x1b[?62;22c")
					}
					mu.Unlock()
					if err != nil {
						return
					}
				}
			}()
			screen := func() *termtest.Screen {
				mu.Lock()
				defer mu.Unlock()
				s := termtest.NewScreen(80, 24)
				s.Write(out.Bytes())
				return s
			}
			waitFor := func(text string) {
				t.Helper()
				deadline := time.Now().Add(5 * time.Second)
				for !screen().Contains(text) {
					if time.Now().After(deadline) {
						t.Fatalf("%q not on screen:\n%s", text, screen().Text())
					}
					time.Sleep(10 * time.Millisecond)
				}
			}

			var got string
			app := New("input-test").ForceInteractive(true)
			app.SetStdout(io.Discard)
			app.Command("ask").Run(func(ctx *Context) error {
				var err error
				got, err = ctx.Input("Enter your name:")
				return err
			})
			result := make(chan error, 1)
			go func() { result <- app.ExecuteArgs([]string{"ask"}) }()

			// Type only once the prompt is on screen, so the terminal is
			// already in raw mode and CR is not translated to LF.
			waitFor("Enter to submit")
			if _, err := master.WriteString("ab"); err != nil {
				t.Fatal(err)
			}
			waitFor("ab")
			if _, err := master.WriteString(tc.enter); err != nil {
				t.Fatal(err)
			}

			select {
			case err := <-result:
				if err != nil {
					t.Fatalf("Input: %v", err)
				}
			case <-time.After(3 * time.Second):
				master.WriteString("\x03") // Ctrl-C unblocks the prompt
				<-result
				t.Fatal("Enter did not submit the prompt")
			}
			if got != "ab" {
				t.Fatalf("Input returned %q, want %q", got, "ab")
			}
		})
	}
}
