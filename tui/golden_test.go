package tui

import (
	"testing"

	"github.com/deepnoodle-ai/wonton/termtest"
)

// Golden tests compare rendered TUI output against saved snapshot files.
// Run with -update to create/update snapshots: go test -update ./tui/...
// Snapshots are stored in testdata/snapshots/

// =============================================================================
// TEXT VIEW TESTS - Edge cases for text rendering
// =============================================================================

func TestGolden_Text_Simple(t *testing.T) {
	view := Text("Hello, World!")
	screen := SprintScreen(view, WithWidth(20))
	termtest.AssertScreen(t, screen)
}

func TestGolden_Text_Empty(t *testing.T) {
	// Empty text should render without crashing
	view := Text("")
	screen := SprintScreen(view, WithWidth(20), WithHeight(3))
	termtest.AssertScreen(t, screen)
}

func TestGolden_Text_Unicode_CJK(t *testing.T) {
	// CJK characters are double-width
	view := Text("Hello: \xe4\xb8\xad\xe6\x96\x87\xe6\xb5\x8b\xe8\xaf\x95") // "中文测试"
	screen := SprintScreen(view, WithWidth(25))
	termtest.AssertScreen(t, screen)
}

func TestGolden_Text_Emoji(t *testing.T) {
	// Emoji are typically double-width
	view := Text("Status: \xe2\x9c\x85 Done") // checkmark emoji
	screen := SprintScreen(view, WithWidth(20))
	termtest.AssertScreen(t, screen)
}

func TestGolden_Text_Truncation(t *testing.T) {
	// Text too long for container should be truncated
	view := Text("This is a very long line that should be truncated")
	screen := SprintScreen(view, WithWidth(20))
	termtest.AssertScreen(t, screen)
}

func TestGolden_Text_MultipleStyles(t *testing.T) {
	// Multiple style attributes combined
	view := Text("Important!").Bold().Underline()
	screen := SprintScreen(view, WithWidth(20))
	termtest.AssertScreen(t, screen)
}

func TestGolden_Text_SemanticStyles(t *testing.T) {
	// Semantic styling (Success, Error, Warning, Info)
	view := Stack(
		Text("Success message").Success(),
		Text("Error message").Error(),
		Text("Warning message").Warning(),
		Text("Info message").Info(),
	)
	screen := SprintScreen(view, WithWidth(25))
	termtest.AssertScreen(t, screen)
}

// =============================================================================
// STACK LAYOUT TESTS - Vertical layout edge cases
// =============================================================================

func TestGolden_Stack_Basic(t *testing.T) {
	view := Stack(
		Text("Line 1"),
		Text("Line 2"),
		Text("Line 3"),
	)
	screen := SprintScreen(view, WithWidth(20))
	termtest.AssertScreen(t, screen)
}

func TestGolden_Stack_Empty(t *testing.T) {
	// Empty stack should handle gracefully
	view := Stack()
	screen := SprintScreen(view, WithWidth(20), WithHeight(3))
	termtest.AssertScreen(t, screen)
}

func TestGolden_Stack_Gap(t *testing.T) {
	// Gap between items
	view := Stack(
		Text("First"),
		Text("Second"),
		Text("Third"),
	).Gap(1)
	screen := SprintScreen(view, WithWidth(20))
	termtest.AssertScreen(t, screen)
}

func TestGolden_Stack_AlignCenter(t *testing.T) {
	// Centered alignment with varying widths
	view := Stack(
		Text("X"),
		Text("XXX"),
		Text("XXXXX"),
		Text("XXX"),
		Text("X"),
	).Align(AlignCenter)
	screen := SprintScreen(view, WithWidth(15))
	termtest.AssertScreen(t, screen)
}

func TestGolden_Stack_AlignRight(t *testing.T) {
	// Right alignment
	view := Stack(
		Text("Short"),
		Text("Medium text"),
		Text("Longer text here"),
	).Align(AlignRight)
	screen := SprintScreen(view, WithWidth(20))
	termtest.AssertScreen(t, screen)
}

func TestGolden_Stack_WithSpacer(t *testing.T) {
	// Spacer pushes content to edges
	view := Stack(
		Text("Top"),
		Spacer(),
		Text("Bottom"),
	)
	screen := SprintScreen(view, WithWidth(15), WithHeight(7))
	termtest.AssertScreen(t, screen)
}

func TestGolden_Stack_MultipleSpacer(t *testing.T) {
	// Multiple spacers distribute space
	view := Stack(
		Text("A"),
		Spacer(),
		Text("B"),
		Spacer(),
		Text("C"),
	)
	screen := SprintScreen(view, WithWidth(10), WithHeight(9))
	termtest.AssertScreen(t, screen)
}

// =============================================================================
// GROUP LAYOUT TESTS - Horizontal layout edge cases
// =============================================================================

