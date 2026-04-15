package runewidth

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
)

// TestGraphemeBreakConformance runs the official Unicode GraphemeBreakTest at
// the version matching [UnicodeVersion] and asserts 100% pass.
//
// The test file lives under testdata/GraphemeBreakTest-<version>.txt. Each
// non-comment line has the form:
//
//	÷ HEX × HEX ÷ HEX ÷	# comment
//
// where "÷" marks a cluster boundary (break) and "×" marks no boundary.
func TestGraphemeBreakConformance(t *testing.T) {
	path := fmt.Sprintf("testdata/GraphemeBreakTest-%s.txt", UnicodeVersion)
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open test file: %v", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	// Some test lines are long (hundreds of bytes of comment).
	scanner.Buffer(make([]byte, 1<<20), 1<<20)

	total, passed, failed := 0, 0, 0
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := scanner.Text()
		if idx := strings.Index(line, "#"); idx >= 0 {
			line = line[:idx]
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		runes, expectedBreaks, err := parseConformanceLine(line)
		if err != nil {
			t.Errorf("line %d: %v", lineNo, err)
			continue
		}
		total++

		// Build the input string and the list of byte offsets at which we
		// expect a boundary (excluding sot/eot).
		var sb strings.Builder
		runeByteStart := make([]int, len(runes))
		for i, r := range runes {
			runeByteStart[i] = sb.Len()
			sb.WriteRune(r)
		}
		s := sb.String()

		var wantBreakOffsets []int
		for i, b := range expectedBreaks {
			if i == 0 || i == len(expectedBreaks)-1 {
				// Skip sot and eot markers — they're always "÷" in the spec but
				// our iterator only reports internal boundaries.
				continue
			}
			if b {
				wantBreakOffsets = append(wantBreakOffsets, runeByteStart[i])
			}
		}

		var gotBreakOffsets []int
		offset := 0
		first := true
		for cluster, _ := range Graphemes(s) {
			if !first {
				gotBreakOffsets = append(gotBreakOffsets, offset)
			}
			first = false
			offset += len(cluster)
		}

		if !intSliceEqual(gotBreakOffsets, wantBreakOffsets) {
			failed++
			if failed <= 10 {
				t.Errorf("line %d %q: got breaks %v, want %v", lineNo, line, gotBreakOffsets, wantBreakOffsets)
			}
			continue
		}
		passed++
	}

	if err := scanner.Err(); err != nil {
		t.Fatalf("scan: %v", err)
	}

	if failed > 0 {
		t.Errorf("GraphemeBreakTest: %d/%d cases passed, %d failed", passed, total, failed)
	} else {
		t.Logf("GraphemeBreakTest: %d/%d cases passed", passed, total)
	}
}

// parseConformanceLine parses a line like "÷ 000D × 000A ÷" into a slice of
// runes and a parallel slice of "is there a break before this position"
// booleans of length len(runes)+1. The first and last booleans correspond to
// sot and eot; entries 1..len(runes)-1 correspond to boundaries between runes.
func parseConformanceLine(line string) ([]rune, []bool, error) {
	tokens := strings.Fields(line)
	var runes []rune
	var breaks []bool
	for _, tok := range tokens {
		switch tok {
		case "÷":
			breaks = append(breaks, true)
		case "×":
			breaks = append(breaks, false)
		default:
			n, err := strconv.ParseUint(tok, 16, 32)
			if err != nil {
				return nil, nil, fmt.Errorf("bad hex %q: %v", tok, err)
			}
			runes = append(runes, rune(n))
		}
	}
	if len(breaks) != len(runes)+1 {
		return nil, nil, fmt.Errorf("malformed: %d runes, %d break markers", len(runes), len(breaks))
	}
	return runes, breaks, nil
}

func intSliceEqual(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
