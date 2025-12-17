package cache

import (
	"context"
	"sync"
)

// InMemoryCache implements the Cache interface using a simple in-memory map.
// It is safe for concurrent use and suitable for testing or small-scale crawling
// where persistence is not required.
//
// Data is lost when the process exits. For production use cases requiring
// persistence, use a disk-based or distributed cache implementation.
type InMemoryCache struct {
	data  map[string][]byte
	mutex sync.RWMutex
}

// NewInMemoryCache creates a new in-memory cache instance. The cache starts empty
// and grows as items are added. There is no automatic eviction or size limit.
func NewInMemoryCache() *InMemoryCache {
	return &InMemoryCache{
		data: make(map[string][]byte),
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

	m.data[key] = value
	return nil
}

func (m *InMemoryCache) Delete(ctx context.Context, key string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	delete(m.data, key)
	return nil
}
