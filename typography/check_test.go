package typography

import (
	"errors"
	"math"
	"strings"
	"testing"

	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

// a text is checked at its edges: a size or depth of zero, a bevel of exactly
// half the depth, an endless size, a smoothing angle past pi, each refused by
// what it says; a bevel just under half, a smoothing angle of exactly pi and of
// zero, and a tolerance of zero are taken.
func TestATextIsCheckedAtItsEdges(t *testing.T) {
	inf, nan := math.Inf(1), math.NaN()
	for said, wrong := range map[string]Text{
		"a size above nought":         {Size: 0, Depth: 1},
		"a size above nought, not +":  {Size: inf, Depth: 1},
		"a size above nought, not N":  {Size: nan, Depth: 1},
		"a depth above nought":        {Size: 1, Depth: 0},
		"a depth above nought, not +": {Size: 1, Depth: inf},
		"a bevel from nought":         {Size: 1, Depth: 1, Bevel: 0.5},
		"to under half the depth, not +": {Size: 1, Depth: 1,
			Bevel: inf},
		"a smoothing angle": {Size: 1, Depth: 1,
			Smooth: math.Pi + 1e-9},
		"a tolerance of nought or more, not +": {Size: 1, Depth: 1,
			Tolerance: inf},
	} {
		wrong.Words = "a"
		err := wrong.check()
		if !errors.Is(err, ErrType) ||
			!strings.Contains(err.Error(), said) {
			t.Errorf("%+v: %v, wanted %q", wrong, err, said)
		}
	}
	for _, right := range []Text{{Bevel: 0.4999}, {Smooth: math.Pi},
		{Smooth: 0}, {Tolerance: 0}, {Bevel: 0.4}} {
		right.Words, right.Size, right.Depth = "a", 1, 1
		if err := right.check(); err != nil {
			t.Errorf("%+v: %v", right, err)
		}
	}
}

// kerning a glyph numbered as many as the font has, the first past its
// last, is an error on either side of the pair.
func TestKerningOnePastTheLastGlyphIsAnError(t *testing.T) {
	face := cff(t)
	n := Glyph(face.face.NumGlyphs())
	for _, pair := range [][2]Glyph{{n, 1}, {1, n}, {n - 1, n}} {
		_, err := face.Kern(pair[0], pair[1])
		if !errors.Is(err, ErrType) {
			t.Errorf("kerning %v of %d: %v", pair, n, err)
		}
	}
	if _, err := face.Kern(n-1, n-1); err != nil {
		t.Errorf("kerning the last glyph with itself: %v", err)
	}
}

// segments that do not start with a move start a contour anyway, at the pen's
// first place, zero, as sfnt's never do but a font could.
func TestSegmentsWithNoMoveStartAContour(t *testing.T) {
	line := func(x, y int) sfnt.Segment {
		return sfnt.Segment{Op: sfnt.SegmentOpLineTo,
			Args: [3]fixed.Point26_6{fixed.P(x, y)}}
	}
	got := contours(sfnt.Segments{line(10, 0), line(10, -10), line(0, 0)})
	if len(got) != 1 || len(got[0]) != 3 || got[0][0][0] != (point{}) ||
		got[0][1][1] != (point{10, 10}) {
		t.Errorf("contours %v", got)
	}
}
