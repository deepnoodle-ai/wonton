# color

ANSI color types, RGB/HSL conversion, and gradient generation for terminal rendering.

## Summary

The color package provides a comprehensive color system for terminal applications. It supports standard ANSI colors (0-15), the full 256-color palette, and 24-bit RGB/TrueColor. It includes RGB to HSL conversion, gradient generation with multiple interpolation methods, and utilities for applying colors to text with automatic reset sequences. The package respects the NO_COLOR environment variable and provides terminal detection to automatically disable colors when appropriate.

## Usage Examples

### Basic Color Application

```go
package main

import (
    "fmt"
    "github.com/deepnoodle-ai/wonton/color"
)

func main() {
    // Apply standard ANSI colors
    fmt.Println(color.Red.Apply("Error message"))
    fmt.Println(color.Green.Apply("Success!"))
    fmt.Println(color.Yellow.Apply("Warning"))
    fmt.Println(color.Cyan.Apply("Info"))

    // Apply bright colors
    fmt.Println(color.BrightBlue.Apply("Highlighted"))

    // Apply as background
    fmt.Println(color.Red.ApplyBg("Red background"))

    // Apply with dim attribute
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

### RGB Colors

```go
package main

import (
    "fmt"
    "github.com/deepnoodle-ai/wonton/color"
)

