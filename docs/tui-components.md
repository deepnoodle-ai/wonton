# TUI Component Guide

This guide documents all available views (components) in the `tui` package. It is designed as a reference for LLMs and developers building terminal user interfaces.

## Quick Reference

| Category    | Components                                                  |
| ----------- | ----------------------------------------------------------- |
| Layout      | `Stack`, `Group`, `ZStack`, `Spacer`, `Empty`               |
| Containers  | `Bordered`, `Padding`, `Width`, `Height`, `Scroll`, `Panel` |
| Text        | `Text`, `Divider`, `HeaderBar`, `StatusBar`, `FocusText`    |
| Input       | `InputField`, `PasswordInput`, `TextArea`                   |
| Buttons     | `Button`, `Clickable`, `StyledButton`, `Toggle`             |
| Lists       | `SelectList`, `FilterableList`, `CheckboxList`, `RadioList` |
| Data        | `Table`, `Tree`, `KeyValue`                                 |
| Content     | `Code`, `Markdown`, `DiffView`                              |
| Progress    | `Progress`, `Loading`, `Meter`                              |
| Drawing     | `Canvas`, `CanvasContext`, `Fill`                           |
| Grids       | `CellGrid`, `ColorGrid`, `CharGrid`                         |
| Links       | `Link`, `InlineLinks`, `LinkRow`, `LinkList`                |
| Files       | `FilePicker`                                                |
| Collections | `ForEach`, `HForEach`                                       |
| Conditional | `If`, `IfElse`, `Switch`                                    |

---

## Layout Components

### Stack

Arranges children vertically (top to bottom).

```go
tui.Stack(
    tui.Text("Header"),
    tui.Text("Body"),
    tui.Text("Footer"),
).Gap(1).Align(tui.AlignCenter)
```

**Constructor**: `Stack(children ...View) *stack`

**Methods**:
| Method                 | Description                                                    |
| ---------------------- | -------------------------------------------------------------- |
| `.Gap(n int)`          | Vertical spacing between children (rows)                       |
| `.Align(a Alignment)`  | Horizontal alignment: `AlignLeft`, `AlignCenter`, `AlignRight` |
| `.Flex(factor int)`    | Flex factor for parent layout distribution                     |
| `.Padding(n int)`      | Add equal padding on all sides                                 |
| `.PaddingHV(h, v int)` | Add horizontal and vertical padding                            |
| `.Bordered()`          | Wrap with a border                                             |

---

### Group

Arranges children horizontally (left to right).

```go
tui.Group(
    tui.Text("Left"),
    tui.Spacer(),
    tui.Text("Right"),
).Gap(2)
```

**Constructor**: `Group(children ...View) *group`

**Methods**:
| Method                | Description                                   |
| --------------------- | --------------------------------------------- |
| `.Gap(n int)`         | Horizontal spacing between children (columns) |
| `.Align(a Alignment)` | Vertical alignment of children                |
| `.Flex(factor int)`   | Flex factor for parent layout distribution    |
| `.Padding(n int)`     | Add equal padding on all sides                |
| `.Bordered()`         | Wrap with a border                            |

---

### ZStack

Layers children on top of each other (z-axis). Later children render on top.

```go
tui.ZStack(
    tui.Fill(' ').Bg(tui.ColorBlue),  // Background
    tui.Text("Overlay Text"),          // Foreground
)
```

**Constructor**: `ZStack(children ...View) *zStack`

**Methods**:
| Method                | Description                                    |
| --------------------- | ---------------------------------------------- |
| `.Align(a Alignment)` | Alignment of smaller children within the stack |

---

### Spacer

Flexible space that expands to fill available room.

```go
tui.Stack(
    tui.Text("Top"),
    tui.Spacer(),           // Pushes footer to bottom
    tui.Text("Bottom"),
)
```

**Constructor**: `Spacer() *spacerView`

**Methods**:
| Method              | Description                      |
| ------------------- | -------------------------------- |
| `.Flex(factor int)` | Relative flex weight (default 1) |
| `.MinWidth(w int)`  | Minimum width                    |
| `.MinHeight(h int)` | Minimum height                   |

---

### Empty

Renders nothing. Useful for conditional rendering.

```go
tui.If(showWidget, widget) // Returns Empty() when false
```

**Constructor**: `Empty() View`

---

## Container Components

### Bordered

Wraps any view with a border and optional title.

```go
tui.Bordered(
    tui.Text("Content inside border"),
).Title("Box Title").Border(&tui.RoundedBorder).BorderFg(tui.ColorCyan)
```

**Constructor**: `Bordered(inner View) *borderedView`

**Methods**:
| Method                        | Description                                                           |
| ----------------------------- | --------------------------------------------------------------------- |
| `.Border(style *BorderStyle)` | Border characters: `&SingleBorder`, `&RoundedBorder`, `&DoubleBorder` |
| `.Title(title string)`        | Title text shown in top border                                        |
| `.TitleStyle(s Style)`        | Style for title text                                                  |
| `.BorderFg(c Color)`          | Border foreground color                                               |
| `.FocusID(id string)`         | Watch this focus ID for styling changes                               |
| `.FocusBorderFg(c Color)`     | Border color when watched element is focused                          |
| `.FocusTitleStyle(s Style)`   | Title style when watched element is focused                           |

**Border Styles**:
- `SingleBorder` - Single line: `│─┌┐└┘`
- `RoundedBorder` - Rounded corners: `│─╭╮╰╯`
- `DoubleBorder` - Double line: `║═╔╗╚╝`

---

### Padding

Adds space around a view.

```go
tui.Padding(2, content)                    // 2 on all sides
tui.PaddingHV(4, 1, content)               // 4 horizontal, 1 vertical
tui.PaddingLTRB(1, 2, 1, 0, content)       // left, top, right, bottom
```

**Constructors**:
| Function                                  | Description                     |
| ----------------------------------------- | ------------------------------- |
| `Padding(n int, inner View)`              | Equal padding on all sides      |
| `PaddingHV(h, v int, inner View)`         | Horizontal and vertical padding |
| `PaddingLTRB(l, t, r, b int, inner View)` | CSS-style individual sides      |

---

### Size Constraints

Control view dimensions.

```go
tui.Width(40, content)           // Fixed width
tui.MaxWidth(80, content)        // Maximum width
tui.MinHeight(10, content)       // Minimum height
tui.Size(40, 20, content)        // Fixed width and height
```

