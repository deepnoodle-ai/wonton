// Command termprobe reports the host terminal's cell width for a battery of
// Unicode clusters. It puts the terminal into raw mode, writes one cluster
// at a time, queries the cursor position with DSR (\x1b[6n), and compares
// the reported column against wonton's runewidth.StringWidth.
//
// Use this when the reflow or TUI examples show overlapping or clipped
// glyphs: if the probe reports a disagreement between runewidth and the
// host, the host's font or cell math is the culprit, not the rendering
// pipeline.
//
// Typical usage:
//
//	go run ./examples/termprobe
//
// Example output lines:
//
//	OK    VS16 heart          ❤️        wonton=2  terminal=2
//	DIFF  keycap #            #️⃣        wonton=2  terminal=1   ← terminal ignored VS16+KEYCAP
//	OK    JP flag             🇯🇵       wonton=2  terminal=2
package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/deepnoodle-ai/wonton/runewidth"
	"golang.org/x/term"
)

type probeCase struct {
	name    string
	cluster string
}

var cases = []probeCase{
	{"ASCII A", "A"},
	{"CJK 中", "\u4E2D"},
	{"VS16 heart", "\u2764\uFE0F"},
	{"VS16 warning", "\u26A0\uFE0F"},
	{"VS16 sun", "\u2600\uFE0F"},
	{"text heart (no VS16)", "\u2764"},
	{"hash keycap", "#\uFE0F\u20E3"},
	{"star keycap", "*\uFE0F\u20E3"},
	{"0 keycap", "0\uFE0F\u20E3"},
	{"waving hand", "\U0001F44B"},
	{"waving hand + skin", "\U0001F44B\U0001F3FD"},
	{"JP flag", "\U0001F1EF\U0001F1F5"},
	{"US flag", "\U0001F1FA\U0001F1F8"},
	{"rainbow flag", "\U0001F3F3\uFE0F\u200D\U0001F308"},
	{"family of four", "\U0001F468\u200D\U0001F469\u200D\U0001F467\u200D\U0001F466"},
	{"two-em dash", "\u2E3A"},
	{"three-em dash", "\u2E3B"},
}

func main() {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		fmt.Fprintln(os.Stderr, "termprobe: stdin is not a terminal")
		os.Exit(2)
	}

	oldState, err := term.MakeRaw(fd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "termprobe: raw mode: %v\r\n", err)
		os.Exit(1)
	}
	defer term.Restore(fd, oldState)

	out := os.Stdout
	reader := bufio.NewReader(os.Stdin)

	// Headline.
	fmt.Fprint(out, "\r\n")
	fmt.Fprint(out, "termprobe — measures this terminal's reported width per cluster.\r\n")
	fmt.Fprint(out, "A DIFF line means the host terminal disagrees with wonton/runewidth.\r\n")
	fmt.Fprint(out, "\r\n")

	type result struct {
		name     string
		cluster  string
		wonton   int
		reported int
		err      error
	}
	results := make([]result, 0, len(cases))

	for _, c := range cases {
		got, err := measure(out, reader, c.cluster)
		results = append(results, result{
			name:     c.name,
			cluster:  c.cluster,
			wonton:   runewidth.StringWidth(c.cluster),
			reported: got,
			err:      err,
		})
	}

	// Report.
	diffs := 0
	for _, r := range results {
		label := "OK   "
		if r.err != nil {
			label = "ERR  "
		} else if r.reported != r.wonton {
			label = "DIFF "
			diffs++
		}
		fmt.Fprintf(out, "%s %-22s %-8s wonton=%d  terminal=%d",
			label, r.name, r.cluster, r.wonton, r.reported)
		if r.err != nil {
			fmt.Fprintf(out, "  err=%v", r.err)
		}
		fmt.Fprint(out, "\r\n")
	}
	fmt.Fprintf(out, "\r\n%d / %d clusters disagree with runewidth.StringWidth.\r\n",
		diffs, len(results))
	if diffs > 0 {
		fmt.Fprint(out, "Overlapping or mis-aligned glyphs in wonton TUI examples on\r\n")
		fmt.Fprint(out, "this terminal are likely due to the font/terminal, not the\r\n")
		fmt.Fprint(out, "rendering pipeline.\r\n")
	}
}

// measure writes one cluster preceded by a CR (column 0), queries cursor
// position, reads the response, and returns the 0-based column the terminal
// reports. Any stray bytes read while the response is pending are dropped.
func measure(out *os.File, r *bufio.Reader, cluster string) (int, error) {
	// Return to column 0, clear the line, write the cluster, then DSR.
	fmt.Fprintf(out, "\r\x1b[2K%s\x1b[6n", cluster)

	col, err := readDSR(r, 300*time.Millisecond)
	// Always clear the probe line before returning so the report reads cleanly.
	fmt.Fprint(out, "\r\x1b[2K")
	if err != nil {
		return 0, err
	}
	// DSR reports 1-based columns; convert to 0-based so it matches StringWidth.
	return col - 1, nil
}

// readDSR reads bytes from r until it sees a CSI cursor-position report
// (\x1b[row;colR) and returns the column. Returns an error on timeout or
// malformed response.
func readDSR(r *bufio.Reader, timeout time.Duration) (int, error) {
	os.Stdin.SetReadDeadline(time.Now().Add(timeout))
	defer os.Stdin.SetReadDeadline(time.Time{})

	var buf []byte
	for {
		b, err := r.ReadByte()
		if err != nil {
			return 0, err
		}
		buf = append(buf, b)
		if b == 'R' {
			break
		}
		if len(buf) > 64 {
			return 0, fmt.Errorf("DSR response too long: %q", buf)
		}
	}
	// Find the CSI start and parse "\x1b[row;colR".
	i := strings.LastIndex(string(buf), "\x1b[")
	if i < 0 {
		return 0, fmt.Errorf("no CSI in DSR response: %q", buf)
	}
	body := string(buf[i+2 : len(buf)-1])
	_, colStr, ok := strings.Cut(body, ";")
	if !ok {
		return 0, fmt.Errorf("no ';' in DSR body: %q", body)
	}
	col, err := strconv.Atoi(colStr)
	if err != nil {
		return 0, fmt.Errorf("parse col from DSR body %q: %w", body, err)
	}
	return col, nil
}
