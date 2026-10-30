package tui

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

// maxRecordingLineSize caps one line of a .cast file. Full-screen frames can
// exceed bufio.Scanner's 64KB default.
const maxRecordingLineSize = 16 * 1024 * 1024

// pollInterval is the longest Play waits before it checks for Pause, Seek,
// and SetSpeed again.
const pollInterval = 50 * time.Millisecond

// PlaybackController manages playback of recorded sessions
type PlaybackController struct {
	events       []RecordingEvent
	header       RecordingHeader
	terminal     *Terminal
	currentIndex int
	startTime    time.Time
	pauseTime    time.Time
	totalPaused  time.Duration
	paused       bool
	speed        float64
	loop         bool
	mu           sync.RWMutex
	stopChan     chan struct{}
	stopped      bool
	seekTo       float64 // position set by a Seek before the first Play
}

// LoadRecording loads a .cast file and returns a playback controller
func LoadRecording(filename string) (*PlaybackController, error) {
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
	scanner.Buffer(make([]byte, 0, 64*1024), maxRecordingLineSize)

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
	lineNum := 1
	for scanner.Scan() {
		lineNum++
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

	return &PlaybackController{
		events:   events,
		header:   header,
		speed:    1.0,
		stopChan: make(chan struct{}),
	}, nil
}

// Play starts playback of the recording
func (p *PlaybackController) Play(terminal *Terminal) error {
	p.mu.Lock()
	if len(p.events) == 0 {
		p.mu.Unlock()
		return nil
	}
	if p.stopped {
		// A previous Stop closed the channel; give this run a fresh one.
		p.stopChan = make(chan struct{})
	}
	stopChan := p.stopChan
	p.terminal = terminal
	offset := 0.0
	if p.startTime.IsZero() {
		// The first Play keeps a Seek or Pause made before it.
		offset = p.seekTo
	} else {
		p.currentIndex = 0
		p.paused = false
	}
	now := time.Now()
	p.startTime = now.Add(-time.Duration(offset / p.speed * float64(time.Second)))
	p.pauseTime = now
	p.stopped = false
	p.totalPaused = 0
	p.mu.Unlock()

	for {
		// Check this run's own channel. A Stop followed by a new Play resets
		// p.stopped, but the closed channel still ends this loop.
		select {
		case <-stopChan:
			return nil
		default:
		}

		p.mu.RLock()
		if p.stopped {
			p.mu.RUnlock()
			return nil
		}

		if p.currentIndex >= len(p.events) {
			if p.loop {
				p.mu.RUnlock()
				p.mu.Lock()
				p.currentIndex = 0
				p.startTime = time.Now()
				p.totalPaused = 0
				if p.paused {
					// Count the pause from the restart, not from before it.
					p.pauseTime = p.startTime
				}
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
			select {
			case <-stopChan:
				return nil
			case <-time.After(pollInterval):
			}
			continue
		}

		p.mu.RLock()
		event := p.events[p.currentIndex]
		speed := p.speed
		currentIndex := p.currentIndex
		elapsed := p.elapsedLocked()
		p.mu.RUnlock()

		// Calculate when this event should fire
		targetTime := event.Time / speed

		if elapsed < targetTime {
			sleepDuration := time.Duration((targetTime - elapsed) * float64(time.Second))
			// Only sleep if duration is meaningful (>= 10ms)
			// This prevents choppy playback from sub-millisecond timing
			const minSleepDuration = 10 * time.Millisecond
			if sleepDuration >= minSleepDuration {
				// Wake at least every pollInterval, so Pause, Seek, and
				// SetSpeed take effect during a long gap between events.
				select {
				case <-stopChan:
					return nil
				case <-time.After(min(sleepDuration, pollInterval)):
				}
				// Re-evaluate the state before writing anything.
				continue
			}
		}

		// Batch output events that are very close together (< 10ms) to smooth playback
		// This prevents choppy "spurts" from rapid Print() calls
		const batchThreshold = 0.01 // 10ms in seconds
		var batchedOutput []string
		batchIndex := currentIndex

		p.mu.RLock()
		if p.paused || p.stopped || p.currentIndex != currentIndex {
			p.mu.RUnlock()
			continue
		}
		for batchIndex < len(p.events) {
			batchEvent := p.events[batchIndex]

			// Only batch output events, skip input events
			if batchEvent.Type == "o" {
				// Check if this event is within the batch window
				timeDiff := (batchEvent.Time - event.Time) / speed
				if batchIndex == currentIndex || timeDiff < batchThreshold {
					batchedOutput = append(batchedOutput, batchEvent.Data)
					batchIndex++
				} else {
					// Event is too far in the future, stop batching
					break
				}
			} else {
				// Skip input events (they're not replayed)
				// But stop batching if we've already collected some events
				if batchIndex > currentIndex {
					break
				}
				batchIndex++
			}
		}
		p.mu.RUnlock()

		// Write all batched output at once
		if len(batchedOutput) > 0 {
			for _, data := range batchedOutput {
				terminal.WriteRaw([]byte(data))
			}
		}

		// Input events ("i") are informational only, not replayed to the terminal

		p.mu.Lock()
		// Keep a position that Seek set while the output was written.
		if p.currentIndex == currentIndex {
			p.currentIndex = batchIndex
		}
		p.mu.Unlock()
	}
}

// Pause pauses playback
func (p *PlaybackController) Pause() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.paused {
		p.paused = true
		p.pauseTime = time.Now()
	}
}

// Resume resumes playback
func (p *PlaybackController) Resume() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.paused {
		p.paused = false
		pauseDuration := time.Since(p.pauseTime)
		p.totalPaused += pauseDuration
	}
}

