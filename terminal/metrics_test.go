package terminal

import (
	"strings"
	"testing"
	"time"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestMetricsBasics(t *testing.T) {
	metrics := NewRenderMetrics()

	// Initial state
	assert.Equal(t, uint64(0), metrics.TotalFrames)
	assert.Equal(t, uint64(0), metrics.CellsUpdated)

	// Record a frame
	metrics.RecordFrame(10, 5, 100, 10*time.Millisecond, 20)

	assert.Equal(t, uint64(1), metrics.TotalFrames)
	assert.Equal(t, uint64(10), metrics.CellsUpdated)
	assert.Equal(t, uint64(5), metrics.ANSICodesEmitted)
	assert.Equal(t, uint64(100), metrics.BytesWritten)
	assert.Equal(t, 10*time.Millisecond, metrics.LastFrameTime)
	assert.Equal(t, 20, metrics.LastDirtyArea)
}

func TestMetricsAverages(t *testing.T) {
	metrics := NewRenderMetrics()

	// Record multiple frames
	metrics.RecordFrame(10, 5, 100, 10*time.Millisecond, 20)
	metrics.RecordFrame(20, 8, 150, 15*time.Millisecond, 30)
	metrics.RecordFrame(30, 10, 200, 20*time.Millisecond, 40)

	// Check averages
	assert.Equal(t, 20.0, metrics.AvgCellsPerFrame())
	assert.InDelta(t, 15.0, float64(metrics.AvgFrameTime().Milliseconds()), 1.0)
	assert.Equal(t, 30.0, metrics.AvgDirtyArea())
}

func TestMetricsMinMax(t *testing.T) {
	metrics := NewRenderMetrics()

	metrics.RecordFrame(10, 5, 100, 10*time.Millisecond, 20)
	metrics.RecordFrame(20, 8, 150, 5*time.Millisecond, 50)
	metrics.RecordFrame(30, 10, 200, 20*time.Millisecond, 30)

	assert.Equal(t, 5*time.Millisecond, metrics.MinFrameTime)
	assert.Equal(t, 20*time.Millisecond, metrics.MaxFrameTime)
	assert.Equal(t, 50, metrics.MaxDirtyArea)
}

func TestMetricsSkippedFrames(t *testing.T) {
	metrics := NewRenderMetrics()

	metrics.RecordFrame(10, 5, 100, 10*time.Millisecond, 20)
	metrics.RecordSkippedFrame()
	metrics.RecordSkippedFrame()

	assert.Equal(t, uint64(1), metrics.TotalFrames)
	assert.Equal(t, uint64(2), metrics.SkippedFrames)
	assert.InDelta(t, 66.67, metrics.Efficiency(), 0.1)
}

func TestMetricsReset(t *testing.T) {
	metrics := NewRenderMetrics()

	metrics.RecordFrame(10, 5, 100, 10*time.Millisecond, 20)
	metrics.RecordFrame(20, 8, 150, 15*time.Millisecond, 30)

	metrics.Reset()

	assert.Equal(t, uint64(0), metrics.TotalFrames)
	assert.Equal(t, uint64(0), metrics.CellsUpdated)
	assert.Equal(t, time.Duration(1<<63-1), metrics.MinFrameTime)
}

func TestMetricsSnapshot(t *testing.T) {
	metrics := NewRenderMetrics()

	metrics.RecordFrame(10, 5, 100, 10*time.Millisecond, 20)
	metrics.RecordFrame(20, 8, 150, 15*time.Millisecond, 30)

	snapshot := metrics.Snapshot()

	assert.Equal(t, uint64(2), snapshot.TotalFrames)
	assert.Equal(t, uint64(30), snapshot.CellsUpdated)
	assert.Equal(t, 15.0, snapshot.AvgCellsPerFrame)
}

func TestMetricsSnapshotString(t *testing.T) {
	snapshot := MetricsSnapshot{
		TotalFrames:      100,
		CellsUpdated:     1000,
		ANSICodesEmitted: 500,
		BytesWritten:     10000,
		SkippedFrames:    50,
		AvgCellsPerFrame: 10.0,
		AvgTimePerFrame:  5 * time.Millisecond,
	}

	str := snapshot.String()
	assert.Contains(t, str, "100 rendered")
	assert.Contains(t, str, "50 skipped")
	assert.Contains(t, str, "1000 total")
}

func TestMetricsSnapshotCompact(t *testing.T) {
	snapshot := MetricsSnapshot{
		TotalFrames:     100,
		CellsUpdated:    1000,
		AvgTimePerFrame: 5 * time.Millisecond,
	}

	compact := snapshot.Compact()
	assert.Contains(t, compact, "frames=100")
	assert.Contains(t, compact, "cells=1000")
}

func TestTerminalMetricsIntegration(t *testing.T) {
	// Create a test terminal with a string builder as output
	var output strings.Builder
	term := NewTestTerminal(80, 24, &output)

	// Metrics should be disabled by default
	snapshot := term.GetMetrics()
	assert.Equal(t, uint64(0), snapshot.TotalFrames)

	// Enable metrics
	term.EnableMetrics()

	// Render a frame
	frame, err := term.BeginFrame()
	assert.NoError(t, err)
	frame.PrintStyled(0, 0, "Hello, World!", NewStyle().WithForeground(ColorRed))
	err = term.EndFrame(frame)
	assert.NoError(t, err)

	// Check metrics were recorded
	snapshot = term.GetMetrics()
	assert.Equal(t, uint64(1), snapshot.TotalFrames)
	assert.Greater(t, snapshot.CellsUpdated, uint64(0))
	assert.Greater(t, snapshot.BytesWritten, uint64(0))

	// Render without changes (should skip)
	frame, err = term.BeginFrame()
	assert.NoError(t, err)
	err = term.EndFrame(frame)
	assert.NoError(t, err)

	snapshot = term.GetMetrics()
	assert.Equal(t, uint64(1), snapshot.TotalFrames)   // Still 1 rendered frame
	assert.Equal(t, uint64(1), snapshot.SkippedFrames) // 1 skipped

	// Reset metrics
	term.ResetMetrics()
	snapshot = term.GetMetrics()
	assert.Equal(t, uint64(0), snapshot.TotalFrames)
	assert.Equal(t, uint64(0), snapshot.SkippedFrames)

	// Disable metrics
	term.DisableMetrics()

	// Render another frame (should not be tracked)
	frame, err = term.BeginFrame()
	assert.NoError(t, err)
	frame.PrintStyled(0, 1, "Not tracked", NewStyle())
	err = term.EndFrame(frame)
	assert.NoError(t, err)

	// Metrics should still be zero
	snapshot = term.GetMetrics()
	assert.Equal(t, uint64(0), snapshot.TotalFrames)
}

func TestMetricsPerformanceOverhead(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance overhead test in short mode")
	}

	// This test measures metrics collection overhead (informational)
	var output strings.Builder
	term := NewTestTerminal(80, 24, &output)

	iterations := 10000 // More iterations for better measurement

	// Measure without metrics
	start := time.Now()
	for i := 0; i < iterations; i++ {
		frame, _ := term.BeginFrame()
		frame.PrintStyled(0, 0, "Test", NewStyle())
		term.EndFrame(frame)
	}
	withoutMetrics := time.Since(start)

	// Reset and enable metrics
	output.Reset()
	term = NewTestTerminal(80, 24, &output)
	term.EnableMetrics()

	// Measure with metrics
	start = time.Now()
	for i := 0; i < iterations; i++ {
		frame, _ := term.BeginFrame()
		frame.PrintStyled(0, 0, "Test", NewStyle())
		term.EndFrame(frame)
	}
	withMetrics := time.Since(start)

	// Calculate and report overhead (informational, not strict requirement)
	overhead := float64(withMetrics-withoutMetrics) / float64(withoutMetrics)
	t.Logf("Performance overhead: %.2f%% (without: %v, with: %v)",
		overhead*100, withoutMetrics, withMetrics)

	// Just verify it's not absurdly high (> 100%)
	assert.Less(t, overhead, 1.0,
		"Metrics overhead should not exceed 100%% (actual: %.2f%%)", overhead*100)
}
