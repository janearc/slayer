package terminal

import "testing"

// sextantOf is the character for a pattern, written from the Unicode
// rule and not from Glyph. the sextants are U+1FB00 onward, one for each
// pattern read as a binary number, top left as its lowest bit, in order.
//
// four patterns are left out, being characters already: none (a space),
// all (U+2588), the left column, 21 (U+258C), and the right, 42 (U+2590).
// a sextant is U+1FB00 plus the patterns from 1 to just before its own
// that are not 21 or 42.
func sextantOf(p int) rune {
	switch p {
	case 0:
		return 0x20
	case 63:
		return 0x2588
	case 21:
		return 0x258C
	case 42:
		return 0x2590
	}
	n := 0
	for q := 1; q < p; q++ {
		if q != 21 && q != 42 {
			n++
		}
	}
	return 0x1FB00 + rune(n)
}

// each of the 64 patterns draws the character the standard gives it. the
// oracle is held to the standard's end points first: U+1FB00 is the top
// left alone (pattern 1), and U+1FB3B the last sextant, everything but
// the top left (pattern 62).
func TestEveryPatternDrawsTheCharacterUnicodeGivesIt(t *testing.T) {
	if sextantOf(1) != 0x1FB00 || sextantOf(62) != 0x1FB3B {
		t.Fatalf("the oracle gives %U and %U for the first and last sextant", sextantOf(1), sextantOf(62))
	}
	for p := range 64 {
		if got, want := Glyph(uint8(p)), sextantOf(p); got != want {
			t.Errorf("Glyph(%d) = %U, want %U", p, got, want)
		}
	}
}

// the two columns are the halves, not sextants, and the patterns either
// side of each are the sextants either side of a gap. U+1FB13 is the
// 20th sextant and pattern 20; none is spent on 21, so 22 is the 21st.
//
// 41 and 43 are the same about the right half.
func TestTheHalvesLeaveAGapInTheSextants(t *testing.T) {
	for _, w := range []struct {
		p    uint8
		want rune
	}{
		{20, 0x1FB13}, {21, 0x258C}, {22, 0x1FB14},
		{41, 0x1FB27}, {42, 0x2590}, {43, 0x1FB28},
	} {
		if got := Glyph(w.p); got != w.want {
			t.Errorf("Glyph(%d) = %U, want %U", w.p, got, w.want)
		}
	}
}

// 64 patterns, 64 different characters. with the oracle above this is
// not a permutation that passes: it is the standard's own order.
func TestSextantsAreSixtyAndAllDifferent(t *testing.T) {
	seen := map[rune]int{}
	sextants := 0
	for p := range 64 {
		g := Glyph(uint8(p))
		if q, dup := seen[g]; dup {
			t.Fatalf("patterns %d and %d both draw %U", q, p, g)
		}
		seen[g] = p
		if g >= 0x1FB00 && g <= 0x1FB3B {
			sextants++
		}
	}
	if sextants != 60 {
		t.Errorf("%d of the characters are in U+1FB00 to U+1FB3B, want 60", sextants)
	}
}
