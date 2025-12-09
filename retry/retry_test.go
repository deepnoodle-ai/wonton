package retry

import (
	"context"
	"errors"
	"math/rand"
	"testing"
	"time"

	"github.com/deepnoodle-ai/wonton/require"
)

func TestDoSuccess(t *testing.T) {
	ctx := context.Background()

	result, err := Do(ctx, func() (string, error) {
		return "success", nil
	})

	require.NoError(t, err)
	require.Equal(t, "success", result)
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

	require.NoError(t, err)
	require.Equal(t, 42, result)
	require.Equal(t, 3, attempts)
}

func TestDoMaxAttempts(t *testing.T) {
	ctx := context.Background()
	attempts := 0

	_, err := Do(ctx, func() (int, error) {
		attempts++
		return 0, errors.New("always fail")
	}, WithMaxAttempts(3), WithBackoff(time.Millisecond, time.Millisecond))

	require.Error(t, err)
	require.Equal(t, 3, attempts)

	var retryErr *Error
	require.True(t, errors.As(err, &retryErr))
	require.Equal(t, 3, retryErr.Attempts)
	require.Len(t, retryErr.Errors, 3)
}

func TestDoContextCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := Do(ctx, func() (int, error) {
		return 0, errors.New("fail")
	}, WithMaxAttempts(100), WithBackoff(100*time.Millisecond, 100*time.Millisecond))

	elapsed := time.Since(start)
	require.Error(t, err)
	require.True(t, elapsed < 200*time.Millisecond, "should have cancelled quickly")
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

	require.Error(t, err)
	require.Equal(t, 1, attempts, "should not retry permanent errors")
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

	require.Error(t, err)
	require.Equal(t, 2, attempts)
	require.True(t, IsPermanent(errors.Unwrap(err)))
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

	require.Error(t, err)
	require.Equal(t, []int{1, 2}, retries, "OnRetry called for attempts 1 and 2")
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

	require.NoError(t, err)
	require.Equal(t, 2, attempts)
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
		require.Equal(t, tt.expected, result, "Backoff(%d)", tt.attempt)
	}
}

func TestConstantBackoff(t *testing.T) {
	cfg := DefaultConfig()
	WithConstantBackoff(500 * time.Millisecond)(&cfg)

	require.Equal(t, 500*time.Millisecond, cfg.InitialBackoff)
	require.Equal(t, 500*time.Millisecond, cfg.MaxBackoff)
	require.Equal(t, 1.0, cfg.BackoffMultiplier)
}

func TestErrorUnwrap(t *testing.T) {
	originalErr := errors.New("original")
	retryErr := &Error{Last: originalErr, Attempts: 3}

	require.True(t, errors.Is(retryErr, originalErr))
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
	require.True(t, delay >= 500*time.Millisecond && delay <= 1500*time.Millisecond,
		"delay should be within jitter range, got %s", delay)
}

func TestSkipPermanentFunction(t *testing.T) {
	predicate := SkipPermanent()
	require.True(t, predicate(errors.New("temporary")))
	require.False(t, predicate(MarkPermanent(errors.New("permanent"))))
}

func TestMarkPermanentNil(t *testing.T) {
	require.Nil(t, MarkPermanent(nil))
}

func TestDoSimplePropagatesError(t *testing.T) {
	err := DoSimple(context.Background(), func() error {
		return errors.New("boom")
	}, WithMaxAttempts(2), WithBackoff(time.Millisecond, time.Millisecond))

	require.Error(t, err)
}

func TestDoAggregatesErrors(t *testing.T) {
	attempts := 0

	_, err := Do(context.Background(), func() (int, error) {
		attempts++
		return 0, errors.New("fail")
	}, WithMaxAttempts(3), WithBackoff(time.Millisecond, time.Millisecond))

	require.Error(t, err)

	var retryErr *Error
	require.ErrorAs(t, err, &retryErr)
	require.Equal(t, 3, retryErr.Attempts)
	require.Len(t, retryErr.Errors, 3)
}

func TestDoZeroMaxAttemptsReliesOnContext(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	_, err := Do(ctx, func() (int, error) {
		return 0, errors.New("fail")
	}, WithMaxAttempts(0), WithBackoff(time.Millisecond, time.Millisecond))

	require.Error(t, err)

	var retryErr *Error
	require.ErrorAs(t, err, &retryErr)
	require.True(t, errors.Is(retryErr.Last, context.DeadlineExceeded))
	require.Greater(t, retryErr.Attempts, 0)
}
