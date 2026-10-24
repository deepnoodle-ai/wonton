//go:build unix

package fetch

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestDownloadNewFileHonorsUmask(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("data"))
	}))
	defer server.Close()

	old := syscall.Umask(0o077)
	defer syscall.Umask(old)

	outputPath := filepath.Join(t.TempDir(), "new.bin")
	_, err := Download(context.Background(), server.URL+"/x", &DownloadOptions{OutputPath: outputPath})
	assert.NoError(t, err)
	info, err := os.Stat(outputPath)
	assert.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}
