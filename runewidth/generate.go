//go:build ignore

// This program generates tables.go from Unicode data files.
//
// Usage:
//
//	go run generate.go                  # uses default Unicode version
//	go run generate.go -version=18.0.0  # override version
//
// Downloaded source files are cached under testdata/unicode/<version>/ so
// subsequent runs are offline-friendly. The cache directory is gitignored.
package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const defaultUnicodeVersion = "17.0.0"

const (
	graphemeBreakPropertyFile = "auxiliary/GraphemeBreakProperty.txt"
	emojiDataFile             = "emoji/emoji-data.txt"
	eastAsianWidthFile        = "EastAsianWidth.txt"
	derivedCorePropertiesFile = "DerivedCoreProperties.txt"
)

type interval struct {
	lo, hi rune
}

type graphemeInterval struct {
	lo, hi rune
	prop   string // symbolic name, e.g. "gbExtend"
}

type incbIntervalGen struct {
	lo, hi rune
	prop   string // symbolic name, e.g. "incbLinker"
}

// UAX#29 Grapheme_Cluster_Break property names → symbolic constants.
var gbPropertyName = map[string]string{
	"Prepend":            "gbPrepend",
	"CR":                 "gbCR",
	"LF":                 "gbLF",
	"Control":            "gbControl",
	"Extend":             "gbExtend",
	"Regional_Indicator": "gbRegionalIndicator",
	"SpacingMark":        "gbSpacingMark",
	"L":                  "gbL",
	"V":                  "gbV",
	"T":                  "gbT",
	"LV":                 "gbLV",
	"LVT":                "gbLVT",
	"ZWJ":                "gbZWJ",
}

// Stable order for emitting the constants (must start at gbOther = 0).
var gbPropertyOrder = []string{
	"gbOther",
	"gbPrepend",
	"gbCR",
	"gbLF",
	"gbControl",
	"gbExtend",
	"gbRegionalIndicator",
	"gbSpacingMark",
	"gbL",
	"gbV",
	"gbT",
	"gbLV",
	"gbLVT",
	"gbZWJ",
	"gbExtendedPictographic",
}

// Indic_Conjunct_Break property values. Used by rule GB9c.
var incbPropertyName = map[string]string{
	"Consonant": "incbConsonant",
	"Linker":    "incbLinker",
	"Extend":    "incbExtend",
}

var incbPropertyOrder = []string{
	"incbNone",
	"incbConsonant",
	"incbLinker",
	"incbExtend",
}