func TestGolden_Group_Basic(t *testing.T) {
	view := Group(
		Text("Left"),
		Text("Right"),
	)
	screen := SprintScreen(view, WithWidth(20))
	termtest.AssertScreen(t, screen)
}

func TestGolden_Group_Gap(t *testing.T) {
	view := Group(
		Text("A"),
		Text("B"),
		Text("C"),
	).Gap(3)
	screen := SprintScreen(view, WithWidth(20))
	termtest.AssertScreen(t, screen)
}

func TestGolden_Group_WithSpacer(t *testing.T) {
	// Spacer in Group pushes to edges horizontally
	view := Group(
		Text("Left"),
		Spacer(),
		Text("Right"),
	)
	screen := SprintScreen(view, WithWidth(30))
	termtest.AssertScreen(t, screen)
}

func TestGolden_Group_VerticalAlign(t *testing.T) {
	// Vertical alignment when children have different heights
	view := Group(
		Stack(Text("A"), Text("B"), Text("C")),
		Text("X"),
	).Align(AlignCenter)
	screen := SprintScreen(view, WithWidth(15))
	termtest.AssertScreen(t, screen)
}

// =============================================================================
// ZSTACK TESTS - Layered layout
// =============================================================================

func TestGolden_ZStack_Layers(t *testing.T) {
	// First child is background, last is foreground
	view := ZStack(
		Fill('.'),
		Text("Overlay"),
	)
	screen := SprintScreen(view, WithWidth(15), WithHeight(3))
	termtest.AssertScreen(t, screen)
}

func TestGolden_ZStack_CenterAlign(t *testing.T) {
	view := ZStack(
		Fill('-'),
		Text("Centered"),
	).Align(AlignCenter)
	screen := SprintScreen(view, WithWidth(20), WithHeight(5))
	termtest.AssertScreen(t, screen)
}

// =============================================================================
// BORDER & PADDING TESTS
// =============================================================================

func TestGolden_Border_Rounded(t *testing.T) {
	view := Bordered(Text("Rounded")).Border(&RoundedBorder)
	screen := SprintScreen(view, WithWidth(20))
	termtest.AssertScreen(t, screen)
}

func TestGolden_Border_Thick(t *testing.T) {
	view := Bordered(Text("Thick")).Border(&ThickBorder)
	screen := SprintScreen(view, WithWidth(20))
	termtest.AssertScreen(t, screen)
}

func TestGolden_Border_ASCII(t *testing.T) {
	view := Bordered(Text("ASCII")).Border(&ASCIIBorder)
	screen := SprintScreen(view, WithWidth(20))
	termtest.AssertScreen(t, screen)
}

func TestGolden_Border_WithTitle(t *testing.T) {
	view := Bordered(Text("Content")).
		Border(&RoundedBorder).
		Title("Title")
	screen := SprintScreen(view, WithWidth(20))
	termtest.AssertScreen(t, screen)
}

func TestGolden_Border_TitleTooLong(t *testing.T) {
	// Title exceeds available width - should truncate
	view := Bordered(Text("X")).
		Border(&RoundedBorder).
		Title("This title is way too long")
	screen := SprintScreen(view, WithWidth(15))
	termtest.AssertScreen(t, screen)
}

func TestGolden_Border_Nested(t *testing.T) {
	// Border inside border
	view := Bordered(
		Bordered(Text("Inner")).Border(&RoundedBorder),
	).Border(&ThickBorder).Title("Outer")
	screen := SprintScreen(view, WithWidth(25))
	termtest.AssertScreen(t, screen)
}

func TestGolden_Padding_Uniform(t *testing.T) {
	view := Padding(2, Text("Padded"))
	screen := SprintScreen(view, WithWidth(20), WithHeight(5))
	termtest.AssertScreen(t, screen)
}

func TestGolden_Padding_HV(t *testing.T) {
	// Different horizontal and vertical padding
	view := PaddingHV(4, 1, Text("HV"))
	screen := SprintScreen(view, WithWidth(15), WithHeight(4))
	termtest.AssertScreen(t, screen)
}

func TestGolden_Padding_LTRB(t *testing.T) {
	// Different padding on each side
	view := PaddingLTRB(1, 0, 3, 2, Text("LTRB"))
	screen := SprintScreen(view, WithWidth(15), WithHeight(4))
	termtest.AssertScreen(t, screen)
}

// =============================================================================
// DIVIDER TESTS
// =============================================================================

func TestGolden_Divider_Simple(t *testing.T) {
	view := Divider()
	screen := SprintScreen(view, WithWidth(30))
	termtest.AssertScreen(t, screen)
}

func TestGolden_Divider_CustomChar(t *testing.T) {
	view := Divider().Char('=')
	screen := SprintScreen(view, WithWidth(30))
	termtest.AssertScreen(t, screen)
}

