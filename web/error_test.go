package web

import (
	"fmt"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestFetchError(t *testing.T) {
	t.Run("Error message format", func(t *testing.T) {
		err := NewFetchError(404, fmt.Errorf("page not found"))
		assert.Equal(t, "fetch failed with status code 404: page not found", err.Error())
	})

	t.Run("Unwrap returns underlying error", func(t *testing.T) {
		underlying := fmt.Errorf("connection refused")
		err := NewFetchError(500, underlying)
		assert.Equal(t, underlying, err.Unwrap())
	})
}

func TestFetchErrorIsRecoverable(t *testing.T) {
	tests := []struct {
		name        string
		statusCode  int
		recoverable bool
	}{
		{"400 Bad Request", 400, false},
		{"401 Unauthorized", 401, false},
		{"403 Forbidden", 403, false},
		{"404 Not Found", 404, false},
		{"429 Too Many Requests", 429, true},
		{"500 Internal Server Error", 500, true},
		{"501 Not Implemented", 501, false},
		{"502 Bad Gateway", 502, true},
		{"503 Service Unavailable", 503, true},
		{"504 Gateway Timeout", 504, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewFetchError(tt.statusCode, fmt.Errorf("test error"))
			assert.Equal(t, tt.recoverable, err.IsRecoverable())
		})
	}
}