**Constructors**:
| Function                        | Description               |
| ------------------------------- | ------------------------- |
| `Width(w int, inner View)`      | Fixed width               |
| `Height(h int, inner View)`     | Fixed height              |
| `Size(w, h int, inner View)`    | Fixed width and height    |
| `MaxWidth(w int, inner View)`   | Maximum width constraint  |
| `MaxHeight(h int, inner View)`  | Maximum height constraint |
| `MinWidth(w int, inner View)`   | Minimum width constraint  |
| `MinHeight(h int, inner View)`  | Minimum height constraint |
| `MinSize(w, h int, inner View)` | Minimum width and height  |

---

### Scroll

Scrollable container for content larger than viewport.

```go
var scrollY int
tui.Scroll(
    tui.Stack(longContent...),
    &scrollY,
).Height(20)
```

**Constructor**: `Scroll(inner View, scrollY *int) *scrollView`

**Methods**:
| Method                         | Description                                         |
| ------------------------------ | --------------------------------------------------- |
| `.Anchor(anchor ScrollAnchor)` | `ScrollAnchorTop` (default) or `ScrollAnchorBottom` |
| `.Bottom()`                    | Shorthand for `ScrollAnchorBottom` (chat-style)     |

---

## Text Components

### Text

Display styled text with Printf-style formatting.

```go
tui.Text("Hello, %s!", name).Bold().Fg(tui.ColorGreen)
tui.Text("Error: %v", err).Error()
tui.Text("Loading...").Animate(Rainbow(3))
```

**Constructor**: `Text(format string, args ...any) *textView`

**Style Methods**:
| Method                  | Description                |
| ----------------------- | -------------------------- |
| `.Fg(c Color)`          | Foreground color           |
| `.FgRGB(r, g, b uint8)` | RGB foreground             |
| `.Bg(c Color)`          | Background color           |
| `.BgRGB(r, g, b uint8)` | RGB background             |
| `.Bold()`               | Bold text                  |
| `.Italic()`             | Italic text                |
| `.Underline()`          | Underlined text            |
| `.Strikethrough()`      | Strikethrough text         |
| `.Dim()`                | Dimmed/faint text          |
| `.Reverse()`            | Swap foreground/background |
| `.Blink()`              | Blinking text              |
| `.Style(s Style)`       | Apply complete style       |

**Semantic Styles**:
| Method       | Result          |
| ------------ | --------------- |
| `.Success()` | Green + Bold    |
| `.Error()`   | Red + Bold      |
| `.Warning()` | Yellow + Bold   |
| `.Info()`    | Cyan            |
| `.Muted()`   | Dim gray        |
| `.Hint()`    | Dim italic gray |

**Layout Methods**:
| Method                | Description                            |
| --------------------- | -------------------------------------- |
| `.Wrap()`             | Enable word wrapping                   |
| `.Truncate()`         | Disable wrapping (default)             |
| `.Align(a Alignment)` | Text alignment                         |
| `.Center()`           | Center align                           |
| `.Right()`            | Right align                            |
| `.FillBg()`           | Fill entire area with background color |
| `.Flex(factor int)`   | Flex factor                            |
| `.Width(w int)`       | Fixed width                            |
| `.Height(h int)`      | Fixed height                           |
| `.MaxWidth(w int)`    | Maximum width                          |

**Animation**:
| Method                              | Description          |
| ----------------------------------- | -------------------- |
| `.Animate(animation TextAnimation)` | Apply text animation |

---

### Divider

Horizontal line separator with optional title.

```go
tui.Divider()
tui.Divider().Title("Section").Fg(tui.ColorCyan)
```

**Constructor**: `Divider() *dividerView`

**Methods**:
| Method                 | Description                     |
| ---------------------- | ------------------------------- |
| `.Char(c rune)`        | Divider character (default `─`) |
| `.Title(title string)` | Centered text in divider        |
| `.Fg(c Color)`         | Foreground color                |
| `.Bg(c Color)`         | Background color                |
| `.Style(s Style)`      | Complete style                  |
| `.Bold()`              | Bold divider                    |
| `.Dim()`               | Dimmed divider                  |

---

### HeaderBar / StatusBar

Full-width bars for headers and footers.

```go
tui.HeaderBar("Application Title").Bold().Bg(tui.ColorBlue)
tui.StatusBar("Ready").Fg(tui.ColorGreen)
```

**Constructors**:
- `HeaderBar(text string) *headerBarView`
- `StatusBar(text string) *headerBarView`

**Methods**:
| Method            | Description      |
| ----------------- | ---------------- |
| `.Fg(c Color)`    | Foreground color |
| `.Bg(c Color)`    | Background color |
| `.Style(s Style)` | Complete style   |
| `.Bold()`         | Bold text        |

---

## Input Components

### InputField

Text input with label, placeholder, and focus handling.

```go
var name string
tui.InputField(&name).
    ID("name-input").
    Label("Name:").
    Placeholder("Enter your name").
    OnSubmit(func(s string) { /* handle submit */ })
```

**Constructor**: `InputField(binding *string) *inputFieldView`

**Configuration**:
| Method                       | Description                                 |
| ---------------------------- | ------------------------------------------- |
| `.ID(id string)`             | Unique focus identifier                     |
| `.Label(text string)`        | Label text before input                     |
| `.LabelStyle(s Style)`       | Label styling                               |
| `.FocusLabelStyle(s Style)`  | Label style when focused                    |
| `.Placeholder(text string)`  | Placeholder text                            |
| `.PlaceholderStyle(s Style)` | Placeholder styling                         |
| `.Mask(r rune)`              | Mask character for passwords                |
| `.Width(w int)`              | Input field width                           |
| `.MaxHeight(lines int)`      | Max height for multiline                    |
| `.Multiline(enabled bool)`   | Enable multiline (Shift+Enter for newlines) |

**Events**:
| Method                       | Description            |
| ---------------------------- | ---------------------- |
| `.OnChange(fn func(string))` | Called on every change |
| `.OnSubmit(fn func(string))` | Called on Enter key    |

**Border**:
| Method                        | Description               |
| ----------------------------- | ------------------------- |
| `.Bordered()`                 | Add border around input   |
| `.Border(style *BorderStyle)` | Specific border style     |
| `.BorderFg(c Color)`          | Border color              |
| `.FocusBorderFg(c Color)`     | Border color when focused |
| `.HorizontalBorderOnly()`     | Only top/bottom borders   |

