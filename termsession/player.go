package termsession

import (
	"errors"
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
	events         []RecordingEvent // original events
	adjustedEvents []RecordingEvent // events with maxIdle applied (nil if maxIdle <= 0)
	header         RecordingHeader
	output         io.Writer
	currentIndex   int
	startTime      time.Time
	pauseTime      time.Time
	totalPaused    time.Duration
	paused         bool
	speed          float64
	loop           bool
	maxIdle        float64
	mu             sync.RWMutex
	stopChan       chan struct{}
	stopped        bool
	playing        bool
	hasInitialSeek bool
	initialSeek    float64
	seekGen        uint64 // incremented by Seek so Play does not overwrite it
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
	header, events, err := LoadCastFile(filename)
	if err != nil {
		return nil, err
	}

	output := opts.Output
	if output == nil {
		output = os.Stdout
	}

	speed := opts.Speed
	if speed <= 0 {
		speed = 1.0
	}

	p := &Player{
		events:   events,
		header:   *header,
		output:   output,
		speed:    speed,
		loop:     opts.Loop,
		maxIdle:  opts.MaxIdle,
		stopChan: make(chan struct{}),
	}

	// Precompute adjusted events if maxIdle is configured
	if opts.MaxIdle > 0 {
		p.adjustedEvents = p.applyMaxIdle(events)
	}

	return p, nil
}

// ErrPlayerStopped is returned when Play() is called on a stopped player.
var ErrPlayerStopped = errors.New("player has been stopped and cannot be restarted")

