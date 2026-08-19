package httpguard

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/deepnoodle-ai/wonton/assert"
)

func transportOf(t *testing.T, client *http.Client) *http.Transport {
	t.Helper()
	transport, ok := client.Transport.(*http.Transport)
	assert.True(t, ok, "transport is %T, want *http.Transport", client.Transport)
	return transport
}

func staticLookup(addresses ...string) LookupFunc {
	return func(context.Context, string) ([]net.IPAddr, error) {
		out := make([]net.IPAddr, 0, len(addresses))
		for _, address := range addresses {
			out = append(out, net.IPAddr{IP: net.ParseIP(address)})
		}
		return out, nil
	}
}

func TestClientRejectsNonPublicLiteralAddresses(t *testing.T) {
	transport := transportOf(t, NewClient())
	for _, address := range []string{
		"127.0.0.1:443",           // loopback
		"10.0.0.1:443",            // private
		"172.16.0.1:443",          // private
		"192.168.1.1:443",         // private
		"169.254.169.254:443",     // cloud metadata
		"0.0.0.0:443",             // unspecified
		"100.64.0.1:443",          // CGNAT
		"192.0.0.1:443",           // IETF protocol assignments
		"192.0.2.1:443",           // TEST-NET-1
		"192.88.99.1:443",         // 6to4 relay anycast
		"198.18.0.1:443",          // benchmarking
		"198.51.100.1:443",        // TEST-NET-2
		"203.0.113.1:443",         // TEST-NET-3
		"224.0.0.1:443",           // multicast
		"240.0.0.1:443",           // reserved
		"255.255.255.255:443",     // broadcast
		"[::1]:443",               // loopback
		"[::]:443",                // unspecified
		"[64:ff9b::a00:1]:443",    // IPv4/IPv6 translation
		"[64:ff9b:1::a00:1]:443",  // IPv4/IPv6 translation, local use
		"[100::1]:443",            // discard-only
		"[2001::1]:443",           // Teredo
		"[2001:2::1]:443",         // benchmarking
		"[2001:db8::1]:443",       // documentation
		"[2002:c000:0201::1]:443", // 6to4
		"[fc00::1]:443",           // unique local
		"[fec0::1]:443",           // site-local
		"[fe80::1]:443",           // link-local
		"[ff02::1]:443",           // link-local multicast
	} {
		_, err := transport.DialContext(context.Background(), "tcp", address)
		assert.ErrorIs(t, err, ErrNonPublicAddress, "DialContext(%q)", address)
	}
}

func TestClientDisablesAmbientProxyAndTotalTimeout(t *testing.T) {
	client := NewClient()
	assert.Equal(t, client.Timeout, time.Duration(0), "the request context owns the total deadline")
	assert.Nil(t, transportOf(t, client).Proxy, "an ambient proxy could relay past the guard")
}

func TestClientRefusesRedirectsByDefault(t *testing.T) {
	client := NewClient()
	request, err := http.NewRequest(http.MethodGet, "https://example.com/next", nil)
	assert.NoError(t, err)
	assert.ErrorIs(t, client.CheckRedirect(request, nil), ErrRedirectRefused)
}

func TestClientAppliesSetupTimeouts(t *testing.T) {
	transport := transportOf(t, NewClient(WithTLSHandshakeTimeout(23*time.Millisecond)))
	assert.Equal(t, transport.TLSHandshakeTimeout, 23*time.Millisecond)
	assert.NotNil(t, transport.DialContext)

	// Non-positive values fall back to the defaults.
	transport = transportOf(t, NewClient(WithTLSHandshakeTimeout(0)))
	assert.Equal(t, transport.TLSHandshakeTimeout, DefaultTLSHandshakeTimeout)
}

func TestClientBoundsDNSLookupWithDNSTimeout(t *testing.T) {
	release := make(chan struct{})
	defer close(release)
	client := NewClient(
		WithDNSTimeout(20*time.Millisecond),
		WithConnectTimeout(time.Minute),
		WithResolver(func(ctx context.Context, _ string) ([]net.IPAddr, error) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-release:
				return nil, errors.New("lookup released")
			}
		}),
	)
	transport := transportOf(t, client)

	result := make(chan error, 1)
	go func() {
		_, err := transport.DialContext(context.Background(), "tcp", "example.com:443")
		result <- err
	}()
	select {
	case err := <-result:
		assert.ErrorIs(t, err, context.DeadlineExceeded)
	case <-time.After(2 * time.Second):
		t.Fatal("DNS lookup did not stop at the DNS timeout")
	}
}

func TestClientRejectsMixedPublicAndPrivateResolution(t *testing.T) {
	// A hostname resolving to both is refused outright: filtering down to the
	// public address would let an attacker pick which one the dial races to.
	client := NewClient(WithResolver(staticLookup("8.8.8.8", "10.0.0.1")))
	_, err := transportOf(t, client).DialContext(context.Background(), "tcp", "example.com:443")
	assert.ErrorIs(t, err, ErrNonPublicAddress)
}

