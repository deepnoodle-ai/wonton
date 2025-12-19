package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

// --- Filename Sanitization Tests ---

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"normal filename", "document.pdf", "document.pdf"},
		{"path traversal attempt", "../../../etc/passwd", "passwd"},
		{"absolute path", "/etc/passwd", "passwd"},
		{"double dots", "..file.txt", "file.txt"},
		{"leading dots", "...hidden", "hidden"},
		{"trailing dots", "file...", "file"},
		{"just dots", "...", ""},
		{"just slash", "/", ""},
		{"just dot", ".", ""},
		{"empty string", "", ""},
		{"spaces around", "  file.txt  ", "file.txt"},
		{"control characters", "file\x00name.txt", "filename.txt"},
		{"forward slash in name", "sub/file.txt", "file.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeFilename(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractFilenameFromContentDisposition(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		expected string
	}{
		{
			name:     "standard attachment",
			header:   `attachment; filename="document.pdf"`,
			expected: "document.pdf",
		},
		{
			name:     "inline with filename",
			header:   `inline; filename="image.png"`,
			expected: "image.png",
		},
		{
			name:     "unquoted filename",
			header:   `attachment; filename=document.pdf`,
			expected: "document.pdf",
		},
		{
			name:     "single quotes - handled by fallback",
			header:   `attachment; filename='document.pdf'`,
			expected: "'document.pdf'", // mime.ParseMediaType treats as literal, fallback handles
		},
		{
			name:     "filename with spaces",
			header:   `attachment; filename="my document.pdf"`,
			expected: "my document.pdf",
		},
		{
			name:     "no filename",
			header:   `attachment`,
			expected: "",
		},
		{
			name:     "empty header",
			header:   "",
			expected: "",
		},
		{
			name:     "case insensitive",
			header:   `attachment; FILENAME="test.pdf"`,
			expected: "test.pdf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractFilenameFromContentDisposition(tt.header)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// --- BinaryFetcher Tests ---

func TestBinaryFetcherNilInput(t *testing.T) {
	fetcher := NewDefaultBinaryFetcher()
	_, err := fetcher.FetchBinary(context.Background(), nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nil")
}

func TestBinaryFetcherBasicFetch(t *testing.T) {
	content := []byte("test file content")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write(content)
	}))
	defer server.Close()

	fetcher := NewDefaultBinaryFetcher()
	result, err := fetcher.FetchBinary(context.Background(), &BinaryFetchInput{
		URL: server.URL + "/test.txt",
	})

	assert.NoError(t, err)
	assert.Equal(t, content, result.Data)
	assert.Equal(t, int64(len(content)), result.Size)
	assert.Equal(t, "text/plain", result.ContentType)
}

func TestBinaryFetcherSaveToFile(t *testing.T) {
	content := []byte("saved file content")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", `attachment; filename="saved.bin"`)
		w.Write(content)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "output.bin")

	fetcher := NewDefaultBinaryFetcher()
	result, err := fetcher.FetchBinary(context.Background(), &BinaryFetchInput{
		URL:        server.URL + "/download",
		OutputPath: outputPath,
	})

	assert.NoError(t, err)
	assert.Equal(t, outputPath, result.DownloadPath)
	assert.Equal(t, int64(len(content)), result.Size)

	// Verify file contents
	savedContent, err := os.ReadFile(outputPath)
	assert.NoError(t, err)
	assert.Equal(t, content, savedContent)
}

func TestBinaryFetcherSaveToDirectory(t *testing.T) {
	content := []byte("directory save test")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Disposition", `attachment; filename="myfile.dat"`)
		w.Write(content)
	}))
	defer server.Close()

	tmpDir := t.TempDir()

	fetcher := NewDefaultBinaryFetcher()
	result, err := fetcher.FetchBinary(context.Background(), &BinaryFetchInput{
		URL:        server.URL + "/download",
		OutputPath: tmpDir,
	})

	assert.NoError(t, err)
	expectedPath := filepath.Join(tmpDir, "myfile.dat")
	assert.Equal(t, expectedPath, result.DownloadPath)

	savedContent, err := os.ReadFile(expectedPath)
	assert.NoError(t, err)
	assert.Equal(t, content, savedContent)
}

