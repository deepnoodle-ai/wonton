package crawler

import (
	"context"

	"github.com/deepnoodle-ai/wonton/fetch"
)

// MockParser implements the Parser interface for testing. It allows you to
// configure custom parsing behavior using SetParseFunc, making it ideal for
// unit tests that need to simulate different parsing scenarios.
type MockParser struct {
	parseFunc func(ctx context.Context, page *fetch.Response) (any, error)
}

// NewMockParser creates a new MockParser instance. By default, it returns
// a simple map with {"parsed": "data"} unless a custom parse function is set.
func NewMockParser() *MockParser {
	return &MockParser{}
}

// SetParseFunc configures a custom parsing function. This function will be
// called by Parse() to produce parsed results. Use this to simulate different
// parsing outcomes in tests.
func (m *MockParser) SetParseFunc(fn func(ctx context.Context, page *fetch.Response) (any, error)) {
	m.parseFunc = fn
}

// Parse processes the page using the configured parse function if set, otherwise
// returns a default map. This implements the Parser interface.
func (m *MockParser) Parse(ctx context.Context, page *fetch.Response) (any, error) {
	if m.parseFunc != nil {
		return m.parseFunc(ctx, page)
	}
	return map[string]string{"parsed": "data"}, nil
}
