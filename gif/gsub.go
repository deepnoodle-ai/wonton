package gif

import (
	"encoding/binary"
	"slices"

	"golang.org/x/image/font/sfnt"
)

// ligatureLookup is one GSUB ligature substitution lookup. It maps a first
// glyph to the ligatures that start with it, in the font's order of
// preference.
type ligatureLookup map[sfnt.GlyphIndex][]ligature

type ligature struct {
	rest  []sfnt.GlyphIndex // components after the first
	glyph sfnt.GlyphIndex   // the glyph that replaces the components
}

// apply replaces each run of glyphs that matches a ligature with the
// ligature glyph, scanning left to right.
func (l ligatureLookup) apply(glyphs []sfnt.GlyphIndex) []sfnt.GlyphIndex {
	out := make([]sfnt.GlyphIndex, 0, len(glyphs))
	for i := 0; i < len(glyphs); {
		n := 1
		g := glyphs[i]
		for _, lig := range l[glyphs[i]] {
			if len(glyphs)-i-1 >= len(lig.rest) && slices.Equal(glyphs[i+1:i+1+len(lig.rest)], lig.rest) {
				g, n = lig.glyph, 1+len(lig.rest)
				break
			}
		}
		out = append(out, g)
		i += n
	}
	return out
}

// maxGSUBWork bounds the entries parseLigatures reads, so a corrupt table
// whose offsets point at the same data many times cannot stall it.
const maxGSUBWork = 1 << 22

// parseLigatures reads the ligature substitutions of a GSUB table's ccmp,
// liga, and rlig features, in lookup order. It reads lookup type 4 and type
// 4 inside a type 7 extension. It skips other lookup types, such as
// contextual substitutions, and the script and language lists. A malformed
// table yields the lookups read before the error.
//
// The format is described at
// https://learn.microsoft.com/en-us/typography/opentype/spec/gsub
func parseLigatures(b []byte) []ligatureLookup {
	r := &gsubReader{b: b}
	featureList := r.u16(6)
	lookupList := r.u16(8)

	var indices []int
	for i, n := 0, r.u16(featureList); i < n && r.ok(); i++ {
		rec := featureList + 2 + i*6
		switch string(r.bytes(rec, 4)) {
		case "ccmp", "liga", "rlig":
		default:
			continue
		}
		feature := featureList + r.u16(rec+4)
		for j, m := 0, r.u16(feature+2); j < m && r.ok(); j++ {
			indices = append(indices, r.u16(feature+4+j*2))
		}
	}
	slices.Sort(indices)
	indices = slices.Compact(indices)

	var lookups []ligatureLookup
	lookupCount := r.u16(lookupList)
	for _, index := range indices {
		if index >= lookupCount || !r.ok() {
			break
		}
		lookup := lookupList + r.u16(lookupList+2+index*2)
		kind := r.u16(lookup)
		table := ligatureLookup{}
		for k, n := 0, r.u16(lookup+4); k < n && r.ok(); k++ {
			sub := lookup + r.u16(lookup+6+k*2)
			subKind := kind
			if kind == 7 { // extension
				subKind = r.u16(sub + 2)
				sub += r.u32(sub + 4)
			}
			if subKind == 4 {
				r.readLigatureSubst(table, sub)
			}
		}
		if len(table) > 0 {
			lookups = append(lookups, table)
		}
	}
	return lookups
}

// readLigatureSubst adds the ligatures of a format 1 ligature substitution
// subtable at offset sub to table.
func (r *gsubReader) readLigatureSubst(table ligatureLookup, sub int) {
	if r.u16(sub) != 1 {
		return
	}
	setCount := r.u16(sub + 4)
	firsts := r.coverage(sub+r.u16(sub+2), setCount)
	for i, first := range firsts {
		set := sub + r.u16(sub+6+i*2)
		for j, n := 0, r.u16(set); j < n && r.ok(); j++ {
			lig := set + r.u16(set+2+j*2)
			glyph := sfnt.GlyphIndex(r.u16(lig))
			count := r.u16(lig + 2)
			if count < 1 {
				continue
			}
			rest := make([]sfnt.GlyphIndex, count-1)
			for c := range rest {
				rest[c] = sfnt.GlyphIndex(r.u16(lig + 4 + c*2))
			}
			if !r.ok() {
				return
			}
			table[first] = append(table[first], ligature{rest: rest, glyph: glyph})
		}
	}
}

// coverage reads the coverage table at off and returns the glyphs at
// coverage indexes 0 to n-1.
func (r *gsubReader) coverage(off, n int) []sfnt.GlyphIndex {
	glyphs := make([]sfnt.GlyphIndex, n)
	switch r.u16(off) {
	case 1:
		for i := 0; i < min(n, r.u16(off+2)) && r.ok(); i++ {
			glyphs[i] = sfnt.GlyphIndex(r.u16(off + 4 + i*2))
		}
	case 2:
		for i, ranges := 0, r.u16(off+2); i < ranges && r.ok(); i++ {
			rec := off + 4 + i*6
			start, end, index := r.u16(rec), r.u16(rec+2), r.u16(rec+4)
			for g := start; g <= end && index+g-start < n && r.ok(); g++ {
				glyphs[index+g-start] = sfnt.GlyphIndex(g)
			}
		}
	default:
		return nil
	}
	if !r.ok() {
		return nil
	}
	return glyphs
}

// gsubReader reads big-endian values from a font table. An out-of-range
// read, or more reads than maxGSUBWork, marks the reader bad, and every
// later read returns zero.
type gsubReader struct {
	b    []byte
	work int
	bad  bool
}

func (r *gsubReader) ok() bool { return !r.bad }

func (r *gsubReader) bytes(off, n int) []byte {
	r.work++
	if r.bad || r.work > maxGSUBWork || off < 0 || n > len(r.b) || off > len(r.b)-n {
		r.bad = true
		return make([]byte, n)
	}
	return r.b[off : off+n]
}

func (r *gsubReader) u16(off int) int { return int(binary.BigEndian.Uint16(r.bytes(off, 2))) }

func (r *gsubReader) u32(off int) int { return int(binary.BigEndian.Uint32(r.bytes(off, 4))) }
