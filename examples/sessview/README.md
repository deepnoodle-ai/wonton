# sessview - Terminal Session Viewer

A TUI application for browsing and replaying asciinema-format terminal recordings.

## Features

- **Play recordings** with variable speed control
- **Browse directories** of recordings with an interactive TUI
- **View metadata** about recordings
- Loop playback support
- Max idle time limiting for faster playback

## Usage

### Play a recording

```bash
go run examples/sessview/main.go play recording.cast

# With options
go run examples/sessview/main.go play --speed 2.0 recording.cast
go run examples/sessview/main.go play --loop recording.cast
go run examples/sessview/main.go play --max-idle 2.0 recording.cast
```

### Browse recordings in a directory

```bash
go run examples/sessview/main.go browse /path/to/recordings

# Browse current directory
go run examples/sessview/main.go browse .
```

In browse mode:
- Use arrow keys to navigate
- Press Enter to play the selected recording
- Press `i` to show detailed info
- Press `r` to reload preview
- Press `q` or Escape to quit

### Show recording information

```bash
go run examples/sessview/main.go info recording.cast
```

## Creating Test Recordings

You can create simple test recordings manually:

```bash
cat > test.cast << 'EOF'
{"version":2,"width":80,"height":24,"timestamp":1700000000,"title":"Test Recording"}
[0.5,"o","Hello, World!\r\n"]
[1.0,"o","This is a test.\r\n"]
[2.0,"o","Done!\r\n"]
EOF
```

Or use the `termsession` package to record actual terminal sessions (see termsession package documentation).

## Dependencies

This example uses the following Wonton packages:

- **cli** - Command-line interface framework
- **tui** - Terminal UI components and layout
- **termsession** - Session recording/playback
- **humanize** - Human-readable formatting
