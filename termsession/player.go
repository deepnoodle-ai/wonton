package termsession

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// Player plays back recorded terminal sessions with timing preservation.
//
// Player loads asciinema v2 format recordings and plays them back to an
// io.Writer, preserving the original timing between events. It supports
// speed adjustment, pause/resume, seeking, and looping.
//
// Playback is performed in a blocking manner by Play(), or you can run
// it in a goroutine and control it with the various control methods.
//
// All methods are safe for concurrent use.
type Player struct {
	events       []RecordingEvent
	header       RecordingHeader
	output       io.Writer
	currentIndex int
	startTime    time.Time
	pauseTime    time.Time
	totalPaused  time.Duration
	paused       bool
	speed        float64
	loop         bool
	maxIdle      float64
	mu           sync.RWMutex
	stopChan     chan struct{}
	stopped      bool
}

// PlayerOptions configures playback behavior.
type PlayerOptions struct {
	Speed   float64   // Playback speed multiplier (1.0 = normal speed, 2.0 = 2x, etc.)
	Loop    bool      // Loop playback when finished (restart from beginning)
	MaxIdle float64   // Max idle time between events in seconds (0 = preserve original timing)
	Output  io.Writer // Output destination (default: os.Stdout)
}

// DefaultPlayerOptions returns sensible defaults for playback.
//
// Returns options with normal speed (1.0), no looping, and output to stdout.
func DefaultPlayerOptions() PlayerOptions {
	return PlayerOptions{
		Speed:   1.0,
		Loop:    false,
		MaxIdle: 0,
		Output:  os.Stdout,
	}
}

// NewPlayer creates a new player from a .cast recording file.
//
// The recording file is loaded completely into memory. Files can be
// gzip-compressed (detected automatically). Invalid or malformed events
// are silently skipped during loading.
//
// Example:
//
//	player, err := NewPlayer("demo.cast", PlayerOptions{
//	    Speed: 2.0,
//	    MaxIdle: 1.0,
//	    Output: os.Stdout,
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	player.Play() // Blocks until playback completes
func NewPlayer(filename string, opts PlayerOptions) (*Player, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open recording: %w", err)
	}
	defer file.Close()

	var reader io.Reader = file

	// Detect gzip compression by checking magic bytes
	magic := make([]byte, 2)
	if _, err := io.ReadFull(file, magic); err != nil {
		return nil, fmt.Errorf("failed to read file header: %w", err)
	}
	if _, err := file.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("failed to seek: %w", err)
	}

	if magic[0] == 0x1f && magic[1] == 0x8b {
		gzipReader, err := gzip.NewReader(file)
		if err != nil {
			return nil, fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer gzipReader.Close()
		reader = gzipReader
	}

	scanner := bufio.NewScanner(reader)

	// Read header (first line)
	if !scanner.Scan() {
		return nil, fmt.Errorf("empty recording file")
	}

	var header RecordingHeader
	if err := json.Unmarshal(scanner.Bytes(), &header); err != nil {
		return nil, fmt.Errorf("failed to parse header: %w", err)
	}

	// Read events (remaining lines)
	var events []RecordingEvent
	for scanner.Scan() {
		var raw []interface{}
		if err := json.Unmarshal(scanner.Bytes(), &raw); err != nil {
			// Skip malformed lines
			continue
		}

		if len(raw) < 3 {
			// Skip incomplete events
			continue
		}

		// Parse [time, type, data] array
		timeVal, ok := raw[0].(float64)
		if !ok {
			continue
		}
		typeVal, ok := raw[1].(string)
		if !ok {
			continue
		}
		dataVal, ok := raw[2].(string)
		if !ok {
			continue
		}

		event := RecordingEvent{
			Time: timeVal,
			Type: typeVal,
			Data: dataVal,
		}
		events = append(events, event)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading recording: %w", err)
	}

	output := opts.Output
	if output == nil {
		output = os.Stdout
	}

	speed := opts.Speed
	if speed <= 0 {
		speed = 1.0
	}

	return &Player{
		events:   events,
		header:   header,
		output:   output,
		speed:    speed,
		loop:     opts.Loop,
		maxIdle:  opts.MaxIdle,
		stopChan: make(chan struct{}),
	}, nil
}

