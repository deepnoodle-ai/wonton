package cache

import (
	"context"
	"errors"
)

var NotFound = errors.New("not found")

func IsNotFound(err error) bool {
	return errors.Is(err, NotFound)
}

type Cache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte) error
	Delete(ctx context.Context, key string) error
}
