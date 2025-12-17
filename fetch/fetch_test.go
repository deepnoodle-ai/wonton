package fetch

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/deepnoodle-ai/wonton/assert"
)

// Test errors.go

func TestRequestError(t *testing.T) {
	err := errors.New("test error")
	reqErr := NewRequestError(err)

	assert.Equal(t, "test error", reqErr.Error())
	assert.Equal(t, err, reqErr.Unwrap())
	assert.Equal(t, 0, reqErr.StatusCode())
	assert.Equal(t, "", reqErr.RawURL())
}

func TestRequestErrorf(t *testing.T) {
	reqErr := NewRequestErrorf("error: %s", "test")
	assert.Equal(t, "error: test", reqErr.Error())
}

func TestRequestError_WithStatusCode(t *testing.T) {
	reqErr := NewRequestError(errors.New("error")).WithStatusCode(404)
	assert.Equal(t, 404, reqErr.StatusCode())
}

func TestRequestError_WithRawURL(t *testing.T) {
	reqErr := NewRequestError(errors.New("error")).WithRawURL("https://example.com")
	assert.Equal(t, "https://example.com", reqErr.RawURL())
}

func TestRequestError_Chaining(t *testing.T) {
	reqErr := NewRequestError(errors.New("error")).
		WithStatusCode(500).
		WithRawURL("https://test.com")

	assert.Equal(t, 500, reqErr.StatusCode())
	assert.Equal(t, "https://test.com", reqErr.RawURL())
}

func TestIsRequestError(t *testing.T) {
	assert.False(t, IsRequestError(nil))
	assert.False(t, IsRequestError(errors.New("regular error")))
	assert.True(t, IsRequestError(NewRequestError(errors.New("request error"))))
}

func TestErrUnsupportedOption(t *testing.T) {
	err := ErrUnsupportedOption("mobile")
	assert.True(t, errors.Is(err, ErrUnsupported))
	assert.Contains(t, err.Error(), "mobile")
}

// Test mock_fetcher.go

func TestMockFetcher_AddResponse(t *testing.T) {
	fetcher := NewMockFetcher()
	response := &Response{
		URL:        "https://example.com",
		StatusCode: 200,
		HTML:       "<html><body>test</body></html>",
	}
	fetcher.AddResponse("https://example.com", response)

	ctx := context.Background()
	result, err := fetcher.Fetch(ctx, &Request{URL: "https://example.com"})
	assert.NoError(t, err)
	assert.Equal(t, response, result)
}

func TestMockFetcher_AddError(t *testing.T) {
	fetcher := NewMockFetcher()
	expectedErr := errors.New("mock error")
	fetcher.AddError("https://example.com", expectedErr)

	ctx := context.Background()
	_, err := fetcher.Fetch(ctx, &Request{URL: "https://example.com"})
	assert.Equal(t, expectedErr, err)
}

