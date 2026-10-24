package fetch

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

func contentTypeServer(t *testing.T, contentType, body string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", contentType)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	return server
}

func TestHTTPFetcherReturnsPlainTextForRawRequests(t *testing.T) {
	server := contentTypeServer(t, "text/plain; charset=utf-8", "User-agent: *\nDisallow: /\n")
	fetcher := NewHTTPFetcher(HTTPFetcherOptions{Client: server.Client()})

	resp, err := fetcher.Fetch(context.Background(), &Request{URL: server.URL, Formats: []string{"raw_html"}})
	assert.NoError(t, err)
	assert.Equal(t, "User-agent: *\nDisallow: /\n", resp.RawHTML)
	assert.Equal(t, "", resp.HTML)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Without a raw request, plain text is still refused.
	_, err = fetcher.Fetch(context.Background(), &Request{URL: server.URL})
	assert.ErrorIs(t, err, ErrUnsupportedContentType)
}

func TestHTTPFetcherMatchesContentTypeCaseInsensitively(t *testing.T) {
	server := contentTypeServer(t, "TEXT/HTML; charset=UTF-8", "<html><body><p>hi</p></body></html>")
	fetcher := NewHTTPFetcher(HTTPFetcherOptions{Client: server.Client()})

	resp, err := fetcher.Fetch(context.Background(), &Request{URL: server.URL})
	assert.NoError(t, err)
	assert.Contains(t, resp.HTML, "hi")
}

func TestHTTPFetcherEmptyHTMLHasTimestamp(t *testing.T) {
	server := contentTypeServer(t, "text/html", "")
	fetcher := NewHTTPFetcher(HTTPFetcherOptions{Client: server.Client()})

	resp, err := fetcher.Fetch(context.Background(), &Request{URL: server.URL})
	assert.NoError(t, err)
	assert.False(t, resp.Timestamp.IsZero())
	assert.NotNil(t, resp.Headers)
}
