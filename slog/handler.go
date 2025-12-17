// Package slog provides a colorized slog.Handler for terminal output.
//
// The handler writes tinted (colorized) logs with sensible defaults that
// work out of the box for most terminal applications. Colors are enabled
// by default and can be disabled via options.
//
// Basic usage:
//
//	logger := slog.New(wontonslog.NewHandler(os.Stderr, nil))
//	logger.Info("server started", "port", 8080)
//
// With options:
//
//	logger := slog.New(wontonslog.NewHandler(os.Stderr, &wontonslog.Options{
//	    Level:      slog.LevelDebug,
//	    TimeFormat: time.Kitchen,
//	    AddSource:  true,
//	}))
//
// Auto-detect terminal and disable colors when not a TTY:
//
//	logger := slog.New(wontonslog.NewHandler(os.Stderr, &wontonslog.Options{
//	    NoColor: !wontonslog.IsTerminal(os.Stderr),
//	}))
package slog

import (
	"context"
	"encoding"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/deepnoodle-ai/wonton/color"
)

const (
	ansiEsc           = '\u001b'
	defaultTimeFormat = time.StampMilli
)

// Handler implements slog.Handler with colorized output.
type Handler struct {
	attrsPrefix string
	groupPrefix string
	groups      []string

	mu *sync.Mutex
	w  io.Writer

	opts Options
}

// NewHandler creates a new colorized slog.Handler that writes to w.
// If opts is nil, default options are used.
//
// Default behavior:
//   - Colors enabled
//   - Level: Info
//   - Time format: time.StampMilli (e.g., "Jan _2 15:04:05.000")
//   - No source location
func NewHandler(w io.Writer, opts *Options) *Handler {
	if opts == nil {
		opts = &Options{}
	}
	opts.setDefaults()

	return &Handler{
		mu:   &sync.Mutex{},
		w:    w,
		opts: *opts,
	}
}

func (h *Handler) clone() *Handler {
	return &Handler{
		attrsPrefix: h.attrsPrefix,
		groupPrefix: h.groupPrefix,
		groups:      h.groups,
		mu:          h.mu, // mutex shared among all clones
		w:           h.w,
		opts:        h.opts,
	}
}

// Enabled reports whether the handler handles records at the given level.
func (h *Handler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.opts.Level.Level()
}

// Handle writes the Record to the handler's output.
func (h *Handler) Handle(_ context.Context, r slog.Record) error {
	buf := newBuffer()
	defer buf.Free()

	rep := h.opts.ReplaceAttr

	// Write time
	if !r.Time.IsZero() {
		val := r.Time.Round(0) // strip monotonic to match Attr behavior
		if rep == nil {
			h.appendTime(buf, r.Time, color.NoColor)
			buf.WriteByte(' ')
		} else if a := rep(nil, slog.Time(slog.TimeKey, val)); a.Key != "" {
			val, c := h.resolve(a.Value)
			if val.Kind() == slog.KindTime {
				h.appendTime(buf, val.Time(), c)
			} else {
				h.appendTintedValue(buf, val, false, c, true)
			}
			buf.WriteByte(' ')
		}
	}

	// Write level
	if rep == nil {
		h.appendLevel(buf, r.Level, color.NoColor)
		buf.WriteByte(' ')
	} else if a := rep(nil, slog.Any(slog.LevelKey, r.Level)); a.Key != "" {
		val, c := h.resolve(a.Value)
		if val.Kind() == slog.KindAny {
			if lvlVal, ok := val.Any().(slog.Level); ok {
				h.appendLevel(buf, lvlVal, c)
			} else {
				h.appendTintedValue(buf, val, false, c, false)
			}
		} else {
			h.appendTintedValue(buf, val, false, c, false)
		}
		buf.WriteByte(' ')
	}

	// Write source
	if h.opts.AddSource {
		fs := runtime.CallersFrames([]uintptr{r.PC})
		f, _ := fs.Next()
		if f.File != "" {
			src := &slog.Source{
				Function: f.Function,
				File:     f.File,
				Line:     f.Line,
			}

			if rep == nil {
				if h.opts.NoColor {
					h.appendSource(buf, src)
				} else {
					buf.WriteString(color.Dim)
					h.appendSource(buf, src)
					buf.WriteString(color.Reset)
				}
				buf.WriteByte(' ')
			} else if a := rep(nil, slog.Any(slog.SourceKey, src)); a.Key != "" {
				val, c := h.resolve(a.Value)
				h.appendTintedValue(buf, val, false, c, true)
				buf.WriteByte(' ')
			}
		}
	}

	// Write message
	if rep == nil {
		buf.WriteString(r.Message)
		buf.WriteByte(' ')
	} else if a := rep(nil, slog.String(slog.MessageKey, r.Message)); a.Key != "" {
		val, c := h.resolve(a.Value)
		h.appendTintedValue(buf, val, false, c, false)
		buf.WriteByte(' ')
	}

	// Write handler attributes
	if len(h.attrsPrefix) > 0 {
		buf.WriteString(h.attrsPrefix)
	}

	// Write record attributes
	r.Attrs(func(attr slog.Attr) bool {
		h.appendAttr(buf, attr, h.groupPrefix, h.groups)
		return true
	})

	if len(*buf) == 0 {
		buf.WriteByte('\n')
	} else {
		(*buf)[len(*buf)-1] = '\n' // replace last space with newline
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	_, err := h.w.Write(*buf)
	return err
}

// WithAttrs returns a new Handler with the given attributes added.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	h2 := h.clone()

	buf := newBuffer()
	defer buf.Free()

	for _, attr := range attrs {
		h.appendAttr(buf, attr, h.groupPrefix, h.groups)
	}
	h2.attrsPrefix = h.attrsPrefix + string(*buf)
	return h2
}