func TestGolden_Divider_WithTitle(t *testing.T) {
	view := Divider().Title("Section")
	screen := SprintScreen(view, WithWidth(30))
	termtest.AssertScreen(t, screen)
}

func TestGolden_Divider_TitleTooWide(t *testing.T) {
	// Title wider than available space
	view := Divider().Title("Very Long Section Title")
	screen := SprintScreen(view, WithWidth(15))
	termtest.AssertScreen(t, screen)
}

// =============================================================================
// HEADER/STATUS BAR TESTS
// =============================================================================

func TestGolden_HeaderBar(t *testing.T) {
	view := HeaderBar("My Application")
	screen := SprintScreen(view, WithWidth(30))
	termtest.AssertScreen(t, screen)
}

func TestGolden_StatusBar(t *testing.T) {
	view := StatusBar("Ready | Ln 1, Col 1")
	screen := SprintScreen(view, WithWidth(30))
	termtest.AssertScreen(t, screen)
}

// =============================================================================
// PROGRESS BAR TESTS
// =============================================================================

func TestGolden_Progress_Zero(t *testing.T) {
	view := Progress(0, 100).Width(20)
	screen := SprintScreen(view, WithWidth(30))
	termtest.AssertScreen(t, screen)
}

func TestGolden_Progress_Half(t *testing.T) {
	view := Progress(50, 100).Width(20)
	screen := SprintScreen(view, WithWidth(30))
	termtest.AssertScreen(t, screen)
}

func TestGolden_Progress_Full(t *testing.T) {
	view := Progress(100, 100).Width(20)
	screen := SprintScreen(view, WithWidth(30))
	termtest.AssertScreen(t, screen)
}

func TestGolden_Progress_WithLabel(t *testing.T) {
	view := Progress(75, 100).Width(15).Label("Loading:")
	screen := SprintScreen(view, WithWidth(35))
	termtest.AssertScreen(t, screen)
}

func TestGolden_Progress_Fraction(t *testing.T) {
	view := Progress(3, 10).Width(15).ShowFraction()
	screen := SprintScreen(view, WithWidth(30))
	termtest.AssertScreen(t, screen)
}

func TestGolden_Progress_Pattern(t *testing.T) {
	view := Progress(60, 100).Width(20).EmptyPattern("\xb7-")
	screen := SprintScreen(view, WithWidth(30))
	termtest.AssertScreen(t, screen)
}

func TestGolden_Progress_NoPercent(t *testing.T) {
	view := Progress(40, 100).Width(20).HidePercent()
	screen := SprintScreen(view, WithWidth(25))
	termtest.AssertScreen(t, screen)
}

// =============================================================================
// TABLE TESTS
// =============================================================================

func TestGolden_Table_Basic(t *testing.T) {
	sel := 0
	view := Table([]TableColumn{
		{Title: "Name"},
		{Title: "Age"},
	}, &sel).Rows([][]string{
		{"Alice", "30"},
		{"Bob", "25"},
		{"Carol", "35"},
	})
	screen := SprintScreen(view, WithWidth(25))
	termtest.AssertScreen(t, screen)
}

func TestGolden_Table_SelectedRow(t *testing.T) {
	sel := 1
	view := Table([]TableColumn{
		{Title: "Name"},
		{Title: "Value"},
	}, &sel).Rows([][]string{
		{"First", "100"},
		{"Second", "200"},
		{"Third", "300"},
	})
	screen := SprintScreen(view, WithWidth(25))
	termtest.AssertScreen(t, screen)
}

func TestGolden_Table_NoHeader(t *testing.T) {
	sel := 0
	view := Table([]TableColumn{
		{Title: "A"},
		{Title: "B"},
	}, &sel).ShowHeader(false).Rows([][]string{
		{"X", "1"},
		{"Y", "2"},
	})
	screen := SprintScreen(view, WithWidth(20))
	termtest.AssertScreen(t, screen)
}

func TestGolden_Table_UppercaseHeaders(t *testing.T) {
	sel := 0
	view := Table([]TableColumn{
		{Title: "name"},
		{Title: "status"},
	}, &sel).UppercaseHeaders(true).Rows([][]string{
		{"test", "ok"},
	})
	screen := SprintScreen(view, WithWidth(25))
	termtest.AssertScreen(t, screen)
}

func TestGolden_Table_ColumnTruncation(t *testing.T) {
	sel := 0
	view := Table([]TableColumn{
		{Title: "Description", Width: 10},
		{Title: "Status"},
	}, &sel).Rows([][]string{
		{"This is a very long description", "Active"},
		{"Short", "Inactive"},
	})
	screen := SprintScreen(view, WithWidth(25))
	termtest.AssertScreen(t, screen)
}

