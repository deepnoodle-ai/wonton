package fetch

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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

func TestDownloadIgnoresParameterEndingInFilename(t *testing.T) {
	for _, header := range []string{`attachment; x-filename="evil.bin"`, `attachment; x-filename=evil bin.bin`} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Disposition", header)
			w.Write([]byte("data"))
		}))
		dir := t.TempDir()
		result, err := Download(context.Background(), server.URL+"/real.txt", &DownloadOptions{
			OutputPath: dir + string(os.PathSeparator),
			Client:     server.Client(),
		})
		server.Close()
		assert.NoError(t, err, header)
		assert.Equal(t, "real.txt", result.Filename, header)
		assert.Equal(t, filepath.Join(dir, "real.txt"), result.Path, header)
	}
}

func TestDownloadLongFilename(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("data"))
	}))
	defer server.Close()

	outputPath := filepath.Join(t.TempDir(), strings.Repeat("a", 240)+".bin")
	result, err := Download(context.Background(), server.URL+"/x", &DownloadOptions{OutputPath: outputPath})
	assert.NoError(t, err)
	assert.Equal(t, outputPath, result.Path)
}

func TestDownloadRefusesSymlinkEscapingOutputDir(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Disposition", `attachment; filename="escape.txt"`)
		w.Write([]byte("pwned"))
	}))
	defer server.Close()

	dir := t.TempDir()
	outside := filepath.Join(t.TempDir(), "secret.txt")
	assert.NoError(t, os.WriteFile(outside, []byte("original"), 0o644))
	if err := os.Symlink(outside, filepath.Join(dir, "escape.txt")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	_, err := Download(context.Background(), server.URL+"/x", &DownloadOptions{
		OutputPath: dir + string(os.PathSeparator),
	})
	assert.Error(t, err)
	data, _ := os.ReadFile(outside)
	assert.Equal(t, "original", string(data))
}

func TestDownloadFollowsSymlinkInsideOutputDir(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Disposition", `attachment; filename="link.txt"`)
		w.Write([]byte("new"))
	}))
	defer server.Close()

	dir := t.TempDir()
	target := filepath.Join(dir, "target.txt")
	assert.NoError(t, os.WriteFile(target, []byte("old"), 0o644))
	if err := os.Symlink(target, filepath.Join(dir, "link.txt")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	_, err := Download(context.Background(), server.URL+"/x", &DownloadOptions{
		OutputPath: dir + string(os.PathSeparator),
	})
	assert.NoError(t, err)
	data, _ := os.ReadFile(target)
	assert.Equal(t, "new", string(data))
}

func TestDownloadWritesThroughSymlink(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("new"))
	}))
	defer server.Close()

	dir := t.TempDir()
	target := filepath.Join(dir, "target.bin")
	link := filepath.Join(dir, "link.bin")
	assert.NoError(t, os.WriteFile(target, []byte("old"), 0o644))
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	_, err := Download(context.Background(), server.URL+"/x", &DownloadOptions{OutputPath: link})
	assert.NoError(t, err)
	info, err := os.Lstat(link)
	assert.NoError(t, err)
	assert.True(t, info.Mode()&os.ModeSymlink != 0, "symlink was replaced by a regular file")
	data, _ := os.ReadFile(target)
	assert.Equal(t, "new", string(data))
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
