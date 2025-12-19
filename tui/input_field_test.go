package tui

import (
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
	"github.com/deepnoodle-ai/wonton/termtest"
)

func TestInputField_Creation(t *testing.T) {
	value := "initial"
	field := InputField(&value)

	assert.NotNil(t, field)
	assert.Equal(t, &value, field.binding)
	assert.Equal(t, 20, field.width) // Default width
	assert.NotEmpty(t, field.id)     // Auto-generated ID
}

func TestInputField_NilBinding(t *testing.T) {
	field := InputField(nil)

	assert.NotNil(t, field)
	assert.Nil(t, field.binding)
	assert.Empty(t, field.id)
}

func TestGenerateInputID(t *testing.T) {
	value := "test"
	id := generateInputID(&value)

	assert.NotEmpty(t, id)
	assert.Contains(t, id, "input_")

	// Different pointers should give different IDs
	value2 := "test"
	id2 := generateInputID(&value2)
	assert.NotEqual(t, id, id2)
}

func TestInputField_ID(t *testing.T) {
	value := ""
	field := InputField(&value).ID("my-input")
	assert.Equal(t, "my-input", field.id)
}

func TestInputField_Label(t *testing.T) {
	value := ""
	field := InputField(&value).Label("Name:")
	assert.Equal(t, "Name:", field.label)
}

func TestInputField_LabelStyle(t *testing.T) {
	value := ""
	style := NewStyle().WithForeground(ColorYellow)
	field := InputField(&value).LabelStyle(style)
	assert.Equal(t, style, field.labelStyle)
}

func TestInputField_FocusLabelStyle(t *testing.T) {
	value := ""
	style := NewStyle().WithForeground(ColorCyan).WithBold()
	field := InputField(&value).FocusLabelStyle(style)
	assert.NotNil(t, field.focusLabelStyle)
	assert.Equal(t, style, *field.focusLabelStyle)
}

func TestInputField_Placeholder(t *testing.T) {
	value := ""
	field := InputField(&value).Placeholder("Enter text...")
	assert.Equal(t, "Enter text...", field.placeholder)
}

func TestInputField_PlaceholderStyle(t *testing.T) {
	value := ""
	style := NewStyle().WithForeground(ColorBrightBlack)
	field := InputField(&value).PlaceholderStyle(style)
	assert.NotNil(t, field.placeholderStyle)
	assert.Equal(t, style, *field.placeholderStyle)
}

func TestInputField_Mask(t *testing.T) {
	value := ""
	field := InputField(&value).Mask('*')
	assert.Equal(t, '*', field.mask)
}

func TestInputField_OnChange(t *testing.T) {
	value := ""
	called := false
	field := InputField(&value).OnChange(func(s string) { called = true })
	assert.NotNil(t, field.onChange)

	// Call to verify it was set
	field.onChange("test")
	assert.True(t, called)
}

func TestInputField_OnSubmit(t *testing.T) {
	value := ""
	called := false
	field := InputField(&value).OnSubmit(func(s string) { called = true })
	assert.NotNil(t, field.onSubmit)

	// Call to verify it was set
	field.onSubmit("test")
	assert.True(t, called)
}

func TestInputField_Width(t *testing.T) {
	value := ""
	field := InputField(&value).Width(40)
	assert.Equal(t, 40, field.width)
}

func TestInputField_MaxHeight(t *testing.T) {
	value := ""
	field := InputField(&value).MaxHeight(5)
	assert.Equal(t, 5, field.maxHeight)
}

func TestInputField_PastePlaceholder(t *testing.T) {
	value := ""
	field := InputField(&value).PastePlaceholder(true)
	assert.True(t, field.pastePlaceholder)

	field = InputField(&value).PastePlaceholder(false)
	assert.False(t, field.pastePlaceholder)
}

func TestInputField_CursorBlink(t *testing.T) {
	value := ""
	field := InputField(&value).CursorBlink(true)
	assert.True(t, field.cursorBlink)

	field = InputField(&value).CursorBlink(false)
	assert.False(t, field.cursorBlink)
}

func TestInputField_Multiline(t *testing.T) {
	value := ""
	field := InputField(&value).Multiline(true)
	assert.True(t, field.multiline)

	field = InputField(&value).Multiline(false)
	assert.False(t, field.multiline)
}

func TestInputField_Bordered(t *testing.T) {
	value := ""
	field := InputField(&value).Bordered()
	assert.True(t, field.bordered)
	assert.NotNil(t, field.border) // Default border set
	assert.Equal(t, &RoundedBorder, field.border)
}

