package retry

import (
	"context"
	"errors"
	"math/rand"
	"testing"
	"time"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestDoSuccess(t *testing.T) {
	ctx := context.Background()

	result, err := Do(ctx, func() (string, error) {
		return "success", nil
	})

	assert.NoError(t, err)
	assert.Equal(t, "success", result)
}

func TestDoEventualSuccess(t *testing.T) {
	ctx := context.Background()
	attempts := 0

	result, err := Do(ctx, func() (int, error) {
		attempts++
		if attempts < 3 {
			return 0, errors.New("not yet")
		}
		return 42, nil
	}, WithMaxAttempts(5), WithBackoff(time.Millisecond, time.Millisecond))

	assert.NoError(t, err)
	assert.Equal(t, 42, result)
	assert.Equal(t, 3, attempts)
}

func TestDoMaxAttempts(t *testing.T) {
	ctx := context.Background()
	attempts := 0

	_, err := Do(ctx, func() (int, error) {
		attempts++
		return 0, errors.New("always fail")
	}, WithMaxAttempts(3), WithBackoff(time.Millisecond, time.Millisecond))

	assert.Error(t, err)
	assert.Equal(t, 3, attempts)

	var retryErr *Error
	assert.True(t, errors.As(err, &retryErr))
	assert.Equal(t, 3, retryErr.Attempts)
	assert.Len(t, retryErr.Errors, 3)
}

func TestDoContextCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := Do(ctx, func() (int, error) {
		return 0, errors.New("fail")
	}, WithMaxAttempts(100), WithBackoff(100*time.Millisecond, 100*time.Millisecond))

	elapsed := time.Since(start)
	assert.Error(t, err)
	assert.True(t, elapsed < 200*time.Millisecond, "should have cancelled quickly")
}

func TestDoRetryIf(t *testing.T) {
	ctx := context.Background()
	attempts := 0

	permanentErr := errors.New("permanent error")

	_, err := Do(ctx, func() (int, error) {
		attempts++
		return 0, permanentErr
	},
		WithMaxAttempts(5),
		WithBackoff(time.Millisecond, time.Millisecond),
		WithRetryIf(func(err error) bool {
			return err != permanentErr
		}),
	)

	assert.Error(t, err)
	assert.Equal(t, 1, attempts, "should not retry permanent errors")
}

func TestDoPermanentError(t *testing.T) {
	ctx := context.Background()
	attempts := 0

	_, err := Do(ctx, func() (int, error) {
		attempts++
		if attempts == 1 {
			return 0, errors.New("temporary")
		}
		return 0, MarkPermanent(errors.New("permanent"))
	},
		WithMaxAttempts(5),
		WithBackoff(time.Millisecond, time.Millisecond),
		WithRetryIf(SkipPermanent()),
	)

	assert.Error(t, err)
	assert.Equal(t, 2, attempts)
	// Use LastError() since Unwrap() now returns []error for Go 1.20+ multi-error support
	var retryErr *Error
	assert.ErrorAs(t, err, &retryErr)
	assert.True(t, IsPermanent(retryErr.LastError()))
}

func TestDoOnRetry(t *testing.T) {
	ctx := context.Background()
	var retries []int

	_, err := Do(ctx, func() (int, error) {
		return 0, errors.New("fail")
	},
		WithMaxAttempts(3),
		WithBackoff(time.Millisecond, time.Millisecond),
		WithOnRetry(func(attempt int, err error, delay time.Duration) {
			retries = append(retries, attempt)
		}),
	)

	assert.Error(t, err)
	assert.Equal(t, []int{1, 2}, retries, "OnRetry called for attempts 1 and 2")
}

func TestDoSimple(t *testing.T) {
	ctx := context.Background()
	attempts := 0

	err := DoSimple(ctx, func() error {
		attempts++
		if attempts < 2 {
			return errors.New("not yet")
		}
		return nil
	}, WithMaxAttempts(5), WithBackoff(time.Millisecond, time.Millisecond))

	assert.NoError(t, err)
	assert.Equal(t, 2, attempts)
}

func TestBackoff(t *testing.T) {
	cfg := Config{
		InitialBackoff:    100 * time.Millisecond,
		MaxBackoff:        time.Second,
		BackoffMultiplier: 2.0,
		Jitter:            0, // No jitter for deterministic testing
	}

	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{1, 100 * time.Millisecond},
		{2, 200 * time.Millisecond},
		{3, 400 * time.Millisecond},
		{4, 800 * time.Millisecond},
		{5, time.Second}, // Capped at max
		{6, time.Second},
	}

	for _, tt := range tests {
		result := Backoff(tt.attempt, cfg)
		assert.Equal(t, tt.expected, result, "Backoff(%d)", tt.attempt)
	}
}

func TestConstantBackoff(t *testing.T) {
	cfg := DefaultConfig()
	WithConstantBackoff(500 * time.Millisecond)(&cfg)

	assert.Equal(t, 500*time.Millisecond, cfg.InitialBackoff)
	assert.Equal(t, 500*time.Millisecond, cfg.MaxBackoff)
	assert.Equal(t, 1.0, cfg.BackoffMultiplier)
}