func TestGolden_Table_WideContent(t *testing.T) {
	// Unicode content with different widths
	sel := 0
	view := Table([]TableColumn{
		{Title: "Lang"},
		{Title: "Hello"},
	}, &sel).Rows([][]string{
		{"EN", "Hello"},
		{"CN", "\xe4\xbd\xa0\xe5\xa5\xbd"}, // "你好"
		{"JP", "\xe3\x81\x93\xe3\x82\x93\xe3\x81\xab\xe3\x81\xa1\xe3\x81\xaf"}, // "こんにちは"
	})
	screen := SprintScreen(view, WithWidth(30))
	termtest.AssertScreen(t, screen)
}

// =============================================================================
// CODE VIEW TESTS
// =============================================================================

func TestGolden_Code_Go(t *testing.T) {
	code := `func main() {
	fmt.Println("Hello")
}`
	view := Code(code, "go").LineNumbers(true)
	screen := SprintScreen(view, WithWidth(30))
	termtest.AssertScreen(t, screen)
}

func TestGolden_Code_NoLineNumbers(t *testing.T) {
	code := `print("Hello")`
	view := Code(code, "python").LineNumbers(false)
	screen := SprintScreen(view, WithWidth(25))
	termtest.AssertScreen(t, screen)
}

func TestGolden_Code_PlainText(t *testing.T) {
	code := "Just some plain text\nWith multiple lines"
	view := Code(code, "text").LineNumbers(true)
	screen := SprintScreen(view, WithWidth(35))
	termtest.AssertScreen(t, screen)
}

// =============================================================================
// COMPLEX NESTED LAYOUTS
// =============================================================================

func TestGolden_Complex_FormLayout(t *testing.T) {
	// Simulates a form with label-value pairs
	view := Bordered(
		Stack(
			Group(Text("Name:"), Spacer(), Text("Alice")),
			Group(Text("Email:"), Spacer(), Text("alice@example.com")),
			Divider(),
			Group(Spacer(), Text("[Submit]")),
		).Gap(1),
	).Border(&RoundedBorder).Title("User Profile")
	screen := SprintScreen(view, WithWidth(35))
	termtest.AssertScreen(t, screen)
}

func TestGolden_Complex_Dashboard(t *testing.T) {
	// Multiple bordered sections
	view := Stack(
		HeaderBar("Dashboard"),
		Group(
			Bordered(Text("Panel A")).Border(&RoundedBorder).Title("Left"),
			Bordered(Text("Panel B")).Border(&RoundedBorder).Title("Right"),
		).Gap(1),
		StatusBar("Connected"),
	)
	screen := SprintScreen(view, WithWidth(35))
	termtest.AssertScreen(t, screen)
}

func TestGolden_Complex_Menu(t *testing.T) {
	// Vertical menu with indicators
	view := Bordered(
		Stack(
			Text("> Option 1").Bold(),
			Text("  Option 2"),
			Text("  Option 3"),
			Divider(),
			Text("  Quit").Dim(),
		),
	).Border(&RoundedBorder).Title("Menu")
	screen := SprintScreen(view, WithWidth(20))
	termtest.AssertScreen(t, screen)
}

func TestGolden_Complex_SplitView(t *testing.T) {
	// Left sidebar with main content
	view := Group(
		// Sidebar
		Bordered(
			Stack(
				Text("Files"),
				Text("Search"),
				Text("Settings"),
			),
		).Border(&RoundedBorder),
		// Main content
		Bordered(
			Stack(
				Text("Main Content"),
				Text("Goes here"),
			),
		).Border(&RoundedBorder).Title("Editor"),
	).Gap(1)
	screen := SprintScreen(view, WithWidth(40))
	termtest.AssertScreen(t, screen)
}

func TestGolden_Complex_StatusWithProgress(t *testing.T) {
	// Progress bar in a status context
	view := Stack(
		Text("Downloading files..."),
		Progress(67, 100).Width(25),
		Text("2 of 3 complete").Dim(),
	)
	screen := SprintScreen(view, WithWidth(35))
	termtest.AssertScreen(t, screen)
}

func TestGolden_Complex_DeepNesting(t *testing.T) {
	// Tests layout engine with deep nesting
	view := Bordered(
		Padding(1,
			Stack(
				Text("Level 1"),
				Bordered(
					Padding(1,
						Text("Level 2"),
					),
				).Border(&RoundedBorder),
			),
		),
	).Border(&ThickBorder).Title("Deep")
	screen := SprintScreen(view, WithWidth(30))
	termtest.AssertScreen(t, screen)
}

// =============================================================================
// FILL VIEW TESTS
// =============================================================================

func TestGolden_Fill_Char(t *testing.T) {
	view := Fill('#')
	screen := SprintScreen(view, WithWidth(10), WithHeight(3))
	termtest.AssertScreen(t, screen)
}

