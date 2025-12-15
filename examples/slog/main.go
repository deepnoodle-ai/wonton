// Example demonstrating the colorized slog handler.
package main

import (
	"errors"
	"log/slog"
	"os"
	"time"

	wontonslog "github.com/deepnoodle-ai/wonton/slog"
)

func main() {
	// Basic usage with default options (colors enabled, INFO level)
	logger := slog.New(wontonslog.NewHandler(os.Stderr, nil))

	logger.Info("Server started",
		slog.String("host", "localhost"),
		slog.Int("port", 8080),
	)

	logger.Debug("Debug message (won't show at INFO level)")

	// Logger with debug level and auto-detection for terminal colors
	debugLogger := slog.New(wontonslog.NewHandler(os.Stderr, &wontonslog.Options{
		Level:   slog.LevelDebug,
		NoColor: !wontonslog.IsTerminal(os.Stderr),
	}))

	debugLogger.Debug("Debug message",
		slog.String("component", "database"),
	)

	// Log with different levels to show color coding
	logger.Info("Processing request",
		slog.String("method", "GET"),
		slog.String("path", "/api/users"),
	)

	logger.Warn("High memory usage",
		slog.Float64("percent", 87.5),
	)

	// Using colored attributes
	logger.Error("Failed to connect",
		wontonslog.Err(errors.New("connection refused")),
		wontonslog.Red(slog.String("host", "db.example.com")),
	)

	// Using convenience color functions
	logger.Info("User action",
		wontonslog.Green(slog.String("action", "login")),
		wontonslog.Blue(slog.String("user", "alice")),
		wontonslog.Cyan(slog.Int("session_id", 12345)),
	)

	// Logger with groups
	requestLogger := logger.WithGroup("request")
	requestLogger.Info("Received",
		slog.String("method", "POST"),
		slog.String("path", "/api/orders"),
		slog.Duration("latency", 45*time.Millisecond),
	)

	// Logger with persistent attributes
	serviceLogger := logger.With(
		slog.String("service", "payment"),
		slog.String("version", "1.2.0"),
	)
	serviceLogger.Info("Transaction completed",
		slog.String("tx_id", "abc123"),
		slog.Float64("amount", 99.99),
	)

	// Custom time format
	customLogger := slog.New(wontonslog.NewHandler(os.Stderr, &wontonslog.Options{
		TimeFormat: time.RFC3339,
	}))
	customLogger.Info("With RFC3339 timestamp")

	// With source location
	sourceLogger := slog.New(wontonslog.NewHandler(os.Stderr, &wontonslog.Options{
		AddSource: true,
	}))
	sourceLogger.Info("Shows file and line number")

	// Remove time from output using ReplaceAttr
	noTimeLogger := slog.New(wontonslog.NewHandler(os.Stderr, &wontonslog.Options{
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey && len(groups) == 0 {
				return slog.Attr{}
			}
			return a
		},
	}))
	noTimeLogger.Info("No timestamp in this log")

	// Using the global default
	slog.SetDefault(slog.New(wontonslog.NewHandler(os.Stderr, wontonslog.DefaultOptions())))
	slog.Info("Using the global default logger")
}