func main() {
    // Create RGB colors
    orange := color.NewRGB(255, 165, 0)
    purple := color.NewRGB(128, 0, 128)
    pink := color.NewRGB(255, 192, 203)

    // Apply as foreground
    fmt.Println(orange.ForegroundSeq() + "Orange text" + color.Reset)

    // Apply as background
    fmt.Println(purple.BackgroundSeq() + "Purple background" + color.Reset)

    // Use convenience method
    fmt.Println(pink.Apply("Pink text", false)) // false = foreground
    fmt.Println(pink.Apply("Pink background", true)) // true = background
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
    // Use extended 256-color palette
    for i := uint8(16); i < 232; i++ {
        c := color.Palette(i)
        fmt.Print(c.Apply("█"))
    }
    fmt.Println()

    // Grayscale colors (232-255)
    for i := uint8(232); i < 256; i++ {
        c := color.Palette(i)
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
    // Simple gradient between two colors
    start := color.NewRGB(255, 0, 0)   // Red
    end := color.NewRGB(0, 0, 255)     // Blue
    gradient := color.Gradient(start, end, 10)

    for i, c := range gradient {
        fmt.Print(c.ForegroundSeq() + "█" + color.Reset)
    }
    fmt.Println()

    // Rainbow gradient
    rainbow := color.RainbowGradient(50)
    for _, c := range rainbow {
        fmt.Print(c.ForegroundSeq() + "█" + color.Reset)
    }
    fmt.Println()

    // Smooth rainbow using HSL
    smoothRainbow := color.SmoothRainbow(50)
    for _, c := range smoothRainbow {
        fmt.Print(c.ForegroundSeq() + "█" + color.Reset)
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
        color.NewRGB(255, 0, 0),    // Red
        color.NewRGB(255, 255, 0),  // Yellow
        color.NewRGB(0, 255, 0),    // Green
        color.NewRGB(0, 255, 255),  // Cyan
        color.NewRGB(0, 0, 255),    // Blue
    }

    gradient := color.MultiGradient(stops, 100)
    for _, c := range gradient {
        fmt.Print(c.ForegroundSeq() + "█" + color.Reset)
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
    // Convert HSL to RGB
    // Hue: 0-360, Saturation: 0-1, Lightness: 0-1

    // Pure red (hue=0)
    red := color.HSLToRGB(0, 1.0, 0.5)
    fmt.Println(red.ForegroundSeq() + "Red" + color.Reset)

    // Pure green (hue=120)
    green := color.HSLToRGB(120, 1.0, 0.5)
    fmt.Println(green.ForegroundSeq() + "Green" + color.Reset)

    // Pure blue (hue=240)
    blue := color.HSLToRGB(240, 1.0, 0.5)
    fmt.Println(blue.ForegroundSeq() + "Blue" + color.Reset)

    // Pastel colors (reduced saturation)
    pastel := color.HSLToRGB(180, 0.5, 0.75)
    fmt.Println(pastel.ForegroundSeq() + "Pastel cyan" + color.Reset)
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
    msg := color.Red.Sprintf("Error: %s", "file not found")
    fmt.Println(msg)

    // Sprint with color
    colored := color.Green.Sprint("Success: ", 42, " items processed")
    fmt.Println(colored)

    // Bold text
    fmt.Println(color.ApplyBold("Important!"))

    // Dim text
    fmt.Println(color.ApplyDim("Less important"))
}
```

### Conditional Coloring

```go
package main

import (
    "fmt"
    "os"
    "github.com/deepnoodle-ai/wonton/color"
)

func main() {
    // Check if we should colorize based on terminal detection
    shouldColor := color.ShouldColorize(os.Stdout)

    if shouldColor {
        fmt.Println(color.Green.Apply("Colors enabled"))
    } else {
        fmt.Println("Colors disabled")
    }

    // Manually control colorization
    color.Enabled = false
    fmt.Println(color.Colorize(color.Red, "Won't be colored"))

    color.Enabled = true
    fmt.Println(color.Colorize(color.Red, "Will be colored"))

    // Conditional colorization
    isError := true
    msg := color.ColorizeIf(isError, color.Red, "Error occurred")
    fmt.Println(msg)
}
```

### NO_COLOR Support

```go
package main

import (
    "fmt"
    "os"
    "github.com/deepnoodle-ai/wonton/color"
)

func main() {
    // Respects NO_COLOR environment variable automatically
    // If NO_COLOR is set, colors are disabled by default

    // Check current state
    fmt.Printf("Colors enabled: %v\n", color.Enabled)

    // Terminal detection that respects NO_COLOR
    if color.ShouldColorize(os.Stdout) {
        fmt.Println(color.Green.Apply("Colorized output"))
    } else {
        fmt.Println("Plain output")
    }
}
```

### Color Progress Bar

```go
package main

import (
    "fmt"
    "github.com/deepnoodle-ai/wonton/color"
)

func main() {
    total := 50
    gradient := color.Gradient(
        color.NewRGB(255, 0, 0),    // Red
        color.NewRGB(0, 255, 0),    // Green
        total,
    )

    for i := 0; i <= total; i++ {
        progress := float64(i) / float64(total)
        filled := int(progress * 50)

        bar := ""
        for j := 0; j < filled; j++ {
            bar += gradient[j].ForegroundSeq() + "█" + color.Reset
        }
        for j := filled; j < 50; j++ {
            bar += "░"
        }

        fmt.Printf("\r[%s] %.0f%%", bar, progress*100)
    }
    fmt.Println()
}
```

## API Reference

### Color Type

| Constant/Function | Description | Value/Return |
|-------------------|-------------|--------------|
| `NoColor` | Represents absence of color | `-1` |
| `Black`, `Red`, `Green`, `Yellow`, `Blue`, `Magenta`, `Cyan`, `White` | Standard colors | `0-7` |
| `BrightBlack` ... `BrightWhite` | Bright colors | `8-15` |
| `Palette(n)` | Extended palette color | `Color` (16-255) |

### Color Methods

| Method | Description | Parameters | Returns |
|--------|-------------|------------|---------|
| `Apply(text)` | Apply color as foreground | `string` | `string` |
| `ApplyDim(text)` | Apply color with dim attribute | `string` | `string` |
| `ApplyBg(text)` | Apply color as background | `string` | `string` |
| `Sprintf(format, args...)` | Format string with color | `string`, `...any` | `string` |
| `Sprint(args...)` | Sprint with color | `...any` | `string` |
| `ForegroundSeq()` | Get ANSI foreground escape sequence | None | `string` |
| `BackgroundSeq()` | Get ANSI background escape sequence | None | `string` |
| `ForegroundSeqDim()` | Get dim foreground escape sequence | None | `string` |
| `ForegroundCode()` | Get SGR parameter for foreground | None | `string` |
| `BackgroundCode()` | Get SGR parameter for background | None | `string` |

### RGB Type

| Function | Description | Parameters | Returns |
|----------|-------------|------------|---------|
| `NewRGB(r, g, b)` | Create RGB color | `uint8`, `uint8`, `uint8` | `RGB` |

### RGB Methods

| Method | Description | Parameters | Returns |
|--------|-------------|------------|---------|
| `ForegroundSeq()` | Get ANSI foreground escape sequence | None | `string` |
| `BackgroundSeq()` | Get ANSI background escape sequence | None | `string` |
| `Apply(text, bg)` | Apply RGB color to text | `string`, `bool` | `string` |

### Gradient Functions

| Function | Description | Parameters | Returns |
|----------|-------------|------------|---------|
| `Gradient(start, end, steps)` | Create gradient between two colors | `RGB`, `RGB`, `int` | `[]RGB` |
| `RainbowGradient(steps)` | Create classic rainbow gradient | `int` | `[]RGB` |
| `SmoothRainbow(steps)` | Create smooth rainbow using HSL | `int` | `[]RGB` |
| `MultiGradient(stops, steps)` | Create gradient through multiple stops | `[]RGB`, `int` | `[]RGB` |

### HSL Functions

| Function | Description | Parameters | Returns |
|----------|-------------|------------|---------|
| `HSLToRGB(h, s, l)` | Convert HSL to RGB | `float64`, `float64`, `float64` | `RGB` |

### Utility Functions

| Function | Description | Parameters | Returns |
|----------|-------------|------------|---------|
| `ApplyBold(text)` | Apply bold formatting | `string` | `string` |
| `ApplyDim(text)` | Apply dim formatting | `string` | `string` |
| `IsTerminal(f)` | Check if file is a terminal | `*os.File` | `bool` |
| `ShouldColorize(f)` | Check if colors should be used | `*os.File` | `bool` |
| `Colorize(c, text)` | Colorize if Enabled is true | `Color`, `string` | `string` |
| `ColorizeIf(enabled, c, text)` | Conditionally colorize | `bool`, `Color`, `string` | `string` |
| `ColorizeRGB(rgb, text)` | Colorize with RGB if Enabled | `RGB`, `string` | `string` |

### Constants

| Constant | Value | Description |
|----------|-------|-------------|
| `Escape` | `"\033["` | ANSI escape sequence prefix |
| `Reset` | `"\033[0m"` | Reset all attributes |
| `Bold` | `"\033[1m"` | Bold attribute |
| `Dim` | `"\033[2m"` | Dim/faint attribute |

### Global Variables

| Variable | Type | Description |
|----------|------|-------------|
| `Enabled` | `bool` | Controls global color output (respects NO_COLOR) |

## Related Packages

- **[tui](../tui/)** - Terminal UI library that uses color for rendering
- **[terminal](../terminal/)** - Low-level terminal control with ANSI support
- **[assert](../assert/)** - Test assertions that use colored diffs
- **[cli](../cli/)** - CLI framework with colored output helpers
