# Session Recording Demo

This example demonstrates Gooey's session recording capabilities using the asciinema v2 format.

## Running the Demo

```bash
go run main.go
```

This will:
1. Start recording to `demo.cast`
2. Display a colored terminal UI with timed output
3. Prompt you to enter text three times
4. Save the recording when complete

## What's Being Recorded

- All terminal output (colors, formatting, positioning)
- **Timing between Print() calls** - delays are preserved
- User input (complete lines as entered)
- Natural pacing and pauses

## Key Features Demonstrated

### Timing Preservation
Recording captures the timing of when your code **calls Print()**, not when frames flush:

```go
terminal.Println("First line")
time.Sleep(300 * time.Millisecond)  // ← Captured in recording!
terminal.Println("Second line")
time.Sleep(150 * time.Millisecond)  // ← Also captured!
terminal.Println("Third line")
```

When you play back the recording, you'll see text appearing with the **original timing** between print statements.

## Playing Back

After creating a recording, play it back with:

```bash
go run ../playback_demo/main.go demo.cast
```

Or upload to asciinema.org:

```bash
# Install asciinema CLI first
brew install asciinema  # or apt/yum/pacman

# Upload
asciinema upload demo.cast
```

## Features Demonstrated

- ✅ Automatic output capture
- ✅ Input event recording
- ✅ gzip compression (default)
- ✅ Secret redaction (enabled by default)
- ✅ Colored terminal output
- ✅ Timing preservation

## File Format

The recording is saved in asciinema v2 format:
- First line: JSON header with metadata
- Remaining lines: JSON events `[timestamp, type, data]`
- Compressed with gzip (`.cast` extension)

## Code Highlights

```go
// Start recording
opts := gooey.DefaultRecordingOptions()
opts.Title = "My Demo"
terminal.StartRecording("demo.cast", opts)
defer terminal.StopRecording()

// Your normal TUI code here - everything is recorded automatically
```

See `main.go` for the full implementation.

## See Also

- [Recording Documentation](../../documentation/recording.md) - Complete guide
- [Playback Demo](../playback_demo/) - Play recordings
- [asciinema.org](https://asciinema.org) - Upload and share
