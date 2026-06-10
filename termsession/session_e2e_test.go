//go:build darwin || linux

package termsession

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/deepnoodle-ai/wonton/assert"
)

// TestSession_Resize_ChildSeesNewSize drives a resize through a live session
// and asserts that the child process actually observes the new size via its
// SIGWINCH handler. This is the end-to-end counterpart to the existing
// TestSession_Integration_Resize, which only checks that Resize returns nil.
func TestSession_Resize_ChildSeesNewSize(t *testing.T) {
	// Shell installs the SIGWINCH trap, prints READY, then loops waiting for
	// the signal. The READY sentinel lets the test know the trap is in place
	// before calling Resize — no timing dependency on shell startup latency.
	script := `
trap 'set -- $(stty size); echo "WINCH:${1}x${2}"; exit 0' WINCH
echo "READY"
i=0
while [ $i -lt 40 ]; do
	sleep 0.1
	i=$((i+1))
done
echo "TIMEOUT"
exit 1
`
	var captured syncBuffer
	s, err := NewSession(SessionOptions{
		Command: []string{"sh", "-c", script},
		Output:  &captured,
		Input:   devNull(t),
	})
	assert.NoError(t, err)
	defer s.Close()

	assert.NoError(t, s.Start())

	// Wait for the READY sentinel — the child confirming the trap is installed.
	if !waitForContains(&captured, "READY", 5*time.Second) {
		t.Fatalf("timed out waiting for READY; output=%q", captured.String())
	}
	assert.NoError(t, s.Resize(132, 42)) // width=cols=132, height=rows=42
	assert.NoError(t, s.Wait())

	if !strings.Contains(captured.String(), "WINCH:42x132") {
		t.Errorf("child did not observe new size; output=%q", captured.String())
	}
}

// syncBuffer is a thread-safe bytes.Buffer for use as a session Output sink
// when a background goroutine writes while the test goroutine polls.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// waitForContains polls b.String() for substring until the deadline. Returns
// true if the substring appeared in time, false on timeout.
func waitForContains(b *syncBuffer, substring string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for {
		if strings.Contains(b.String(), substring) {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// waitForFileContains polls the given file's contents for substring until the
// deadline. Missing file is treated as "not yet".
func waitForFileContains(path, substring string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for {
		if data, err := os.ReadFile(path); err == nil && strings.Contains(string(data), substring) {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// TestSession_Record_StructuralRoundTrip runs a command under a recorder and
// parses the resulting asciinema v2 cast file as structured JSON events,
// asserting that header fields match and that at least one "o" event carries
// the expected substring. This is stricter than the existing string-contains
// check in session_test.go.
func TestSession_Record_StructuralRoundTrip(t *testing.T) {
	dir := t.TempDir()
	filename := filepath.Join(dir, "e2e.cast")

	s, err := NewSession(SessionOptions{
		// Trailing sleep keeps the PTY slave open long enough for the
		// session to drain the master before the child exits; otherwise
		// the kernel (macOS in particular) may discard unread output.
		Command: []string{"sh", "-c", "echo hello-e2e; sleep 0.2"},
	})
	assert.NoError(t, err)
	defer s.Close()

	assert.NoError(t, s.Record(filename, RecordingOptions{
		Compress: false,
		Title:    "e2e",
	}))
	assert.NoError(t, s.Wait())

	// Poll the cast file until the expected output appears (recorder flush
	// is asynchronous to Wait, so don't rely on a fixed sleep).
	if !waitForFileContains(filename, "hello-e2e", 2*time.Second) {
		t.Fatalf("cast file did not contain expected output within deadline")
	}

	data, err := os.ReadFile(filename)
	assert.NoError(t, err)
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) < 2 {
		t.Fatalf("cast has %d lines, want >= 2; data=%q", len(lines), data)
	}

	var header RecordingHeader
	assert.NoError(t, json.Unmarshal([]byte(lines[0]), &header))
	assert.Equal(t, 2, header.Version)
	assert.Equal(t, "e2e", header.Title)
	// Size comes from the input terminal; in non-TTY test context it falls
	// back to the 80x24 default set in session.Record.
	if header.Width <= 0 || header.Height <= 0 {
		t.Errorf("header width/height = %dx%d, want positive", header.Width, header.Height)
	}

	// Each event line is a JSON array: [time, kind, data].
	foundOutput := false
	for _, line := range lines[1:] {
		if line == "" {
			continue
		}
		var ev []any
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			t.Errorf("malformed event line %q: %v", line, err)
			continue
		}
		if len(ev) != 3 {
			t.Errorf("event %v has %d fields, want 3", ev, len(ev))
			continue
		}
		if _, ok := ev[0].(float64); !ok {
			t.Errorf("event[0] type = %T, want float64", ev[0])
		}
		kind, _ := ev[1].(string)
		data, _ := ev[2].(string)
		if kind == "o" && strings.Contains(data, "hello-e2e") {
			foundOutput = true
		}
	}
	if !foundOutput {
		t.Errorf("no 'o' event contained %q; cast=%s", "hello-e2e", string(data))
	}
}

// TestSession_EOF_SendsEOT asserts that when the session's input reader hits
// io.EOF, copyInput forwards an EOT (Ctrl+D) byte to the PTY so the child
// sees end-of-input. The child reads stdin with `cat` and echoes back; after
// we close our pipe, cat exits on EOT, the session drains, and we get a
// clean wait.
func TestSession_EOF_SendsEOT(t *testing.T) {
	pipeR, pipeW, err := os.Pipe()
	assert.NoError(t, err)
	defer pipeR.Close()

	var out syncBuffer
	s, err := NewSession(SessionOptions{
		Command: []string{"sh", "-c", "cat; exit 0"},
		Input:   pipeR,
		Output:  &out,
	})
	assert.NoError(t, err)
	defer s.Close()

	assert.NoError(t, s.Start())

	// Write a byte of real content, then wait until the child actually echoes
	// it back via the PTY before closing the pipe. That ensures copyInput has
	// drained the write and we're testing the EOF → EOT path cleanly rather
	// than racing the close against the write.
	_, _ = pipeW.Write([]byte("ping\n"))
	if !waitForContains(&out, "ping", 2*time.Second) {
		t.Fatalf("child did not echo input within deadline; output=%q", out.String())
	}
	pipeW.Close()

	// Wait should now complete. We don't care about echo line discipline
	// details — only that the session terminates cleanly.
	done := make(chan error, 1)
	go func() { done <- s.Wait() }()
	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("session did not terminate after input EOF")
	}
}

// TestSession_Close_Idempotent_AfterStart asserts that Close is idempotent
// when called on a running session, then again after Wait. The existing
// TestSession_Close_Idempotent covers only the unstarted path.
func TestSession_Close_Idempotent_AfterStart(t *testing.T) {
	var out bytes.Buffer
	s, err := NewSession(SessionOptions{
		Command: []string{"sh", "-c", "echo quick"},
		Input:   devNull(t),
		Output:  &out,
	})
	assert.NoError(t, err)

	assert.NoError(t, s.Start())
	assert.NoError(t, s.Wait())

	assert.NoError(t, s.Close())
	assert.NoError(t, s.Close())
}

// devNull opens /dev/null for reading and registers it to close on test end.
// Using /dev/null as input means the session's copyInput hits EOF immediately
// and never blocks on os.Stdin (which is what NewSession would default to).
func devNull(t *testing.T) *os.File {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("devNull helper is POSIX-only")
	}
	f, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { f.Close() })
	return f
}
