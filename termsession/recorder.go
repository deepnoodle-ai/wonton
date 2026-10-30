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
	"unicode/utf8"

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
// where time is seconds elapsed, type is "o" (output), "i" (input), or
// "r" (resize), and data is the event payload.
type RecordingEvent struct {
	Time float64 // Seconds since recording start
	Type string  // "o" for terminal output, "i" for user input, "r" for resize
	Data string  // The event payload (output may contain ANSI codes; resize is "COLSxROWS")
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
	file           *os.File
	writer         *bufio.Writer
	gzipWriter     *gzip.Writer
	startTime      time.Time
	lastEventTime  time.Time
	pauseStart     time.Time
	timeAdjust     time.Duration // cumulative time removed for idle clamping and pauses
	mu             sync.Mutex
	compress       bool
	redactSecrets  bool
	idleTimeLimit  time.Duration
	paused         bool
	closed         bool
	width          int
	height         int
	recordedWidth  int
	recordedHeight int

	// Redaction matches each chunk with the end of the chunks before it, so
	// a secret split across PTY reads is still redacted.
	outputRedactor terminal.Redactor
	inputRedactor  terminal.Redactor

	// The start of a UTF-8 character split across calls, held back until
	// the rest arrives, so it is not written to the JSON as invalid UTF-8.
	outPartial string
	inPartial  string
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
	// One clock reading for both, so the first event's timestamp is measured
	// from the same instant the idle accounting starts from.
	now := time.Now()

	r := &Recorder{
		startTime:      now,
		lastEventTime:  now,
		compress:       opts.Compress,
		redactSecrets:  opts.RedactSecrets,
		idleTimeLimit:  time.Duration(opts.IdleTimeLimit * float64(time.Second)),
		width:          width,
		height:         height,
		recordedWidth:  width,
		recordedHeight: height,
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
// If RedactSecrets is enabled, passwords and API keys will be automatically
// redacted, including a secret split across calls. The run of text at the end
// of data is then held back until whitespace, an escape sequence, the next
// Pause, or Close ends it, so it can appear later in a replay. For example,
// a prompt with no trailing space, such as "Continue? [y/N]", is recorded
// with the next output, which is often the echo of the answer.
// Recording is skipped if the recorder is paused or if data is empty.
//
// This method is safe to call from multiple goroutines.
func (r *Recorder) RecordOutput(data string) {
	if r == nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.paused || r.closed {
		return
	}

	if r.redactSecrets {
		// The redactor can hold back the end of data until the next event.
		data = r.outputRedactor.Redact(data)
	}
	data, r.outPartial = splitIncompleteRune(r.outPartial + data)

	if len(data) == 0 {
		return
	}

	r.writeEvent(RecordingEvent{
		Time: r.eventTimeLocked(time.Now()),
		Type: "o",
		Data: data,
	})
}

// eventTimeLocked computes the timestamp for an event occurring at now,
// applying the idle time limit and accumulated pause adjustments.
// Caller must hold r.mu.
// The subtraction is kept in integer nanoseconds and converted to seconds
// exactly once, at the end. Converting each term separately would round each
// one independently, which can order two timestamps backwards when the true
// difference between them is zero -- as happens on hosts whose clock
// granularity is coarser than the interval being measured.
func (r *Recorder) eventTimeLocked(now time.Time) float64 {
	elapsed := now.Sub(r.startTime)

	// Apply idle time limit if configured
	if r.idleTimeLimit > 0 {
		timeSinceLastEvent := now.Sub(r.lastEventTime)
		if timeSinceLastEvent > r.idleTimeLimit {
			// Accumulate the excess time so subsequent events stay consistent
			r.timeAdjust += timeSinceLastEvent - r.idleTimeLimit
		}
	}

	r.lastEventTime = now

	return (elapsed - r.timeAdjust).Seconds()
}

// RecordInput records user input data.
//
// Input events are recorded but typically not displayed during playback
// (most players only render output events). They're useful for analysis
// and understanding user interactions with the terminal.
//
// If RedactSecrets is enabled, the same redaction applied to output is
// applied to input, and keystrokes are matched together, so a secret typed
// after its key ("export TOKEN=...") is redacted. A password typed at a
// prompt has no key in the input, and editing keys such as backspace can
// break up a secret; pause recording around sensitive prompts for stronger
// guarantees.
//
// This method is safe to call from multiple goroutines.
func (r *Recorder) RecordInput(data string) {
	if r == nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.paused || r.closed {
		return
	}

	if r.redactSecrets {
		data = r.inputRedactor.Redact(data)
	}
	data, r.inPartial = splitIncompleteRune(r.inPartial + data)

	if len(data) == 0 {
		return
	}

	r.writeEvent(RecordingEvent{
		Time: r.eventTimeLocked(time.Now()),
		Type: "i",
		Data: data,
	})
}

// flushRedactorsLocked records the text the redactors hold back, and any
// partial UTF-8 character, when nothing more can continue it, and resets
// them. Caller must hold r.mu.
func (r *Recorder) flushRedactorsLocked() {
	now := time.Now()
	if data := r.outPartial + r.outputRedactor.Flush(); data != "" {
		r.writeEvent(RecordingEvent{Time: r.eventTimeLocked(now), Type: "o", Data: data})
	}
	if data := r.inPartial + r.inputRedactor.Flush(); data != "" {
		r.writeEvent(RecordingEvent{Time: r.eventTimeLocked(now), Type: "i", Data: data})
	}
	r.outPartial, r.inPartial = "", ""
}

// splitIncompleteRune splits s before the start of an incomplete multi-byte
// UTF-8 character at its end. partial is empty if s ends on a character
// boundary, or with bytes that can never become a valid character.
func splitIncompleteRune(s string) (complete, partial string) {
	for back := 1; back <= utf8.UTFMax && back <= len(s); back++ {
		b := s[len(s)-back]
		if utf8.RuneStart(b) {
			if b >= 0xC0 && !utf8.FullRuneInString(s[len(s)-back:]) {
				return s[:len(s)-back], s[len(s)-back:]
			}
			break
		}
	}
	return s, ""
}

// Pause temporarily suspends recording.
//
// While paused, calls to RecordOutput and RecordInput are ignored.
// Use Resume to continue recording. This is useful for temporarily
// stopping recording during sensitive operations. Text held back for
// redaction is recorded before the pause.
func (r *Recorder) Pause() {
	if r == nil {
		return
	}

	r.mu.Lock()
	if !r.paused {
		if !r.closed {
			// Text during the pause is not seen, so the text before it must
			// not be joined to the text after it.
			r.flushRedactorsLocked()
		}
		r.paused = true
		r.pauseStart = time.Now()
	}
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
	if r.paused {
		r.paused = false
		// Remove the paused duration from the timeline so the recording
		// resumes from the point where Pause was called. Shift the last
		// event time by the same duration, so idle time before and after the
		// pause counts as one gap against IdleTimeLimit.
		paused := time.Since(r.pauseStart)
		r.timeAdjust += paused
		r.lastEventTime = r.lastEventTime.Add(paused)
		if !r.closed && (r.width != r.recordedWidth || r.height != r.recordedHeight) {
			r.recordedWidth = r.width
			r.recordedHeight = r.height
			r.writeEvent(RecordingEvent{
				Time: r.eventTimeLocked(time.Now()),
				Type: "r",
				Data: fmt.Sprintf("%dx%d", r.width, r.height),
			})
		}
	}
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
// This should be called when the terminal is resized. If the size actually
// changed, an asciicast v2 resize event ([time, "r", "COLSxROWS"]) is
// written so players can reproduce the resize during playback.
func (r *Recorder) UpdateSize(width, height int) {
	if r == nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return
	}
	r.width = width
	r.height = height

	if r.paused {
		return
	}
	if width == r.recordedWidth && height == r.recordedHeight {
		return
	}
	r.recordedWidth = width
	r.recordedHeight = height

	r.writeEvent(RecordingEvent{
		Time: r.eventTimeLocked(time.Now()),
		Type: "r",
		Data: fmt.Sprintf("%dx%d", width, height),
	})
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
// recorded events are written to disk immediately. With RedactSecrets,
// Flush does not write the text held back for redaction, since that would
// split a secret that the next chunk continues; Pause or Close writes it.
// Flush likewise holds back the start of a UTF-8 character that the next
// chunk completes.
func (r *Recorder) Flush() error {
	if r == nil {
		return nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return nil
	}

	if r.writer != nil {
		if err := r.writer.Flush(); err != nil {
			return err
		}
	}
	if r.gzipWriter != nil {
		if err := r.gzipWriter.Flush(); err != nil {
			return err
		}
	}
	return nil
}

// Close finalizes the recording and closes the file.
//
// This flushes all buffered data, including text held back for redaction,
// and closes any compression streams.
// Always call Close when finished recording to ensure data is not lost.
// It's safe to call Close multiple times.
func (r *Recorder) Close() error {
	if r == nil {
		return nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return nil
	}
	r.closed = true

	var firstErr error

	if r.writer != nil {
		r.flushRedactorsLocked()
		if err := r.writer.Flush(); err != nil && firstErr == nil {
			firstErr = err
		}
		r.writer = nil
	}

	if r.gzipWriter != nil {
		if err := r.gzipWriter.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
		r.gzipWriter = nil
	}

	if r.file != nil {
		if err := r.file.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
		r.file = nil
	}

	return firstErr
}
