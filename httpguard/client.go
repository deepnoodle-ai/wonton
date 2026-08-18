// Package httpguard provides an HTTP client for requests to URLs your program
// did not choose: a webhook target a user configured, a link submitted through
// an API, a redirect a crawler decided to follow.
//
// Such a URL is an SSRF vector. It can name a hostname that resolves to
// 127.0.0.1, to a cloud metadata endpoint at 169.254.169.254, or to a service
// reachable only from inside your network. [NewClient] returns an
// [http.Client] that refuses to connect to any of those:
//
//   - Every address the hostname resolves to is validated, and the connection
//     is refused unless all of them are public. A hostname that mixes a public
//     and a private address is rejected outright rather than filtered down.
//   - The connection is made to a validated address, not re-resolved from the
//     hostname, so a name cannot resolve to a public address for the check and
//     a private one for the dial (DNS rebinding).
//   - Redirects are refused by default. [WithMaxRedirects] enables a bounded
//     number of them, and each hop is still HTTPS-only and re-validated at
//     dial time.
//   - Ambient HTTP(S)_PROXY environment variables are ignored, so a proxy
//     cannot relay the request somewhere the guard would have refused.
//
// TLS is unaffected: the handshake still uses the hostname for SNI and
// certificate verification, because only the dial address is pinned.
//
// The client sets no total timeout. Bound the whole request with the request
// context; the options here bound only connection setup, which a context
// deadline alone does not distinguish.
package httpguard

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

// Errors reported by clients from [NewClient]. Both are wrapped with detail,
// so test them with [errors.Is].
var (
	// ErrNonPublicAddress means the target resolved to, or literally named, an
	// address that is not publicly routable.
	ErrNonPublicAddress = errors.New("outbound target is not a public address")

	// ErrRedirectRefused means a redirect was not followed: redirects are
	// disabled, the chain exceeded the limit, or the target was not a plain
	// HTTPS URL.
	ErrRedirectRefused = errors.New("outbound redirect refused")
)

// Default bounds on connection setup, used when the corresponding option is
// not supplied or is not positive.
const (
	DefaultDNSTimeout          = 5 * time.Second
	DefaultConnectTimeout      = 10 * time.Second
	DefaultTLSHandshakeTimeout = 10 * time.Second
)

// nonPublicRanges are the ranges that Go's own net.IP predicates do not
// classify: shared address space, documentation and benchmark ranges, IPv4
// translation and transition ranges, and reserved space. Loopback, private,
// link-local, multicast, and unspecified addresses are handled by
// [ValidatePublicIP] directly.
var nonPublicRanges = []*net.IPNet{
	mustCIDR("0.0.0.0/8"),       // "this network"
	mustCIDR("100.64.0.0/10"),   // CGNAT / shared address space
	mustCIDR("192.0.0.0/24"),    // IETF protocol assignments
	mustCIDR("192.0.2.0/24"),    // TEST-NET-1
	mustCIDR("192.88.99.0/24"),  // 6to4 relay anycast (deprecated)
	mustCIDR("198.18.0.0/15"),   // benchmarking
	mustCIDR("198.51.100.0/24"), // TEST-NET-2
	mustCIDR("203.0.113.0/24"),  // TEST-NET-3
	mustCIDR("240.0.0.0/4"),     // reserved
	mustCIDR("64:ff9b::/96"),    // IPv4/IPv6 translation
	mustCIDR("64:ff9b:1::/48"),  // IPv4/IPv6 translation (local use)
	mustCIDR("100::/64"),        // discard-only
	mustCIDR("2001::/32"),       // Teredo
	mustCIDR("2001:2::/48"),     // benchmarking
	mustCIDR("2001:db8::/32"),   // documentation
	mustCIDR("2002::/16"),       // 6to4
	mustCIDR("fec0::/10"),       // site-local (deprecated)
}

// LookupFunc resolves a hostname to its addresses. It matches the signature of
// [net.Resolver.LookupIPAddr].
type LookupFunc func(ctx context.Context, host string) ([]net.IPAddr, error)

// DialFunc establishes a connection to an address. It matches the signature of
// [net.Dialer.DialContext].
type DialFunc func(ctx context.Context, network, address string) (net.Conn, error)

type config struct {
	dnsTimeout          time.Duration
	connectTimeout      time.Duration
	tlsHandshakeTimeout time.Duration
	maxRedirects        int
	lookup              LookupFunc
	dial                DialFunc
}

// Option configures a client returned by [NewClient].
type Option func(*config)

// WithDNSTimeout bounds hostname resolution. Defaults to
// [DefaultDNSTimeout]. Values <= 0 are ignored.
func WithDNSTimeout(d time.Duration) Option {
	return func(c *config) {
		if d > 0 {
			c.dnsTimeout = d
		}
	}
}

// WithConnectTimeout bounds the TCP connect phase, across every address tried.
// Defaults to [DefaultConnectTimeout]. Values <= 0 are ignored.
func WithConnectTimeout(d time.Duration) Option {
	return func(c *config) {
		if d > 0 {
			c.connectTimeout = d
		}
	}
}

// WithTLSHandshakeTimeout bounds the TLS handshake. Defaults to
// [DefaultTLSHandshakeTimeout]. Values <= 0 are ignored.
func WithTLSHandshakeTimeout(d time.Duration) Option {
	return func(c *config) {
		if d > 0 {
			c.tlsHandshakeTimeout = d
		}
	}
}

// WithMaxRedirects allows up to n redirects instead of refusing them outright.
// Every hop must be a plain HTTPS URL — no scheme downgrade, no userinfo, no
// fragment, no opaque form — and is re-validated at dial time like any other
// request. n <= 0 keeps the default of refusing all redirects.
func WithMaxRedirects(n int) Option {
	return func(c *config) { c.maxRedirects = n }
}