func TestBinaryFetcherSizeLimitInMemory(t *testing.T) {
	content := []byte("this content is too large for the limit")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(content)
	}))
	defer server.Close()

	fetcher := NewDefaultBinaryFetcher()
	_, err := fetcher.FetchBinary(context.Background(), &BinaryFetchInput{
		URL:          server.URL + "/large",
		MaxSizeBytes: 10,
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum")
}

func TestBinaryFetcherSizeLimitToFile(t *testing.T) {
	content := []byte("this content is too large for the limit")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(content)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "large.bin")

	fetcher := NewDefaultBinaryFetcher()
	_, err := fetcher.FetchBinary(context.Background(), &BinaryFetchInput{
		URL:          server.URL + "/large",
		OutputPath:   outputPath,
		MaxSizeBytes: 10,
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum")

	// File should be cleaned up
	_, statErr := os.Stat(outputPath)
	assert.True(t, os.IsNotExist(statErr))
}

func TestBinaryFetcherMIMETypeVerification(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/pdf; charset=utf-8")
		w.Write([]byte("pdf content"))
	}))
	defer server.Close()

	fetcher := NewDefaultBinaryFetcher()

	// Should succeed - media type matches even with charset parameter
	result, err := fetcher.FetchBinary(context.Background(), &BinaryFetchInput{
		URL:            server.URL + "/doc.pdf",
		ExpectedType:   "application/pdf",
		VerifyMimeType: true,
	})
	assert.NoError(t, err)
	assert.Equal(t, "application/pdf; charset=utf-8", result.ContentType)

	// Should fail - wrong media type
	_, err = fetcher.FetchBinary(context.Background(), &BinaryFetchInput{
		URL:            server.URL + "/doc.pdf",
		ExpectedType:   "image/png",
		VerifyMimeType: true,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mismatch")
}

func TestBinaryFetcherHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	fetcher := NewDefaultBinaryFetcher()
	_, err := fetcher.FetchBinary(context.Background(), &BinaryFetchInput{
		URL: server.URL + "/missing",
	})

	assert.Error(t, err)
	fetchErr, ok := err.(*FetchError)
	assert.True(t, ok)
	assert.Equal(t, 404, fetchErr.StatusCode)
}

func TestBinaryFetcherPathTraversalPrevention(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Malicious server tries to write outside target directory
		w.Header().Set("Content-Disposition", `attachment; filename="../../../etc/passwd"`)
		w.Write([]byte("malicious content"))
	}))
	defer server.Close()

	tmpDir := t.TempDir()

	fetcher := NewDefaultBinaryFetcher()
	result, err := fetcher.FetchBinary(context.Background(), &BinaryFetchInput{
		URL:        server.URL + "/malicious",
		OutputPath: tmpDir,
	})

	// Should succeed but sanitize the filename
	assert.NoError(t, err)
	assert.Equal(t, "passwd", result.Filename)
	// File should be inside tmpDir
	assert.True(t, filepath.HasPrefix(result.DownloadPath, tmpDir))
}

func TestBinaryFetcherCreateDirs(t *testing.T) {
	content := []byte("nested directory content")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(content)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "a", "b", "c", "file.txt")

	fetcher := NewDefaultBinaryFetcher()
	result, err := fetcher.FetchBinary(context.Background(), &BinaryFetchInput{
		URL:        server.URL + "/file.txt",
		OutputPath: outputPath,
		CreateDirs: true,
	})

	assert.NoError(t, err)
	assert.Equal(t, outputPath, result.DownloadPath)

	savedContent, err := os.ReadFile(outputPath)
	assert.NoError(t, err)
	assert.Equal(t, content, savedContent)
}

func TestBinaryFetcherCustomHeaders(t *testing.T) {
	var receivedAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		w.Write([]byte("protected content"))
	}))
	defer server.Close()

	fetcher := NewDefaultBinaryFetcher()
	_, err := fetcher.FetchBinary(context.Background(), &BinaryFetchInput{
		URL: server.URL + "/protected",
		Headers: map[string]string{
			"Authorization": "Bearer token123",
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, "Bearer token123", receivedAuth)
}