**Cursor**:
| Method                                 | Description                                                  |
| -------------------------------------- | ------------------------------------------------------------ |
| `.CursorBlink(enabled bool)`           | Enable cursor blinking                                       |
| `.CursorShape(shape InputCursorStyle)` | `InputCursorBlock`, `InputCursorUnderline`, `InputCursorBar` |
| `.CursorColor(color Color)`            | Custom cursor color                                          |

**Prompt**:
| Method                  | Description                |
| ----------------------- | -------------------------- |
| `.Prompt(text string)`  | Left-side prompt character |
| `.PromptStyle(s Style)` | Prompt styling             |

---

### PasswordInput

Secure password entry widget.

```go
input := tui.NewPasswordInput(terminal)
input.WithPrompt("Password: ", promptStyle)
input.WithMaskChar('*')
password, err := input.Read()
```

**Constructor**: `NewPasswordInput(terminal *Terminal) *PasswordInput`

**Methods**:
| Method                                    | Description                 |
| ----------------------------------------- | --------------------------- |
| `.WithPrompt(prompt string, style Style)` | Set prompt                  |
| `.WithPlaceholder(placeholder string)`    | Placeholder text            |
| `.WithMaxLength(length int)`              | Maximum input length        |
| `.WithMaskChar(char rune)`                | Mask character              |
| `.ShowCharacters(show bool)`              | Show masked characters      |
| `.EnableSecureMode(enable bool)`          | iTerm2/VS Code secure input |
| `.DisableClipboard(disable bool)`         | Block clipboard access      |
| `.ConfirmPaste(confirm bool)`             | Require paste confirmation  |
| `.Read() (*SecureString, error)`          | Read password (blocking)    |

---

### TextArea

Multi-line scrollable text display with optional editing.

```go
var content string
var scrollY int
tui.TextArea(&content).
    ID("editor").
    Title("Document").
    LineNumbers(true).
    HighlightCurrentLine(true).
    Height(20)
```

**Constructor**: `TextArea(binding *string) *textAreaView`

**Configuration**:
| Method                     | Description                    |
| -------------------------- | ------------------------------ |
| `.ID(id string)`           | Unique focus identifier        |
| `.Content(content string)` | Static content (if no binding) |
| `.ScrollY(scrollY *int)`   | Scroll position binding        |
| `.Width(w int)`            | Fixed width                    |
| `.Height(h int)`           | Fixed height                   |
| `.Size(w, h int)`          | Fixed dimensions               |

**Display**:
| Method                           | Description              |
| -------------------------------- | ------------------------ |
| `.TextStyle(s Style)`            | Content text styling     |
| `.EmptyPlaceholder(text string)` | Message when empty       |
| `.EmptyStyle(s Style)`           | Empty message styling    |
| `.Title(title string)`           | Title in border          |
| `.TitleStyle(s Style)`           | Title styling            |
| `.FocusTitleStyle(s Style)`      | Title style when focused |

**Line Numbers**:
| Method                      | Description         |
| --------------------------- | ------------------- |
| `.LineNumbers(show bool)`   | Show line numbers   |
| `.LineNumberStyle(s Style)` | Line number styling |
| `.LineNumberFg(c Color)`    | Line number color   |

**Current Line**:
| Method                                  | Description           |
| --------------------------------------- | --------------------- |
| `.HighlightCurrentLine(highlight bool)` | Highlight cursor line |
| `.CurrentLineStyle(s Style)`            | Current line styling  |
| `.CursorLine(line *int)`                | Cursor line binding   |

**Border**:
| Method                        | Description               |
| ----------------------------- | ------------------------- |
| `.Bordered()`                 | Add full border           |
| `.Border(style *BorderStyle)` | Specific border style     |
| `.BorderFg(c Color)`          | Border color              |
| `.FocusBorderFg(c Color)`     | Border color when focused |
| `.LeftBorderOnly()`           | Only left border          |

---

## Button Components

### Button

Keyboard-focusable button activated with Enter or Space.

```go
tui.Button("Submit", func() {
    // handle click
}).ID("submit-btn")
```

**Constructor**: `Button(label string, callback func()) *buttonView`

**Methods**:
| Method                 | Description             |
| ---------------------- | ----------------------- |
| `.ID(id string)`       | Unique focus identifier |
| `.Fg(c Color)`         | Foreground color        |
| `.Bg(c Color)`         | Background color        |
| `.Bold()`              | Bold text               |
| `.Reverse()`           | Reverse colors          |
| `.Style(s Style)`      | Complete style          |
| `.FocusStyle(s Style)` | Style when focused      |
| `.Width(w int)`        | Fixed width             |

---

### Clickable

Mouse-only interactive text element.

```go
tui.Clickable("[ + ]", func() {
    count++
}).Fg(tui.ColorGreen)
```

**Constructor**: `Clickable(label string, callback func()) *clickableView`

**Methods**:
| Method            | Description      |
| ----------------- | ---------------- |
| `.Fg(c Color)`    | Foreground color |
| `.Bg(c Color)`    | Background color |
| `.Bold()`         | Bold text        |
| `.Reverse()`      | Reverse colors   |
| `.Style(s Style)` | Complete style   |
| `.Width(w int)`   | Fixed width      |

---

### StyledButton

Filled button with configurable dimensions.

```go
tui.StyledButton("Save", onSave).
    Size(20, 3).
    Bg(tui.ColorBlue).
    Bold()
```

**Constructor**: `StyledButton(label string, callback func()) *styledButtonView`

**Methods**:
| Method                           | Description                |
| -------------------------------- | -------------------------- |
| `.Width(w int)`                  | Button width               |
| `.Height(h int)`                 | Button height              |
| `.Size(w, h int)`                | Width and height           |
| `.Fg(c Color)`                   | Foreground color           |
| `.Bg(c Color)`                   | Background color           |
| `.Style(s Style)`                | Complete style             |
| `.HoverStyle(s Style)`           | Style on hover             |
| `.Bold()`                        | Bold text                  |
| `.Border(style borderStyleType)` | Border style               |
| `.Centered(centered bool)`       | Center text (default true) |

---

### Toggle

On/off toggle switch.

```go
var enabled bool
tui.Toggle(&enabled).
    OnLabel("ON").
    OffLabel("OFF").
    OnChange(func(v bool) { /* handle change */ })
```

**Constructor**: `Toggle(value *bool) *toggleView`

