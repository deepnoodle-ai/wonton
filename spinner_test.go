package gooey

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSpinner(t *testing.T) {
	term := NewTestTerminal(80, 24, &bytes.Buffer{})
	spinner := NewSpinner(term, SpinnerDots)

	require.NotNil(t, spinner)
	assert.NotNil(t, spinner.frames)
	assert.Greater(t, len(spinner.frames), 0)
	assert.Greater(t, spinner.interval, time.Duration(0))
	assert.False(t, spinner.active)
}

func TestSpinner_WithStyle(t *testing.T) {
	term := NewTestTerminal(80, 24, &bytes.Buffer{})
	spinner := NewSpinner(term, SpinnerDots)

	style := NewStyle().WithForeground(ColorGreen)
	spinner.WithStyle(style)

	assert.Equal(t, ColorGreen, spinner.style.Foreground)
}

func TestSpinner_WithMessage(t *testing.T) {
	term := NewTestTerminal(80, 24, &bytes.Buffer{})
	spinner := NewSpinner(term, SpinnerDots)

	spinner.WithMessage("Loading...")

	assert.Equal(t, "Loading...", spinner.message)
}

func TestSpinner_StartStop(t *testing.T) {
	term := NewTestTerminal(80, 24, &bytes.Buffer{})
	spinner := NewSpinner(term, SpinnerDots)

	spinner.Start()
	assert.True(t, spinner.active)

	// Give it a moment to run
	time.Sleep(10 * time.Millisecond)

	spinner.Stop()
	assert.False(t, spinner.active)
}

func TestSpinner_DoubleStart(t *testing.T) {
	term := NewTestTerminal(80, 24, &bytes.Buffer{})
	spinner := NewSpinner(term, SpinnerDots)

	spinner.Start()
	assert.True(t, spinner.active)

	// Starting again should be a no-op
	require.NotPanics(t, func() {
		spinner.Start()
	})

	spinner.Stop()
}

func TestSpinner_DoubleStop(t *testing.T) {
	term := NewTestTerminal(80, 24, &bytes.Buffer{})
	spinner := NewSpinner(term, SpinnerDots)

	spinner.Start()
	spinner.Stop()

	// Stopping again should be a no-op
	require.NotPanics(t, func() {
		spinner.Stop()
	})
}

func TestSpinnerStyle_Dots(t *testing.T) {
	assert.NotNil(t, SpinnerDots.Frames)
	assert.Greater(t, len(SpinnerDots.Frames), 0)
	assert.Greater(t, SpinnerDots.Interval, time.Duration(0))
}

func TestSpinnerStyle_Line(t *testing.T) {
	assert.Equal(t, []string{"-", "\\", "|", "/"}, SpinnerLine.Frames)
	assert.Equal(t, 100*time.Millisecond, SpinnerLine.Interval)
}

func TestSpinnerStyle_Arrows(t *testing.T) {
	assert.NotNil(t, SpinnerArrows.Frames)
	assert.Equal(t, 8, len(SpinnerArrows.Frames))
}

func TestSpinnerStyle_AllStylesHaveFrames(t *testing.T) {
	styles := []SpinnerStyle{
		SpinnerDots,
		SpinnerLine,
		SpinnerArrows,
		SpinnerCircle,
		SpinnerSquare,
		SpinnerBounce,
		SpinnerBar,
		SpinnerStars,
		SpinnerStarField,
		SpinnerAsterisk,
		SpinnerSparkle,
	}

	for _, style := range styles {
		assert.Greater(t, len(style.Frames), 0, "Spinner style should have frames")
		assert.Greater(t, style.Interval, time.Duration(0), "Spinner style should have positive interval")
	}
}

func TestNewProgressBar(t *testing.T) {
	term := NewTestTerminal(80, 24, &bytes.Buffer{})
	bar := NewProgressBar(term, 100)

	assert.Equal(t, 100, bar.total)
	assert.Equal(t, 0, bar.current)
	assert.Equal(t, 40, bar.width)
	assert.True(t, bar.showPercent)
	assert.False(t, bar.showNumbers)
}

func TestProgressBar_WithWidth(t *testing.T) {
	term := NewTestTerminal(80, 24, &bytes.Buffer{})
	bar := NewProgressBar(term, 100).WithWidth(60)

	assert.Equal(t, 60, bar.width)
}

func TestProgressBar_WithStyle(t *testing.T) {
	term := NewTestTerminal(80, 24, &bytes.Buffer{})
	style := NewStyle().WithForeground(ColorGreen)
	bar := NewProgressBar(term, 100).WithStyle(style)

	assert.Equal(t, ColorGreen, bar.style.Foreground)
}

func TestProgressBar_WithChars(t *testing.T) {
	term := NewTestTerminal(80, 24, &bytes.Buffer{})
	bar := NewProgressBar(term, 100).WithChars("=", "-")

	assert.Equal(t, "=", bar.fillChar)
	assert.Equal(t, "-", bar.emptyChar)
}

