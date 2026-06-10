package web

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestIsBinaryURL(t *testing.T) {
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
			name:     "webm video",
			url:      "https://example.com/video.webm",
			expected: true,
		},
		{
			name:     "avif image",
			url:      "https://example.com/image.avif",
			expected: true,
		},
		{
			name:     "javascript file",
			url:      "https://example.com/app.js",
			expected: true,
		},
		{
			name:     "stylesheet",
			url:      "https://example.com/style.css",
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
			name:     "xml file is crawlable",
			url:      "https://example.com/sitemap.xml",
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
		{
			name:     "binary extension in directory not file",
			url:      "https://example.com/images.zip/index.html",
			expected: false,
		},
		{
			name:     "binary file inside dotted directory",
			url:      "https://example.com/v1.2/archive.tar.gz",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, _ := url.Parse(tt.url)
			result := IsBinaryURL(u)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsBinaryURLNil(t *testing.T) {
	// Test that nil URL returns false without panicking
	assert.False(t, IsBinaryURL(nil))
}

func TestIsBinaryExtension(t *testing.T) {
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
			name:     "html not binary",
			ext:      ".html",
			expected: false,
		},
		{
			name:     "without leading dot",
			ext:      "jpg",
			expected: true,
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
			result := IsBinaryExtension(tt.ext)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtensionSet(t *testing.T) {
	t.Run("Add normalizes case and dots", func(t *testing.T) {
		s := NewExtensionSet("JPG", ".PNG", " .gif ")
		assert.True(t, s.Contains(".jpg"))
		assert.True(t, s.Contains("png"))
		assert.True(t, s.Contains(".GIF"))
		assert.False(t, s.Contains(".webp"))
	})

	t.Run("Remove deletes extensions", func(t *testing.T) {
		s := NewExtensionSet(".jpg", ".png")
		s.Remove("jpg")
		assert.False(t, s.Contains(".jpg"))
		assert.True(t, s.Contains(".png"))
	})

	t.Run("Clone is independent", func(t *testing.T) {
		s := NewExtensionSet(".jpg")
		clone := s.Clone()
		clone.Add(".png")
		clone.Remove(".jpg")
		assert.True(t, s.Contains(".jpg"))
		assert.False(t, s.Contains(".png"))
		assert.True(t, clone.Contains(".png"))
	})

	t.Run("ContainsURL", func(t *testing.T) {
		s := NewExtensionSet(".pdf")
		u, _ := url.Parse("https://example.com/file.pdf")
		assert.True(t, s.ContainsURL(u))
		u, _ = url.Parse("https://example.com/page.html")
		assert.False(t, s.ContainsURL(u))
		assert.False(t, s.ContainsURL(nil))
	})

	t.Run("empty extension is not added", func(t *testing.T) {
		s := NewExtensionSet("", ".")
		assert.Equal(t, 0, len(s))
	})
}

// Example demonstrates detecting file downloads from URLs.
func ExampleIsBinaryURL() {
	imageURL, _ := url.Parse("https://example.com/photo.jpg")
	fmt.Println(IsBinaryURL(imageURL))

	videoURL, _ := url.Parse("https://example.com/video.mp4")
	fmt.Println(IsBinaryURL(videoURL))

	pageURL, _ := url.Parse("https://example.com/page.html")
	fmt.Println(IsBinaryURL(pageURL))

	// Output:
	// true
	// true
	// false
}

// Example demonstrates customizing the default extension set.
func ExampleExtensionSet() {
	// Crawl PDFs, but skip everything else in the default set
	exts := BinaryExtensions.Clone()
	exts.Remove(".pdf")

	pdfURL, _ := url.Parse("https://example.com/paper.pdf")
	zipURL, _ := url.Parse("https://example.com/archive.zip")
	fmt.Println(exts.ContainsURL(pdfURL))
	fmt.Println(exts.ContainsURL(zipURL))

	// Output:
	// false
	// true
}
