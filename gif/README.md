# gif

The gif package provides utilities for creating animated GIF images using only the Go standard library. It offers a builder-style API for constructing GIFs frame by frame with drawing primitives for shapes, lines, and fills.

## Usage Examples

### Basic Animation

```go
package main

import (
    "log"

    "github.com/deepnoodle-ai/wonton/gif"
)

func main() {
    // Create a 200x200 GIF
    g := gif.New(200, 200)

    // Add 20 frames
    for i := 0; i < 20; i++ {
        g.AddFrame(func(f *gif.Frame) {
            // Fill background
            f.Fill(gif.White)

            // Draw a moving circle
            x := 50 + i*5
            y := 100
            f.FillCircle(x, y, 20, gif.Red)
        })
    }

    // Save to file
    if err := g.Save("animation.gif"); err != nil {
        log.Fatal(err)
    }
}
```

### Custom Frame Delays

```go
g := gif.New(100, 100)

// Add frame with custom delay (in hundredths of a second)
g.AddFrameWithDelay(func(f *gif.Frame) {
    f.Fill(gif.Blue)
}, 50) // 500ms delay

g.AddFrameWithDelay(func(f *gif.Frame) {
    f.Fill(gif.Red)
}, 100) // 1 second delay

g.Save("slow-animation.gif")
```

### Custom Palette

```go
// Create a custom color palette
palette := gif.Palette{
    gif.White,
    gif.Black,
    gif.RGB(255, 100, 0),  // Orange
    gif.RGB(0, 150, 255),  // Light blue
    gif.RGB(255, 0, 255),  // Magenta
}

g := gif.NewWithPalette(150, 150, palette)

g.AddFrame(func(f *gif.Frame) {
    f.Fill(gif.White)
    f.FillCircle(75, 75, 40, gif.RGB(255, 100, 0))
})

g.Save("custom-colors.gif")
```

### Drawing Shapes

```go
g := gif.New(300, 300)

g.AddFrame(func(f *gif.Frame) {
    f.Fill(gif.White)

    // Draw rectangle outline
    f.DrawRect(50, 50, 100, 80, gif.Black)

    // Fill rectangle
    f.FillRect(180, 50, 100, 80, gif.Blue)

    // Draw circle outline
    f.DrawCircle(100, 200, 30, gif.Red)

    // Fill circle
    f.FillCircle(230, 200, 30, gif.Green)

    // Draw line
    f.DrawLine(20, 20, 280, 280, gif.Black)
})

g.Save("shapes.gif")
```

### Loop Control

```go
g := gif.New(100, 100)

// Play once (no loop)
g.SetLoopCount(-1)

// Loop 5 times
g.SetLoopCount(5)

// Loop forever (default)
g.SetLoopCount(0)

// Add frames...
g.Save("looping.gif")
```

### Working with Pixels

```go
g := gif.New(100, 100)

g.AddFrame(func(f *gif.Frame) {
    f.Fill(gif.White)

    // Set individual pixels
    for x := 0; x < f.Width(); x++ {
        for y := 0; y < f.Height(); y++ {
            // Create a gradient
            intensity := uint8(x * 255 / f.Width())
            color := gif.RGB(intensity, intensity, intensity)
            f.SetPixel(x, y, color)
        }
    }
})

g.Save("gradient.gif")
```

### Using Palette Indices

```go
palette := gif.Palette{gif.White, gif.Black, gif.Red, gif.Green, gif.Blue}
g := gif.NewWithPalette(100, 100, palette)

g.AddFrame(func(f *gif.Frame) {
    // Fast pixel setting using palette indices
    for y := 0; y < f.Height(); y++ {
        for x := 0; x < f.Width(); x++ {
            idx := uint8((x + y) % len(palette))
            f.SetPixelIndex(x, y, idx)
        }
    }
})

g.Save("indexed.gif")
```

### Bouncing Ball Animation