func TestProgressBar_ShowNumbers(t *testing.T) {
	term := NewTestTerminal(80, 24, &bytes.Buffer{})
	bar := NewProgressBar(term, 100).ShowNumbers()

	assert.True(t, bar.showNumbers)
}

func TestProgressBar_Update(t *testing.T) {
	term := NewTestTerminal(80, 24, &bytes.Buffer{})
	bar := NewProgressBar(term, 100)

	bar.Update(50, "Half done")

	assert.Equal(t, 50, bar.current)
	assert.Equal(t, "Half done", bar.message)
}

func TestProgressBar_Update_OverTotal(t *testing.T) {
	term := NewTestTerminal(80, 24, &bytes.Buffer{})
	bar := NewProgressBar(term, 100)

	bar.Update(150, "Over")

	assert.Equal(t, 100, bar.current, "Current should be capped at total")
}

func TestProgressBar_Increment(t *testing.T) {
	term := NewTestTerminal(80, 24, &bytes.Buffer{})
	bar := NewProgressBar(term, 100)

	bar.Increment("Step 1")
	assert.Equal(t, 1, bar.current)

	bar.Increment("Step 2")
	assert.Equal(t, 2, bar.current)
}

func TestProgressBar_Increment_OverTotal(t *testing.T) {
	term := NewTestTerminal(80, 24, &bytes.Buffer{})
	bar := NewProgressBar(term, 10)

	for i := 0; i < 15; i++ {
		bar.Increment("Step")
	}

	assert.Equal(t, 10, bar.current, "Current should be capped at total")
}

func TestProgressBar_Complete(t *testing.T) {
	term := NewTestTerminal(80, 24, &bytes.Buffer{})
	bar := NewProgressBar(term, 100)

	bar.Complete("Done")

	assert.Equal(t, 100, bar.current)
	assert.Equal(t, "Done", bar.message)
}

func TestNewMultiProgress(t *testing.T) {
	term := NewTestTerminal(80, 24, &bytes.Buffer{})
	mp := NewMultiProgress(term)

	assert.NotNil(t, mp)
	assert.Equal(t, 0, len(mp.items))
}

func TestMultiProgress_Add_ProgressBar(t *testing.T) {
	term := NewTestTerminal(80, 24, &bytes.Buffer{})
	mp := NewMultiProgress(term)

	item := mp.Add("task1", 100, false)

	require.NotNil(t, item)
	assert.Equal(t, "task1", item.ID)
	assert.Equal(t, 100, item.Total)
	assert.False(t, item.SpinnerOnly)
	assert.NotNil(t, item.Bar)
	assert.Nil(t, item.Spinner)
	assert.Equal(t, 1, len(mp.items))
}

func TestMultiProgress_Add_Spinner(t *testing.T) {
	term := NewTestTerminal(80, 24, &bytes.Buffer{})
	mp := NewMultiProgress(term)

	item := mp.Add("task1", 0, true)

	require.NotNil(t, item)
	assert.Equal(t, "task1", item.ID)
	assert.True(t, item.SpinnerOnly)
	assert.Nil(t, item.Bar)
	assert.NotNil(t, item.Spinner)
}

func TestMultiProgress_Update(t *testing.T) {
	term := NewTestTerminal(80, 24, &bytes.Buffer{})
	mp := NewMultiProgress(term)

	mp.Add("task1", 100, false)
	mp.Update("task1", 50, "Half done")

	assert.Equal(t, 50, mp.items[0].Current)
	assert.Equal(t, "Half done", mp.items[0].Message)
}

func TestMultiProgress_Update_NonexistentTask(t *testing.T) {
	term := NewTestTerminal(80, 24, &bytes.Buffer{})
	mp := NewMultiProgress(term)

	// Should not panic
	require.NotPanics(t, func() {
		mp.Update("nonexistent", 50, "Test")
	})
}

func TestMultiProgress_StartStop(t *testing.T) {
	term := NewTestTerminal(80, 24, &bytes.Buffer{})
	mp := NewMultiProgress(term)

	mp.Add("task1", 100, true)

	require.NotPanics(t, func() {
		mp.Start()
		time.Sleep(10 * time.Millisecond)
		mp.Stop()
	})
}

func TestProgressItem_Fields(t *testing.T) {
	item := &ProgressItem{
		ID:          "test",
		Message:     "Testing",
		Current:     50,
		Total:       100,
		Style:       NewStyle().WithForeground(ColorGreen),
		SpinnerOnly: false,
	}

	assert.Equal(t, "test", item.ID)
	assert.Equal(t, "Testing", item.Message)
	assert.Equal(t, 50, item.Current)
	assert.Equal(t, 100, item.Total)
	assert.False(t, item.SpinnerOnly)
	assert.Equal(t, ColorGreen, item.Style.Foreground)
}
