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
//     number of them; each hop is HTTPS-only unless [WithHTTPRedirects] says
//     otherwise, and is address-validated at dial time like any other
//     request.
//   - Credential headers (Authorization, Proxy-Authorization, and cookies) are
//     dropped when a redirect chain that began over HTTPS is downgraded to
//     plain HTTP. The standard client keeps them on a same-host downgrade,
//     which puts a bearer token on the wire in the clear.
//   - Ambient HTTP(S)_PROXY environment variables are ignored, so a proxy
//     cannot relay the request somewhere the guard would have refused.
//
// TLS is unaffected: the handshake still uses the hostname for SNI and
// certificate verification, because only the dial address is pinned.
//
// A caller that needs to reach one private host without giving up the guard
// everywhere else can widen the address check with [WithAddressValidator].
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
	"net/url"
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

	// DefaultMaxIdleConns is the total idle connection pool size, across all
	// hosts, used when [WithMaxIdleConns] is not supplied.
	DefaultMaxIdleConns = 32
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
	mustCIDR("2001:10::/28"),    // ORCHID (deprecated)
	mustCIDR("2001:db8::/32"),   // documentation
	mustCIDR("2002::/16"),       // 6to4
	mustCIDR("3fff::/20"),       // documentation (RFC 9637)
	mustCIDR("5f00::/16"),       // SRv6 SIDs (RFC 9602)
	mustCIDR("fec0::/10"),       // site-local (deprecated)
}

// LookupFunc resolves a hostname to its addresses. It matches the signature of
// [net.Resolver.LookupIPAddr].
type LookupFunc func(ctx context.Context, host string) ([]net.IPAddr, error)

// DialFunc establishes a connection to an address. It matches the signature of
// [net.Dialer.DialContext].
type DialFunc func(ctx context.Context, network, address string) (net.Conn, error)

// ValidateFunc reports whether an address may be dialed, returning an error
// when it may not. It matches the signature of [ValidatePublicIP].
type ValidateFunc func(ip net.IP) error