// Play starts playback of the recording.
//
// This method blocks until playback completes, is stopped via Stop(),
// or an error occurs. Events are written to the configured output writer
// with timing preserved according to the speed multiplier.
//
// If Loop is enabled, playback will restart from the beginning when it
// reaches the end. Call Stop() from another goroutine to end looped playback.
//
// After Stop() is called, the player cannot be restarted. Create a new
// player if you need to play the recording again.
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
	if p.stopped {
		p.mu.Unlock()
		return ErrPlayerStopped
	}
	now := time.Now()
	if p.hasInitialSeek {
		p.startTime = now.Add(-time.Duration((p.initialSeek / p.speed) * float64(time.Second)))
		p.hasInitialSeek = false
	} else {
		p.startTime = now
		p.currentIndex = 0
	}
	p.paused = false
	p.totalPaused = 0
	p.playing = true
	p.mu.Unlock()

	defer func() {
		p.mu.Lock()
		p.playing = false
		p.mu.Unlock()
	}()

	// Use precomputed adjusted events if maxIdle was configured
	events := p.events
	if p.adjustedEvents != nil {
		events = p.adjustedEvents
	}

	// Nothing to play; avoid busy-spinning when Loop is enabled.
	if len(events) == 0 {
		return nil
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

		if p.paused {
			p.mu.RUnlock()
			time.Sleep(50 * time.Millisecond)
			continue
		}

		event := events[p.currentIndex]
		speed := p.speed
		currentIndex := p.currentIndex
		seekGen := p.seekGen
		elapsed := p.elapsedLocked()
		p.mu.RUnlock()

		// Calculate when this event should fire
		targetTime := event.Time / speed

		if elapsed < targetTime {
			sleepDuration := time.Duration((targetTime - elapsed) * float64(time.Second))
			// Only sleep if duration is meaningful (>= 10ms)
			const minSleepDuration = 10 * time.Millisecond
			if sleepDuration >= minSleepDuration {
				// Sleep in bounded slices so Pause, Seek, SetSpeed, and Stop
				// take effect promptly even during long idle gaps.
				const maxSleepSlice = 50 * time.Millisecond
				if sleepDuration > maxSleepSlice {
					sleepDuration = maxSleepSlice
				}
				select {
				case <-p.stopChan:
					return nil
				case <-time.After(sleepDuration):
				}
				// Re-evaluate state (pause/seek/speed) before writing
				continue
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
		// The batch was written without the lock held. If Seek ran in the
		// meantime, keep the position it chose.
		if p.seekGen == seekGen {
			p.currentIndex = batchIndex
		}
		p.mu.Unlock()
	}
}

// applyMaxIdle adjusts event times to cap idle periods
func (p *Player) applyMaxIdle(events []RecordingEvent) []RecordingEvent {
	if len(events) == 0 {
		return events
	}

	result := make([]RecordingEvent, len(events))
	var adjustedTime float64
	var prevTime float64

	for i, event := range events {
		gap := event.Time - prevTime
		if gap > p.maxIdle {
			gap = p.maxIdle
		}
		adjustedTime += gap
		prevTime = event.Time

		result[i] = RecordingEvent{
			Time: adjustedTime,
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

// activeEvents returns the events slice to use for playback
// (adjustedEvents if maxIdle was configured, otherwise events).
func (p *Player) activeEvents() []RecordingEvent {
	if p.adjustedEvents != nil {
		return p.adjustedEvents
	}
	return p.events
}

// elapsedLocked returns the playback-clock seconds elapsed since Play started,
// excluding paused time (including the in-flight pause, if currently paused).
// Caller must hold p.mu (read or write).
func (p *Player) elapsedLocked() float64 {
	// Read the clock once, so a paused player reports a steady value.
	now := time.Now()
	elapsed := now.Sub(p.startTime) - p.totalPaused
	if p.paused {
		elapsed -= now.Sub(p.pauseTime)
	}
	return elapsed.Seconds()
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

	if p.playing {
		// Adjust the start time so the current position in the recording is
		// preserved: position (in recording seconds) is elapsed * speed, and it
		// must be unchanged under the new speed.
		elapsed := p.elapsedLocked()
		if elapsed > 0 {
			position := elapsed * p.speed
			adjustment := elapsed - position/speed
			p.startTime = p.startTime.Add(time.Duration(adjustment * float64(time.Second)))
		}
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
// Playback continues with the first event at or after the target time.
// This can be called before or during playback.
func (p *Player) Seek(seconds float64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if seconds < 0 {
		seconds = 0
	}

	events := p.activeEvents()

	// Play next the first event at or after the target time. Events before
	// it have already elapsed at that point in the recording.
	targetIndex := len(events)
	for i, event := range events {
		if event.Time >= seconds {
			targetIndex = i
			break
		}
	}

	p.currentIndex = targetIndex
	p.seekGen++
	if p.playing {
		// Adjust start time to account for the seek
		elapsed := p.elapsedLocked()
		adjustment := elapsed - (seconds / p.speed)
		p.startTime = p.startTime.Add(time.Duration(adjustment * float64(time.Second)))
	} else {
		p.initialSeek = seconds
		p.hasInitialSeek = true
	}
}

// GetHeader returns the recording metadata.
//
// This includes terminal dimensions, title, timestamp, and environment variables.
func (p *Player) GetHeader() RecordingHeader {
	return p.header
}

// GetDuration returns the total duration of the recording in seconds.
//
// If MaxIdle was configured, returns the adjusted duration (with idle times capped).
// Returns 0 if the recording has no events.
func (p *Player) GetDuration() float64 {
	p.mu.RLock()
	defer p.mu.RUnlock()

	events := p.activeEvents()
	if len(events) == 0 {
		return 0
	}
	return events[len(events)-1].Time
}

// GetPosition returns the current playback position in seconds.
//
// During playback this is the time on the playback clock, which advances
// through idle gaps between events. Otherwise it is the position Seek chose,
// or the time of the last event played. If MaxIdle was configured, returns
// the adjusted position.
func (p *Player) GetPosition() float64 {
	p.mu.RLock()
	defer p.mu.RUnlock()

	events := p.activeEvents()
	if len(events) == 0 {
		return 0
	}
	duration := events[len(events)-1].Time
	switch {
	case p.playing:
		return min(max(p.elapsedLocked()*p.speed, 0), duration)
	case p.hasInitialSeek:
		return min(p.initialSeek, duration)
	case p.currentIndex == 0:
		return 0
	}
	return events[min(p.currentIndex, len(events))-1].Time
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