func TestGolden_Fill_Block(t *testing.T) {
	view := Fill('\u2588') // Full block
	screen := SprintScreen(view, WithWidth(8), WithHeight(2))
	termtest.AssertScreen(t, screen)
}

// =============================================================================
// SIZE CONSTRAINT TESTS
// =============================================================================

func TestGolden_Width_Fixed(t *testing.T) {
	view := Width(10, Text("Constrained text here"))
	screen := SprintScreen(view, WithWidth(20))
	termtest.AssertScreen(t, screen)
}

func TestGolden_Height_Fixed(t *testing.T) {
	view := Height(3, Stack(
		Text("Line 1"),
		Text("Line 2"),
		Text("Line 3"),
		Text("Line 4"),
		Text("Line 5"),
	))
	screen := SprintScreen(view, WithWidth(15), WithHeight(5))
	termtest.AssertScreen(t, screen)
}

func TestGolden_MinWidth(t *testing.T) {
	view := MinWidth(15, Text("Short"))
	screen := SprintScreen(view, WithWidth(20))
	termtest.AssertScreen(t, screen)
}

// =============================================================================
// EDGE CASES
// =============================================================================

func TestGolden_Edge_VeryNarrow(t *testing.T) {
	// Extremely narrow container
	view := Stack(
		Text("ABC"),
		Text("DEFGH"),
	)
	screen := SprintScreen(view, WithWidth(3))
	termtest.AssertScreen(t, screen)
}

func TestGolden_Edge_SingleColumn(t *testing.T) {
	// Width of 1
	view := Stack(
		Text("X"),
		Text("Y"),
	)
	screen := SprintScreen(view, WithWidth(1))
	termtest.AssertScreen(t, screen)
}

func TestGolden_Edge_ZeroGap(t *testing.T) {
	// Explicit zero gap (should have no spacing)
	view := Stack(
		Text("A"),
		Text("B"),
	).Gap(0)
	screen := SprintScreen(view, WithWidth(10))
	termtest.AssertScreen(t, screen)
}

func TestGolden_Edge_LargeGap(t *testing.T) {
	// Large gap between items
	view := Stack(
		Text("Top"),
		Text("Bottom"),
	).Gap(3)
	screen := SprintScreen(view, WithWidth(10))
	termtest.AssertScreen(t, screen)
}

// =============================================================================
// FLEX BEHAVIOR TESTS - Space distribution and flexible layouts
// =============================================================================

func TestGolden_Flex_EqualDistribution(t *testing.T) {
	// Two spacers should split space equally
	view := Stack(
		Text("A"),
		Spacer(),
		Text("B"),
		Spacer(),
		Text("C"),
	)
	screen := SprintScreen(view, WithWidth(10), WithHeight(11))
	termtest.AssertScreen(t, screen)
}

func TestGolden_Flex_UnequalFactors(t *testing.T) {
	// Flex(1) gets 1/3, Flex(2) gets 2/3 of remaining space
	view := Stack(
		Text("Top"),
		Spacer().Flex(1),
		Text("Mid"),
		Spacer().Flex(2),
		Text("Bot"),
	)
	screen := SprintScreen(view, WithWidth(5), WithHeight(12))
	termtest.AssertScreen(t, screen)
}

func TestGolden_Flex_GroupHorizontal(t *testing.T) {
	// Spacer in Group distributes horizontal space
	view := Group(
		Text("[A]"),
		Spacer(),
		Text("[B]"),
		Spacer(),
		Text("[C]"),
	)
	screen := SprintScreen(view, WithWidth(25))
	termtest.AssertScreen(t, screen)
}

func TestGolden_Flex_GroupUnequalFactors(t *testing.T) {
	// Unequal flex factors in horizontal layout
	view := Group(
		Text("L"),
		Spacer().Flex(1),
		Text("M"),
		Spacer().Flex(3),
		Text("R"),
	)
	screen := SprintScreen(view, WithWidth(20))
	termtest.AssertScreen(t, screen)
}

func TestGolden_Flex_MixedFixedAndFlex(t *testing.T) {
	// Fixed-size children with flex spacer
	view := Stack(
		Text("Fixed 1"),
		Text("Fixed 2"),
		Spacer(),
		Text("Fixed 3"),
	)
	screen := SprintScreen(view, WithWidth(12), WithHeight(8))
	termtest.AssertScreen(t, screen)
}

func TestGolden_Flex_FillExpands(t *testing.T) {
	// Fill view is flexible and expands
	view := Stack(
		Text("Header"),
		Fill('.'),
		Text("Footer"),
	)
	screen := SprintScreen(view, WithWidth(15), WithHeight(7))
	termtest.AssertScreen(t, screen)
}

