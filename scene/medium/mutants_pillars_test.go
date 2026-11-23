package medium

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
)

// mutantsPillarsFirst is the place of the first of a list of columns or tips.
const mutantsPillarsFirst = nought // defined: the first place in a list

// mutantsPillarsStreaks is the table of points in the filaments test: where
// the streak noise of a single pillar is deepest.
const mutantsPillarsStreaks = "testdata/mutants-pillars-streaks.csv"

// mutantsPillarsStanding is the pillars most of these tests stand in their
// box: three columns of a height and a width, Thick, with a seed.
func mutantsPillarsStanding(t *testing.T) Pillars {
	t.Helper()
	return Pillars{Count: pickedCount(t, "mutants-pillars-count"),
		Height: picked(t, "mutants-pillars-height"),
		Width:  picked(t, "mutants-pillars-width"),
		Thick:  picked(t, "mutants-pillars-thick"),
		Seed:   uint64(pickedCount(t, "mutants-pillars-seed"))}
}

// mutantsPillarsSingle is one pillar of the standing size, Thick one, with
// the seed given.
func mutantsPillarsSingle(t *testing.T, seed uint64) Pillars {
	t.Helper()
	standing := mutantsPillarsStanding(t)
	return Pillars{Count: unity, Height: standing.Height,
		Width: standing.Width, Thick: unity, Seed: seed}
}

// the wall along the bottom is Thick, a fifth of the height up and a little
// ragged: on the middle of the base the noise that ragged it is zero (it is
// read at the middle of space there, whatever the height), and the wall is
// whole up to 0.12 of the height, where it begins to ease.
//
// no pillar is above one, so the density there is Thick. pillars of no count,
// height or width are none; of one of any of them, they are not.
func TestPillarsAreAWallAtTheFoot(t *testing.T) {
	standing := mutantsPillarsStanding(t)
	lone, low, narrow := standing, standing, standing
	lone.Count, low.Height, narrow.Width = unity, unity, unity
	share := picked(t, "mutants-pillars-wall-share")
	for _, pillars := range []Pillars{standing, lone, low, narrow} {
		at := mutantsUp(share * pillars.Height)
		if got := pillars.Density(at); got != standing.Thick {
			t.Errorf("%+v: the wall is %v, want %v", pillars, got,
				standing.Thick)
		}
	}
	heightless, widthless, countless := standing, standing, standing
	heightless.Height, widthless.Width, countless.Count = nought, nought, nought
	for _, pillars := range []Pillars{heightless, widthless, countless} {
		if got := pillars.Density(mutantsOrigin()); got != nought {
			t.Errorf("%+v: %v at the foot, want none", pillars, got)
		}
		at := mutantsUp(share * pillars.Height)
		if got := pillars.Density(at); got != nought {
			t.Errorf("%+v: %v in the wall, want none", pillars, got)
		}
	}
}

// the box the pillars stand in is closed: on the floor of it is the wall,
// Thick, and at its top, and at its sides, the fog, a three-thousandth of
// Thick, where no column reaches (a column's height is under the whole).
func TestThePillarsBoxIsClosed(t *testing.T) {
	pillars := mutantsPillarsStanding(t)
	if got := pillars.Density(mutantsOrigin()); got != pillars.Thick {
		t.Errorf("on the floor the wall is %v, want %v", got, pillars.Thick)
	}
	top, side := pillars.Height, pillars.Width/double
	want := pillars.Thick * picked(t, "mutants-pillars-fog")
	fine := picked(t, "mutants-pillars-tolerance-fog")
	for _, at := range []maths.Vec{maths.V(nought, top, nought),
		maths.V(side, top, nought), maths.V(-side, top, nought),
		maths.V(nought, top, side), maths.V(nought, top, -side)} {
		if got := pillars.Density(at); !closeTo(got, want, fine) {
			t.Errorf("at %v the fog is %v, want %v", at, got, want)
		}
	}
}

// mutantsPillarsTipWant is the density a column's tip has by the working in
// the comment of the tip test: the larger of the fog and the column's edge
// shifted by the noise at the tip.
func mutantsPillarsTipWant(t *testing.T, pillars Pillars,
	tip maths.Vec) float64 {
	t.Helper()
	base := pillars.Width / picked(t, "mutants-pillars-base-divisor")
	scale := picked(t, "mutants-pillars-tip-scale") / base
	offset := uint64(pickedCount(t, "mutants-pillars-noise-seed-offset"))
	octaves := pickedCount(t, "mutants-pillars-octaves")
	grain := noise.Fractal(pillars.Seed+offset, tip.X*scale, tip.Y*scale,
		tip.Z*scale, octaves)
	shift := (grain - halfway) * picked(t, "mutants-pillars-build") * base
	edge := picked(t, "mutants-pillars-edge")
	body := unity - maths.Smooth(-edge*base, nought, shift)
	return math.Max(picked(t, "mutants-pillars-fog"), body)
}