// WithGroup returns a new Handler with the given group name.
func (h *Handler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	h2 := h.clone()
	h2.groupPrefix += name + "."
	h2.groups = append(h2.groups, name)
	return h2
}

func (h *Handler) appendTime(buf *buffer, t time.Time, c color.Color) {
	if h.opts.NoColor {
		*buf = t.AppendFormat(*buf, h.opts.TimeFormat)
	} else {
		if c != color.NoColor {
			buf.WriteString(c.ForegroundSeqDim())
		} else {
			buf.WriteString(color.Dim)
		}
		*buf = t.AppendFormat(*buf, h.opts.TimeFormat)
		buf.WriteString(color.Reset)
	}
}

func (h *Handler) appendLevel(buf *buffer, level slog.Level, colorOverride color.Color) {
	formatLevel := func(base string, offset slog.Level) []byte {
		if offset == 0 {
			return []byte(base)
		} else if offset > 0 {
			return strconv.AppendInt(append([]byte(base), '+'), int64(offset), 10)
		}
		return strconv.AppendInt([]byte(base), int64(offset), 10)
	}

	if !h.opts.NoColor {
		if colorOverride != color.NoColor {
			buf.WriteString(colorOverride.ForegroundSeq())
		} else {
			switch {
			case level < slog.LevelInfo:
				// Debug: no color
			case level < slog.LevelWarn:
				buf.WriteString(color.BrightGreen.ForegroundSeq())
			case level < slog.LevelError:
				buf.WriteString(color.BrightYellow.ForegroundSeq())
			default:
				buf.WriteString(color.BrightRed.ForegroundSeq())
			}
		}
	}

	switch {
	case level < slog.LevelInfo:
		buf.Write(formatLevel("DBG", level-slog.LevelDebug))
	case level < slog.LevelWarn:
		buf.Write(formatLevel("INF", level-slog.LevelInfo))
	case level < slog.LevelError:
		buf.Write(formatLevel("WRN", level-slog.LevelWarn))
	default:
		buf.Write(formatLevel("ERR", level-slog.LevelError))
	}

	if !h.opts.NoColor && (colorOverride != color.NoColor || level >= slog.LevelInfo) {
		buf.WriteString(color.Reset)
	}
}

func (h *Handler) appendSource(buf *buffer, src *slog.Source) {
	dir, file := filepath.Split(src.File)
	buf.WriteString(filepath.Join(filepath.Base(dir), file))
	buf.WriteByte(':')
	*buf = strconv.AppendInt(*buf, int64(src.Line), 10)
}