func TestGolden_Flex_NestedFlexContainers(t *testing.T) {
	// Flex container inside another flex container
	view := Stack(
		Text("Outer Top"),
		Group(
			Text("L"),
			Spacer(),
			Text("R"),
		),
		Spacer(),
		Text("Outer Bot"),
	)
	screen := SprintScreen(view, WithWidth(20), WithHeight(8))
	termtest.AssertScreen(t, screen)
}

func TestGolden_Flex_SpacerMinWidth(t *testing.T) {
	// Spacer with minimum width constraint
	view := Group(
		Text("A"),
		Spacer().MinWidth(10),
		Text("B"),
	)
	screen := SprintScreen(view, WithWidth(20))
	termtest.AssertScreen(t, screen)
}

func TestGolden_Flex_SpacerMinHeight(t *testing.T) {
	// Spacer with minimum height constraint
	view := Stack(
		Text("Top"),
		Spacer().MinHeight(3),
		Text("Bottom"),
	)
	screen := SprintScreen(view, WithWidth(10), WithHeight(8))
	termtest.AssertScreen(t, screen)
}

func TestGolden_Flex_NoFlexChildren(t *testing.T) {
	// All fixed children - should size to content
	view := Stack(
		Text("Line 1"),
		Text("Line 2"),
		Text("Line 3"),
	)
	screen := SprintScreen(view, WithWidth(15), WithHeight(10))
	termtest.AssertScreen(t, screen)
}

func TestGolden_Flex_SingleFlexChild(t *testing.T) {
	// Single flex child takes all remaining space
	view := Stack(
		Text("Header"),
		Fill('#'),
	)
	screen := SprintScreen(view, WithWidth(12), WithHeight(5))
	termtest.AssertScreen(t, screen)
}

// =============================================================================
// SIZE CONSTRAINT TESTS - Min/Max/Fixed sizing
// =============================================================================

func TestGolden_Size_FixedWidth(t *testing.T) {
	// Fixed width larger than content
	view := Width(20, Text("Short"))
	screen := SprintScreen(view, WithWidth(25))
	termtest.AssertScreen(t, screen)
}

func TestGolden_Size_FixedWidthTruncates(t *testing.T) {
	// Fixed width smaller than content - should truncate
	view := Width(8, Text("This is too long"))
	screen := SprintScreen(view, WithWidth(20))
	termtest.AssertScreen(t, screen)
}

func TestGolden_Size_FixedHeight(t *testing.T) {
	// Fixed height larger than content
	view := Height(5, Text("Single line"))
	screen := SprintScreen(view, WithWidth(15), WithHeight(7))
	termtest.AssertScreen(t, screen)
}

func TestGolden_Size_FixedHeightClips(t *testing.T) {
	// Fixed height smaller than content - should clip
	view := Height(2, Stack(
		Text("Line 1"),
		Text("Line 2"),
		Text("Line 3"),
		Text("Line 4"),
	))
	screen := SprintScreen(view, WithWidth(10), WithHeight(5))
	termtest.AssertScreen(t, screen)
}

func TestGolden_Size_FixedBoth(t *testing.T) {
	// Fixed width and height
	view := Size(15, 4, Text("Fixed box"))
	screen := SprintScreen(view, WithWidth(20), WithHeight(6))
	termtest.AssertScreen(t, screen)
}

func TestGolden_Size_MinWidthExpands(t *testing.T) {
	// MinWidth expands small content
	view := MinWidth(20, Text("Tiny"))
	screen := SprintScreen(view, WithWidth(25))
	termtest.AssertScreen(t, screen)
}

func TestGolden_Size_MinWidthNoEffect(t *testing.T) {
	// MinWidth has no effect when content is larger
	view := MinWidth(5, Text("Already wide enough"))
	screen := SprintScreen(view, WithWidth(25))
	termtest.AssertScreen(t, screen)
}

func TestGolden_Size_MaxWidthConstrains(t *testing.T) {
	// MaxWidth constrains large content
	view := MaxWidth(10, Text("This text is way too long for the max"))
	screen := SprintScreen(view, WithWidth(30))
	termtest.AssertScreen(t, screen)
}

func TestGolden_Size_MaxWidthNoEffect(t *testing.T) {
	// MaxWidth has no effect when content is smaller
	view := MaxWidth(30, Text("Small"))
	screen := SprintScreen(view, WithWidth(35))
	termtest.AssertScreen(t, screen)
}

func TestGolden_Size_MinHeightExpands(t *testing.T) {
	// MinHeight expands short content
	view := MinHeight(5, Text("One line"))
	screen := SprintScreen(view, WithWidth(15), WithHeight(7))
	termtest.AssertScreen(t, screen)
}

func TestGolden_Size_MaxHeightClips(t *testing.T) {
	// MaxHeight clips tall content
	view := MaxHeight(3, Stack(
		Text("1"), Text("2"), Text("3"), Text("4"), Text("5"),
	))
	screen := SprintScreen(view, WithWidth(10), WithHeight(5))
	termtest.AssertScreen(t, screen)
}

