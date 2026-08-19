# httpguard

An HTTP client for requests to URLs your program did not choose — a webhook target a user configured, a link submitted through an API, a redirect a crawler decided to follow.

## Summary

Such a URL is an SSRF vector. It can name a hostname that resolves to `127.0.0.1`, to the cloud metadata endpoint at `169.254.169.254`, or to a service reachable only from inside your network. `NewClient` returns a standard `*http.Client` that refuses to connect to any of those.

Four properties make that hold:

- **Every resolved address is validated.** A hostname that mixes a public and a private address is rejected outright rather than filtered down to the public one.
- **The dial goes to a validated address, not the hostname.** A name cannot resolve to a public address for the check and a private one for the connection (DNS rebinding). TLS is unaffected: the handshake still uses the hostname for SNI and certificate verification, because only the dial address is pinned.
- **Redirects are refused by default.** `WithMaxRedirects` enables a bounded number of them; each hop is HTTPS-only unless `WithHTTPRedirects` says otherwise, and is address-validated at dial time like any other request.
- **Ambient proxies are ignored.** `HTTP_PROXY` and friends cannot relay the request somewhere the guard would have refused.

The client sets no total timeout. Bound the whole request with its context; the options here bound only connection setup, which a context deadline alone does not distinguish.

## Usage Examples

### Fetching a User-Supplied URL

```go
package main

import (
    "context"
    "errors"
    "fmt"
    "io"
    "net/http"
    "time"

    "github.com/deepnoodle-ai/wonton/httpguard"
)

var client = httpguard.NewClient()

func fetch(ctx context.Context, url string) ([]byte, error) {
    ctx, cancel := context.WithTimeout(ctx, 15*time.Second) // total request budget
    defer cancel()

    req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
    if err != nil {
        return nil, err
    }
    resp, err := client.Do(req)
    if err != nil {
        if errors.Is(err, httpguard.ErrNonPublicAddress) {
            return nil, fmt.Errorf("%q points inside the network", url)
        }
        return nil, err
    }
    defer resp.Body.Close()
    return io.ReadAll(io.LimitReader(resp.Body, 1<<20))
}
```

Build one client and share it — an `*http.Client` is safe for concurrent use, and reusing it reuses connections.

### Tuning Setup Timeouts

```go
client := httpguard.NewClient(
    httpguard.WithDNSTimeout(2*time.Second),
    httpguard.WithConnectTimeout(5*time.Second),
    httpguard.WithTLSHandshakeTimeout(5*time.Second),
)
```

The connect timeout covers the whole connect phase. When a hostname resolves
to several addresses, they are tried in order until one connects — so a host
with an AAAA record still works on an IPv4-only network — and the timeout
bounds all the attempts together, not each one.

### Allowing Redirects

```go
// Follow up to 3 redirects. Every hop must be an HTTPS URL with no userinfo,
// and is address-validated on dial like any other request.
client := httpguard.NewClient(httpguard.WithMaxRedirects(3))

// Allow plain HTTP hops too. Every hop is still address-validated, so this
// does not widen where the request can reach — but a target can downgrade the
// connection to plaintext, so don't use it for credentialed requests.
client = httpguard.NewClient(httpguard.WithMaxRedirects(3), httpguard.WithHTTPRedirects())
```

Leave redirects off unless you need them. A redirect is an attacker-controlled
second request, and it is the usual way an SSRF filter gets bypassed.

### Validating an Address Directly

`ValidatePublicIP` is the same check the client applies, exported for callers
that want to reject a target earlier — when a webhook URL is saved, say,
rather than when it first fires:

```go
for _, ip := range mustResolve(host) {
    if err := httpguard.ValidatePublicIP(ip); err != nil {
        return fmt.Errorf("webhook host %q is not publicly reachable: %w", host, err)
    }
}
```

Treat that as an early warning, not a substitute: DNS can change between the
check and the request, which is exactly why the client validates again at dial
time.

### Handling Errors

```go
switch {
case errors.Is(err, httpguard.ErrNonPublicAddress):
    // The target resolved to a private, loopback, link-local, or reserved address.
case errors.Is(err, httpguard.ErrRedirectRefused):
    // A redirect was disabled, over the limit, or not a plain HTTPS URL.
}
```

Both sentinels arrive wrapped — in an `*url.Error` from `client.Do`, and with
the offending address or URL for diagnostics — so compare with `errors.Is`
rather than `==`.

## API Reference

### Functions

| Function                | Description                                                   | Returns        |
| ----------------------- | ------------------------------------------------------------- | -------------- |
| `NewClient(opts...)`    | An `*http.Client` that connects only to public addresses      | `*http.Client` |
| `ValidatePublicIP(ip)`  | The client's address check, for use on its own                | `error`        |

### Options

| Option                          | Default   | Description                                            |
| ------------------------------- | --------- | ------------------------------------------------------ |
| `WithDNSTimeout(d)`             | 5s        | Bounds hostname resolution                             |
| `WithConnectTimeout(d)`         | 10s       | Bounds the connect phase, across every address tried   |
| `WithTLSHandshakeTimeout(d)`    | 10s       | Bounds the TLS handshake                               |
| `WithResponseHeaderTimeout(d)`  | none      | Bounds how long the server may take to send headers    |
| `WithMaxRedirects(n)`           | 0         | Allows up to n guarded HTTPS redirects                 |
| `WithHTTPRedirects()`           | off       | Also allows plain HTTP hops (still address-validated)  |
| `WithResolver(lookup)`          | system    | Replaces the resolver; results are validated the same  |
| `WithDialContext(dial)`         | `net.Dialer` | Replaces the dialer; called only with a validated IP |

Non-positive durations fall back to the defaults.

### Errors

| Error                 | Meaning                                                                        |
| --------------------- | ------------------------------------------------------------------------------ |
| `ErrNonPublicAddress` | The target resolved to, or named, an address that is not publicly routable      |
| `ErrRedirectRefused`  | A redirect was refused: disabled, over the limit, or not an allowed target       |

### Rejected Address Space

Loopback, private (RFC 1918 and IPv6 unique-local), link-local — including the
`169.254.169.254` metadata address — multicast, unspecified, and broadcast
addresses, plus shared address space (`100.64.0.0/10`), documentation and
benchmark ranges, IPv4/IPv6 transition ranges (`64:ff9b::/96`, `2001::/32`,
`2002::/16`), and reserved space (`240.0.0.0/4`). IPv4-mapped IPv6 addresses
are checked as IPv4, so `::ffff:127.0.0.1` is rejected as loopback.

## What This Does Not Do

The guard is a network-boundary control, not a whole SSRF policy. It does not:

- **Limit the response.** Wrap the body in an `io.LimitReader`.
- **Restrict the scheme or port** of the request you make. Validate the URL
  before you hand it over if you only want `https` on 443.
- **Enforce an allowlist.** "Publicly routable" is a large set. If you can name
  the hosts you expect, check them too.
- **Bound the total request.** Use a context deadline.

## Related Packages

- **[fetch](../fetch/)** — HTTP page fetching and downloads, with its own client
- **[crawler](../crawler/)** — concurrent crawling, where redirect targets come from the pages themselves
- **[retry](../retry/)** — backoff for the transient failures a guarded request still hits