func TestMockFetcher_NoMockConfigured(t *testing.T) {
	fetcher := NewMockFetcher()

	ctx := context.Background()
	_, err := fetcher.Fetch(ctx, &Request{URL: "https://unknown.com"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no mock response configured")
}

// Test actions.go

func TestBaseAction_GetType(t *testing.T) {
	action := BaseAction{Type: "test"}
	assert.Equal(t, "test", action.GetType())
}

func TestNewScreenshotAction(t *testing.T) {
	action := NewScreenshotAction(ScreenshotActionOptions{FullPage: true})
	assert.Equal(t, "screenshot", action.Action.GetType())

	screenshotAction, ok := action.Action.(*ScreenshotAction)
	assert.True(t, ok)
	assert.True(t, screenshotAction.FullPage)
}

func TestNewWaitAction(t *testing.T) {
	action := NewWaitAction(WaitActionOptions{
		Selector:     ".content",
		Milliseconds: 1000,
	})
	assert.Equal(t, "wait", action.Action.GetType())

	waitAction, ok := action.Action.(*WaitAction)
	assert.True(t, ok)
	assert.Equal(t, ".content", waitAction.Selector)
	assert.Equal(t, 1000, waitAction.Milliseconds)
}

func TestAction_MarshalJSON(t *testing.T) {
	action := NewScreenshotAction(ScreenshotActionOptions{FullPage: true})
	data, err := json.Marshal(action)
	assert.NoError(t, err)

	// Verify the marshaled JSON has the expected structure
	assert.Contains(t, string(data), `"type":"screenshot"`)
	assert.Contains(t, string(data), `"full_page":true`)
}

func TestAction_UnmarshalJSON_Screenshot(t *testing.T) {
	data := []byte(`{"type":"screenshot","full_page":true}`)
	var action Action
	err := json.Unmarshal(data, &action)
	assert.NoError(t, err)

	screenshotAction, ok := action.Action.(*ScreenshotAction)
	assert.True(t, ok)
	assert.Equal(t, "screenshot", screenshotAction.Type)
	assert.True(t, screenshotAction.FullPage)
}

func TestAction_UnmarshalJSON_Wait(t *testing.T) {
	data := []byte(`{"type":"wait","selector":".content","milliseconds":500}`)
	var action Action
	err := json.Unmarshal(data, &action)
	assert.NoError(t, err)

	waitAction, ok := action.Action.(*WaitAction)
	assert.True(t, ok)
	assert.Equal(t, "wait", waitAction.Type)
	assert.Equal(t, ".content", waitAction.Selector)
	assert.Equal(t, 500, waitAction.Milliseconds)
}

func TestAction_UnmarshalJSON_Unknown(t *testing.T) {
	data := []byte(`{"type":"unknown"}`)
	var action Action
	err := json.Unmarshal(data, &action)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown action type")
}

func TestAction_UnmarshalJSON_InvalidJSON(t *testing.T) {
	data := []byte(`{invalid}`)
	var action Action
	err := json.Unmarshal(data, &action)
	assert.Error(t, err)
}

// Test http_fetcher.go

func TestNewHTTPFetcher_Defaults(t *testing.T) {
	fetcher := NewHTTPFetcher(HTTPFetcherOptions{})
	assert.NotNil(t, fetcher)
	assert.Equal(t, DefaultTimeout, fetcher.timeout)
	assert.Equal(t, int64(DefaultMaxBodySize), fetcher.maxBodySize)
}

func TestNewHTTPFetcher_CustomOptions(t *testing.T) {
	client := &http.Client{}
	headers := map[string]string{"User-Agent": "test"}
	fetcher := NewHTTPFetcher(HTTPFetcherOptions{
		Timeout:     10 * time.Second,
		Headers:     headers,
		Client:      client,
		MaxBodySize: 1024,
	})

	assert.Equal(t, 10*time.Second, fetcher.timeout)
	assert.Equal(t, int64(1024), fetcher.maxBodySize)
}

func TestHTTPFetcher_ValidateRequest_Unsupported(t *testing.T) {
	fetcher := NewHTTPFetcher(HTTPFetcherOptions{})

	tests := []struct {
		name    string
		request *Request
		errMsg  string
	}{
		{"MaxAge", &Request{MaxAge: 1000}, "MaxAge"},
		{"WaitFor", &Request{WaitFor: 1000}, "WaitFor"},
		{"Mobile", &Request{Mobile: true}, "Mobile"},
		{"Actions", &Request{Actions: []Action{{}}}, "Actions"},
		{"StorageState", &Request{StorageState: map[string]any{"key": "value"}}, "StorageState"},
		{"screenshot format", &Request{Formats: []string{"screenshot"}}, "screenshot"},
		{"json format", &Request{Formats: []string{"json"}}, "json"},
		{"summary format", &Request{Formats: []string{"summary"}}, "summary"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := fetcher.validateRequest(tt.request)
			assert.Error(t, err)
			assert.True(t, errors.Is(err, ErrUnsupported))
		})
	}
}

func TestHTTPFetcher_ValidateRequest_Supported(t *testing.T) {
	fetcher := NewHTTPFetcher(HTTPFetcherOptions{})

	tests := []struct {
		name    string
		request *Request
	}{
		{"empty", &Request{}},
		{"markdown format", &Request{Formats: []string{"markdown"}}},
		{"html format", &Request{Formats: []string{"html"}}},
		{"raw_html format", &Request{Formats: []string{"raw_html"}}},
		{"links format", &Request{Formats: []string{"links"}}},
		{"images format", &Request{Formats: []string{"images"}}},
		{"branding format", &Request{Formats: []string{"branding"}}},
		{"multiple formats", &Request{Formats: []string{"html", "markdown", "links"}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := fetcher.validateRequest(tt.request)
			assert.NoError(t, err)
		})
	}
}

func TestHTTPFetcher_Fetch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<html><head><title>Test</title></head><body>Hello</body></html>"))
	}))
	defer server.Close()

	fetcher := NewHTTPFetcher(HTTPFetcherOptions{})
	ctx := context.Background()

	resp, err := fetcher.Fetch(ctx, &Request{URL: server.URL})
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Contains(t, resp.HTML, "Hello")
	assert.Equal(t, "Test", resp.Metadata.Title)
}

