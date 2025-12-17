# Retry helpers

The `retry` package wraps unreliable operations in configurable retry logic. It
handles exponential backoff, optional jitter, custom retry predicates, and hooks
for emitting metrics or logs before each attempt.

## Features

- Context-aware retries: abort immediately when the context is cancelled.
- Functional options for backoff strategy, max attempts, jitter, and callbacks.
- Multiple backoff strategies: exponential, linear, constant, and full jitter.
- Reusable `Retrier` type for high-frequency operations (minimizes allocations).
- `Timer` interface for testing without real sleeps.
- `retry.Error` exposes all errors via Go 1.20+ multi-error unwrapping.
- Works with any return type thanks to Go generics.

## Basic Example

```go
result, err := retry.Do(ctx, func() (string, error) {
    return callRemoteAPI()
},
    retry.WithMaxAttempts(5),
    retry.WithBackoff(200*time.Millisecond, 3*time.Second),
    retry.WithJitter(0.25),
)
```

## Reusable Retrier

For high-frequency operations, create a `Retrier` once and reuse it:

```go
retrier := retry.NewRetrier(
    retry.WithMaxAttempts(5),
    retry.WithBackoff(100*time.Millisecond, 5*time.Second),
)

for item := range items {
    err := retrier.Do(ctx, func() error {
        return process(item)
    })
}
```

## Backoff Strategies

```go
// Exponential backoff (default): 100ms, 200ms, 400ms, ...
retry.WithBackoff(100*time.Millisecond, 30*time.Second)

// Linear backoff: 100ms, 150ms, 200ms, 250ms, ...
retry.WithLinearBackoff(100*time.Millisecond, 50*time.Millisecond, 1*time.Second)

// Constant backoff: 500ms, 500ms, 500ms, ...
retry.WithConstantBackoff(500*time.Millisecond)

// Full jitter (AWS recommended): random between 0 and exponential ceiling
retry.WithFullJitter()
```

## Testing with Mock Timer

```go
type mockTimer struct {
    delays []time.Duration
}

func (m *mockTimer) After(d time.Duration) <-chan time.Time {
    m.delays = append(m.delays, d)
    ch := make(chan time.Time, 1)
    ch <- time.Now()
    return ch
}

mock := &mockTimer{}
_, err := retry.Do(ctx, fn, retry.WithTimer(mock))
// mock.delays contains all the delays that would have been used
```

## Error Handling

The returned error implements Go 1.20+ multi-error unwrapping:

```go
var retryErr *retry.Error
if errors.As(err, &retryErr) {
    fmt.Printf("Failed after %d attempts\n", retryErr.Attempts)
    fmt.Printf("Last error: %v\n", retryErr.LastError())
}

// errors.Is checks ALL wrapped errors
if errors.Is(err, someSpecificError) {
    // Any attempt returned someSpecificError
}
```