func TestInputField_Border(t *testing.T) {
	value := ""
	field := InputField(&value).Border(&DoubleBorder)
	assert.True(t, field.bordered) // Border() implies Bordered()
	assert.Equal(t, &DoubleBorder, field.border)
}

func TestInputField_BorderFg(t *testing.T) {
	value := ""
	field := InputField(&value).BorderFg(ColorMagenta)
	assert.Equal(t, ColorMagenta, field.borderFg)
}

func TestInputField_FocusBorderFg(t *testing.T) {
	value := ""
	field := InputField(&value).FocusBorderFg(ColorCyan)
	assert.Equal(t, ColorCyan, field.focusBorderFg)
	assert.True(t, field.hasFocusBorder)
}

func TestInputField_HorizontalBorderOnly(t *testing.T) {
	value := ""
	field := InputField(&value).HorizontalBorderOnly()
	assert.True(t, field.bordered) // Implies bordered
	assert.True(t, field.horizontalBarOnly)
}

func TestInputField_Prompt(t *testing.T) {
	value := ""
	field := InputField(&value).Prompt(">")
	assert.Equal(t, ">", field.prompt)
	assert.True(t, field.hasPrompt)
}

func TestInputField_PromptStyle(t *testing.T) {
	value := ""
	style := NewStyle().WithForeground(ColorGreen)
	field := InputField(&value).PromptStyle(style)
	assert.Equal(t, style, field.promptStyle)
}

func TestInputField_CursorShape(t *testing.T) {
	value := ""
	field := InputField(&value).CursorShape(InputCursorUnderline)
	assert.Equal(t, InputCursorUnderline, field.cursorShape)

	field = InputField(&value).CursorShape(InputCursorBar)
	assert.Equal(t, InputCursorBar, field.cursorShape)
}

func TestInputField_CursorColor(t *testing.T) {
	value := ""
	field := InputField(&value).CursorColor(ColorYellow)
	assert.NotNil(t, field.cursorColor)
	assert.Equal(t, ColorYellow, *field.cursorColor)
}

func TestInputField_Chaining(t *testing.T) {
	value := ""
	field := InputField(&value).
		ID("email-input").
		Label("Email:").
		Placeholder("user@example.com").
		Width(50).
		Bordered().
		BorderFg(ColorWhite).
		FocusBorderFg(ColorCyan)

	assert.Equal(t, "email-input", field.id)
	assert.Equal(t, "Email:", field.label)
	assert.Equal(t, "user@example.com", field.placeholder)
	assert.Equal(t, 50, field.width)
	assert.True(t, field.bordered)
	assert.Equal(t, ColorWhite, field.borderFg)
	assert.Equal(t, ColorCyan, field.focusBorderFg)
}

func TestInputField_Size_Basic(t *testing.T) {
	value := ""
	field := InputField(&value).Width(30)

	w, h := field.size(100, 100)
	assert.Equal(t, 30, w)
	assert.Equal(t, 1, h) // Single line for empty input
}

func TestInputField_Size_WithBorder(t *testing.T) {
	value := ""
	field := InputField(&value).Width(30).Bordered()

	w, h := field.size(100, 100)
	assert.Equal(t, 30, w)
	assert.Equal(t, 3, h) // 1 line + 2 for border (top/bottom)
}

func TestInputField_Size_WithHorizontalBorder(t *testing.T) {
	value := ""
	field := InputField(&value).Width(30).HorizontalBorderOnly()

	w, h := field.size(100, 100)
	assert.Equal(t, 30, w)
	assert.Equal(t, 3, h) // 1 line + 2 for horizontal bars
}

func TestInputField_Size_WithLabel_NoBorder(t *testing.T) {
	value := ""
	field := InputField(&value).Width(30).Label("Name: ")

	w, h := field.size(100, 100)
	// Label is 6 chars + input width 30 = 36 when not bordered
	assert.Equal(t, 36, w)
	assert.Equal(t, 1, h)
}

func TestInputField_Size_WithLabel_Bordered(t *testing.T) {
	value := ""
	field := InputField(&value).Width(30).Label("Name:").Bordered()

	w, h := field.size(100, 100)
	// Label is embedded in border, so just input width
	assert.Equal(t, 30, w)
	assert.Equal(t, 3, h)
}

func TestInputField_Size_WithPrompt(t *testing.T) {
	value := ""
	field := InputField(&value).Width(30).Prompt(">")

	w, h := field.size(100, 100)
	// Prompt adds 2 chars ("> " with space)
	assert.Equal(t, 32, w)
	assert.Equal(t, 1, h)
}

