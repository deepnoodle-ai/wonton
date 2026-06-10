package fetch

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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

func TestFilenameFromContentDisposition(t *testing.T) {
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
			name:     "single quotes stripped",
			header:   `attachment; filename='document.pdf'`,
			expected: "document.pdf",
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
			result := filenameFromContentDisposition(tt.header)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// --- Download Tests ---

func TestDownloadEmptyURL(t *testing.T) {
	_, err := Download(context.Background(), "", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

func TestDownloadToMemory(t *testing.T) {
	content := []byte("test file content")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write(content)
	}))
	defer server.Close()

	// nil options downloads to memory
	result, err := Download(context.Background(), server.URL+"/test.txt", nil)

	assert.NoError(t, err)
	assert.Equal(t, content, result.Data)
	assert.Equal(t, int64(len(content)), result.Size)
	assert.Equal(t, "text/plain", result.ContentType)
	assert.Equal(t, "test.txt", result.Filename)
}

func TestDownloadDefaultUserAgent(t *testing.T) {
	var gotUA string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	_, err := Download(context.Background(), server.URL+"/x.txt", nil)
	assert.NoError(t, err)
	assert.True(t, strings.HasPrefix(gotUA, "wonton/"))

	// Caller-provided User-Agent wins
	_, err = Download(context.Background(), server.URL+"/x.txt", &DownloadOptions{
		Headers: map[string]string{"User-Agent": "MyApp/2.0"},
	})
	assert.NoError(t, err)
	assert.Equal(t, "MyApp/2.0", gotUA)
}

func TestDownloadSaveToFile(t *testing.T) {
	content := []byte("saved file content")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", `attachment; filename="saved.bin"`)
		w.Write(content)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "output.bin")

	result, err := Download(context.Background(), server.URL+"/download", &DownloadOptions{
		OutputPath: outputPath,
	})

	assert.NoError(t, err)
	assert.Equal(t, outputPath, result.Path)
	assert.Equal(t, int64(len(content)), result.Size)

	// Verify file contents
	savedContent, err := os.ReadFile(outputPath)
	assert.NoError(t, err)
	assert.Equal(t, content, savedContent)
}

func TestDownloadSaveToDirectory(t *testing.T) {
	content := []byte("directory save test")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Disposition", `attachment; filename="myfile.dat"`)
		w.Write(content)
	}))
	defer server.Close()

	tmpDir := t.TempDir()

	result, err := Download(context.Background(), server.URL+"/download", &DownloadOptions{
		OutputPath: tmpDir,
	})

	assert.NoError(t, err)
	expectedPath := filepath.Join(tmpDir, "myfile.dat")
	assert.Equal(t, expectedPath, result.Path)

	savedContent, err := os.ReadFile(expectedPath)
	assert.NoError(t, err)
	assert.Equal(t, content, savedContent)
}

func TestDownloadSizeLimitInMemory(t *testing.T) {
	content := []byte("this content is too large for the limit")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(content)
	}))
	defer server.Close()

	_, err := Download(context.Background(), server.URL+"/large", &DownloadOptions{
		MaxSizeBytes: 10,
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum")
}

func TestDownloadSizeLimitToFile(t *testing.T) {
	content := []byte("this content is too large for the limit")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(content)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "large.bin")

	_, err := Download(context.Background(), server.URL+"/large", &DownloadOptions{
		OutputPath:   outputPath,
		MaxSizeBytes: 10,
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum")

	// File should be cleaned up
	_, statErr := os.Stat(outputPath)
	assert.True(t, os.IsNotExist(statErr))
}

func TestDownloadExpectedType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/pdf; charset=utf-8")
		w.Write([]byte("pdf content"))
	}))
	defer server.Close()

	// Should succeed - media type matches even with charset parameter
	result, err := Download(context.Background(), server.URL+"/doc.pdf", &DownloadOptions{
		ExpectedType: "application/pdf",
	})
	assert.NoError(t, err)
	assert.Equal(t, "application/pdf; charset=utf-8", result.ContentType)

	// Should fail - wrong media type
	_, err = Download(context.Background(), server.URL+"/doc.pdf", &DownloadOptions{
		ExpectedType: "image/png",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mismatch")
}

func TestDownloadHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	_, err := Download(context.Background(), server.URL+"/missing", nil)

	assert.Error(t, err)
	var fetchErr *Error
	assert.True(t, errors.As(err, &fetchErr))
	assert.Equal(t, 404, fetchErr.StatusCode)
	assert.Equal(t, server.URL+"/missing", fetchErr.URL)
	assert.False(t, IsRetryable(err))
}

func TestDownloadRetryableHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	_, err := Download(context.Background(), server.URL+"/busy", nil)

	assert.Error(t, err)
	assert.True(t, IsRetryable(err))
}

func TestDownloadPathTraversalPrevention(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Malicious server tries to write outside target directory
		w.Header().Set("Content-Disposition", `attachment; filename="../../../etc/passwd"`)
		w.Write([]byte("malicious content"))
	}))
	defer server.Close()

	tmpDir := t.TempDir()

	result, err := Download(context.Background(), server.URL+"/malicious", &DownloadOptions{
		OutputPath: tmpDir,
	})

	// Should succeed but sanitize the filename
	assert.NoError(t, err)
	assert.Equal(t, "passwd", result.Filename)
	// File should be inside tmpDir
	assert.True(t, strings.HasPrefix(result.Path, tmpDir+string(filepath.Separator)))
}