func (h *Handler) resolve(val slog.Value) (slog.Value, color.Color) {
	if !h.opts.NoColor && val.Kind() == slog.KindLogValuer {
		if cv, ok := val.Any().(coloredValue); ok {
			return cv.Value.Resolve(), cv.Color
		}
	}
	return val.Resolve(), color.NoColor
}

func (h *Handler) appendAttr(buf *buffer, attr slog.Attr, groupsPrefix string, groups []string) {
	c := color.NoColor
	attr.Value, c = h.resolve(attr.Value)

	if rep := h.opts.ReplaceAttr; rep != nil && attr.Value.Kind() != slog.KindGroup {
		attr = rep(groups, attr)
		var colorRep color.Color
		attr.Value, colorRep = h.resolve(attr.Value)
		if colorRep != color.NoColor {
			c = colorRep
		}
	}

	if attr.Equal(slog.Attr{}) {
		return
	}

	if attr.Value.Kind() == slog.KindGroup {
		if attr.Key != "" {
			groupsPrefix += attr.Key + "."
			groups = append(groups, attr.Key)
		}
		for _, groupAttr := range attr.Value.Group() {
			h.appendAttr(buf, groupAttr, groupsPrefix, groups)
		}
		return
	}

	if h.opts.NoColor {
		h.appendKey(buf, attr.Key, groupsPrefix)
		h.appendValue(buf, attr.Value, true)
	} else {
		if c != color.NoColor {
			buf.WriteString(c.ForegroundSeqDim())
			h.appendKey(buf, attr.Key, groupsPrefix)
			buf.WriteString("\033[22m") // reset faint only
			h.appendValue(buf, attr.Value, true)
			buf.WriteString(color.Reset)
		} else {
			buf.WriteString(color.Dim)
			h.appendKey(buf, attr.Key, groupsPrefix)
			buf.WriteString(color.Reset)
			h.appendValue(buf, attr.Value, true)
		}
	}
	buf.WriteByte(' ')
}

func (h *Handler) appendKey(buf *buffer, key, groups string) {
	appendString(buf, groups+key, true, !h.opts.NoColor)
	buf.WriteByte('=')
}

func (h *Handler) appendValue(buf *buffer, v slog.Value, quote bool) {
	switch v.Kind() {
	case slog.KindString:
		appendString(buf, v.String(), quote, !h.opts.NoColor)
	case slog.KindInt64:
		*buf = strconv.AppendInt(*buf, v.Int64(), 10)
	case slog.KindUint64:
		*buf = strconv.AppendUint(*buf, v.Uint64(), 10)
	case slog.KindFloat64:
		*buf = strconv.AppendFloat(*buf, v.Float64(), 'g', -1, 64)
	case slog.KindBool:
		*buf = strconv.AppendBool(*buf, v.Bool())
	case slog.KindDuration:
		appendString(buf, v.Duration().String(), quote, !h.opts.NoColor)
	case slog.KindTime:
		*buf = appendRFC3339Millis(*buf, v.Time())
	case slog.KindAny:
		h.appendAnyValue(buf, v, quote)
	}
}

func (h *Handler) appendAnyValue(buf *buffer, v slog.Value, quote bool) {
	defer func() {
		if r := recover(); r != nil {
			if rv := reflect.ValueOf(v.Any()); rv.Kind() == reflect.Pointer && rv.IsNil() {
				buf.WriteString("<nil>")
				return
			}
			appendString(buf, fmt.Sprintf("!PANIC: %v", r), true, !h.opts.NoColor)
		}
	}()

	switch cv := v.Any().(type) {
	case encoding.TextMarshaler:
		data, err := cv.MarshalText()
		if err != nil {
			break
		}
		appendString(buf, string(data), quote, !h.opts.NoColor)
	case *slog.Source:
		h.appendSource(buf, cv)
	default:
		appendString(buf, fmt.Sprintf("%+v", cv), quote, !h.opts.NoColor)
	}
}

func (h *Handler) appendTintedValue(buf *buffer, val slog.Value, quote bool, c color.Color, faint bool) {
	if h.opts.NoColor {
		h.appendValue(buf, val, quote)
	} else {
		if c != color.NoColor {
			if faint {
				buf.WriteString(c.ForegroundSeqDim())
			} else {
				buf.WriteString(c.ForegroundSeq())
			}
		} else if faint {
			buf.WriteString(color.Dim)
		}
		h.appendValue(buf, val, quote)
		if c != color.NoColor || faint {
			buf.WriteString(color.Reset)
		}
	}
}