type config struct {
	dnsTimeout            time.Duration
	connectTimeout        time.Duration
	tlsHandshakeTimeout   time.Duration
	responseHeaderTimeout time.Duration
	maxRedirects          int
	allowHTTPRedirects    bool
	maxIdleConns          int
	maxIdleConnsPerHost   int
	lookup                LookupFunc
	dial                  DialFunc
	validate              ValidateFunc
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

// WithResponseHeaderTimeout bounds how long the server may take to send
// response headers after the request is written — a defense against a target
// that accepts the connection and then stalls. Unset means no bound, matching
// the standard transport. Values <= 0 are ignored.
func WithResponseHeaderTimeout(d time.Duration) Option {
	return func(c *config) {
		if d > 0 {
			c.responseHeaderTimeout = d
		}
	}
}

// WithMaxRedirects allows up to n redirects instead of refusing them outright.
// Every hop must be an HTTPS URL with no userinfo and no opaque form, and is
// address-validated at dial time like any other request. n <= 0 keeps the
// default of refusing all redirects. See [WithHTTPRedirects] to allow plain
// HTTP hops as well.
func WithMaxRedirects(n int) Option {
	return func(c *config) { c.maxRedirects = n }
}

// WithHTTPRedirects also allows plain HTTP redirect hops, which
// [WithMaxRedirects] alone refuses. Every hop is still address-validated, so
// this does not widen where a request can reach — but it lets a target
// downgrade the connection to plaintext. Credential headers are dropped on a
// downgraded hop (see [NewClient]); the body and everything else still cross
// in the clear, so do not use this for a signed or sensitive payload. It has
// no effect unless [WithMaxRedirects] enabled redirects.
func WithHTTPRedirects() Option {
	return func(c *config) { c.allowHTTPRedirects = true }
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

// WithMaxIdleConnsPerHost sets how many idle keep-alive connections the client
// keeps for a single host. Unset means [net/http.DefaultMaxIdleConnsPerHost],
// which is 2 — so a program making many concurrent requests to one host tears
// down and re-establishes a connection (and its TLS handshake) for all but two
// of them. Raise it when outbound traffic concentrates on a few hosts, such as
// webhook fan-out to one busy receiver. Values <= 0 are ignored.
//
// Keep it at or below the value given to [WithMaxIdleConns], which bounds the
// pool across all hosts: a per-host figure above the total cannot be reached.
func WithMaxIdleConnsPerHost(n int) Option {
	return func(c *config) {
		if n > 0 {
			c.maxIdleConnsPerHost = n
		}
	}
}

// WithMaxIdleConns sets the total number of idle keep-alive connections kept
// across all hosts. Defaults to [DefaultMaxIdleConns]. Values <= 0 are
// ignored; use a negative value only through the standard transport, since
// this option cannot express "unlimited".
func WithMaxIdleConns(n int) Option {
	return func(c *config) {
		if n > 0 {
			c.maxIdleConns = n
		}
	}
}

// WithAddressValidator replaces [ValidatePublicIP] as the check applied to
// every address, for a caller that needs to reach one specific private host —
// a dev server, an internal API — without giving up the guard everywhere
// else. It is the narrow escape hatch: widening the check here is far better
// than dropping the client altogether, which is the only other way to reach a
// private address.
//
// Build on [ValidatePublicIP] rather than replacing it wholesale, so that
// everything outside the exception stays guarded:
//
//	_, devNet, _ := net.ParseCIDR("10.1.2.0/24")
//	client := httpguard.NewClient(httpguard.WithAddressValidator(
//		func(ip net.IP) error {
//			if devNet.Contains(ip) {
//				return nil
//			}
//			return httpguard.ValidatePublicIP(ip)
//		}))
//
// The replacement is called for literal targets and for every resolved
// address alike, and the dial is still pinned to an address it accepted. A nil
// argument is ignored.
func WithAddressValidator(validate ValidateFunc) Option {
	return func(c *config) {
		if validate != nil {
			c.validate = validate
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
		maxIdleConns:        DefaultMaxIdleConns,
		lookup:              net.DefaultResolver.LookupIPAddr,
		validate:            ValidatePublicIP,
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	if cfg.dial == nil {
		dialer := &net.Dialer{Timeout: cfg.connectTimeout, KeepAlive: 30 * time.Second}
		cfg.dial = dialer.DialContext
	}
	transport := &http.Transport{
		Proxy:                 nil, // never relay through an ambient proxy
		DialContext:           cfg.dialGuarded,
		TLSHandshakeTimeout:   cfg.tlsHandshakeTimeout,
		ResponseHeaderTimeout: cfg.responseHeaderTimeout,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          cfg.maxIdleConns,
		MaxIdleConnsPerHost:   cfg.maxIdleConnsPerHost,
		IdleConnTimeout:       30 * time.Second,
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
		if err := c.validate(literal); err != nil {
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
		if err := c.validate(addr.IP); err != nil {
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
	var errs []error
	for _, address := range addresses {
		conn, err := c.dial(connectCtx, network, address)
		if err == nil && conn != nil {
			return conn, nil
		}
		if err == nil {
			// A custom dialer that reports neither a connection nor a
			// failure would otherwise hand the transport a nil conn.
			err = fmt.Errorf("dialer returned no connection for %s", address)
		}
		errs = append(errs, err)
		if connectCtx.Err() != nil {
			break
		}
	}
	switch len(errs) {
	case 0: // unreachable: callers always pass at least one address
		return nil, errors.New("outbound target has no address to dial")
	case 1:
		return nil, errs[0]
	default:
		// Report every attempt: with fallback, the first failure is usually
		// the one that explains the outcome.
		return nil, errors.Join(errs...)
	}
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
	if !c.allowedRedirectScheme(target.Scheme) || target.Opaque != "" ||
		target.Host == "" || target.Hostname() == "" || target.User != nil {
		return fmt.Errorf("%w: %q is not an allowed redirect target", ErrRedirectRefused, target.Redacted())
	}
	if isDowngrade(target, via) {
		stripCredentials(req.Header)
	}
	return nil
}

func (c *config) allowedRedirectScheme(scheme string) bool {
	if strings.EqualFold(scheme, "https") {
		return true
	}
	return c.allowHTTPRedirects && strings.EqualFold(scheme, "http")
}

// credentialHeaders carry a secret that must not cross a plaintext hop. The
// standard client already drops them when a redirect leaves the original
// domain, but it compares hosts only, so an https://host -> http://host hop
// keeps them and puts them on the wire in the clear.
var credentialHeaders = []string{
	"Authorization",
	"Proxy-Authorization",
	"Cookie",
	"Cookie2",
}

// isDowngrade reports whether target is plain HTTP while some earlier request
// in the chain was HTTPS. The whole chain matters, not just the previous hop:
// the standard client copies credential headers forward from the original
// request, so that is where the secret entered.
func isDowngrade(target *url.URL, via []*http.Request) bool {
	if !strings.EqualFold(target.Scheme, "http") {
		return false
	}
	for _, previous := range via {
		if previous != nil && previous.URL != nil && strings.EqualFold(previous.URL.Scheme, "https") {
			return true
		}
	}
	return false
}

func stripCredentials(header http.Header) {
	for _, name := range credentialHeaders {
		header.Del(name)
	}
}

// ValidatePublicIP reports whether ip is publicly routable, returning an error
// wrapping [ErrNonPublicAddress] when it is not. Loopback, private, link-local
// (including the 169.254.169.254 metadata address), multicast, unspecified,
// shared, documentation, benchmark, and reserved addresses are all rejected.
//
// Clients from [NewClient] apply this to every address they resolve, unless
// [WithAddressValidator] replaced it. It is exported for callers that want the
// same check before they enqueue a URL, or at another layer of their own.
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