**Methods**:
| Method                     | Description         |
| -------------------------- | ------------------- |
| `.OnLabel(label string)`   | Label for ON state  |
| `.OffLabel(label string)`  | Label for OFF state |
| `.OnStyle(s Style)`        | Style for ON state  |
| `.OffStyle(s Style)`       | Style for OFF state |
| `.OnChange(fn func(bool))` | Change callback     |
| `.ShowLabels(show bool)`   | Show/hide labels    |

---

## List Components

### SelectList

Simple selectable list with keyboard navigation.

```go
var selected int
items := []string{"Option A", "Option B", "Option C"}
tui.SelectListStrings(items, &selected).
    OnSelect(func(item tui.ListItem, idx int) { /* handle selection */ }).
    Height(10)
```

**Constructors**:
- `SelectList(items []ListItem, selected *int) *selectListView`
- `SelectListStrings(labels []string, selected *int) *selectListView`

**Methods**:
| Method                                        | Description                    |
| --------------------------------------------- | ------------------------------ |
| `.OnSelect(fn func(item ListItem, index int)` | Selection callback             |
| `.Fg(c Color)`                                | Normal item color              |
| `.Bg(c Color)`                                | Normal item background         |
| `.Style(s Style)`                             | Normal item style              |
| `.SelectedFg(c Color)`                        | Selected item color            |
| `.SelectedBg(c Color)`                        | Selected item background       |
| `.SelectedStyle(s Style)`                     | Selected item style            |
| `.CursorChar(c string)`                       | Cursor indicator (default `▸`) |
| `.ShowCursor(show bool)`                      | Show/hide cursor               |
| `.Width(w int)`                               | Fixed width                    |
| `.Height(h int)`                              | Fixed height                   |
| `.Size(w, h int)`                             | Fixed dimensions               |

---

### FilterableList

List with search filtering and custom rendering.

```go
var selected int
var filter string
tui.FilterableList(items, &selected).
    Filter(&filter).
    FilterPlaceholder("Type to search...").
    Height(15)
```

**Constructors**:
- `FilterableList(items []ListItem, selected *int) *listView`
- `FilterableListStrings(labels []string, selected *int) *listView`

**Methods**:
| Method                                                   | Description                        |
| -------------------------------------------------------- | ---------------------------------- |
| `.OnSelect(fn func(item ListItem, index int))`           | Selection callback                 |
| `.Filter(filterText *string)`                            | Enable filtering with text binding |
| `.FilterPlaceholder(text string)`                        | Filter input placeholder           |
| `.FilterFunc(fn func(item ListItem, query string) bool)` | Custom filter logic                |
| `.Renderer(fn ListItemRenderer)`                         | Custom item renderer               |
| `.ItemHeight(h int)`                                     | Height per item                    |
| `.ScrollY(scrollY *int)`                                 | Scroll position binding            |
| `.Fg(c Color)`                                           | Normal item color                  |
| `.SelectedFg(c Color)`                                   | Selected item color                |
| `.SelectedBg(c Color)`                                   | Selected item background           |
| `.Style(s Style)`                                        | Normal item style                  |
| `.SelectedStyle(s Style)`                                | Selected item style                |
| `.Width(w int)`                                          | Fixed width                        |
| `.Height(h int)`                                         | Fixed height                       |
| `.Size(w, h int)`                                        | Fixed dimensions                   |

---

### CheckboxList

List with checkable items.

```go
items := []string{"Feature A", "Feature B", "Feature C"}
checked := []bool{true, false, false}
var cursor int
tui.CheckboxListStrings(items, checked, &cursor).
    OnToggle(func(idx int, isChecked bool) { /* handle toggle */ })
```

**Constructors**:
- `CheckboxList(items []ListItem, checked []bool, cursor *int) *checkboxListView`
- `CheckboxListStrings(labels []string, checked []bool, cursor *int) *checkboxListView`

**Methods**:
| Method                                        | Description             |
| --------------------------------------------- | ----------------------- |
| `.OnToggle(fn func(index int, checked bool))` | Toggle callback         |
| `.CheckedChar(ch string)`                     | Checked box character   |
| `.UncheckedChar(ch string)`                   | Unchecked box character |
| `.Fg(c Color)`                                | Normal item color       |
| `.Bg(c Color)`                                | Normal item background  |
| `.CursorFg(c Color)`                          | Cursor line color       |
| `.CursorBg(c Color)`                          | Cursor line background  |
| `.CheckedFg(c Color)`                         | Checked item color      |
| `.CheckedBg(c Color)`                         | Checked item background |
| `.HighlightFg(c Color)`                       | Hover color             |
| `.HighlightBg(c Color)`                       | Hover background        |
| `.Style(s Style)`                             | Normal style            |
| `.CursorStyle(s Style)`                       | Cursor line style       |
| `.CheckedStyle(s Style)`                      | Checked item style      |
| `.HighlightStyle(s Style)`                    | Hover style             |
| `.Width(w int)`                               | Fixed width             |
| `.Height(h int)`                              | Fixed height            |
| `.Size(w, h int)`                             | Fixed dimensions        |

---

### RadioList

Single-selection radio button list.

```go
var selected int
options := []string{"Small", "Medium", "Large"}
tui.RadioListStrings(options, &selected).
    OnSelect(func(idx int) { /* handle selection */ })
```

**Constructors**:
- `RadioList(items []ListItem, selected *int) *radioListView`
- `RadioListStrings(labels []string, selected *int) *radioListView`

**Methods**:
| Method                          | Description                |
| ------------------------------- | -------------------------- |
| `.OnSelect(fn func(index int))` | Selection callback         |
| `.SelectedChar(ch string)`      | Selected radio character   |
| `.UnselectedChar(ch string)`    | Unselected radio character |
| `.Fg(c Color)`                  | Normal color               |
| `.CursorFg(c Color)`            | Cursor line color          |
| `.Style(s Style)`               | Normal style               |
| `.CursorStyle(s Style)`         | Cursor line style          |
| `.Width(w int)`                 | Fixed width                |
| `.Height(h int)`                | Fixed height               |
| `.Size(w, h int)`               | Fixed dimensions           |

---

## Data Components

### Table

Data table with columns, headers, and row selection.

```go
var selected int
columns := []tui.TableColumn{
    {Header: "Name", Width: 20},
    {Header: "Age", Width: 5},
    {Header: "City", Width: 15},
}
rows := [][]string{
    {"Alice", "28", "New York"},
    {"Bob", "35", "Boston"},
}
tui.Table(columns, &selected).
    Rows(rows).
    Bordered().
    Striped().
    OnSelect(func(row int) { /* handle selection */ })
```

