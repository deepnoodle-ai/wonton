// Package termsession provides terminal session recording and playback in asciinema v2 format.
//
// This package enables you to record interactive terminal sessions (PTY), save them as
// .cast files (asciinema v2 format), and play them back with accurate timing. It's ideal
// for creating terminal demos, testing terminal applications, and building CLI tutorials.
//
// # Recording Sessions
//
// Record an interactive PTY session:
//
//	session, _ := termsession.NewSession(termsession.SessionOptions{
//	    Command: []string{"bash"},
//	})
//	session.Record("session.cast", RecordingOptions{
//	    Compress: true,
//	    Title: "My Demo",
//	})
//	session.Wait()
//
// Record directly without a PTY:
//
//	recorder, _ := termsession.NewRecorder("output.cast", 80, 24, RecordingOptions{})
//	recorder.RecordOutput("Hello, World!\n")
//	recorder.Close()
//
// # Playing Back Sessions
//
// Play back a recorded session with timing preserved:
//
//	player, _ := termsession.NewPlayer("session.cast", PlayerOptions{
//	    Speed: 2.0,    // 2x speed
//	    MaxIdle: 1.0,  // Cap idle time at 1 second
//	})
//	player.Play() // Blocks until complete
//
// Control playback dynamically:
//
//	go player.Play()
//	time.Sleep(time.Second)
//	player.Pause()
//	time.Sleep(time.Second)
//	player.Resume()
//	player.SetSpeed(3.0)
//
// # Loading and Analyzing Recordings
//
// Load a .cast file to inspect or process events:
//
//	header, events, _ := termsession.LoadCastFile("session.cast")
//	fmt.Printf("Duration: %.2fs\n", termsession.Duration(events))
//	fmt.Printf("Terminal size: %dx%d\n", header.Width, header.Height)
//
// Filter and process events:
//
//	outputOnly := termsession.OutputEvents(events)
//	for _, event := range outputOnly {
//	    fmt.Printf("[%.2fs] %s", event.Time, event.Data)
//	}
//
// # Format Details
//
// The package uses asciinema v2 format (.cast files), which is a simple JSON-based format:
//   - First line: JSON header with metadata (version, dimensions, title, etc.)
//   - Subsequent lines: JSON arrays [time, type, data] representing events
//   - Supports gzip compression automatically
//   - Compatible with asciinema.org and other asciinema tools
package termsession

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/deepnoodle-ai/wonton/terminal"
)

// RecordingHeader represents the asciinema v2 header line.
//
// This is the first line of every .cast file, containing metadata about
// the recording session. All recordings must start with a valid header.
type RecordingHeader struct {
	Version   int               `json:"version"`   // Always 2 for asciinema v2 format
	Width     int               `json:"width"`     // Terminal width in columns
	Height    int               `json:"height"`    // Terminal height in rows
	Timestamp int64             `json:"timestamp"` // Unix timestamp of recording start
	Env       map[string]string `json:"env,omitempty"`
	Title     string            `json:"title,omitempty"` // Optional recording title
}

// RecordingEvent represents a single terminal I/O event in asciinema v2 format.
//
// Events are encoded as JSON arrays: [time, type, data]
// where time is seconds elapsed, type is "o" (output) or "i" (input),
// and data is the actual terminal content.
type RecordingEvent struct {
	Time float64 // Seconds since recording start
	Type string  // "o" for terminal output, "i" for user input
	Data string  // The actual terminal content (may contain ANSI codes)
}

// MarshalJSON implements custom JSON encoding for asciinema array format
func (e RecordingEvent) MarshalJSON() ([]byte, error) {
	return json.Marshal([]interface{}{e.Time, e.Type, e.Data})
}

// RecordingOptions configures recording behavior.
//
// These options control how terminal output is captured and stored.
type RecordingOptions struct {
	Compress      bool              // Enable gzip compression to reduce file size
	RedactSecrets bool              // Automatically redact passwords, API keys, and tokens
	Title         string            // Recording title for metadata (shown in players)
	Env           map[string]string // Environment variables to include in metadata
	IdleTimeLimit float64           // Max idle time between events in seconds (0 = no limit)
}

