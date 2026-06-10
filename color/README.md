# color

ANSI color types, hex parsing, HSL conversion, and gradient generation for
terminal rendering.

## Summary

The color package provides a complete color system for terminal applications.
It supports standard ANSI colors (0-15), the full 256-color palette, and
24-bit RGB/TrueColor with hex parsing and HSL conversion in both directions.
It includes gradient generation with multiple interpolation methods and
helpers for applying colors to text with automatic reset sequences.

Color output is handled automatically: the `Apply` helpers emit plain text
when `NO_COLOR` is set, when `CLICOLOR=0` is set, or when stdout is not a
terminal, and `FORCE_COLOR`/`CLICOLOR_FORCE` override in the other direction.
The obvious code is also the correct code — no conditional logic required.

## Usage Examples

### Basic Color Application

```go
package main

import (
    "fmt"
    "github.com/deepnoodle-ai/wonton/color"
)

func main() {
    // Apply standard ANSI colors. Output is automatically plain when piped
    // or when NO_COLOR is set.
    fmt.Println(color.Red.Apply("Error message"))
    fmt.Println(color.Green.Apply("Success!"))
    fmt.Println(color.Yellow.Apply("Warning"))
    fmt.Println(color.Cyan.Apply("Info"))

    // Apply bright colors
    fmt.Println(color.BrightBlue.Apply("Highlighted"))

    // Apply as background
    fmt.Println(color.Red.ApplyBg("Red background"))

    // Apply with bold or dim attributes
    fmt.Println(color.Red.ApplyBold("FAILED"))
    fmt.Println(color.White.ApplyDim("Dimmed text"))
}
```

### Standard ANSI Colors

```go
package main

import (
    "fmt"
    "github.com/deepnoodle-ai/wonton/color"
)

func main() {
    // Standard colors (0-7)
    fmt.Println(color.Black.Apply("Black"))
    fmt.Println(color.Red.Apply("Red"))
    fmt.Println(color.Green.Apply("Green"))
    fmt.Println(color.Yellow.Apply("Yellow"))
    fmt.Println(color.Blue.Apply("Blue"))
    fmt.Println(color.Magenta.Apply("Magenta"))
    fmt.Println(color.Cyan.Apply("Cyan"))
    fmt.Println(color.White.Apply("White"))

    // Bright colors (8-15)
    fmt.Println(color.BrightBlack.Apply("Bright Black"))
    fmt.Println(color.BrightRed.Apply("Bright Red"))
    fmt.Println(color.BrightGreen.Apply("Bright Green"))
    fmt.Println(color.BrightYellow.Apply("Bright Yellow"))
    fmt.Println(color.BrightBlue.Apply("Bright Blue"))
    fmt.Println(color.BrightMagenta.Apply("Bright Magenta"))
    fmt.Println(color.BrightCyan.Apply("Bright Cyan"))
    fmt.Println(color.BrightWhite.Apply("Bright White"))
}
```

### RGB Colors and Hex Parsing

```go
package main

import (
    "fmt"
    "github.com/deepnoodle-ai/wonton/color"
)

func main() {
    // Create RGB colors from channel values or hex strings
    orange := color.NewRGB(255, 165, 0)
    purple := color.MustHex("#800080")          // panics on invalid input
    pink, err := color.Hex("#ffc0cb")           // returns an error instead
    if err != nil {
        panic(err)
    }

    fmt.Println(orange.Apply("Orange text"))
    fmt.Println(purple.ApplyBg("Purple background"))
    fmt.Println(pink.Sprintf("Pink %s", "formatted"))

    // Convert back to hex
    fmt.Println(orange.Hex()) // "#ffa500"

    // Interpolate between colors
    mid := orange.Lerp(purple, 0.5)
    fmt.Println(mid.Apply("Halfway between orange and purple"))
}
```

### 256-Color Palette

```go
package main

import (
    "fmt"
    "github.com/deepnoodle-ai/wonton/color"
)

func main() {
    // Use extended 256-color palette: 16-231 is a 6x6x6 color cube
    for i := uint8(16); i < 232; i++ {
        c := color.Palette(i)
        fmt.Print(c.Apply("█"))
    }
    fmt.Println()

    // Grayscale ramp (232-255)
    for i := 232; i < 256; i++ {
        c := color.Palette(uint8(i))
        fmt.Print(c.Apply("█"))
    }
    fmt.Println()
}
```

### Gradients

