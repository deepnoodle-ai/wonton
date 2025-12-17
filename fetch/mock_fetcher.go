package fetch

import (
	"context"
	"fmt"
	"sync"
)

// MockFetcher implements the Fetcher interface for testing.
//
// MockFetcher allows you to configure canned responses and errors for specific URLs,
// making it easy to test code that depends on the Fetcher interface without making
// actual HTTP requests. It is thread-safe and can be used in concurrent tests.
//
// Example:
//
//	mock := fetch.NewMockFetcher()
//	mock.AddResponse("https://example.com", &fetch.Response{
//		StatusCode: 200,
//		HTML:       "<html><body>Test</body></html>",
//	})
//	mock.AddError("https://error.com", errors.New("connection failed"))
type MockFetcher struct {
	responses map[string]*Response
	errors    map[string]error
	mutex     sync.RWMutex
}

// NewMockFetcher creates a new MockFetcher instance.
//
// The returned fetcher has no configured responses or errors. Use AddResponse
// and AddError to configure the mock behavior.
func NewMockFetcher() *MockFetcher {
	return &MockFetcher{
		responses: make(map[string]*Response),
		errors:    make(map[string]error),
	}
}

// AddResponse configures a mock response for the given URL.
//
// When Fetch is called with this URL, the configured response will be returned.
// If both a response and an error are configured for the same URL, the error
// takes precedence.
func (m *MockFetcher) AddResponse(url string, response *Response) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.responses[url] = response
}

// AddError configures a mock error for the given URL.
//
// When Fetch is called with this URL, the configured error will be returned.
// Errors take precedence over responses if both are configured for the same URL.
func (m *MockFetcher) AddError(url string, err error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.errors[url] = err
}

// Fetch implements the Fetcher interface.
//
// Returns the configured response or error for the request URL. If no response
// or error is configured, returns an error indicating the URL was not mocked.
// The context parameter is ignored by the mock.
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
