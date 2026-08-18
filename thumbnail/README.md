# thumbnail

Render small preview images for files: a downscaled version of a raster image, or a synthetic card for text, code, and everything else.

## Summary

`Render` never fails on bad input. An undecodable image, an oversized one, or a format the package cannot read all degrade to a typed card, so every file gets a stable visual and callers need no fallback path of their own. A non-nil error means an encoding fault, not bad input.

The package is pure Go and depends only on the standard library and `golang.org/x/image`: stdlib decoders for PNG/JPEG/GIF, `x/image/webp` for WebP, `x/image/draw` for Catmull-Rom downscaling, and `x/image/font` with the embedded Go fonts for the cards. There is no FFmpeg, Chromium, or ImageMagick dependency and no system-font requirement, so it runs unchanged in a minimal container image.

Output is `image/jpeg` (quality 82) for opaque photographic thumbnails and `image/png` for cards and for images that may carry transparency.

## Usage Examples

### Rendering a Thumbnail

```go
package main

import (
    "fmt"
    "log"
    "os"

    "github.com/deepnoodle-ai/wonton/thumbnail"
)

func main() {
    source, err := os.ReadFile("screenshot.png")
    if err != nil {
        log.Fatal(err)
    }

    res, err := thumbnail.Render(thumbnail.Request{
        MimeType: "image/png",
        Name:     "screenshot.png",
        Source:   source,
        Width:    320,
        Height:   200,
    })
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(res.MimeType, res.Renderer, len(res.Bytes)) // image/png image 14231
    os.WriteFile("thumb.png", res.Bytes, 0o644)
}
```

Width and Height default to 320x200 when unset. The source image is scaled to
fit inside the box and centered, preserving aspect ratio; it is never upscaled
past its native size, and the surrounding area is filled with the card
background (white for JPEG sources).

### Text and Code Cards

Text, markdown, JSON, and source files render as a card showing a type label,
the file name, and the first few lines in a monospace face:

```go
res, _ := thumbnail.Render(thumbnail.Request{
    MimeType: "text/markdown",
    Name:     "notes.md",
    Source:   []byte("# Release notes\n\n- Fixed the thing\n"),
})
// res.Renderer == thumbnail.RendererDocument
```

The type is detected from the MIME type or, when that is missing or generic,
from the file extension — so `main.go` renders as a code card even with no
MIME type at all.

### Fallback Cards

Anything else — PDFs, archives, video, unknown binaries — renders as a typed
card with the extension or MIME family as its glyph:

```go
res, _ := thumbnail.Render(thumbnail.Request{MimeType: "application/pdf", Name: "report.pdf"})
// res.Renderer == thumbnail.RendererFallback
```

`Result.Note` carries the diagnostic when a fallback happened because
something failed (a corrupt image, a source over the pixel cap, an
invalid-UTF-8 text file). It is empty on the happy path.

### Budgeting the Read

`Render` needs the whole file for an image but only a leading slice for text.
`IsImageMime` and `IsTextMime` let a caller decide how many bytes to read
before rendering:

```go
switch {
case thumbnail.IsImageMime(mime):
    source = readAll(object) // capped by the caller
case thumbnail.IsTextMime(mime, name):
    source = readN(object, 8<<10) // a few lines is plenty
}
```

### Bounding Decode Cost

Decoding allocates roughly `width * height * 4` bytes, so a hostile image can
cost far more memory than its compressed size suggests. `Render` reads the
image header first and falls back to a card when the decoded area would exceed
`DefaultMaxSourcePixels` (24 MP). Set `MaxSourcePixels` to override:

```go
res, _ := thumbnail.Render(thumbnail.Request{
    MimeType:        mime,
    Name:            name,
    Source:          source,
    MaxSourcePixels: 4 << 20, // 4 MP
})
```

Decoder panics on corrupt input are recovered and turned into a fallback card.

## API Reference

### Functions

| Function                | Description                                              | Returns           |
| ----------------------- | -------------------------------------------------------- | ----------------- |
| `Render(req)`           | Renders a thumbnail, degrading to a card on bad input    | `Result`, `error` |
| `IsImageMime(mime)`     | Whether the MIME is a raster format this package decodes | `bool`            |
| `IsTextMime(mime, name)`| Whether the MIME/name should render as a document card   | `bool`            |

### Request

| Field             | Type     | Description                                                 |
| ----------------- | -------- | ----------------------------------------------------------- |
| `MimeType`        | `string` | Source MIME type; parameters (`; charset=…`) are ignored     |
| `Name`            | `string` | Display name, used for card labels and extension detection   |
| `Source`          | `[]byte` | Source bytes; whole file for images, leading slice for text  |
| `Width`, `Height` | `int`    | Target box; defaults to 320x200 when unset                   |
| `MaxSourcePixels` | `int`    | Decoded-area cap; `DefaultMaxSourcePixels` when unset        |

### Result

| Field             | Type     | Description                                          |
| ----------------- | -------- | ---------------------------------------------------- |
| `Bytes`           | `[]byte` | Encoded thumbnail                                    |
| `MimeType`        | `string` | `image/jpeg` or `image/png`                          |
| `Width`, `Height` | `int`    | Rendered dimensions                                  |
| `Renderer`        | `string` | `RendererImage`, `RendererDocument`, or `RendererFallback` |
| `Note`            | `string` | Fallback reason; empty on the happy path             |

### Supported Sources

| Category | Handled as        | Formats                                                        |
| -------- | ----------------- | -------------------------------------------------------------- |
| Image    | Downscaled raster | PNG, JPEG, GIF, WebP                                            |
| Text     | Document card     | `text/*`, JSON, XML, JavaScript, `+json`/`+xml`, code extensions |
| Other    | Typed card        | Everything else                                                 |

## Related Packages

- **[gif](../gif/)** — animated GIF creation, including terminal recordings
- **[fetch](../fetch/)** — fetching the bytes you are thumbnailing
