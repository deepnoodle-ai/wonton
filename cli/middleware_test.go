package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestConfirmMiddleware(t *testing.T) {
	t.Run("proceeds when user answers yes", func(t *testing.T) {
		executed := false
		handler := func(ctx *Context) error {
			executed = true
			return nil
		}

		stdin := strings.NewReader("yes\n")
		stdout := &bytes.Buffer{}

		ctx := &Context{
			interactive: true,
			stdin:       stdin,
			stdout:      stdout,
		}

		middleware := ConfirmBefore("Are you sure?")
		wrappedHandler := middleware(handler)

		err := wrappedHandler(ctx)
		assert.NoError(t, err)
		assert.True(t, executed)
		assert.Contains(t, stdout.String(), "Are you sure?")
	})

	t.Run("proceeds when user answers y", func(t *testing.T) {
		executed := false
		handler := func(ctx *Context) error {
			executed = true
			return nil
		}

		stdin := strings.NewReader("y\n")
		stdout := &bytes.Buffer{}

		ctx := &Context{
			interactive: true,
			stdin:       stdin,
			stdout:      stdout,
		}

		middleware := ConfirmBefore("Continue?")
		wrappedHandler := middleware(handler)

		err := wrappedHandler(ctx)
		assert.NoError(t, err)
		assert.True(t, executed)
	})

	t.Run("cancels when user answers no", func(t *testing.T) {
		executed := false
		handler := func(ctx *Context) error {
			executed = true
			return nil
		}

		stdin := strings.NewReader("no\n")
		stdout := &bytes.Buffer{}

		ctx := &Context{
			interactive: true,
			stdin:       stdin,
			stdout:      stdout,
		}

		middleware := ConfirmBefore("Delete all?")
		wrappedHandler := middleware(handler)

		err := wrappedHandler(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cancelled")
		assert.False(t, executed)
	})

	t.Run("cancels when user answers n", func(t *testing.T) {
		executed := false
		handler := func(ctx *Context) error {
			executed = true
			return nil
		}

		stdin := strings.NewReader("n\n")
		stdout := &bytes.Buffer{}

		ctx := &Context{
			interactive: true,
			stdin:       stdin,
			stdout:      stdout,
		}

		middleware := ConfirmBefore("Proceed?")
		wrappedHandler := middleware(handler)

		err := wrappedHandler(ctx)
		assert.Error(t, err)
		assert.False(t, executed)
	})

	t.Run("cancels when user presses enter", func(t *testing.T) {
		executed := false
		handler := func(ctx *Context) error {
			executed = true
			return nil
		}

		stdin := strings.NewReader("\n")
		stdout := &bytes.Buffer{}

		ctx := &Context{
			interactive: true,
			stdin:       stdin,
			stdout:      stdout,
		}

		middleware := ConfirmBefore("Dangerous operation?")
		wrappedHandler := middleware(handler)

		err := wrappedHandler(ctx)
		assert.Error(t, err)
		assert.False(t, executed)
	})

	t.Run("fails in non-interactive mode", func(t *testing.T) {
		executed := false
		handler := func(ctx *Context) error {
			executed = true
			return nil
		}

		ctx := &Context{
			interactive: false,
			stdin:       &bytes.Buffer{},
			stdout:      &bytes.Buffer{},
		}

		middleware := ConfirmBefore("Continue?")
		wrappedHandler := middleware(handler)

		err := wrappedHandler(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "non-interactively")
		assert.False(t, executed)
	})
}
