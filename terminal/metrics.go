package terminal

import (
	"fmt"
	"sync"
	"time"
)

// RenderMetrics tracks performance statistics for the terminal rendering system.
//
// Metrics collection is disabled by default for maximum performance. Enable it
// using Terminal.EnableMetrics() when you need to diagnose performance issues
// or optimize rendering code.
//
// # Usage
//
//	term.EnableMetrics()
//
//	// Render some frames...
//	for i := 0; i < 100; i++ {
//	    frame, _ := term.BeginFrame()
//	    // ... draw content ...
//	    term.EndFrame(frame)
//	}
//
//	// Get performance metrics
//	snapshot := term.GetMetrics()
//	fmt.Println(snapshot.String())
//	// Output: Frames: 100, avg 45.2ms/frame, 1234 cells/frame, etc.
//
// # Metrics Tracked
//
// RenderMetrics tracks:
//   - Frame count (total rendered, skipped when no changes)
//   - Cell updates (total cells written, average per frame)
//   - ANSI codes emitted (escape sequences sent to terminal)
//   - Bytes written (total output to terminal)
//   - Timing (min/max/average frame render times, FPS)
//   - Dirty regions (areas that changed between frames)
//
// # Thread Safety
//
// RenderMetrics is thread-safe. Snapshot() returns a point-in-time copy
// that can be safely used without locking.
type RenderMetrics struct {
	mu sync.RWMutex

	// Frame metrics
	TotalFrames      uint64 // Total number of frames rendered
	CellsUpdated     uint64 // Total cells written to terminal
	ANSICodesEmitted uint64 // Total ANSI escape codes emitted
	BytesWritten     uint64 // Total bytes written to terminal
	SkippedFrames    uint64 // Frames skipped due to no changes

	// Timing metrics
	TotalRenderTime time.Duration // Cumulative time spent rendering
	LastFrameTime   time.Duration // Time taken for last frame
	MinFrameTime    time.Duration // Fastest frame
	MaxFrameTime    time.Duration // Slowest frame

	// Dirty region metrics
	TotalDirtyArea uint64 // Sum of all dirty region areas
	LastDirtyArea  int    // Area of last dirty region (width * height)
	MaxDirtyArea   int    // Largest dirty region seen

	// Per-frame averages (calculated on demand)
	avgCellsPerFrame float64
	avgTimePerFrame  float64
	avgDirtyArea     float64
}

// NewRenderMetrics creates a new metrics tracker
func NewRenderMetrics() *RenderMetrics {
	return &RenderMetrics{
		MinFrameTime: time.Duration(1<<63 - 1), // Max duration initially
	}
}

// RecordFrame records metrics for a completed frame render
func (m *RenderMetrics) RecordFrame(cellsUpdated int, ansiCodes int, bytesWritten int, duration time.Duration, dirtyArea int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.TotalFrames++
	m.CellsUpdated += uint64(cellsUpdated)
	m.ANSICodesEmitted += uint64(ansiCodes)
	m.BytesWritten += uint64(bytesWritten)
	m.TotalRenderTime += duration
	m.LastFrameTime = duration
	m.TotalDirtyArea += uint64(dirtyArea)
	m.LastDirtyArea = dirtyArea

	// Update min/max
	if duration < m.MinFrameTime && duration > 0 {
		m.MinFrameTime = duration
	}
	if duration > m.MaxFrameTime {
		m.MaxFrameTime = duration
	}
	if dirtyArea > m.MaxDirtyArea {
		m.MaxDirtyArea = dirtyArea
	}

	// Recalculate averages
	if m.TotalFrames > 0 {
		m.avgCellsPerFrame = float64(m.CellsUpdated) / float64(m.TotalFrames)
		m.avgTimePerFrame = float64(m.TotalRenderTime) / float64(m.TotalFrames)
		m.avgDirtyArea = float64(m.TotalDirtyArea) / float64(m.TotalFrames)
	}
}

// RecordSkippedFrame increments the skipped frame counter
func (m *RenderMetrics) RecordSkippedFrame() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.SkippedFrames++
}

// Reset clears all metrics
func (m *RenderMetrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.TotalFrames = 0
	m.CellsUpdated = 0
	m.ANSICodesEmitted = 0
	m.BytesWritten = 0
	m.SkippedFrames = 0
	m.TotalRenderTime = 0
	m.LastFrameTime = 0
	m.MinFrameTime = time.Duration(1<<63 - 1)
	m.MaxFrameTime = 0
	m.TotalDirtyArea = 0
	m.LastDirtyArea = 0
	m.MaxDirtyArea = 0
	m.avgCellsPerFrame = 0
	m.avgTimePerFrame = 0
	m.avgDirtyArea = 0
}