// at the very height of a column's tip its dome has closed to a point,
// so the edge there is the noise's shift alone: the distance from the
// middle moves by (n - 0.5) * 0.9 * base, and the column is 1 - Smooth(
// -0.3 base, 0, that) where the point is at its middle.
//
// for a single pillar the tip is Thick times the larger of that and the
// fog. n is four octaves of maths/noise at the tip scaled by 1.5 over the
// base, for the seed plus ten. over fifty seeds the tip is the fog for
// some and well above it for others.
func TestAColumnsTipIsItsNoise(t *testing.T) {
	above := nought
	aboveFog := picked(t, "mutants-pillars-above-fog")
	tolerance := picked(t, "mutants-pillars-tolerance-tip")
	for seed := range uint64(pickedCount(t, "mutants-pillars-tip-seeds")) {
		pillars := mutantsPillarsSingle(t, seed)
		tip := pillars.Tips()[mutantsPillarsFirst]
		want := mutantsPillarsTipWant(t, pillars, tip)
		if want > aboveFog {
			above++
		}
		if got := pillars.Density(tip); !closeTo(got, want, tolerance) {
			t.Errorf("seed %d: the tip is %v, want %v", seed, got, want)
		}
	}
	if least := pickedCount(t, "mutants-pillars-tips-least"); above < least {
		t.Fatalf("only %d of fifty tips were above the fog", above)
	}
}

// a column of gas is thinned by its filaments and never thickened, so a
// negative Filaments is none: over a spread of points through the columns,
// a Filaments of minus one is the density of none.
func TestANegativeFilamentsIsNone(t *testing.T) {
	plain := mutantsPillarsStanding(t)
	plain.Thick = unity
	plain.Seed = uint64(pickedCount(t, "mutants-pillars-filaments-seed"))
	negative := plain
	negative.Filaments = -unity
	divisor := picked(t, "mutants-pillars-base-divisor")
	base := plain.Width / (float64(plain.Count) * divisor)
	streamX := uint64(pickedCount(t, "mutants-pillars-stream-x"))
	streamY := uint64(pickedCount(t, "mutants-pillars-stream-y"))
	streamZ := uint64(pickedCount(t, "mutants-pillars-stream-z"))
	for _, column := range plain.columns() {
		for index := range int64(pickedCount(t, "mutants-pillars-samples")) {
			at := maths.V(column.x+(noise.Hash(streamX, index)-halfway)*base,
				noise.Hash(streamY, index)*column.height,
				column.z+(noise.Hash(streamZ, index)-halfway)*base)
			thinned, whole := negative.Density(at), plain.Density(at)
			if thinned != whole {
				t.Fatalf("at %v a Filaments of minus one is %v, none %v", at,
					thinned, whole)
			}
		}
	}
}

// the filaments take Filaments of the gas where the streak is deepest and
// no more: a Filaments of a half leaves half of a column's whole gas
// there.
//
// the streak noise is below minus a half at these two points in a single
// pillar's body, where the streak is deepest. they were found by search
// and the test checks that they still are, and that the body is whole.
func TestFilamentsTakeNoMoreThanTheirShare(t *testing.T) {
	plainSeed := uint64(pickedCount(t, "mutants-pillars-filaments-seed"))
	plain := mutantsPillarsSingle(t, plainSeed)
	half := plain
	half.Filaments = halfway
	base := plain.columns()[mutantsPillarsFirst].base
	scale := picked(t, "mutants-pillars-streak-scale") / base
	stretch := picked(t, "mutants-pillars-streak-stretch")
	seed := plain.Seed + uint64(pickedCount(t,
		"mutants-pillars-noise-seed-offset")) + uint64(pickedCount(t,
		"mutants-pillars-streak-seed-offset"))
	octaves := pickedCount(t, "mutants-pillars-octaves")
	tolerance := picked(t, "mutants-pillars-tolerance-share")
	for _, row := range readSheet(t, mutantsPillarsStreaks) {
		at := maths.V(cell(t, row, "x"), cell(t, row, "y"), cell(t, row, "z"))
		streak := noise.Fractal(seed, at.X*scale, at.Y*scale/stretch,
			at.Z*scale, octaves)
		if streak >= -halfway {
			t.Fatalf("at %v the streak noise is %v, not below minus a half",
				at, streak)
		}
		if got := plain.Density(at); got != unity {
			t.Fatalf("at %v the column is %v, not whole", at, got)
		}
		if got := half.Density(at); !closeTo(got, halfway, tolerance) {
			t.Errorf("at %v half the filaments leave %v, want 0.5", at, got)
		}
	}
}
