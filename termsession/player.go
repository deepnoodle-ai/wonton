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

// Player plays back recorded terminal sessions
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

// PlayerOptions configures playback behavior
type PlayerOptions struct {
	Speed   float64   // Playback speed multiplier (default: 1.0)
	Loop    bool      // Loop playback when finished
	MaxIdle float64   // Max idle time between events (0 = preserve original)
	Output  io.Writer // Output destination (default: os.Stdout)
}

// DefaultPlayerOptions returns sensible defaults
func DefaultPlayerOptions() PlayerOptions {
	return PlayerOptions{
		Speed:   1.0,
		Loop:    false,
		MaxIdle: 0,
		Output:  os.Stdout,
	}
}

// NewPlayer loads a recording file and creates a player
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

// Play starts playback of the recording (blocks until complete or stopped)
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

// Pause pauses playback
func (p *Player) Pause() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.paused {
		p.paused = true
		p.pauseTime = time.Now()
	}
}

// Resume resumes playback
func (p *Player) Resume() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.paused {
		p.paused = false
		pauseDuration := time.Since(p.pauseTime)
		p.totalPaused += pauseDuration
	}
}

// TogglePause toggles pause/resume
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

// IsPaused returns true if playback is paused
func (p *Player) IsPaused() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.paused
}

// Stop stops playback completely
func (p *Player) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.stopped {
		p.stopped = true
		close(p.stopChan)
	}
}

// SetSpeed sets the playback speed multiplier
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

// Speed returns the current playback speed
func (p *Player) Speed() float64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.speed
}

// SetLoop enables or disables looping
func (p *Player) SetLoop(loop bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.loop = loop
}

// Seek jumps to a specific time offset in the recording
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

// GetHeader returns the recording metadata
func (p *Player) GetHeader() RecordingHeader {
	return p.header
}

// GetDuration returns the total duration of the recording in seconds
func (p *Player) GetDuration() float64 {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(p.events) == 0 {
		return 0
	}
	return p.events[len(p.events)-1].Time
}

// GetPosition returns the current playback position in seconds
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

// GetProgress returns playback progress as a value between 0.0 and 1.0
func (p *Player) GetProgress() float64 {
	duration := p.GetDuration()
	if duration == 0 {
		return 0
	}
	return p.GetPosition() / duration
}

// EventCount returns the total number of events in the recording
func (p *Player) EventCount() int {
	return len(p.events)
}
