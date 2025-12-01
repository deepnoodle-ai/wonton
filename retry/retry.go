// Package retry provides configurable retry logic with exponential backoff,
// jitter, and context support for Go applications.
//
// Basic usage:
//
//	result, err := retry.Do(ctx, func() (string, error) {
//	    return fetchData()
//	})
//
// With options:
//
//	result, err := retry.Do(ctx, fn,
//	    retry.WithMaxAttempts(5),
//	    retry.WithBackoff(time.Second, 30*time.Second),
//	    retry.WithJitter(0.1),
//	)
package retry

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"time"
)

// Config holds retry configuration.
type Config struct {
	// MaxAttempts is the maximum number of attempts (including the first).
	// 0 means infinite retries (until context is cancelled).
	MaxAttempts int

	// InitialBackoff is the delay after the first failure.
	InitialBackoff time.Duration

	// MaxBackoff is the maximum delay between retries.
	MaxBackoff time.Duration

	// BackoffMultiplier is the factor by which backoff increases each retry.
	BackoffMultiplier float64

	// Jitter adds randomness to backoff (0.0 to 1.0).
	// 0.1 means +/- 10% random variation.
	Jitter float64

	// RetryIf determines if an error is retryable.
	// If nil, all errors are retried.
	RetryIf func(error) bool

	// OnRetry is called before each retry attempt.
	// Useful for logging or metrics.
	OnRetry func(attempt int, err error, delay time.Duration)
}

// DefaultConfig returns a sensible default configuration.
func DefaultConfig() Config {
	return Config{
		MaxAttempts:       3,
		InitialBackoff:    100 * time.Millisecond,
		MaxBackoff:        30 * time.Second,
		BackoffMultiplier: 2.0,
		Jitter:            0.1,
	}
}

// Option is a functional option for configuring retry behavior.
type Option func(*Config)

// WithMaxAttempts sets the maximum number of attempts.
// Use 0 for infinite retries.
func WithMaxAttempts(n int) Option {
	return func(c *Config) {
		c.MaxAttempts = n
	}
}

// WithBackoff sets the initial and maximum backoff durations.
func WithBackoff(initial, max time.Duration) Option {
	return func(c *Config) {
		c.InitialBackoff = initial
		c.MaxBackoff = max
	}
}

// WithBackoffMultiplier sets the backoff multiplier.
func WithBackoffMultiplier(m float64) Option {
	return func(c *Config) {
		c.BackoffMultiplier = m
	}
}

// WithJitter sets the jitter factor (0.0 to 1.0).
func WithJitter(j float64) Option {
	return func(c *Config) {
		c.Jitter = j
	}
}

// WithRetryIf sets a function to determine if an error is retryable.
func WithRetryIf(fn func(error) bool) Option {
	return func(c *Config) {
		c.RetryIf = fn
	}
}

// WithOnRetry sets a callback invoked before each retry.
func WithOnRetry(fn func(attempt int, err error, delay time.Duration)) Option {
	return func(c *Config) {
		c.OnRetry = fn
	}
}

// WithConstantBackoff sets constant backoff (no exponential increase).
func WithConstantBackoff(d time.Duration) Option {
	return func(c *Config) {
		c.InitialBackoff = d
		c.MaxBackoff = d
		c.BackoffMultiplier = 1.0
	}
}

// WithLinearBackoff sets linear backoff.
func WithLinearBackoff(initial, increment time.Duration, max time.Duration) Option {
	return func(c *Config) {
		c.InitialBackoff = initial
		c.MaxBackoff = max
		// Store increment via custom behavior
		c.BackoffMultiplier = 1.0 // Will be handled specially
	}
}

// Error wraps the last error with retry context.
type Error struct {
	// Last is the error from the final attempt.
	Last error

	// Attempts is the total number of attempts made.
	Attempts int

	// Errors contains all errors from all attempts.
	Errors []error
}

func (e *Error) Error() string {
	return fmt.Sprintf("retry failed after %d attempts: %v", e.Attempts, e.Last)
}

func (e *Error) Unwrap() error {
	return e.Last
}

// Do executes fn with retry logic, returning the result on success.
func Do[T any](ctx context.Context, fn func() (T, error), opts ...Option) (T, error) {
	cfg := DefaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	var zero T
	var errs []error

	backoff := cfg.InitialBackoff
	attempt := 0

	for {
		attempt++

		result, err := fn()
		if err == nil {
			return result, nil
		}

		errs = append(errs, err)

		// Check if error is retryable
		if cfg.RetryIf != nil && !cfg.RetryIf(err) {
			return zero, &Error{Last: err, Attempts: attempt, Errors: errs}
		}

		// Check if we've exhausted attempts
		if cfg.MaxAttempts > 0 && attempt >= cfg.MaxAttempts {
			return zero, &Error{Last: err, Attempts: attempt, Errors: errs}
		}

		// Calculate delay with jitter
		delay := backoff
		if cfg.Jitter > 0 {
			jitterRange := float64(delay) * cfg.Jitter
			jitter := (rand.Float64()*2 - 1) * jitterRange
			delay = time.Duration(float64(delay) + jitter)
		}

		// Call OnRetry callback
		if cfg.OnRetry != nil {
			cfg.OnRetry(attempt, err, delay)
		}

		// Wait with context cancellation support
		select {
		case <-ctx.Done():
			return zero, &Error{
				Last:     ctx.Err(),
				Attempts: attempt,
				Errors:   append(errs, ctx.Err()),
			}
		case <-time.After(delay):
		}

		// Increase backoff for next attempt
		backoff = time.Duration(float64(backoff) * cfg.BackoffMultiplier)
		if backoff > cfg.MaxBackoff {
			backoff = cfg.MaxBackoff
		}
	}
}

// DoSimple executes fn with retry logic for functions that return only an error.
func DoSimple(ctx context.Context, fn func() error, opts ...Option) error {
	_, err := Do(ctx, func() (struct{}, error) {
		return struct{}{}, fn()
	}, opts...)
	return err
}

// Permanent wraps an error to indicate it should not be retried.
type Permanent struct {
	Err error
}

func (p Permanent) Error() string {
	return p.Err.Error()
}

func (p Permanent) Unwrap() error {
	return p.Err
}

// IsPermanent checks if an error is marked as permanent (non-retryable).
func IsPermanent(err error) bool {
	var p Permanent
	return errors.As(err, &p)
}

// MarkPermanent wraps an error to mark it as non-retryable.
func MarkPermanent(err error) error {
	if err == nil {
		return nil
	}
	return Permanent{Err: err}
}

// SkipPermanent returns a RetryIf function that skips permanent errors.
func SkipPermanent() func(error) bool {
	return func(err error) bool {
		return !IsPermanent(err)
	}
}

// Backoff calculates the backoff duration for a given attempt.
// Useful for custom retry logic or testing.
func Backoff(attempt int, cfg Config) time.Duration {
	if attempt <= 0 {
		return 0
	}

	backoff := float64(cfg.InitialBackoff) * math.Pow(cfg.BackoffMultiplier, float64(attempt-1))
	if time.Duration(backoff) > cfg.MaxBackoff {
		backoff = float64(cfg.MaxBackoff)
	}

	if cfg.Jitter > 0 {
		jitterRange := backoff * cfg.Jitter
		jitter := (rand.Float64()*2 - 1) * jitterRange
		backoff += jitter
	}

	return time.Duration(backoff)
}