func appendRFC3339Millis(b []byte, t time.Time) []byte {
	const prefixLen = len("2006-01-02T15:04:05.000")
	n := len(b)
	t = t.Truncate(time.Millisecond).Add(time.Millisecond / 10)
	b = t.AppendFormat(b, time.RFC3339Nano)
	b = append(b[:n+prefixLen], b[n+prefixLen+1:]...)
	return b
}

func appendString(buf *buffer, s string, quote, preserveColor bool) {
	if quote && !preserveColor {
		// Trim ANSI escape sequences when not preserving color
		var inEscape bool
		s = cutRunes(s, func(r rune) bool {
			if r == ansiEsc {
				inEscape = true
			} else if inEscape && unicode.IsLetter(r) {
				inEscape = false
				return true
			}
			return inEscape
		})
	}

	quote = quote && needsQuoting(s)
	switch {
	case preserveColor && quote:
		s = strconv.Quote(s)
		s = strings.ReplaceAll(s, `\x1b`, string(ansiEsc))
		buf.WriteString(s)
	case !preserveColor && quote:
		*buf = strconv.AppendQuote(*buf, s)
	default:
		buf.WriteString(s)
	}
}

func cutRunes(s string, f func(r rune) bool) string {
	var res []rune
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError {
			break
		}
		if !f(r) {
			res = append(res, r)
		}
		i += size
	}
	return string(res)
}

func needsQuoting(s string) bool {
	if len(s) == 0 {
		return true
	}
	for i := 0; i < len(s); {
		b := s[i]
		if b < utf8.RuneSelf {
			if b != '\\' && (b == ' ' || b == '=' || !safeSet[b]) {
				return true
			}
			i++
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError || unicode.IsSpace(r) || !unicode.IsPrint(r) {
			return true
		}
		i += size
	}
	return false
}

// safeSet defines characters that don't need quoting (extended with ANSI escape)
var safeSet = [utf8.RuneSelf]bool{
	' ':      true,
	'!':      true,
	'"':      false,
	'#':      true,
	'$':      true,
	'%':      true,
	'&':      true,
	'\'':     true,
	'(':      true,
	')':      true,
	'*':      true,
	'+':      true,
	',':      true,
	'-':      true,
	'.':      true,
	'/':      true,
	'0':      true,
	'1':      true,
	'2':      true,
	'3':      true,
	'4':      true,
	'5':      true,
	'6':      true,
	'7':      true,
	'8':      true,
	'9':      true,
	':':      true,
	';':      true,
	'<':      true,
	'=':      true,
	'>':      true,
	'?':      true,
	'@':      true,
	'A':      true,
	'B':      true,
	'C':      true,
	'D':      true,
	'E':      true,
	'F':      true,
	'G':      true,
	'H':      true,
	'I':      true,
	'J':      true,
	'K':      true,
	'L':      true,
	'M':      true,
	'N':      true,
	'O':      true,
	'P':      true,
	'Q':      true,
	'R':      true,
	'S':      true,
	'T':      true,
	'U':      true,
	'V':      true,
	'W':      true,
	'X':      true,
	'Y':      true,
	'Z':      true,
	'[':      true,
	'\\':     false,
	']':      true,
	'^':      true,
	'_':      true,
	'`':      true,
	'a':      true,
	'b':      true,
	'c':      true,
	'd':      true,
	'e':      true,
	'f':      true,
	'g':      true,
	'h':      true,
	'i':      true,
	'j':      true,
	'k':      true,
	'l':      true,
	'm':      true,
	'n':      true,
	'o':      true,
	'p':      true,
	'q':      true,
	'r':      true,
	's':      true,
	't':      true,
	'u':      true,
	'v':      true,
	'w':      true,
	'x':      true,
	'y':      true,
	'z':      true,
	'{':      true,
	'|':      true,
	'}':      true,
	'~':      true,
	'\u007f': true,
	'\u001b': true,
}
