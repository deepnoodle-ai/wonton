//go:build linux

package pty_test

import (
	"strings"
	"testing"

	"github.com/deepnoodle-ai/wonton/pty"
)

// TestLinuxSlaveName asserts that on Linux, Open() returns a slave whose
// path is /dev/pts/N. This pins the TIOCGPTN + /dev/pts/ lookup path in
// pty_linux.go:open.
func TestLinuxSlaveName(t *testing.T) {
	master, slave, err := pty.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer master.Close()
	defer slave.Close()

	name := slave.Name()
	if !strings.HasPrefix(name, "/dev/pts/") {
		t.Errorf("slave.Name() = %q, want prefix /dev/pts/", name)
	}
}