func main() {
	version := flag.String("version", defaultUnicodeVersion, "Unicode version to generate tables for")
	flag.Parse()

	fmt.Printf("Generating Unicode %s tables...\n", *version)

	gbpData := mustFetch(*version, graphemeBreakPropertyFile)
	emojiData := mustFetch(*version, emojiDataFile)
	eawData := mustFetch(*version, eastAsianWidthFile)
	dcpData := mustFetch(*version, derivedCorePropertiesFile)

	// Parse raw property files.
	gbpRaw := parsePropertyFile(gbpData)
	emojiRaw := parsePropertyFile(emojiData)
	eawRaw := parsePropertyFile(eawData)
	incbRaw := parseNamedPropertyFile(dcpData, "InCB")

	// Build the grapheme break property table.
	// Start with values from GraphemeBreakProperty.txt. Then layer
	// Extended_Pictographic over any code point that does not already have a
	// grapheme break property assigned. This matches Unicode's
	// GraphemeBreakProperty.txt semantics: Extended_Pictographic is applied as
	// a separate rule class in GB11, and characters that are both "Control" and
	// "Extended_Pictographic" should still be Control for break purposes.
	assigned := make(map[rune]string)
	for propName, sym := range gbPropertyName {
		for _, iv := range gbpRaw[propName] {
			for r := iv.lo; r <= iv.hi; r++ {
				assigned[r] = sym
			}
		}
	}
	for _, iv := range emojiRaw["Extended_Pictographic"] {
		for r := iv.lo; r <= iv.hi; r++ {
			if _, ok := assigned[r]; !ok {
				assigned[r] = "gbExtendedPictographic"
			}
		}
	}

	graphemeBreakProperty := buildGraphemeTable(assigned)

	// Width tables.
	var doublewidth []interval
	doublewidth = append(doublewidth, eawRaw["W"]...)
	doublewidth = append(doublewidth, eawRaw["F"]...)
	doublewidth = sortAndMerge(doublewidth)

	emojiPresentation := sortAndMerge(emojiRaw["Emoji_Presentation"])

	// Indic_Conjunct_Break table. Only Consonant and Linker are actually needed
	// for GB9c — Extend_ConjunctLinker (InCB=Extend) characters are also in
	// GraphemeBreakProperty=Extend and don't require a separate lookup to make
	// the rule work. We keep incbExtend in the property table anyway so that
	// the data matches the Unicode reference exactly.
	incbAssigned := make(map[rune]string)
	for propName, sym := range incbPropertyName {
		for _, iv := range incbRaw[propName] {
			for r := iv.lo; r <= iv.hi; r++ {
				incbAssigned[r] = sym
			}
		}
	}
	incbTable := buildIncbTable(incbAssigned)

	// Build the 2-stage BMP lookup table. For every rune 0..0xFFFF we pack
	// (gbProperty, isDoubleWidth, isEmojiPresentation) into one byte, then
	// split the 65536-byte flat table into 256 blocks of 256 bytes and
	// deduplicate identical blocks. The resulting stage1+stage2 typically
	// compresses to ~10KB because most BMP blocks are all-zero or highly
	// repetitive.
	//
	// Bit layout of each byte:
	//   bits 0..3 — gbProperty (15 values, fits in 4 bits)
	//   bit  4   — isDoubleWidth (EAW W or F)
	//   bit  5   — isEmojiPresentation
	//   bits 6..7 — reserved
	gbByName := map[string]byte{
		"gbOther":                0,
		"gbPrepend":              1,
		"gbCR":                   2,
		"gbLF":                   3,
		"gbControl":              4,
		"gbExtend":               5,
		"gbRegionalIndicator":    6,
		"gbSpacingMark":          7,
		"gbL":                    8,
		"gbV":                    9,
		"gbT":                    10,
		"gbLV":                   11,
		"gbLVT":                  12,
		"gbZWJ":                  13,
		"gbExtendedPictographic": 14,
	}
	doubleWidthSet := make(map[rune]bool)
	for _, iv := range doublewidth {
		for r := iv.lo; r <= iv.hi; r++ {
			doubleWidthSet[r] = true
		}
	}
	emojiPresSet := make(map[rune]bool)
	for _, iv := range emojiPresentation {
		for r := iv.lo; r <= iv.hi; r++ {
			emojiPresSet[r] = true
		}
	}

	// Cover both BMP (0..0xFFFF) and SMP (0x10000..0x1FFFF) with the 2-stage
	// table. This brings emoji and regional indicators into the O(1) fast
	// path. Higher planes (SIP, TIP, SSP) contain mostly unassigned or
	// plain-wide CJK characters and remain on the binary-search fallback.
	const flatCovered = 0x20000
	flat := make([]byte, flatCovered)
	for r := rune(0); r < flatCovered; r++ {
		var b byte
		if sym, ok := assigned[r]; ok {
			b = gbByName[sym]
		}
		if doubleWidthSet[r] {
			b |= 1 << 4
		}
		if emojiPresSet[r] {
			b |= 1 << 5
		}
		flat[r] = b
	}
	stage1, stage2 := compressBMPBlocks(flat)

	fmt.Printf("Table sizes: graphemeBreakProperty=%d, doublewidth=%d, emojiPresentation=%d, incb=%d, stage1=%d, stage2Blocks=%d (%d bytes)\n",
		len(graphemeBreakProperty), len(doublewidth), len(emojiPresentation), len(incbTable),
		len(stage1), len(stage2)/256, len(stage2))

	var buf bytes.Buffer
	fmt.Fprintf(&buf, `// Code generated by go run generate.go; DO NOT EDIT.
//
// Unicode version: %s
// Sources:
//   - %s
//   - %s
//   - %s
//   - %s
// Table sizes:
//   graphemeBreakProperty: %d intervals
//   doublewidth:           %d intervals
//   emojiPresentation:     %d intervals
//   incb:                  %d intervals
//   bmpStage2:             %d blocks (%d bytes)

package runewidth

// UnicodeVersion is the version of the Unicode Character Database from which
// the tables in this file were generated.
const UnicodeVersion = %q

`,
		*version,
		graphemeBreakPropertyFile, emojiDataFile, eastAsianWidthFile, derivedCorePropertiesFile,
		len(graphemeBreakProperty), len(doublewidth), len(emojiPresentation), len(incbTable),
		len(stage2)/256, len(stage2),
		*version,
	)

	// Emit gbProperty constants.
	fmt.Fprintf(&buf, "// Grapheme_Cluster_Break property values from UAX#29 plus\n")
	fmt.Fprintf(&buf, "// Extended_Pictographic from UTS#51. gbOther is the default (no property assigned).\n")
	fmt.Fprintf(&buf, "const (\n")
	for i, name := range gbPropertyOrder {
		if i == 0 {
			fmt.Fprintf(&buf, "\t%s uint8 = iota\n", name)
		} else {
			fmt.Fprintf(&buf, "\t%s\n", name)
		}
	}
	fmt.Fprintf(&buf, ")\n\n")

	writeGraphemeTable(&buf, "graphemeBreakProperty", graphemeBreakProperty)
	writeIntervalTable(&buf, "doublewidth", doublewidth)
	writeIntervalTable(&buf, "emojiPresentation", emojiPresentation)

	// Emit InCB constants and table.
	fmt.Fprintf(&buf, "// Indic_Conjunct_Break property values from DerivedCoreProperties.\n")
	fmt.Fprintf(&buf, "// Used by grapheme break rule GB9c.\n")
	fmt.Fprintf(&buf, "const (\n")
	for i, name := range incbPropertyOrder {
		if i == 0 {
			fmt.Fprintf(&buf, "\t%s uint8 = iota\n", name)
		} else {
			fmt.Fprintf(&buf, "\t%s\n", name)
		}
	}
	fmt.Fprintf(&buf, ")\n\n")
	writeIncbTable(&buf, "incbProperty", incbTable)

	// Emit BMP stage1/stage2 tables as string literals for compact source and
	// fast compilation. Each byte is a packed property byte.
	fmt.Fprintf(&buf, "// bmpStage1 maps (rune >> 8) to an index into bmpStage2. For any rune r in the\n")
	fmt.Fprintf(&buf, "// Basic Multilingual Plane (r < 0x10000), its packed property byte is\n")
	fmt.Fprintf(&buf, "//   bmpStage2[int(bmpStage1[r>>8])*256 + int(r&0xFF)]\n")
	fmt.Fprintf(&buf, "var bmpStage1 = %q\n\n", string(stage1))
	fmt.Fprintf(&buf, "// bmpStage2 holds %d deduplicated 256-byte blocks, %d bytes total.\n", len(stage2)/256, len(stage2))
	fmt.Fprintf(&buf, "var bmpStage2 = %q\n\n", string(stage2))

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		fmt.Fprintf(os.Stderr, "gofmt error: %v\n%s", err, buf.String())
		os.Exit(1)
	}

	if err := os.WriteFile("tables.go", formatted, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "write error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Generated tables.go")
}

