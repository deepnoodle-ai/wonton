# termsession

Terminal session recording and playback in asciinema v2 format. Records PTY I/O with timing information, supports compression, secret redaction, and playback control.

## Usage Examples

### Recording a Session

```go
package main

import (
	"log"

	"github.com/deepnoodle-ai/wonton/termsession"
)

func main() {
	// Create a session that records a bash shell
	session, err := termsession.NewSession(termsession.SessionOptions{
		Command: []string{"bash"},
	})
	if err != nil {
		log.Fatal(err)
	}
	defer session.Close()

	// Start recording to file
	opts := termsession.RecordingOptions{
		Compress:      true,  // gzip compression
		RedactSecrets: true,  // redact passwords/tokens
		Title:         "Demo Session",
		IdleTimeLimit: 2.0,   // cap idle time at 2 seconds
	}

	if err := session.Record("session.cast", opts); err != nil {
		log.Fatal(err)
	}

	// Wait for session to complete
	if err := session.Wait(); err != nil {
		log.Printf("Session exited with error: %v", err)
	}

	log.Printf("Exit code: %d", session.ExitCode())
}
```

### Standalone Recording

```go
// Record output from any source
recorder, err := termsession.NewRecorder("output.cast", 80, 24, termsession.RecordingOptions{
	Compress: true,
})
if err != nil {
	log.Fatal(err)
}
defer recorder.Close()

// Record terminal output
recorder.RecordOutput("Hello, World!\r\n")
recorder.RecordOutput("\x1b[32mGreen text\x1b[0m\r\n")

// Pause/resume recording
recorder.Pause()
// ... do something you don't want recorded ...
recorder.Resume()

// Flush to disk
recorder.Flush()
```

### Playing Back a Recording

```go
package main

import (
	"log"

	"github.com/deepnoodle-ai/wonton/termsession"
)

func main() {
	// Load and play a recording
	player, err := termsession.NewPlayer("session.cast", termsession.PlayerOptions{
		Speed:   1.5,    // 1.5x speed
		Loop:    false,  // don't loop
		MaxIdle: 2.0,    // cap idle time at 2 seconds
	})
	if err != nil {
		log.Fatal(err)
	}

	// Get recording metadata
	header := player.GetHeader()
	log.Printf("Recording: %s (%dx%d)", header.Title, header.Width, header.Height)
	log.Printf("Duration: %.2f seconds", player.GetDuration())

	// Play blocks until complete
	if err := player.Play(); err != nil {
		log.Fatal(err)
	}
}
```

### Interactive Playback Control

```go
// Create player
player, err := termsession.NewPlayer("session.cast", termsession.PlayerOptions{
	Speed: 1.0,
})
if err != nil {
	log.Fatal(err)
}

// Start playback in goroutine
go func() {
	if err := player.Play(); err != nil {
		log.Printf("Playback error: %v", err)
	}
}()

// Control playback
time.Sleep(2 * time.Second)
player.Pause()

time.Sleep(1 * time.Second)
player.Resume()

// Change speed on the fly
player.SetSpeed(2.0)

// Seek to position
player.Seek(10.0) // jump to 10 seconds

// Stop playback
player.Stop()
```

### Loading and Analyzing Recordings

```go
// Load recording for analysis
header, events, err := termsession.LoadCastFile("session.cast")
if err != nil {
	log.Fatal(err)
}

log.Printf("Terminal size: %dx%d", header.Width, header.Height)
log.Printf("Recorded: %s", time.Unix(header.Timestamp, 0))
log.Printf("Total events: %d", len(events))

// Calculate duration
duration := termsession.Duration(events)
log.Printf("Duration: %.2f seconds", duration)

// Filter to output events only
outputEvents := termsession.OutputEvents(events)
log.Printf("Output events: %d", len(outputEvents))

// Process events
for _, event := range outputEvents {
	log.Printf("[%.3fs] %s", event.Time, event.Data)
}
```

## API Reference

### Session Types

