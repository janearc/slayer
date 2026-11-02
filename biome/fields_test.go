package biome_test

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"pgregory.net/rapid"
)

// point is a place drawn within the test's reach of the origin.
func point(draw *rapid.T) maths.Vec {
	reach := picked(draw, "reach")
	along := rapid.Float64Range(-reach, reach)
	return maths.V(along.Draw(draw, "x"), along.Draw(draw, "y"),
		along.Draw(draw, "z"))
}

// differs is how many of the test map's middles two readings disagree at.
func differs(test reporter, one, other func(maths.Vec) float64) int {
	test.Helper()
	tiling, apart := flat(test), nought
	for _, tile := range tiling.Tiles() {
		at := tiling.Centre(tile)
		if one(at) != other(at) {
			apart++
		}
	}
	return apart
}

// the fields hold no state: read twice at a point, each is what it was.
func TestAFieldIsTheSameEveryTime(t *testing.T) {
	rapid.Check(t, sameEveryTime)
}

func sameEveryTime(draw *rapid.T) {
	made, at := fields(draw, nought), point(draw)
	if made.Height(at) != made.Height(at) || made.Wet(at) != made.Wet(at) {
		draw.Fatalf("at %v a field read twice is two numbers", at)
	}
}

// anti-tests: another seed is another world, in its height and its wet.
func TestAnotherSeedIsAnotherField(t *testing.T) {
	made, other := fields(t, nought), fields(t, nought)
	other.Seed = uint64(picked(t, "other-seed"))
	if differs(t, made.Height, other.Height) == nought {
		t.Error("two seeds make one height")
	}
	if differs(t, made.Wet, other.Wet) == nought {
		t.Error("two seeds make one wet")
	}
}

// a field of a scale at a point is the field of a scale of one at the
// point shrunk by the scale: the scale is how wide the field is.
func TestAScaleIsHowWideTheFieldIs(t *testing.T) {
	rapid.Check(t, scaleIsWidth)
}

func scaleIsWidth(draw *rapid.T) {
	within := picked(draw, "within")
	scale := rapid.Float64Range(picked(draw, "least-scale"),
		picked(draw, "most-scale")).Draw(draw, "scale")
	at := point(draw)
	wide, unit := fields(draw, nought), fields(draw, nought)
	wide.Scale, unit.Scale = scale, unity
	shrunk := maths.V(at.X/scale, at.Y/scale, at.Z/scale)
	if math.Abs(wide.Height(at)-unit.Height(shrunk)) > within {
		draw.Fatalf("at %v, scale %v: height %v, not %v", at, scale,
			wide.Height(at), unit.Height(shrunk))
	}
	if math.Abs(wide.Wet(at)-unit.Wet(shrunk)) > within {
		draw.Fatalf("at %v, scale %v: wet %v, not %v", at, scale,
			wide.Wet(at), unit.Wet(shrunk))
	}
}

// anti-tests: a scale is not ignored. The test map's scale is not a
// scale of one, in its height or its wet.
func TestAScaleChangesTheField(t *testing.T) {
	made, unit := fields(t, nought), fields(t, nought)
	unit.Scale = unity
	if differs(t, made.Height, unit.Height) == nought {
		t.Error("the height is the same at a scale of one")
	}
	if differs(t, made.Wet, unit.Wet) == nought {
		t.Error("the wet is the same at a scale of one")
	}
}

// a scale of zero, or one below it, is a scale of one.
func TestNoScaleIsAScaleOfOne(t *testing.T) {
	unit := fields(t, nought)
	unit.Scale = unity
	for _, scale := range []float64{nought, -picked(t, "scale")} {
		none := fields(t, nought)
		none.Scale = scale
		if differs(t, none.Height, unit.Height) != nought ||
			differs(t, none.Wet, unit.Wet) != nought {
			t.Errorf("a scale of %v is not a scale of one", scale)
		}
	}
}

// anti-tests: a scale between zero and one is its own and not one.
func TestASmallScaleIsNotOne(t *testing.T) {
	small, unit := fields(t, nought), fields(t, nought)
	small.Scale, unit.Scale = picked(t, "least-scale"), unity
	if differs(t, small.Height, unit.Height) == nought {
		t.Errorf("a scale of %v is a scale of one", small.Scale)
	}
}

// the fields stay within one either side of zero, wherever they are read.
func TestTheFieldsStayWithinOne(t *testing.T) {
	rapid.Check(t, withinOne)
}

func withinOne(draw *rapid.T) {
	made, at := fields(draw, nought), point(draw)
	height, wet := made.Height(at), made.Wet(at)
	if !(math.Abs(height) <= unity) || !(math.Abs(wet) <= unity) {
		draw.Fatalf("at %v the height is %v and the wet %v", at, height,
			wet)
	}
}

// anti-tests: the wet is a field of its own, not the height again.
func TestTheWetIsNotTheHeight(t *testing.T) {
	made := fields(t, nought)
	if differs(t, made.Height, made.Wet) == nought {
		t.Error("the wet is the height")
	}
}

// how far a point stands above the sea is its height less the sea where it is
// higher, and zero where it is not.
func TestAboveIsTheHeightOverTheSea(t *testing.T) {
	rapid.Check(t, aboveTheSea)
}

func aboveTheSea(draw *rapid.T) {
	made := fields(draw, rapid.Float64Range(-unity, unity).Draw(draw, "sea"))
	at := point(draw)
	height, got := made.Height(at), made.Above(at)
	switch {
	case height > made.Sea && got != height-made.Sea:
		draw.Fatalf("at %v, %v over a sea of %v, is %v above it", at,
			height, made.Sea, got)
	case height <= made.Sea && got != nought:
		draw.Fatalf("at %v, %v under a sea of %v, is %v above it", at,
			height, made.Sea, got)
	}
}

// anti-tests: a higher sea lifts nothing higher above it.
func TestRaisingTheSeaRaisesNothing(t *testing.T) {
	rapid.Check(t, raisingRaisesNothing)
}

func raisingRaisesNothing(draw *rapid.T) {
	sea := rapid.Float64Range(-unity, unity).Draw(draw, "sea")
	rise := rapid.Float64Range(nought, unity).Draw(draw, "rise")
	low, high := fields(draw, sea), fields(draw, sea+rise)
	at := point(draw)
	if high.Above(at) > low.Above(at) {
		draw.Fatalf("at %v a sea %v higher leaves it %v above, not %v", at,
			rise, high.Above(at), low.Above(at))
	}
}
