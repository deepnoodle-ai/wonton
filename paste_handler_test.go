package gooey

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestPasteHandlerAccept tests that PasteAccept allows paste through
func TestPasteHandlerAccept(t *testing.T) {
	terminal := NewTestTerminal(80, 24, &bytes.Buffer{})
	terminal.EnableRawMode()

	// Simulate bracketed paste input followed by Enter
	pasteContent := "Hello, World!"
	input := "\033[200~" + pasteContent + "\033[201~\r"
	reader := strings.NewReader(input)

	inputHandler := NewInput(terminal)
	inputHandler.SetReader(reader)

	// Handler that accepts all pastes
	inputHandler.WithPasteHandler(func(info PasteInfo) (PasteHandlerDecision, string) {
		require.Equal(t, pasteContent, info.Content)
		require.Equal(t, 1, info.LineCount)
		require.Equal(t, len(pasteContent), info.ByteCount)
		return PasteAccept, ""
	})

	result, err := inputHandler.Read()
	require.NoError(t, err)
	require.Equal(t, pasteContent, result)
}

// TestPasteHandlerReject tests that PasteReject blocks paste
func TestPasteHandlerReject(t *testing.T) {
	terminal := NewTestTerminal(80, 24, &bytes.Buffer{})
	terminal.EnableRawMode()

	// Simpler test: Just verify paste is rejected and nothing is added to buffer
	pasteContent := "REJECTED_CONTENT"
	input := "\033[200~" + pasteContent + "\033[201~\n"
	reader := strings.NewReader(input)

	inputHandler := NewInput(terminal)
	inputHandler.SetReader(reader)

	rejectCalled := false
	// Handler that rejects all pastes
	inputHandler.WithPasteHandler(func(info PasteInfo) (PasteHandlerDecision, string) {
		rejectCalled = true
		require.Equal(t, pasteContent, info.Content)
		return PasteReject, ""
	})

	result, err := inputHandler.Read()
	require.NoError(t, err)
	// Handler should have been called
	require.True(t, rejectCalled)
	// Result should be empty since paste was rejected and nothing else was typed
	require.Equal(t, "", result)
}

// TestPasteHandlerModified tests that PasteModified replaces content
func TestPasteHandlerModified(t *testing.T) {
	terminal := NewTestTerminal(80, 24, &bytes.Buffer{})
	terminal.EnableRawMode()

	// Simulate bracketed paste input followed by Enter
	pasteContent := "hello world"
	modifiedContent := "HELLO WORLD"
	input := "\033[200~" + pasteContent + "\033[201~\r"
	reader := strings.NewReader(input)

	inputHandler := NewInput(terminal)
	inputHandler.SetReader(reader)

	// Handler that uppercases all pastes
	inputHandler.WithPasteHandler(func(info PasteInfo) (PasteHandlerDecision, string) {
		return PasteModified, strings.ToUpper(info.Content)
	})

	result, err := inputHandler.Read()
	require.NoError(t, err)
	require.Equal(t, modifiedContent, result)
}

// TestPasteHandlerSizeLimit tests rejecting large pastes
func TestPasteHandlerSizeLimit(t *testing.T) {
	terminal := NewTestTerminal(80, 24, &bytes.Buffer{})
	terminal.EnableRawMode()

	// Create large paste content
	largeContent := strings.Repeat("X", 1500)
	input := "\033[200~" + largeContent + "\033[201~\n"
	reader := strings.NewReader(input)

	inputHandler := NewInput(terminal)
	inputHandler.SetReader(reader)

	rejectCalled := false
	// Handler that rejects pastes over 1000 bytes
	inputHandler.WithPasteHandler(func(info PasteInfo) (PasteHandlerDecision, string) {
		if info.ByteCount > 1000 {
			rejectCalled = true
			return PasteReject, ""
		}
		return PasteAccept, ""
	})

	result, err := inputHandler.Read()
	require.NoError(t, err)
	require.True(t, rejectCalled)
	// Should be empty since large paste was rejected
	require.Equal(t, "", result)
}

