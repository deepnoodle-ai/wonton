package gooey

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
	p.terminal = terminal
	p.startTime = time.Now()
	p.currentIndex = 0
	p.paused = false
	p.stopped = false
	p.totalPaused = 0
	p.mu.Unlock()

	for {
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
		event := p.events[p.currentIndex]
		speed := p.speed
		currentIndex := p.currentIndex
		p.mu.RUnlock()

		// Calculate when this event should fire
		targetTime := event.Time / speed
		elapsed := time.Since(p.startTime).Seconds() - p.totalPaused.Seconds()

		if elapsed < targetTime {
			sleepDuration := time.Duration((targetTime - elapsed) * float64(time.Second))
			// Only sleep if duration is meaningful (>= 10ms)
			// This prevents choppy playback from sub-millisecond timing
			const minSleepDuration = 10 * time.Millisecond
			if sleepDuration >= minSleepDuration {
				select {
				case <-p.stopChan:
					return nil
				case <-time.After(sleepDuration):
				}
			}
		}

		// Batch output events that are very close together (< 10ms) to smooth playback
		// This prevents choppy "spurts" from rapid Print() calls
		const batchThreshold = 0.01 // 10ms in seconds
		var batchedOutput []string
		batchIndex := currentIndex

		p.mu.RLock()
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
			terminal.mu.Lock()
			if terminal.out != nil {
				for _, data := range batchedOutput {
					terminal.out.Write([]byte(data))
				}
				// Try to sync if it's a File
				if f, ok := terminal.out.(*os.File); ok {
					f.Sync()
				}
			}
			terminal.mu.Unlock()
		}

		// Input events ("i") are informational only, not replayed to the terminal

		p.mu.Lock()
		p.currentIndex = batchIndex
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

	// Adjust timing to prevent jumps when speed changes
	if !p.paused {
		elapsed := time.Since(p.startTime).Seconds() - p.totalPaused.Seconds()
		currentEventTime := float64(0)
		if p.currentIndex < len(p.events) {
			currentEventTime = p.events[p.currentIndex].Time
		}

		// Calculate new start time to maintain position
		newElapsed := currentEventTime / speed
		adjustment := elapsed - newElapsed
		p.startTime = p.startTime.Add(time.Duration(adjustment * float64(time.Second)))
	}

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

	// Binary search to find the event closest to the target time
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
func (p *PlaybackController) GetHeader() RecordingHeader {
	return p.header
}

// GetDuration returns the total duration of the recording in seconds
func (p *PlaybackController) GetDuration() float64 {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(p.events) == 0 {
		return 0
	}
	return p.events[len(p.events)-1].Time
}

// GetPosition returns the current playback position in seconds
func (p *PlaybackController) GetPosition() float64 {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.currentIndex >= len(p.events) {
		return p.GetDuration()
	}
	if p.currentIndex == 0 {
		return 0
	}
	return p.events[p.currentIndex].Time
}

// GetProgress returns playback progress as a value between 0.0 and 1.0
func (p *PlaybackController) GetProgress() float64 {
	duration := p.GetDuration()
	if duration == 0 {
		return 0
	}
	return p.GetPosition() / duration
}