func TestGolden_Size_MinAndMaxWidth(t *testing.T) {
	// Both min and max width constraints
	view := MaxWidth(15, MinWidth(10, Text("Mid")))
	screen := SprintScreen(view, WithWidth(20))
	termtest.AssertScreen(t, screen)
}

func TestGolden_Size_MinSize(t *testing.T) {
	// MinSize sets both dimensions
	view := MinSize(12, 4, Text("X"))
	screen := SprintScreen(view, WithWidth(15), WithHeight(6))
	termtest.AssertScreen(t, screen)
}

func TestGolden_Size_NestedConstraints(t *testing.T) {
	// Nested size constraints
	view := MaxWidth(20, MinWidth(15, Width(18, Text("Nested"))))
	screen := SprintScreen(view, WithWidth(25))
	termtest.AssertScreen(t, screen)
}

// =============================================================================
// PARENT/CHILD SIZE INTERACTION TESTS
// =============================================================================

func TestGolden_ParentChild_AutoWidthFromChildren(t *testing.T) {
	// Stack width determined by widest child
	view := Stack(
		Text("Short"),
		Text("This is longer"),
		Text("Medium"),
	)
	screen := SprintScreen(view, WithWidth(25))
	termtest.AssertScreen(t, screen)
}

func TestGolden_ParentChild_AutoHeightFromChildren(t *testing.T) {
	// Group height determined by tallest child
	view := Group(
		Text("A"),
		Stack(Text("B1"), Text("B2"), Text("B3")),
		Text("C"),
	)
	screen := SprintScreen(view, WithWidth(15))
	termtest.AssertScreen(t, screen)
}

func TestGolden_ParentChild_FixedParentFlexChildren(t *testing.T) {
	// Fixed parent with flex children inside
	view := Size(20, 8, Stack(
		Text("Header"),
		Spacer(),
		Text("Footer"),
	))
	screen := SprintScreen(view, WithWidth(25), WithHeight(10))
	termtest.AssertScreen(t, screen)
}

func TestGolden_ParentChild_ConstrainedParent(t *testing.T) {
	// Parent with max constraint, children adapt
	view := MaxWidth(15, Stack(
		Text("This is a very long line"),
		Text("Short"),
	))
	screen := SprintScreen(view, WithWidth(30))
	termtest.AssertScreen(t, screen)
}

func TestGolden_ParentChild_ChildExceedsParent(t *testing.T) {
	// Child wants more space than parent provides
	view := Width(10, Text("Way too long for this"))
	screen := SprintScreen(view, WithWidth(15))
	termtest.AssertScreen(t, screen)
}

func TestGolden_ParentChild_NestedAutoSize(t *testing.T) {
	// Nested containers all auto-sizing from content
	view := Stack(
		Group(Text("A1"), Text("A2")),
		Group(Text("B1"), Text("B2"), Text("B3")),
	)
	screen := SprintScreen(view, WithWidth(20))
	termtest.AssertScreen(t, screen)
}

func TestGolden_ParentChild_BorderedAutoSize(t *testing.T) {
	// Border adds 2 to each dimension
	view := Bordered(Text("Content")).Border(&RoundedBorder)
	screen := SprintScreen(view, WithWidth(20))
	termtest.AssertScreen(t, screen)
}

func TestGolden_ParentChild_PaddingAutoSize(t *testing.T) {
	// Padding adds to dimensions
	view := Padding(2, Text("Padded"))
	screen := SprintScreen(view, WithWidth(20))
	termtest.AssertScreen(t, screen)
}

func TestGolden_ParentChild_ComplexNesting(t *testing.T) {
	// Complex nested layout with various size behaviors
	view := Stack(
		Group(
			MinWidth(8, Text("A")),
			Spacer(),
			MaxWidth(8, Text("Very long B")),
		),
		Divider(),
		Group(
			Width(5, Text("Fixed")),
			Text("Auto"),
		).Gap(2),
	)
	screen := SprintScreen(view, WithWidth(30))
	termtest.AssertScreen(t, screen)
}

// =============================================================================
// FLEX WITH CONSTRAINTS TESTS
// =============================================================================

func TestGolden_FlexConstraint_FlexWithMinSize(t *testing.T) {
	// Flex child with minimum size
	view := Stack(
		Text("Top"),
		MinHeight(3, Fill('.')),
		Text("Bottom"),
	)
	screen := SprintScreen(view, WithWidth(15), WithHeight(8))
	termtest.AssertScreen(t, screen)
}

func TestGolden_FlexConstraint_FlexWithMaxSize(t *testing.T) {
	// Flex child with maximum size
	view := Stack(
		Text("Top"),
		MaxHeight(2, Fill('#')),
		Spacer(),
		Text("Bottom"),
	)
	screen := SprintScreen(view, WithWidth(15), WithHeight(10))
	termtest.AssertScreen(t, screen)
}

