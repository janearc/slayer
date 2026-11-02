package typography

import (
	"os"
	"testing"

	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
)

// cff is x/image's own cff test font, copied into testdata.
func cff(t *testing.T) *Font {
	t.Helper()
	raw, err := os.ReadFile("testdata/CFFTest.otf")
	if err != nil {
		t.Fatal(err)
	}
	return mustFont(t, raw)
}

// glyphRings is a letter's rings, flattened to a unit and cleaned, as
// solids reads them before it nests them.
func glyphRings(t *testing.T, face *Font, letter rune) []ring {
	t.Helper()
	glyph, err := face.Index(letter)
	if err != nil {
		t.Fatal(err)
	}
	outline, err := face.outline(glyph)
	if err != nil {
		t.Fatal(err)
	}
	var made []ring
	for _, contour := range outline {
		if r := cleaned(flattened(contour, 1)); r != nil {
			made = append(made, r)
		}
	}
	return made
}

// shape is how many solids and how many holes in each.
func shapeOf(solids []solid) []int {
	var made []int
	for _, one := range solids {
		made = append(made, len(one.holes))
	}
	return made
}

// same is whether two lists of hole counts are alike.
func same(a, b []int) bool {
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

// "o" is one solid with one hole, "i" is two solids with none, "8"
// and "B" are one with two, and "%" is its slash and two rings; in go's
// regular and its bold.
func TestLettersNestAsTheyAreDrawn(t *testing.T) {
	drawn := map[rune][]int{'o': {1}, 'i': {0, 0}, '8': {2}, 'B': {2},
		'l': {0}, '%': {0, 1, 1}, 'j': {0, 0}}
	for _, raw := range [][]byte{goregular.TTF, gobold.TTF} {
		face := mustFont(t, raw)
		for letter, want := range drawn {
			got := shapeOf(nested(glyphRings(t, face, letter)))
			if !same(got, want) {
				t.Errorf("%q nests as %v, want %v", letter, got,
					want)
			}
		}
	}
}

// truetype winds an outer ring clockwise and cff counterclockwise, as
// the fonts here do: go's "o" outer has a negative area as read, and the
// cff font's "0" a positive one. nested gives both one solid with one
// hole, the outer counterclockwise and the hole clockwise.
func TestNestingIsTheSameWhicheverWayTheFontWinds(t *testing.T) {
	for name, each := range map[string]struct {
		face   *Font
		letter rune
		outer  float64 // the sign of the bigger ring's area, as read
	}{"truetype o": {mustFont(t, goregular.TTF), 'o', -1},
		"cff 0": {cff(t), '0', 1}} {
		rings := glyphRings(t, each.face, each.letter)
		big := rings[0]
		for _, r := range rings {
			if abs(r.area()) > abs(big.area()) {
				big = r
			}
		}
		if big.area()*each.outer <= 0 {
			t.Errorf("%s: the outer as read has area %v", name,
				big.area())
		}
		solids := nested(rings)
		if !same(shapeOf(solids), []int{1}) {
			t.Fatalf("%s nests as %v", name, shapeOf(solids))
		}
		outer, hole := solids[0].outer.area(), solids[0].holes[0].area()
		if outer <= 0 || hole >= 0 {
			t.Errorf("%s: outer %v, hole %v", name, outer, hole)
		}
	}
}

// every ring of a letter turned the other way, as a font that winds the
// other way would give it, nests the same: the same solids, the same
// holes, the same areas.
func TestEveryRingTurnedNestsTheSame(t *testing.T) {
	face := mustFont(t, goregular.TTF)
	for _, letter := range "oi8B%@aeg" {
		rings := glyphRings(t, face, letter)
		var turned []ring
		for _, r := range rings {
			turned = append(turned, r.reversed())
		}
		one, two := nested(rings), nested(turned)
		if !same(shapeOf(one), shapeOf(two)) {
			t.Errorf("%q: %v, turned %v", letter, shapeOf(one),
				shapeOf(two))
			continue
		}
		for i := range one {
			was, now := one[i].outer.area(), two[i].outer.area()
			if abs(was-now) > 1e-6 {
				t.Errorf("%q solid %d: %v, turned %v", letter,
					i, was, now)
			}
		}
	}
}

// abs is x without its sign.
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// a point at the same place as the one before, and a point on the line
// between its neighbours, are cleaned away; a corner stays, and a ring
// of fewer than three points left is nothing.
func TestCleaningKeepsOnlyCorners(t *testing.T) {
	square := ring{{0, 0}, {5, 0}, {10, 0}, {10, 0}, {10, 10}, {0, 10},
		{0, 0}}
	got := cleaned(square)
	if len(got) != 4 || got.area() != 100 {
		t.Errorf("the square cleaned is %v", got)
	}
	if got := cleaned(ring{{0, 0}, {1, 1}, {2, 2}}); got != nil {
		t.Errorf("three points on a line cleaned are %v", got)
	}
	spike := ring{{0, 0}, {10, 0}, {20, 0}, {10, 0}, {10, 10}}
	if got := cleaned(spike); len(got) != 3 {
		t.Errorf("a ring with a spike cleaned is %v", got)
	}
}

// two squares that overlap cross, and so do two that share a corner; a
// square inside another does not, and neither do a ring's own steps
// where one meets the next.
func TestRingsThatCrossAreFound(t *testing.T) {
	square := func(x, y, side float64) ring {
		return ring{{x, y}, {x + side, y}, {x + side, y + side},
			{x, y + side}}
	}
	for name, each := range map[string]struct {
		rings []ring
		want  bool
	}{
		"overlapping": {[]ring{square(0, 0, 10), square(5, 5, 10)},
			true},
		"corner to corner": {[]ring{square(0, 0, 10),
			square(10, 10, 5)}, true},
		"one inside": {[]ring{square(0, 0, 10), square(2, 2, 5)},
			false},
		"alone": {[]ring{square(0, 0, 10)}, false},
		"a bow tie": {[]ring{{{0, 0}, {10, 10}, {10, 0}, {0, 10}}},
			true},
	} {
		if got := crossed(each.rings); got != each.want {
			t.Errorf("%s: crossed %v", name, got)
		}
	}
}

// holds counts a point inside a ring and not one outside it, whichever
// way the ring turns.
func TestARingHoldsWhatIsInsideIt(t *testing.T) {
	square := ring{{0, 0}, {10, 0}, {10, 10}, {0, 10}}
	for _, r := range []ring{square, square.reversed()} {
		if !r.holds(point{5, 5}) || r.holds(point{15, 5}) ||
			r.holds(point{5, -1}) {
			t.Errorf("%v holds wrongly", r)
		}
	}
}
