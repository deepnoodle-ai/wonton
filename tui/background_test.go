package tui

import (
	"os"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
	"golang.org/x/term"
)

func TestParseBackgroundReply(t *testing.T) {
	tests := []struct {
		name  string
		reply string
		want  RGB
		ok    bool
	}{
		{"4 digits BEL", "\x1b]11;rgb:1c1c/1c1c/1c1c\x07", RGB{R: 28, G: 28, B: 28}, true},
		{"4 digits ST", "\x1b]11;rgb:ffff/ffff/ffff\x1b\\", RGB{R: 255, G: 255, B: 255}, true},
		{"2 digits", "\x1b]11;rgb:fd/f6/e3\x07", RGB{R: 253, G: 246, B: 227}, true},
		{"1 digit", "\x1b]11;rgb:f/0/8\x07", RGB{R: 255, G: 0, B: 136}, true},
		{"3 digits", "\x1b]11;rgb:fff/000/800\x07", RGB{R: 255, G: 0, B: 128}, true},
		{"upper case", "\x1b]11;rgb:FFFF/8080/0000\x07", RGB{R: 255, G: 128, B: 0}, true},
		{"rgba", "\x1b]11;rgba:0000/2b2b/3636/ffff\x1b\\", RGB{R: 0, G: 43, B: 54}, true},
		{"C1 ST", "\x1b]11;rgb:0000/0000/0000\x9c", RGB{}, true},
		{"with DA1 and noise", "x\x1b]11;rgb:2020/2020/2020\x07\x1b[?62;22c", RGB{R: 32, G: 32, B: 32}, true},
		{"bad reply then good", "\x1b]11;garbage\x07\x1b]11;rgb:ff/ff/ff\x07", RGB{R: 255, G: 255, B: 255}, true},
		{"unterminated", "\x1b]11;rgb:ffff/ffff/ffff", RGB{}, false},
		{"ESC that is not ST", "\x1b]11;rgb:ffff/ffff/ffff\x1b[c", RGB{}, false},
		{"empty", "", RGB{}, false},
		{"DA1 only", "\x1b[?62;22c", RGB{}, false},
		{"other OSC", "\x1b]10;rgb:ffff/ffff/ffff\x07", RGB{}, false},
		{"two components", "\x1b]11;rgb:ffff/ffff\x07", RGB{}, false},
		{"five digits", "\x1b]11;rgb:fffff/ffff/ffff\x07", RGB{}, false},
		{"empty component", "\x1b]11;rgb://ffff/ffff\x07", RGB{}, false},
		{"not hex", "\x1b]11;rgb:zzzz/ffff/ffff\x07", RGB{}, false},
		{"sign", "\x1b]11;rgb:+fff/ffff/ffff\x07", RGB{}, false},
		{"hash form", "\x1b]11;#ffffff\x07", RGB{}, false},
		{"query echoed", "\x1b]11;?\x07", RGB{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseBackgroundReply([]byte(tt.reply))
			assert.Equal(t, ok, tt.ok)
			assert.Equal(t, got, tt.want)
		})
	}
}

func TestBackgroundOf(t *testing.T) {
	assert.Equal(t, backgroundOf(RGB{}), TerminalBackgroundDark)
	assert.Equal(t, backgroundOf(RGB{R: 0, G: 43, B: 54}), TerminalBackgroundDark)  // Solarized dark
	assert.Equal(t, backgroundOf(RGB{R: 40, G: 42, B: 54}), TerminalBackgroundDark) // Dracula
	assert.Equal(t, backgroundOf(RGB{R: 253, G: 246, B: 227}), TerminalBackgroundLight)
	assert.Equal(t, backgroundOf(RGB{R: 255, G: 255, B: 255}), TerminalBackgroundLight)
}

func TestColorFGBGBackground(t *testing.T) {
	tests := map[string]TerminalBackground{
		"":             TerminalBackgroundUnknown,
		"15;0":         TerminalBackgroundDark,
		"0;15":         TerminalBackgroundLight,
		"7;8":          TerminalBackgroundDark,
		"0;7":          TerminalBackgroundLight,
		"15;default;0": TerminalBackgroundDark,
		"0;default;15": TerminalBackgroundLight,
		"15;default":   TerminalBackgroundUnknown,
		"15;16":        TerminalBackgroundUnknown,
		"15;-1":        TerminalBackgroundUnknown,
		"junk":         TerminalBackgroundUnknown,
	}
	for value, want := range tests {
		assert.Equal(t, colorFGBGBackground(value), want, "COLORFGBG=%q", value)
	}
}

func TestTerminalBackgroundString(t *testing.T) {
	assert.Equal(t, TerminalBackgroundUnknown.String(), "unknown")
	assert.Equal(t, TerminalBackgroundDark.String(), "dark")
	assert.Equal(t, TerminalBackgroundLight.String(), "light")
}

func TestSkipBackgroundQuery(t *testing.T) {
	assert.True(t, skipBackgroundQuery(""))
	assert.True(t, skipBackgroundQuery("dumb"))
	assert.False(t, skipBackgroundQuery("xterm-256color"))
}

func TestDetectTerminalBackgroundWithoutTerminal(t *testing.T) {
	// Under go test, standard output is not a terminal, so detection must
	// return at once without writing a query.
	if term.IsTerminal(int(os.Stdout.Fd())) {
		t.Skip("standard output is a terminal")
	}
	got, err := DetectTerminalBackground(0)
	assert.NoError(t, err)
	assert.Equal(t, got, TerminalBackgroundUnknown)
}
