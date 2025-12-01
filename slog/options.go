package slog

import (
	"log/slog"
	"os"
	"time"

	"golang.org/x/sys/unix"
)

// Options configures the Handler. A zero Options consists entirely of
// default values, which produce colorized output at Info level.
//
// Options can be used as a drop-in replacement for slog.HandlerOptions.
type Options struct {
	// AddSource causes the handler to compute the source code position
	// of the log statement and add a SourceKey attribute to the output.
	AddSource bool

	// Level reports the minimum record level that will be logged.
	// The handler discards records with lower levels.
	// If Level is nil, the handler assumes LevelInfo.
	Level slog.Leveler

	// ReplaceAttr is called to rewrite each non-group attribute before
	// it is logged. See slog.HandlerOptions for details.
	//
	// The built-in attributes with keys "time", "level", "source", and "msg"
	// are passed to this function first, with an empty groups slice.
	ReplaceAttr func(groups []string, attr slog.Attr) slog.Attr

	// TimeFormat is the time format for the timestamp.
	// Default: time.StampMilli (e.g., "Jan _2 15:04:05.000")
	TimeFormat string

	// NoColor disables colorized output.
	// Default: false (colors enabled)
	NoColor bool
}

func (o *Options) setDefaults() {
	if o.Level == nil {
		o.Level = slog.LevelInfo
	}
	if o.TimeFormat == "" {
		o.TimeFormat = defaultTimeFormat
	}
}

// DefaultOptions returns options with sensible defaults for terminal output.
// Colors are automatically enabled if stderr is a terminal.
func DefaultOptions() *Options {
	return &Options{
		Level:      slog.LevelInfo,
		TimeFormat: time.StampMilli,
		NoColor:    !IsTerminal(os.Stderr),
	}
}

// IsTerminal returns true if the given file is a terminal.
// This is useful for automatically disabling colors when output is redirected.
//
// Example:
//
//	opts := &slog.Options{
//	    NoColor: !slog.IsTerminal(os.Stderr),
//	}
func IsTerminal(f *os.File) bool {
	_, err := unix.IoctlGetTermios(int(f.Fd()), ioctlReadTermios)
	return err == nil
}
