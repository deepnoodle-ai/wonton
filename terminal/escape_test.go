package terminal

import (
	"strings"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

// These tests cover untrusted text and URLs injecting terminal escape
// sequences, e.g. a URL that ends the OSC 8 sequence early and follows it
// with its own control sequences.

const injectedTitle = "\033]0;pwned\007"

func TestStart_StripsControlCharacters(t *testing.T) {
	got := Start("https://a.example/\033\\" + injectedTitle)
	assert.Equal(t, "\033]8;;https://a.example/\\]0;pwned\033\\", got)
}

func TestStart_StripsC1Controls(t *testing.T) {
	// U+009C is the C1 string terminator; a raw 0x9c byte is the 8-bit form.
	got := Start("https://a.example/\u009c\u009b2J\x9c")
	assert.Equal(t, "\033]8;;https://a.example/2J�\033\\", got)
}

func TestStartWithID_SanitizesID(t *testing.T) {
	// A ';' in the id would end the parameters and change the link target.
	got := StartWithID("https://good.example", "x;https://evil.example")
	assert.Equal(t, "\033]8;id=xhttps//evil.example;https://good.example\033\\", got)

	got = StartWithID("https://good.example", "a\033\\"+injectedTitle)
	assert.Equal(t, "\033]8;id=a\\]0pwned;https://good.example\033\\", got)
}

func TestFormatWithID_URLCannotBreakOut(t *testing.T) {
	got := FormatWithID("https://a.example\033\\"+injectedTitle, "text", "id1")
	assert.False(t, strings.Contains(got, injectedTitle))
}

func TestFallback_StripsControlCharactersFromURL(t *testing.T) {
	assert.Equal(t, "text (https://a.example]0;pwned)", Fallback("https://a.example"+injectedTitle, "text"))
}

func TestValidateURL_RejectsC1Controls(t *testing.T) {
	assert.Error(t, ValidateURL("https://a.example/\u009c"))
	assert.Error(t, ValidateURL("https://a.example/\x9c"))
	assert.Error(t, ValidateAbsoluteURL("https://a.example/\u009b2J"))
}

func TestStripOSC8_BELTerminator(t *testing.T) {
	assert.Equal(t, "Click", StripOSC8("\033]8;;https://example.com\007Click\033]8;;\007"))
	// A BEL-terminated link before an ESC-terminated one must not swallow
	// the text between them.
	assert.Equal(t, "a b", StripOSC8("\033]8;;https://x\007a\033]8;;\007 \033]8;;https://y\033\\b\033]8;;\033\\"))
}

func renderFrame(t *testing.T, draw func(f RenderFrame)) string {
	t.Helper()
	var out strings.Builder
	term := NewTestTerminal(40, 3, &out)
	frame, err := term.BeginFrame()
	assert.NoError(t, err)
	draw(frame)
	assert.NoError(t, term.EndFrame(frame))
	return out.String()
}

func TestFlush_StyleURLCannotBreakOut(t *testing.T) {
	out := renderFrame(t, func(f RenderFrame) {
		_ = f.PrintStyled(0, 0, "link", NewStyle().WithURL("https://a.example\033\\"+injectedTitle))
	})
	assert.False(t, strings.Contains(out, injectedTitle))
	assert.False(t, strings.Contains(out, "\007"))
}

func TestFlush_TextControlCharactersDropped(t *testing.T) {
	// A zero-width control character left at the end of a line stays in its
	// cell, so two adjacent prints can assemble a complete escape sequence.
	out := renderFrame(t, func(f RenderFrame) {
		_ = f.PrintStyled(0, 0, "a\033\n", NewStyle())
		_ = f.PrintStyled(1, 0, "]0;pwned\007\n", NewStyle())
	})
	assert.False(t, strings.Contains(out, injectedTitle))
	assert.False(t, strings.Contains(out, "\007"))
}

func TestPrint_UnbufferedStripsControlCharacters(t *testing.T) {
	var out strings.Builder
	term := NewTestTerminal(40, 3, &out)
	term.buffered = false
	term.Print("hi" + injectedTitle + "\033[2J\tthere\n")
	got := out.String()
	assert.False(t, strings.Contains(got, injectedTitle))
	assert.False(t, strings.Contains(got, "\033[2J"))
	assert.True(t, strings.Contains(got, "hi]0;pwned[2J\tthere\n"))
}

func TestPrintHyperlink_InvalidURLDropsStyleURL(t *testing.T) {
	link := NewHyperlink("https://ok.example\x7f", "text")
	link.Style = link.Style.WithURL("https://style.example\033\\" + injectedTitle)
	out := renderFrame(t, func(f RenderFrame) {
		_ = f.PrintHyperlink(0, 0, link)
	})
	assert.False(t, strings.Contains(out, "style.example"))
}
