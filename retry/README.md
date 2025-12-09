# Retry helpers

The `retry` package wraps unreliable operations in configurable retry logic. It
handles exponential backoff, optional jitter, custom retry predicates, and hooks
for emitting metrics or logs before each attempt.

## Features

- Context-aware retries: abort immediately when the context is cancelled.
- Functional options for backoff strategy, max attempts, jitter, and callbacks.
- `retry.Error` exposes the final error plus the list of per-attempt failures.
- Works with any return type thanks to Go generics.

## Example

```go
package main

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/deepnoodle-ai/wonton/retry"
)

var errTemporary = errors.New("temporary")

func callRemoteAPI() (string, error) {
	// Replace with real logic.
	return "", errTemporary
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := retry.Do(ctx, func() (string, error) {
		response, err := callRemoteAPI()
		if err != nil {
			return "", err
		}
		return response, nil
	},
		retry.WithMaxAttempts(5),
		retry.WithBackoff(200*time.Millisecond, 3*time.Second),
		retry.WithJitter(0.25),
		retry.WithRetryIf(func(err error) bool {
			return errors.Is(err, errTemporary)
		}),
		retry.WithOnRetry(func(attempt int, err error, delay time.Duration) {
			log.Printf("attempt %d failed: %v (retrying in %s)", attempt, err, delay)
		}),
	)

	if err != nil {
		log.Fatalf("permanent failure: %v", err)
	}
	log.Printf("success: %s", result)
}
```

See `retry.WithConstantBackoff`, `retry.WithLinearBackoff`, and `retry.WithParser`
for additional dial-in points.