**Constructor**: `Table(columns []TableColumn, selected *int) *tableView`

**TableColumn Type**:
```go
type TableColumn struct {
    Header   string
    Width    int  // 0 = auto
    MinWidth int
}
```

**Data**:
| Method                        | Description            |
| ----------------------------- | ---------------------- |
| `.Rows(rows [][]string)`      | Set table data         |
| `.OnSelect(fn func(row int))` | Row selection callback |

**Display**:
| Method                              | Description                     |
| ----------------------------------- | ------------------------------- |
| `.ShowHeader(show bool)`            | Show/hide header row            |
| `.UppercaseHeaders(uppercase bool)` | Auto-uppercase headers          |
| `.MaxColumnWidth(maxWidth int)`     | Maximum column width            |
| `.ColumnGap(gap int)`               | Gap between columns (default 2) |
| `.FillWidth()`                      | Expand to fill container        |
| `.HeaderBottomBorder(show bool)`    | Border under header             |
| `.Bordered()`                       | Add border around table         |
| `.Striped()`                        | Alternating row colors          |

**Styling**:
| Method                               | Description                |
| ------------------------------------ | -------------------------- |
| `.Fg(c Color)`                       | Normal row color           |
| `.Bg(c Color)`                       | Normal row background      |
| `.Style(s Style)`                    | Normal row style           |
| `.HeaderStyle(s Style)`              | Header styling             |
| `.HeaderFg(c Color)`                 | Header color               |
| `.SelectedStyle(s Style)`            | Selected row style         |
| `.SelectedFg(c Color)`               | Selected row color         |
| `.SelectedBg(c Color)`               | Selected row background    |
| `.InvertSelectedColors(invert bool)` | Swap fg/bg on selected row |

**Dimensions**:
| Method            | Description      |
| ----------------- | ---------------- |
| `.Width(w int)`   | Fixed width      |
| `.Height(h int)`  | Fixed height     |
| `.Size(w, h int)` | Fixed dimensions |

---

### Tree

Hierarchical tree view with expand/collapse.

```go
root := tui.NewTreeNode("Root").
    AddChild(tui.NewTreeNode("Child 1")).
    AddChild(tui.NewTreeNode("Child 2").
        AddChild(tui.NewTreeNode("Grandchild")))

tui.Tree(root).
    OnSelect(func(node *tui.TreeNode) { /* handle selection */ }).
    Height(20)
```

**TreeNode Constructor**: `NewTreeNode(label string) *TreeNode`

**TreeNode Methods**:
| Method                                | Description           |
| ------------------------------------- | --------------------- |
| `.AddChild(child *TreeNode)`          | Add single child      |
| `.AddChildren(children ...*TreeNode)` | Add multiple children |
| `.SetData(data any)`                  | Attach user data      |
| `.SetExpanded(expanded bool)`         | Control expansion     |
| `.IsLeaf()`                           | Check if leaf node    |
| `.ExpandAll()`                        | Recursively expand    |
| `.CollapseAll()`                      | Recursively collapse  |

**Tree View Constructor**: `Tree(root *TreeNode) *treeView`

**Tree View Methods**:
| Method                                | Description             |
| ------------------------------------- | ----------------------- |
| `.Selected(node *TreeNode)`           | Set selected node       |
| `.OnSelect(fn func(*TreeNode))`       | Selection callback      |
| `.ScrollY(scrollY *int)`              | Scroll position binding |
| `.Width(w int)`                       | Fixed width             |
| `.Height(h int)`                      | Fixed height            |
| `.Size(w, h int)`                     | Fixed dimensions        |
| `.Fg(c Color)`                        | Foreground color        |
| `.Bg(c Color)`                        | Background color        |
| `.Style(s Style)`                     | Normal node style       |
| `.SelectedStyle(s Style)`             | Selected node style     |
| `.ExpandChar(c string)`               | Expand indicator        |
| `.CollapseChar(c string)`             | Collapse indicator      |
| `.LeafChar(c string)`                 | Leaf indicator          |
| `.BranchChars(chars TreeBranchChars)` | Line drawing characters |
| `.GetVisibleCount()`                  | Number of visible nodes |
| `.FindNode(label string)`             | Search by label         |

---

### KeyValue

Label-value pair display.

```go
tui.KeyValue("Status", "Active").
    LabelFg(tui.ColorCyan).
    ValueFg(tui.ColorGreen)
```

**Constructor**: `KeyValue(label, value string) *keyValueView`

**Methods**:
| Method                   | Description              |
| ------------------------ | ------------------------ |
| `.LabelFg(c Color)`      | Label color              |
| `.ValueFg(c Color)`      | Value color              |
| `.LabelStyle(s Style)`   | Label style              |
| `.ValueStyle(s Style)`   | Value style              |
| `.Separator(sep string)` | Separator (default `: `) |
| `.Width(w int)`          | Fixed width              |
| `.Dim()`                 | Dim the value            |

---

## Content Components

### Code

Syntax-highlighted code display.

```go
code := `func main() {
    fmt.Println("Hello, World!")
}`
tui.Code(code, "go").
    Theme("monokai").
    LineNumbers(true).
    Height(10)
```

**Constructor**: `Code(code string, language string) *codeView`

**Methods**:
| Method                    | Description                      |
| ------------------------- | -------------------------------- |
| `.Language(lang string)`  | Programming language             |
| `.Theme(theme string)`    | Color theme                      |
| `.LineNumbers(show bool)` | Show line numbers                |
| `.StartLine(n int)`       | Starting line number (default 1) |
| `.ScrollY(scrollY *int)`  | Scroll position binding          |
| `.Width(w int)`           | Fixed width                      |
| `.Height(h int)`          | Fixed height                     |
| `.Size(w, h int)`         | Fixed dimensions                 |
| `.TabWidth(w int)`        | Spaces per tab (default 4)       |

**Available Themes**: `monokai`, `dracula`, `github`, `vs`, `solarized-dark`, `solarized-light`, etc.

**Helper Functions**:
- `AvailableThemes() []string` - List available themes
- `AvailableLanguages() []string` - List supported languages

---

### Markdown

Rendered markdown content.

```go
var scrollY int
content := `# Heading
This is **bold** and *italic*.
- List item 1
- List item 2
`
tui.Markdown(content, &scrollY).
    MaxWidth(80).
    Height(20)
```