func TestClientRejectsEmptyAndFailedResolution(t *testing.T) {
	client := NewClient(WithResolver(staticLookup()))
	_, err := transportOf(t, client).DialContext(context.Background(), "tcp", "example.com:443")
	assert.ErrorContains(t, err, "resolved to no addresses")

	lookupErr := &net.DNSError{Err: "not found", IsNotFound: true}
	client = NewClient(WithResolver(func(context.Context, string) ([]net.IPAddr, error) {
		return nil, lookupErr
	}))
	_, err = transportOf(t, client).DialContext(context.Background(), "tcp", "example.com:443")
	assert.ErrorIs(t, err, lookupErr)
}

func TestClientDialsTheValidatedAddressNotTheHostname(t *testing.T) {
	// Pinning is what closes the DNS-rebinding hole: the connection goes to an
	// address that was checked, not to a name that could resolve again.
	sentinel := errors.New("stop after observing the pinned dial")
	var dialed []string
	client := NewClient(
		WithResolver(staticLookup("8.8.8.8")),
		WithDialContext(func(_ context.Context, _, address string) (net.Conn, error) {
			dialed = append(dialed, address)
			return nil, sentinel
		}),
	)
	_, err := transportOf(t, client).DialContext(context.Background(), "tcp", "example.com:443")
	assert.ErrorIs(t, err, sentinel)
	assert.Equal(t, dialed, []string{"8.8.8.8:443"})
}

func TestClientPassesLiteralAddressesThroughUnchanged(t *testing.T) {
	sentinel := errors.New("stop after observing the dial")
	var dialed []string
	client := NewClient(WithDialContext(func(_ context.Context, _, address string) (net.Conn, error) {
		dialed = append(dialed, address)
		return nil, sentinel
	}))
	_, err := transportOf(t, client).DialContext(context.Background(), "tcp", "[2606:4700:4700::1111]:443")
	assert.ErrorIs(t, err, sentinel)
	assert.Equal(t, dialed, []string{"[2606:4700:4700::1111]:443"})
}

func TestClientFallsBackToLaterValidatedAddresses(t *testing.T) {
	// A host whose first address is unreachable (an AAAA record on an
	// IPv4-only network, say) still connects, since every address was
	// validated before any dial.
	unreachable := errors.New("network unreachable")
	sentinel := errors.New("stop after observing the second dial")
	var dialed []string
	client := NewClient(
		WithResolver(staticLookup("2606:4700:4700::1111", "1.1.1.1")),
		WithDialContext(func(_ context.Context, _, address string) (net.Conn, error) {
			dialed = append(dialed, address)
			if len(dialed) == 1 {
				return nil, unreachable
			}
			return nil, sentinel
		}),
	)
	_, err := transportOf(t, client).DialContext(context.Background(), "tcp", "example.com:443")
	assert.ErrorIs(t, err, sentinel)
	assert.Equal(t, dialed, []string{"[2606:4700:4700::1111]:443", "1.1.1.1:443"})
}

func TestClientStopsDialingAtTheConnectTimeout(t *testing.T) {
	client := NewClient(
		WithConnectTimeout(20*time.Millisecond),
		WithResolver(staticLookup("1.1.1.1", "8.8.8.8", "9.9.9.9")),
		WithDialContext(func(ctx context.Context, _, _ string) (net.Conn, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		}),
	)
	start := time.Now()
	_, err := transportOf(t, client).DialContext(context.Background(), "tcp", "example.com:443")
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Less(t, time.Since(start), 2*time.Second, "the connect timeout covers all addresses, not each one")
}

func TestClientAllowsBoundedGuardedHTTPSRedirects(t *testing.T) {
	client := NewClient(WithMaxRedirects(2))
	next, err := http.NewRequest(http.MethodGet, "https://example.com/next", nil)
	assert.NoError(t, err)

	assert.NoError(t, client.CheckRedirect(next, []*http.Request{next}), "first redirect")
	assert.NoError(t, client.CheckRedirect(next, []*http.Request{next, next}), "second redirect")
	assert.ErrorIs(t,
		client.CheckRedirect(next, []*http.Request{next, next, next}),
		ErrRedirectRefused,
		"third redirect exceeds the limit")

	for _, target := range []string{
		"http://example.com/next",       // scheme downgrade
		"https://user@example.com/next", // userinfo
		"https:opaque",                  // opaque form
	} {
		request, err := http.NewRequest(http.MethodGet, target, nil)
		assert.NoError(t, err)
		assert.ErrorIs(t, client.CheckRedirect(request, []*http.Request{next}), ErrRedirectRefused, "redirect to %q", target)
	}

	// A fragment never reaches the server, so it cannot change where the
	// request goes and is not a reason to refuse the hop.
	fragment, err := http.NewRequest(http.MethodGet, "https://example.com/next#section", nil)
	assert.NoError(t, err)
	assert.NoError(t, client.CheckRedirect(fragment, []*http.Request{next}))

	assert.ErrorIs(t, client.CheckRedirect(nil, []*http.Request{next}), ErrRedirectRefused, "nil redirect target")
}