func TestErrorUnwrap(t *testing.T) {
	originalErr := errors.New("original")
	otherErr := errors.New("other")
	retryErr := &Error{
		Last:     originalErr,
		Attempts: 3,
		Errors:   []error{otherErr, originalErr},
	}

	// errors.Is traverses the Unwrap() []error slice
	assert.True(t, errors.Is(retryErr, originalErr))
	assert.True(t, errors.Is(retryErr, otherErr))
	assert.Equal(t, originalErr, retryErr.LastError())
}

func TestBackoffWithJitterRange(t *testing.T) {
	rand.Seed(1)
	cfg := Config{
		InitialBackoff:    time.Second,
		MaxBackoff:        time.Second,
		BackoffMultiplier: 1,
		Jitter:            0.5,
	}

	delay := Backoff(1, cfg)
	assert.True(t, delay >= 500*time.Millisecond && delay <= 1500*time.Millisecond,
		"delay should be within jitter range, got %s", delay)
}

func TestSkipPermanentFunction(t *testing.T) {
	predicate := SkipPermanent()
	assert.True(t, predicate(errors.New("temporary")))
	assert.False(t, predicate(MarkPermanent(errors.New("permanent"))))
}

func TestMarkPermanentNil(t *testing.T) {
	assert.Nil(t, MarkPermanent(nil))
}

func TestDoSimplePropagatesError(t *testing.T) {
	err := DoSimple(context.Background(), func() error {
		return errors.New("boom")
	}, WithMaxAttempts(2), WithBackoff(time.Millisecond, time.Millisecond))

	assert.Error(t, err)
}

func TestDoAggregatesErrors(t *testing.T) {
	attempts := 0

	_, err := Do(context.Background(), func() (int, error) {
		attempts++
		return 0, errors.New("fail")
	}, WithMaxAttempts(3), WithBackoff(time.Millisecond, time.Millisecond))

	assert.Error(t, err)

	var retryErr *Error
	assert.ErrorAs(t, err, &retryErr)
	assert.Equal(t, 3, retryErr.Attempts)
	assert.Len(t, retryErr.Errors, 3)
}

func TestDoZeroMaxAttemptsReliesOnContext(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	_, err := Do(ctx, func() (int, error) {
		return 0, errors.New("fail")
	}, WithMaxAttempts(0), WithBackoff(time.Millisecond, time.Millisecond))

	assert.Error(t, err)

	var retryErr *Error
	assert.ErrorAs(t, err, &retryErr)
	assert.True(t, errors.Is(retryErr.Last, context.DeadlineExceeded))
	assert.Greater(t, retryErr.Attempts, 0)
}

// mockTimer is a Timer that returns immediately for testing.
type mockTimer struct {
	delays []time.Duration
}

func (m *mockTimer) After(d time.Duration) <-chan time.Time {
	m.delays = append(m.delays, d)
	ch := make(chan time.Time, 1)
	ch <- time.Now()
	return ch
}

func TestWithTimer(t *testing.T) {
	ctx := context.Background()
	mock := &mockTimer{}

	_, err := Do(ctx, func() (int, error) {
		return 0, errors.New("fail")
	},
		WithMaxAttempts(4),
		WithBackoff(100*time.Millisecond, time.Second),
		WithJitter(0), // No jitter for deterministic testing
		WithTimer(mock),
	)

	assert.Error(t, err)
	// 4 attempts = 3 delays (after attempts 1, 2, 3)
	assert.Len(t, mock.delays, 3)
	// Exponential: 100ms, 200ms, 400ms
	assert.Equal(t, 100*time.Millisecond, mock.delays[0])
	assert.Equal(t, 200*time.Millisecond, mock.delays[1])
	assert.Equal(t, 400*time.Millisecond, mock.delays[2])
}

func TestLinearBackoff(t *testing.T) {
	cfg := &Config{
		InitialBackoff:  100 * time.Millisecond,
		LinearIncrement: 50 * time.Millisecond,
		MaxBackoff:      300 * time.Millisecond,
		Jitter:          0,
	}

	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{1, 100 * time.Millisecond},  // 100 + 0*50
		{2, 150 * time.Millisecond},  // 100 + 1*50
		{3, 200 * time.Millisecond},  // 100 + 2*50
		{4, 250 * time.Millisecond},  // 100 + 3*50
		{5, 300 * time.Millisecond},  // 100 + 4*50 = 300 (at cap)
		{6, 300 * time.Millisecond},  // capped
	}

	for _, tt := range tests {
		result := LinearBackoff(tt.attempt, cfg)
		assert.Equal(t, tt.expected, result, "LinearBackoff(%d)", tt.attempt)
	}
}