**Constructor**: `Markdown(content string, scrollY *int) *markdownView`

**Methods**:
| Method                        | Description          |
| ----------------------------- | -------------------- |
| `.Theme(theme MarkdownTheme)` | Markdown theme       |
| `.MaxWidth(w int)`            | Text wrap width      |
| `.Height(h int)`              | Fixed height         |
| `.GetLineCount()`             | Total rendered lines |

---

### DiffView

Unified diff display with syntax highlighting.

```go
var scrollY int
diffText := `--- a/file.go
+++ b/file.go
@@ -1,3 +1,4 @@
 package main
+import "fmt"
 func main() {
 }
`
view, _ := tui.DiffViewFromText(diffText, "go", &scrollY)
view.ShowLineNumbers(true).Height(20)
```

**Constructors**:
- `DiffView(diff *Diff, language string, scrollY *int) *diffView`
- `DiffViewFromText(diffText, language string, scrollY *int) (*diffView, error)`

**Methods**:
| Method                          | Description                      |
| ------------------------------- | -------------------------------- |
| `.Theme(theme DiffTheme)`       | Diff color theme                 |
| `.Language(lang string)`        | Language for syntax highlighting |
| `.ShowLineNumbers(show bool)`   | Show line numbers                |
| `.SyntaxHighlight(enable bool)` | Enable syntax highlighting       |
| `.Height(h int)`                | Fixed height                     |
| `.GetLineCount()`               | Total rendered lines             |

---

## Progress Components

### Progress

Progress bar with percentage display.

```go
tui.Progress(50, 100).
    Width(40).
    ShowPercent().
    Label("Downloading: ").
    Fg(tui.ColorGreen)
```

**Constructor**: `Progress(current, total int) *progressView`

**Configuration**:
| Method                          | Description                    |
| ------------------------------- | ------------------------------ |
| `.Width(w int)`                 | Bar width                      |
| `.FilledChar(c rune)`           | Filled character (default `█`) |
| `.EmptyChar(c rune)`            | Empty character (default `░`)  |
| `.EmptyPattern(pattern string)` | Repeating pattern for empty    |
| `.Label(label string)`          | Prefix label                   |

**Display**:
| Method                   | Description           |
| ------------------------ | --------------------- |
| `.ShowPercent()`         | Show percentage       |
| `.HidePercent()`         | Hide percentage       |
| `.ShowFraction()`        | Show current/total    |
| `.PercentStyle(s Style)` | Percentage text style |
| `.PercentFg(c Color)`    | Percentage text color |

**Styling**:
| Method                 | Description          |
| ---------------------- | -------------------- |
| `.Fg(c Color)`         | Filled portion color |
| `.Style(s Style)`      | Filled portion style |
| `.EmptyFg(c Color)`    | Empty portion color  |
| `.EmptyStyle(s Style)` | Empty portion style  |

**Animation**:
| Method                                    | Description               |
| ----------------------------------------- | ------------------------- |
| `.Shimmer(highlightColor RGB, speed int)` | Moving highlight effect   |
| `.Pulse(color RGB, speed int)`            | Pulsing brightness effect |

---

### Loading

Animated spinner with label.

```go
func (app *App) HandleEvent(ev tui.Event) []tui.Cmd {
    if tick, ok := ev.(tui.TickEvent); ok {
        app.frame = tick.Frame
    }
    return nil
}

func (app *App) View() tui.View {
    return tui.Loading(app.frame).Label("Loading...")
}
```

**Constructor**: `Loading(frame uint64) *loadingView`

**Methods**:
| Method                     | Description                 |
| -------------------------- | --------------------------- |
| `.CharSet(chars []string)` | Spinner frames              |
| `.Speed(frames int)`       | Frames per character change |
| `.Fg(c Color)`             | Spinner color               |
| `.Bg(c Color)`             | Spinner background          |
| `.Style(s Style)`          | Spinner style               |
| `.Label(label string)`     | Label text                  |

---

### Meter

Labeled gauge display.

```go
tui.Meter("CPU", 75, 100).
    Width(30).
    ShowValue(true)
```

**Constructor**: `Meter(label string, value, max int) *meterView`

**Methods**:
| Method                  | Description      |
| ----------------------- | ---------------- |
| `.Width(w int)`         | Bar width        |
| `.FilledChar(c rune)`   | Filled character |
| `.EmptyChar(c rune)`    | Empty character  |
| `.Fg(c Color)`          | Bar color        |
| `.LabelFg(c Color)`     | Label color      |
| `.Style(s Style)`       | Bar style        |
| `.ShowValue(show bool)` | Show percentage  |

---

## Drawing Components

### Canvas

Imperative drawing within declarative UI.

```go
tui.Canvas(func(frame tui.RenderFrame, bounds image.Rectangle) {
    frame.PrintStyled(0, 0, "Custom", style)
})
```

**Constructors**:
- `Canvas(draw func(frame RenderFrame, bounds image.Rectangle)) *canvasView`
- `CanvasContext(draw func(ctx *RenderContext)) *canvasView` - Access to frame counter

**Methods**:
| Method            | Description      |
| ----------------- | ---------------- |
| `.Size(w, h int)` | Preferred size   |
| `.Width(w int)`   | Preferred width  |
| `.Height(h int)`  | Preferred height |

**RenderContext Methods**:
| Method                                                | Description                 |
| ----------------------------------------------------- | --------------------------- |
| `.Size()`                                             | Get (width, height)         |
| `.Frame()`                                            | Get animation frame counter |
| `.SetCell(x, y int, ch rune, style Style)`            | Draw single cell            |
| `.Print(x, y int, text string, style Style)`          | Draw text                   |
| `.PrintTruncated(x, y int, text string, style Style)` | Draw truncated text         |

---

### Fill

Fill area with a character.

```go
tui.Fill('░').Fg(tui.ColorBrightBlack)
```

**Constructor**: `Fill(char rune) *fillView`

**Methods**:
| Method                  | Description      |
| ----------------------- | ---------------- |
| `.Fg(c Color)`          | Foreground color |
| `.FgRGB(r, g, b uint8)` | RGB foreground   |
| `.Bg(c Color)`          | Background color |
| `.BgRGB(r, g, b uint8)` | RGB background   |
| `.Style(s Style)`       | Complete style   |

---

## Collection Components

### ForEach

Render a slice of items as a vertical Stack.

```go
tui.ForEach(items, func(item Item, i int) tui.View {
    return tui.Text("%d. %s", i+1, item.Name)
}).Gap(1).Separator(tui.Divider())
```

