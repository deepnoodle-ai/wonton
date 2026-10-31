package terminal

import (
	"testing"

	"github.com/deepnoodle-ai/wonton/termtest"
)

// Unbuffered output goes to a tty in raw mode, where a bare "\n" keeps the
// column. Each newline in printed text must reach the terminal as "\r\n".
func TestTerminal_Unbuffered_MultiLinePrintInRawMode(t *testing.T) {
	styles := map[string]Style{
		"plain":  NewStyle(),
		"styled": NewStyle().WithForeground(ColorRed),
	}
	for name, style := range styles {
		t.Run(name, func(t *testing.T) {
			term, out := newUnbufferedTestTerminal(20, 4)
			term.PrintAtStyled(2, 1, "ab\ncd\nef", style)

			screen := termtest.NewScreen(20, 4, termtest.WithRawMode(true))
			screen.Write(out.Bytes())
			termtest.AssertRow(t, screen, 1, "  ab")
			termtest.AssertRow(t, screen, 2, "cd")
			termtest.AssertRow(t, screen, 3, "ef")
		})
	}
}
