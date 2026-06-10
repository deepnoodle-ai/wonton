package color_test

import (
	"os"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
	"github.com/deepnoodle-ai/wonton/color"
)

// clearEnvForTest unsets each env var for the duration of the test, using
// t.Setenv to capture the original value so it is restored on exit.
func clearEnvForTest(t *testing.T, keys ...string) {
	t.Helper()
	for _, k := range keys {
		t.Setenv(k, "")
		os.Unsetenv(k)
	}
}

func TestShouldColorize_RespectsNO_COLOR(t *testing.T) {
	// Isolate forcing env vars so this test is deterministic regardless of
	// the host environment.
	clearEnvForTest(t, "FORCE_COLOR", "CLICOLOR_FORCE", "CLICOLOR")

	t.Run("non-empty disables", func(t *testing.T) {
		t.Setenv("NO_COLOR", "1")
		assert.False(t, color.ShouldColorize(os.Stdout))
	})

	t.Run("empty does not disable", func(t *testing.T) {
		t.Setenv("NO_COLOR", "")
		// Result depends on TTY; just verify no panic.
		_ = color.ShouldColorize(os.Stdout)
	})

	t.Run("unset leaves TTY detection", func(t *testing.T) {
		clearEnvForTest(t, "NO_COLOR")
		_ = color.ShouldColorize(os.Stdout)
	})
}

func TestShouldColorize_Precedence(t *testing.T) {
	clearEnvForTest(t, "NO_COLOR", "FORCE_COLOR", "CLICOLOR", "CLICOLOR_FORCE")

	t.Run("FORCE_COLOR beats NO_COLOR", func(t *testing.T) {
		t.Setenv("NO_COLOR", "1")
		t.Setenv("FORCE_COLOR", "1")
		assert.True(t, color.ShouldColorize(os.Stdout))
	})

	t.Run("CLICOLOR_FORCE forces on", func(t *testing.T) {
		t.Setenv("NO_COLOR", "1")
		t.Setenv("CLICOLOR_FORCE", "1")
		assert.True(t, color.ShouldColorize(os.Stdout))
	})

	t.Run("FORCE_COLOR=0 does not force on", func(t *testing.T) {
		t.Setenv("NO_COLOR", "1")
		t.Setenv("FORCE_COLOR", "0")
		assert.False(t, color.ShouldColorize(os.Stdout))
	})

	t.Run("CLICOLOR=0 disables", func(t *testing.T) {
		t.Setenv("CLICOLOR", "0")
		assert.False(t, color.ShouldColorize(os.Stdout))
	})
}