// TestPasteHandlerLineCount tests PasteInfo line counting
func TestPasteHandlerLineCount(t *testing.T) {
	terminal := NewTestTerminal(80, 24, &bytes.Buffer{})
	terminal.EnableRawMode()

	// Multi-line paste
	pasteContent := "line1\nline2\nline3"
	input := "\033[200~" + pasteContent + "\033[201~\r"
	reader := strings.NewReader(input)

	inputHandler := NewInput(terminal)
	inputHandler.SetReader(reader)

	var capturedInfo PasteInfo
	inputHandler.WithPasteHandler(func(info PasteInfo) (PasteHandlerDecision, string) {
		capturedInfo = info
		return PasteAccept, ""
	})

	_, err := inputHandler.Read()
	require.NoError(t, err)
	require.Equal(t, 3, capturedInfo.LineCount)
	require.Equal(t, len(pasteContent), capturedInfo.ByteCount)
}

// TestPasteHandlerANSIStripping tests stripping escape codes
func TestPasteHandlerANSIStripping(t *testing.T) {
	terminal := NewTestTerminal(80, 24, &bytes.Buffer{})
	terminal.EnableRawMode()

	// Paste with ANSI codes
	pasteContent := "\033[31mRed\033[0m Text"
	expectedContent := "Red Text"
	input := "\033[200~" + pasteContent + "\033[201~\r"
	reader := strings.NewReader(input)

	inputHandler := NewInput(terminal)
	inputHandler.SetReader(reader)

	// Handler that strips ANSI codes
	inputHandler.WithPasteHandler(func(info PasteInfo) (PasteHandlerDecision, string) {
		// Simple ANSI stripping (matches \x1b[...m)
		cleaned := info.Content
		cleaned = strings.ReplaceAll(cleaned, "\033[31m", "")
		cleaned = strings.ReplaceAll(cleaned, "\033[0m", "")
		if cleaned != info.Content {
			return PasteModified, cleaned
		}
		return PasteAccept, ""
	})

	result, err := inputHandler.Read()
	require.NoError(t, err)
	require.Equal(t, expectedContent, result)
}

// TestPasteDisplayModeNormal tests default paste display
func TestPasteDisplayModeNormal(t *testing.T) {
	terminal := NewTestTerminal(80, 24, &bytes.Buffer{})
	terminal.EnableRawMode()

	pasteContent := "normal paste"
	input := "\033[200~" + pasteContent + "\033[201~\r"
	reader := strings.NewReader(input)

	inputHandler := NewInput(terminal)
	inputHandler.SetReader(reader)
	inputHandler.WithPasteDisplayMode(PasteDisplayNormal)

	result, err := inputHandler.Read()
	require.NoError(t, err)
	require.Equal(t, pasteContent, result)
}

// TestPasteDisplayModePlaceholder tests placeholder display
// Note: This tests that content is accepted even with placeholder display
func TestPasteDisplayModePlaceholder(t *testing.T) {
	var buf bytes.Buffer
	terminal := NewTestTerminal(80, 24, &buf)
	terminal.EnableRawMode()

	pasteContent := "line1\nline2\nline3"
	input := "\033[200~" + pasteContent + "\033[201~\r"
	reader := strings.NewReader(input)

	inputHandler := NewInput(terminal)
	inputHandler.SetReader(reader)
	inputHandler.WithPasteDisplayMode(PasteDisplayPlaceholder)

	result, err := inputHandler.Read()
	require.NoError(t, err)
	// Content should still be in the result
	require.Equal(t, pasteContent, result)

	// Check that placeholder was shown in output
	output := buf.String()
	require.Contains(t, output, "[pasted")
	require.Contains(t, output, "lines]")
}

// TestPasteDisplayModeHidden tests hidden paste display
func TestPasteDisplayModeHidden(t *testing.T) {
	var buf bytes.Buffer
	terminal := NewTestTerminal(80, 24, &buf)
	terminal.EnableRawMode()

	pasteContent := "secret content"
	input := "\033[200~" + pasteContent + "\033[201~\r"
	reader := strings.NewReader(input)

	inputHandler := NewInput(terminal)
	inputHandler.SetReader(reader)
	inputHandler.WithPasteDisplayMode(PasteDisplayHidden)

	result, err := inputHandler.Read()
	require.NoError(t, err)
	require.Equal(t, pasteContent, result)
	// The output should not contain the pasted content
	// (it would only show the prompt, not the paste)
}

