package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestTableUppercaseHeaders(t *testing.T) {
	columns := []TableColumn{
		{Title: "name", Width: 10},
		{Title: "age", Width: 5},
	}
	rows := [][]string{
		{"Alice", "30"},
		{"Bob", "25"},
	}
	selected := 0

	table := Table(columns, &selected).
		Rows(rows).
		UppercaseHeaders(true)

	assert.True(t, table.uppercaseHeaders)
}

func TestTableMaxColumnWidth(t *testing.T) {
	columns := []TableColumn{
		{Title: "Name"},
		{Title: "Description"},
	}
	rows := [][]string{
		{"Alice", "This is a very long description that should be truncated"},
		{"Bob", "Short"},
	}
	selected := 0

	table := Table(columns, &selected).
		Rows(rows).
		MaxColumnWidth(20)

	assert.Equal(t, 20, table.maxColumnWidth)

	// Calculate widths and verify they respect the max
	table.calculateColumnWidths()
	for _, width := range table.columnWidths {
		assert.LessOrEqual(t, width, 20, "Column width should not exceed max")
	}
}

func TestTableInvertSelectedColors(t *testing.T) {
	columns := []TableColumn{
		{Title: "Name", Width: 10},
	}
	rows := [][]string{{"Alice"}}
	selected := 0

	table := Table(columns, &selected).
		Rows(rows).
		InvertSelectedColors(true)

	assert.True(t, table.invertSelectedColors)
}

func TestTableHeaderBottomBorder(t *testing.T) {
	columns := []TableColumn{
		{Title: "Name", Width: 10},
	}
	rows := [][]string{{"Alice"}}
	selected := 0

	table := Table(columns, &selected).
		Rows(rows).
		HeaderBottomBorder(true)

	assert.True(t, table.headerBottomBorder)
}

func TestInvertColors(t *testing.T) {
	style := NewStyle().
		WithForeground(ColorBlue).
		WithBackground(ColorWhite)

	inverted := invertColors(style)

	assert.Equal(t, ColorWhite, inverted.Foreground)
	assert.Equal(t, ColorBlue, inverted.Background)
}

func TestInvertColorsWithRGB(t *testing.T) {
	fgRGB := NewRGB(255, 0, 0)
	bgRGB := NewRGB(0, 0, 255)

	style := NewStyle().
		WithFgRGB(fgRGB).
		WithBgRGB(bgRGB)

	inverted := invertColors(style)

	assert.NotNil(t, inverted.FgRGB)
	assert.NotNil(t, inverted.BgRGB)
	assert.Equal(t, bgRGB, *inverted.FgRGB)
	assert.Equal(t, fgRGB, *inverted.BgRGB)
}

func TestTableChainedMethods(t *testing.T) {
	columns := []TableColumn{
		{Title: "id", Width: 5},
		{Title: "name", Width: 20},
	}
	rows := [][]string{
		{"1", "Alice"},
		{"2", "Bob"},
	}
	selected := 0

	// Test that all new methods can be chained together
	table := Table(columns, &selected).
		Rows(rows).
		UppercaseHeaders(true).
		MaxColumnWidth(30).
		InvertSelectedColors(true).
		HeaderBottomBorder(true).
		SelectedBg(ColorBlue).
		SelectedFg(ColorWhite)

	assert.NotNil(t, table)
	assert.True(t, table.uppercaseHeaders)
	assert.Equal(t, 30, table.maxColumnWidth)
	assert.True(t, table.invertSelectedColors)
	assert.True(t, table.headerBottomBorder)
	assert.Equal(t, ColorBlue, table.selectedStyle.Background)
	assert.Equal(t, ColorWhite, table.selectedStyle.Foreground)
}

func TestTableMaxColumnWidthZeroMeansUnlimited(t *testing.T) {
	columns := []TableColumn{
		{Title: "Name"},
	}
	rows := [][]string{
		{"This is a very long name that exceeds typical column widths"},
	}
	selected := 0

	table := Table(columns, &selected).
		Rows(rows).
		MaxColumnWidth(0) // 0 means no limit

	table.calculateColumnWidths()

	// With no limit, the column should be sized to fit content + padding
	expectedMinWidth := len("This is a very long name that exceeds typical column widths") + 2
	assert.GreaterOrEqual(t, table.columnWidths[0], expectedMinWidth)
}

func TestTableExplicitColumnWidthNotAffectedByMax(t *testing.T) {
	columns := []TableColumn{
		{Title: "Name", Width: 40}, // Explicit width
	}
	rows := [][]string{{"Alice"}}
	selected := 0

	table := Table(columns, &selected).
		Rows(rows).
		MaxColumnWidth(20) // Max is less than explicit width

	table.calculateColumnWidths()

	// Explicit width should be respected, then limited by max
	assert.Equal(t, 20, table.columnWidths[0])
}

func TestTablePrintCompletesWhenColumnsNeedShrinking(t *testing.T) {
	columns := []TableColumn{
		{Title: "id"},
		{Title: "status"},
		{Title: "queue"},
		{Title: "created_at"},
		{Title: "updated_at"},
		{Title: "actor_id"},
	}
	rows := [][]string{{
		"run_01kqawy7f8fsjtyctj7kdp91ts",
		"completed",
		"default",
		"2026-04-28T16:36:53.864878-04:00",
		"2026-04-28T16:36:54.488712-04:00",
		"user_3D08Ntv0MGl8gAqBLlJriuXwivw",
	}}
	widths := []int{60, 80, 100, 120, 150, 200}

	for _, width := range widths {
		width := width
		t.Run(fmt.Sprintf("width_%d", width), func(t *testing.T) {
			selected := -1
			var buf strings.Builder
			done := make(chan error, 1)

			go func() {
				done <- Print(Table(columns, &selected).Rows(rows), PrintConfig{
					Width:  width,
					Output: &buf,
				})
			}()

			select {
			case err := <-done:
				assert.NoError(t, err)
				assert.NotEmpty(t, buf.String())
			case <-time.After(time.Second):
				assert.True(t, false, "table render timed out at width %d", width)
			}
		})
	}
}