// WithResolver replaces the resolver used to look up hostnames, for a caching
// or otherwise custom resolver. Its results are validated exactly as the
// default resolver's are.
func WithResolver(lookup LookupFunc) Option {
	return func(c *config) {
		if lookup != nil {
			c.lookup = lookup
		}
	}
}

// WithDialContext replaces the dialer. It is called only with an address that
// has already been validated as public — a literal IP and port, never a
// hostname — so a custom dialer must connect to the address it is given and
// not re-resolve anything.
func WithDialContext(dial DialFunc) Option {
	return func(c *config) {
		if dial != nil {
			c.dial = dial
		}
	}
}

// NewClient returns an [http.Client] that connects only to public addresses.
//
// The returned client has no total timeout: bound the request with its
// context. Callers may adjust the returned client (to add a total timeout, for
// instance), but replacing its Transport or CheckRedirect removes the guard.
func NewClient(opts ...Option) *http.Client {
	cfg := config{
		dnsTimeout:          DefaultDNSTimeout,
		connectTimeout:      DefaultConnectTimeout,
		tlsHandshakeTimeout: DefaultTLSHandshakeTimeout,
		lookup:              net.DefaultResolver.LookupIPAddr,
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	if cfg.dial == nil {
		dialer := &net.Dialer{Timeout: cfg.connectTimeout, KeepAlive: 30 * time.Second}
		cfg.dial = dialer.DialContext
	}
	transport := &http.Transport{
		Proxy:               nil, // never relay through an ambient proxy
		DialContext:         cfg.dialGuarded,
		TLSHandshakeTimeout: cfg.tlsHandshakeTimeout,
		ForceAttemptHTTP2:   true,
		MaxIdleConns:        32,
		IdleConnTimeout:     30 * time.Second,
	}
	return &http.Client{
		Transport:     transport,
		CheckRedirect: cfg.checkRedirect,
	}
}

// dialGuarded resolves and validates the target, then connects to a validated
// address rather than to the hostname.
func (c *config) dialGuarded(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	if literal := net.ParseIP(host); literal != nil {
		if err := ValidatePublicIP(literal); err != nil {
			return nil, err
		}
		return c.connect(ctx, network, []string{address})
	}

	dnsCtx, cancelDNS := context.WithTimeout(ctx, c.dnsTimeout)
	resolved, err := c.lookup(dnsCtx, host)
	cancelDNS()
	if err != nil {
		return nil, err
	}
	if len(resolved) == 0 {
		return nil, fmt.Errorf("outbound target %q resolved to no addresses", host)
	}
	// Validate every address before dialing any of them: a hostname that
	// mixes public and non-public results is refused rather than filtered.
	addresses := make([]string, 0, len(resolved))
	for _, addr := range resolved {
		if err := ValidatePublicIP(addr.IP); err != nil {
			return nil, err
		}
		addresses = append(addresses, net.JoinHostPort(addr.IP.String(), port))
	}
	return c.connect(ctx, network, addresses)
}

// connect dials the validated addresses in order until one succeeds, so a host
// whose first address is unreachable (a AAAA record on an IPv4-only network,
// say) still connects. The whole phase shares one connect deadline.
func (c *config) connect(ctx context.Context, network string, addresses []string) (net.Conn, error) {
	connectCtx, cancel := context.WithTimeout(ctx, c.connectTimeout)
	defer cancel()
	var err error
	for _, address := range addresses {
		var conn net.Conn
		conn, err = c.dial(connectCtx, network, address)
		if err == nil {
			return conn, nil
		}
		if connectCtx.Err() != nil {
			break
		}
	}
	return nil, err
}

func (c *config) checkRedirect(req *http.Request, via []*http.Request) error {
	if c.maxRedirects <= 0 {
		return ErrRedirectRefused
	}
	if len(via) > c.maxRedirects {
		return fmt.Errorf("%w: more than %d redirects", ErrRedirectRefused, c.maxRedirects)
	}
	if req == nil || req.URL == nil {
		return fmt.Errorf("%w: no redirect target", ErrRedirectRefused)
	}
	target := req.URL
	if !strings.EqualFold(target.Scheme, "https") || target.Opaque != "" ||
		target.Host == "" || target.Hostname() == "" || target.User != nil || target.Fragment != "" {
		return fmt.Errorf("%w: %q is not a plain HTTPS URL", ErrRedirectRefused, target.Redacted())
	}
	return nil
}

// ValidatePublicIP reports whether ip is publicly routable, returning an error
// wrapping [ErrNonPublicAddress] when it is not. Loopback, private, link-local
// (including the 169.254.169.254 metadata address), multicast, unspecified,
// shared, documentation, benchmark, and reserved addresses are all rejected.
//
// Clients from [NewClient] apply this to every address they resolve. It is
// exported for callers that want the same check before they enqueue a URL, or
// at another layer of their own.
func ValidatePublicIP(ip net.IP) error {
	if ip == nil {
		return fmt.Errorf("%w: target is not resolvable", ErrNonPublicAddress)
	}
	if v4 := ip.To4(); v4 != nil {
		ip = v4
	}
	if !ip.IsGlobalUnicast() || ip.IsLoopback() || ip.IsUnspecified() || ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsInterfaceLocalMulticast() ||
		ip.IsMulticast() || inNonPublicRange(ip) {
		return fmt.Errorf("%w: %s", ErrNonPublicAddress, ip)
	}
	return nil
}

func inNonPublicRange(ip net.IP) bool {
	for _, network := range nonPublicRanges {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

func mustCIDR(value string) *net.IPNet {
	_, network, err := net.ParseCIDR(value)
	if err != nil {
		panic(err)
	}
	return network
}
