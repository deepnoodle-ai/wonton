// Package cache provides a simple cache interface for storing fetched web pages.
// It is used by the crawler to avoid redundant network requests.
//
// The cache interface is minimal by design, allowing various backend implementations
// such as in-memory caches, Redis, or disk-based storage.
package cache

import (
	"context"
	"errors"
)

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
