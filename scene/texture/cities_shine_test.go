package texture

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
)

var cityGlow = maths.RGB(1, 0.5, 0.25)

// shineSpecs is the rungs of a world with n of them: the coarse rung, a
// metro's heart and open sea, at n-2 or none, and the fine rung at n,
// whose one tile has the id given and is in a town at 35 degrees. a rung
// that is neither is junk.
//
// the fine rung is 0.5 across and a tenth from every edge, so a point is
// 0.1 / (0.5 * 0.5) = 0.4 of the way in, where Smooth(0, 0.6, 0.4) is
// (4/9)(3 - 4/3) = 20/27.
func shineSpecs(n, coarseNearest int, around map[int][]int, fineID int) []rungSpec {
	specs := make([]rungSpec, n+1)
	for i := range specs {
		specs[i] = junk
	}
	specs[max(0, n-2)] = coarseSpecs(1.2, coarseNearest, around)
	specs[n] = rungSpec{spacing: 0.5, edge: 0.1, nearest: fineID,
		centres: map[int]maths.Vec{fineID: town}}
	return specs
}

// cityWant is the light of a lit tile worked by hand: the glow, times the
// boost of a metro's heart (1 + 3*core) or 1, times a size of 0.4 +
// 0.6*hash, times the 20/27 of the edge, times the clump.
//
// the clump is 0.5 plus fractal noise at the point times three over the
// fine spacing of 0.5, which is 6, and not less than 0.2.
func cityWant(boost float64, id int64, p maths.Vec) maths.Light {
	size := 0.4 + 0.6*noise.Hash(citySeed+1, id)
	clump := 0.5 + noise.Fractal(citySeed+3, p.X*6, p.Y*6, p.Z*6, 2)
	return cityGlow.Scale(boost * size * (20.0 / 27) * math.Max(0.2, clump))
}

// cityWorld is a world of two rungs with the cities of these tests.
func cityWorld(specs []rungSpec, rungsN int, share float64) Cities {
	return Cities{Tiling: rungs{specs: specs}, Land: inland, Rungs: rungsN,
		Share: share, Metro: 10, Glow: cityGlow, Seed: citySeed}
}

// Which tiles are lit, and by how much, in a metro's heart, next to a
// metro and away from one. a fine tile is lit when its hash at seed 63
// is not past its share.
//
// in the heart the share is the greater of three times the share and
// core squared (8/13 squared is 0.3787); next to a metro it is three
// times; elsewhere, the share itself. each id is picked so its hash is
// between the bounds a changed sum would use.
func TestCitiesLightByTheHashAgainstTheShare(t *testing.T) {
	heartAt := map[int][]int{}
	nextTo := map[int][]int{1: {0}}
	apart := map[int][]int{1: {2}}
	for _, c := range []struct {
		why    string
		coarse int
		around map[int][]int
		id     int
		share  float64
		lo, hi float64 // the hash is between these
		lit    bool
		boost  float64
	}{
		{"heart: core squared over three shares", 0, heartAt, 108, 0.05, 0.15, 0.3787, true, 37.0 / 13},
		{"heart: three shares over core squared", 0, heartAt, 119, 0.2, 0.3787, 0.6, true, 37.0 / 13},
		{"heart: past three shares", 0, heartAt, 143, 0.2, 0.6, 0.8, false, 0},
		{"next to a metro: over a third of three shares", 1, nextTo, 106, 0.1, 0.0334, 0.3, true, 1},
		{"next to a metro: past three shares", 1, nextTo, 103, 0.1, 0.3, 0.4, false, 0},
		{"apart: past the share", 1, apart, 100, 0.1, 0.1, 0.3, false, 0},
		{"apart: under the share", 1, apart, 101, 0.1, 0, 0.1, true, 1},
	} {
		if h := noise.Hash(citySeed, int64(c.id)); !(h > c.lo && h < c.hi) {
			t.Fatalf("%s: the hash of %d is %v, not between %v and %v", c.why, c.id, h, c.lo, c.hi)
		}
		w := cityWorld(shineSpecs(2, c.coarse, c.around, c.id), 2, c.share)
		if got := (Cities{}).weight(town); got != 1 {
			t.Fatalf("the weight of the town was meant to be one and is %v", got)
		}
		got := w.At(pointOfThem)
		if !c.lit {
			if got != (maths.Light{}) {
				t.Errorf("%s: lit with %v", c.why, got)
			}
			continue
		}
		want := cityWant(c.boost, int64(c.id), pointOfThem)
		if !near(got, want) {
			t.Errorf("%s: %v, want %v", c.why, got, want)
		}
	}
}

