//go:build darwin || linux

package terminal

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/creack/pty"
	"golang.org/x/term"
)

type restorationFaultWriter struct {
	strings.Builder
	call, failAt, bytes int
}

func (w *restorationFaultWriter) Write(data []byte) (int, error) {
	w.call++
	if w.call == w.failAt {
		n := min(w.bytes, len(data))
		w.Builder.Write(data[:n])
		return n, io.ErrShortWrite
	}
	return w.Builder.Write(data)
}

// Override StringWriter so io.WriteString uses the fault-aware Write method.
func (w *restorationFaultWriter) WriteString(text string) (int, error) { return w.Write([]byte(text)) }
func TestHandoffTracksEachRestorationControlBeforeShortWrite(t *testing.T) {
	for failAt := 1; failAt <= 3; failAt++ {
		for count := 0; count <= 16; count++ {
			t.Run(fmt.Sprintf("write%d-bytes%d", failAt, count), func(t *testing.T) {
				master, slave, err := pty.Open()
				if err != nil {
					t.Fatal(err)
				}
				defer master.Close()
				defer slave.Close()
				original, err := term.MakeRaw(int(slave.Fd()))
				if err != nil {
					t.Fatal(err)
				}
				defer term.Restore(int(slave.Fd()), original)
				out := &restorationFaultWriter{}
				terminal := &Terminal{fd: int(slave.Fd()), out: out, oldState: original, rawMode: true, altScreen: true, bracketedPaste: true, kittyEnabled: true}
				transition, err := terminal.handoffSnapshot()
				if err != nil {
					t.Fatal(err)
				}
				if err := transition.Release(); err != nil {
					t.Fatal(err)
				}
				out.Reset()
				out.call = 0
				out.failAt = failAt
				out.bytes = count
				if err := transition.Restore(false); !errors.Is(err, io.ErrShortWrite) {
					t.Fatalf("restore=%v", err)
				}
				out.failAt = 0
				if err := transition.Restore(true); err != nil {
					t.Fatal(err)
				}
				if err := transition.Restore(true); err != nil {
					t.Fatal(err)
				}
				if terminal.altScreen || terminal.kittyEnabled || terminal.kittyFraming || terminal.rawMode {
					t.Fatalf("unrestored modes=%+v", terminal)
				}
				text := out.String()
				if h, l := strings.Count(text, "\x1b[?1049h"), strings.Count(text, "\x1b[?1049l"); h != l {
					t.Fatalf("screen changes h=%d l=%d: %q", h, l, text)
				}
				if pushes, pops := strings.Count(text, "\x1b[>1u"), strings.Count(text, "\x1b[<u"); pushes != pops {
					t.Fatalf("keyboard changes push=%d pop=%d: %q", pushes, pops, text)
				}
			})
		}
	}
}
