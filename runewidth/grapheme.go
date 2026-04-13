package runewidth

import (
	"iter"
	"unicode/utf8"
)

// UAX#29 grapheme cluster parser states. Values chosen to pack into a uint8.
const (
	gsAny uint8 = iota
	gsCR
	gsControlLF
	gsL
	gsLVV
	gsLVTT
	gsPrepend
	gsExtPic
	gsExtPicZWJ
	gsRIOdd
	gsRIEven
)

// graphemeTransition implements the UAX#29 grapheme cluster break rules
// (GB1-GB13, GB999). Given a current state and the next code point's grapheme
// break property, it returns the new state and whether there is a cluster
// boundary between the previous and next code point.
//
// Rule numbers in the comments below match Unicode TR29. The generic GB9/GB9a
// Extend/ZWJ/SpacingMark case resets the state to gsAny, matching uniseg's
// transition table. This is important: after "L × Extend" the next L must
// still break (÷) because Extend breaks the Hangul-syllable progression.
//
// See https://www.unicode.org/reports/tr29/#Grapheme_Cluster_Boundary_Rules
func graphemeTransition(state, prop uint8) (newState uint8, boundary bool) {
	// Hot path: plain character followed by plain character. This covers
	// ASCII-to-ASCII, CJK-to-CJK, and every other "next rune has no special
	// grapheme property" case — which is the vast majority of text.
	if state == gsAny && prop == gbOther {
		return gsAny, true
	}
	// GB3: CR × LF
	if state == gsCR && prop == gbLF {
		return gsControlLF, false
	}
	// GB4: (Control|CR|LF) ÷
	if state == gsCR || state == gsControlLF {
		return stateFromProp(prop), true
	}
	// GB5: ÷ (Control|CR|LF)
	switch prop {
	case gbControl, gbLF:
		return gsControlLF, true
	case gbCR:
		return gsCR, true
	}

	// State-specific progressions that must win over the generic Extend rule.
	switch state {
	case gsL:
		// GB6: L × (L|V|LV|LVT)
		switch prop {
		case gbL:
			return gsL, false
		case gbV, gbLV:
			return gsLVV, false
		case gbLVT:
			return gsLVTT, false
		}
	case gsLVV:
		// GB7: (LV|V) × (V|T)
		switch prop {
		case gbV:
			return gsLVV, false
		case gbT:
			return gsLVTT, false
		}
	case gsLVTT:
		// GB8: (LVT|T) × T
		if prop == gbT {
			return gsLVTT, false
		}
	case gsPrepend:
		// GB9b: Prepend × Any (except the CR/LF/Control already consumed above)
		return stateFromProp(prop), false
	case gsExtPic:
		// GB11: ExtPic Extend* × ZWJ × ExtPic
		switch prop {
		case gbExtend:
			return gsExtPic, false
		case gbZWJ:
			return gsExtPicZWJ, false
		}
	case gsExtPicZWJ:
		if prop == gbExtendedPictographic {
			return gsExtPic, false
		}
	case gsRIOdd:
		// GB12/GB13: sot (RI RI)* RI × RI
		if prop == gbRegionalIndicator {
			return gsRIEven, false
		}
	}

	// GB9, GB9a: × (Extend|ZWJ|SpacingMark). These do not start a new cluster,
	// but they do reset the state to gsAny — the previous base character's
	// Hangul/RI/Prepend context no longer extends through them.
	switch prop {
	case gbExtend, gbZWJ, gbSpacingMark:
		return gsAny, false
	}

	// GB999: Any ÷ Any — new cluster, enter state based on new prop.
	return stateFromProp(prop), true
}

// stateFromProp chooses the initial state when entering a new cluster whose
// first code point has the given property.
func stateFromProp(prop uint8) uint8 {
	switch prop {
	case gbCR:
		return gsCR
	case gbLF, gbControl:
		return gsControlLF
	case gbL:
		return gsL
	case gbLV:
		return gsLVV
	case gbLVT:
		return gsLVTT
	case gbV:
		return gsLVV
	case gbT:
		return gsLVTT
	case gbPrepend:
		return gsPrepend
	case gbExtendedPictographic:
		return gsExtPic
	case gbRegionalIndicator:
		return gsRIOdd
	}
	return gsAny
}