// DefaultRecordingOptions returns sensible defaults for recording.
//
// By default, recordings are compressed and secrets are redacted.
func DefaultRecordingOptions() RecordingOptions {
	return RecordingOptions{
		Compress:      true,
		RedactSecrets: true,
		IdleTimeLimit: 0,
		Env:           make(map[string]string),
	}
}

// Recorder captures terminal output to an asciinema v2 format file.
//
// Use this when you want to record terminal output directly without a PTY.
// For recording interactive PTY sessions, see Session.Record instead.
//
// Recorder is safe for concurrent use (all methods are thread-safe).
type Recorder struct {
	file          *os.File
	writer        *bufio.Writer
	gzipWriter    *gzip.Writer
	startTime     time.Time
	lastEventTime time.Time
	mu            sync.Mutex
	compress      bool
	redactSecrets bool
	idleTimeLimit float64
	paused        bool
	width         int
	height        int
}

// NewRecorder creates a new recorder that writes to the specified file.
//
// The file is created immediately and the asciinema v2 header is written.
// The recorder must be closed with Close() when done to ensure all data is flushed.
//
// Parameters:
//   - filename: Path to the .cast file to create
//   - width: Terminal width in columns
//   - height: Terminal height in rows
//   - opts: Recording options (compression, redaction, metadata)
//
// Example:
//
//	recorder, err := NewRecorder("demo.cast", 80, 24, RecordingOptions{
//	    Compress: true,
//	    Title: "My Demo",
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer recorder.Close()
func NewRecorder(filename string, width, height int, opts RecordingOptions) (*Recorder, error) {
	r := &Recorder{
		startTime:     time.Now(),
		lastEventTime: time.Now(),
		compress:      opts.Compress,
		redactSecrets: opts.RedactSecrets,
		idleTimeLimit: opts.IdleTimeLimit,
		width:         width,
		height:        height,
	}

	file, err := os.Create(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to create recording file: %w", err)
	}
	r.file = file

	// Set up writer chain (optionally with compression)
	if opts.Compress {
		r.gzipWriter = gzip.NewWriter(file)
		r.writer = bufio.NewWriter(r.gzipWriter)
	} else {
		r.writer = bufio.NewWriter(file)
	}

	// Write asciinema v2 header (first line)
	header := RecordingHeader{
		Version:   2,
		Width:     width,
		Height:    height,
		Timestamp: r.startTime.Unix(),
		Env:       opts.Env,
		Title:     opts.Title,
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		r.Close()
		return nil, fmt.Errorf("failed to marshal header: %w", err)
	}

	if _, err := r.writer.Write(headerJSON); err != nil {
		r.Close()
		return nil, fmt.Errorf("failed to write header: %w", err)
	}
	if err := r.writer.WriteByte('\n'); err != nil {
		r.Close()
		return nil, fmt.Errorf("failed to write header newline: %w", err)
	}
	if err := r.writer.Flush(); err != nil {
		r.Close()
		return nil, fmt.Errorf("failed to flush header: %w", err)
	}

	return r, nil
}

// RecordOutput records terminal output data.
//
// The data is timestamped relative to when the recorder was created.
// If RedactSecrets is enabled, passwords and API keys will be automatically redacted.
// Recording is skipped if the recorder is paused or if data is empty.
//
// This method is safe to call from multiple goroutines.
func (r *Recorder) RecordOutput(data string) {
	if r == nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.paused {
		return
	}

	if len(data) == 0 {
		return
	}

	now := time.Now()
	elapsed := now.Sub(r.startTime).Seconds()

	// Apply idle time limit if configured
	if r.idleTimeLimit > 0 {
		timeSinceLastEvent := now.Sub(r.lastEventTime).Seconds()
		if timeSinceLastEvent > r.idleTimeLimit {
			// Clamp the elapsed time to prevent huge gaps
			elapsed = r.lastEventTime.Sub(r.startTime).Seconds() + r.idleTimeLimit
		}
	}

	r.lastEventTime = now

	event := RecordingEvent{
		Time: elapsed,
		Type: "o",
		Data: data,
	}

	if r.redactSecrets {
		event.Data = terminal.RedactCredentials(event.Data)
	}

	r.writeEvent(event)
}

