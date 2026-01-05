package tui

import (
	"strings"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

// If() function tests

func TestIf_True(t *testing.T) {
	view := If(true, Text("Content"))
	assert.NotNil(t, view)

	// Should return the view, not Empty
	w, h := view.size(100, 100)
	assert.Equal(t, 7, w) // "Content"
	assert.Equal(t, 1, h)
}

func TestIf_False(t *testing.T) {
	view := If(false, Text("Content"))
	assert.NotNil(t, view)

	// Should return Empty()
	w, h := view.size(100, 100)
	assert.Equal(t, 0, w)
	assert.Equal(t, 0, h)
}

func TestIf_RenderTrue(t *testing.T) {
	var buf strings.Builder
	view := If(true, Text("Visible"))

	err := Print(view, PrintConfig{Width: 80, Output: &buf})
	assert.NoError(t, err)

	output := buf.String()
	assert.True(t, strings.Contains(output, "Visible"), "should contain text when true")
}

func TestIf_RenderFalse(t *testing.T) {
	var buf strings.Builder
	view := If(false, Text("Hidden"))

	err := Print(view, PrintConfig{Width: 80, Output: &buf})
	assert.NoError(t, err)

	output := buf.String()
	assert.False(t, strings.Contains(output, "Hidden"), "should not contain text when false")
}

func TestIf_InStack(t *testing.T) {
	stack := Stack(
		Text("Always"),
		If(true, Text("Sometimes")),
		If(false, Text("Never")),
		Text("Also always"),
	)

	w, h := stack.size(100, 100)
	assert.Equal(t, 11, w) // "Also always" is longest (11 chars)
	assert.Equal(t, 3, h)  // "Always", "Sometimes", "Also always" (Empty contributes 0)
}

func TestIf_WithStyledView(t *testing.T) {
	var buf strings.Builder
	view := If(true, Text("Warning").Fg(ColorYellow).Bold())

	err := Print(view, PrintConfig{Width: 80, Output: &buf})
	assert.NoError(t, err)

	output := buf.String()
	assert.True(t, strings.Contains(output, "Warning"), "should contain text")
	assert.True(t, strings.Contains(output, "\033["), "should contain ANSI escape codes")
}

func TestIf_WithComplexView(t *testing.T) {
	view := If(true, Stack(
		Text("Line 1"),
		Text("Line 2"),
	))

	w, h := view.size(100, 100)
	assert.Equal(t, 6, w)
	assert.Equal(t, 2, h)
}

// IfElse() function tests

func TestIfElse_True(t *testing.T) {
	view := IfElse(true, Text("Then"), Text("Else"))
	assert.NotNil(t, view)

	w, h := view.size(100, 100)
	assert.Equal(t, 4, w) // "Then"
	assert.Equal(t, 1, h)
}

func TestIfElse_False(t *testing.T) {
	view := IfElse(false, Text("Then"), Text("Else"))
	assert.NotNil(t, view)

	w, h := view.size(100, 100)
	assert.Equal(t, 4, w) // "Else"
	assert.Equal(t, 1, h)
}

func TestIfElse_RenderTrue(t *testing.T) {
	var buf strings.Builder
	view := IfElse(true, Text("Logged in"), Text("Not logged in"))

	err := Print(view, PrintConfig{Width: 80, Output: &buf})
	assert.NoError(t, err)

	output := buf.String()
	assert.True(t, strings.Contains(output, "Logged in"), "should contain then view")
	assert.False(t, strings.Contains(output, "Not logged in"), "should not contain else view")
}

func TestIfElse_RenderFalse(t *testing.T) {
	var buf strings.Builder
	view := IfElse(false, Text("Logged in"), Text("Not logged in"))

	err := Print(view, PrintConfig{Width: 80, Output: &buf})
	assert.NoError(t, err)

	output := buf.String()
	assert.False(t, strings.Contains(output, "Logged in"), "should not contain then view")
	assert.True(t, strings.Contains(output, "Not logged in"), "should contain else view")
}

func TestIfElse_DifferentSizes(t *testing.T) {
	// Test that the returned view has the size of the selected branch
	view1 := IfElse(true, Text("Short"), Text("Much longer text"))
	w1, h1 := view1.size(100, 100)
	assert.Equal(t, 5, w1) // "Short"
	assert.Equal(t, 1, h1)

	view2 := IfElse(false, Text("Short"), Text("Much longer text"))
	w2, h2 := view2.size(100, 100)
	assert.Equal(t, 16, w2) // "Much longer text"
	assert.Equal(t, 1, h2)
}

func TestIfElse_WithComplexViews(t *testing.T) {
	view := IfElse(
		true,
		Stack(Text("A"), Text("B"), Text("C")),
		Stack(Text("X"), Text("Y")),
	)

	w, h := view.size(100, 100)
	assert.Equal(t, 1, w)
	assert.Equal(t, 3, h) // Then branch has 3 items

	view2 := IfElse(
		false,
		Stack(Text("A"), Text("B"), Text("C")),
		Stack(Text("X"), Text("Y")),
	)

	w2, h2 := view2.size(100, 100)
	assert.Equal(t, 1, w2)
	assert.Equal(t, 2, h2) // Else branch has 2 items
}

func TestIfElse_InStack(t *testing.T) {
	stack := Stack(
		Text("Header"),
		IfElse(true, Text("True branch"), Text("False branch")),
		Text("Footer"),
	)

	var buf strings.Builder
	err := Print(stack, PrintConfig{Width: 80, Output: &buf})
	assert.NoError(t, err)

	output := buf.String()
	assert.True(t, strings.Contains(output, "Header"), "should contain header")
	assert.True(t, strings.Contains(output, "True branch"), "should contain true branch")
	assert.False(t, strings.Contains(output, "False branch"), "should not contain false branch")
	assert.True(t, strings.Contains(output, "Footer"), "should contain footer")
}

// Switch/Case/Default tests

func TestCase(t *testing.T) {
	c := Case("value", Text("View"))
	assert.Equal(t, "value", c.value)
	assert.NotNil(t, c.view)
	assert.False(t, c.isDefault)
}

func TestDefault(t *testing.T) {
	d := Default[string](Text("Default view"))
	assert.NotNil(t, d.view)
	assert.True(t, d.isDefault)
}

func TestSwitch_MatchFirst(t *testing.T) {
	view := Switch("loading",
		Case("loading", Text("Loading...")),
		Case("error", Text("Error!")),
		Case("ready", Text("Ready")),
	)

	assert.NotNil(t, view)
	w, h := view.size(100, 100)
	assert.Equal(t, 10, w) // "Loading..."
	assert.Equal(t, 1, h)
}

func TestSwitch_MatchMiddle(t *testing.T) {
	view := Switch("error",
		Case("loading", Text("Loading...")),
		Case("error", Text("Error!")),
		Case("ready", Text("Ready")),
	)

	w, h := view.size(100, 100)
	assert.Equal(t, 6, w) // "Error!"
	assert.Equal(t, 1, h)
}

func TestSwitch_MatchLast(t *testing.T) {
	view := Switch("ready",
		Case("loading", Text("Loading...")),
		Case("error", Text("Error!")),
		Case("ready", Text("Ready")),
	)

	w, h := view.size(100, 100)
	assert.Equal(t, 5, w) // "Ready"
	assert.Equal(t, 1, h)
}

func TestSwitch_NoMatch_NoDefault(t *testing.T) {
	view := Switch("unknown",
		Case("loading", Text("Loading...")),
		Case("error", Text("Error!")),
		Case("ready", Text("Ready")),
	)

	// Should return Empty()
	w, h := view.size(100, 100)
	assert.Equal(t, 0, w)
	assert.Equal(t, 0, h)
}

func TestSwitch_NoMatch_WithDefault(t *testing.T) {
	view := Switch("unknown",
		Case("loading", Text("Loading...")),
		Case("error", Text("Error!")),
		Case("ready", Text("Ready")),
		Default[string](Text("Unknown state")),
	)

	w, h := view.size(100, 100)
	assert.Equal(t, 13, w) // "Unknown state"
	assert.Equal(t, 1, h)
}

func TestSwitch_DefaultPosition(t *testing.T) {
	// Default can be anywhere in the list
	view := Switch("unknown",
		Default[string](Text("Unknown")),
		Case("loading", Text("Loading...")),
		Case("error", Text("Error!")),
	)

	w, h := view.size(100, 100)
	assert.Equal(t, 7, w) // "Unknown"
	assert.Equal(t, 1, h)
}

func TestSwitch_MultipleDefaults(t *testing.T) {
	// If multiple defaults, the last one wins
	view := Switch("unknown",
		Case("loading", Text("Loading...")),
		Default[string](Text("First default")),
		Case("error", Text("Error!")),
		Default[string](Text("Second default")),
	)

	w, h := view.size(100, 100)
	assert.Equal(t, 14, w) // "Second default"
	assert.Equal(t, 1, h)
}

func TestSwitch_RenderMatch(t *testing.T) {
	var buf strings.Builder
	view := Switch("error",
		Case("loading", Text("Loading...").Fg(ColorYellow)),
		Case("error", Text("Error!").Fg(ColorRed)),
		Case("ready", Text("Ready").Fg(ColorGreen)),
		Default[string](Text("Unknown")),
	)

	err := Print(view, PrintConfig{Width: 80, Output: &buf})
	assert.NoError(t, err)

	output := buf.String()
	assert.True(t, strings.Contains(output, "Error!"), "should contain matched case")
	assert.False(t, strings.Contains(output, "Loading"), "should not contain other cases")
	assert.False(t, strings.Contains(output, "Ready"), "should not contain other cases")
	assert.False(t, strings.Contains(output, "Unknown"), "should not contain default")
}

func TestSwitch_RenderNoMatchWithDefault(t *testing.T) {
	var buf strings.Builder
	view := Switch("unknown",
		Case("loading", Text("Loading...")),
		Case("error", Text("Error!")),
		Default[string](Text("Unknown state")),
	)

	err := Print(view, PrintConfig{Width: 80, Output: &buf})
	assert.NoError(t, err)

	output := buf.String()
	assert.True(t, strings.Contains(output, "Unknown state"), "should contain default")
	assert.False(t, strings.Contains(output, "Loading"), "should not contain cases")
	assert.False(t, strings.Contains(output, "Error"), "should not contain cases")
}

func TestSwitch_RenderNoMatchNoDefault(t *testing.T) {
	var buf strings.Builder
	view := Switch("unknown",
		Case("loading", Text("Loading...")),
		Case("error", Text("Error!")),
	)

	err := Print(view, PrintConfig{Width: 80, Output: &buf})
	assert.NoError(t, err)

	output := buf.String()
	assert.False(t, strings.Contains(output, "Loading"), "should not contain cases")
	assert.False(t, strings.Contains(output, "Error"), "should not contain cases")
}

func TestSwitch_IntType(t *testing.T) {
	view := Switch(2,
		Case(0, Text("Zero")),
		Case(1, Text("One")),
		Case(2, Text("Two")),
		Default[int](Text("Other")),
	)

	w, h := view.size(100, 100)
	assert.Equal(t, 3, w) // "Two"
	assert.Equal(t, 1, h)
}

func TestSwitch_BoolType(t *testing.T) {
	view := Switch(true,
		Case(true, Text("Yes")),
		Case(false, Text("No")),
	)

	w, h := view.size(100, 100)
	assert.Equal(t, 3, w) // "Yes"
	assert.Equal(t, 1, h)
}

func TestSwitch_WithComplexViews(t *testing.T) {
	view := Switch("loading",
		Case("loading", Stack(
			Text("Loading..."),
			Text("Please wait"),
		)),
		Case("error", Stack(
			Text("Error occurred"),
			Text("Try again"),
		)),
		Default[string](Text("Done")),
	)

	w, h := view.size(100, 100)
	assert.Equal(t, 11, w) // "Please wait"
	assert.Equal(t, 2, h)
}

func TestSwitch_InStack(t *testing.T) {
	status := "ready"
	stack := Stack(
		Text("Application Status:"),
		Switch(status,
			Case("loading", Text("Loading...").Fg(ColorYellow)),
			Case("error", Text("Error!").Fg(ColorRed)),
			Case("ready", Text("Ready").Fg(ColorGreen)),
		),
	)

	var buf strings.Builder
	err := Print(stack, PrintConfig{Width: 80, Output: &buf})
	assert.NoError(t, err)

	output := buf.String()
	assert.True(t, strings.Contains(output, "Application Status:"), "should contain label")
	assert.True(t, strings.Contains(output, "Ready"), "should contain matched case")
	assert.False(t, strings.Contains(output, "Loading"), "should not contain other cases")
}

func TestSwitch_EmptyValue(t *testing.T) {
	view := Switch("",
		Case("", Text("Empty string")),
		Case("value", Text("Has value")),
		Default[string](Text("Default")),
	)

	w, h := view.size(100, 100)
	assert.Equal(t, 12, w) // "Empty string"
	assert.Equal(t, 1, h)
}

func TestSwitch_ZeroValue(t *testing.T) {
	view := Switch(0,
		Case(0, Text("Zero")),
		Case(1, Text("One")),
		Default[int](Text("Other")),
	)

	w, h := view.size(100, 100)
	assert.Equal(t, 4, w) // "Zero"
	assert.Equal(t, 1, h)
}

// Integration tests

func TestConditionals_Combined(t *testing.T) {
	isLoggedIn := true
	status := "ready"
	showDebug := false

	view := Stack(
		If(isLoggedIn, Text("Logged in as user")),
		IfElse(isLoggedIn, Text("Welcome!"), Text("Please log in")),
		Switch(status,
			Case("loading", Text("Loading...")),
			Case("ready", Text("Ready")),
			Default[string](Text("Unknown")),
		),
		If(showDebug, Text("Debug info")),
	)

	var buf strings.Builder
	err := Print(view, PrintConfig{Width: 80, Output: &buf})
	assert.NoError(t, err)

	output := buf.String()
	assert.True(t, strings.Contains(output, "Logged in as user"), "should show if true")
	assert.True(t, strings.Contains(output, "Welcome!"), "should show then branch")
	assert.True(t, strings.Contains(output, "Ready"), "should show matched case")
	assert.False(t, strings.Contains(output, "Debug info"), "should not show if false")
	assert.False(t, strings.Contains(output, "Please log in"), "should not show else branch")
}

func TestConditionals_Nested(t *testing.T) {
	outer := true
	inner := false

	view := If(outer, Stack(
		Text("Outer is true"),
		IfElse(inner,
			Text("Inner is true"),
			Text("Inner is false"),
		),
	))

	var buf strings.Builder
	err := Print(view, PrintConfig{Width: 80, Output: &buf})
	assert.NoError(t, err)

	output := buf.String()
	assert.True(t, strings.Contains(output, "Outer is true"), "should show outer")
	assert.True(t, strings.Contains(output, "Inner is false"), "should show nested else")
	assert.False(t, strings.Contains(output, "Inner is true"), "should not show nested then")
}

func TestConditionals_WithStyling(t *testing.T) {
	hasError := true
	view := If(hasError, Text("Warning!").Fg(ColorRed).Bold())

	var buf strings.Builder
	err := Print(view, PrintConfig{Width: 80, Output: &buf})
	assert.NoError(t, err)

	output := buf.String()
	assert.True(t, strings.Contains(output, "Warning!"), "should contain text")
	assert.True(t, strings.Contains(output, "\033["), "should contain ANSI codes for styling")
}
