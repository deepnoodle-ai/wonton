package fetch

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
	"github.com/deepnoodle-ai/wonton/httpguard"
)

// TestMain points the package defaults at unguarded clients for the rest of
// the suite. The real defaults refuse non-public addresses, and every test
// server here listens on 127.0.0.1 — the same swap a caller makes to fetch
// from localhost or an intranet host. TestDefaultClientsRefuse... below builds
// the guarded defaults explicitly and checks they still do.
func TestMain(m *testing.M) {
	DefaultHTTPClient = &http.Client{Timeout: DefaultTimeout}
	DefaultDownloadClient = &http.Client{}
	os.Exit(m.Run())
}

func TestDefaultClientsRefuseNonPublicAddresses(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<html><body>should not be reachable</body></html>"))
	}))
	defer server.Close()

	fetcher := NewHTTPFetcher(HTTPFetcherOptions{Client: newGuardedClient(DefaultTimeout)})
	_, err := fetcher.Fetch(context.Background(), &Request{URL: server.URL})
	assert.ErrorIs(t, err, httpguard.ErrNonPublicAddress)

	_, err = Download(context.Background(), server.URL, &DownloadOptions{
		Client: newGuardedDownloadClient(),
	})
	assert.ErrorIs(t, err, httpguard.ErrNonPublicAddress)
}

func TestGuardedClientFollowsRedirects(t *testing.T) {
	// A fetcher that cannot follow redirects is not much use, so the guarded
	// default keeps them — including plain-HTTP hops, which plenty of
	// legitimate sites still use.
	client := newGuardedClient(DefaultTimeout)
	assert.Equal(t, client.Timeout, DefaultTimeout)

	next, err := http.NewRequest(http.MethodGet, "http://example.com/next", nil)
	assert.NoError(t, err)
	assert.NoError(t, client.CheckRedirect(next, []*http.Request{next}))

	chain := make([]*http.Request, defaultMaxRedirects+1)
	for i := range chain {
		chain[i] = next
	}
	assert.ErrorIs(t, client.CheckRedirect(next, chain), httpguard.ErrRedirectRefused,
		"the redirect chain is still bounded")
}