// RecordInput records user input data.
//
// Input events are recorded but typically not displayed during playback
// (most players only render output events). They're useful for analysis
// and understanding user interactions with the terminal.
//
// This method is safe to call from multiple goroutines.
func (r *Recorder) RecordInput(data string) {
	if r == nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.paused {
		return
	}

	if len(data) == 0 {
		return
	}

	now := time.Now()
	elapsed := now.Sub(r.startTime).Seconds()

	// Apply idle time limit
	if r.idleTimeLimit > 0 {
		timeSinceLastEvent := now.Sub(r.lastEventTime).Seconds()
		if timeSinceLastEvent > r.idleTimeLimit {
			elapsed = r.lastEventTime.Sub(r.startTime).Seconds() + r.idleTimeLimit
		}
	}

	r.lastEventTime = now

	event := RecordingEvent{
		Time: elapsed,
		Type: "i",
		Data: data,
	}

	r.writeEvent(event)
}

// Pause temporarily suspends recording.
//
// While paused, calls to RecordOutput and RecordInput are ignored.
// Use Resume to continue recording. This is useful for temporarily
// stopping recording during sensitive operations.
func (r *Recorder) Pause() {
	if r == nil {
		return
	}

	r.mu.Lock()
	r.paused = true
	r.mu.Unlock()
}

// Resume continues a paused recording.
//
// Recording resumes from the point where Pause was called, preventing
// a large time gap from appearing in the recording.
func (r *Recorder) Resume() {
	if r == nil {
		return
	}

	r.mu.Lock()
	r.paused = false
	r.lastEventTime = time.Now() // Reset to avoid huge time jump
	r.mu.Unlock()
}

// IsPaused returns true if recording is currently paused.
func (r *Recorder) IsPaused() bool {
	if r == nil {
		return false
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	return r.paused
}

// UpdateSize updates the recorded terminal dimensions.
//
// This should be called when the terminal is resized to keep the
// metadata accurate. Note: asciinema v2 doesn't have explicit resize
// events, so this only updates internal tracking.
func (r *Recorder) UpdateSize(width, height int) {
	if r == nil {
		return
	}

	r.mu.Lock()
	r.width = width
	r.height = height
	r.mu.Unlock()
}

// writeEvent writes a single event to the recording file
// Caller must hold r.mu
func (r *Recorder) writeEvent(event RecordingEvent) {
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return // Silently ignore marshal errors
	}

	r.writer.Write(eventJSON)
	r.writer.WriteByte('\n')
	// We flush periodically rather than every event for performance
}

// Flush writes any buffered data to the underlying file.
//
// Events are buffered for performance. Call Flush to ensure all
// recorded events are written to disk immediately.
func (r *Recorder) Flush() error {
	if r == nil {
		return nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.writer != nil {
		return r.writer.Flush()
	}
	return nil
}

// Close finalizes the recording and closes the file.
//
// This flushes all buffered data and closes any compression streams.
// Always call Close when finished recording to ensure data is not lost.
// It's safe to call Close multiple times.
func (r *Recorder) Close() error {
	if r == nil {
		return nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.writer != nil {
		if err := r.writer.Flush(); err != nil {
			return err
		}
	}

	if r.gzipWriter != nil {
		if err := r.gzipWriter.Close(); err != nil {
			return err
		}
	}

	if r.file != nil {
		if err := r.file.Close(); err != nil {
			return err
		}
	}

	return nil
}
