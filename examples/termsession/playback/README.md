# Session Playback Demo

This example demonstrates playing back asciinema v2 recordings created with Wonton.

## Usage

```bash
go run ./examples/termsession/playback <recording.cast>
```

For example, after creating a recording with the `termrec` example:

```bash
# First, create a recording
go run ./examples/termrec record demo.cast

# Then play it back
go run ./examples/termsession/playback demo.cast
```

## What It Does

1. Loads the `.cast` file (auto-detects gzip compression)
2. Displays recording metadata (title, size, duration)
3. Waits for you to press Enter
4. Plays back the recording with accurate timing
5. Shows "Playback complete!" when done

## Playback Features

The `PlaybackController` provides:

- Accurate timing playback
- Speed control (1.0x, 2.0x, etc.)
- Pause/Resume
- Seeking to timestamps
- Loop mode
- Stop playback

This demo plays a recording straight through; the speed, pause, seek, and
loop controls are available programmatically from another goroutine.

## API Example

```go
// Load recording
controller, _ := tui.LoadRecording("demo.cast")

// Get metadata
header := controller.GetHeader()
duration := controller.GetDuration()

// Control playback (from another goroutine while Play is running)
controller.SetSpeed(2.0)   // 2x speed
controller.Pause()
controller.Resume()
controller.Seek(30.5)      // Jump to 30.5 seconds

// Play it
terminal, _ := tui.NewTerminal()
controller.Play(terminal)
```

## Compatible Files

This player works with:

- Wonton recordings (`.cast`)
- Any asciinema v2 format files
- Compressed (gzip) or uncompressed
- Recordings from other asciinema tools

## See Also

- [termrec](../../termrec/) - Record sessions and export to GIF
- [sessview](../../sessview/) - Browse and replay recordings in a TUI
- [asciinema Format Spec](https://github.com/asciinema/asciinema/blob/develop/doc/asciicast-v2.md)
