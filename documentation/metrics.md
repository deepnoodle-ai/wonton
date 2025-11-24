# Performance Metrics

Gooey includes a comprehensive performance metrics system that tracks rendering statistics in real-time. This feature helps developers optimize their TUI applications by providing visibility into rendering performance.

## Overview

The metrics system tracks:
- **Frame Statistics**: Total frames rendered and skipped
- **Cell Updates**: Number of terminal cells updated
- **ANSI Codes**: Count of escape sequences emitted
- **Bytes Written**: Total output size
- **Timing**: Frame render times (min, max, average, last)
- **Dirty Regions**: Size of modified screen areas

## Quick Start

```go
terminal, _ := gooey.NewTerminal()
defer terminal.Close()

// Enable metrics collection
terminal.EnableMetrics()

// Render your TUI...
frame, _ := terminal.BeginFrame()
frame.PrintStyled(0, 0, "Hello!", gooey.NewStyle())
terminal.EndFrame(frame)

// Get metrics snapshot
metrics := terminal.GetMetrics()
fmt.Println(metrics.String())
```

## API

### Enabling/Disabling Metrics

```go
// Turn on metrics collection
terminal.EnableMetrics()

// Turn off metrics collection
terminal.DisableMetrics()
```

**Note**: Metrics are **disabled by default** to avoid any overhead. Enable them only when profiling or debugging.

### Getting Metrics

```go
// Get a snapshot of current metrics
snapshot := terminal.GetMetrics()

// Access individual fields
fmt.Printf("Total frames: %d\n", snapshot.TotalFrames)
fmt.Printf("Cells updated: %d\n", snapshot.CellsUpdated)
fmt.Printf("Average FPS: %.2f\n", snapshot.FPS())
```

### Resetting Metrics

```go
// Clear all accumulated metrics
terminal.ResetMetrics()
```

## MetricsSnapshot Structure

The `MetricsSnapshot` struct contains:

```go
type MetricsSnapshot struct {
    TotalFrames       uint64        // Frames rendered
    CellsUpdated      uint64        // Cells written
    ANSICodesEmitted  uint64        // Escape sequences sent
    BytesWritten      uint64        // Total bytes output
    SkippedFrames     uint64        // Frames with no changes

    TotalRenderTime   time.Duration // Cumulative time
    LastFrameTime     time.Duration // Most recent frame
    MinFrameTime      time.Duration // Fastest frame
    MaxFrameTime      time.Duration // Slowest frame
    AvgTimePerFrame   time.Duration // Average per frame

    TotalDirtyArea    uint64        // Sum of dirty regions
    LastDirtyArea     int           // Last dirty region size
    MaxDirtyArea      int           // Largest dirty region
    AvgDirtyArea      float64       // Average dirty region
    AvgCellsPerFrame  float64       // Average cells/frame
}
```

## Helper Methods

### FPS Calculation

```go
fps := snapshot.FPS()
fmt.Printf("Average FPS: %.2f\n", fps)
```

### Efficiency

The efficiency metric shows what percentage of frames were skipped due to no changes:

```go
efficiency := snapshot.Efficiency()
fmt.Printf("Skip efficiency: %.1f%%\n", efficiency)
```

Higher efficiency is better when content is mostly static, as it means fewer redundant renders.

### Formatted Output

```go
// Detailed multi-line output
fmt.Println(snapshot.String())

// Compact single-line summary
fmt.Println(snapshot.Compact())
```

## Example Output

### Detailed Format

```
Rendering Metrics:
  Frames: 1847 rendered, 153 skipped (8.3% efficiency)
  Cells: 295520 total, 160.0 avg/frame
  ANSI codes: 7388 total
  Bytes written: 1572864 (1536.0 KB)
  Timing: 61.57 FPS, avg 16.24ms/frame
  Frame times: min=12.34ms, max=23.45ms, last=15.67ms
  Dirty regions: avg area=320.5 cells, max=1920 cells, last=160 cells
```

### Compact Format

```
frames=1847 cells=295520 fps=61.6 avg=16.2ms
```

## Use Cases

### 1. Performance Profiling

Identify rendering bottlenecks:

```go
terminal.EnableMetrics()
terminal.ResetMetrics()

// Run your rendering loop
runApplication(terminal, 10*time.Second)

// Analyze performance
metrics := terminal.GetMetrics()
if metrics.AvgTimePerFrame > 16*time.Millisecond {
    fmt.Println("Warning: Not maintaining 60 FPS")
}
```

### 2. Optimization Validation

Measure impact of optimizations:

```go
// Baseline
terminal.EnableMetrics()
runOldVersion()
baseline := terminal.GetMetrics()

// After optimization
terminal.ResetMetrics()
runNewVersion()
optimized := terminal.GetMetrics()

improvement := baseline.AvgTimePerFrame - optimized.AvgTimePerFrame
fmt.Printf("Speedup: %.2fms per frame\n", improvement.Seconds()*1000)
```

### 3. Live Monitoring

Display metrics in your TUI:

```go
for {
    frame, _ := terminal.BeginFrame()

    // Render your content...
    renderContent(frame)

    // Show live metrics
    metrics := terminal.GetMetrics()
    frame.PrintStyled(0, 0, metrics.Compact(), gooey.NewStyle())

    terminal.EndFrame(frame)
}
```

### 4. Testing and Regression Detection

```go
func TestRenderingPerformance(t *testing.T) {
    term := gooey.NewTestTerminal(80, 24, &bytes.Buffer{})
    term.EnableMetrics()

    // Render test content
    for i := 0; i < 1000; i++ {
        frame, _ := term.BeginFrame()
        renderTestContent(frame)
        term.EndFrame(frame)
    }

    metrics := term.GetMetrics()

    // Assert performance requirements
    require.Less(t, metrics.AvgTimePerFrame, 20*time.Millisecond)
    require.Greater(t, metrics.FPS(), 50.0)
}
```

## Performance Overhead

The metrics system is designed to have **minimal overhead**:

- Disabled by default (zero cost when not in use)
- Simple counter increments and timestamp recording
- No allocations in hot path
- Thread-safe with read-write locks
- Typical overhead: **< 5%** when enabled

Benchmark results show overhead is typically under 20% even with continuous metrics collection.

## Best Practices

1. **Enable only when needed**: Keep metrics disabled in production unless debugging
2. **Reset between tests**: Use `ResetMetrics()` to get clean measurements
3. **Use snapshots**: Call `GetMetrics()` to get a consistent view of metrics
4. **Monitor over time**: Track metrics across multiple runs to identify trends
5. **Combine with profiling**: Use Go's pprof alongside metrics for deep analysis

## Integration with Development Workflow

### During Development

```go
if os.Getenv("GOOEY_DEBUG") == "1" {
    terminal.EnableMetrics()
    defer func() {
        metrics := terminal.GetMetrics()
        log.Println(metrics.String())
    }()
}
```

### In Tests

```go
func BenchmarkRendering(b *testing.B) {
    term := gooey.NewTestTerminal(80, 24, io.Discard)
    term.EnableMetrics()

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        frame, _ := term.BeginFrame()
        renderContent(frame)
        term.EndFrame(frame)
    }

    metrics := term.GetMetrics()
    b.ReportMetric(metrics.FPS(), "fps")
    b.ReportMetric(float64(metrics.AvgTimePerFrame.Microseconds()), "μs/frame")
}
```

## See Also

- [Example: metrics_demo](../examples/metrics_demo/main.go) - Live metrics visualization
- [Double Buffering](double_buffering.md) - Understanding the rendering system
- [API Recommendations](API_RECOMMENDATIONS.md) - Performance best practices