func TestHTTPFetcher_Fetch_CustomHeaders(t *testing.T) {
	var receivedHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeader = r.Header.Get("X-Custom")
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<html><body>test</body></html>"))
	}))
	defer server.Close()

	fetcher := NewHTTPFetcher(HTTPFetcherOptions{})
	ctx := context.Background()

	_, err := fetcher.Fetch(ctx, &Request{
		URL:     server.URL,
		Headers: map[string]string{"X-Custom": "test-value"},
	})
	assert.NoError(t, err)
	assert.Equal(t, "test-value", receivedHeader)
}

func TestHTTPFetcher_Fetch_WrongContentType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"key":"value"}`))
	}))
	defer server.Close()

	fetcher := NewHTTPFetcher(HTTPFetcherOptions{})
	ctx := context.Background()

	_, err := fetcher.Fetch(ctx, &Request{URL: server.URL})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected content type")
}

func TestHTTPFetcher_Fetch_WithTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<html><body>test</body></html>"))
	}))
	defer server.Close()

	fetcher := NewHTTPFetcher(HTTPFetcherOptions{})
	ctx := context.Background()

	// Request timeout should cause the request to fail
	_, err := fetcher.Fetch(ctx, &Request{
		URL:     server.URL,
		Timeout: 10, // 10ms timeout
	})
	assert.Error(t, err)
}

func TestProcessRequest_EmptyHTML(t *testing.T) {
	resp, err := ProcessRequest(&Request{URL: "https://example.com"}, "")
	assert.NoError(t, err)
	assert.Equal(t, "https://example.com", resp.URL)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestProcessRequest_WithMarkdown(t *testing.T) {
	html := "<html><body><h1>Hello</h1><p>World</p></body></html>"
	resp, err := ProcessRequest(&Request{
		URL:     "https://example.com",
		Formats: []string{"markdown"},
	}, html)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp.Markdown)
	assert.Contains(t, resp.Markdown, "Hello")
}

func TestProcessRequest_WithHTML(t *testing.T) {
	html := "<html><body><p>Test</p></body></html>"
	resp, err := ProcessRequest(&Request{
		URL:     "https://example.com",
		Formats: []string{"html"},
	}, html)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp.HTML)
}

func TestProcessRequest_WithRawHTML(t *testing.T) {
	html := "<html><body><p>Test</p></body></html>"
	resp, err := ProcessRequest(&Request{
		URL:     "https://example.com",
		Formats: []string{"raw_html"},
	}, html)
	assert.NoError(t, err)
	assert.Equal(t, html, resp.RawHTML)
}

func TestProcessRequest_WithLinks(t *testing.T) {
	html := `<html><body><a href="https://example.com">Link</a></body></html>`
	resp, err := ProcessRequest(&Request{
		URL:     "https://example.com",
		Formats: []string{"links"},
	}, html)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp.Links)
}

func TestProcessRequest_WithImages(t *testing.T) {
	html := `<html><body><img src="image.png" alt="test"></body></html>`
	resp, err := ProcessRequest(&Request{
		URL:     "https://example.com",
		Formats: []string{"images"},
	}, html)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp.Images)
}

func TestProcessRequest_WithBranding(t *testing.T) {
	html := `<html>
		<head>
			<meta name="theme-color" content="#ff0000">
			<link rel="icon" href="/favicon.ico">
		</head>
		<body></body>
	</html>`
	resp, err := ProcessRequest(&Request{
		URL:     "https://example.com",
		Formats: []string{"branding"},
	}, html)
	assert.NoError(t, err)
	assert.NotNil(t, resp.Branding)
	assert.NotNil(t, resp.Branding.Colors)
	assert.Equal(t, "#ff0000", resp.Branding.Colors.Primary)
}

func TestProcessRequest_WithOnlyMainContent(t *testing.T) {
	html := `<html>
		<body>
			<nav>Navigation</nav>
			<main><p>Main content</p></main>
			<footer>Footer</footer>
		</body>
	</html>`
	resp, err := ProcessRequest(&Request{
		URL:             "https://example.com",
		OnlyMainContent: true,
		Formats:         []string{"html"},
	}, html)
	assert.NoError(t, err)
	assert.Contains(t, resp.HTML, "Main content")
}

func TestProcessRequest_WithPrettify(t *testing.T) {
	html := "<html><body><p>Test</p></body></html>"
	resp, err := ProcessRequest(&Request{
		URL:      "https://example.com",
		Prettify: true,
	}, html)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp.HTML)
}

func TestProcessRequest_DefaultFormats(t *testing.T) {
	// When no formats are specified, HTML should be included by default
	html := "<html><body><p>Test</p></body></html>"
	resp, err := ProcessRequest(&Request{URL: "https://example.com"}, html)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp.HTML)
	assert.Empty(t, resp.Markdown)
	assert.Empty(t, resp.RawHTML)
}
