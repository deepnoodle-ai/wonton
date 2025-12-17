// Package termsession provides terminal session recording and playback.
//
// Recording captures PTY I/O to asciinema v2 format (.cast files):
//
//	session, _ := termsession.NewSession(termsession.SessionOptions{
//	    Command: []string{"bash"},
//	})
//	session.Record("session.cast", RecordingOptions{})
//	session.Wait()
//
// Playback replays recordings with timing preservation:
//
//	player, _ := termsession.NewPlayer("session.cast", PlayerOptions{})
//	player.Play() // Blocks until complete
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

// RecordingHeader represents asciinema v2 header (first line of .cast file)
type RecordingHeader struct {
	Version   int               `json:"version"`
	Width     int               `json:"width"`
	Height    int               `json:"height"`
	Timestamp int64             `json:"timestamp"`
	Env       map[string]string `json:"env,omitempty"`
	Title     string            `json:"title,omitempty"`
}

// RecordingEvent represents a single event [time, type, data]
type RecordingEvent struct {
	Time float64 // Seconds since recording start
	Type string  // "o" (output) or "i" (input)
	Data string  // The actual content
}

// MarshalJSON implements custom JSON encoding for asciinema array format
func (e RecordingEvent) MarshalJSON() ([]byte, error) {
	return json.Marshal([]interface{}{e.Time, e.Type, e.Data})
}

// RecordingOptions configures recording behavior
type RecordingOptions struct {
	Compress      bool              // Enable gzip compression
	RedactSecrets bool              // Redact passwords, tokens, etc.
	Title         string            // Recording title (metadata)
	Env           map[string]string // Environment variables (metadata)
	IdleTimeLimit float64           // Max idle time between events (0 = no limit)
}

// DefaultRecordingOptions returns sensible defaults
func DefaultRecordingOptions() RecordingOptions {
	return RecordingOptions{
		Compress:      true,
		RedactSecrets: true,
		IdleTimeLimit: 0,
		Env:           make(map[string]string),
	}
}

// Recorder handles standalone session recording to asciinema v2 format
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

// NewRecorder creates a new standalone recorder
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

// RecordOutput captures terminal output
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

// RecordInput captures user input
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

// Pause temporarily suspends recording
func (r *Recorder) Pause() {
	if r == nil {
		return
	}

	r.mu.Lock()
	r.paused = true
	r.mu.Unlock()
}

// Resume continues a paused recording
func (r *Recorder) Resume() {
	if r == nil {
		return
	}

	r.mu.Lock()
	r.paused = false
	r.lastEventTime = time.Now() // Reset to avoid huge time jump
	r.mu.Unlock()
}

// IsPaused returns true if recording is paused
func (r *Recorder) IsPaused() bool {
	if r == nil {
		return false
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	return r.paused
}

// UpdateSize updates the recorded terminal dimensions (for resize events)
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

// Flush writes any buffered data to the file
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

// Close finalizes the recording and closes the file
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