```go
package main

import (
    "fmt"
    "github.com/deepnoodle-ai/wonton/color"
)

func main() {
    // Color a string with a gradient across its characters
    fmt.Println(color.ApplyGradient("Hello, gradient!",
        color.MustHex("#ff5f5f"),
        color.MustHex("#5f87ff"),
    ))

    // Or generate gradient color steps to apply yourself
    start := color.NewRGB(255, 0, 0) // Red
    end := color.NewRGB(0, 0, 255)   // Blue
    for _, c := range color.Gradient(start, end, 10) {
        fmt.Print(c.Apply("█"))
    }
    fmt.Println()

    // Rainbow gradient with classic color stops
    for _, c := range color.RainbowGradient(50) {
        fmt.Print(c.Apply("█"))
    }
    fmt.Println()

    // Smooth rainbow using HSL hue rotation
    for _, c := range color.SmoothRainbow(50) {
        fmt.Print(c.Apply("█"))
    }
    fmt.Println()
}
```

### Multi-Stop Gradients

```go
package main

import (
    "fmt"
    "github.com/deepnoodle-ai/wonton/color"
)

func main() {
    // Gradient through multiple color stops
    stops := []color.RGB{
        color.NewRGB(255, 0, 0),   // Red
        color.NewRGB(255, 255, 0), // Yellow
        color.NewRGB(0, 255, 0),   // Green
        color.NewRGB(0, 255, 255), // Cyan
        color.NewRGB(0, 0, 255),   // Blue
    }

    for _, c := range color.MultiGradient(stops, 100) {
        fmt.Print(c.Apply("█"))
    }
    fmt.Println()
}
```

### HSL Color Conversion

```go
package main

import (
    "fmt"
    "github.com/deepnoodle-ai/wonton/color"
)

func main() {
    // HSL to RGB: hue 0-360, saturation 0-1, lightness 0-1
    red := color.HSLToRGB(0, 1.0, 0.5)
    green := color.HSLToRGB(120, 1.0, 0.5)
    pastel := color.HSLToRGB(180, 0.5, 0.75)

    fmt.Println(red.Apply("Red"))
    fmt.Println(green.Apply("Green"))
    fmt.Println(pastel.Apply("Pastel cyan"))

    // RGB to HSL: derive variations of an existing color
    h, s, l := color.MustHex("#ff8000").HSL()
    darker := color.HSLToRGB(h, s, l*0.6)
    desaturated := color.HSLToRGB(h, s*0.4, l)
    fmt.Println(darker.Apply("Darker orange"))
    fmt.Println(desaturated.Apply("Muted orange"))
}
```

### Formatting Helpers

```go
package main

import (
    "fmt"
    "github.com/deepnoodle-ai/wonton/color"
)

func main() {
    // Sprintf with color
    fmt.Println(color.Red.Sprintf("Error: %s", "file not found"))

    // Sprint with color
    fmt.Println(color.Green.Sprint("Success: ", 42, " items processed"))

    // Bold and dim text without color
    fmt.Println(color.ApplyBold("Important!"))
    fmt.Println(color.ApplyDim("Less important"))

    // Bold and dim combined with color
    fmt.Println(color.Red.ApplyBold("Critical failure"))
    fmt.Println(color.Cyan.ApplyDim("(hint: use --help)"))
}
```

### Controlling Color Output

```go
package main

import (
    "fmt"
    "os"
    "github.com/deepnoodle-ai/wonton/color"
)

func main() {
    // color.Enabled is initialized automatically at startup:
    //   1. FORCE_COLOR / CLICOLOR_FORCE force colors on
    //   2. NO_COLOR / CLICOLOR=0 force colors off
    //   3. Otherwise: on iff stdout is a terminal
    fmt.Printf("Colors enabled: %v\n", color.Enabled)

    // Override it for special cases, e.g. coloring stderr while stdout
    // is piped:
    color.Enabled = color.ShouldColorize(os.Stderr)
    fmt.Fprintln(os.Stderr, color.Red.Apply("error: something failed"))

    // Or take manual control, e.g. from a --no-color flag:
    color.Enabled = false
    fmt.Println(color.Red.Apply("This prints as plain text"))
}
```

### Low-Level Escape Sequences

```go
package main

import (
    "fmt"
    "github.com/deepnoodle-ai/wonton/color"
)

func main() {
    // The sequence functions always emit escape codes, regardless of
    // color.Enabled. Use them when managing terminal state yourself.
    orange := color.NewRGB(255, 165, 0)
    fmt.Println(orange.ForegroundSeq() + "Orange text" + color.Reset)
    fmt.Println(color.Blue.BackgroundSeq() + "Blue background" + color.Reset)

    // SGR parameters for composing your own sequences
    fmt.Printf("red fg code: %s\n", color.Red.ForegroundCode())  // "31"
    fmt.Printf("red bg code: %s\n", color.Red.BackgroundCode())  // "41"
}
```

