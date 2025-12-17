package terminal

import (
	"strings"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestStart(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
	}{
		{
			name: "simple https URL",
			url:  "https://example.com",
			want: "\033]8;;https://example.com\033\\",
		},
		{
			name: "URL with path",
			url:  "https://example.com/path/to/page",
			want: "\033]8;;https://example.com/path/to/page\033\\",
		},
		{
			name: "URL with query params",
			url:  "https://example.com?foo=bar&baz=qux",
			want: "\033]8;;https://example.com?foo=bar&baz=qux\033\\",
		},
		{
			name: "URL with fragment",
			url:  "https://example.com#section",
			want: "\033]8;;https://example.com#section\033\\",
		},
		{
			name: "file URL",
			url:  "file:///home/user/doc.txt",
			want: "\033]8;;file:///home/user/doc.txt\033\\",
		},
		{
			name: "empty URL",
			url:  "",
			want: "\033]8;;\033\\",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Start(tt.url)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestStartWithID(t *testing.T) {
	tests := []struct {
		name string
		url  string
		id   string
		want string
	}{
		{
			name: "simple URL with ID",
			url:  "https://example.com",
			id:   "link1",
			want: "\033]8;id=link1;https://example.com\033\\",
		},
		{
			name: "URL with numeric ID",
			url:  "https://example.com",
			id:   "12345",
			want: "\033]8;id=12345;https://example.com\033\\",
		},
		{
			name: "URL with empty ID",
			url:  "https://example.com",
			id:   "",
			want: "\033]8;id=;https://example.com\033\\",
		},
		{
			name: "URL with UUID-style ID",
			url:  "https://example.com",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			want: "\033]8;id=550e8400-e29b-41d4-a716-446655440000;https://example.com\033\\",
		},
		{
			name: "URL with underscore ID",
			url:  "https://example.com",
			id:   "my_link_id",
			want: "\033]8;id=my_link_id;https://example.com\033\\",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StartWithID(tt.url, tt.id)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestEnd(t *testing.T) {
	result := End()
	expected := "\033]8;;\033\\"
	assert.Equal(t, expected, result)
}

func TestFormat(t *testing.T) {
	tests := []struct {
		name string
		url  string
		text string
		want string
	}{
		{
			name: "simple link",
			url:  "https://example.com",
			text: "Click here",
			want: "\033]8;;https://example.com\033\\Click here\033]8;;\033\\",
		},
		{
			name: "link with empty text",
			url:  "https://example.com",
			text: "",
			want: "\033]8;;https://example.com\033\\\033]8;;\033\\",
		},
		{
			name: "link with unicode text",
			url:  "https://example.com",
			text: "链接 🔗",
			want: "\033]8;;https://example.com\033\\链接 🔗\033]8;;\033\\",
		},
		{
			name: "link with special characters",
			url:  "https://example.com?a=1&b=2",
			text: "Link <with> special & chars",
			want: "\033]8;;https://example.com?a=1&b=2\033\\Link <with> special & chars\033]8;;\033\\",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Format(tt.url, tt.text)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestFormatWithID(t *testing.T) {
	tests := []struct {
		name string
		url  string
		text string
		id   string
		want string
	}{
		{
			name: "link with ID",
			url:  "https://example.com",
			text: "Click here",
			id:   "mylink",
			want: "\033]8;id=mylink;https://example.com\033\\Click here\033]8;;\033\\",
		},
		{
			name: "link with empty ID",
			url:  "https://example.com",
			text: "Click here",
			id:   "",
			want: "\033]8;id=;https://example.com\033\\Click here\033]8;;\033\\",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatWithID(tt.url, tt.text, tt.id)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestValidateURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{
			name:    "valid https URL",
			url:     "https://example.com",
			wantErr: false,
		},
		{
			name:    "valid http URL",
			url:     "http://example.com",
			wantErr: false,
		},
		{
			name:    "valid file URL",
			url:     "file:///home/user/doc.txt",
			wantErr: false,
		},
		{
			name:    "relative URL",
			url:     "/path/to/page",
			wantErr: false,
		},
		{
			name:    "URL without scheme",
			url:     "example.com/path",
			wantErr: false, // url.Parse is lenient
		},
		{
			name:    "empty URL",
			url:     "",
			wantErr: true,
		},
		{
			name:    "URL with fragment",
			url:     "https://example.com#section",
			wantErr: false,
		},
		{
			name:    "URL with query params",
			url:     "https://example.com?foo=bar",
			wantErr: false,
		},
		{
			name:    "localhost URL",
			url:     "http://localhost:8080",
			wantErr: false,
		},
		{
			name:    "IP address URL",
			url:     "http://192.168.1.1:8080/path",
			wantErr: false,
		},
		{
			name:    "unicode URL",
			url:     "https://例え.com/パス",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateURL(tt.url)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateAbsoluteURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{
			name:    "valid https URL",
			url:     "https://example.com",
			wantErr: false,
		},
		{
			name:    "valid http URL",
			url:     "http://example.com",
			wantErr: false,
		},
		{
			name:    "valid file URL",
			url:     "file:///home/user/doc.txt",
			wantErr: false,
		},
		{
			name:    "relative URL - no scheme",
			url:     "/path/to/page",
			wantErr: true, // Must have scheme
		},
		{
			name:    "URL without scheme",
			url:     "example.com/path",
			wantErr: true, // Must have scheme
		},
		{
			name:    "empty URL",
			url:     "",
			wantErr: true,
		},
		{
			name:    "mailto URL",
			url:     "mailto:user@example.com",
			wantErr: false,
		},
		{
			name:    "data URL",
			url:     "data:text/plain;base64,SGVsbG8=",
			wantErr: false,
		},
		{
			name:    "custom scheme URL",
			url:     "myapp://open/document",
			wantErr: false,
		},
		{
			name:    "ftp URL",
			url:     "ftp://ftp.example.com/file.txt",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAbsoluteURL(tt.url)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.url == "" {
					assert.Contains(t, err.Error(), "cannot be empty")
				} else if !strings.Contains(tt.url, "://") {
					assert.Contains(t, err.Error(), "must have a scheme")
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFallback(t *testing.T) {
	tests := []struct {
		name string
		url  string
		text string
		want string
	}{
		{
			name: "simple fallback",
			url:  "https://example.com",
			text: "Example",
			want: "Example (https://example.com)",
		},
		{
			name: "long URL",
			url:  "https://example.com/very/long/path?with=params",
			text: "Link",
			want: "Link (https://example.com/very/long/path?with=params)",
		},
		{
			name: "empty text",
			url:  "https://example.com",
			text: "",
			want: " (https://example.com)",
		},
		{
			name: "empty URL",
			url:  "",
			text: "Text",
			want: "Text ()",
		},
		{
			name: "unicode content",
			url:  "https://example.com",
			text: "链接",
			want: "链接 (https://example.com)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Fallback(tt.url, tt.text)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestStripOSC8(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		{
			name: "no OSC8 sequences",
			text: "Plain text",
			want: "Plain text",
		},
		{
			name: "simple OSC8 link",
			text: "\033]8;;https://example.com\033\\Click here\033]8;;\033\\",
			want: "Click here",
		},
		{
			name: "OSC8 link with ID",
			text: "\033]8;id=link1;https://example.com\033\\Click here\033]8;;\033\\",
			want: "Click here",
		},
		{
			name: "multiple OSC8 links",
			text: "\033]8;;https://a.com\033\\Link A\033]8;;\033\\ and \033]8;;https://b.com\033\\Link B\033]8;;\033\\",
			want: "Link A and Link B",
		},
		{
			name: "nested text with OSC8",
			text: "Before \033]8;;https://example.com\033\\Link\033]8;;\033\\ After",
			want: "Before Link After",
		},
		{
			name: "empty OSC8 URL",
			text: "\033]8;;\033\\Text\033]8;;\033\\",
			want: "Text",
		},
		{
			name: "incomplete OSC8 start - missing ST",
			text: "\033]8;;https://example.com",
			want: "\033]8;;https://example.com", // Should remain unchanged
		},
		{
			name: "unicode content in link",
			text: "\033]8;;https://example.com\033\\链接 🔗\033]8;;\033\\",
			want: "链接 🔗",
		},
		{
			name: "OSC8 with complex URL",
			text: "\033]8;;https://example.com/path?foo=bar&baz=qux#section\033\\Link\033]8;;\033\\",
			want: "Link",
		},
		{
			name: "empty string",
			text: "",
			want: "",
		},
		{
			name: "only spaces",
			text: "   ",
			want: "   ",
		},
		{
			name: "OSC8 with ANSI colors mixed",
			text: "\033[31m\033]8;;https://example.com\033\\Red Link\033]8;;\033\\\033[0m",
			want: "\033[31mRed Link\033[0m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StripOSC8(tt.text)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestStripOSC8_Performance(t *testing.T) {
	// Test with a very long string containing many OSC8 sequences
	var builder strings.Builder
	for i := 0; i < 1000; i++ {
		builder.WriteString("\033]8;;https://example.com/")
		builder.WriteString(strings.Repeat("a", 50))
		builder.WriteString("\033\\Link ")
		builder.WriteString(string(rune('A' + i%26)))
		builder.WriteString("\033]8;;\033\\ ")
	}

	input := builder.String()
	result := StripOSC8(input)

	// Should not contain any OSC8 sequences
	assert.NotContains(t, result, "\033]8;")
	// Should contain the link text
	assert.Contains(t, result, "Link A")
}

func TestFormatRoundTrip(t *testing.T) {
	// Test that Format creates something StripOSC8 can extract text from
	tests := []struct {
		url  string
		text string
	}{
		{"https://example.com", "Simple"},
		{"https://example.com/path?q=1", "Complex URL"},
		{"file:///tmp/test.txt", "Local file"},
		{"https://example.com", "Text with spaces and 数字"},
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			formatted := Format(tt.url, tt.text)
			stripped := StripOSC8(formatted)
			assert.Equal(t, tt.text, stripped)
		})
	}
}

func TestFormatWithIDRoundTrip(t *testing.T) {
	// Test that FormatWithID creates something StripOSC8 can extract text from
	url := "https://example.com"
	text := "Test Link"
	id := "testid123"

	formatted := FormatWithID(url, text, id)
	stripped := StripOSC8(formatted)
	assert.Equal(t, text, stripped)
}