// TogglePause toggles pause/resume
func (p *PlaybackController) TogglePause() {
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
func (p *PlaybackController) IsPaused() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.paused
}

// Stop stops playback completely
func (p *PlaybackController) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.stopped {
		p.stopped = true
		close(p.stopChan)
	}
}

// SetSpeed sets the playback speed multiplier (1.0 = normal, 2.0 = 2x, 0.5 = half speed)
func (p *PlaybackController) SetSpeed(speed float64) {
	if speed <= 0 {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.startTime.IsZero() {
		p.speed = speed // not started: there is no clock to adjust
		return
	}

	// Adjust timing to keep the recording position when speed changes.
	// This also applies while paused, so Resume continues from the same spot.
	elapsed := p.elapsedLocked()
	newElapsed := elapsed * p.speed / speed
	adjustment := elapsed - newElapsed
	p.startTime = p.startTime.Add(time.Duration(adjustment * float64(time.Second)))

	p.speed = speed
}

// Speed returns the current playback speed
func (p *PlaybackController) Speed() float64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.speed
}

// SetLoop enables or disables looping
func (p *PlaybackController) SetLoop(loop bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.loop = loop
}

// Seek jumps to a specific time offset in the recording
func (p *PlaybackController) Seek(seconds float64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Resume at the first event not yet due; past the end, none are left.
	targetIndex := len(p.events)
	for i, event := range p.events {
		if event.Time >= seconds {
			targetIndex = i
			break
		}
	}

	p.currentIndex = targetIndex
	if p.startTime.IsZero() {
		p.seekTo = seconds // not started: Play starts here
		return
	}
	// Adjust start time to account for the seek
	elapsed := p.elapsedLocked()
	adjustment := elapsed - (seconds / p.speed)
	p.startTime = p.startTime.Add(time.Duration(adjustment * float64(time.Second)))
}

// GetHeader returns the recording metadata
func (p *PlaybackController) GetHeader() RecordingHeader {
	return p.header
}

// GetDuration returns the total duration of the recording in seconds
func (p *PlaybackController) GetDuration() float64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.durationLocked()
}

// durationLocked returns the recording duration. Caller must hold p.mu.
func (p *PlaybackController) durationLocked() float64 {
	if len(p.events) == 0 {
		return 0
	}
	return p.events[len(p.events)-1].Time
}

// GetPosition returns the current playback position in seconds
func (p *PlaybackController) GetPosition() float64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.positionLocked()
}

// positionLocked returns the current position. Caller must hold p.mu.
func (p *PlaybackController) positionLocked() float64 {
	if p.currentIndex >= len(p.events) {
		return p.durationLocked()
	}
	if p.startTime.IsZero() {
		return min(max(p.seekTo, 0), p.durationLocked()) // not started
	}
	// Follow the playback clock, in recording time, not the time of the
	// next unplayed event.
	return min(max(p.elapsedLocked()*p.speed, 0), p.durationLocked())
}

// GetProgress returns playback progress as a value between 0.0 and 1.0
func (p *PlaybackController) GetProgress() float64 {
	p.mu.RLock()
	defer p.mu.RUnlock()

	duration := p.durationLocked()
	if duration == 0 {
		return 0
	}
	return p.positionLocked() / duration
}

// elapsedLocked returns the playback time elapsed in seconds, excluding
// pauses. While paused, the clock stops at the pause time, because Resume
// adds the current pause to totalPaused. Caller must hold p.mu.
func (p *PlaybackController) elapsedLocked() float64 {
	now := time.Now()
	if p.paused {
		now = p.pauseTime
	}
	return now.Sub(p.startTime).Seconds() - p.totalPaused.Seconds()
}