// firstGraphemeCluster extracts the first grapheme cluster from s. It returns
// the cluster substring, the remainder of s, and the cluster's display width
// (terminal cells). If s is empty, it returns empty values.
//
// Clusters are independent: each call re-derives its initial state from the
// first rune, so there is no state to thread between calls. The function
// allocates nothing and is safe to call in a tight loop.
func firstGraphemeCluster(s string) (cluster, rest string, width int) {
	if len(s) == 0 {
		return "", "", 0
	}

	r, size := utf8.DecodeRuneInString(s)
	firstPacked := runeProps(r)
	firstProp := firstPacked & packedGBMask
	st := stateFromProp(firstProp)
	width = runeWidthFromPacked(r, firstPacked)

	// Track keycap sequence pieces so we can force width=2 for
	// (base ∈ {#, *, 0-9}) VS16 U+20E3, which uniseg does not handle.
	isKeycapBase := r == '#' || r == '*' || (r >= '0' && r <= '9')
	sawVS16 := false
	sawKeycap := false

	// Indic_Conjunct_Break sub-state for rule GB9c. The sub-state stays at
	// incbStateNone until we enter a cluster whose first rune is an InCB
	// Consonant — the vast majority of clusters. Once incbStateNone, no
	// subsequent runes need the incb lookup, so we skip it entirely.
	incbSub := incbInitial(r)

	length := size

	for length < len(s) {
		r2, sz := utf8.DecodeRuneInString(s[length:])
		packed := runeProps(r2)
		prop := packed & packedGBMask
		var boundary bool
		st, boundary = graphemeTransition(st, prop)

		// GB9c override path. Only runs when the cluster has already been
		// advanced past a Consonant; incbStateNone is the common case and
		// hits the else branch with zero extra lookups.
		if incbSub != incbStateNone {
			if boundary && incbSub == incbAfterLinker && lookupInCB(r2) == incbConsonant {
				boundary = false
				incbSub = incbAfterConsonant
			} else {
				incbSub = incbAdvance(incbSub, r2, prop)
			}
		}

		if boundary {
			// Emoji keycap sequence override — applied before the return so
			// that clusters embedded in larger strings get the correct width.
			// See the matching block after the loop for the EOS case.
			if isKeycapBase && sawVS16 && sawKeycap && width < 2 {
				width = 2
			}
			return s[:length], s[length:], width
		}

		// Width accumulation rules matching uniseg v0.4.7:
		//   - Extended_Pictographic clusters take their width from the base rune
		//     plus VS15/VS16 overrides.
		//   - Regional indicator and Hangul syllable clusters (L/V/T/LV/LVT)
		//     take their width from the base rune; trailing Jamo do not add
		//     additional width to the syllable block.
		//   - All other clusters accumulate runeWidthFromPacked of subsequent runes.
		if firstProp == gbExtendedPictographic {
			switch r2 {
			case 0xFE0E:
				width = 1
			case 0xFE0F:
				width = 2
			}
		} else if firstProp != gbRegionalIndicator &&
			firstProp != gbL &&
			firstProp != gbV &&
			firstProp != gbT &&
			firstProp != gbLV &&
			firstProp != gbLVT {
			width += runeWidthFromPacked(r2, packed)
		}

		if r2 == 0xFE0F {
			sawVS16 = true
		}
		if r2 == 0x20E3 {
			sawKeycap = true
		}

		length += sz
	}

	// Emoji keycap sequence: base {#, *, 0-9} + VS16 + U+20E3 → width 2.
	if isKeycapBase && sawVS16 && sawKeycap && width < 2 {
		width = 2
	}

	return s, "", width
}

// Indic_Conjunct_Break sub-states for rule GB9c tracking.
const (
	incbStateNone uint8 = iota
	incbAfterConsonant
	incbAfterLinker
)

