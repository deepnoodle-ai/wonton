package thumbnail

import (
	"image"
	"image/color"
	"image/draw"
	"path"
	"strings"
	"sync"
	"unicode/utf8"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// Card palette — a muted slate scheme that reads as neutral platform chrome
// rather than decoration.
var (
	cardBackground  = color.RGBA{R: 0xF1, G: 0xF4, B: 0xF8, A: 0xFF}
	whiteBackground = color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
	headerColor     = color.RGBA{R: 0x2E, G: 0x33, B: 0x40, A: 0xFF}
	titleColor      = color.RGBA{R: 0x1F, G: 0x24, B: 0x2E, A: 0xFF}
	bodyColor       = color.RGBA{R: 0x4A, G: 0x51, B: 0x5E, A: 0xFF}
	mutedColor      = color.RGBA{R: 0x8A, G: 0x91, B: 0x9E, A: 0xFF}
	accentColor     = color.RGBA{R: 0x3B, G: 0x82, B: 0xA6, A: 0xFF}
)

var (
	fontsOnce     sync.Once
	regularFont   *opentype.Font
	monoFont      *opentype.Font
	fontLoadErr   error
	faceCache     = map[faceKey]font.Face{}
	faceCacheLock sync.Mutex
)

type faceKey struct {
	mono bool
	size int
}

func loadFonts() error {
	fontsOnce.Do(func() {
		regularFont, fontLoadErr = opentype.Parse(goregular.TTF)
		if fontLoadErr != nil {
			return
		}
		monoFont, fontLoadErr = opentype.Parse(gomono.TTF)
	})
	return fontLoadErr
}

func face(mono bool, sizePx float64) (font.Face, error) {
	if err := loadFonts(); err != nil {
		return nil, err
	}
	size := int(sizePx + 0.5)
	if size < 6 {
		size = 6
	}
	key := faceKey{mono: mono, size: size}
	faceCacheLock.Lock()
	defer faceCacheLock.Unlock()
	if f, ok := faceCache[key]; ok {
		return f, nil
	}
	src := regularFont
	if mono {
		src = monoFont
	}
	f, err := opentype.NewFace(src, &opentype.FaceOptions{Size: float64(size), DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		return nil, err
	}
	faceCache[key] = f
	return f, nil
}

// renderDocumentCard draws a preview card for UTF-8 text/markdown/json/code
// content: a type label, the file name, and the first few raw lines in a
// monospace face. Invalid-UTF-8 sources fall through to the typed fallback card.
func renderDocumentCard(req Request) (Result, error) {
	if !utf8.Valid(req.Source) {
		return renderFallbackCard(req, "non-utf8 text source")
	}
	if err := loadFonts(); err != nil {
		return renderFallbackCard(req, "fonts unavailable")
	}
	scale := float64(req.Height) / 200.0
	canvas := newCanvas(req)

	pad := int(16 * scale)
	label := documentLabel(req)
	if y, err := drawLabel(canvas, label, pad, int(20*scale), scale); err == nil {
		drawTitle(canvas, baseName(req.Name), pad, y+int(22*scale), scale, req.Width-2*pad)
		drawBodyLines(canvas, string(req.Source), pad, y+int(46*scale), scale, req.Width-2*pad, req.Height-pad)
	}
	return encodePNG(canvas, req, RendererDocument, "")
}

// renderFallbackCard draws a generic typed file card for media/binary/unknown
// content (and as the safety net for any failed render).
func renderFallbackCard(req Request, note string) (Result, error) {
	if err := loadFonts(); err != nil {
		// Last resort: a blank neutral tile still beats a broken image.
		canvas := newCanvas(req)
		return encodePNG(canvas, req, RendererFallback, "fonts unavailable: "+note)
	}
	scale := float64(req.Height) / 200.0
	canvas := newCanvas(req)
	pad := int(16 * scale)

	big := strings.ToUpper(typeLabel(req))
	if f, err := face(false, 34*scale); err == nil {
		w := font.MeasureString(f, big)
		x := (fixed.I(req.Width) - w) / 2
		drawTextFixed(canvas, f, accentColor, x, fixed.I(req.Height/2-int(6*scale)), big)
	}
	if f, err := face(false, 12*scale); err == nil {
		sub := normalizeMime(req.MimeType)
		if sub == "" {
			sub = "file"
		}
		sub = truncate(f, sub, fixed.I(req.Width-2*pad))
		w := font.MeasureString(f, sub)
		x := (fixed.I(req.Width) - w) / 2
		drawTextFixed(canvas, f, mutedColor, x, fixed.I(req.Height/2+int(20*scale)), sub)
	}
	if name := baseName(req.Name); name != "" {
		drawTitle(canvas, name, pad, req.Height-pad, scale, req.Width-2*pad)
	}
	return encodePNG(canvas, req, RendererFallback, note)
}

func newCanvas(req Request) *image.RGBA {
	canvas := image.NewRGBA(image.Rect(0, 0, req.Width, req.Height))
	draw.Draw(canvas, canvas.Bounds(), image.NewUniform(cardBackground), image.Point{}, draw.Src)
	return canvas
}

// drawLabel draws a small uppercase label at the top and returns the y baseline
// used so callers can stack content below it.
func drawLabel(dst draw.Image, label string, x, top int, scale float64) (int, error) {
	f, err := face(false, 11*scale)
	if err != nil {
		return top, err
	}
	baseline := top + f.Metrics().Ascent.Ceil()
	drawTextFixed(dst, f, headerColor, fixed.I(x), fixed.I(baseline), strings.ToUpper(label))
	return baseline, nil
}

func drawTitle(dst draw.Image, title string, x, baseline int, scale float64, maxW int) {
	if title == "" {
		return
	}
	f, err := face(false, 15*scale)
	if err != nil {
		return
	}
	title = truncate(f, title, fixed.I(maxW))
	drawTextFixed(dst, f, titleColor, fixed.I(x), fixed.I(baseline), title)
}

func drawBodyLines(dst draw.Image, body string, x, top int, scale float64, maxW, maxY int) {
	f, err := face(true, 11*scale)
	if err != nil {
		return
	}
	lineH := (f.Metrics().Ascent + f.Metrics().Descent).Ceil() + int(2*scale)
	if lineH < 8 {
		lineH = 8
	}
	baseline := top + f.Metrics().Ascent.Ceil()
	for _, raw := range strings.Split(body, "\n") {
		if baseline > maxY {
			break
		}
		line := normalizeLine(raw)
		if line == "" {
			baseline += lineH
			continue
		}
		line = truncate(f, line, fixed.I(maxW))
		drawTextFixed(dst, f, bodyColor, fixed.I(x), fixed.I(baseline), line)
		baseline += lineH
	}
}

func drawTextFixed(dst draw.Image, f font.Face, col color.Color, x, baseline fixed.Int26_6, s string) {
	d := &font.Drawer{
		Dst:  dst,
		Src:  image.NewUniform(col),
		Face: f,
		Dot:  fixed.Point26_6{X: x, Y: baseline},
	}
	d.DrawString(s)
}

// truncate shortens s with an ellipsis so it fits within maxW.
func truncate(f font.Face, s string, maxW fixed.Int26_6) string {
	if font.MeasureString(f, s) <= maxW {
		return s
	}
	const ell = "…"
	ellW := font.MeasureString(f, ell)
	runes := []rune(s)
	for len(runes) > 0 {
		runes = runes[:len(runes)-1]
		if font.MeasureString(f, string(runes))+ellW <= maxW {
			return strings.TrimRight(string(runes), " ") + ell
		}
	}
	return ell
}

// normalizeLine expands tabs and strips control characters so a single source
// line renders predictably.
func normalizeLine(s string) string {
	s = strings.ReplaceAll(s, "\t", "    ")
	s = strings.TrimRight(s, "\r")
	var b strings.Builder
	for _, r := range s {
		if r < 0x20 || r == 0x7f {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func documentLabel(req Request) string {
	if ext := codeExtension(req.Name); ext != "" {
		return ext
	}
	m := normalizeMime(req.MimeType)
	if i := strings.IndexByte(m, '/'); i >= 0 {
		return m[i+1:]
	}
	if m != "" {
		return m
	}
	return "text"
}

// typeLabel is the big glyph on a fallback card: extension, or the MIME family.
func typeLabel(req Request) string {
	if ext := codeExtension(req.Name); ext != "" {
		return ext
	}
	if name := baseName(req.Name); name != "" {
		if i := strings.LastIndexByte(name, '.'); i >= 0 && i < len(name)-1 {
			return name[i+1:]
		}
	}
	m := normalizeMime(req.MimeType)
	if i := strings.IndexByte(m, '/'); i >= 0 {
		return m[:i]
	}
	if m != "" {
		return m
	}
	return "file"
}

func baseName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	return path.Base(name)
}
