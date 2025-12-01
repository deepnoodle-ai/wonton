# Session Recording and Playback

Gooey provides built-in support for recording and playing back terminal sessions using the industry-standard **asciinema v2** format (.cast files). This enables you to:

- Create demos and tutorials
- Record bugs for reproduction
- Document complex workflows
- Test applications with recorded inputs
- Share terminal sessions with others

## Table of Contents

- [Recording Sessions](#recording-sessions)
- [Playing Back Recordings](#playing-back-recordings)
- [File Format](#file-format)
- [Privacy and Security](#privacy-and-security)
- [Advanced Features](#advanced-features)
- [API Reference](#api-reference)
- [Examples](#examples)

## Recording Sessions

### Basic Recording

To record a terminal session, call `StartRecording()` before your main event loop and `StopRecording()` when done:

```go
terminal, _ := tui.NewTerminal()
defer terminal.Restore()

// Start recording
opts := tui.DefaultRecordingOptions()
opts.Title = "My Demo"
err := terminal.StartRecording("demo.cast", opts)
if err != nil {
    log.Fatal(err)
}
defer terminal.StopRecording()

// Your normal TUI code here
// All output and input will be recorded automatically
```

### Recording Options

The `RecordingOptions` struct controls how recordings are made:

```go
type RecordingOptions struct {
    Compress      bool              // Enable gzip compression (default: true)
    RedactSecrets bool              // Redact passwords/tokens (default: true)
    Title         string            // Recording title (metadata)
    Env           map[string]string // Environment variables (metadata)
    IdleTimeLimit float64           // Max idle time in seconds (0 = unlimited)
}
```

**Example with custom options:**

```go
opts := tui.RecordingOptions{
    Compress:      true,
    RedactSecrets: true,
    Title:         "User Onboarding Flow",
    Env: map[string]string{
        "TERM": os.Getenv("TERM"),
        "SHELL": "/bin/bash",
    },
    IdleTimeLimit: 2.0, // Collapse idle times > 2s
}

terminal.StartRecording("onboarding.cast", opts)
```

### What Gets Recorded

#### Output Recording
Recording captures each **logical write operation** (`Print()`, `Println()`, etc.) with its original timing, regardless of whether you use frame-based rendering or direct output.

All terminal output is captured, including:
- Text content and positioning
- ANSI escape sequences (colors, formatting)
- Cursor movements
- Timing between operations

**How it works:**
- Each call to `Print()` / `Println()` is recorded when called, not when flushed
- Frame-based rendering (`BeginFrame` / `EndFrame`) works naturally - the timing of your `Print()` calls is preserved
- Delays between print operations are captured accurately

**Example - Frame-based rendering (recommended):**
```go
terminal.Println("First line")
time.Sleep(500 * time.Millisecond)
terminal.Println("Second line appears after 500ms")
time.Sleep(200 * time.Millisecond)
terminal.Println("Third line appears after 200ms more")

// Recording captures the timing between these Print() calls,
// even though they may all flush together in one frame
```

**Example - Multiple operations with timing:**
```go
for i := 0; i < 5; i++ {
    terminal.Printf("Line %d\n", i)
    time.Sleep(100 * time.Millisecond)
}
// Playback will show each line appearing with 100ms delay
```

**Note:** Frame-based rendering (`BeginFrame/EndFrame`) batches terminal writes for efficiency, but recordings still capture the timing of each `Print()` call within the frame.

#### Input Recording
User input events are captured as they occur:
- Keyboard input (characters, special keys)
- Timing information
- Control sequences (Ctrl+C, etc.)

**Note:** Mouse events are not currently recorded.

### Pausing Recording

For sensitive sections (like password entry), you can pause recording:

```go
terminal.PauseRecording()
password := input.ReadPassword("Enter password: ")
terminal.ResumeRecording()
```

While paused, no output or input is recorded. Resume to continue recording.

## Playing Back Recordings

### Basic Playback

Load and play a recording with the `PlaybackController`:

```go
// Load recording
controller, err := tui.LoadRecording("demo.cast")
if err != nil {
    log.Fatal(err)
}

// Create terminal for playback
terminal, _ := tui.NewTerminal()
defer terminal.Restore()

// Play it back
err = controller.Play(terminal)
if err != nil {
    log.Fatal(err)
}
```

### Playback Controls

The `PlaybackController` provides full control over playback:

```go
// Pause/resume
controller.Pause()
controller.Resume()
controller.TogglePause()

// Speed control (1.0 = normal, 2.0 = 2x, 0.5 = half speed)
controller.SetSpeed(2.0)
currentSpeed := controller.Speed()

// Seek to position (in seconds)
controller.Seek(30.5)

// Loop mode
controller.SetLoop(true)

// Stop playback
controller.Stop()
```

### Playback Metadata

Access recording information:

```go
header := controller.GetHeader()
fmt.Printf("Title: %s\n", header.Title)
fmt.Printf("Terminal size: %dx%d\n", header.Width, header.Height)
fmt.Printf("Timestamp: %d\n", header.Timestamp)

duration := controller.GetDuration()    // Total duration in seconds
position := controller.GetPosition()    // Current position in seconds
progress := controller.GetProgress()    // Progress as 0.0-1.0
```

### Interactive Playback Example

```go
controller, _ := tui.LoadRecording("demo.cast")
terminal, _ := tui.NewTerminal()
defer terminal.Restore()

// Enable raw mode for keyboard controls
terminal.EnableRawMode()
defer terminal.DisableRawMode()

// Start playback in background
go controller.Play(terminal)

// Control loop
input := tui.NewInput(terminal)
for {
    event, _ := input.ReadKeyWithTimeout(100 * time.Millisecond)

    switch event.Rune {
    case ' ':
        controller.TogglePause()
    case '+':
        controller.SetSpeed(controller.Speed() + 0.5)
    case '-':
        controller.SetSpeed(controller.Speed() - 0.5)
    case 'q':
        controller.Stop()
        return
    }
}
```

## File Format

Recordings use the **asciinema v2** format, which is:
- **Line-oriented JSON** (newline-delimited)
- **Human-readable** (plain text)
- **Compact** (especially with gzip)
- **Widely supported** (asciinema.org, players, etc.)

### File Structure

**Header (first line):**
```json
{"version": 2, "width": 80, "height": 24, "timestamp": 1234567890, "env": {"TERM": "xterm-256color"}, "title": "My Demo"}
```

**Events (subsequent lines):**
```json
[0.123, "o", "Hello, World!"]
[1.456, "o", "\u001b[1;32mColored text\u001b[0m"]
[2.789, "i", "ls -la\r"]
```

Each event is: `[time_offset, event_type, data]`
- `time_offset`: Seconds since recording started (float)
- `event_type`: `"o"` (output) or `"i"` (input)
- `data`: String content (terminal output or user input)

### Compression

By default, recordings are gzip-compressed (`.cast` or `.cast.gz`). This typically achieves 70-90% size reduction for terminal output with lots of repetition.

Enable/disable compression:
```go
opts := tui.DefaultRecordingOptions()
opts.Compress = true  // or false
terminal.StartRecording("demo.cast", opts)
```

The playback system auto-detects gzip compression by checking file magic bytes.

## Privacy and Security

### Secret Redaction

When `RedactSecrets` is enabled (default), the recording system automatically redacts common secrets:

**Detected patterns:**
- Passwords: `password: ******`
- API keys: `api_key: ******`
- Tokens: `token: ******`, `Bearer ******`
- Long hex strings (likely tokens)
- JWT tokens
- AWS/GitHub keys
- Generic `secret=******` patterns

**Example:**

```go
// Before redaction (what you type)
"api_key=sk_live_1234567890abcdef"

// After redaction (what's recorded)
"api_key=[REDACTED]"
```

### Manual Redaction

For additional control:

```go
// Pause recording for sensitive operations
terminal.PauseRecording()
input.WithPrompt("Enter API key: ", tui.NewStyle())
apiKey, _ := input.ReadSimple()
terminal.ResumeRecording()

// Or test redaction
redacted := tui.RedactCredentials("password=secret123")
// Returns: "password=[REDACTED]"
```

### Best Practices

1. **Review before sharing** - Always review recordings before uploading
2. **Use pause/resume** - Pause during credential entry
3. **Enable redaction** - Keep `RedactSecrets: true` (default)
4. **Limit metadata** - Don't include sensitive info in title/env
5. **Secure storage** - Store recordings securely; they may contain app state

### What Redaction Doesn't Protect

⚠️ **Limitations:**
- Screen content isn't redacted (only input/output streams)
- App-specific secrets that don't match patterns
- Secrets embedded in binary data or images
- Context that reveals sensitive information

For maximum security, use `PauseRecording()` during sensitive operations.

## Advanced Features

### Idle Time Limiting

Collapse long pauses to keep recordings concise:

```go
opts := tui.DefaultRecordingOptions()
opts.IdleTimeLimit = 2.0  // Max 2 seconds between events

terminal.StartRecording("demo.cast", opts)

// If user idles for 10 seconds, only 2 seconds will be recorded
// This keeps recordings shorter for demos
```

### Loop Mode

Continuously loop playback:

```go
controller, _ := tui.LoadRecording("demo.cast")
controller.SetLoop(true)
controller.Play(terminal) // Plays forever until stopped
```

### Speed Control

Adjust playback speed dynamically:

```go
controller.SetSpeed(2.0)   // 2x speed (faster)
controller.SetSpeed(0.5)   // 0.5x speed (slower)
controller.SetSpeed(1.0)   // Normal speed
```

### Seeking

Jump to specific timestamps:

```go
controller.Seek(30.5)  // Jump to 30.5 seconds
controller.Seek(0)     // Restart from beginning
```

## API Reference

### Terminal Methods

```go
// StartRecording begins recording a session
func (t *Terminal) StartRecording(filename string, opts RecordingOptions) error

// StopRecording finalizes and closes the recording
func (t *Terminal) StopRecording() error

// PauseRecording temporarily suspends recording
func (t *Terminal) PauseRecording()

// ResumeRecording resumes a paused recording
func (t *Terminal) ResumeRecording()

// IsRecording returns true if a recording is active
func (t *Terminal) IsRecording() bool
```

### PlaybackController Methods

```go
// LoadRecording loads a .cast file
func LoadRecording(filename string) (*PlaybackController, error)

// Play starts playback of the recording
func (p *PlaybackController) Play(terminal *Terminal) error

// Pause/Resume
func (p *PlaybackController) Pause()
func (p *PlaybackController) Resume()
func (p *PlaybackController) TogglePause()
func (p *PlaybackController) IsPaused() bool

// Speed control
func (p *PlaybackController) SetSpeed(speed float64)
func (p *PlaybackController) Speed() float64

// Seeking
func (p *PlaybackController) Seek(seconds float64)

// Loop mode
func (p *PlaybackController) SetLoop(loop bool)

// Stop playback
func (p *PlaybackController) Stop()

// Metadata
func (p *PlaybackController) GetHeader() RecordingHeader
func (p *PlaybackController) GetDuration() float64
func (p *PlaybackController) GetPosition() float64
func (p *PlaybackController) GetProgress() float64
```

### RecordingOptions

```go
type RecordingOptions struct {
    Compress      bool              // Enable gzip compression
    RedactSecrets bool              // Redact passwords/tokens
    Title         string            // Recording title
    Env           map[string]string // Environment variables
    IdleTimeLimit float64           // Max idle time (0 = unlimited)
}

// DefaultRecordingOptions returns sensible defaults
func DefaultRecordingOptions() RecordingOptions
```

### RecordingHeader

```go
type RecordingHeader struct {
    Version   int               `json:"version"`
    Width     int               `json:"width"`
    Height    int               `json:"height"`
    Timestamp int64             `json:"timestamp"`
    Env       map[string]string `json:"env,omitempty"`
    Title     string            `json:"title,omitempty"`
}
```

## Examples

### Full Recording Example

See `examples/recording_demo/main.go` for a complete demo that shows:
- Starting a recording
- Animated output (rainbow text)
- User input capture
- Proper cleanup with defer

Run it:
```bash
cd examples/recording_demo
go run main.go
# Type some text, press 'q' to finish
# Creates demo.cast file
```

### Full Playback Example

See `examples/playback_demo/main.go` for a complete player with:
- Loading recordings
- Interactive controls (pause, speed, seek)
- Status display
- Keyboard shortcuts

Run it:
```bash
cd examples/playback_demo
go run main.go ../recording_demo/demo.cast
# Use Space, +/-, q to control playback
```

### Minimal Recording

```go
package main

import "github.com/myzie/gooey"

func main() {
    terminal, _ := tui.NewTerminal()
    defer terminal.Restore()

    opts := tui.DefaultRecordingOptions()
    terminal.StartRecording("session.cast", opts)
    defer terminal.StopRecording()

    terminal.Println("Hello, World!")
    terminal.Flush()
}
```

### Minimal Playback

```go
package main

import "github.com/myzie/gooey"

func main() {
    controller, _ := tui.LoadRecording("session.cast")
    terminal, _ := tui.NewTerminal()
    defer terminal.Restore()

    controller.Play(terminal)
}
```

## Sharing Recordings

### Upload to asciinema.org

You can upload recordings to asciinema.org for easy sharing:

```bash
# Install asciinema CLI
brew install asciinema  # or apt/yum

# Upload
asciinema upload demo.cast
```

### Embed in Documentation

Many tools support asciinema v2 format:
- [asciinema-player](https://github.com/asciinema/asciinema-player) - Web player
- [asciinema.org](https://asciinema.org) - Hosting platform
- Markdown renderers (GitHub, GitLab with extensions)

### Convert to GIF

Use [agg](https://github.com/asciinema/agg) or [asciicast2gif](https://github.com/asciinema/asciicast2gif):

```bash
agg demo.cast demo.gif
```

## Troubleshooting

### Recording is empty
- Ensure you call `StopRecording()` to flush buffer
- Check file permissions
- Verify terminal has output

### Playback timing doesn't match expectations
Recording captures the timing of **when your code calls** `Print()` / `Println()`, not when frames flush. If you want specific timing:

```go
terminal.Println("First")
time.Sleep(1 * time.Second)  // ← This delay is captured
terminal.Println("Second")
```

If all your prints happen in rapid succession, playback will be rapid. Add delays between prints to control playback timing.

### Playback looks wrong
- Terminal size mismatch (recording: 80x24, playback: 120x40)
- ANSI escape codes not supported in playback terminal
- Compression format issue (file corrupted?)

### Secrets not redacted
- Pattern doesn't match default regex
- Add custom redaction or use `PauseRecording()`
- Review `recording_privacy.go` patterns

### File too large
- Enable compression: `opts.Compress = true`
- Use idle time limiting: `opts.IdleTimeLimit = 2.0`
- Recordings with lots of animation can be large

## Technical Details

### Performance

Recording has minimal overhead:
- **Output**: ~1% CPU (buffered writes)
- **Input**: Negligible (rare events)
- **Disk I/O**: Async writes, buffered

### Thread Safety

Recording is thread-safe:
- Terminal uses mutex for recorder access
- Recorder uses internal mutex for writes
- Safe to record from multiple goroutines

### Memory Usage

Memory scales with buffering:
- Default: 4KB buffer per writer
- Gzip: 32KB compression window
- Peak: ~40-50KB per active recording

### Limitations

Current limitations:
- Mouse events not recorded
- Terminal resize events not captured
- Binary data (images) recorded as ANSI but may not replay correctly
- Very high FPS animations may create large files

## Future Enhancements

Planned features:
- Editing tools (cut, trim, merge)
- Annotations (add text overlays)
- Direct asciinema.org upload
- HTML player generation
- Programmatic assertions for testing

## See Also

- [Animations Guide](animations.md) - Creating animated TUIs
- [Input Guide](INPUT_GUIDE.md) - Handling user input
- [Metrics Guide](metrics.md) - Performance profiling
- [asciinema v2 format](https://github.com/asciinema/asciinema/blob/develop/doc/asciicast-v2.md) - Official specification