**Constructor**: `ForEach[T any](items []T, mapper func(item T, index int) View) *forEachView[T]`

**Methods**:
| Method                 | Description        |
| ---------------------- | ------------------ |
| `.Gap(n int)`          | Vertical spacing   |
| `.Separator(sep View)` | View between items |

---

### HForEach

Render a slice of items as a horizontal Group.

```go
tui.HForEach(tabs, func(tab Tab, i int) tui.View {
    return tui.Text(tab.Title).Padding(1)
}).Gap(2)
```

**Constructor**: `HForEach[T any](items []T, mapper func(item T, index int) View) *hForEachView[T]`

**Methods**:
| Method                 | Description        |
| ---------------------- | ------------------ |
| `.Gap(n int)`          | Horizontal spacing |
| `.Separator(sep View)` | View between items |

---

## Conditional Components

### If

Render view only when condition is true.

```go
tui.If(showWarning, tui.Text("Warning!").Warning())
```

**Function**: `If(condition bool, view View) View`

Returns `Empty()` when condition is false.

---

### IfElse

Render one of two views based on condition.

```go
tui.IfElse(isLoggedIn,
    tui.Text("Welcome back!"),
    tui.Text("Please log in"),
)
```

**Function**: `IfElse(condition bool, thenView, elseView View) View`

---

### Switch

Pattern matching on a value.

```go
tui.Switch(status,
    tui.Case("loading", tui.Text("Loading...").Info()),
    tui.Case("error", tui.Text("Error!").Error()),
    tui.Case("ready", tui.Text("Ready").Success()),
    tui.Default[string](tui.Text("Unknown")),
)
```

**Functions**:
- `Case[T comparable](value T, view View) CaseView[T]`
- `Default[T comparable](view View) CaseView[T]`
- `Switch[T comparable](value T, cases ...CaseView[T]) View`

---

## Text Animations

Apply animations to text using `.Animate()`:

```go
tui.Text("Rainbow!").Animate(Rainbow(3))
tui.Text("Alert!").Animate(Pulse(tui.NewRGB(255, 0, 0), 10))
```

**Available Animations**:

| Animation                         | Description           | Parameters                         |
| --------------------------------- | --------------------- | ---------------------------------- |
| `Rainbow(speed)`                  | Rainbow color cycling | speed: cycle speed                 |
| `Pulse(color, speed)`             | Pulsing brightness    | color: base RGB, speed: pulse rate |
| `Wave(speed, colors...)`          | Wave color effect     | speed, multiple colors             |
| `Slide(speed, base, highlight)`   | Sliding highlight     | speed, base RGB, highlight RGB     |
| `Sparkle(speed, base, spark)`     | Sparkle effect        | speed, base RGB, spark RGB         |
| `Typewriter(speed, text, cursor)` | Typewriter reveal     | speed, text RGB, cursor RGB        |
| `Glitch(speed, base, glitch)`     | Glitch effect         | speed, base RGB, glitch RGB        |

**Animation Methods**:
| Method                          | Description                 |
| ------------------------------- | --------------------------- |
| `.Reverse()`                    | Reverse animation direction |
| `.WithLength(n int)`            | Rainbow length              |
| `.Brightness(min, max float64)` | Pulse brightness range      |
| `.WithAmplitude(a float64)`     | Wave amplitude              |
| `.WithWidth(w int)`             | Slide highlight width       |
| `.WithDensity(d int)`           | Sparkle density             |
| `.WithLoop(loop bool)`          | Typewriter looping          |
| `.WithHoldFrames(n int)`        | Typewriter hold time        |
| `.WithIntensity(i int)`         | Glitch intensity            |

---

## Common Types

### Style

```go
style := tui.NewStyle().
    WithForeground(tui.ColorGreen).
    WithBackground(tui.ColorBlack).
    WithBold().
    WithItalic()
```

### Color Constants

Basic: `ColorBlack`, `ColorRed`, `ColorGreen`, `ColorYellow`, `ColorBlue`, `ColorMagenta`, `ColorCyan`, `ColorWhite`

Bright: `ColorBrightBlack`, `ColorBrightRed`, `ColorBrightGreen`, etc.

RGB: `NewRGB(r, g, b uint8)`

### Alignment

- `AlignLeft` - Left/Top alignment
- `AlignCenter` - Center alignment
- `AlignRight` - Right/Bottom alignment

### ListItem

```go
type ListItem struct {
    Label string
    Value string
    Icon  string
}
```

---

## Link Components

### Link

Clickable hyperlink that opens in terminal (if supported).

```go
tui.Link("https://github.com", "GitHub").Fg(tui.ColorCyan)
tui.Link("https://example.com", "")  // Uses URL as display text
```

**Constructor**: `Link(url, text string) *hyperlinkView`

**Methods**:
| Method               | Description                             |
| -------------------- | --------------------------------------- |
| `.Fg(c Color)`       | Link color                              |
| `.Bg(c Color)`       | Link background                         |
| `.Bold()`            | Bold text                               |
| `.Underline(u bool)` | Enable/disable underline (default true) |
| `.Style(s Style)`    | Complete style                          |
| `.ShowURL()`         | Show URL in fallback format             |

---

### InlineLinks

Horizontal row of hyperlinks with separators.

```go
tui.InlineLinks(" | ",
    tui.NewHyperlink("https://go.dev", "Go"),
    tui.NewHyperlink("https://github.com", "GitHub"),
)
```

**Constructor**: `InlineLinks(separator string, links ...Hyperlink) *inlineLinkView`

**Methods**:
| Method            | Description         |
| ----------------- | ------------------- |
| `.Style(s Style)` | Style for all links |

---

### LinkRow

Label + hyperlink pair for tables of links.

```go
tui.LinkRow("Documentation", "https://docs.example.com", "docs.example.com").
    LabelFg(tui.ColorWhite).
    LinkFg(tui.ColorCyan)
```

**Constructor**: `LinkRow(label, url, linkText string) *linkRowView`

**Methods**:
| Method                 | Description                  |
| ---------------------- | ---------------------------- |
| `.LabelFg(c Color)`    | Label color                  |
| `.LinkFg(c Color)`     | Link color                   |
| `.LabelStyle(s Style)` | Label style                  |
| `.LinkStyle(s Style)`  | Link style                   |
| `.Gap(g int)`          | Space between label and link |

---

### LinkList

Vertical list of hyperlinks.

