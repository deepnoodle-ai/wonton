package tui

import (
	"testing"

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
