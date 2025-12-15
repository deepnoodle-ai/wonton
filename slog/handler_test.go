package slog

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestHandler_BasicOutput(t *testing.T) {
	var buf bytes.Buffer

	h := NewHandler(&buf, &Options{NoColor: true})
	logger := slog.New(h)

	logger.Info("hello world")

	output := buf.String()
	assert.Contains(t, output, "INF")
	assert.Contains(t, output, "hello world")
}

func TestHandler_Levels(t *testing.T) {
	tests := []struct {
		level    slog.Level
		expected string
	}{
		{slog.LevelDebug, "DBG"},
		{slog.LevelInfo, "INF"},
		{slog.LevelWarn, "WRN"},
		{slog.LevelError, "ERR"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			var buf bytes.Buffer

			h := NewHandler(&buf, &Options{
				NoColor: true,
				Level:   slog.LevelDebug,
			})
			logger := slog.New(h)

			logger.Log(context.Background(), tt.level, "test message")

			output := buf.String()
			assert.Contains(t, output, tt.expected)
		})
	}
}

func TestHandler_Attributes(t *testing.T) {
	var buf bytes.Buffer

	h := NewHandler(&buf, &Options{NoColor: true})
	logger := slog.New(h)

	logger.Info("test",
		slog.String("name", "alice"),
		slog.Int("age", 30),
		slog.Bool("active", true),
		slog.Float64("score", 98.5),
	)

	output := buf.String()
	assert.Contains(t, output, "name=alice")
	assert.Contains(t, output, "age=30")
	assert.Contains(t, output, "active=true")
	assert.Contains(t, output, "score=98.5")
}

func TestHandler_Groups(t *testing.T) {
	var buf bytes.Buffer

	h := NewHandler(&buf, &Options{NoColor: true})
	logger := slog.New(h)

	logger.Info("test", slog.Group("user",
		slog.String("name", "bob"),
		slog.Int("id", 123),
	))

	output := buf.String()
	assert.Contains(t, output, "user.name=bob")
	assert.Contains(t, output, "user.id=123")
}

func TestHandler_WithGroup(t *testing.T) {
	var buf bytes.Buffer

	h := NewHandler(&buf, &Options{NoColor: true})
	logger := slog.New(h).WithGroup("request")

	logger.Info("received",
		slog.String("method", "GET"),
		slog.String("path", "/api/users"),
	)

	output := buf.String()
	assert.Contains(t, output, "request.method=GET")
	assert.Contains(t, output, "request.path=/api/users")
}

func TestHandler_WithAttrs(t *testing.T) {
	var buf bytes.Buffer

	h := NewHandler(&buf, &Options{NoColor: true})
	logger := slog.New(h).With(
		slog.String("service", "api"),
		slog.Int("version", 2),
	)

	logger.Info("started")

	output := buf.String()
	assert.Contains(t, output, "service=api")
	assert.Contains(t, output, "version=2")
}

func TestHandler_ColoredOutput(t *testing.T) {
	var buf bytes.Buffer

	h := NewHandler(&buf, &Options{NoColor: false})
	logger := slog.New(h)

	logger.Info("colored message")

	output := buf.String()
	// Should contain ANSI escape codes
	assert.Contains(t, output, "\u001b[")
}

func TestHandler_NoColorOutput(t *testing.T) {
	var buf bytes.Buffer

	h := NewHandler(&buf, &Options{NoColor: true})
	logger := slog.New(h)

	logger.Info("plain message")

	output := buf.String()
	// Should not contain ANSI escape codes
	assert.NotContains(t, output, "\u001b[")
}

func TestHandler_LevelFiltering(t *testing.T) {
	var buf bytes.Buffer

	h := NewHandler(&buf, &Options{
		NoColor: true,
		Level:   slog.LevelWarn,
	})
	logger := slog.New(h)

	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warn message")
	logger.Error("error message")

	output := buf.String()
	assert.NotContains(t, output, "debug message")
	assert.NotContains(t, output, "info message")
	assert.Contains(t, output, "warn message")
	assert.Contains(t, output, "error message")
}

func TestHandler_ReplaceAttr(t *testing.T) {
	var buf bytes.Buffer

	h := NewHandler(&buf, &Options{
		NoColor: true,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Remove time from output
			if a.Key == slog.TimeKey && len(groups) == 0 {
				return slog.Attr{}
			}
			return a
		},
	})
	logger := slog.New(h)

	logger.Info("no time")

	output := buf.String()
	// Should start with level, not time
	assert.True(t, strings.HasPrefix(output, "INF"))
}

