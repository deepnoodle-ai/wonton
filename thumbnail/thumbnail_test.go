package thumbnail

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"strings"
	"sync"
	"testing"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"

	"github.com/deepnoodle-ai/wonton/assert"
)

func sampleImage(w, h int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: 128, A: 255})
		}
	}
	return img
}

func pngBytes(t *testing.T, w, h int) []byte {
	t.Helper()
	var buf bytes.Buffer
	assert.NoError(t, png.Encode(&buf, sampleImage(w, h)))
	return buf.Bytes()
}

func jpegBytes(t *testing.T, w, h int) []byte {
	t.Helper()
	var buf bytes.Buffer
	assert.NoError(t, jpeg.Encode(&buf, sampleImage(w, h), &jpeg.Options{Quality: 90}))
	return buf.Bytes()
}

func decodeResult(t *testing.T, res Result) image.Image {
	t.Helper()
	img, _, err := image.Decode(bytes.NewReader(res.Bytes))
	assert.NoError(t, err)
	return img
}

func TestRenderPNGImageProducesPNGThumbnail(t *testing.T) {
	res, err := Render(Request{MimeType: "image/png", Name: "shot.png", Source: pngBytes(t, 800, 600), Width: 320, Height: 200})
	assert.NoError(t, err)
	assert.Equal(t, res.Renderer, RendererImage)
	assert.Equal(t, res.MimeType, "image/png") // png source may have alpha
	assert.Equal(t, res.Width, 320)
	assert.Equal(t, res.Height, 200)
	assert.Empty(t, res.Note)
	img := decodeResult(t, res)
	assert.Equal(t, img.Bounds().Dx(), 320)
	assert.Equal(t, img.Bounds().Dy(), 200)
}

func TestRenderJPEGImageProducesJPEGThumbnail(t *testing.T) {
	res, err := Render(Request{MimeType: "image/jpeg", Name: "photo.jpg", Source: jpegBytes(t, 640, 480), Width: 320, Height: 200})
	assert.NoError(t, err)
	assert.Equal(t, res.Renderer, RendererImage)
	assert.Equal(t, res.MimeType, "image/jpeg")
	decodeResult(t, res)
}

func TestRenderHonorsMimeParameters(t *testing.T) {
	res, err := Render(Request{MimeType: "image/jpeg; charset=binary", Name: "photo.jpg", Source: jpegBytes(t, 64, 64), Width: 320, Height: 200})
	assert.NoError(t, err)
	assert.Equal(t, res.Renderer, RendererImage)
	assert.Equal(t, res.MimeType, "image/jpeg")
}

func TestRenderDefaultsToStandardSize(t *testing.T) {
	res, err := Render(Request{MimeType: "image/png", Name: "shot.png", Source: pngBytes(t, 800, 600)})
	assert.NoError(t, err)
	assert.Equal(t, res.Width, 320)
	assert.Equal(t, res.Height, 200)
	img := decodeResult(t, res)
	assert.Equal(t, img.Bounds().Dx(), 320)
	assert.Equal(t, img.Bounds().Dy(), 200)
}

func TestRenderTextProducesDocumentCard(t *testing.T) {
	res, err := Render(Request{
		MimeType: "text/markdown",
		Name:     "notes.md",
		Source:   []byte("# Title\n\nsome body text\nmore lines here\n"),
		Width:    320,
		Height:   200,
	})
	assert.NoError(t, err)
	assert.Equal(t, res.Renderer, RendererDocument)
	assert.Equal(t, res.MimeType, "image/png")
	img := decodeResult(t, res)
	assert.Equal(t, img.Bounds().Dx(), 320)
}

func TestRenderDetectsCodeByExtensionWithoutMime(t *testing.T) {
	res, err := Render(Request{Name: "main.go", Source: []byte("package main\n"), Width: 320, Height: 200})
	assert.NoError(t, err)
	assert.Equal(t, res.Renderer, RendererDocument)
}

func TestRenderUnknownBinaryProducesFallbackCard(t *testing.T) {
	res, err := Render(Request{MimeType: "application/pdf", Name: "report.pdf", Width: 320, Height: 200})
	assert.NoError(t, err)
	assert.Equal(t, res.Renderer, RendererFallback)
	assert.Equal(t, res.MimeType, "image/png")
	decodeResult(t, res)
}

func TestRenderCorruptImageFallsBackToCard(t *testing.T) {
	// An image MIME with non-decodable bytes must not error — it degrades to a
	// typed card so the source still gets a stable visual.
	res, err := Render(Request{MimeType: "image/png", Name: "broken.png", Source: []byte("not a real png"), Width: 320, Height: 200})
	assert.NoError(t, err)
	assert.Equal(t, res.Renderer, RendererFallback)
	assert.NotEmpty(t, res.Note, "the fallback records why the image render failed")
	decodeResult(t, res)
}

func TestRenderOverPixelCapFallsBack(t *testing.T) {
	res, err := Render(Request{
		MimeType:        "image/png",
		Name:            "huge.png",
		Source:          pngBytes(t, 200, 200),
		Width:           320,
		Height:          200,
		MaxSourcePixels: 100, // force the decoded-area cap to trip
	})
	assert.NoError(t, err)
	assert.Equal(t, res.Renderer, RendererFallback)
}

// gifHeader returns a GIF header and logical screen descriptor claiming
// w x h pixels. image.DecodeConfig reads only this far.
func gifHeader(w, h uint16) []byte {
	var buf bytes.Buffer
	buf.WriteString("GIF89a")
	_ = binary.Write(&buf, binary.LittleEndian, w)
	_ = binary.Write(&buf, binary.LittleEndian, h)
	buf.Write([]byte{0, 0, 0}) // no global color table, background, aspect
	return buf.Bytes()
}

