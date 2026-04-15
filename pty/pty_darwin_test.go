//go:build darwin

package pty_test

import (
	"strings"
	"testing"

	"github.com/deepnoodle-ai/wonton/pty"
)

// TestDarwinSlaveName asserts that on Darwin, Open() returns a slave whose
// path is /dev/ttys<nnn>. This pins the TIOCPTYGNAME lookup path in
// pty_darwin.go:open.
func TestDarwinSlaveName(t *testing.T) {
	master, slave, err := pty.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer master.Close()
	defer slave.Close()

	name := slave.Name()
	if !strings.HasPrefix(name, "/dev/ttys") {
		t.Errorf("slave.Name() = %q, want prefix /dev/ttys", name)
	}
}