func TestWithLinearBackoff(t *testing.T) {
	ctx := context.Background()
	mock := &mockTimer{}

	_, err := Do(ctx, func() (int, error) {
		return 0, errors.New("fail")
	},
		WithMaxAttempts(4),
		WithLinearBackoff(100*time.Millisecond, 50*time.Millisecond, time.Second),
		WithJitter(0),
		WithTimer(mock),
	)

	assert.Error(t, err)
	assert.Len(t, mock.delays, 3)
	// Linear: 100ms, 150ms, 200ms
	assert.Equal(t, 100*time.Millisecond, mock.delays[0])
	assert.Equal(t, 150*time.Millisecond, mock.delays[1])
	assert.Equal(t, 200*time.Millisecond, mock.delays[2])
}

func TestFullJitterBackoff(t *testing.T) {
	cfg := &Config{
		InitialBackoff:    100 * time.Millisecond,
		MaxBackoff:        time.Second,
		BackoffMultiplier: 2.0,
	}

	// Full jitter should return values between 0 and ceiling
	for i := 0; i < 100; i++ {
		delay := FullJitterBackoff(1, cfg)
		assert.True(t, delay >= 0 && delay <= 100*time.Millisecond,
			"delay should be between 0 and 100ms, got %s", delay)
	}

	// At attempt 3, ceiling is 100ms * 2^2 = 400ms
	for i := 0; i < 100; i++ {
		delay := FullJitterBackoff(3, cfg)
		assert.True(t, delay >= 0 && delay <= 400*time.Millisecond,
			"delay should be between 0 and 400ms, got %s", delay)
	}
}

func TestWithFullJitter(t *testing.T) {
	ctx := context.Background()
	mock := &mockTimer{}

	_, err := Do(ctx, func() (int, error) {
		return 0, errors.New("fail")
	},
		WithMaxAttempts(3),
		WithBackoff(100*time.Millisecond, time.Second),
		WithFullJitter(),
		WithTimer(mock),
	)

	assert.Error(t, err)
	assert.Len(t, mock.delays, 2)
	// Full jitter: random between 0 and ceiling
	assert.True(t, mock.delays[0] >= 0 && mock.delays[0] <= 100*time.Millisecond)
	assert.True(t, mock.delays[1] >= 0 && mock.delays[1] <= 200*time.Millisecond)
}

func TestRetrier(t *testing.T) {
	ctx := context.Background()
	mock := &mockTimer{}

	retrier := NewRetrier(
		WithMaxAttempts(3),
		WithBackoff(10*time.Millisecond, 100*time.Millisecond),
		WithJitter(0),
		WithTimer(mock),
	)

	// First use
	attempts := 0
	err := retrier.Do(ctx, func() error {
		attempts++
		if attempts < 2 {
			return errors.New("fail")
		}
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, 2, attempts)

	// Reuse the same retrier
	mock.delays = nil
	attempts = 0
	err = retrier.Do(ctx, func() error {
		attempts++
		return errors.New("always fail")
	})
	assert.Error(t, err)
	assert.Equal(t, 3, attempts)
	assert.Len(t, mock.delays, 2) // 3 attempts = 2 delays
}

func TestRetrierConfig(t *testing.T) {
	retrier := NewRetrier(
		WithMaxAttempts(5),
		WithBackoff(200*time.Millisecond, 5*time.Second),
	)

	cfg := retrier.Config()
	assert.Equal(t, 5, cfg.MaxAttempts)
	assert.Equal(t, 200*time.Millisecond, cfg.InitialBackoff)
	assert.Equal(t, 5*time.Second, cfg.MaxBackoff)
}

func TestDelayFuncZeroAttempt(t *testing.T) {
	cfg := &Config{
		InitialBackoff:    100 * time.Millisecond,
		MaxBackoff:        time.Second,
		BackoffMultiplier: 2.0,
		LinearIncrement:   50 * time.Millisecond,
	}

	// All delay functions should return 0 for attempt <= 0
	assert.Equal(t, time.Duration(0), ExponentialBackoff(0, cfg))
	assert.Equal(t, time.Duration(0), ExponentialBackoff(-1, cfg))
	assert.Equal(t, time.Duration(0), LinearBackoff(0, cfg))
	assert.Equal(t, time.Duration(0), ConstantBackoff(0, cfg))
	assert.Equal(t, time.Duration(0), FullJitterBackoff(0, cfg))
}

func TestWithDelayFunc(t *testing.T) {
	ctx := context.Background()
	mock := &mockTimer{}

	// Custom delay function that always returns 42ms
	customDelay := func(attempt int, cfg *Config) time.Duration {
		return 42 * time.Millisecond
	}

	_, err := Do(ctx, func() (int, error) {
		return 0, errors.New("fail")
	},
		WithMaxAttempts(3),
		WithDelayFunc(customDelay),
		WithTimer(mock),
	)

	assert.Error(t, err)
	assert.Len(t, mock.delays, 2)
	assert.Equal(t, 42*time.Millisecond, mock.delays[0])
	assert.Equal(t, 42*time.Millisecond, mock.delays[1])
}
