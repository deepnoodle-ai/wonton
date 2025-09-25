# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with
code in this repository.

## Project Overview

Gooey is a Terminal GUI library for Go that provides advanced terminal
manipulation capabilities with comprehensive animation support. The library
focuses on creating dynamic, visually appealing terminal user interfaces with
features like rainbow text animations, pulsing effects, animated layouts, and
interactive input components.

## Development Commands

### Building

```bash
go build ./...
```

### Testing

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests for a specific package
go test github.com/deepnoodle-ai/gooey

# Run a single test
go test -run TestName ./...
```

### Code Quality

```bash
# Format code
go fmt ./...

# Run static analysis
go vet ./...
```

### Running Examples

```bash
# All features demo
go run examples/all/main.go
```

## Architecture

### Core Components

The library is organized around several key abstractions:

1. **Terminal** (`terminal.go`): Low-level terminal control including cursor movement, screen clearing, raw mode, and alternate screen buffer management. All terminal operations go through this abstraction.

2. **ScreenManager** (`screen_manager.go`): Manages virtual screen regions and buffered output to prevent flickering during updates.

3. **Layout System**:
   - `layout.go`: Base layout functionality for organizing screen regions
   - `animated_layout.go`: Enhanced layouts with animation support for headers, content areas, and footers

4. **Input Handling**:
   - `input.go`: Core input processing and event handling
   - `input_enhanced.go`: Enhanced input with special key support
   - `input_fixed.go`: Fixed-position input fields
   - `input_interactive.go`: Interactive mode with real-time updates
   - `input_simple.go`: Basic input functionality

5. **Animation System**:
   - `animator.go`: Main animation engine running at 30+ FPS with element management
   - `animated_layout.go`: Layouts with dedicated animation areas
   - Color animations: Rainbow, Pulse, Wave effects with RGB support

6. **UI Components** (`components.go`): Reusable UI elements like buttons, status bars, progress indicators

7. **Styling** (`style.go`): Text styling system with colors, attributes (bold, italic, underline), and RGB color support

8. **Mouse Support** (`mouse.go`): Mouse event handling and tracking

### Key Design Patterns

- **Synchronization**: Uses sync.Mutex throughout for thread-safe terminal operations
- **Animation Loop**: Dedicated goroutines for smooth 30+ FPS animations
- **Buffer Management**: Virtual screen regions to prevent flicker
- **ANSI Escape Sequences**: Direct terminal control via ANSI codes
- **Alternate Screen**: Preserves terminal state using alternate screen buffer

### Animation Features

The library provides several animation types defined in ANIMATIONS.md:

- RainbowAnimation: Moving rainbow effects with configurable speed and length
- PulseAnimation: Brightness pulsing with custom RGB colors
- WaveAnimation: Wave-like color transitions
- Custom animations can be created by implementing the animation interface

## Module Dependencies

- `golang.org/x/term`: Terminal control and raw mode
- `github.com/mattn/go-runewidth`: Unicode character width handling
