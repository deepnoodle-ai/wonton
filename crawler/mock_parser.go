package crawler

import (
	"context"

	"github.com/deepnoodle-ai/wonton/fetch"
)

// MockParser implements the Parser interface for testing
type MockParser struct {
	parseFunc func(ctx context.Context, page *fetch.Response) (any, error)
}

func NewMockParser() *MockParser {
	return &MockParser{}
}

func (m *MockParser) SetParseFunc(fn func(ctx context.Context, page *fetch.Response) (any, error)) {
	m.parseFunc = fn
}

func (m *MockParser) Parse(ctx context.Context, page *fetch.Response) (any, error) {
	if m.parseFunc != nil {
		return m.parseFunc(ctx, page)
	}
	return map[string]string{"parsed": "data"}, nil
}