// MetricsSnapshot represents a point-in-time snapshot of rendering metrics
type MetricsSnapshot struct {
	TotalFrames      uint64
	CellsUpdated     uint64
	ANSICodesEmitted uint64
	BytesWritten     uint64
	SkippedFrames    uint64
	TotalRenderTime  time.Duration
	LastFrameTime    time.Duration
	MinFrameTime     time.Duration
	MaxFrameTime     time.Duration
	TotalDirtyArea   uint64
	LastDirtyArea    int
	MaxDirtyArea     int
	AvgCellsPerFrame float64
	AvgTimePerFrame  time.Duration
	AvgDirtyArea     float64
}

// Snapshot returns a point-in-time copy of the metrics (thread-safe)
func (m *RenderMetrics) Snapshot() MetricsSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return MetricsSnapshot{
		TotalFrames:      m.TotalFrames,
		CellsUpdated:     m.CellsUpdated,
		ANSICodesEmitted: m.ANSICodesEmitted,
		BytesWritten:     m.BytesWritten,
		SkippedFrames:    m.SkippedFrames,
		TotalRenderTime:  m.TotalRenderTime,
		LastFrameTime:    m.LastFrameTime,
		MinFrameTime:     m.MinFrameTime,
		MaxFrameTime:     m.MaxFrameTime,
		TotalDirtyArea:   m.TotalDirtyArea,
		LastDirtyArea:    m.LastDirtyArea,
		MaxDirtyArea:     m.MaxDirtyArea,
		AvgCellsPerFrame: m.avgCellsPerFrame,
		AvgTimePerFrame:  time.Duration(m.avgTimePerFrame),
		AvgDirtyArea:     m.avgDirtyArea,
	}
}

// AvgCellsPerFrame returns the average number of cells updated per frame
func (m *RenderMetrics) AvgCellsPerFrame() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.avgCellsPerFrame
}

// AvgFrameTime returns the average time taken per frame
func (m *RenderMetrics) AvgFrameTime() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return time.Duration(m.avgTimePerFrame)
}

// AvgDirtyArea returns the average dirty region area
func (m *RenderMetrics) AvgDirtyArea() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.avgDirtyArea
}

// FPS returns the average frames per second based on total time
func (m *RenderMetrics) FPS() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.TotalRenderTime == 0 {
		return 0
	}
	seconds := m.TotalRenderTime.Seconds()
	if seconds == 0 {
		return 0
	}
	return float64(m.TotalFrames) / seconds
}

// Efficiency returns the percentage of frames that resulted in actual rendering
// (vs skipped due to no changes). Higher is better when content is static.
func (m *RenderMetrics) Efficiency() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	total := m.TotalFrames + m.SkippedFrames
	if total == 0 {
		return 100.0
	}
	return (float64(m.SkippedFrames) / float64(total)) * 100.0
}

// FPS calculates frames per second from the snapshot
func (s *MetricsSnapshot) FPS() float64 {
	if s.TotalRenderTime == 0 {
		return 0
	}
	seconds := s.TotalRenderTime.Seconds()
	if seconds == 0 {
		return 0
	}
	return float64(s.TotalFrames) / seconds
}

// Efficiency calculates the skip efficiency from the snapshot
func (s *MetricsSnapshot) Efficiency() float64 {
	total := s.TotalFrames + s.SkippedFrames
	if total == 0 {
		return 100.0
	}
	return (float64(s.SkippedFrames) / float64(total)) * 100.0
}

// String returns a formatted string representation of the metrics
func (s *MetricsSnapshot) String() string {
	if s.TotalFrames == 0 && s.SkippedFrames == 0 {
		return "No rendering activity"
	}

	return fmt.Sprintf(`Rendering Metrics:
  Frames: %d rendered, %d skipped (%.1f%% efficiency)
  Cells: %d total, %.1f avg/frame
  ANSI codes: %d total
  Bytes written: %d (%.1f KB)
  Timing: %.2f FPS, avg %.2fms/frame
  Frame times: min=%.2fms, max=%.2fms, last=%.2fms
  Dirty regions: avg area=%.1f cells, max=%d cells, last=%d cells`,
		s.TotalFrames,
		s.SkippedFrames,
		s.Efficiency(),
		s.CellsUpdated,
		s.AvgCellsPerFrame,
		s.ANSICodesEmitted,
		s.BytesWritten,
		float64(s.BytesWritten)/1024.0,
		s.FPS(),
		s.AvgTimePerFrame.Seconds()*1000,
		s.MinFrameTime.Seconds()*1000,
		s.MaxFrameTime.Seconds()*1000,
		s.LastFrameTime.Seconds()*1000,
		s.AvgDirtyArea,
		s.MaxDirtyArea,
		s.LastDirtyArea,
	)
}

// Compact returns a compact single-line representation of key metrics
func (s *MetricsSnapshot) Compact() string {
	return fmt.Sprintf("frames=%d cells=%d fps=%.1f avg=%.1fms",
		s.TotalFrames,
		s.CellsUpdated,
		s.FPS(),
		s.AvgTimePerFrame.Seconds()*1000,
	)
}
