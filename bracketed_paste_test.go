package gooey

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestBracketedPasteSimple tests basic bracketed paste parsing
func TestBracketedPasteSimple(t *testing.T) {
	// Create input with bracketed paste sequence
	input := "\033[200~Hello, World!\033[201~"
	reader := strings.NewReader(input)
	decoder := NewKeyDecoder(reader)

	// Read the paste event
	event, err := decoder.ReadKeyEvent()
	require.NoError(t, err)
	require.Equal(t, "Hello, World!", event.Paste)
	require.Equal(t, rune(0), event.Rune)
}

// TestBracketedPasteMultiline tests paste with newlines
func TestBracketedPasteMultiline(t *testing.T) {
	content := "Line 1\nLine 2\nLine 3"
	input := "\033[200~" + content + "\033[201~"
	reader := strings.NewReader(input)
	decoder := NewKeyDecoder(reader)

	event, err := decoder.ReadKeyEvent()
	require.NoError(t, err)
	require.Equal(t, content, event.Paste)
}

// TestBracketedPasteWithSpecialChars tests paste with special characters
func TestBracketedPasteWithSpecialChars(t *testing.T) {
	content := "echo 'dangerous command'\nrm -rf /\n"
	input := "\033[200~" + content + "\033[201~"
	reader := strings.NewReader(input)
	decoder := NewKeyDecoder(reader)

	event, err := decoder.ReadKeyEvent()
	require.NoError(t, err)
	require.Equal(t, content, event.Paste)

	// Verify newlines are preserved but treated as paste, not execution
	require.Contains(t, event.Paste, "\n")
}

// TestBracketedPasteWithTabs tests paste with tab characters
func TestBracketedPasteWithTabs(t *testing.T) {
	content := "func main() {\n\tfmt.Println(\"hello\")\n}"
	input := "\033[200~" + content + "\033[201~"
	reader := strings.NewReader(input)
	decoder := NewKeyDecoder(reader)

	event, err := decoder.ReadKeyEvent()
	require.NoError(t, err)
	require.Equal(t, content, event.Paste)
	require.Contains(t, event.Paste, "\t")
}

// TestBracketedPasteEmpty tests empty paste
func TestBracketedPasteEmpty(t *testing.T) {
	input := "\033[200~\033[201~"
	reader := strings.NewReader(input)
	decoder := NewKeyDecoder(reader)

	event, err := decoder.ReadKeyEvent()
	require.NoError(t, err)
	require.Equal(t, "", event.Paste)
}

// TestBracketedPasteWithEscapeSequences tests paste containing escape codes
func TestBracketedPasteWithEscapeSequences(t *testing.T) {
	// Paste content that contains ANSI codes (should be preserved as-is)
	content := "\033[31mRed Text\033[0m"
	input := "\033[200~" + content + "\033[201~"
	reader := strings.NewReader(input)
	decoder := NewKeyDecoder(reader)

	event, err := decoder.ReadKeyEvent()
	require.NoError(t, err)
	// The content should be preserved but shouldn't be interpreted during paste
	require.Equal(t, content, event.Paste)
}

// TestBracketedPasteUnicode tests paste with Unicode characters
func TestBracketedPasteUnicode(t *testing.T) {
	content := "Hello 世界! 🎉 Emoji test"
	input := "\033[200~" + content + "\033[201~"
	reader := strings.NewReader(input)
	decoder := NewKeyDecoder(reader)

	event, err := decoder.ReadKeyEvent()
	require.NoError(t, err)
	require.Equal(t, content, event.Paste)
}

// TestBracketedPasteLargeContent tests paste with large content
func TestBracketedPasteLargeContent(t *testing.T) {
	// Create large content (1000 lines)
	var lines []string
	for i := 0; i < 1000; i++ {
		lines = append(lines, "This is line number "+string(rune('0'+i%10)))
	}
	content := strings.Join(lines, "\n")

	input := "\033[200~" + content + "\033[201~"
	reader := strings.NewReader(input)
	decoder := NewKeyDecoder(reader)

	event, err := decoder.ReadKeyEvent()
	require.NoError(t, err)
	require.Equal(t, content, event.Paste)
	require.Equal(t, 1000, len(strings.Split(event.Paste, "\n")))
}

// TestBracketedPasteIncomplete tests incomplete paste sequence (EOF before end)
func TestBracketedPasteIncomplete(t *testing.T) {
	// Start paste but don't finish it
	input := "\033[200~Some content without end"
	reader := strings.NewReader(input)
	decoder := NewKeyDecoder(reader)

	event, err := decoder.ReadKeyEvent()
	// Should return what we have so far
	require.Error(t, err) // EOF
	require.Equal(t, "Some content without end", event.Paste)
}