// buildGraphemeTable converts a rune→propname map into a sorted, merged
// sequence of [lo, hi, prop] intervals. Adjacent runes with the same property
// are coalesced.
func buildGraphemeTable(assigned map[rune]string) []graphemeInterval {
	if len(assigned) == 0 {
		return nil
	}
	runes := make([]rune, 0, len(assigned))
	for r := range assigned {
		runes = append(runes, r)
	}
	sort.Slice(runes, func(i, j int) bool { return runes[i] < runes[j] })

	var out []graphemeInterval
	cur := graphemeInterval{lo: runes[0], hi: runes[0], prop: assigned[runes[0]]}
	for _, r := range runes[1:] {
		p := assigned[r]
		if p == cur.prop && r == cur.hi+1 {
			cur.hi = r
			continue
		}
		out = append(out, cur)
		cur = graphemeInterval{lo: r, hi: r, prop: p}
	}
	out = append(out, cur)
	return out
}

func mustFetch(version, relPath string) []byte {
	cacheDir := filepath.Join("testdata", "unicode", version)
	cachePath := filepath.Join(cacheDir, filepath.Base(relPath))

	if data, err := os.ReadFile(cachePath); err == nil {
		fmt.Printf("  cache %s\n", cachePath)
		return data
	}

	url := fmt.Sprintf("https://www.unicode.org/Public/%s/ucd/%s", version, relPath)
	fmt.Printf("  GET %s\n", url)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fetch %s: %v\n", url, err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		fmt.Fprintf(os.Stderr, "fetch %s: status %d\n", url, resp.StatusCode)
		os.Exit(1)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read %s: %v\n", url, err)
		os.Exit(1)
	}
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "mkdir %s: %v\n", cacheDir, err)
		os.Exit(1)
	}
	if err := os.WriteFile(cachePath, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "cache write %s: %v\n", cachePath, err)
		os.Exit(1)
	}
	return data
}

