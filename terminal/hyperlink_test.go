package terminal

import (
	"strings"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestNewHyperlink(t *testing.T) {
	link := NewHyperlink("https://example.com", "Example")

	assert.Equal(t, "https://example.com", link.URL)
	assert.Equal(t, "Example", link.Text)
	assert.NotNil(t, link.Style)
	// Default style should be blue and underlined
	assert.True(t, link.Style.Underline)
	assert.Equal(t, ColorBlue, link.Style.Foreground)
}

func TestHyperlink_BasicWithStyle(t *testing.T) {
	link := NewHyperlink("https://example.com", "Example")
	customStyle := NewStyle().WithForeground(ColorRed).WithBold()

	link = link.WithStyle(customStyle)

	assert.Equal(t, ColorRed, link.Style.Foreground)
	assert.True(t, link.Style.Bold)
	assert.False(t, link.Style.Underline) // Should not have underline anymore
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
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestOSC8Start(t *testing.T) {
	url := "https://example.com"
	result := OSC8Start(url)

	// Should contain OSC 8 start sequence
	assert.Contains(t, result, "\033]8;;")
	assert.Contains(t, result, url)
	assert.Contains(t, result, "\033\\")

	// Should match expected format exactly
	expected := "\033]8;;https://example.com\033\\"
	assert.Equal(t, expected, result)
}

func TestOSC8End(t *testing.T) {
	result := OSC8End()

	// Should be the OSC 8 end sequence
	expected := "\033]8;;\033\\"
	assert.Equal(t, expected, result)
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
			assert.Contains(t, result, "\033]8;;"+tt.link.URL+"\033\\")
			// Should contain the text
			assert.Contains(t, result, tt.link.Text)
			// Should contain OSC 8 end
			assert.Contains(t, result, "\033]8;;\033\\")

			// If we have a specific expected value, check it
			if tt.want != "" && tt.link.Style.IsEmpty() {
				assert.Equal(t, tt.want, result)
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
			assert.Contains(t, result, tt.link.Text)
			// Should contain the URL in parentheses
			assert.Contains(t, result, "("+tt.link.URL+")")
			// Should not contain OSC 8 sequences
			assert.NotContains(t, result, "\033]8;;")
		})
	}
}

func TestHyperlink_FormatWithOption(t *testing.T) {
	link := NewHyperlink("https://example.com", "Example")

	// Test with OSC 8 enabled
	osc8Result := link.FormatWithOption(true)
	assert.Contains(t, osc8Result, "\033]8;;")

	// Test with OSC 8 disabled (fallback)
	fallbackResult := link.FormatWithOption(false)
	assert.NotContains(t, fallbackResult, "\033]8;;")
	assert.Contains(t, fallbackResult, "(https://example.com)")
}

func TestTerminalRenderFrame_PrintHyperlink(t *testing.T) {
	term := NewTestTerminal(80, 24, &strings.Builder{})

	frame, err := term.BeginFrame()
	assert.NoError(t, err)

	link := NewHyperlink("https://example.com", "Example")
	err = frame.PrintHyperlink(0, 0, link)
	assert.NoError(t, err)

	err = term.EndFrame(frame)
	assert.NoError(t, err)
}

func TestTerminalRenderFrame_PrintHyperlink_InvalidLink(t *testing.T) {
	term := NewTestTerminal(80, 24, &strings.Builder{})

	frame, err := term.BeginFrame()
	assert.NoError(t, err)

	// Link with empty URL should fall back to printing just text
	link := Hyperlink{
		URL:   "",
		Text:  "Example",
		Style: NewStyle(),
	}
	err = frame.PrintHyperlink(0, 0, link)
	assert.NoError(t, err) // Should not error, just fall back

	err = term.EndFrame(frame)
	assert.NoError(t, err)
}

func TestTerminalRenderFrame_PrintHyperlinkFallback(t *testing.T) {
	term := NewTestTerminal(80, 24, &strings.Builder{})

	frame, err := term.BeginFrame()
	assert.NoError(t, err)

	link := NewHyperlink("https://example.com", "Example")
	err = frame.PrintHyperlinkFallback(0, 0, link)
	assert.NoError(t, err)

	err = term.EndFrame(frame)
	assert.NoError(t, err)
}

func TestHyperlink_StylePreservation(t *testing.T) {
	link := NewHyperlink("https://example.com", "Example")
	customStyle := NewStyle().WithForeground(ColorMagenta).WithBold().WithItalic()
	link = link.WithStyle(customStyle)

	formatted := link.Format()

	// Should contain ANSI codes for magenta, bold, and italic
	assert.Contains(t, formatted, "35") // Magenta foreground
	assert.Contains(t, formatted, "1")  // Bold
	assert.Contains(t, formatted, "3")  // Italic

	// Should contain reset code
	assert.Contains(t, formatted, "\033[0m")
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
	assert.Contains(t, f1, "example.com")
	assert.Contains(t, f2, "github.com")
	assert.Contains(t, f3, "google.com")

	// Each should contain its own text
	assert.Contains(t, f1, "Example")
	assert.Contains(t, f2, "GitHub")
	assert.Contains(t, f3, "Google")
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
			assert.NoError(t, err)

			// Format should work
			formatted := link.Format()
			assert.Contains(t, formatted, tt.text)

			// Fallback should work
			fallback := link.FormatFallback()
			assert.Contains(t, fallback, tt.text)
			assert.Contains(t, fallback, tt.url)
		})
	}
}

func TestOSC8_EscapeSequenceFormat(t *testing.T) {
	// Verify the exact format of OSC 8 sequences
	url := "https://example.com"
	start := OSC8Start(url)
	end := OSC8End()

	// Start sequence: ESC ] 8 ; ; URL ESC \
	assert.True(t, strings.HasPrefix(start, "\033]8;;"))
	assert.True(t, strings.HasSuffix(start, "\033\\"))

	// End sequence: ESC ] 8 ; ; ESC \
	assert.Equal(t, "\033]8;;\033\\", end)

	// Combined should be valid
	combined := start + "text" + end
	assert.Contains(t, combined, url)
	assert.Contains(t, combined, "text")
}