// TestNormalInputAfterPaste tests that normal input works after paste
func TestNormalInputAfterPaste(t *testing.T) {
	// Paste followed by normal key press
	input := "\033[200~Pasted\033[201~a"
	reader := strings.NewReader(input)
	decoder := NewKeyDecoder(reader)

	// First event: paste
	event1, err1 := decoder.ReadKeyEvent()
	require.NoError(t, err1)
	require.Equal(t, "Pasted", event1.Paste)

	// Second event: normal 'a' key
	event2, err2 := decoder.ReadKeyEvent()
	require.NoError(t, err2)
	require.Equal(t, 'a', event2.Rune)
	require.Equal(t, "", event2.Paste)
}

// TestMixedTypingAndPaste tests that regular typing and paste work together
func TestMixedTypingAndPaste(t *testing.T) {
	// Test simple case: type 'a', paste "XYZ", type 'b'
	pasteInput := "a\033[200~XYZ\033[201~b"
	inputReader := strings.NewReader(pasteInput)
	decoder := NewKeyDecoder(inputReader)

	// First event: 'a'
	event1, err := decoder.ReadKeyEvent()
	require.NoError(t, err)
	require.Equal(t, 'a', event1.Rune)
	require.Equal(t, "", event1.Paste)

	// Second event: paste("XYZ")
	event2, err := decoder.ReadKeyEvent()
	require.NoError(t, err)
	require.Equal(t, "XYZ", event2.Paste)
	require.Equal(t, rune(0), event2.Rune)

	// Third event: 'b'
	event3, err := decoder.ReadKeyEvent()
	require.NoError(t, err)
	require.Equal(t, 'b', event3.Rune)
	require.Equal(t, "", event3.Paste)
}

// TestBracketedPasteEscapeCodeSafety tests that escape codes don't execute during paste
func TestBracketedPasteEscapeCodeSafety(t *testing.T) {
	// Malicious paste with cursor movement codes
	content := "Hello\033[H\033[2JGoodbye" // Home + Clear screen + "Goodbye"
	input := "\033[200~" + content + "\033[201~"
	reader := strings.NewReader(input)
	decoder := NewKeyDecoder(reader)

	event, err := decoder.ReadKeyEvent()
	require.NoError(t, err)
	// Content should be preserved as-is, not interpreted
	require.Equal(t, content, event.Paste)
	// The escape codes should be in the string, not executed
	require.Contains(t, event.Paste, "\033[H")
	require.Contains(t, event.Paste, "\033[2J")
}

// TestBracketedPasteWithFalseEndSequence tests paste containing partial end sequences
func TestBracketedPasteWithFalseEndSequence(t *testing.T) {
	// Content contains sequences that look like end but aren't
	content := "File ESC[201 is not ESC[20"
	input := "\033[200~" + content + "\033[201~"
	reader := strings.NewReader(input)
	decoder := NewKeyDecoder(reader)

	event, err := decoder.ReadKeyEvent()
	require.NoError(t, err)
	require.Equal(t, content, event.Paste)
}

// TestTerminalBracketedPasteMethods tests terminal enable/disable methods
func TestTerminalBracketedPasteMethods(t *testing.T) {
	var buf bytes.Buffer
	terminal := NewTestTerminal(80, 24, &buf)

	// Enable bracketed paste
	terminal.EnableBracketedPaste()
	output := buf.String()
	require.Contains(t, output, "\033[?2004h")

	// Clear buffer
	buf.Reset()

	// Disable bracketed paste
	terminal.DisableBracketedPaste()
	output = buf.String()
	require.Contains(t, output, "\033[?2004l")
}

// Benchmark bracketed paste parsing
func BenchmarkBracketedPasteSmall(b *testing.B) {
	input := "\033[200~Hello, World!\033[201~"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := strings.NewReader(input)
		decoder := NewKeyDecoder(reader)
		decoder.ReadKeyEvent()
	}
}

func BenchmarkBracketedPasteLarge(b *testing.B) {
	// 10KB of content
	content := strings.Repeat("Lorem ipsum dolor sit amet\n", 400)
	input := "\033[200~" + content + "\033[201~"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := strings.NewReader(input)
		decoder := NewKeyDecoder(reader)
		decoder.ReadKeyEvent()
	}
}
