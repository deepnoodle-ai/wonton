package cache

import (
	"context"
	"sync"
)

// InMemoryCache implements the cache.Cache interface for testing
type InMemoryCache struct {
	data  map[string][]byte
	mutex sync.RWMutex
}

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
