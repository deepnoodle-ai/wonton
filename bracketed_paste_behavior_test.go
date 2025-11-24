package gooey

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestBracketedPastePreservesNewlines verifies that newlines in paste
// are preserved in the buffer and don't trigger form submission
func TestBracketedPastePreservesNewlines(t *testing.T) {
	// Simulate pasting "line1\nline2\n" followed by Enter
	input := "\033[200~line1\nline2\n\033[201~\r"
	reader := strings.NewReader(input)
	decoder := NewKeyDecoder(reader)

	// First event: paste containing newlines
	event1, err := decoder.ReadKeyEvent()
	require.NoError(t, err)
	require.Equal(t, "line1\nline2\n", event1.Paste)
	require.NotEmpty(t, event1.Paste, "Should be a paste event")

	// Second event: actual Enter keypress
	event2, err := decoder.ReadKeyEvent()
	require.NoError(t, err)
	require.Equal(t, KeyEnter, event2.Key)
	require.Empty(t, event2.Paste, "Enter keypress should not be a paste")
}

// TestBracketedPasteSecurity demonstrates security benefit
func TestBracketedPasteSecurity(t *testing.T) {
	// Malicious paste: command with embedded newlines
	malicious := "safe_command\nrm -rf /\n"

	// WITH bracketed paste
	withBracketed := "\033[200~" + malicious + "\033[201~"
	reader := strings.NewReader(withBracketed)
	decoder := NewKeyDecoder(reader)

	event, err := decoder.ReadKeyEvent()
	require.NoError(t, err)

	// The entire malicious content is in ONE event
	require.Equal(t, malicious, event.Paste)
	require.NotEmpty(t, event.Paste)

	// Application can now:
	// 1. Display the content for review
	// 2. Sanitize or validate it
	// 3. Require explicit confirmation
	// 4. Block dangerous commands
	require.Contains(t, event.Paste, "rm -rf")
	// Application code could detect this and warn the user!
}
