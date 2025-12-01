# Session Playback Demo

This example demonstrates playing back asciinema v2 recordings created with Gooey.

## Usage

```bash
go run main.go <recording.cast>
```

For example, after creating a recording with the recording demo:

```bash
# First, create a recording
cd ../recording_demo
go run main.go

# Then play it back
cd ../playback_demo
go run main.go ../recording_demo/demo.cast
```

## What It Does

1. Loads the `.cast` file (auto-detects gzip compression)
2. Displays recording metadata (title, size, duration)
3. Waits for you to press Enter
4. Plays back the recording with accurate timing
5. Shows "Playback complete!" when done

## Playback Features

The `PlaybackController` provides:
- ✅ Accurate timing playback
- ✅ Speed control (1.0x, 2.0x, etc.)
- ✅ Pause/Resume
- ✅ Seeking to timestamps
- ✅ Loop mode
- ✅ Stop playback

## API Example

```go
// Load recording
controller, _ := tui.LoadRecording("demo.cast")

// Get metadata
header := controller.GetHeader()
duration := controller.GetDuration()

// Control playback
controller.SetSpeed(2.0)   // 2x speed
controller.Pause()
controller.Resume()
controller.Seek(30.5)      // Jump to 30.5 seconds

// Play it
terminal, _ := tui.NewTerminal()
controller.Play(terminal)
```

## Advanced Playback

For interactive playback with keyboard controls, see the full implementation in `main.go` which demonstrates:
- Real-time speed adjustment
- Pause/resume toggle
- Progress display
- Looping

## Compatible Files

This player works with:
- Gooey recordings (`.cast`)
- Any asciinema v2 format files
- Compressed (gzip) or uncompressed
- Recordings from other asciinema tools

## See Also

- [Recording Demo](../recording_demo/) - Create recordings
- [Recording Documentation](../../documentation/recording.md) - Complete guide
- [asciinema Format Spec](https://github.com/asciinema/asciinema/blob/develop/doc/asciicast-v2.md)
