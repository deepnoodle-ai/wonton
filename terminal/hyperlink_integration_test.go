package terminal

import (
	"strings"
	"testing"

	"github.com/deepnoodle-ai/gooey/require"
)

func TestHyperlink_Integration(t *testing.T) {
	// Create a terminal with a string builder to capture output
	var output strings.Builder
	term := NewTestTerminal(80, 24, &output)

	// Create a frame and render a hyperlink
	frame, err := term.BeginFrame()
	require.NoError(t, err)

	link := NewHyperlink("https://example.com", "Example")
	err = frame.PrintHyperlink(0, 0, link)
	require.NoError(t, err)

	err = term.EndFrame(frame)
	require.NoError(t, err)

	// Get the output
	result := output.String()

	// Should contain OSC 8 start sequence with URL
	require.Contains(t, result, "\033]8;;https://example.com\033\\")

	// Should contain the link text
	require.Contains(t, result, "Example")

	// Should contain OSC 8 end sequence
	require.Contains(t, result, "\033]8;;\033\\")

	// The escape codes should be properly formed (with ESC character)
	// Not appearing as literal text without the ESC character
	require.True(t, strings.Contains(result, "\033]8;;"), "Should have OSC 8 escape sequences")

	// Verify the order: OSC8 start, then text, then OSC8 end
	startIdx := strings.Index(result, "\033]8;;https://example.com\033\\")
	textIdx := strings.Index(result, "Example")
	endIdx := strings.Index(result, "\033]8;;\033\\")

	require.True(t, startIdx < textIdx, "OSC 8 start should come before text")
	require.True(t, textIdx < endIdx, "Text should come before OSC 8 end")
}

func TestHyperlink_IntegrationMultiple(t *testing.T) {
	// Create a terminal with a string builder to capture output
	var output strings.Builder
	term := NewTestTerminal(80, 24, &output)

	// Create a frame and render multiple hyperlinks
	frame, err := term.BeginFrame()
	require.NoError(t, err)

	link1 := NewHyperlink("https://example.com", "First")
	link2 := NewHyperlink("https://github.com", "Second")

	err = frame.PrintHyperlink(0, 0, link1)
	require.NoError(t, err)

	err = frame.PrintHyperlink(10, 0, link2)
	require.NoError(t, err)

	err = term.EndFrame(frame)
	require.NoError(t, err)

	// Get the output
	result := output.String()

	// Should contain both URLs
	require.Contains(t, result, "https://example.com")
	require.Contains(t, result, "https://github.com")

	// Should contain both link texts
	require.Contains(t, result, "First")
	require.Contains(t, result, "Second")

	// Count OSC 8 start sequences (should be 2)
	startCount := strings.Count(result, "\033]8;;")
	require.Equal(t, 4, startCount, "Should have 2 starts + 2 ends (each contains \\033]8;;)")

	// Specifically count non-empty URLs (the starts)
	require.Contains(t, result, "\033]8;;https://example.com\033\\")
	require.Contains(t, result, "\033]8;;https://github.com\033\\")
}

func TestHyperlink_IntegrationFallback(t *testing.T) {
	// Create a terminal with a string builder to capture output
	var output strings.Builder
	term := NewTestTerminal(80, 24, &output)

	// Create a frame and render a hyperlink with fallback
	frame, err := term.BeginFrame()
	require.NoError(t, err)

	link := NewHyperlink("https://example.com", "Example")
	err = frame.PrintHyperlinkFallback(0, 0, link)
	require.NoError(t, err)

	err = term.EndFrame(frame)
	require.NoError(t, err)

	// Get the output
	result := output.String()

	// Should contain the text
	require.Contains(t, result, "Example")

	// Should contain the URL in parentheses
	require.Contains(t, result, "(https://example.com)")

	// Should NOT contain OSC 8 sequences
	require.NotContains(t, result, "\033]8;;https://example.com\033\\")
}

func TestHyperlink_IntegrationWithStyle(t *testing.T) {
	// Create a terminal with a string builder to capture output
	var output strings.Builder
	term := NewTestTerminal(80, 24, &output)

	// Create a frame and render a styled hyperlink
	frame, err := term.BeginFrame()
	require.NoError(t, err)

	link := NewHyperlink("https://example.com", "Example")
	customStyle := NewStyle().WithForeground(ColorRed).WithBold()
	link = link.WithStyle(customStyle)

	err = frame.PrintHyperlink(0, 0, link)
	require.NoError(t, err)

	err = term.EndFrame(frame)
	require.NoError(t, err)

	// Get the output
	result := output.String()

	// Should contain OSC 8 sequences
	require.Contains(t, result, "\033]8;;https://example.com\033\\")
	require.Contains(t, result, "Example")

	// Should contain style codes (red = 31, bold = 1)
	require.Contains(t, result, "31") // Red foreground
	require.Contains(t, result, "1")  // Bold
}