```go
g := gif.New(200, 200)

ballY := 20
velocityY := 3
gravity := 1

for i := 0; i < 100; i++ {
    g.AddFrame(func(f *gif.Frame) {
        f.Fill(gif.White)
        f.FillCircle(100, ballY, 15, gif.Red)
    })

    ballY += velocityY
    velocityY += gravity

    // Bounce when hitting bottom
    if ballY >= 180 {
        velocityY = -velocityY * 9 / 10 // Damping
        ballY = 180
    }
}

g.Save("bouncing-ball.gif")
```

### Grayscale Palette

```go
// Create grayscale palette with 256 shades
palette := gif.Grayscale(256)
g := gif.NewWithPalette(200, 200, palette)

g.AddFrame(func(f *gif.Frame) {
    for y := 0; y < f.Height(); y++ {
        for x := 0; x < f.Width(); x++ {
            // Create radial gradient
            dx := x - 100
            dy := y - 100
            dist := int(math.Sqrt(float64(dx*dx + dy*dy)))
            shade := uint8(255 - min(dist*2, 255))
            f.SetPixel(x, y, gif.RGB(shade, shade, shade))
        }
    }
})

g.Save("grayscale.gif")
```

### Web-Safe Colors

```go
// Use 216-color web-safe palette
palette := gif.WebSafe()
g := gif.NewWithPalette(300, 300, palette)

// Draw with web-safe colors
g.AddFrame(func(f *gif.Frame) {
    f.Fill(gif.White)
    // Colors will be mapped to nearest web-safe color
    f.FillCircle(150, 150, 100, gif.RGB(123, 45, 67))
})

g.Save("websafe.gif")
```

### Exporting to Bytes

```go
g := gif.New(100, 100)

// Add frames...

// Get as byte slice instead of saving to file
data, err := g.Bytes()
if err != nil {
    log.Fatal(err)
}

// Use data (e.g., send over HTTP, embed in response)
```

### Encoding to Writer

```go
import (
    "bytes"
    "io"
)

g := gif.New(100, 100)
// Add frames...

var buf bytes.Buffer
if err := g.Encode(&buf); err != nil {
    log.Fatal(err)
}

// Use buf.Bytes() or write to any io.Writer
```

## API Reference

### Constructor Functions

| Function | Description | Parameters | Returns |
|----------|-------------|------------|---------|
| `New(width, height)` | Creates GIF with default palette | `int`, `int` | `*GIF` |
| `NewWithPalette(w, h, pal)` | Creates GIF with custom palette | `int`, `int`, `Palette` | `*GIF` |

### GIF Methods

| Method | Description | Parameters | Returns |
|--------|-------------|------------|---------|
| `AddFrame(fn)` | Adds frame with 100ms delay | `func(*Frame)` | `*GIF` |
| `AddFrameWithDelay(fn, delay)` | Adds frame with custom delay | `func(*Frame)`, `int` | `*GIF` |
| `AddImage(img, delay)` | Adds existing paletted image | `*image.Paletted`, `int` | `*GIF` |
| `SetLoopCount(count)` | Sets loop count (0=forever, -1=once) | `int` | `*GIF` |
| `Width()` | Returns GIF width | - | `int` |
| `Height()` | Returns GIF height | - | `int` |
| `FrameCount()` | Returns number of frames | - | `int` |
| `Save(filename)` | Saves to file | `string` | `error` |
| `Encode(w)` | Encodes to writer | `io.Writer` | `error` |
| `Bytes()` | Returns as byte slice | - | `([]byte, error)` |

### Frame Drawing Methods