```go
tui.LinkList(
    tui.NewHyperlink("https://go.dev", "Go Website"),
    tui.NewHyperlink("https://pkg.go.dev", "Package Docs"),
).Gap(1)
```

**Constructor**: `LinkList(links ...Hyperlink) *linkListView`

**Methods**:
| Method            | Description                    |
| ----------------- | ------------------------------ |
| `.Style(s Style)` | Link styling                   |
| `.Gap(g int)`     | Vertical spacing between links |

---

## Grid Components

### CellGrid

Grid of clickable cells with customizable content.

```go
tui.CellGrid(5, 5).
    CellSize(6, 3).
    Gap(1).
    OnClick(func(col, row int) { /* handle click */ })
```

**Constructor**: `CellGrid(cols, rows int) *gridView`

**Methods**:
| Method                                               | Description                  |
| ---------------------------------------------------- | ---------------------------- |
| `.CellSize(width, height int)`                       | Size of each cell            |
| `.Gap(gap int)`                                      | Gap between cells            |
| `.OnClick(fn func(col, row int))`                    | Click callback               |
| `.SetCell(col, row int, style Style)`                | Set cell style               |
| `.SetCellChar(col, row int, char rune, style Style)` | Set cell character and style |
| `.SetAllCells(fn func(col, row int) Style)`          | Set all cells via callback   |

---

### ColorGrid

Grid where clicking cycles through colors (useful for pixel art, games).

```go
state := make([][]int, 5)
for i := range state {
    state[i] = make([]int, 5)
}
colors := []tui.Color{tui.ColorBlack, tui.ColorRed, tui.ColorGreen, tui.ColorBlue}

tui.ColorGrid(5, 5, state, colors).
    CellSize(4, 2).
    OnStateChange(func(col, row, colorIdx int) { /* handle change */ })
```

**Constructor**: `ColorGrid(cols, rows int, state [][]int, colors []Color) *colorGridView`

**Methods**:
| Method                                              | Description           |
| --------------------------------------------------- | --------------------- |
| `.CellSize(width, height int)`                      | Size of each cell     |
| `.Gap(gap int)`                                     | Gap between cells     |
| `.OnStateChange(fn func(col, row, colorIndex int))` | State change callback |

---

### CharGrid

Grid display for character-based content.

```go
data := [][]rune{
    {'#', '.', '#'},
    {'.', '#', '.'},
    {'#', '.', '#'},
}
tui.CharGrid(data)
```

**Constructor**: `CharGrid(data [][]rune) *charGridView`

---

## Panel Component

### Panel

Filled box/rectangle with optional border and content.

```go
tui.Panel(nil).Size(20, 5).Bg(tui.ColorBlue)
tui.Panel(tui.Text("Hello")).Border(tui.BorderSingle).Title("Box")
```

**Constructor**: `Panel(content View) *panelView`

**Methods**:
| Method                           | Description          |
| -------------------------------- | -------------------- |
| `.Width(w int)`                  | Panel width          |
| `.Height(h int)`                 | Panel height         |
| `.Size(w, h int)`                | Width and height     |
| `.FillChar(c rune)`              | Background character |
| `.Border(style borderStyleType)` | Border style         |
| `.BorderColor(c Color)`          | Border color         |
| `.Bg(c Color)`                   | Background color     |
| `.Title(title string)`           | Title in border      |

**Border Style Constants**:
- `BorderNone` - No border
- `BorderSingle` - Single line
- `BorderDouble` - Double line
- `BorderRounded` - Rounded corners
- `BorderHeavy` - Heavy/thick lines

---

## Focus Components

### FocusText

Text that changes style based on a watched focus ID.

```go
tui.FocusText("Name: ", "name-input").
    Style(dimStyle).
    FocusStyle(brightStyle)
```

**Constructor**: `FocusText(content string, focusID string) *focusTextView`

**Methods**:
| Method                 | Description                           |
| ---------------------- | ------------------------------------- |
| `.Style(s Style)`      | Normal (unfocused) style              |
| `.FocusStyle(s Style)` | Style when watched element is focused |
| `.Fg(c Color)`         | Normal foreground color               |
| `.Bg(c Color)`         | Normal background color               |
| `.Bold()`              | Bold in normal style                  |
| `.Dim()`               | Dim in normal style                   |

---

## File Components

### FilePicker

File browser with filter input and file list.

```go
var filter string
var selected int
tui.FilePicker(files, &filter, &selected).
    CurrentPath("/home/user").
    OnSelect(func(item tui.ListItem, index int) { /* handle selection */ }).
    Size(60, 20)
```

**Constructor**: `FilePicker(items []ListItem, filter *string, selected *int) *filePickerView`

**Methods**:
| Method                                        | Description               |
| --------------------------------------------- | ------------------------- |
| `.CurrentPath(path string)`                   | Display current directory |
| `.OnSelect(fn func(item ListItem, index int)` | Selection callback        |
| `.ShowHidden(show bool)`                      | Show hidden files         |
| `.Fg(c Color)`                                | List item color           |
| `.Bg(c Color)`                                | List item background      |
| `.Style(s Style)`                             | List item style           |
| `.InputStyle(s Style)`                        | Filter input style        |
| `.PathStyle(s Style)`                         | Path display style        |
| `.Width(w int)`                               | Fixed width               |
| `.Height(h int)`                              | Fixed height              |
| `.Size(w, h int)`                             | Fixed dimensions          |

---

## Best Practices

1. **Use semantic styling** for common patterns:
   ```go
   tui.Text("Success!").Success()  // Not: .Fg(tui.ColorGreen).Bold()
   ```

2. **Use ID() for focusable elements**:
   ```go
   tui.InputField(&name).ID("name-input")
   tui.Button("Submit", fn).ID("submit-btn")
   ```

3. **Use ForEach for dynamic lists**:
   ```go
   tui.ForEach(items, func(item Item, i int) tui.View {
       return renderItem(item)
   })
   ```

4. **Use conditional views** instead of nil checks:
   ```go
   tui.If(showError, tui.Text(errMsg).Error())
   ```

5. **Use Bordered() for visual grouping**:
   ```go
   tui.Stack(content...).Bordered().Title("Section")
   ```

6. **Use Gap() for consistent spacing**:
   ```go
   tui.Stack(children...).Gap(1)
   ```

7. **Use binding pointers** for two-way data flow:
   ```go
   var value string
   tui.InputField(&value)  // Updates value on change
   ```
