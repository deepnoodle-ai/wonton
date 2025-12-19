package web

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestIsMediaURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{
			name:     "image file",
			url:      "https://example.com/image.jpg",
			expected: true,
		},
		{
			name:     "video file",
			url:      "https://example.com/video.mp4",
			expected: true,
		},
		{
			name:     "audio file",
			url:      "https://example.com/audio.mp3",
			expected: true,
		},
		{
			name:     "document file",
			url:      "https://example.com/doc.pdf",
			expected: true,
		},
		{
			name:     "uppercase extension",
			url:      "https://example.com/IMAGE.JPG",
			expected: true,
		},
		{
			name:     "html file",
			url:      "https://example.com/page.html",
			expected: false,
		},
		{
			name:     "no extension",
			url:      "https://example.com/page",
			expected: false,
		},
		{
			name:     "path with dot but no extension",
			url:      "https://example.com/path.with.dots/page",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, _ := url.Parse(tt.url)
			result := IsMediaURL(u)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsMediaURLNil(t *testing.T) {
	// Test that nil URL returns false without panicking
	assert.False(t, IsMediaURL(nil))
}

func TestIsMediaExtension(t *testing.T) {
	tests := []struct {
		name     string
		ext      string
		expected bool
	}{
		{
			name:     "lowercase jpg",
			ext:      ".jpg",
			expected: true,
		},
		{
			name:     "uppercase JPG",
			ext:      ".JPG",
			expected: true,
		},
		{
			name:     "mixed case Mp4",
			ext:      ".Mp4",
			expected: true,
		},
		{
			name:     "html not media",
			ext:      ".html",
			expected: false,
		},
		{
			name:     "without leading dot",
			ext:      "jpg",
			expected: false,
		},
		{
			name:     "empty string",
			ext:      "",
			expected: false,
		},
		{
			name:     "pdf",
			ext:      ".pdf",
			expected: true,
		},
		{
			name:     "zip",
			ext:      ".zip",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsMediaExtension(tt.ext)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Example demonstrates detecting media files from URLs.
func ExampleIsMediaURL() {
	imageURL, _ := url.Parse("https://example.com/photo.jpg")
	fmt.Println(IsMediaURL(imageURL))

	videoURL, _ := url.Parse("https://example.com/video.mp4")
	fmt.Println(IsMediaURL(videoURL))

	pageURL, _ := url.Parse("https://example.com/page.html")
	fmt.Println(IsMediaURL(pageURL))

	// Output:
	// true
	// true
	// false
}
