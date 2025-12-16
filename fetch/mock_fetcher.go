package fetch

import (
	"context"
	"fmt"
	"sync"
)

// MockFetcher implements the Fetcher interface for testing.
type MockFetcher struct {
	responses map[string]*Response
	errors    map[string]error
	mutex     sync.RWMutex
}

// NewMockFetcher creates a new mock fetcher.
func NewMockFetcher() *MockFetcher {
	return &MockFetcher{
		responses: make(map[string]*Response),
		errors:    make(map[string]error),
	}
}

// AddResponse adds a mock response for the given URL.
func (m *MockFetcher) AddResponse(url string, response *Response) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.responses[url] = response
}

// AddError adds a mock error for the given URL.
func (m *MockFetcher) AddError(url string, err error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.errors[url] = err
}

// Fetch implements the Fetcher interface.
func (m *MockFetcher) Fetch(ctx context.Context, req *Request) (*Response, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if err, exists := m.errors[req.URL]; exists {
		return nil, err
	}
	if response, exists := m.responses[req.URL]; exists {
		return response, nil
	}
	return nil, fmt.Errorf("no mock response configured for url %q", req.URL)
}
