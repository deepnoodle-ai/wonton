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
//
// Reusable retrier for high-frequency operations:
//
//	retrier := retry.NewRetrier(
//	    retry.WithMaxAttempts(5),
//	    retry.WithBackoff(time.Second, 30*time.Second),
//	)
//	for item := range items {
//	    err := retrier.Do(ctx, func() error { return process(item) })
//	}
package retry

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"time"
)

// Timer is the interface for time-based operations.
// Implement this interface to control timing in tests.
type Timer interface {
	After(time.Duration) <-chan time.Time
}

// defaultTimer uses the standard library's time.After.
type defaultTimer struct{}

func (defaultTimer) After(d time.Duration) <-chan time.Time {
	return time.After(d)
}

// DelayFunc calculates the delay before the next retry attempt.
// attempt starts at 1 for the delay after the first failure.
type DelayFunc func(attempt int, cfg *Config) time.Duration

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

	// LinearIncrement is the amount added to backoff each retry for linear backoff.
	LinearIncrement time.Duration

	// Jitter adds randomness to backoff (0.0 to 1.0).
	// 0.1 means +/- 10% random variation.
	Jitter float64

	// RetryIf determines if an error is retryable.
	// If nil, all errors are retried.
	RetryIf func(error) bool

	// OnRetry is called before each retry attempt.
	// Useful for logging or metrics.
	OnRetry func(attempt int, err error, delay time.Duration)

	// Timer controls time-based operations. Use WithTimer to inject a mock
	// for testing. If nil, uses time.After.
	Timer Timer

	// DelayFunc calculates the delay for each retry. If nil, uses
	// ExponentialBackoff.
	DelayFunc DelayFunc
}

// DefaultConfig returns a sensible default configuration.
func DefaultConfig() Config {
	return Config{
		MaxAttempts:       3,
		InitialBackoff:    100 * time.Millisecond,
		MaxBackoff:        30 * time.Second,
		BackoffMultiplier: 2.0,
		Jitter:            0.1,
		Timer:             defaultTimer{},
		DelayFunc:         ExponentialBackoff,
	}
}

// ExponentialBackoff calculates delay using exponential backoff with jitter.
// delay = InitialBackoff * (BackoffMultiplier ^ (attempt-1)), capped at MaxBackoff.
func ExponentialBackoff(attempt int, cfg *Config) time.Duration {
	if attempt <= 0 {
		return 0
	}
	backoff := float64(cfg.InitialBackoff) * math.Pow(cfg.BackoffMultiplier, float64(attempt-1))
	if time.Duration(backoff) > cfg.MaxBackoff {
		backoff = float64(cfg.MaxBackoff)
	}
	return applyJitter(time.Duration(backoff), cfg.Jitter)
}

// LinearBackoff calculates delay using linear backoff with jitter.
// delay = InitialBackoff + (attempt-1) * LinearIncrement, capped at MaxBackoff.
func LinearBackoff(attempt int, cfg *Config) time.Duration {
	if attempt <= 0 {
		return 0
	}
	backoff := cfg.InitialBackoff + time.Duration(attempt-1)*cfg.LinearIncrement
	if backoff > cfg.MaxBackoff {
		backoff = cfg.MaxBackoff
	}
	return applyJitter(backoff, cfg.Jitter)
}

// ConstantBackoff returns the same delay for all attempts (with jitter).
func ConstantBackoff(attempt int, cfg *Config) time.Duration {
	if attempt <= 0 {
		return 0
	}
	return applyJitter(cfg.InitialBackoff, cfg.Jitter)
}

// FullJitterBackoff calculates delay using exponential backoff with full jitter.
// The delay is a random value between 0 and the exponential backoff ceiling.
// This is the "Full Jitter" algorithm recommended by AWS for avoiding thundering herd.
func FullJitterBackoff(attempt int, cfg *Config) time.Duration {
	if attempt <= 0 {
		return 0
	}
	ceiling := float64(cfg.InitialBackoff) * math.Pow(cfg.BackoffMultiplier, float64(attempt-1))
	if ceiling > float64(cfg.MaxBackoff) {
		ceiling = float64(cfg.MaxBackoff)
	}
	return time.Duration(rand.Float64() * ceiling)
}

