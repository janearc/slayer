package typography

import (
	"errors"
	"math"
	"testing"

	"golang.org/x/image/font/gofont/goregular"
)

// a font with no kerning table at all, as the cff test font is, kerns every
// pair by zero rather than failing the line; kerning a glyph the font does not
// have is an error.
func TestAFontWithNoKerningKernsNothing(t *testing.T) {
	face := cff(t)
	zero, _ := face.Index('0')
	one, _ := face.Index('1')
	if got, err := face.Kern(zero, one); got != 0 || err != nil {
		t.Errorf("01 kerned %v, %v", got, err)
	}
	if _, err := face.Kern(zero, 60000); !errors.Is(err, ErrType) {
		t.Errorf("kerning a glyph the font has not: %v", err)
	}
}

// a glyph the font does not have is an error naming it, whether its
// outline or its solids are asked for.
func TestAGlyphTheFontHasNotIsAnError(t *testing.T) {
	face := mustFont(t, goregular.TTF)
	if _, err := face.outline(60000); !errors.Is(err, ErrType) {
		t.Errorf("glyph 60000's outline: %v", err)
	}
	if _, err := face.solids(60000, 1); !errors.Is(err, ErrType) {
		t.Errorf("glyph 60000's solids: %v", err)
	}
}

// an outline whose contours cross makes no solids: two squares that
// overlap are errCrossed.
func TestAnOutlineThatCrossesIsRefused(t *testing.T) {
	square := func(x, y float64) []piece {
		a, b := point{x, y}, point{x + 10, y}
		c, d := point{x + 10, y + 10}, point{x, y + 10}
		return []piece{{a, b}, {b, c}, {c, d}, {d, a}}
	}
	crossing := [][]piece{square(0, 0), square(5, 5)}
	if _, err := solidsOf(crossing, 1); !errors.Is(err, errCrossed) {
		t.Errorf("overlapping squares: %v", err)
	}
	apart := [][]piece{square(0, 0), square(20, 0)}
	if got, err := solidsOf(apart, 1); err != nil || len(got) != 2 {
		t.Errorf("squares apart: %d solids, %v", len(got), err)
	}
}

// a diamond with a diamond hole, the hole's rightmost point level with
// the outer's first point, is bridged to that point: the ray from the
// hole meets the outer exactly at the corner its first step starts.
func TestABridgeToTheRingsFirstCorner(t *testing.T) {
	outer := ring{{20, 5}, {10, 15}, {0, 5}, {10, -5}}
	hole := ring{{12, 5}, {10, 3}, {8, 5}, {10, 7}}
	if got := seen(outer, point{12, 5}); got != 0 {
		t.Errorf("the hole sees corner %d, not the first", got)
	}
	tris, err := faces(solid{outer: outer, holes: []ring{hole}})
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := covered(tris); math.Abs(got-(outer.area()+hole.area())) >
		1e-9 {
		t.Errorf("faces %v of %v", got, outer.area()+hole.area())
	}
}

// a point outside a ring to its right sees nothing of it: no step is to
// its right, and the first place is all there is to give.
func TestAPointRightOfARingSeesNothing(t *testing.T) {
	square := ring{{0, 0}, {10, 0}, {10, 10}, {0, 10}}
	if got := seen(square, point{50, 5}); got != 0 {
		t.Errorf("a point right of the square sees %d", got)
	}
}

// of the places at a point, the one whose corner opens toward m is
// taken; where none opens toward it, the place asked for is kept.
func TestTheCornerThatOpensTowardMIsTaken(t *testing.T) {
	square := ring{{0, 0}, {10, 0}, {10, 10}, {0, 10}}
	if got := opening(square, 2, point{5, 5}); got != 2 {
		t.Errorf("the corner at 10, 10 opens to the middle: %d", got)
	}
	if got := opening(square, 2, point{15, 15}); got != 2 {
		t.Errorf("no corner opens toward 15, 15, and %d is given", got)
	}
	if opens(square[1], square[2], square[3], point{15, 15}) {
		t.Error("the square's corner opens outward")
	}
}

// a solid whose face cannot be cut, an outer turned the wrong way, is
// an error and not a mesh with its face missing.
func TestAFaceThatCannotBeCutIsAnError(t *testing.T) {
	backward := solid{outer: ring{{0, 0}, {0, 10}, {10, 10}, {10, 0}}}
	var made builder
	err := shape{scale: 1, depth: 1}.solid(&made, backward)
	if !errors.Is(err, errNoEar) {
		t.Errorf("a backward outer: %v", err)
	}
}

// a solid is kept only while its outer turns counterclockwise and its
// holes clockwise inside it.
func TestASolidIsKeptOnlyTheRightWayRound(t *testing.T) {
	outer := ring{{0, 0}, {10, 0}, {10, 10}, {0, 10}}
	hole := ring{{2, 2}, {2, 8}, {8, 8}, {8, 2}}
	for name, each := range map[string]struct {
		one  solid
		want bool
	}{"right": {solid{outer: outer, holes: []ring{hole}}, true},
		"outer backward": {solid{outer: outer.reversed()}, false},
		"hole forward": {solid{outer: outer,
			holes: []ring{hole.reversed()}}, false},
	} {
		if got := kept(each.one); got != each.want {
			t.Errorf("%s: kept %v", name, got)
		}
	}
}

// a triangle moved in further than it is wide turns inside out, step by
// step, until fewer than three steps are left: errBevel.
func TestATriangleMovedInTooFarIsRefused(t *testing.T) {
	triangle := ring{{0, 0}, {10, 0}, {5, 8}}
	if _, _, err := insetRing(triangle, 6); !errors.Is(err, errBevel) {
		t.Errorf("a triangle moved in by 6: %v", err)
	}
}