// The clump is held where this point is read, between 0.2 and 1.4, so a
// change to either bound would show.
func TestCitiesClumpIsBetweenTheBoundsAtThePoint(t *testing.T) {
	clump := 0.5 + noise.Fractal(citySeed+3, 0.3*6, 0.4*6, 1.2*6, 2)
	if !(clump > 0.2 && clump < 1.4) {
		t.Errorf("the clump at the point is %v", clump)
	}
}

// A tile is lit when its hash is not past the share: a hash equal to the
// share is lit, and one a step past is not. the town's weight is one, so
// the share is the hash.
func TestCitiesLightAtTheEdgeOfTheShare(t *testing.T) {
	const id = 101
	h := noise.Hash(citySeed, id)
	apart := map[int][]int{1: {2}}
	specs := shineSpecs(2, 1, apart, id)
	if got := cityWorld(specs, 2, h).At(pointOfThem); !near(got, cityWant(1, id, pointOfThem)) {
		t.Errorf("a share equal to the hash: %v", got)
	}
	if got := cityWorld(specs, 2, math.Nextafter(h, 0)).At(pointOfThem); got != (maths.Light{}) {
		t.Errorf("a share a step under the hash: %v", got)
	}
}

// With no share the city is dark, even in a metro's heart where the core alone
// would light the tile; and no share is no share less than zero.
func TestCitiesWithNoShareAreDark(t *testing.T) {
	const id = 108
	if h := noise.Hash(citySeed, id); !(h > 0.15 && h < 0.3787) {
		t.Fatalf("the hash of %d is %v", id, h)
	}
	specs := shineSpecs(2, 0, map[int][]int{}, id)
	for _, share := range []float64{0, -0.5} {
		if got := cityWorld(specs, 2, share).At(pointOfThem); got != (maths.Light{}) {
			t.Errorf("a share of %v lit %v", share, got)
		}
	}
	if got := cityWorld(specs, 2, 0.05).At(pointOfThem); got == (maths.Light{}) {
		t.Error("the same tile with a share is dark")
	}
}

// Without a tiling or without land there is nothing to stand on, either
// alone.
func TestCitiesNeedATilingAndLand(t *testing.T) {
	specs := shineSpecs(2, 0, map[int][]int{}, 108)
	noTiling := cityWorld(specs, 2, 0.05)
	noTiling.Tiling = nil
	noLand := cityWorld(specs, 2, 0.05)
	noLand.Land = nil
	for name, c := range map[string]Cities{"no tiling": noTiling, "no land": noLand} {
		if got := c.At(pointOfThem); got != (maths.Light{}) {
			t.Errorf("%s: %v", name, got)
		}
	}
}

// The same heart is found with a world of one rung, where the coarse
// rung is the world's own, and of three, where it is the first below.
func TestCitiesFewAndManyRungsFindTheSameHeart(t *testing.T) {
	for _, n := range []int{1, 2, 3} {
		w := cityWorld(shineSpecs(n, 0, map[int][]int{}, 108), n, 0.05)
		want := cityWant(37.0/13, 108, pointOfThem)
		if got := w.At(pointOfThem); !near(got, want) {
			t.Errorf("rungs %d: %v, want %v", n, got, want)
		}
	}
}