// applyJitter adds +/- jitter percentage to a duration.
func applyJitter(d time.Duration, jitter float64) time.Duration {
	if jitter <= 0 {
		return d
	}
	jitterRange := float64(d) * jitter
	jitterAmount := (rand.Float64()*2 - 1) * jitterRange
	return time.Duration(float64(d) + jitterAmount)
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
		c.DelayFunc = ConstantBackoff
	}
}

// WithLinearBackoff sets linear backoff where delay increases by increment each retry.
// delay = initial + (attempt-1) * increment, capped at max.
func WithLinearBackoff(initial, increment, max time.Duration) Option {
	return func(c *Config) {
		c.InitialBackoff = initial
		c.LinearIncrement = increment
		c.MaxBackoff = max
		c.DelayFunc = LinearBackoff
	}
}

// WithFullJitter uses exponential backoff with full jitter (AWS recommended).
// The delay is random between 0 and the exponential backoff ceiling.
func WithFullJitter() Option {
	return func(c *Config) {
		c.DelayFunc = FullJitterBackoff
	}
}

// WithTimer sets a custom timer for controlling time-based operations.
// This is primarily useful for testing to avoid real sleeps.
func WithTimer(t Timer) Option {
	return func(c *Config) {
		c.Timer = t
	}
}

// WithDelayFunc sets a custom delay function.
func WithDelayFunc(fn DelayFunc) Option {
	return func(c *Config) {
		c.DelayFunc = fn
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

// Unwrap returns all errors from all retry attempts.
// This implements the Go 1.20+ multi-error interface, allowing errors.Is
// and errors.As to check all wrapped errors, not just the last one.
//
// Note: errors.Unwrap(err) returns nil because it only calls Unwrap() error.
// Use errors.Is or errors.As to inspect wrapped errors, or call LastError()
// directly.
func (e *Error) Unwrap() []error {
	return e.Errors
}

// LastError returns the error from the final retry attempt.
// This is a convenience method for cases where you need direct access
// to the last error rather than using errors.Is/errors.As.
func (e *Error) LastError() error {
	return e.Last
}

// Do executes fn with retry logic, returning the result on success.
func Do[T any](ctx context.Context, fn func() (T, error), opts ...Option) (T, error) {
	cfg := DefaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	return doWithConfig(ctx, fn, &cfg)
}

func doWithConfig[T any](ctx context.Context, fn func() (T, error), cfg *Config) (T, error) {
	var zero T
	var errs []error
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

		// Calculate delay using the configured delay function
		delay := cfg.DelayFunc(attempt, cfg)

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
		case <-cfg.Timer.After(delay):
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
// Deprecated: Use ExponentialBackoff or other delay functions directly.
func Backoff(attempt int, cfg Config) time.Duration {
	return ExponentialBackoff(attempt, &cfg)
}

// Retrier is a reusable retry configuration.
// Create once with NewRetrier and reuse for multiple operations to minimize allocations.
type Retrier struct {
	config Config
}

// NewRetrier creates a new Retrier with the given options.
// The returned Retrier can be safely reused across multiple retry operations.
func NewRetrier(opts ...Option) *Retrier {
	cfg := DefaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	return &Retrier{config: cfg}
}

// Do executes the retryable function using this Retrier's configuration.
func (r *Retrier) Do(ctx context.Context, fn func() error) error {
	_, err := doWithConfig(ctx, func() (struct{}, error) {
		return struct{}{}, fn()
	}, &r.config)
	return err
}

// Config returns a copy of the Retrier's configuration.
func (r *Retrier) Config() Config {
	return r.config
}