## API Reference

### Color Type

| Constant/Function | Description | Value/Return |
|-------------------|-------------|--------------|
| `Default` | The terminal's default color | `-1` |
| `Black`, `Red`, `Green`, `Yellow`, `Blue`, `Magenta`, `Cyan`, `White` | Standard colors | `0-7` |
| `BrightBlack` ... `BrightWhite` | Bright colors | `8-15` |
| `Palette(n)` | 256-color palette entry | `Color` |

### Color Methods

| Method | Description | Returns |
|--------|-------------|---------|
| `Apply(text)` | Apply as foreground (respects `Enabled`) | `string` |
| `ApplyBg(text)` | Apply as background (respects `Enabled`) | `string` |
| `ApplyDim(text)` | Apply with dim attribute (respects `Enabled`) | `string` |
| `ApplyBold(text)` | Apply with bold attribute (respects `Enabled`) | `string` |
| `Sprintf(format, args...)` | Format string with color (respects `Enabled`) | `string` |
| `Sprint(args...)` | Sprint with color (respects `Enabled`) | `string` |
| `ForegroundSeq()` | ANSI foreground escape sequence | `string` |
| `BackgroundSeq()` | ANSI background escape sequence | `string` |
| `ForegroundSeqDim()` | Dim foreground escape sequence | `string` |
| `ForegroundSeqBold()` | Bold foreground escape sequence | `string` |
| `ForegroundCode()` | SGR parameter for foreground | `string` |
| `BackgroundCode()` | SGR parameter for background | `string` |

### RGB Type

| Function | Description | Returns |
|----------|-------------|---------|
| `NewRGB(r, g, b)` | Create RGB color from channels | `RGB` |
| `Hex(s)` | Parse `#rrggbb` / `#rgb` hex string | `(RGB, error)` |
| `MustHex(s)` | Like `Hex` but panics on invalid input | `RGB` |

### RGB Methods

| Method | Description | Returns |
|--------|-------------|---------|
| `Apply(text)` | Apply as foreground (respects `Enabled`) | `string` |
| `ApplyBg(text)` | Apply as background (respects `Enabled`) | `string` |
| `Sprintf(format, args...)` | Format string with color (respects `Enabled`) | `string` |
| `Sprint(args...)` | Sprint with color (respects `Enabled`) | `string` |
| `Hex()` | Format as `#rrggbb` | `string` |
| `HSL()` | Convert to HSL | `(h, s, l float64)` |
| `Lerp(other, t)` | Interpolate toward another color | `RGB` |
| `ForegroundSeq()` | ANSI foreground escape sequence | `string` |
| `BackgroundSeq()` | ANSI background escape sequence | `string` |

### Gradient Functions

| Function | Description | Returns |
|----------|-------------|---------|
| `Gradient(start, end, steps)` | Gradient between two colors | `[]RGB` |
| `MultiGradient(stops, steps)` | Gradient through multiple stops | `[]RGB` |
| `RainbowGradient(steps)` | Classic rainbow gradient | `[]RGB` |
| `SmoothRainbow(steps)` | Smooth rainbow using HSL | `[]RGB` |
| `ApplyGradient(text, stops...)` | Color text with a gradient (respects `Enabled`) | `string` |

All gradient functions return an empty slice for `steps <= 0` and the first
color for `steps == 1`.

### HSL Functions

| Function | Description | Returns |
|----------|-------------|---------|
| `HSLToRGB(h, s, l)` | Convert HSL to RGB | `RGB` |

### Utility Functions and Variables

| Name | Description |
|------|-------------|
| `Enabled` | Whether the Apply helpers emit color; auto-initialized at startup |
| `ShouldColorize(f)` | Whether colors should be used for a specific stream |

### Constants

| Constant | Value | Description |
|----------|-------|-------------|
| `Reset` | `"\033[0m"` | Reset all attributes |
| `Bold` | `"\033[1m"` | Bold attribute |
| `Dim` | `"\033[2m"` | Dim/faint attribute |

## Related Packages

- **[tui](../tui/)** - Terminal UI library that uses color for rendering
- **[terminal](../terminal/)** - Low-level terminal control with ANSI support
- **[assert](../assert/)** - Test assertions that use colored diffs
- **[cli](../cli/)** - CLI framework with colored output helpers
- **[tty](../tty/)** - Terminal detection used by `ShouldColorize`
