package texture

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// the colours and the numbers the cover tests are drawn with, all picked
var (
	coveredRock  = maths.RGB(0.2, 0.2, 0.2)
	coveredSand  = maths.RGB(0.7, 0.6, 0.3)
	coveredGrass = maths.RGB(0.1, 0.4, 0.1)
)

const (
	// coveredSea is where the sea stands in the tests' own space. picked.
	coveredSea = 50.0
	// coveredBlend is the blend of every range the tests make. picked.
	coveredBlend = 4.0
	// coveredTolerance is how near two lights must be to be one. picked.
	coveredTolerance = 1e-9
)

// level is the unit normal of flat ground, and steep that of a wall's
// foot leaning one run for each rise.
var (
	coveredLevel = maths.V(0, 1, 0)
	coveredSteep = maths.V(math.Sqrt2/2, math.Sqrt2/2, 0)
)

// coveredShore is a cover of rock with sand low and gentle, and grass
// high and gentle, as a beach is cut out of a cliff coast.
func coveredShore() Cover {
	gentle := NewRange(math.Inf(-1), 0.5, coveredBlend/10)
	lowRange := NewRange(math.Inf(-1), 6, coveredBlend)
	highRange := NewRange(20, math.Inf(1), coveredBlend)
	return NewCover(coveredRock, coveredSea).
		Over(coveredSand, lowRange, gentle).
		Over(coveredGrass, highRange, gentle)
}

// sameLight is whether two lights are the same to the tests' tolerance.
func sameLight(one, other maths.Light) bool {
	return math.Abs(one.R-other.R) < coveredTolerance &&
		math.Abs(one.G-other.G) < coveredTolerance &&
		math.Abs(one.B-other.B) < coveredTolerance
}

// Low gentle ground is sand, high gentle ground is grass, and steep ground
// is rock at any height.
func TestCoverIsSandLowGrassHighRockSteep(t *testing.T) {
	cover := coveredShore()
	for name, want := range map[string]struct {
		height float64
		normal maths.Vec
		colour maths.Light
	}{
		"beach":  {coveredSea + 1, coveredLevel, coveredSand},
		"meadow": {coveredSea + 40, coveredLevel, coveredGrass},
		"cliff":  {coveredSea + 1, coveredSteep, coveredRock},
		"crag":   {coveredSea + 40, coveredSteep, coveredRock},
	} {
		got := cover.Facing(maths.V(0, want.height, 0), maths.Vec{},
			want.normal)
		if !sameLight(got, want.colour) {
			t.Errorf("%s: %v, want %v", name, got, want.colour)
		}
	}
}

// A blend is centred on the edge it softens: the colour at the edge is
// half the one on either side, and the weight is eased: a quarter of the
// way across the blend the sand has the smoothstep's share, 0.15625 from
// the far side, which a straight ramp (0.25) does not give. Through the
// blend the red falls strictly, step after step.
func TestCoverBlendsAcrossItsWidthCentredOnTheEdge(t *testing.T) {
	lowRange := NewRange(math.Inf(-1), 6, coveredBlend)
	anyRange := NewRange(math.Inf(-1), math.Inf(1), 0)
	cover := NewCover(coveredRock, coveredSea).Over(coveredSand,
		lowRange, anyRange)
	mixed := func(share float64) maths.Light {
		return coveredRock.Scale(1 - share).Add(coveredSand.Scale(share))
	}
	// picked: the smoothstep of one quarter is 0.25 * 0.25 * (3 - 0.5)
	const quarter = 0.15625
	for name, want := range map[string]struct {
		height float64
		share  float64
	}{
		"the edge":            {6, 0.5},
		"a quarter in, below": {5, 1 - quarter},
		"a quarter in, above": {7, quarter},
	} {
		got := cover.At(maths.V(0, coveredSea+want.height, 0))
		if !sameLight(got, mixed(want.share)) {
			t.Errorf("%s: %v, want the sand's share %v: %v", name, got,
				want.share, mixed(want.share))
		}
	}
	last := coveredSand.R
	for step := 1; step <= 14; step++ {
		height := coveredSea + 4 + float64(step)/4
		now := cover.At(maths.V(0, height, 0)).R
		if !(now < last) {
			t.Errorf("the red did not fall going up at step %d: %v after %v",
				step, now, last)
		}
		last = now
	}
}

