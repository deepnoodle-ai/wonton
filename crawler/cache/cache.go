// Package cache provides cache interfaces for storing fetched web pages. It is
// used by the crawler to avoid redundant network requests and, with
// ResponseCache, to preserve validators and extracted links for efficient
// conditional revalidation.
//
// The cache interface is minimal by design, allowing various backend implementations
// such as in-memory caches, Redis, or disk-based storage.
package cache

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/deepnoodle-ai/wonton/fetch"
)

// ResponseSchemaVersion is the current schema used for typed cached
// responses. Callers should include it in cache keys so incompatible future
// entry formats do not collide with existing data.
const ResponseSchemaVersion = 1

// ResponseKey returns the schema-versioned key used by the crawler for a URL's
// typed response entry. Use it to seed, inspect, or delete ResponseCache data.
func ResponseKey(rawURL string) string {
	return "crawler:response:v" + strconv.Itoa(ResponseSchemaVersion) + ":" + rawURL
}

// NotFound is returned by Cache.Get when the requested key does not exist.
// Use IsNotFound to check for this specific error.
var NotFound = errors.New("not found")

// IsNotFound returns true if the error is a NotFound error from Cache.Get.
// This is the recommended way to check for missing keys.
func IsNotFound(err error) bool {
	return errors.Is(err, NotFound)
}

// Cache provides key-value storage for fetched web pages. Keys are typically
// URLs and values are the raw HTML or response content.
//
// Implementations must be safe for concurrent use.
type Cache interface {
	// Get retrieves the value for the given key. Returns NotFound error
	// if the key does not exist.
	Get(ctx context.Context, key string) ([]byte, error)

	// Set stores a value for the given key. If the key already exists,
	// the value is replaced.
	Set(ctx context.Context, key string, value []byte) error

	// Delete removes the value for the given key. No error is returned
	// if the key does not exist.
	Delete(ctx context.Context, key string) error

	// Close releases any resources held by the cache. This should be called
	// when the cache is no longer needed, especially for persistent caches
	// that need to flush data or close connections.
	// Returns nil if the cache has no resources to release.
	Close() error
}

// Entry preserves the HTTP metadata needed to reuse or conditionally
// revalidate a fetched page without parsing its HTML again.
type Entry struct {
	URL           string            `json:"url"`
	StatusCode    int               `json:"status_code"`
	Headers       map[string]string `json:"headers,omitempty"`
	Body          []byte            `json:"body,omitempty"`
	Links         []fetch.Link      `json:"links,omitempty"`
	ETag          string            `json:"etag,omitempty"`
	LastModified  string            `json:"last_modified,omitempty"`
	FetchedAt     time.Time         `json:"fetched_at"`
	SchemaVersion int               `json:"schema_version"`
}

// ResponseCache stores complete crawler responses. A Cache may implement this
// interface in addition to Cache; the crawler detects and prefers it while
// retaining support for HTML-only Cache implementations.
type ResponseCache interface {
	// GetEntry retrieves a typed response. It returns NotFound when key does
	// not exist.
	GetEntry(ctx context.Context, key string) (*Entry, error)

	// SetEntry stores a typed response, replacing any existing entry for key.
	SetEntry(ctx context.Context, key string, entry *Entry) error
}