func TestRenderImagePixelCapDoesNotOverflow(t *testing.T) {
	// 65535*65535 overflows a 32-bit int and wraps negative, which would
	// pass the cap. (PNG's decoder rejects such sizes itself on 32-bit
	// platforms; GIF's does not.)
	_, err := renderImage(Request{
		MimeType: "image/gif",
		Source:   gifHeader(65535, 65535),
		Width:    320,
		Height:   200,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds decode cap")
}

func TestRenderConcurrentCards(t *testing.T) {
	// Cards share fonts. Run with -race to check that renders do not share
	// mutable font.Face state.
	// Errors are collected and checked on the test goroutine: assert calls
	// t.Fatalf, which must not run on a worker goroutine.
	var wg sync.WaitGroup
	errs := make(chan error, 8*5*2)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 5; j++ {
				_, err := Render(Request{MimeType: "text/plain", Name: "a.txt", Source: []byte("hello\nworld"), Width: 320, Height: 200})
				errs <- err
				_, err = Render(Request{MimeType: "application/pdf", Name: "b.pdf", Width: 320, Height: 200})
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		assert.NoError(t, err)
	}
}

// countingFace counts glyph advance lookups.
type countingFace struct {
	font.Face
	n int
}

func (c *countingFace) GlyphAdvance(r rune) (fixed.Int26_6, bool) {
	c.n++
	return c.Face.GlyphAdvance(r)
}

func TestTruncateIsLinear(t *testing.T) {
	f, err := face(true, 11)
	assert.NoError(t, err)
	cf := &countingFace{Face: f}
	long := strings.Repeat("a", 20_000)
	got := truncate(cf, long, fixed.I(300))
	assert.True(t, strings.HasSuffix(got, "…"))
	assert.True(t, font.MeasureString(f, got) <= fixed.I(300))
	assert.True(t, font.MeasureString(f, strings.TrimSuffix(got, "…")+"a…") > fixed.I(300), "truncate should keep the longest prefix that fits")
	assert.True(t, cf.n < 3*len(long), "GlyphAdvance called %d times", cf.n)
}

func TestTruncate(t *testing.T) {
	f, err := face(false, 12)
	assert.NoError(t, err)
	assert.Equal(t, "short", truncate(f, "short", fixed.I(1000)))
	assert.Equal(t, "…", truncate(f, "wide", fixed.I(1)))
	// Trailing spaces before the ellipsis are trimmed.
	w := font.MeasureString(f, "ab …")
	assert.Equal(t, "ab…", truncate(f, "ab cdef", w))
}

func TestRenderInvalidUTF8TextFallsBack(t *testing.T) {
	res, err := Render(Request{MimeType: "text/plain", Name: "data.txt", Source: []byte{0xff, 0xfe, 0xfd}, Width: 320, Height: 200})
	assert.NoError(t, err)
	assert.Equal(t, res.Renderer, RendererFallback)
}

func TestRenderIsDeterministic(t *testing.T) {
	req := Request{MimeType: "text/markdown", Name: "notes.md", Source: []byte("# Title\nbody\n"), Width: 320, Height: 200}
	first, err := Render(req)
	assert.NoError(t, err)
	second, err := Render(req)
	assert.NoError(t, err)
	assert.Equal(t, first.Bytes, second.Bytes, "the same request renders byte-identical output")
}

func TestContainRectFitsAndCenters(t *testing.T) {
	// Wider than the box: full width, letterboxed vertically.
	r := containRect(800, 400, 320, 200)
	assert.Equal(t, r.Dx(), 320)
	assert.Equal(t, r.Dy(), 160)
	assert.Equal(t, r.Min.X, 0)
	assert.Equal(t, r.Min.Y, 20)

	// Taller than the box: full height, pillarboxed horizontally.
	r = containRect(400, 800, 320, 200)
	assert.Equal(t, r.Dy(), 200)
	assert.Equal(t, r.Dx(), 100)
	assert.Equal(t, r.Min.X, 110)

	// Smaller than the box: never upscaled past native size.
	r = containRect(64, 32, 320, 200)
	assert.Equal(t, r.Dx(), 64)
	assert.Equal(t, r.Dy(), 32)

	// Degenerate input still yields the full box rather than an empty rect.
	assert.Equal(t, containRect(0, 0, 320, 200), image.Rect(0, 0, 320, 200))
}

func TestIsImageMime(t *testing.T) {
	for _, mime := range []string{"image/png", "IMAGE/JPEG", "image/jpg", "image/webp", "image/gif", "image/png; foo=bar"} {
		assert.True(t, IsImageMime(mime), "IsImageMime(%q)", mime)
	}
	for _, mime := range []string{"image/svg+xml", "image/tiff", "text/plain", ""} {
		assert.False(t, IsImageMime(mime), "IsImageMime(%q)", mime)
	}
}

func TestIsTextMime(t *testing.T) {
	for _, tc := range []struct{ mime, name string }{
		{"text/plain", ""},
		{"application/json", ""},
		{"application/xml", ""},
		{"application/javascript", ""},
		{"application/ld+json", ""},
		{"application/atom+xml", ""},
		{"", "main.go"},
		{"application/octet-stream", "query.sql"},
	} {
		assert.True(t, IsTextMime(tc.mime, tc.name), "IsTextMime(%q, %q)", tc.mime, tc.name)
	}
	for _, tc := range []struct{ mime, name string }{
		{"application/pdf", "report.pdf"},
		{"image/png", "shot.png"},
		{"", ""},
		{"", "archive"},
	} {
		assert.False(t, IsTextMime(tc.mime, tc.name), "IsTextMime(%q, %q)", tc.mime, tc.name)
	}
}
