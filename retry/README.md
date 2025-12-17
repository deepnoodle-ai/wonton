# retry

Configurable retry logic with exponential backoff, jitter, and context support. Implements best practices for retrying failed operations in distributed systems.

## Features

- Exponential, linear, constant, and full-jitter backoff strategies
- Context-aware cancellation
- Configurable retry conditions
- Permanent error marking to skip retries
- Callback hooks for logging and metrics
- Type-safe with Go generics
- Reusable Retrier instances for high-frequency operations

## Usage Examples

### Basic Retry

```go
package main

import (
    "context"
    "fmt"
    "github.com/deepnoodle-ai/wonton/retry"
)

func main() {
    ctx := context.Background()

    // Retry a function that returns a value
    result, err := retry.Do(ctx, func() (string, error) {
        data, err := fetchData()
        return data, err
    })

    if err != nil {
        fmt.Printf("Failed after retries: %v\n", err)
        return
    }

    fmt.Printf("Success: %s\n", result)
}
```

### Custom Options

```go
// Configure retry behavior
result, err := retry.Do(ctx, fetchData,
    retry.WithMaxAttempts(5),
    retry.WithBackoff(time.Second, 30*time.Second),
    retry.WithBackoffMultiplier(2.0),
    retry.WithJitter(0.1),
)
```

### Simple Error-Only Retry

```go
// For functions that only return errors
err := retry.DoSimple(ctx, func() error {
    return performOperation()
},
    retry.WithMaxAttempts(3),
    retry.WithBackoff(100*time.Millisecond, 5*time.Second),
)
```

### Conditional Retry

```go
// Only retry on specific errors
err := retry.DoSimple(ctx, fetchData,
    retry.WithRetryIf(func(err error) bool {
        // Retry on network errors, not on validation errors
        return errors.Is(err, ErrNetwork)
    }),
)
```

### Permanent Errors

```go
func validateAndFetch(id string) error {
    if id == "" {
        // Don't retry validation errors
        return retry.MarkPermanent(errors.New("invalid ID"))
    }
    return fetchByID(id)
}

err := retry.DoSimple(ctx, func() error {
    return validateAndFetch("123")
},
    retry.WithRetryIf(retry.SkipPermanent()),
)
```

### Logging Retries

```go
err := retry.DoSimple(ctx, operation,
    retry.WithOnRetry(func(attempt int, err error, delay time.Duration) {
        log.Printf("Retry %d after error: %v (waiting %v)", attempt, err, delay)
    }),
)
```

### Backoff Strategies

```go
// Exponential backoff (default)
// delay = initial * (multiplier ^ (attempt-1))
retry.Do(ctx, fn,
    retry.WithBackoff(100*time.Millisecond, 30*time.Second),
    retry.WithBackoffMultiplier(2.0),
)

// Linear backoff
// delay = initial + (attempt-1) * increment
retry.Do(ctx, fn,
    retry.WithLinearBackoff(100*time.Millisecond, 500*time.Millisecond, 10*time.Second),
)

// Constant backoff
// delay = constant for all attempts
retry.Do(ctx, fn,
    retry.WithConstantBackoff(time.Second),
)

// Full jitter (AWS recommended for thundering herd)
// delay = random(0, exponential ceiling)
retry.Do(ctx, fn,
    retry.WithBackoff(100*time.Millisecond, 30*time.Second),
    retry.WithFullJitter(),
)
```

### Reusable Retrier

```go
// Create once, reuse many times to minimize allocations
retrier := retry.NewRetrier(
    retry.WithMaxAttempts(5),
    retry.WithBackoff(time.Second, 30*time.Second),
    retry.WithOnRetry(logRetry),
)

// Use for multiple operations
for item := range items {
    err := retrier.Do(ctx, func() error {
        return process(item)
    })
    if err != nil {
        log.Printf("Failed to process %v: %v", item, err)
    }
}
```

### Infinite Retries

```go
// Retry forever (until context cancelled)
err := retry.DoSimple(ctx, operation,
    retry.WithMaxAttempts(0), // 0 = infinite
    retry.WithBackoff(time.Second, time.Minute),
)
```