func TestInputField_Size_Constrained(t *testing.T) {
	value := ""
	field := InputField(&value).Width(100)

	w, h := field.size(50, 10)
	assert.Equal(t, 50, w) // Constrained by maxWidth
	assert.LessOrEqual(t, h, 10)
}

func TestInputField_Size_WithMaxHeight(t *testing.T) {
	value := "line1\nline2\nline3\nline4\nline5"
	field := InputField(&value).Width(30).MaxHeight(3)

	w, h := field.size(100, 100)
	assert.Equal(t, 30, w)
	assert.LessOrEqual(t, h, 3) // Constrained by maxHeight
}

func TestInputField_PasswordInput(t *testing.T) {
	value := ""
	field := InputField(&value).
		Label("Password:").
		Mask('*').
		Width(30)

	assert.Equal(t, '*', field.mask)
	assert.Equal(t, "Password:", field.label)
}

func TestInputField_MultilineInput(t *testing.T) {
	value := ""
	field := InputField(&value).
		Multiline(true).
		MaxHeight(10).
		Width(60)

	assert.True(t, field.multiline)
	assert.Equal(t, 10, field.maxHeight)
}

func TestInputCursorStyle_Values(t *testing.T) {
	// Verify cursor style constants
	assert.Equal(t, InputCursorStyle(0), InputCursorBlock)
	assert.Equal(t, InputCursorStyle(1), InputCursorUnderline)
	assert.Equal(t, InputCursorStyle(2), InputCursorBar)
}

// Render tests using termtest with SprintScreen helper

func TestInputField_Render_WithValue(t *testing.T) {
	value := "hello"
	field := InputField(&value).Width(20)
	screen := SprintScreen(field, WithWidth(30))
	termtest.AssertRowContains(t, screen, 0, "hello")
}

func TestInputField_Render_WithLabel(t *testing.T) {
	value := "test"
	field := InputField(&value).Label("Name: ").Width(20)
	screen := SprintScreen(field, WithWidth(40))

	termtest.AssertRowContains(t, screen, 0, "Name:")
	termtest.AssertRowContains(t, screen, 0, "test")
}

func TestInputField_Render_WithBorder(t *testing.T) {
	value := "content"
	field := InputField(&value).Width(20).Bordered()
	screen := SprintScreen(field, WithWidth(30))

	// Check for border characters (rounded border uses ╭, ╮, ╰, ╯)
	termtest.AssertRowContains(t, screen, 0, "╭")
	termtest.AssertRowContains(t, screen, 1, "content")
	termtest.AssertRowContains(t, screen, 2, "╰")
}

func TestInputField_Render_WithLabelAndBorder(t *testing.T) {
	value := "john"
	field := InputField(&value).Label("Username:").Width(25).Bordered()
	screen := SprintScreen(field, WithWidth(30))

	// Label should be embedded in the top border
	termtest.AssertRowContains(t, screen, 0, "Username")
	termtest.AssertRowContains(t, screen, 1, "john")
}

func TestInputField_Render_WithPrompt(t *testing.T) {
	value := "command"
	field := InputField(&value).Prompt(">").Width(20)
	screen := SprintScreen(field, WithWidth(30))

	termtest.AssertRowContains(t, screen, 0, ">")
	termtest.AssertRowContains(t, screen, 0, "command")
}

func TestInputField_Render_Placeholder(t *testing.T) {
	value := ""
	field := InputField(&value).Placeholder("Enter name...").Width(20)
	screen := SprintScreen(field, WithWidth(30))
	termtest.AssertRowContains(t, screen, 0, "Enter name...")
}

func TestInputField_Render_InStack(t *testing.T) {
	name := "Alice"
	email := "alice@example.com"

	form := Stack(
		InputField(&name).Label("Name: ").Width(20),
		InputField(&email).Label("Email: ").Width(25),
	)
	screen := SprintScreen(form, WithWidth(40))

	termtest.AssertRowContains(t, screen, 0, "Name:")
	termtest.AssertRowContains(t, screen, 0, "Alice")
	termtest.AssertRowContains(t, screen, 1, "Email:")
	termtest.AssertRowContains(t, screen, 1, "alice@example.com")
}

func TestInputField_Render_HorizontalBorder(t *testing.T) {
	value := "text"
	field := InputField(&value).Width(20).HorizontalBorderOnly()
	screen := SprintScreen(field, WithWidth(30))

	// Check for horizontal border character
	termtest.AssertRowContains(t, screen, 0, "─")
	termtest.AssertRowContains(t, screen, 1, "text")
	termtest.AssertRowContains(t, screen, 2, "─")
}
