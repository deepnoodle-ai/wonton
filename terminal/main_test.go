package terminal

import (
	"os"
	"strings"
	"testing"
)

// TestMain keeps tests independent of the developer's terminal. Inside tmux
// the enhanced-keyboard sequences differ, so tests that expect the plain ones
// would fail; tests that need tmux set the variables themselves.
func TestMain(m *testing.M) {
	_ = os.Unsetenv("TMUX")
	if os.Getenv("TERM_PROGRAM") == "tmux" {
		_ = os.Unsetenv("TERM_PROGRAM")
	}
	if strings.HasPrefix(os.Getenv("TERM"), "tmux") {
		_ = os.Setenv("TERM", "xterm-256color")
	}
	os.Exit(m.Run())
}