func TestDownloadCreateDirs(t *testing.T) {
	content := []byte("nested directory content")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(content)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "a", "b", "c", "file.txt")

	result, err := Download(context.Background(), server.URL+"/file.txt", &DownloadOptions{
		OutputPath: outputPath,
		CreateDirs: true,
	})

	assert.NoError(t, err)
	assert.Equal(t, outputPath, result.Path)

	savedContent, err := os.ReadFile(outputPath)
	assert.NoError(t, err)
	assert.Equal(t, content, savedContent)
}

func TestDownloadCustomHeaders(t *testing.T) {
	var receivedAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		w.Write([]byte("protected content"))
	}))
	defer server.Close()

	_, err := Download(context.Background(), server.URL+"/protected", &DownloadOptions{
		Headers: map[string]string{
			"Authorization": "Bearer token123",
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, "Bearer token123", receivedAuth)
}

func TestDownloadExplicitOutputPathNoResponseFilename(t *testing.T) {
	// Server responds at root path with no Content-Disposition header
	content := []byte("content without filename")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(content)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "explicit-name.bin")

	// URL is just "/" with no filename derivable from response
	result, err := Download(context.Background(), server.URL+"/", &DownloadOptions{
		OutputPath: outputPath,
	})

	// Should succeed because OutputPath is an explicit file path
	assert.NoError(t, err)
	assert.Equal(t, "explicit-name.bin", result.Filename)
	assert.Equal(t, outputPath, result.Path)

	savedContent, err := os.ReadFile(outputPath)
	assert.NoError(t, err)
	assert.Equal(t, content, savedContent)
}

func TestDownloadDirectoryOutputNoFilename(t *testing.T) {
	// Server responds at root path with no Content-Disposition header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("content"))
	}))
	defer server.Close()

	tmpDir := t.TempDir()

	// URL is just "/" - no filename can be derived
	_, err := Download(context.Background(), server.URL+"/", &DownloadOptions{
		OutputPath: tmpDir, // directory, so filename must come from response
	})

	// Should fail because no filename can be determined
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "filename")
}

func TestDownloadInvalidExpectedType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/pdf")
		w.Write([]byte("content"))
	}))
	defer server.Close()

	_, err := Download(context.Background(), server.URL+"/file.pdf", &DownloadOptions{
		ExpectedType: "not a valid mime type;;;",
	})

	// Should fail with clear error about invalid expected type
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid expected type")
}

func TestDownloadNonExistentDirectoryWithCreateDirs(t *testing.T) {
	content := []byte("content for new directory")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Disposition", `attachment; filename="downloaded.bin"`)
		w.Write(content)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	// Path with trailing slash indicates it should be a directory
	newDirPath := filepath.Join(tmpDir, "newdir", "subdir") + string(filepath.Separator)

	result, err := Download(context.Background(), server.URL+"/download", &DownloadOptions{
		OutputPath: newDirPath,
		CreateDirs: true,
	})

	// Should succeed - create the directory and use filename from response
	assert.NoError(t, err)
	assert.Equal(t, "downloaded.bin", result.Filename)
	expectedPath := filepath.Join(tmpDir, "newdir", "subdir", "downloaded.bin")
	assert.Equal(t, expectedPath, result.Path)

	// Verify file was created in the right place
	savedContent, err := os.ReadFile(expectedPath)
	assert.NoError(t, err)
	assert.Equal(t, content, savedContent)
}

func TestDownloadNonExistentDirectoryNoTrailingSlash(t *testing.T) {
	content := []byte("content")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Disposition", `attachment; filename="downloaded.bin"`)
		w.Write(content)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	// Path without trailing slash - treated as file path, not directory
	newFilePath := filepath.Join(tmpDir, "newdir", "myfile.dat")

	result, err := Download(context.Background(), server.URL+"/download", &DownloadOptions{
		OutputPath: newFilePath,
		CreateDirs: true,
	})

	// Should succeed - create parent dirs and use explicit filename
	assert.NoError(t, err)
	assert.Equal(t, "myfile.dat", result.Filename)
	assert.Equal(t, newFilePath, result.Path)

	// Verify file was created
	savedContent, err := os.ReadFile(newFilePath)
	assert.NoError(t, err)
	assert.Equal(t, content, savedContent)
}

func TestDownloadOverwritesExistingFile(t *testing.T) {
	content := []byte("new content")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(content)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "file.txt")
	assert.NoError(t, os.WriteFile(outputPath, []byte("old content that is longer"), 0644))

	_, err := Download(context.Background(), server.URL+"/file.txt", &DownloadOptions{
		OutputPath: outputPath,
	})
	assert.NoError(t, err)

	saved, err := os.ReadFile(outputPath)
	assert.NoError(t, err)
	assert.Equal(t, content, saved)
}
