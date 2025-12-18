// Package retry provides configurable retry logic with exponential backoff,
// jitter, and context support for Go applications.
//
// This package implements best practices for retrying failed operations in
// distributed systems, including multiple backoff strategies, context-aware
// cancellation, and permanent error handling.
//
// # Basic Usage
//
// The Do function retries operations that return a value:
//
//	result, err := retry.Do(ctx, func() (string, error) {
//	    return fetchData()
//	})
//
// For operations that only return errors, use DoSimple:
//
//	err := retry.DoSimple(ctx, func() error {
//	    return performOperation()
//	})
//
// # Configuration
//
// Customize retry behavior with functional options:
//
//	result, err := retry.Do(ctx, fn,
//	    retry.WithMaxAttempts(5),
//	    retry.WithBackoff(time.Second, 30*time.Second),
//	    retry.WithJitter(0.1),
//	)
//
// # Backoff Strategies
//
// The package supports multiple backoff strategies:
//
//   - Exponential (default): delay = initial * (multiplier ^ (attempt-1))
//   - Linear: delay = initial + (attempt-1) * increment
//   - Constant: same delay for all attempts
//   - Full Jitter: random value between 0 and exponential ceiling (AWS recommended)
//
// # Permanent Errors
//
// Mark errors as non-retryable to exit early:
//
//	if validationFailed {
//	    return retry.MarkPermanent(errors.New("invalid input"))
//	}
//
// Use SkipPermanent() with WithRetryIf to respect permanent error markers:
//
//	retry.Do(ctx, fn, retry.WithRetryIf(retry.SkipPermanent()))
//
// # Reusable Retrier
//
// For high-frequency operations, create a reusable Retrier to minimize allocations:
//
//	retrier := retry.NewRetrier(
//	    retry.WithMaxAttempts(5),
//	    retry.WithBackoff(time.Second, 30*time.Second),
//	)
//	for item := range items {
//	    err := retrier.Do(ctx, func() error { return process(item) })
//	}
//
// # Context Cancellation
//
// All retry operations respect context cancellation. If the context is cancelled,
// the retry loop stops immediately and returns the context error:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
//	defer cancel()
//	result, err := retry.Do(ctx, fn)
package retry

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"time"
)

// Timer is the interface for time-based operations used during retry delays.
//
// The default implementation uses time.After from the standard library.
// Implement this interface to control timing in tests, allowing tests to
// run without actual sleep delays.
//
// Example test implementation:
//
//	type mockTimer struct {
//	    delays []time.Duration
//	}
//
//	func (m *mockTimer) After(d time.Duration) <-chan time.Time {
//	    m.delays = append(m.delays, d)
//	    ch := make(chan time.Time, 1)
//	    ch <- time.Now()
//	    return ch
//	}
type Timer interface {
	After(time.Duration) <-chan time.Time
}

// defaultTimer uses the standard library's time.After.
type defaultTimer struct{}

func (defaultTimer) After(d time.Duration) <-chan time.Time {
	return time.After(d)
}

// DelayFunc calculates the delay before the next retry attempt.
//
// The attempt parameter starts at 1 for the delay after the first failure.
// The function receives the full Config to allow access to all configuration
// parameters when calculating delays.
//
// Built-in delay functions include ExponentialBackoff, LinearBackoff,
// ConstantBackoff, and FullJitterBackoff.
type DelayFunc func(attempt int, cfg *Config) time.Duration

// Config holds retry configuration parameters.
//
// Create a Config using DefaultConfig() and modify it with functional options,
// or construct it directly for advanced use cases. When used with Do or DoSimple,
// options are applied to a default configuration automatically.
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
//
// Default values:
//   - MaxAttempts: 3
//   - InitialBackoff: 100ms
//   - MaxBackoff: 30s
//   - BackoffMultiplier: 2.0
//   - Jitter: 0.1 (10% randomness)
//   - DelayFunc: ExponentialBackoff
//
// These defaults work well for most use cases and follow best practices
// for distributed systems.
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
//
// Formula: delay = InitialBackoff * (BackoffMultiplier ^ (attempt-1)), capped at MaxBackoff.
//
// This is the default delay function and is recommended for most use cases.
// It prevents retry storms by exponentially increasing delays between attempts.
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
//
// Formula: delay = InitialBackoff + (attempt-1) * LinearIncrement, capped at MaxBackoff.
//
// Use this when you want predictable, steadily increasing delays between retries
// rather than exponential growth.
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
//
// The delay equals InitialBackoff for every retry attempt. Use this when you
// want a fixed delay between retries, such as polling an endpoint at regular
// intervals.
func ConstantBackoff(attempt int, cfg *Config) time.Duration {
	if attempt <= 0 {
		return 0
	}
	return applyJitter(cfg.InitialBackoff, cfg.Jitter)
}