### Error Inspection

```go
_, err := retry.Do(ctx, fetchData)
if err != nil {
    var retryErr *retry.Error
    if errors.As(err, &retryErr) {
        fmt.Printf("Failed after %d attempts\n", retryErr.Attempts)
        fmt.Printf("Last error: %v\n", retryErr.LastError())

        // Check all errors using Go 1.20+ multi-error support
        for _, e := range retryErr.Errors {
            fmt.Printf("  - %v\n", e)
        }
    }
}
```

## API Reference

### Functions

| Function | Description | Inputs | Outputs |
|----------|-------------|--------|---------|
| `Do` | Executes function with retry logic | `ctx context.Context, fn func() (T, error), opts ...Option` | `T, error` |
| `DoSimple` | Executes error-only function with retry logic | `ctx context.Context, fn func() error, opts ...Option` | `error` |
| `NewRetrier` | Creates reusable retrier with options | `opts ...Option` | `*Retrier` |
| `MarkPermanent` | Wraps error as non-retryable | `err error` | `error` |
| `IsPermanent` | Checks if error is marked permanent | `err error` | `bool` |
| `SkipPermanent` | Returns RetryIf function that skips permanent errors | None | `func(error) bool` |

### Configuration Options

| Option | Description | Default |
|--------|-------------|---------|
| `WithMaxAttempts(n int)` | Maximum retry attempts (0 = infinite) | 3 |
| `WithBackoff(initial, max time.Duration)` | Initial and max backoff durations | 100ms, 30s |
| `WithBackoffMultiplier(m float64)` | Exponential backoff multiplier | 2.0 |
| `WithJitter(j float64)` | Jitter factor (0.0 to 1.0) | 0.1 |
| `WithRetryIf(fn func(error) bool)` | Determines if error is retryable | nil (all retryable) |
| `WithOnRetry(fn func(int, error, time.Duration))` | Callback before each retry | nil |
| `WithConstantBackoff(d time.Duration)` | Use constant backoff | N/A |
| `WithLinearBackoff(initial, increment, max time.Duration)` | Use linear backoff | N/A |
| `WithFullJitter()` | Use full jitter backoff (AWS style) | N/A |
| `WithDelayFunc(fn DelayFunc)` | Custom delay calculation | N/A |
| `WithTimer(t Timer)` | Custom timer (for testing) | N/A |

### Types

#### Config

```go
type Config struct {
    MaxAttempts       int
    InitialBackoff    time.Duration
    MaxBackoff        time.Duration
    BackoffMultiplier float64
    LinearIncrement   time.Duration
    Jitter            float64
    RetryIf           func(error) bool
    OnRetry           func(attempt int, err error, delay time.Duration)
    Timer             Timer
    DelayFunc         DelayFunc
}
```

#### Error

Wraps the final error with retry context.

```go
type Error struct {
    Last     error    // Error from final attempt
    Attempts int      // Total attempts made
    Errors   []error  // All errors from all attempts
}
```

Methods:
- `LastError() error` - Returns the final error
- `Unwrap() []error` - Returns all errors (Go 1.20+ multi-error support)

#### Retrier

Reusable retry configuration.

```go
type Retrier struct {
    // private fields
}
```

Methods:
- `Do(ctx context.Context, fn func() error) error` - Execute with retry logic
- `Config() Config` - Returns copy of configuration

## Related Packages

- [fetch](../fetch) - HTTP fetching with built-in retry support
- [sse](../sse) - SSE client with automatic reconnection

## Design Notes

The package uses exponential backoff with jitter by default, which is recommended for most use cases. Jitter prevents the "thundering herd" problem where multiple clients retry simultaneously.

Context cancellation is checked before each retry attempt. If the context is cancelled, the retry loop stops immediately and returns the context error.

The generic `Do` function supports returning values on success, eliminating the need for closure variable capture. Use `DoSimple` for operations that only return errors.

Testing is simplified with the `WithTimer` option, which allows injection of a mock timer to avoid real sleep delays in tests.