func TestWithHTTPRedirectsAllowsPlainHTTPHops(t *testing.T) {
	strict := NewClient(WithMaxRedirects(2))
	relaxed := NewClient(WithMaxRedirects(2), WithHTTPRedirects())

	plain, err := http.NewRequest(http.MethodGet, "http://example.com/next", nil)
	assert.NoError(t, err)
	secure, err := http.NewRequest(http.MethodGet, "https://example.com/next", nil)
	assert.NoError(t, err)

	assert.ErrorIs(t, strict.CheckRedirect(plain, []*http.Request{secure}), ErrRedirectRefused)
	assert.NoError(t, relaxed.CheckRedirect(plain, []*http.Request{secure}))
	assert.NoError(t, relaxed.CheckRedirect(secure, []*http.Request{secure}))

	// The bound and the other target rules still apply.
	assert.ErrorIs(t,
		relaxed.CheckRedirect(plain, []*http.Request{secure, secure, secure}),
		ErrRedirectRefused,
		"the redirect limit still applies")
	withUser, err := http.NewRequest(http.MethodGet, "http://user@example.com/next", nil)
	assert.NoError(t, err)
	assert.ErrorIs(t, relaxed.CheckRedirect(withUser, []*http.Request{secure}), ErrRedirectRefused)

	// WithHTTPRedirects does nothing on its own.
	assert.ErrorIs(t,
		NewClient(WithHTTPRedirects()).CheckRedirect(secure, []*http.Request{secure}),
		ErrRedirectRefused,
		"redirects stay off unless WithMaxRedirects enables them")
}

func TestWithResponseHeaderTimeout(t *testing.T) {
	transport := transportOf(t, NewClient(WithResponseHeaderTimeout(9*time.Second)))
	assert.Equal(t, transport.ResponseHeaderTimeout, 9*time.Second)

	transport = transportOf(t, NewClient())
	assert.Equal(t, transport.ResponseHeaderTimeout, time.Duration(0), "unset means no bound")
}

func TestValidatePublicIP(t *testing.T) {
	for _, value := range []string{"8.8.8.8", "1.1.1.1", "2606:4700:4700::1111"} {
		assert.NoError(t, ValidatePublicIP(net.ParseIP(value)), "ValidatePublicIP(%q)", value)
	}
	for _, value := range []string{"127.0.0.1", "10.0.0.1", "169.254.169.254", "::1", "fc00::1"} {
		assert.ErrorIs(t, ValidatePublicIP(net.ParseIP(value)), ErrNonPublicAddress, "ValidatePublicIP(%q)", value)
	}
	// net.ParseIP returns nil for anything unparseable.
	assert.ErrorIs(t, ValidatePublicIP(net.ParseIP("not-an-ip")), ErrNonPublicAddress)
	assert.ErrorIs(t, ValidatePublicIP(nil), ErrNonPublicAddress)
}

func TestValidatePublicIPTreatsIPv4MappedAddressesAsIPv4(t *testing.T) {
	// ::ffff:127.0.0.1 must be rejected as loopback, not accepted as an
	// unrecognized IPv6 address.
	assert.ErrorIs(t, ValidatePublicIP(net.ParseIP("::ffff:127.0.0.1")), ErrNonPublicAddress)
	assert.ErrorIs(t, ValidatePublicIP(net.ParseIP("::ffff:10.0.0.1")), ErrNonPublicAddress)
	assert.NoError(t, ValidatePublicIP(net.ParseIP("::ffff:8.8.8.8")))
}

func TestClientRoundTripsThroughTheGuardedTransport(t *testing.T) {
	// The guard rejects a loopback address, so point a stub resolver at a
	// public address and dial the test server instead. Everything else — the
	// Host header, the redirect policy — behaves as it would in production.
	var gotHost string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHost = r.Host
		if r.URL.Path == "/redirect" {
			http.Redirect(w, r, "http://example.com/elsewhere", http.StatusFound)
			return
		}
		w.Write([]byte("hello"))
	}))
	defer server.Close()

	client := NewClient(
		WithResolver(staticLookup("8.8.8.8")),
		WithDialContext(func(ctx context.Context, network, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, network, server.Listener.Addr().String())
		}),
	)

	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://example.com/", nil)
	assert.NoError(t, err)
	resp, err := client.Do(request)
	assert.NoError(t, err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, string(body), "hello")
	assert.Equal(t, gotHost, "example.com", "pinning the dial address must not change the Host header")

	redirected, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://example.com/redirect", nil)
	assert.NoError(t, err)
	_, err = client.Do(redirected)
	assert.ErrorIs(t, err, ErrRedirectRefused)
}