// FullJitterBackoff calculates delay using exponential backoff with full jitter.
//
// The delay is a random value between 0 and the exponential backoff ceiling.
// This is the "Full Jitter" algorithm recommended by AWS for avoiding the
// thundering herd problem in distributed systems.
//
// Use this when you have many clients that might retry simultaneously and need
// to spread out retry attempts to avoid overwhelming a recovering service.
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

// WithMaxAttempts sets the maximum number of attempts (including the initial attempt).
//
// Set to 0 for infinite retries that will only stop when the context is cancelled.
// The default is 3 attempts.
func WithMaxAttempts(n int) Option {
	return func(c *Config) {
		c.MaxAttempts = n
	}
}

// WithBackoff sets the initial and maximum backoff durations.
//
// The initial duration is used for the first retry delay. Subsequent delays
// are calculated by the delay function (exponential by default) and capped at max.
func WithBackoff(initial, max time.Duration) Option {
	return func(c *Config) {
		c.InitialBackoff = initial
		c.MaxBackoff = max
	}
}

// WithBackoffMultiplier sets the backoff multiplier for exponential backoff.
//
// Each retry delay is multiplied by this factor (default 2.0).
// For example, with initial=100ms and multiplier=2.0, delays will be
// 100ms, 200ms, 400ms, 800ms, etc.
func WithBackoffMultiplier(m float64) Option {
	return func(c *Config) {
		c.BackoffMultiplier = m
	}
}

// WithJitter sets the jitter factor (0.0 to 1.0).
//
// Jitter adds randomness to retry delays to prevent thundering herd problems.
// A value of 0.1 means +/- 10% random variation. Set to 0 to disable jitter.
// The default is 0.1.
func WithJitter(j float64) Option {
	return func(c *Config) {
		c.Jitter = j
	}
}

// WithRetryIf sets a function to determine if an error is retryable.
//
// The function receives each error and returns true to retry or false to stop.
// If not set, all errors are retried. Use this to skip retrying specific error
// types like validation errors or authentication failures.
func WithRetryIf(fn func(error) bool) Option {
	return func(c *Config) {
		c.RetryIf = fn
	}
}

// WithOnRetry sets a callback invoked before each retry attempt.
//
// The callback receives the attempt number (starting at 1), the error that
// occurred, and the delay before the next retry. Use this for logging,
// metrics, or other side effects.
func WithOnRetry(fn func(attempt int, err error, delay time.Duration)) Option {
	return func(c *Config) {
		c.OnRetry = fn
	}
}

// WithConstantBackoff sets constant backoff (no exponential increase).
//
// This sets InitialBackoff, MaxBackoff, and BackoffMultiplier to produce a
// constant delay. Use this for simple polling or when you want the same
// delay between all retry attempts.
func WithConstantBackoff(d time.Duration) Option {
	return func(c *Config) {
		c.InitialBackoff = d
		c.MaxBackoff = d
		c.BackoffMultiplier = 1.0
		c.DelayFunc = ConstantBackoff
	}
}

// WithLinearBackoff sets linear backoff where delay increases by a fixed increment.
//
// Formula: delay = initial + (attempt-1) * increment, capped at max.
//
// This produces predictable, steadily increasing delays. For example, with
// initial=100ms and increment=50ms: 100ms, 150ms, 200ms, 250ms, etc.
func WithLinearBackoff(initial, increment, max time.Duration) Option {
	return func(c *Config) {
		c.InitialBackoff = initial
		c.LinearIncrement = increment
		c.MaxBackoff = max
		c.DelayFunc = LinearBackoff
	}
}

// WithFullJitter uses exponential backoff with full jitter (AWS recommended).
//
// The delay is a random value between 0 and the exponential backoff ceiling.
// This is more aggressive than the default jitter and helps prevent thundering
// herd problems in distributed systems with many clients.
func WithFullJitter() Option {
	return func(c *Config) {
		c.DelayFunc = FullJitterBackoff
	}
}

