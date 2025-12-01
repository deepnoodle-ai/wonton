package terminal

import (
	"strings"
	"testing"

	"github.com/deepnoodle-ai/gooey/require"
)

func TestNewHyperlink(t *testing.T) {
	link := NewHyperlink("https://example.com", "Example")

	require.Equal(t, "https://example.com", link.URL)
	require.Equal(t, "Example", link.Text)
	require.NotNil(t, link.Style)
	// Default style should be blue and underlined
	require.True(t, link.Style.Underline)
	require.Equal(t, ColorBlue, link.Style.Foreground)
}

func TestHyperlink_BasicWithStyle(t *testing.T) {
	link := NewHyperlink("https://example.com", "Example")
	customStyle := NewStyle().WithForeground(ColorRed).WithBold()

	link = link.WithStyle(customStyle)

	require.Equal(t, ColorRed, link.Style.Foreground)
	require.True(t, link.Style.Bold)
	require.False(t, link.Style.Underline) // Should not have underline anymore
}

func TestHyperlink_Validate(t *testing.T) {
	tests := []struct {
		name    string
		link    Hyperlink
		wantErr bool
	}{
		{
			name:    "valid hyperlink",
			link:    NewHyperlink("https://example.com", "Example"),
			wantErr: false,
		},
		{
			name:    "empty URL",
			link:    Hyperlink{URL: "", Text: "Example"},
			wantErr: true,
		},
		{
			name:    "empty text",
			link:    Hyperlink{URL: "https://example.com", Text: ""},
			wantErr: true,
		},
		{
			name:    "invalid URL",
			link:    Hyperlink{URL: "not a valid url", Text: "Example"},
			wantErr: false, // url.Parse is lenient and accepts this
		},
		{
			name:    "relative URL",
			link:    NewHyperlink("/path/to/page", "Page"),
			wantErr: false,
		},
		{
			name:    "URL with fragment",
			link:    NewHyperlink("https://example.com#section", "Section"),
			wantErr: false,
		},
		{
			name:    "URL with query params",
			link:    NewHyperlink("https://example.com?foo=bar&baz=qux", "Query"),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.link.Validate()
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestOSC8Start(t *testing.T) {
	url := "https://example.com"
	result := OSC8Start(url)

	// Should contain OSC 8 start sequence
	require.Contains(t, result, "\033]8;;")
	require.Contains(t, result, url)
	require.Contains(t, result, "\033\\")

	// Should match expected format exactly
	expected := "\033]8;;https://example.com\033\\"
	require.Equal(t, expected, result)
}

func TestOSC8End(t *testing.T) {
	result := OSC8End()

	// Should be the OSC 8 end sequence
	expected := "\033]8;;\033\\"
	require.Equal(t, expected, result)
}

func TestHyperlink_Format(t *testing.T) {
	tests := []struct {
		name string
		link Hyperlink
		want string
	}{
		{
			name: "simple hyperlink",
			link: Hyperlink{
				URL:   "https://example.com",
				Text:  "Example",
				Style: NewStyle(),
			},
			want: "\033]8;;https://example.com\033\\Example\033]8;;\033\\",
		},
		{
			name: "hyperlink with style",
			link: Hyperlink{
				URL:   "https://example.com",
				Text:  "Example",
				Style: NewStyle().WithForeground(ColorRed),
			},
			// Should contain OSC 8 sequences and ANSI color codes
		},
		{
			name: "hyperlink with URL encoding",
			link: Hyperlink{
				URL:   "https://example.com/path?foo=bar&baz=qux",
				Text:  "Complex URL",
				Style: NewStyle(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.link.Format()

			// Should contain OSC 8 start
			require.Contains(t, result, "\033]8;;"+tt.link.URL+"\033\\")
			// Should contain the text
			require.Contains(t, result, tt.link.Text)
			// Should contain OSC 8 end
			require.Contains(t, result, "\033]8;;\033\\")

			// If we have a specific expected value, check it
			if tt.want != "" && tt.link.Style.IsEmpty() {
				require.Equal(t, tt.want, result)
			}
		})
	}
}

func TestHyperlink_FormatFallback(t *testing.T) {
	tests := []struct {
		name string
		link Hyperlink
	}{
		{
			name: "simple hyperlink",
			link: NewHyperlink("https://example.com", "Example"),
		},
		{
			name: "long URL",
			link: NewHyperlink("https://example.com/very/long/path/to/resource?with=many&query=params", "Resource"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.link.FormatFallback()

			// Should contain the text
			require.Contains(t, result, tt.link.Text)
			// Should contain the URL in parentheses
			require.Contains(t, result, "("+tt.link.URL+")")
			// Should not contain OSC 8 sequences
			require.NotContains(t, result, "\033]8;;")
		})
	}
}

func TestHyperlink_FormatWithOption(t *testing.T) {
	link := NewHyperlink("https://example.com", "Example")

	// Test with OSC 8 enabled
	osc8Result := link.FormatWithOption(true)
	require.Contains(t, osc8Result, "\033]8;;")

	// Test with OSC 8 disabled (fallback)
	fallbackResult := link.FormatWithOption(false)
	require.NotContains(t, fallbackResult, "\033]8;;")
	require.Contains(t, fallbackResult, "(https://example.com)")
}

func TestTerminalRenderFrame_PrintHyperlink(t *testing.T) {
	term := NewTestTerminal(80, 24, &strings.Builder{})

	frame, err := term.BeginFrame()
	require.NoError(t, err)

	link := NewHyperlink("https://example.com", "Example")
	err = frame.PrintHyperlink(0, 0, link)
	require.NoError(t, err)

	err = term.EndFrame(frame)
	require.NoError(t, err)
}

func TestTerminalRenderFrame_PrintHyperlink_InvalidLink(t *testing.T) {
	term := NewTestTerminal(80, 24, &strings.Builder{})

	frame, err := term.BeginFrame()
	require.NoError(t, err)

	// Link with empty URL should fall back to printing just text
	link := Hyperlink{
		URL:   "",
		Text:  "Example",
		Style: NewStyle(),
	}
	err = frame.PrintHyperlink(0, 0, link)
	require.NoError(t, err) // Should not error, just fall back

	err = term.EndFrame(frame)
	require.NoError(t, err)
}

func TestTerminalRenderFrame_PrintHyperlinkFallback(t *testing.T) {
	term := NewTestTerminal(80, 24, &strings.Builder{})

	frame, err := term.BeginFrame()
	require.NoError(t, err)

	link := NewHyperlink("https://example.com", "Example")
	err = frame.PrintHyperlinkFallback(0, 0, link)
	require.NoError(t, err)

	err = term.EndFrame(frame)
	require.NoError(t, err)
}

func TestHyperlink_StylePreservation(t *testing.T) {
	link := NewHyperlink("https://example.com", "Example")
	customStyle := NewStyle().WithForeground(ColorMagenta).WithBold().WithItalic()
	link = link.WithStyle(customStyle)

	formatted := link.Format()

	// Should contain ANSI codes for magenta, bold, and italic
	require.Contains(t, formatted, "35") // Magenta foreground
	require.Contains(t, formatted, "1")  // Bold
	require.Contains(t, formatted, "3")  // Italic

	// Should contain reset code
	require.Contains(t, formatted, "\033[0m")
}

func TestHyperlink_MultipleLinks(t *testing.T) {
	// Test that multiple links can be created without interference
	link1 := NewHyperlink("https://example.com", "Example")
	link2 := NewHyperlink("https://github.com", "GitHub")
	link3 := NewHyperlink("https://google.com", "Google")

	f1 := link1.Format()
	f2 := link2.Format()
	f3 := link3.Format()

	// Each should contain its own URL
	require.Contains(t, f1, "example.com")
	require.Contains(t, f2, "github.com")
	require.Contains(t, f3, "google.com")

	// Each should contain its own text
	require.Contains(t, f1, "Example")
	require.Contains(t, f2, "GitHub")
	require.Contains(t, f3, "Google")
}

func TestHyperlink_SpecialCharacters(t *testing.T) {
	tests := []struct {
		name string
		url  string
		text string
	}{
		{
			name: "URL with special characters",
			url:  "https://example.com/path?foo=bar&baz=qux#section",
			text: "Complex",
		},
		{
			name: "text with unicode",
			url:  "https://example.com",
			text: "例え 🔗 Link",
		},
		{
			name: "URL with unicode",
			url:  "https://例え.com/パス",
			text: "Unicode URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			link := NewHyperlink(tt.url, tt.text)

			// Validate should pass
			err := link.Validate()
			require.NoError(t, err)

			// Format should work
			formatted := link.Format()
			require.Contains(t, formatted, tt.text)

			// Fallback should work
			fallback := link.FormatFallback()
			require.Contains(t, fallback, tt.text)
			require.Contains(t, fallback, tt.url)
		})
	}
}

func TestOSC8_EscapeSequenceFormat(t *testing.T) {
	// Verify the exact format of OSC 8 sequences
	url := "https://example.com"
	start := OSC8Start(url)
	end := OSC8End()

	// Start sequence: ESC ] 8 ; ; URL ESC \
	require.True(t, strings.HasPrefix(start, "\033]8;;"))
	require.True(t, strings.HasSuffix(start, "\033\\"))

	// End sequence: ESC ] 8 ; ; ESC \
	require.Equal(t, "\033]8;;\033\\", end)

	// Combined should be valid
	combined := start + "text" + end
	require.Contains(t, combined, url)
	require.Contains(t, combined, "text")
}
