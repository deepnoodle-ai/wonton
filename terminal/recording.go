package terminal

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
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
// Implements custom JSON marshaling for asciinema format
type RecordingEvent struct {
	Time float64 // Seconds since recording start
	Type string  // "o" (output) or "i" (input)
	Data string  // The actual content
}

// MarshalJSON implements custom JSON encoding for asciinema array format
func (e RecordingEvent) MarshalJSON() ([]byte, error) {
	return json.Marshal([]interface{}{e.Time, e.Type, e.Data})
}

// Recorder handles session recording to asciinema v2 format
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

// StartRecording begins recording a session to the specified file
func (t *Terminal) StartRecording(filename string, opts RecordingOptions) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.recorder != nil {
		return fmt.Errorf("recording already in progress")
	}

	recorder := &Recorder{
		startTime:     time.Now(),
		lastEventTime: time.Now(),
		compress:      opts.Compress,
		redactSecrets: opts.RedactSecrets,
		idleTimeLimit: opts.IdleTimeLimit,
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create recording file: %w", err)
	}
	recorder.file = file

	// Set up writer chain (optionally with compression)
	if opts.Compress {
		recorder.gzipWriter = gzip.NewWriter(file)
		recorder.writer = bufio.NewWriter(recorder.gzipWriter)
	} else {
		recorder.writer = bufio.NewWriter(file)
	}

	// Write asciinema v2 header (first line)
	header := RecordingHeader{
		Version:   2,
		Width:     t.width,
		Height:    t.height,
		Timestamp: recorder.startTime.Unix(),
		Env:       opts.Env,
		Title:     opts.Title,
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		recorder.close()
		return fmt.Errorf("failed to marshal header: %w", err)
	}

	if _, err := recorder.writer.Write(headerJSON); err != nil {
		recorder.close()
		return fmt.Errorf("failed to write header: %w", err)
	}
	if err := recorder.writer.WriteByte('\n'); err != nil {
		recorder.close()
		return fmt.Errorf("failed to write header newline: %w", err)
	}
	if err := recorder.writer.Flush(); err != nil {
		recorder.close()
		return fmt.Errorf("failed to flush header: %w", err)
	}

	t.recorder = recorder
	return nil
}

// StopRecording finalizes and closes the recording
func (t *Terminal) StopRecording() error {
	t.mu.Lock()
	recorder := t.recorder
	t.recorder = nil
	t.mu.Unlock()

	if recorder == nil {
		return nil
	}

	return recorder.close()
}

// PauseRecording temporarily suspends recording (useful for sensitive sections)
func (t *Terminal) PauseRecording() {
	t.mu.Lock()
	recorder := t.recorder
	t.mu.Unlock()

	if recorder != nil {
		recorder.mu.Lock()
		recorder.paused = true
		recorder.mu.Unlock()
	}
}

// ResumeRecording resumes a paused recording
func (t *Terminal) ResumeRecording() {
	t.mu.Lock()
	recorder := t.recorder
	t.mu.Unlock()

	if recorder != nil {
		recorder.mu.Lock()
		recorder.paused = false
		recorder.lastEventTime = time.Now() // Reset to avoid huge time jump
		recorder.mu.Unlock()
	}
}

// IsRecording returns true if a recording is active
func (t *Terminal) IsRecording() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.recorder != nil
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
		event.Data = redactSecretPatterns(event.Data)
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
	// The close() method ensures final flush
}

// close finalizes the recording and closes the file
func (r *Recorder) close() error {
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