func TestGolden_FlexConstraint_ConstrainedFlexGroup(t *testing.T) {
	// Group with constrained flex children
	view := Group(
		MinWidth(5, Text("A")),
		Spacer(),
		MaxWidth(8, Text("Very long text here")),
	)
	screen := SprintScreen(view, WithWidth(25))
	termtest.AssertScreen(t, screen)
}

func TestGolden_FlexConstraint_FixedInFlex(t *testing.T) {
	// Fixed size child in flex container
	view := Stack(
		Spacer(),
		Size(10, 2, Text("Fixed")),
		Spacer(),
	)
	screen := SprintScreen(view, WithWidth(15), WithHeight(8))
	termtest.AssertScreen(t, screen)
}

// =============================================================================
// AUTO-WIDTH SCENARIOS
// =============================================================================

func TestGolden_AutoWidth_StackTakesWidest(t *testing.T) {
	// Stack expands to fit widest child
	view := Bordered(
		Stack(
			Text("A"),
			Text("BBB"),
			Text("CCCCC"),
			Text("BB"),
		),
	).Border(&RoundedBorder)
	screen := SprintScreen(view, WithWidth(20))
	termtest.AssertScreen(t, screen)
}

func TestGolden_AutoWidth_GroupSumsChildren(t *testing.T) {
	// Group width is sum of children + gaps
	view := Bordered(
		Group(
			Text("AA"),
			Text("BB"),
			Text("CC"),
		).Gap(1),
	).Border(&RoundedBorder)
	screen := SprintScreen(view, WithWidth(20))
	termtest.AssertScreen(t, screen)
}

func TestGolden_AutoWidth_EmptyContainer(t *testing.T) {
	// Empty container has zero auto-width
	view := Stack(
		Text("Before"),
		Stack(), // Empty nested stack
		Text("After"),
	)
	screen := SprintScreen(view, WithWidth(15))
	termtest.AssertScreen(t, screen)
}

func TestGolden_AutoWidth_DividerFillsWidth(t *testing.T) {
	// Divider fills available width
	view := Stack(
		Text("Header"),
		Divider(),
		Text("Content"),
	)
	screen := SprintScreen(view, WithWidth(20))
	termtest.AssertScreen(t, screen)
}

// =============================================================================
// TABLE COLUMN WIDTH TESTS
// =============================================================================

func TestGolden_TableWidth_AutoColumns(t *testing.T) {
	// Columns auto-size to content
	sel := 0
	view := Table([]TableColumn{
		{Title: "ID"},
		{Title: "Name"},
		{Title: "Description"},
	}, &sel).Rows([][]string{
		{"1", "Alice", "Developer"},
		{"2", "Bob", "Manager"},
		{"100", "Carol", "Lead engineer"},
	})
	screen := SprintScreen(view, WithWidth(45))
	termtest.AssertScreen(t, screen)
}

func TestGolden_TableWidth_FixedColumns(t *testing.T) {
	// Columns with fixed widths
	sel := 0
	view := Table([]TableColumn{
		{Title: "A", Width: 5},
		{Title: "B", Width: 10},
		{Title: "C", Width: 5},
	}, &sel).Rows([][]string{
		{"X", "YYYYYYYY", "Z"},
	})
	screen := SprintScreen(view, WithWidth(30))
	termtest.AssertScreen(t, screen)
}

func TestGolden_TableWidth_MixedColumns(t *testing.T) {
	// Mix of fixed and auto columns
	sel := 0
	view := Table([]TableColumn{
		{Title: "Fixed", Width: 8},
		{Title: "Auto"},
		{Title: "Also Auto"},
	}, &sel).Rows([][]string{
		{"XXXXXXXX", "Short", "Medium length"},
	})
	screen := SprintScreen(view, WithWidth(40))
	termtest.AssertScreen(t, screen)
}

func TestGolden_TableWidth_MinWidthRespected(t *testing.T) {
	// Column min width prevents shrinking
	sel := 0
	view := Table([]TableColumn{
		{Title: "Name", MinWidth: 10},
		{Title: "Val"},
	}, &sel).Rows([][]string{
		{"A", "1"},
	})
	screen := SprintScreen(view, WithWidth(25))
	termtest.AssertScreen(t, screen)
}

func TestGolden_TableWidth_FillWidth(t *testing.T) {
	// Table expands to fill container
	sel := 0
	view := Table([]TableColumn{
		{Title: "Col A"},
		{Title: "Col B"},
	}, &sel).FillWidth().Rows([][]string{
		{"X", "Y"},
	})
	screen := SprintScreen(view, WithWidth(30))
	termtest.AssertScreen(t, screen)
}