// A layer that says only a height, or only a slope, is laid by that rule
// alone: the other range holds every number and does not take it away.
func TestCoverLayerWithOneRuleHoldsByThatRuleAlone(t *testing.T) {
	anything := NewRange(math.Inf(-1), math.Inf(1), 0)
	heightRange := NewRange(math.Inf(-1), 6, coveredBlend)
	byHeight := NewCover(coveredRock, coveredSea).Over(coveredSand,
		heightRange, anything)
	slopeRange := NewRange(math.Inf(-1), 0.5, coveredBlend/10)
	bySlope := NewCover(coveredRock, coveredSea).Over(coveredGrass, anything,
		slopeRange)
	for name, want := range map[string]struct {
		cover  Cover
		height float64
		normal maths.Vec
		colour maths.Light
	}{
		"low, level, by height":  {byHeight, 1, coveredLevel, coveredSand},
		"low, steep, by height":  {byHeight, 1, coveredSteep, coveredSand},
		"high, level, by height": {byHeight, 40, coveredLevel, coveredRock},
		"high, level, by slope":  {bySlope, 40, coveredLevel, coveredGrass},
		"low, level, by slope":   {bySlope, -40, coveredLevel, coveredGrass},
		"high, steep, by slope":  {bySlope, 40, coveredSteep, coveredRock},
	} {
		got := want.cover.Facing(maths.V(0, coveredSea+want.height, 0),
			maths.Vec{}, want.normal)
		if !sameLight(got, want.colour) {
			t.Errorf("%s: %v, want %v", name, got, want.colour)
		}
	}
}

// Anti: a cover with no layer is its base everywhere, whatever the height
// and the lean.
func TestCoverWithNoLayerIsItsBaseEverywhere(t *testing.T) {
	cover := NewCover(coveredRock, coveredSea)
	for _, height := range []float64{-300, 0, coveredSea, 900} {
		for _, normal := range []maths.Vec{coveredLevel, coveredSteep} {
			if got := cover.Facing(maths.V(0, height, 0), maths.Vec{},
				normal); got != coveredRock {
				t.Errorf("height %v: %v, want the base", height, got)
			}
		}
	}
}

// Anti: laying a layer leaves the cover it is laid on as it was, and two
// covers laid from one are each their own. The parent has three layers,
// so a list grown by append holds room for a fourth: an Over that wrote
// into that room would give both children the one laid last.
func TestOverLeavesTheCoverItIsCalledOnAlone(t *testing.T) {
	high := func(from float64) Range {
		return NewRange(from, math.Inf(1), 0)
	}
	anything := NewRange(math.Inf(-1), math.Inf(1), 0)
	parent := NewCover(coveredRock, coveredSea).
		Over(coveredRock, high(100), anything).
		Over(coveredRock, high(200), anything).
		Over(coveredRock, high(300), anything)
	sandy := parent.Over(coveredSand, high(400), anything)
	grassy := parent.Over(coveredGrass, high(400), anything)
	top := maths.V(0, coveredSea+500, 0)
	if got := sandy.At(top); got != coveredSand {
		t.Errorf("the first child paints %v at its top, want sand", got)
	}
	if got := grassy.At(top); got != coveredGrass {
		t.Errorf("the second child paints %v at its top, want grass", got)
	}
	if got := parent.At(top); got != coveredRock {
		t.Errorf("the parent paints %v at its top after two Overs", got)
	}
}

// The slope of a normal is rise over run, a wall is infinite, and the
// colour does not depend on which side of the ground the eye is on.
func TestCoverReadsTheLeanOfANormal(t *testing.T) {
	if got := leanOf(coveredLevel); got != 0 {
		t.Errorf("level ground leans %v", got)
	}
	if got := leanOf(coveredSteep); !(math.Abs(got-1) <= coveredTolerance) {
		t.Errorf("a forty-five degree face leans %v, want 1", got)
	}
	if got := leanOf(maths.V(1, 0, 0)); !math.IsInf(got, 1) {
		t.Errorf("a wall leans %v, want infinity", got)
	}
	cover := coveredShore()
	under := maths.V(0, -1, 0)
	if cover.Facing(maths.V(0, coveredSea+1, 0), maths.Vec{}, under) !=
		cover.Facing(maths.V(0, coveredSea+1, 0), maths.Vec{}, coveredLevel) {
		t.Error("the underside of level ground was painted unlike its top")
	}
}

// A range with no ends holds every number whole; a range with one end holds
// everything on its side of it.
func TestRangeWeighsWhatItHolds(t *testing.T) {
	everything := NewRange(math.Inf(-1), math.Inf(1), 0)
	for _, value := range []float64{-1e9, 0, 3, 1e9} {
		if got := everything.Weight(value); got != 1 {
			t.Errorf("a range with no ends weighs %v at %v", got, value)
		}
	}
	above := NewRange(10, math.Inf(1), coveredBlend)
	if above.Weight(0) != 0 || above.Weight(100) != 1 {
		t.Errorf("a range above ten weighs %v below and %v far above",
			above.Weight(0), above.Weight(100))
	}
	between := NewRange(10, 20, coveredBlend)
	if between.Weight(15) != 1 || between.Weight(5) != 0 ||
		between.Weight(25) != 0 {
		t.Errorf("a range from ten to twenty weighs %v inside, %v and %v out",
			between.Weight(15), between.Weight(5), between.Weight(25))
	}
}