| Method | Description | Parameters | Returns |
|--------|-------------|------------|---------|
| `SetPixel(x, y, color)` | Sets a single pixel | `int`, `int`, `color.Color` | - |
| `SetPixelIndex(x, y, idx)` | Sets pixel by palette index | `int`, `int`, `uint8` | - |
| `Fill(color)` | Fills entire frame | `color.Color` | - |
| `FillRect(x, y, w, h, c)` | Fills rectangle | `int`, `int`, `int`, `int`, `color.Color` | - |
| `DrawRect(x, y, w, h, c)` | Draws rectangle outline | `int`, `int`, `int`, `int`, `color.Color` | - |
| `DrawLine(x0, y0, x1, y1, c)` | Draws line (Bresenham) | `int`, `int`, `int`, `int`, `color.Color` | - |
| `DrawCircle(cx, cy, r, c)` | Draws circle outline | `int`, `int`, `int`, `color.Color` | - |
| `FillCircle(cx, cy, r, c)` | Fills circle | `int`, `int`, `int`, `color.Color` | - |
| `Width()` | Returns frame width | - | `int` |
| `Height()` | Returns frame height | - | `int` |
| `Image()` | Returns underlying image | - | `*image.Paletted` |

### Color Functions

| Function | Description | Parameters | Returns |
|----------|-------------|------------|---------|
| `RGB(r, g, b)` | Creates RGB color | `uint8`, `uint8`, `uint8` | `color.RGBA` |
| `RGBA(r, g, b, a)` | Creates RGBA color | `uint8`, `uint8`, `uint8`, `uint8` | `color.RGBA` |
| `Grayscale(n)` | Creates n-shade grayscale palette | `int` | `Palette` |
| `WebSafe()` | Creates 216-color web-safe palette | - | `Palette` |

### Pre-defined Colors

| Constant | Value |
|----------|-------|
| `Black` | `color.RGBA{0, 0, 0, 255}` |
| `White` | `color.RGBA{255, 255, 255, 255}` |
| `Red` | `color.RGBA{255, 0, 0, 255}` |
| `Green` | `color.RGBA{0, 255, 0, 255}` |
| `Blue` | `color.RGBA{0, 0, 255, 255}` |
| `Yellow` | `color.RGBA{255, 255, 0, 255}` |
| `Cyan` | `color.RGBA{0, 255, 255, 255}` |
| `Magenta` | `color.RGBA{255, 0, 255, 255}` |
| `Transparent` | `color.RGBA{0, 0, 0, 0}` |

### Pre-defined Palettes

| Constant | Description | Size |
|----------|-------------|------|
| `DefaultPalette` | Basic 8-color palette | 8 colors |

### Types

| Type | Description |
|------|-------------|
| `Palette` | Alias for `[]color.Color` (max 256 colors) |
| `GIF` | Animated GIF builder |
| `Frame` | Single frame with drawing methods |

## Frame Timing

Frame delays are specified in hundredths of a second:
- `10` = 100ms (10 FPS)
- `5` = 50ms (20 FPS)
- `3` = 30ms (33 FPS)
- `2` = 20ms (50 FPS)

Default delay when using `AddFrame()` is `10` (100ms).

## Palette Limitations

GIF format supports up to 256 colors per frame. Colors not in the palette are automatically matched to the nearest palette color.

## Drawing Algorithms

- Lines use Bresenham's line algorithm for efficiency
- Circles use midpoint circle algorithm
- Filled circles use simple pixel scanning

## Performance Tips

1. Use `SetPixelIndex()` instead of `SetPixel()` when possible (faster)
2. Keep palettes small (fewer colors = smaller files)
3. Use grayscale palettes for monochrome animations
4. Consider frame delay vs. file size tradeoffs

## Related Packages

- [color](../color/) - Advanced color manipulation and gradients
- [terminal](../terminal/) - Terminal graphics and animations
- [termsession](../termsession/) - Terminal session recording

## Implementation Notes

- Uses only Go standard library (image/gif package)
- All coordinates are in pixels, origin at top-left (0, 0)
- Drawing operations clip to frame boundaries automatically
- GIF disposal method is set to DisposalBackground (clear to background)
- No built-in text rendering (can use image.Draw for text)
- Thread-safe for building (but not for concurrent frame modification)
