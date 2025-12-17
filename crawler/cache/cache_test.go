package cache

import (
	"context"
	"errors"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestIsNotFound(t *testing.T) {
	assert.True(t, IsNotFound(NotFound))
	assert.False(t, IsNotFound(nil))
	assert.False(t, IsNotFound(errors.New("some other error")))

	// Test wrapped error
	wrapped := errors.Join(NotFound, errors.New("wrapped"))
	assert.True(t, IsNotFound(wrapped))
}

func TestInMemoryCache_SetAndGet(t *testing.T) {
	cache := NewInMemoryCache()
	ctx := context.Background()

	// Set a value
	err := cache.Set(ctx, "key1", []byte("value1"))
	assert.NoError(t, err)

	// Get the value
	value, err := cache.Get(ctx, "key1")
	assert.NoError(t, err)
	assert.Equal(t, []byte("value1"), value)
}

func TestInMemoryCache_GetNotFound(t *testing.T) {
	cache := NewInMemoryCache()
	ctx := context.Background()

	_, err := cache.Get(ctx, "nonexistent")
	assert.True(t, IsNotFound(err))
}

func TestInMemoryCache_Delete(t *testing.T) {
	cache := NewInMemoryCache()
	ctx := context.Background()

	// Set a value
	err := cache.Set(ctx, "key1", []byte("value1"))
	assert.NoError(t, err)

	// Delete the value
	err = cache.Delete(ctx, "key1")
	assert.NoError(t, err)

	// Verify it's gone
	_, err = cache.Get(ctx, "key1")
	assert.True(t, IsNotFound(err))
}

func TestInMemoryCache_DeleteNonexistent(t *testing.T) {
	cache := NewInMemoryCache()
	ctx := context.Background()

	// Deleting a nonexistent key should not error
	err := cache.Delete(ctx, "nonexistent")
	assert.NoError(t, err)
}

func TestInMemoryCache_Overwrite(t *testing.T) {
	cache := NewInMemoryCache()
	ctx := context.Background()

	// Set initial value
	err := cache.Set(ctx, "key1", []byte("value1"))
	assert.NoError(t, err)

	// Overwrite with new value
	err = cache.Set(ctx, "key1", []byte("value2"))
	assert.NoError(t, err)

	// Verify the new value
	value, err := cache.Get(ctx, "key1")
	assert.NoError(t, err)
	assert.Equal(t, []byte("value2"), value)
}

func TestInMemoryCache_MultipleKeys(t *testing.T) {
	cache := NewInMemoryCache()
	ctx := context.Background()

	// Set multiple values
	err := cache.Set(ctx, "key1", []byte("value1"))
	assert.NoError(t, err)
	err = cache.Set(ctx, "key2", []byte("value2"))
	assert.NoError(t, err)
	err = cache.Set(ctx, "key3", []byte("value3"))
	assert.NoError(t, err)

	// Verify all values
	value, err := cache.Get(ctx, "key1")
	assert.NoError(t, err)
	assert.Equal(t, []byte("value1"), value)

	value, err = cache.Get(ctx, "key2")
	assert.NoError(t, err)
	assert.Equal(t, []byte("value2"), value)

	value, err = cache.Get(ctx, "key3")
	assert.NoError(t, err)
	assert.Equal(t, []byte("value3"), value)
}

func TestInMemoryCache_EmptyValue(t *testing.T) {
	cache := NewInMemoryCache()
	ctx := context.Background()

	// Set an empty value
	err := cache.Set(ctx, "empty", []byte{})
	assert.NoError(t, err)

	// Get the empty value
	value, err := cache.Get(ctx, "empty")
	assert.NoError(t, err)
	assert.Equal(t, []byte{}, value)
}

func TestInMemoryCache_NilValue(t *testing.T) {
	cache := NewInMemoryCache()
	ctx := context.Background()

	// Set a nil value
	err := cache.Set(ctx, "nil", nil)
	assert.NoError(t, err)

	// Get the nil value
	value, err := cache.Get(ctx, "nil")
	assert.NoError(t, err)
	assert.Nil(t, value)
}
