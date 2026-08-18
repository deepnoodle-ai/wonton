// Package thumbnail renders small preview images for files: a downscaled
// version of a raster image, or a synthetic card for text, code, and
// everything else.
//
// [Render] never fails on bad input. An undecodable image, an oversized one,
// or a format the package cannot read all degrade to a typed card, so every
// file gets a stable visual and callers need no fallback path of their own.
//
// It is pure Go and depends only on the standard library and
// golang.org/x/image: stdlib decoders for PNG/JPEG/GIF, x/image/webp for WebP,
// x/image/draw for high-quality downscaling, and x/image/font with the
// embedded Go fonts for the cards. There is no FFmpeg, Chromium, or
// ImageMagick dependency and no system-font requirement, so it runs unchanged
// in a minimal container image.
//
// Output is image/jpeg (quality 82) for opaque photographic thumbnails and
// image/png for cards and images that may carry transparency.
package thumbnail

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"
	"strings"

	xdraw "golang.org/x/image/draw"

	// Register image decoders for image.Decode / image.DecodeConfig.
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	_ "golang.org/x/image/webp"
)

// DefaultMaxSourcePixels bounds the decoded area of a source image. Larger
// images fall back to a typed card rather than decoding (a decode allocates
// roughly width*height*4 bytes).
const DefaultMaxSourcePixels = 24 * 1000 * 1000 // 24 MP

const jpegQuality = 82

// Renderer names reported in [Result.Renderer] for diagnostics.
const (
	RendererImage    = "image"
	RendererDocument = "document"
	RendererFallback = "fallback"
)

// Request is the input to Render.
type Request struct {
	// MimeType is the source MIME type (e.g. image/png, text/markdown).
	MimeType string
	// Name is the display name of the source; used for card labels and
	// extension detection.
	Name string
	// Source is the source bytes. For images this should be the whole file
	// (the caller caps how much it reads); for text a leading slice is enough,
	// since only the first few lines are drawn.
	Source []byte
	// Width and Height are the target thumbnail box.
	Width  int
	Height int
	// MaxSourcePixels overrides DefaultMaxSourcePixels when > 0.
	MaxSourcePixels int
}

// Result is a rendered thumbnail.
type Result struct {
	Bytes    []byte
	MimeType string // image/jpeg or image/png
	Width    int
	Height   int
	Renderer string // RendererImage | RendererDocument | RendererFallback
	Note     string // fallback reason / diagnostic, empty on the happy path
}

// Render produces a thumbnail for the request. It never returns an error for
// an unsupported or malformed source: image failures and non-image types fall
// through to a deterministic typed card, so every input gets a stable visual.
// A non-nil error indicates an encoding fault, not bad input.
//
// Width and Height default to 320x200 when unset.
func Render(req Request) (res Result, err error) {
	w, h := req.Width, req.Height
	if w <= 0 || h <= 0 {
		v := defaultVariant()
		w, h = v.w, v.h
	}
	req.Width, req.Height = w, h

	// Guard against decoder panics on hostile/corrupt input by falling back.
	defer func() {
		if r := recover(); r != nil {
			res, err = renderFallbackCard(req, fmt.Sprintf("panic: %v", r))
		}
	}()

	switch {
	case isImageMime(req.MimeType):
		out, rerr := renderImage(req)
		if rerr != nil {
			return renderFallbackCard(req, rerr.Error())
		}
		return out, nil
	case isTextMime(req.MimeType, req.Name):
		return renderDocumentCard(req)
	default:
		return renderFallbackCard(req, "")
	}
}

func renderImage(req Request) (Result, error) {
	maxPixels := req.MaxSourcePixels
	if maxPixels <= 0 {
		maxPixels = DefaultMaxSourcePixels
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(req.Source))
	if err != nil {
		return Result{}, fmt.Errorf("decode config: %w", err)
	}
	if cfg.Width <= 0 || cfg.Height <= 0 {
		return Result{}, fmt.Errorf("non-positive source dimensions %dx%d", cfg.Width, cfg.Height)
	}
	if cfg.Width*cfg.Height > maxPixels {
		return Result{}, fmt.Errorf("source %dx%d exceeds decode cap", cfg.Width, cfg.Height)
	}
	src, _, err := image.Decode(bytes.NewReader(req.Source))
	if err != nil {
		return Result{}, fmt.Errorf("decode: %w", err)
	}

	opaque := mimeIsOpaque(req.MimeType)
	canvas := image.NewRGBA(image.Rect(0, 0, req.Width, req.Height))
	bg := cardBackground
	if opaque {
		bg = whiteBackground
	}
	draw.Draw(canvas, canvas.Bounds(), image.NewUniform(bg), image.Point{}, draw.Src)

	dst := containRect(src.Bounds().Dx(), src.Bounds().Dy(), req.Width, req.Height)
	xdraw.CatmullRom.Scale(canvas, dst, src, src.Bounds(), xdraw.Over, nil)

	if opaque {
		return encodeJPEG(canvas, req, RendererImage, "")
	}
	return encodePNG(canvas, req, RendererImage, "")
}