// WithTimer sets a custom timer for controlling time-based operations.
//
// This is primarily useful for testing to avoid real sleep delays, allowing
// tests to run instantly. In production, the default time.After is used.
func WithTimer(t Timer) Option {
	return func(c *Config) {
		c.Timer = t
	}
}

// WithDelayFunc sets a custom delay calculation function.
//
// Use this to implement custom backoff strategies beyond the built-in
// exponential, linear, constant, and full jitter options.
func WithDelayFunc(fn DelayFunc) Option {
	return func(c *Config) {
		c.DelayFunc = fn
	}
}

// Error wraps the final error with retry context, including all errors from
// all retry attempts.
//
// This type implements Go 1.20+ multi-error unwrapping, allowing errors.Is
// and errors.As to check all wrapped errors. Use LastError() to access the
// final error directly.
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
//
// The function retries on any error unless a custom RetryIf function is provided.
// It returns immediately on success or when MaxAttempts is reached. The context
// is checked before each retry, allowing for early cancellation.
//
// On failure, returns a *Error containing the last error and all errors from
// all attempts. Use errors.As to inspect the retry error and access attempt
// counts and the complete error history.
//
// Type parameter T can be any type, allowing type-safe return values without
// requiring closure variable capture.
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
		// Check context before each attempt (including the first)
		if ctx.Err() != nil {
			return zero, &Error{
				Last:     ctx.Err(),
				Attempts: attempt,
				Errors:   append(errs, ctx.Err()),
			}
		}

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
//
// This is a convenience wrapper around Do for operations that don't return a
// value. It behaves identically to Do but with a simpler function signature.
//
// Example:
//
//	err := retry.DoSimple(ctx, func() error {
//	    return performOperation()
//	}, retry.WithMaxAttempts(3))
func DoSimple(ctx context.Context, fn func() error, opts ...Option) error {
	_, err := Do(ctx, func() (struct{}, error) {
		return struct{}{}, fn()
	}, opts...)
	return err
}

// Permanent wraps an error to indicate it should not be retried.
//
// When an error is wrapped with Permanent and a RetryIf function checks
// IsPermanent, the retry loop will exit immediately instead of retrying.
//
// Use MarkPermanent to create permanent errors and SkipPermanent() to create
// a RetryIf function that respects permanent error markers.
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
//
// Use this to indicate that an error should not be retried, such as validation
// errors or other errors that won't be fixed by retrying.
//
// Returns nil if err is nil. When used with WithRetryIf(SkipPermanent()),
// permanent errors will cause the retry loop to exit immediately.
func MarkPermanent(err error) error {
	if err == nil {
		return nil
	}
	return Permanent{Err: err}
}

// SkipPermanent returns a RetryIf function that skips permanent errors.
//
// Use with WithRetryIf to automatically respect errors marked with MarkPermanent:
//
//	retry.Do(ctx, fn, retry.WithRetryIf(retry.SkipPermanent()))
//
// This allows operations to signal that an error is permanent and should not
// be retried, while still allowing other errors to be retried normally.
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

// Retrier is a reusable retry configuration optimized for high-frequency operations.
//
// Create once with NewRetrier and reuse for multiple operations to minimize
// allocations and configuration overhead. All retry options are configured at
// creation time and applied to every Do call.
type Retrier struct {
	config Config
}

// NewRetrier creates a new Retrier with the given options.
//
// The returned Retrier can be safely reused across multiple retry operations,
// minimizing allocations in high-frequency scenarios. All options are configured
// once at creation time.
//
// Example:
//
//	retrier := retry.NewRetrier(
//	    retry.WithMaxAttempts(5),
//	    retry.WithBackoff(time.Second, 30*time.Second),
//	)
//	for item := range items {
//	    err := retrier.Do(ctx, func() error { return process(item) })
//	}
func NewRetrier(opts ...Option) *Retrier {
	cfg := DefaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	return &Retrier{config: cfg}
}

// Do executes the retryable function using this Retrier's configuration.
//
// This method is optimized for repeated use with the same retry configuration.
// The function signature accepts only error-returning functions; use the
// package-level Do function for operations that return values.
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