// parsePropertyFile parses Unicode data files with format:
//
//	XXXX          ; Property # ...
//	XXXX..YYYY    ; Property # ...
func parsePropertyFile(data []byte) map[string][]interval {
	result := make(map[string][]interval)
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		if idx := strings.Index(line, "#"); idx >= 0 {
			line = line[:idx]
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ";", 2)
		if len(parts) != 2 {
			continue
		}
		codepoints := strings.TrimSpace(parts[0])
		property := strings.TrimSpace(parts[1])

		var lo, hi rune
		if dotdot := strings.Index(codepoints, ".."); dotdot >= 0 {
			lo = parseHex(codepoints[:dotdot])
			hi = parseHex(codepoints[dotdot+2:])
		} else {
			lo = parseHex(codepoints)
			hi = lo
		}
		result[property] = append(result[property], interval{lo, hi})
	}
	return result
}

func parseHex(s string) rune {
	s = strings.TrimSpace(s)
	n, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse hex %q: %v\n", s, err)
		os.Exit(1)
	}
	return rune(n)
}

func sortAndMerge(intervals []interval) []interval {
	if len(intervals) == 0 {
		return nil
	}
	sort.Slice(intervals, func(i, j int) bool {
		return intervals[i].lo < intervals[j].lo
	})
	merged := []interval{intervals[0]}
	for _, iv := range intervals[1:] {
		last := &merged[len(merged)-1]
		if iv.lo <= last.hi+1 {
			if iv.hi > last.hi {
				last.hi = iv.hi
			}
		} else {
			merged = append(merged, iv)
		}
	}
	return merged
}

func writeIntervalTable(buf *bytes.Buffer, name string, table []interval) {
	fmt.Fprintf(buf, "// %s contains %d intervals.\n", name, len(table))
	fmt.Fprintf(buf, "var %s = []interval{\n", name)
	for _, iv := range table {
		fmt.Fprintf(buf, "\t{0x%04X, 0x%04X},\n", iv.lo, iv.hi)
	}
	fmt.Fprintf(buf, "}\n\n")
}

func writeGraphemeTable(buf *bytes.Buffer, name string, table []graphemeInterval) {
	fmt.Fprintf(buf, "// %s contains %d intervals.\n", name, len(table))
	fmt.Fprintf(buf, "var %s = []graphemeInterval{\n", name)
	for _, iv := range table {
		fmt.Fprintf(buf, "\t{0x%04X, 0x%04X, %s},\n", iv.lo, iv.hi, iv.prop)
	}
	fmt.Fprintf(buf, "}\n\n")
}

