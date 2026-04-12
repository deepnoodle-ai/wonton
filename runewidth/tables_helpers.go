package runewidth

// interval is a closed inclusive range [lo, hi] of runes. Tables are sorted by
// lo and non-overlapping so that membership can be tested with binary search.
type interval struct {
	lo, hi rune
}

// graphemeInterval is a closed inclusive range [lo, hi] of runes sharing the
// same UAX#29 Grapheme_Cluster_Break property (plus the Extended_Pictographic
// pseudo-property). Tables are sorted by lo and non-overlapping.
type graphemeInterval struct {
	lo, hi rune
	prop   uint8
}

// incbInterval is a closed inclusive range [lo, hi] of runes sharing the same
// Indic_Conjunct_Break property value from DerivedCoreProperties. Used by
// UAX#29 grapheme break rule GB9c.
type incbInterval struct {
	lo, hi rune
	prop   uint8
}

// inTable reports whether r falls inside any interval in table.
func inTable(table []interval, r rune) bool {
	lo, hi := 0, len(table)-1
	for lo <= hi {
		mid := (lo + hi) / 2
		switch {
		case table[mid].hi < r:
			lo = mid + 1
		case table[mid].lo > r:
			hi = mid - 1
		default:
			return true
		}
	}
	return false
}

// Packed BMP byte layout: bits 0..3 gbProperty, bit 4 doublewidth, bit 5
// emojiPresentation. Helpers below extract the fields.
const (
	packedGBMask       = 0x0F
	packedDoubleWidth  = 1 << 4
	packedEmojiPresent = 1 << 5
)

// stagedCovered is the upper bound (exclusive) of runes served by the
// stage1/stage2 direct-lookup table. Code points below it hit O(1); above
// it fall back to binary search.
const stagedCovered = 0x20000

// runeProps returns the packed property byte for r. For runes inside the
// staged table range this is a 2-stage O(1) lookup. For higher planes it
// falls back to binary searches on graphemeBreakProperty, doublewidth, and
// emojiPresentation.
func runeProps(r rune) byte {
	if uint32(r) < stagedCovered {
		return bmpStage2[int(bmpStage1[r>>8])*256+int(r&0xFF)]
	}
	if r > 0x10FFFF || r < 0 {
		return 0
	}
	var b byte
	b = slowLookupGB(r)
	if slowInDoubleWidth(r) {
		b |= packedDoubleWidth
	}
	if slowInEmojiPres(r) {
		b |= packedEmojiPresent
	}
	return b
}

// lookupGB returns the Grapheme_Cluster_Break property for r, or gbOther if r
// has none assigned.
func lookupGB(r rune) uint8 {
	if uint32(r) < stagedCovered {
		return bmpStage2[int(bmpStage1[r>>8])*256+int(r&0xFF)] & packedGBMask
	}
	if r > 0x10FFFF || r < 0 {
		return gbOther
	}
	return slowLookupGB(r)
}

// slowLookupGB performs a binary search on graphemeBreakProperty. Used for
// supplementary plane runes that bypass the BMP fast path.
func slowLookupGB(r rune) uint8 {
	lo, hi := 0, len(graphemeBreakProperty)-1
	for lo <= hi {
		mid := (lo + hi) / 2
		iv := graphemeBreakProperty[mid]
		switch {
		case iv.hi < r:
			lo = mid + 1
		case iv.lo > r:
			hi = mid - 1
		default:
			return iv.prop
		}
	}
	return gbOther
}

func slowInDoubleWidth(r rune) bool { return inTable(doublewidth, r) }
func slowInEmojiPres(r rune) bool   { return inTable(emojiPresentation, r) }

// lookupInCB returns the Indic_Conjunct_Break property for r, or incbNone if
// r has none assigned. A range-based fast reject handles most common
// characters (ASCII, CJK, emoji, most of Latin) without a binary search.
func lookupInCB(r rune) uint8 {
	// Fast reject for runes that can't have any InCB property.
	// All InCB assignments are in the range [0x0300, 0xE01EF], and within
	// that span combining marks (0x0300..) are the most common hit.
	if r < 0x0300 || r > 0xE01EF {
		return incbNone
	}
	lo, hi := 0, len(incbProperty)-1
	for lo <= hi {
		mid := (lo + hi) / 2
		iv := incbProperty[mid]
		switch {
		case iv.hi < r:
			lo = mid + 1
		case iv.lo > r:
			hi = mid - 1
		default:
			return iv.prop
		}
	}
	return incbNone
}
