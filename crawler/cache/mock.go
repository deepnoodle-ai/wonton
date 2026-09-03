package cache

import (
	"context"
	"sync"

	"github.com/deepnoodle-ai/wonton/fetch"
)

// InMemoryCache implements the Cache interface using a simple in-memory map.
// It is safe for concurrent use and suitable for testing or small-scale crawling
// where persistence is not required.
//
// Data is lost when the process exits. For production use cases requiring
// persistence, use a disk-based or distributed cache implementation.
type InMemoryCache struct {
	data    map[string][]byte
	entries map[string]*Entry
	mutex   sync.RWMutex
}

var _ Cache = (*InMemoryCache)(nil)
var _ ResponseCache = (*InMemoryCache)(nil)

// NewInMemoryCache creates a new in-memory cache instance. The cache starts empty
// and grows as items are added. There is no automatic eviction or size limit.
func NewInMemoryCache() *InMemoryCache {
	return &InMemoryCache{
		data:    make(map[string][]byte),
		entries: make(map[string]*Entry),
	}
}

func (m *InMemoryCache) Get(ctx context.Context, key string) ([]byte, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if value, exists := m.data[key]; exists {
		return value, nil
	}
	return nil, NotFound
}

func (m *InMemoryCache) Set(ctx context.Context, key string, value []byte) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if value == nil {
		m.data[key] = nil
	} else {
		m.data[key] = append([]byte{}, value...)
	}
	return nil
}

// GetEntry retrieves a defensive copy of a typed response entry.
func (m *InMemoryCache) GetEntry(ctx context.Context, key string) (*Entry, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	entry, exists := m.entries[key]
	if !exists {
		return nil, NotFound
	}
	return cloneEntry(entry), nil
}

// SetEntry stores a defensive copy of a typed response entry.
func (m *InMemoryCache) SetEntry(ctx context.Context, key string, entry *Entry) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.entries == nil {
		m.entries = make(map[string]*Entry)
	}
	m.entries[key] = cloneEntry(entry)
	return nil
}

func (m *InMemoryCache) Delete(ctx context.Context, key string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	delete(m.data, key)
	delete(m.entries, key)
	return nil
}

// Close releases resources held by the cache. For InMemoryCache, this clears
// the map and returns nil since there are no external resources to release.
func (m *InMemoryCache) Close() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Clear the map to free memory
	m.data = make(map[string][]byte)
	m.entries = make(map[string]*Entry)
	return nil
}

func cloneEntry(entry *Entry) *Entry {
	if entry == nil {
		return nil
	}
	copy := *entry
	copy.Body = append([]byte(nil), entry.Body...)
	copy.Links = append([]fetch.Link(nil), entry.Links...)
	if entry.Headers != nil {
		copy.Headers = make(map[string]string, len(entry.Headers))
		for key, value := range entry.Headers {
			copy.Headers[key] = value
		}
	}
	return &copy
}