func writeIncbTable(buf *bytes.Buffer, name string, table []incbIntervalGen) {
	fmt.Fprintf(buf, "// %s contains %d intervals.\n", name, len(table))
	fmt.Fprintf(buf, "var %s = []incbInterval{\n", name)
	for _, iv := range table {
		fmt.Fprintf(buf, "\t{0x%04X, 0x%04X, %s},\n", iv.lo, iv.hi, iv.prop)
	}
	fmt.Fprintf(buf, "}\n\n")
}

// parseNamedPropertyFile parses files like DerivedCoreProperties.txt where
// each line has the form:
//
//	XXXX(..YYYY)? ; PropertyName ; Value # comment
//
// It returns a map of value → intervals, for only the given property name.
func parseNamedPropertyFile(data []byte, propertyName string) map[string][]interval {
	result := make(map[string][]interval)
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		if idx := strings.Index(line, "#"); idx >= 0 {
			line = line[:idx]
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, ";")
		if len(parts) < 2 {
			continue
		}
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		name := parts[1]
		if name != propertyName {
			continue
		}
		if len(parts) < 3 {
			continue
		}
		value := parts[2]
		codepoints := parts[0]
		var lo, hi rune
		if dotdot := strings.Index(codepoints, ".."); dotdot >= 0 {
			lo = parseHex(codepoints[:dotdot])
			hi = parseHex(codepoints[dotdot+2:])
		} else {
			lo = parseHex(codepoints)
			hi = lo
		}
		result[value] = append(result[value], interval{lo, hi})
	}
	return result
}

// compressBMPBlocks splits a flat table of n*256 bytes into n blocks of 256
// bytes and returns (stage1, stage2) where stage1[i] is an index into stage2
// identifying the block for range [i*256, i*256+255].
//
// Duplicate blocks are coalesced: if two blocks are identical, they share a
// single stage2 entry. Huge swaths of unused Unicode are all-zero and
// collapse to a single shared block.
//
// The number of unique blocks is capped at 255 (fits in a uint8 stage1
// index). If that ever overflows the function panics with a clear message
// so the mistake is caught at generate time, not at runtime.
func compressBMPBlocks(flat []byte) (stage1, stage2 []byte) {
	if len(flat)%256 != 0 {
		panic(fmt.Sprintf("compressBMPBlocks: length %d is not a multiple of 256", len(flat)))
	}
	nBlocks := len(flat) / 256
	stage1 = make([]byte, nBlocks)
	type blockKey [256]byte
	indexOf := make(map[blockKey]int)
	var stage2Blocks []blockKey
	for blockIdx := 0; blockIdx < nBlocks; blockIdx++ {
		var key blockKey
		copy(key[:], flat[blockIdx*256:(blockIdx+1)*256])
		if idx, ok := indexOf[key]; ok {
			stage1[blockIdx] = byte(idx)
			continue
		}
		idx := len(stage2Blocks)
		if idx > 255 {
			panic(fmt.Sprintf("compressBMPBlocks: more than 256 unique blocks (%d); widen stage1 to uint16", idx+1))
		}
		indexOf[key] = idx
		stage2Blocks = append(stage2Blocks, key)
		stage1[blockIdx] = byte(idx)
	}
	stage2 = make([]byte, 0, len(stage2Blocks)*256)
	for _, b := range stage2Blocks {
		stage2 = append(stage2, b[:]...)
	}
	return stage1, stage2
}

func buildIncbTable(assigned map[rune]string) []incbIntervalGen {
	if len(assigned) == 0 {
		return nil
	}
	runes := make([]rune, 0, len(assigned))
	for r := range assigned {
		runes = append(runes, r)
	}
	sort.Slice(runes, func(i, j int) bool { return runes[i] < runes[j] })

	var out []incbIntervalGen
	cur := incbIntervalGen{lo: runes[0], hi: runes[0], prop: assigned[runes[0]]}
	for _, r := range runes[1:] {
		p := assigned[r]
		if p == cur.prop && r == cur.hi+1 {
			cur.hi = r
			continue
		}
		out = append(out, cur)
		cur = incbIntervalGen{lo: r, hi: r, prop: p}
	}
	out = append(out, cur)
	return out
}