| Type | Description |
|------|-------------|
| `Session` | Interactive PTY session with recording capability |
| `SessionOptions` | Configuration for creating a session |
| `RecordingOptions` | Configuration for recording behavior |
| `Recorder` | Standalone recorder for terminal output |
| `Player` | Playback engine with control capabilities |
| `PlayerOptions` | Configuration for playback behavior |
| `RecordingHeader` | Metadata from asciinema v2 header |
| `RecordingEvent` | Single recording event with timing |

### Session Functions

| Function | Description | Inputs | Outputs |
|----------|-------------|--------|---------|
| `NewSession` | Creates a new PTY session | `SessionOptions` | `*Session, error` |
| `Session.Record` | Starts session and records to file | `filename string, opts RecordingOptions` | `error` |
| `Session.Start` | Starts session without recording | none | `error` |
| `Session.Wait` | Blocks until session ends | none | `error` |
| `Session.Close` | Terminates session and cleans up | none | `error` |
| `Session.ExitCode` | Returns exit code after Wait | none | `int` |
| `Session.Resize` | Changes terminal dimensions | `width, height int` | `error` |
| `Session.PauseRecording` | Pauses recording | none | none |
| `Session.ResumeRecording` | Resumes recording | none | none |
| `Session.IsRecording` | Checks if recording is active | none | `bool` |

### Recorder Functions

| Function | Description | Inputs | Outputs |
|----------|-------------|--------|---------|
| `NewRecorder` | Creates standalone recorder | `filename string, width, height int, opts RecordingOptions` | `*Recorder, error` |
| `RecordOutput` | Records terminal output | `data string` | none |
| `RecordInput` | Records user input | `data string` | none |
| `Pause` | Pauses recording | none | none |
| `Resume` | Resumes recording | none | none |
| `IsPaused` | Checks pause state | none | `bool` |
| `UpdateSize` | Updates terminal dimensions | `width, height int` | none |
| `Flush` | Writes buffered data to disk | none | `error` |
| `Close` | Finalizes recording | none | `error` |

### Player Functions

| Function | Description | Inputs | Outputs |
|----------|-------------|--------|---------|
| `NewPlayer` | Creates player from file | `filename string, opts PlayerOptions` | `*Player, error` |
| `Play` | Starts playback (blocks) | none | `error` |
| `Pause` | Pauses playback | none | none |
| `Resume` | Resumes playback | none | none |
| `TogglePause` | Toggles pause state | none | none |
| `IsPaused` | Checks pause state | none | `bool` |
| `Stop` | Stops playback completely | none | none |
| `SetSpeed` | Changes playback speed | `speed float64` | none |
| `Speed` | Gets current speed | none | `float64` |
| `SetLoop` | Enables/disables looping | `loop bool` | none |
| `Seek` | Jumps to time offset | `seconds float64` | none |
| `GetHeader` | Returns recording metadata | none | `RecordingHeader` |
| `GetDuration` | Returns total duration | none | `float64` |
| `GetPosition` | Returns current position | none | `float64` |
| `GetProgress` | Returns progress (0.0-1.0) | none | `float64` |
| `EventCount` | Returns total events | none | `int` |

### Utility Functions

| Function | Description | Inputs | Outputs |
|----------|-------------|--------|---------|
| `LoadCastFile` | Loads recording from file | `filename string` | `*RecordingHeader, []RecordingEvent, error` |
| `LoadCast` | Loads recording from reader | `r io.Reader` | `*RecordingHeader, []RecordingEvent, error` |
| `Duration` | Calculates total duration | `events []RecordingEvent` | `float64` |
| `OutputEvents` | Filters to output events only | `events []RecordingEvent` | `[]RecordingEvent` |
| `DefaultRecordingOptions` | Returns default recording options | none | `RecordingOptions` |
| `DefaultPlayerOptions` | Returns default player options | none | `PlayerOptions` |

## Related Packages

- [terminal](../terminal) - Low-level terminal control and ANSI sequences
- [termtest](../termtest) - Terminal output testing with screen simulation
- [tui](../tui) - Declarative terminal UI framework