// TestPasteHandlerWithMaxLength tests paste respects max length
func TestPasteHandlerWithMaxLength(t *testing.T) {
	terminal := NewTestTerminal(80, 24, &bytes.Buffer{})
	terminal.EnableRawMode()

	pasteContent := "This is a very long paste that exceeds max length"
	input := "\033[200~" + pasteContent + "\033[201~\r"
	reader := strings.NewReader(input)

	inputHandler := NewInput(terminal)
	inputHandler.SetReader(reader)
	inputHandler.WithMaxLength(20)

	result, err := inputHandler.Read()
	require.NoError(t, err)
	// Should be truncated to 20 characters
	require.Equal(t, 20, len(result))
	require.Equal(t, pasteContent[:20], result)
}

// TestMultiplePasteEvents tests handling multiple paste events in sequence
func TestMultiplePasteEvents(t *testing.T) {
	terminal := NewTestTerminal(80, 24, &bytes.Buffer{})
	terminal.EnableRawMode()

	paste1 := "first"
	paste2 := "second"
	input := "\033[200~" + paste1 + "\033[201~\033[200~" + paste2 + "\033[201~\r"
	reader := strings.NewReader(input)

	inputHandler := NewInput(terminal)
	inputHandler.SetReader(reader)

	result, err := inputHandler.Read()
	require.NoError(t, err)
	require.Equal(t, paste1+paste2, result)
}

// TestPasteHandlerNoHandler tests paste without handler (default behavior)
func TestPasteHandlerNoHandler(t *testing.T) {
	terminal := NewTestTerminal(80, 24, &bytes.Buffer{})
	terminal.EnableRawMode()

	pasteContent := "unhandled paste"
	input := "\033[200~" + pasteContent + "\033[201~\r"
	reader := strings.NewReader(input)

	inputHandler := NewInput(terminal)
	inputHandler.SetReader(reader)
	// No paste handler set - should accept by default

	result, err := inputHandler.Read()
	require.NoError(t, err)
	require.Equal(t, pasteContent, result)
}

// BenchmarkPasteHandlerAccept benchmarks paste with accept handler
func BenchmarkPasteHandlerAccept(b *testing.B) {
	content := strings.Repeat("Lorem ipsum dolor sit amet\n", 40)
	input := "\033[200~" + content + "\033[201~\r"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		terminal := NewTestTerminal(80, 24, &bytes.Buffer{})
		terminal.EnableRawMode()

		reader := strings.NewReader(input)
		inputHandler := NewInput(terminal)
		inputHandler.SetReader(reader)
		inputHandler.WithPasteHandler(func(info PasteInfo) (PasteHandlerDecision, string) {
			return PasteAccept, ""
		})

		inputHandler.Read()
	}
}

// TestCtrlCCancelsInput tests that Ctrl+C cancels input like ESC does
func TestCtrlCCancelsInput(t *testing.T) {
	terminal := NewTestTerminal(80, 24, &bytes.Buffer{})
	terminal.EnableRawMode()

	// Just Ctrl+C without other text
	input := "\x03" // \x03 is Ctrl+C
	reader := strings.NewReader(input)

	inputHandler := NewInput(terminal)
	inputHandler.SetReader(reader)

	result, err := inputHandler.Read()
	// Should return error when cancelled with Ctrl+C
	require.Error(t, err)
	require.Contains(t, err.Error(), "cancelled")
	// Result should be empty since input was cancelled
	require.Equal(t, "", result)
}

// BenchmarkPasteHandlerModified benchmarks paste with modification
func BenchmarkPasteHandlerModified(b *testing.B) {
	content := strings.Repeat("Lorem ipsum dolor sit amet\n", 40)
	input := "\033[200~" + content + "\033[201~\r"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		terminal := NewTestTerminal(80, 24, &bytes.Buffer{})
		terminal.EnableRawMode()

		reader := strings.NewReader(input)
		inputHandler := NewInput(terminal)
		inputHandler.SetReader(reader)
		inputHandler.WithPasteHandler(func(info PasteInfo) (PasteHandlerDecision, string) {
			return PasteModified, strings.ToUpper(info.Content)
		})

		inputHandler.Read()
	}
}