// incbInitial returns the starting GB9c sub-state for a cluster whose first
// rune is r. If r is an Indic consonant, the cluster enters incbAfterConsonant
// so subsequent Linker+Consonant continuations can suppress a break.
//
// This is called once per cluster (on the base rune) from firstGraphemeCluster,
// so the fast reject matters: ASCII, CJK, and emoji cluster bases must not
// pay for a binary search on every call. InCB=Consonant code points fall into
// four disjoint bands — the check below is exact against the generated
// incbProperty table, and TestIncbInitialBounds will flag any drift if a
// future Unicode version grows the set.
func incbInitial(r rune) uint8 {
	inBand := (r >= 0x0915 && r <= 0x1BBD) ||
		(r >= 0xA989 && r <= 0xABDA) ||
		(r >= 0x10A00 && r <= 0x10A35) ||
		(r >= 0x11103 && r <= 0x11F33)
	if !inBand {
		return incbStateNone
	}
	if lookupInCB(r) == incbConsonant {
		return incbAfterConsonant
	}
	return incbStateNone
}

// incbAdvance updates the GB9c sub-state given the next rune's properties,
// assuming no break has been taken between the previous rune and this one.
// Callers only invoke this when state != incbStateNone, so that case is
// handled by the caller and not reachable here.
//
// The "between" set in GB9c is (InCB=Extend | InCB=Linker), which includes
// ZWJ (U+200D has InCB=Extend). The explicit gbZWJ check is belt-and-braces
// in case a future Unicode update changes that classification.
func incbAdvance(state uint8, r rune, gbp uint8) uint8 {
	incb := lookupInCB(r)
	extendLike := gbp == gbExtend || gbp == gbZWJ || incb == incbExtend || incb == incbLinker

	switch state {
	case incbAfterConsonant:
		if incb == incbLinker {
			return incbAfterLinker
		}
		if extendLike {
			return incbAfterConsonant
		}
		if incb == incbConsonant {
			return incbAfterConsonant
		}
	case incbAfterLinker:
		if extendLike {
			return incbAfterLinker
		}
		if incb == incbConsonant {
			return incbAfterConsonant
		}
	}
	return incbStateNone
}

// graphemeIter walks s calling fn for each grapheme cluster with the cluster
// substring and its display width. It allocates nothing.
func graphemeIter(s string, fn func(cluster string, width int)) {
	for len(s) > 0 {
		var c string
		var w int
		c, s, w = firstGraphemeCluster(s)
		fn(c, w)
	}
}

// Graphemes returns an iterator over the grapheme clusters in s. Each yielded
// pair is (cluster, width) where cluster is a substring of s and width is the
// number of terminal cells the cluster occupies.
//
// Cluster boundaries follow Unicode Standard Annex #29 (UAX#29) at the
// Unicode version given by [UnicodeVersion]. The iterator allocates nothing.
//
// This is a cluster-aware alternative to ranging over runes for operations
// that must respect user-perceived characters — for example, advancing a
// text cursor one visual character at a time, where a "character" may be
// multiple runes (emoji ZWJ sequences, base + combining marks, flags,
// keycaps, and so on).
func Graphemes(s string) iter.Seq2[string, int] {
	return func(yield func(string, int) bool) {
		for len(s) > 0 {
			var c string
			var w int
			c, s, w = firstGraphemeCluster(s)
			if !yield(c, w) {
				return
			}
		}
	}
}

// runeWidthFromPacked returns the display width of a single rune given the
// packed property byte produced by [runeProps]. Width > 2 is only returned
// for U+2E3A (TWO-EM DASH) and U+2E3B (THREE-EM DASH); all other wide
// characters are 2, and control/extend/ZWJ are 0.
func runeWidthFromPacked(r rune, packed byte) int {
	if r == 0xAD {
		return 0 // Soft hyphen — shared by all APIs (see RuneWidth).
	}
	prop := packed & packedGBMask
	switch prop {
	case gbControl, gbCR, gbLF, gbExtend, gbZWJ:
		return 0
	case gbRegionalIndicator:
		return 2
	case gbExtendedPictographic:
		if packed&packedEmojiPresent != 0 {
			return 2
		}
		return 1
	}
	switch r {
	case 0x2E3A:
		return 3
	case 0x2E3B:
		return 4
	}
	if packed&packedDoubleWidth != 0 {
		return 2
	}
	return 1
}

// runeWidthForRune is the public [RuneWidth] fallback: it looks up the packed
// property byte from the tables and delegates to runeWidthFromPacked.
func runeWidthForRune(r rune) int {
	return runeWidthFromPacked(r, runeProps(r))
}