// Play starts playback of the recording.
//
// This method blocks until playback completes, is stopped via Stop(),
// or an error occurs. Events are written to the configured output writer
// with timing preserved according to the speed multiplier.
//
// If Loop is enabled, playback will restart from the beginning when it
// reaches the end. Call Stop() from another goroutine to end looped playback.
//
// Example:
//
//	// Blocking playback
//	err := player.Play()
//
//	// Playback in background with controls
//	go player.Play()
//	time.Sleep(5 * time.Second)
//	player.Pause()
func (p *Player) Play() error {
	p.mu.Lock()
	p.startTime = time.Now()
	p.currentIndex = 0
	p.paused = false
	p.stopped = false
	p.totalPaused = 0
	p.mu.Unlock()

	// Preprocess events to apply maxIdle if configured
	events := p.events
	if p.maxIdle > 0 {
		events = p.applyMaxIdle(events)
	}

	for {
		p.mu.RLock()
		if p.stopped {
			p.mu.RUnlock()
			return nil
		}

		if p.currentIndex >= len(events) {
			if p.loop {
				p.mu.RUnlock()
				p.mu.Lock()
				p.currentIndex = 0
				p.startTime = time.Now()
				p.totalPaused = 0
				p.mu.Unlock()
				continue
			} else {
				p.mu.RUnlock()
				return nil
			}
		}

		paused := p.paused
		p.mu.RUnlock()

		// Handle pause state
		if paused {
			time.Sleep(50 * time.Millisecond)
			continue
		}

		p.mu.RLock()
		event := events[p.currentIndex]
		speed := p.speed
		currentIndex := p.currentIndex
		p.mu.RUnlock()

		// Calculate when this event should fire
		targetTime := event.Time / speed
		elapsed := time.Since(p.startTime).Seconds() - p.totalPaused.Seconds()

		if elapsed < targetTime {
			sleepDuration := time.Duration((targetTime - elapsed) * float64(time.Second))
			// Only sleep if duration is meaningful (>= 10ms)
			const minSleepDuration = 10 * time.Millisecond
			if sleepDuration >= minSleepDuration {
				select {
				case <-p.stopChan:
					return nil
				case <-time.After(sleepDuration):
				}
			}
		}

		// Batch output events that are very close together (< 10ms)
		const batchThreshold = 0.01 // 10ms in seconds
		var batchedOutput []string
		batchIndex := currentIndex

		p.mu.RLock()
		for batchIndex < len(events) {
			batchEvent := events[batchIndex]

			if batchEvent.Type == "o" {
				timeDiff := (batchEvent.Time - event.Time) / speed
				if batchIndex == currentIndex || timeDiff < batchThreshold {
					batchedOutput = append(batchedOutput, batchEvent.Data)
					batchIndex++
				} else {
					break
				}
			} else {
				// Skip input events (they're informational only)
				if batchIndex > currentIndex {
					break
				}
				batchIndex++
			}
		}
		p.mu.RUnlock()

		// Write all batched output at once
		for _, data := range batchedOutput {
			p.output.Write([]byte(data))
		}

		p.mu.Lock()
		p.currentIndex = batchIndex
		p.mu.Unlock()
	}
}

// applyMaxIdle adjusts event times to cap idle periods
func (p *Player) applyMaxIdle(events []RecordingEvent) []RecordingEvent {
	if len(events) == 0 {
		return events
	}

	result := make([]RecordingEvent, len(events))
	var adjustment float64

	for i, event := range events {
		if i > 0 {
			gap := event.Time - events[i-1].Time
			if gap > p.maxIdle {
				adjustment += gap - p.maxIdle
			}
		}

		result[i] = RecordingEvent{
			Time: event.Time - adjustment,
			Type: event.Type,
			Data: event.Data,
		}
	}

	return result
}

// Pause pauses playback.
//
// Playback can be resumed with Resume(). While paused, no events
// are written to the output. The pause time is tracked internally
// to prevent time jumps when resuming.
func (p *Player) Pause() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.paused {
		p.paused = true
		p.pauseTime = time.Now()
	}
}