func TestHandler_CustomTimeFormat(t *testing.T) {
	var buf bytes.Buffer

	h := NewHandler(&buf, &Options{
		NoColor:    true,
		TimeFormat: time.Kitchen,
	})
	logger := slog.New(h)

	logger.Info("custom time")

	output := buf.String()
	// Kitchen format is like "3:04PM"
	assert.Regexp(t, `\d{1,2}:\d{2}(AM|PM)`, output)
}

func TestHandler_Err(t *testing.T) {
	var buf bytes.Buffer

	h := NewHandler(&buf, &Options{NoColor: true})
	logger := slog.New(h)

	err := errors.New("connection refused")
	logger.Error("failed to connect", Err(err))

	output := buf.String()
	assert.Contains(t, output, "err=")
	assert.Contains(t, output, "connection refused")
}

func TestHandler_ColoredAttr(t *testing.T) {
	var buf bytes.Buffer

	h := NewHandler(&buf, &Options{NoColor: false})
	logger := slog.New(h)

	logger.Info("test",
		Red(slog.String("error", "fail")),
		Green(slog.String("status", "ok")),
	)

	output := buf.String()
	// Should contain color codes around the attributes
	assert.Contains(t, output, "\u001b[")
}

func TestHandler_Duration(t *testing.T) {
	var buf bytes.Buffer

	h := NewHandler(&buf, &Options{NoColor: true})
	logger := slog.New(h)

	logger.Info("request completed",
		slog.Duration("elapsed", 150*time.Millisecond),
	)

	output := buf.String()
	assert.Contains(t, output, "elapsed=150ms")
}

func TestHandler_AddSource(t *testing.T) {
	var buf bytes.Buffer

	h := NewHandler(&buf, &Options{
		NoColor:   true,
		AddSource: true,
	})
	logger := slog.New(h)

	logger.Info("with source")

	output := buf.String()
	// Should contain file:line
	assert.Contains(t, output, "handler_test.go:")
}

func TestHandler_EmptyMessage(t *testing.T) {
	var buf bytes.Buffer

	h := NewHandler(&buf, &Options{NoColor: true})
	logger := slog.New(h)

	logger.Info("", slog.String("key", "value"))

	output := buf.String()
	assert.Contains(t, output, "key=value")
}

func TestHandler_QuotedStrings(t *testing.T) {
	var buf bytes.Buffer

	h := NewHandler(&buf, &Options{NoColor: true})
	logger := slog.New(h)

	logger.Info("test",
		slog.String("with space", "hello world"),
		slog.String("with=equals", "a=b"),
	)

	output := buf.String()
	// Strings with spaces should be quoted
	assert.Contains(t, output, `"hello world"`)
}

func TestHandler_Concurrent(t *testing.T) {
	var buf bytes.Buffer

	h := NewHandler(&buf, &Options{NoColor: true})
	logger := slog.New(h)

	done := make(chan bool)
	for i := 0; i < 100; i++ {
		go func(n int) {
			logger.Info("concurrent log", slog.Int("n", n))
			done <- true
		}(i)
	}

	for i := 0; i < 100; i++ {
		<-done
	}

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	assert.Len(t, lines, 100)
}

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()
	assert.NotNil(t, opts)
	assert.Equal(t, slog.LevelInfo, opts.Level)
	assert.Equal(t, time.StampMilli, opts.TimeFormat)
}

func TestColorConstants(t *testing.T) {
	assert.Equal(t, uint8(0), ColorBlack)
	assert.Equal(t, uint8(1), ColorRed)
	assert.Equal(t, uint8(2), ColorGreen)
	assert.Equal(t, uint8(9), ColorBrightRed)
}

func BenchmarkHandler(b *testing.B) {
	var buf bytes.Buffer
	h := NewHandler(&buf, &Options{NoColor: true})
	logger := slog.New(h)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message",
			slog.String("key", "value"),
			slog.Int("count", i),
		)
		buf.Reset()
	}
}

func BenchmarkHandlerWithColor(b *testing.B) {
	var buf bytes.Buffer
	h := NewHandler(&buf, &Options{NoColor: false})
	logger := slog.New(h)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message",
			slog.String("key", "value"),
			slog.Int("count", i),
		)
		buf.Reset()
	}
}