// containRect centers a srcW x srcH image scaled to fit inside dstW x dstH.
func containRect(srcW, srcH, dstW, dstH int) image.Rectangle {
	if srcW <= 0 || srcH <= 0 {
		return image.Rect(0, 0, dstW, dstH)
	}
	scale := float64(dstW) / float64(srcW)
	if s := float64(dstH) / float64(srcH); s < scale {
		scale = s
	}
	if scale > 1 {
		scale = 1 // never upscale past native size
	}
	w := int(float64(srcW) * scale)
	h := int(float64(srcH) * scale)
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	x := (dstW - w) / 2
	y := (dstH - h) / 2
	return image.Rect(x, y, x+w, y+h)
}

func encodeJPEG(img image.Image, req Request, renderer, note string) (Result, error) {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: jpegQuality}); err != nil {
		return Result{}, fmt.Errorf("encode jpeg: %w", err)
	}
	return Result{Bytes: buf.Bytes(), MimeType: "image/jpeg", Width: req.Width, Height: req.Height, Renderer: renderer, Note: note}, nil
}

func encodePNG(img image.Image, req Request, renderer, note string) (Result, error) {
	var buf bytes.Buffer
	enc := png.Encoder{CompressionLevel: png.BestCompression}
	if err := enc.Encode(&buf, img); err != nil {
		return Result{}, fmt.Errorf("encode png: %w", err)
	}
	return Result{Bytes: buf.Bytes(), MimeType: "image/png", Width: req.Width, Height: req.Height, Renderer: renderer, Note: note}, nil
}

func normalizeMime(mime string) string {
	mime = strings.ToLower(strings.TrimSpace(mime))
	if i := strings.IndexByte(mime, ';'); i >= 0 {
		mime = strings.TrimSpace(mime[:i])
	}
	return mime
}

func isImageMime(mime string) bool {
	switch normalizeMime(mime) {
	case "image/png", "image/jpeg", "image/jpg", "image/webp", "image/gif":
		return true
	}
	return false
}

// mimeIsOpaque reports whether the source format is reliably opaque (no alpha),
// so the thumbnail can be JPEG. PNG/GIF/WebP may carry transparency.
func mimeIsOpaque(mime string) bool {
	switch normalizeMime(mime) {
	case "image/jpeg", "image/jpg":
		return true
	}
	return false
}

func isTextMime(mime, name string) bool {
	m := normalizeMime(mime)
	if strings.HasPrefix(m, "text/") ||
		m == "application/json" ||
		m == "application/xml" ||
		m == "application/javascript" ||
		strings.HasSuffix(m, "+json") ||
		strings.HasSuffix(m, "+xml") {
		return true
	}
	return codeExtension(name) != ""
}

// codeExtension returns the lower-cased extension (without dot) for known
// code/text-like artifact names, or "".
func codeExtension(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	idx := strings.LastIndexByte(name, '.')
	if idx < 0 || idx == len(name)-1 {
		return ""
	}
	ext := name[idx+1:]
	switch ext {
	case "md", "markdown", "txt", "json", "jsonl", "yaml", "yml", "toml", "csv", "tsv",
		"go", "py", "js", "jsx", "ts", "tsx", "rb", "rs", "java", "c", "h", "cpp", "cc",
		"sh", "bash", "zsh", "sql", "html", "htm", "css", "scss", "xml", "ini", "env", "log":
		return ext
	}
	return ""
}

type variant struct{ w, h int }

func defaultVariant() variant { return variant{w: 320, h: 200} }

// IsImageMime reports whether the MIME is a raster format this package can
// decode and downscale. Exported for callers that decide how many source bytes
// to read before rendering.
func IsImageMime(mime string) bool { return isImageMime(mime) }

// IsTextMime reports whether the MIME/name should render as a text document
// card. Exported for the same byte-budget decision as IsImageMime.
func IsTextMime(mime, name string) bool { return isTextMime(mime, name) }
