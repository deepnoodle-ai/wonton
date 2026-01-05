package tui

import (
	"strings"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

// TestForEach_Empty tests ForEach with an empty slice
func TestForEach_Empty(t *testing.T) {
	items := []string{}
	view := ForEach(items, func(item string, i int) View {
		return Text("%s", item)
	})

	w, h := view.size(100, 100)
	assert.Equal(t, 0, w)
	assert.Equal(t, 0, h)
}

// TestForEach_SingleItem tests ForEach with a single item
func TestForEach_SingleItem(t *testing.T) {
	items := []string{"Hello"}
	view := ForEach(items, func(item string, i int) View {
		return Text("%s", item)
	})

	w, h := view.size(100, 100)
	assert.Equal(t, 5, w) // "Hello" is 5 chars
	assert.Equal(t, 1, h) // single line
}

// TestForEach_MultipleItems tests ForEach with multiple items
func TestForEach_MultipleItems(t *testing.T) {
	items := []string{"One", "Two", "Three"}
	view := ForEach(items, func(item string, i int) View {
		return Text("%s", item)
	})

	w, h := view.size(100, 100)
	assert.Equal(t, 5, w) // "Three" is widest at 5 chars
	assert.Equal(t, 3, h) // 3 lines
}

// TestForEach_WithIndex tests that the index is passed correctly
func TestForEach_WithIndex(t *testing.T) {
	items := []string{"A", "B", "C"}
	view := ForEach(items, func(item string, i int) View {
		return Text("%d. %s", i+1, item)
	})

	var buf strings.Builder
	err := Print(view, PrintConfig{Width: 20, Output: &buf})
	assert.NoError(t, err)

	output := buf.String()
	assert.True(t, strings.Contains(output, "1. A"), "should contain 1. A")
	assert.True(t, strings.Contains(output, "2. B"), "should contain 2. B")
	assert.True(t, strings.Contains(output, "3. C"), "should contain 3. C")
}

// TestForEach_Separator tests the Separator method
func TestForEach_Separator(t *testing.T) {
	items := []string{"First", "Second", "Third"}
	view := ForEach(items, func(item string, i int) View {
		return Text("%s", item)
	}).Separator(Text("---"))

	w, h := view.size(100, 100)
	assert.Equal(t, 6, w) // "Second" is widest at 6 chars
	// 3 items + 2 separators = 5 lines
	assert.Equal(t, 5, h)
}

// TestForEach_SeparatorRendering tests that separators are rendered
func TestForEach_SeparatorRendering(t *testing.T) {
	items := []string{"A", "B"}
	view := ForEach(items, func(item string, i int) View {
		return Text("%s", item)
	}).Separator(Text("-"))

	var buf strings.Builder
	err := Print(view, PrintConfig{Width: 20, Output: &buf})
	assert.NoError(t, err)

	output := buf.String()
	assert.True(t, strings.Contains(output, "A"), "should contain A")
	assert.True(t, strings.Contains(output, "-"), "should contain separator")
	assert.True(t, strings.Contains(output, "B"), "should contain B")
}

// TestForEach_Gap tests the Gap method
func TestForEach_Gap(t *testing.T) {
	items := []string{"A", "B", "C"}
	view := ForEach(items, func(item string, i int) View {
		return Text("%s", item)
	}).Gap(2)

	w, h := view.size(100, 100)
	assert.Equal(t, 1, w) // single char width
	// 3 items + 2 gaps of 2 = 3 + 4 = 7
	assert.Equal(t, 7, h)
}

// TestForEach_GapWithSeparator tests Gap and Separator together
func TestForEach_GapWithSeparator(t *testing.T) {
	items := []string{"X", "Y"}
	view := ForEach(items, func(item string, i int) View {
		return Text("%s", item)
	}).Separator(Text("-")).Gap(1)

	w, h := view.size(100, 100)
	assert.Equal(t, 1, w)
	// 2 items + 1 separator + 2 gaps = 2 + 1 + 2 = 5
	assert.Equal(t, 5, h)
}

// TestForEach_Render tests the render behavior
func TestForEach_Render(t *testing.T) {
	items := []int{1, 2, 3}
	view := ForEach(items, func(item int, i int) View {
		return Text("Item %d", item)
	})

	var buf strings.Builder
	err := Print(view, PrintConfig{Width: 20, Output: &buf})
	assert.NoError(t, err)

	output := buf.String()
	assert.True(t, strings.Contains(output, "Item 1"), "should contain Item 1")
	assert.True(t, strings.Contains(output, "Item 2"), "should contain Item 2")
	assert.True(t, strings.Contains(output, "Item 3"), "should contain Item 3")
}

// TestForEach_ComplexViews tests ForEach with complex child views
func TestForEach_ComplexViews(t *testing.T) {
	items := []string{"Bold", "Italic", "Underline"}
	view := ForEach(items, func(item string, i int) View {
		switch i {
		case 0:
			return Text("%s", item).Bold()
		case 1:
			return Text("%s", item).Italic()
		default:
			return Text("%s", item).Underline()
		}
	})

	var buf strings.Builder
	err := Print(view, PrintConfig{Width: 40, Output: &buf})
	assert.NoError(t, err)

	output := buf.String()
	assert.True(t, strings.Contains(output, "Bold"), "should contain Bold")
	assert.True(t, strings.Contains(output, "Italic"), "should contain Italic")
	assert.True(t, strings.Contains(output, "Underline"), "should contain Underline")
}

// TestForEach_StructItems tests ForEach with struct items
func TestForEach_StructItems(t *testing.T) {
	type Task struct {
		Name string
		Done bool
	}

	tasks := []Task{
		{Name: "Write tests", Done: true},
		{Name: "Fix bugs", Done: false},
		{Name: "Deploy", Done: false},
	}

	view := ForEach(tasks, func(task Task, i int) View {
		status := "[ ]"
		if task.Done {
			status = "[x]"
		}
		return Text("%s %s", status, task.Name)
	})

	var buf strings.Builder
	err := Print(view, PrintConfig{Width: 40, Output: &buf})
	assert.NoError(t, err)

	output := buf.String()
	assert.True(t, strings.Contains(output, "[x] Write tests"), "should contain completed task")
	assert.True(t, strings.Contains(output, "[ ] Fix bugs"), "should contain incomplete task")
}

// TestHForEach_Empty tests HForEach with an empty slice
func TestHForEach_Empty(t *testing.T) {
	items := []string{}
	view := HForEach(items, func(item string, i int) View {
		return Text("%s", item)
	})

	w, h := view.size(100, 100)
	assert.Equal(t, 0, w)
	assert.Equal(t, 0, h)
}

// TestHForEach_SingleItem tests HForEach with a single item
func TestHForEach_SingleItem(t *testing.T) {
	items := []string{"Hello"}
	view := HForEach(items, func(item string, i int) View {
		return Text("%s", item)
	})

	w, h := view.size(100, 100)
	assert.Equal(t, 5, w) // "Hello" is 5 chars
	assert.Equal(t, 1, h) // single line
}

// TestHForEach_MultipleItems tests HForEach with multiple items
func TestHForEach_MultipleItems(t *testing.T) {
	items := []string{"One", "Two", "Three"}
	view := HForEach(items, func(item string, i int) View {
		return Text("%s", item)
	})

	w, h := view.size(100, 100)
	// Total width: 3 + 3 + 5 = 11
	assert.Equal(t, 11, w)
	assert.Equal(t, 1, h) // all on same line
}

// TestHForEach_WithIndex tests that the index is passed correctly
func TestHForEach_WithIndex(t *testing.T) {
	items := []string{"A", "B", "C"}
	view := HForEach(items, func(item string, i int) View {
		return Text("%d", i)
	})

	var buf strings.Builder
	err := Print(view, PrintConfig{Width: 20, Output: &buf})
	assert.NoError(t, err)

	output := buf.String()
	assert.True(t, strings.Contains(output, "0"), "should contain 0")
	assert.True(t, strings.Contains(output, "1"), "should contain 1")
	assert.True(t, strings.Contains(output, "2"), "should contain 2")
}

// TestHForEach_Separator tests the Separator method
func TestHForEach_Separator(t *testing.T) {
	items := []string{"A", "B", "C"}
	view := HForEach(items, func(item string, i int) View {
		return Text("%s", item)
	}).Separator(Text("|"))

	w, h := view.size(100, 100)
	// A + | + B + | + C = 1+1+1+1+1 = 5
	assert.Equal(t, 5, w)
	assert.Equal(t, 1, h)
}

// TestHForEach_SeparatorRendering tests that separators are rendered
func TestHForEach_SeparatorRendering(t *testing.T) {
	items := []string{"X", "Y", "Z"}
	view := HForEach(items, func(item string, i int) View {
		return Text("%s", item)
	}).Separator(Text(" | "))

	var buf strings.Builder
	err := Print(view, PrintConfig{Width: 40, Output: &buf})
	assert.NoError(t, err)

	output := buf.String()
	assert.True(t, strings.Contains(output, "X"), "should contain X")
	assert.True(t, strings.Contains(output, "|"), "should contain separator")
	assert.True(t, strings.Contains(output, "Y"), "should contain Y")
	assert.True(t, strings.Contains(output, "Z"), "should contain Z")
}

// TestHForEach_Gap tests the Gap method
func TestHForEach_Gap(t *testing.T) {
	items := []string{"A", "B", "C"}
	view := HForEach(items, func(item string, i int) View {
		return Text("%s", item)
	}).Gap(3)

	w, h := view.size(100, 100)
	// 3 chars + 2 gaps of 3 = 3 + 6 = 9
	assert.Equal(t, 9, w)
	assert.Equal(t, 1, h)
}

// TestHForEach_GapWithSeparator tests Gap and Separator together
func TestHForEach_GapWithSeparator(t *testing.T) {
	items := []string{"A", "B"}
	view := HForEach(items, func(item string, i int) View {
		return Text("%s", item)
	}).Separator(Text("|")).Gap(1)

	w, h := view.size(100, 100)
	// A + gap + | + gap + B = 1 + 1 + 1 + 1 + 1 = 5
	assert.Equal(t, 5, w)
	assert.Equal(t, 1, h)
}

// TestHForEach_Render tests the render behavior
func TestHForEach_Render(t *testing.T) {
	items := []string{"Red", "Green", "Blue"}
	view := HForEach(items, func(item string, i int) View {
		return Text("%s", item)
	})

	var buf strings.Builder
	err := Print(view, PrintConfig{Width: 40, Output: &buf})
	assert.NoError(t, err)

	output := buf.String()
	assert.True(t, strings.Contains(output, "Red"), "should contain Red")
	assert.True(t, strings.Contains(output, "Green"), "should contain Green")
	assert.True(t, strings.Contains(output, "Blue"), "should contain Blue")
}

// TestHForEach_TabItems tests HForEach with tab-like items
func TestHForEach_TabItems(t *testing.T) {
	tabs := []string{"Home", "Profile", "Settings"}
	view := HForEach(tabs, func(tab string, i int) View {
		return Bordered(Text(" %s ", tab))
	}).Gap(1)

	var buf strings.Builder
	err := Print(view, PrintConfig{Width: 60, Output: &buf})
	assert.NoError(t, err)

	output := buf.String()
	assert.True(t, strings.Contains(output, "Home"), "should contain Home")
	assert.True(t, strings.Contains(output, "Profile"), "should contain Profile")
	assert.True(t, strings.Contains(output, "Settings"), "should contain Settings")
}

// TestHForEach_ComplexViews tests HForEach with styled views
func TestHForEach_ComplexViews(t *testing.T) {
	items := []int{1, 2, 3}
	view := HForEach(items, func(item int, i int) View {
		return Text("[%d]", item).Bold().Fg(ColorBlue)
	}).Separator(Text(" "))

	var buf strings.Builder
	err := Print(view, PrintConfig{Width: 40, Output: &buf})
	assert.NoError(t, err)

	output := buf.String()
	assert.True(t, strings.Contains(output, "[1]"), "should contain [1]")
	assert.True(t, strings.Contains(output, "[2]"), "should contain [2]")
	assert.True(t, strings.Contains(output, "[3]"), "should contain [3]")
}

// TestForEach_CacheInvalidation tests that cache is cleared on size calls
func TestForEach_CacheInvalidation(t *testing.T) {
	counter := 0
	items := []string{"A", "B"}
	view := ForEach(items, func(item string, i int) View {
		counter++
		return Text("%s", item)
	})

	// First size call
	view.size(100, 100)
	firstCount := counter

	// Second size call should rebuild (cache cleared)
	view.size(100, 100)
	secondCount := counter

	// Counter should have incremented on both calls
	assert.Equal(t, 2, firstCount)
	assert.Equal(t, 4, secondCount)
}

// TestHForEach_CacheInvalidation tests that cache is cleared on size calls
func TestHForEach_CacheInvalidation(t *testing.T) {
	counter := 0
	items := []string{"X", "Y"}
	view := HForEach(items, func(item string, i int) View {
		counter++
		return Text("%s", item)
	})

	// First size call
	view.size(100, 100)
	firstCount := counter

	// Second size call should rebuild (cache cleared)
	view.size(100, 100)
	secondCount := counter

	// Counter should have incremented on both calls
	assert.Equal(t, 2, firstCount)
	assert.Equal(t, 4, secondCount)
}

// TestForEach_VaryingHeights tests ForEach with children of varying heights
func TestForEach_VaryingHeights(t *testing.T) {
	items := []int{1, 2, 3}
	view := ForEach(items, func(item int, i int) View {
		// Create multi-line text for some items
		if i == 1 {
			return Stack(
				Text("Line 1"),
				Text("Line 2"),
			)
		}
		return Text("Item %d", item)
	})

	_, h := view.size(100, 100)
	// Should account for the multi-line item
	assert.True(t, h >= 4, "height should account for multi-line items")
}

// TestHForEach_VaryingHeights tests HForEach with children of varying heights
func TestHForEach_VaryingHeights(t *testing.T) {
	items := []int{1, 2, 3}
	view := HForEach(items, func(item int, i int) View {
		// Create multi-line text for some items
		if i == 1 {
			return Stack(
				Text("Tall"),
				Text("Item"),
			)
		}
		return Text("%d", item)
	})

	_, h := view.size(100, 100)
	// Height should be the tallest item (2 lines)
	assert.Equal(t, 2, h)
}

// TestForEach_NoSeparatorOnFirst tests that separator is not added before first item
func TestForEach_NoSeparatorOnFirst(t *testing.T) {
	items := []string{"First"}
	view := ForEach(items, func(item string, i int) View {
		return Text("%s", item)
	}).Separator(Text("---"))

	w, h := view.size(100, 100)
	// Only one item, no separator
	assert.Equal(t, 5, w) // "First"
	assert.Equal(t, 1, h)
}

// TestHForEach_NoSeparatorOnFirst tests that separator is not added before first item
func TestHForEach_NoSeparatorOnFirst(t *testing.T) {
	items := []string{"Only"}
	view := HForEach(items, func(item string, i int) View {
		return Text("%s", item)
	}).Separator(Text("|"))

	w, h := view.size(100, 100)
	// Only one item, no separator
	assert.Equal(t, 4, w) // "Only"
	assert.Equal(t, 1, h)
}

// TestForEach_Chaining tests method chaining
func TestForEach_Chaining(t *testing.T) {
	items := []string{"A", "B", "C"}
	view := ForEach(items, func(item string, i int) View {
		return Text("%s", item)
	}).Separator(Text("-")).Gap(1)

	// Should return non-nil and allow chaining
	assert.NotNil(t, view)

	w, h := view.size(100, 100)
	assert.Equal(t, 1, w)
	// 5 views (3 items + 2 separators) + 4 gaps = 5 + 4 = 9
	assert.Equal(t, 9, h)
}

// TestHForEach_Chaining tests method chaining
func TestHForEach_Chaining(t *testing.T) {
	items := []string{"X", "Y"}
	view := HForEach(items, func(item string, i int) View {
		return Text("%s", item)
	}).Separator(Text("|")).Gap(2)

	// Should return non-nil and allow chaining
	assert.NotNil(t, view)

	w, h := view.size(100, 100)
	// 3 views (2 items + 1 separator) + 2 gaps of 2 = 3 + 4 = 7
	assert.Equal(t, 7, w)
	assert.Equal(t, 1, h)
}

// TestForEach_GenericTypes tests different generic type parameters
func TestForEach_GenericTypes(t *testing.T) {
	// Test with ints
	ints := []int{10, 20, 30}
	intView := ForEach(ints, func(item int, i int) View {
		return Text("%d", item)
	})
	assert.NotNil(t, intView)

	// Test with floats
	floats := []float64{1.5, 2.5, 3.5}
	floatView := ForEach(floats, func(item float64, i int) View {
		return Text("%.1f", item)
	})
	assert.NotNil(t, floatView)

	// Test with bools
	bools := []bool{true, false, true}
	boolView := ForEach(bools, func(item bool, i int) View {
		if item {
			return Text("Yes")
		}
		return Text("No")
	})
	assert.NotNil(t, boolView)
}

// TestHForEach_GenericTypes tests different generic type parameters
func TestHForEach_GenericTypes(t *testing.T) {
	// Test with runes
	runes := []rune{'A', 'B', 'C'}
	runeView := HForEach(runes, func(item rune, i int) View {
		return Text("%c", item)
	})
	assert.NotNil(t, runeView)

	// Test with bytes
	bytes := []byte{65, 66, 67}
	byteView := HForEach(bytes, func(item byte, i int) View {
		return Text("%d", item)
	})
	assert.NotNil(t, byteView)
}

// TestForEach_NoMaxConstraints tests ForEach with no size constraints
func TestForEach_NoMaxConstraints(t *testing.T) {
	items := []string{"One", "Two", "Three"}
	view := ForEach(items, func(item string, i int) View {
		return Text("%s", item)
	})

	w, h := view.size(0, 0)
	assert.Equal(t, 5, w) // "Three" is widest
	assert.Equal(t, 3, h)
}

// TestHForEach_NoMaxConstraints tests HForEach with no size constraints
func TestHForEach_NoMaxConstraints(t *testing.T) {
	items := []string{"A", "B", "C"}
	view := HForEach(items, func(item string, i int) View {
		return Text("%s", item)
	})

	w, h := view.size(0, 0)
	assert.Equal(t, 3, w) // total width
	assert.Equal(t, 1, h)
}
