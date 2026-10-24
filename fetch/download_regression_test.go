package fetch

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestWithinDir(t *testing.T) {
	sep := string(filepath.Separator)
	root := filepath.VolumeName(os.TempDir()) + sep
	assert.True(t, withinDir(root, filepath.Join(root, "file.bin")), "file directly in the root directory")
	assert.True(t, withinDir(filepath.Join(root, "a"), filepath.Join(root, "a", "b")))
	assert.True(t, withinDir(filepath.Join(root, "a"), filepath.Join(root, "a")))
	assert.False(t, withinDir(filepath.Join(root, "a"), filepath.Join(root, "ab")))
	assert.False(t, withinDir(filepath.Join(root, "a"), root))
}

func TestDownloadFallsBackToURLWhenContentDispositionIsUnusable(t *testing.T) {
	for _, header := range []string{`attachment; filename="."`, `attachment; filename="/"`, `attachment; filename=".."`} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Disposition", header)
			w.Write([]byte("data"))
		}))
		result, err := Download(context.Background(), server.URL+"/report.csv", nil)
		server.Close()
		assert.NoError(t, err, header)
		assert.Equal(t, "report.csv", result.Filename, header)
	}
}

func TestDownloadKeepsExistingFileWhenTransferFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// Flush before the body is complete so no Content-Length is sent and
		// the size limit trips during the transfer, not before it.
		w.Write([]byte("this content "))
		w.(http.Flusher).Flush()
		w.Write([]byte("is too large for the limit"))
	}))
	defer server.Close()

	dir := t.TempDir()
	outputPath := filepath.Join(dir, "keep.bin")
	assert.NoError(t, os.WriteFile(outputPath, []byte("original"), 0o644))

	_, err := Download(context.Background(), server.URL+"/large", &DownloadOptions{
		OutputPath:   outputPath,
		MaxSizeBytes: 10,
	})
	assert.Error(t, err)

	data, readErr := os.ReadFile(outputPath)
	assert.NoError(t, readErr)
	assert.Equal(t, "original", string(data))

	entries, _ := os.ReadDir(dir)
	assert.Equal(t, 1, len(entries), "temporary file left behind")
}