// Resume resumes a paused playback.
//
// If playback is not paused, this is a no-op.
func (p *Player) Resume() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.paused {
		p.paused = false
		pauseDuration := time.Since(p.pauseTime)
		p.totalPaused += pauseDuration
	}
}

// TogglePause toggles between paused and playing states.
func (p *Player) TogglePause() {
	p.mu.RLock()
	paused := p.paused
	p.mu.RUnlock()

	if paused {
		p.Resume()
	} else {
		p.Pause()
	}
}

// IsPaused returns true if playback is currently paused.
func (p *Player) IsPaused() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.paused
}

// Stop stops playback immediately and permanently.
//
// After calling Stop, the player cannot be restarted. Create a new
// player if you need to play the recording again.
func (p *Player) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.stopped {
		p.stopped = true
		close(p.stopChan)
	}
}

// SetSpeed changes the playback speed multiplier.
//
// The speed is adjusted smoothly to prevent jumps in playback position.
// Values less than or equal to 0 are ignored. Common values:
//   - 0.5 = half speed (slower)
//   - 1.0 = normal speed
//   - 2.0 = double speed (faster)
func (p *Player) SetSpeed(speed float64) {
	if speed <= 0 {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Adjust timing to prevent jumps when speed changes
	if !p.paused && p.currentIndex < len(p.events) {
		elapsed := time.Since(p.startTime).Seconds() - p.totalPaused.Seconds()
		currentEventTime := p.events[p.currentIndex].Time

		// Calculate new start time to maintain position
		newElapsed := currentEventTime / speed
		adjustment := elapsed - newElapsed
		p.startTime = p.startTime.Add(time.Duration(adjustment * float64(time.Second)))
	}

	p.speed = speed
}

// Speed returns the current playback speed multiplier.
func (p *Player) Speed() float64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.speed
}

// SetLoop enables or disables looping.
//
// When enabled, playback restarts from the beginning when it reaches the end.
func (p *Player) SetLoop(loop bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.loop = loop
}

// Seek jumps to a specific time offset in the recording.
//
// The time is specified in seconds from the start of the recording.
// Seeking adjusts the playback position to the event closest to the
// target time. This can be called during playback.
func (p *Player) Seek(seconds float64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Find the event closest to the target time
	targetIndex := 0
	for i, event := range p.events {
		if event.Time > seconds {
			break
		}
		targetIndex = i
	}

	p.currentIndex = targetIndex
	// Adjust start time to account for the seek
	elapsed := time.Since(p.startTime).Seconds() - p.totalPaused.Seconds()
	adjustment := elapsed - (seconds / p.speed)
	p.startTime = p.startTime.Add(time.Duration(adjustment * float64(time.Second)))
}

// GetHeader returns the recording metadata.
//
// This includes terminal dimensions, title, timestamp, and environment variables.
func (p *Player) GetHeader() RecordingHeader {
	return p.header
}

// GetDuration returns the total duration of the recording in seconds.
//
// Returns 0 if the recording has no events.
func (p *Player) GetDuration() float64 {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(p.events) == 0 {
		return 0
	}
	return p.events[len(p.events)-1].Time
}

// GetPosition returns the current playback position in seconds.
//
// This is the timestamp of the current event being played.
func (p *Player) GetPosition() float64 {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.currentIndex >= len(p.events) {
		if len(p.events) == 0 {
			return 0
		}
		return p.events[len(p.events)-1].Time
	}
	if p.currentIndex == 0 {
		return 0
	}
	return p.events[p.currentIndex].Time
}

// GetProgress returns playback progress as a fraction between 0.0 and 1.0.
//
// Returns 0.0 at the start, 1.0 at the end, and values in between during playback.
func (p *Player) GetProgress() float64 {
	duration := p.GetDuration()
	if duration == 0 {
		return 0
	}
	return p.GetPosition() / duration
}

// EventCount returns the total number of events in the recording.
func (p *Player) EventCount() int {
	return len(p.events)
}
