package trace

import (
	"math"
	"testing"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/strategies"
	"github.com/janearc/slayer/scene/thing"
)

// splitBound is the most one pick weighs, in times the light of the lamps
// its node holds: the square of one plus lampSplitSpread. A step's light
// is at least its node's, so it is the most a pick weighs in times the
// step's.
var splitBound = (1 + lampSplitSpread) * (1 + lampSplitSpread)

// splitStarts is how many starts the tests of the split average over.
const splitStarts = 1 << 14

// treed is a scene over lamps with the lamp tree it would be staged with.
func treed(lamps []thing.Lamp) *staged {
	scene := &staged{lamps: lamps}
	scene.tree, scene.others = treeOf(lamps)
	return scene
}

// nearAmongFar is a scene of four lamps and a tree of them built by hand,
// and the point to light it at is the origin: one lamp thirty metres off,
// two eight hundred off on either side of it, and one of three hundred
// times the strength three kilometres off. The origin is inside the box of
// the two nearer lamps' node, which it is far from the far ones' as well,
// and a walk from the root takes the near lamp one time in eighty.
func nearAmongFar(t *testing.T) *staged {
	t.Helper()
	spots := []struct {
		at       maths.Vec
		strength float64
	}{
		{maths.V(30, 0, 0), 1},
		{maths.V(-800, 800, 0), 1},
		{maths.V(800, -800, 0), 1},
		{maths.V(3000, 0, 0), 300},
	}
	var lamps []thing.Lamp
	var places [][3]float64
	var powers []float64
	var standing []treeLamp
	for index, spot := range spots {
		lamps = append(lamps, emit.Lit{At: spot.at, Beyond: 0.1,
			Strength: maths.Grey(spot.strength)})
		places = append(places, [3]float64{spot.at.X, spot.at.Y, spot.at.Z})
		powers = append(powers, spot.strength)
		standing = append(standing, treeLamp{index: index, at: spot.at,
			power: spot.strength})
	}
	shared, err := strategies.BuildLightTree(places, powers, lampOrder)
	if err != nil {
		t.Fatal(err)
	}
	return &staged{lamps: lamps, tree: &lampTree{shared: shared,
		lamps: standing}}
}

// luminanceAt is the light of every lamp of a scene at a point, summed.
func luminanceAt(scene *staged, point maths.Vec) float64 {
	sum := 0.0
	for _, lamp := range scene.lamps {
		_, _, light := lamp.Light(point, maths.Vec{})
		sum += light.Luminance()
	}
	return sum
}

// TestNoPickOfTheSplitWeighsMoreThanFourTimesTheStepsLight is the anti-test
// of the walk from the root: with the near lamp inside a wide box a pick
// of it, by its light times its count, is far over the light of the step,
// and it must not be. With the tree split every lamp here is taken, once.
func TestNoPickOfTheSplitWeighsMoreThanFourTimesTheStepsLight(t *testing.T) {
	scene := nearAmongFar(t)
	point := maths.V(0, 0, 0)
	total := luminanceAt(scene, point)
	for each := range splitStarts {
		start := (float64(each) + 0.5) / splitStarts
		picks := scene.splitFrom(nil, point, maths.Vec{}, start,
			lampSplitMost)
		for _, one := range picks {
			if weight := one.light.Luminance() * one.counts; weight >
				splitBound*total {
				t.Fatalf("start %v: lamp %d weighs %g, the step's light is "+
					"%g", start, one.index, weight, total)
			}
		}
		if len(picks) != len(scene.lamps) {
			t.Fatalf("start %v: %d lamps taken of %d", start, len(picks),
				len(scene.lamps))
		}
	}
}

// powerWalk is a faulty pick made on purpose: one lamp of the tree walked
// to from its root by the lamps' power alone, with no regard for where
// the point is, and counted one over its chance. It is the fault the
// split is there to avoid, kept here so that the controls of the split's
// tests do not lean on how well or badly the tracer's own walk does.
//
// Among nearAmongFar's lamps, of power 1, 1, 1 and 300, it takes the
// near lamp one time in 303 and then counts it 303 times.
func powerWalk(scene *staged, point maths.Vec, start float64) []lit {
	lamp, chance := strategies.WalkByPower(scene.tree.shared, 0, start)
	taken := scene.tree.lamps[lamp]
	toward, far, light := scene.lamps[taken.index].Light(point, maths.Vec{})
	return []lit{{index: taken.index, toward: toward, far: far,
		light: light, counts: 1 / chance}}
}

// TestAWalkByPowerAloneTakesTheNearLampFarOverTheStepsLight is the control
// of the test above: the bound tells a pick that weighs too much. A walk
// by power alone takes the near lamp, which sends the origin 1/900, one
// time in 303 and counts it 303 times: 0.337, where the step's whole
// light is 0.00115 and the bound four times that.
func TestAWalkByPowerAloneTakesTheNearLampFarOverTheStepsLight(t *testing.T) {
	scene := nearAmongFar(t)
	point := maths.V(0, 0, 0)
	total := luminanceAt(scene, point)
	most := 0.0
	for each := range splitStarts {
		start := (float64(each) + 0.5) / splitStarts
		for _, one := range powerWalk(scene, point, start) {
			most = math.Max(most, one.light.Luminance()*one.counts)
		}
	}
	if !(most > splitBound*total) || !within(most, 303.0/900, 1e-9) {
		t.Errorf("the heaviest pick by power weighs %g, the step's light "+
			"is %g", most, total)
	}
}

// TestTheSplitSumsTheLampsOnAverage says that the picks of the split,
// each counted as it is, sum on average over their starts to the light of
// every standing lamp at the point, as the walks' do.
func TestTheSplitSumsTheLampsOnAverage(t *testing.T) {
	lamps := streetOfWindows(t)
	scene := treed(lamps)
	point, facing := maths.V(10, 1, 5), maths.V(0, 0, -1)
	every := 0.0
	for _, lamp := range lamps[1:] {
		_, _, light := lamp.Light(point, facing)
		every += light.R
	}
	mean := 0.0
	for each := range splitStarts {
		for _, one := range scene.splitFrom(nil, point, facing,
			(float64(each)+0.5)/splitStarts, lampSplitMost) {
			mean += one.light.R * one.counts / splitStarts
		}
	}
	if math.Abs(mean-every) > 1e-2*every {
		t.Errorf("picks sum to %v on average, every lamp to %v", mean, every)
	}
}

// TestTheSplitSumsTheLampsAmongFarOnesExactly says that where the cut
// takes every lamp of the scene alone the sum is not an average but exact.
func TestTheSplitSumsTheLampsAmongFarOnesExactly(t *testing.T) {
	scene := nearAmongFar(t)
	point := maths.V(0, 0, 0)
	got := 0.0
	for _, one := range scene.splitFrom(nil, point, maths.Vec{}, 0.37,
		lampSplitMost) {
		got += one.light.Luminance() * one.counts
	}
	if want := luminanceAt(scene, point); math.Abs(got-want) > 1e-9*want {
		t.Errorf("the split sums to %v, every lamp to %v", got, want)
	}
}

// twoSuns is the street of windows with a second sun of another strength
// from another way, so that two lamps stand outside the tree and more than
// lampsMarched lamps are asked of the rest by worth.
func twoSuns(t *testing.T) []thing.Lamp {
	t.Helper()
	second, err := emit.NewDistant(maths.V(1, 1, 0).Unit(), maths.Grey(2))
	if err != nil {
		t.Fatal(err)
	}
	return append(streetOfWindows(t), second)
}

// distantTaken is how many times each distant lamp of a scene is among
// the lamps asked, and the count it has.
func distantTaken(scene *staged, asked []lit) (map[int]int, map[int]float64) {
	times, counts := map[int]int{}, map[int]float64{}
	for _, one := range asked {
		for _, other := range scene.others {
			if one.index == other && math.IsInf(one.far, 1) {
				times[one.index]++
				counts[one.index] += one.counts
			}
		}
	}
	return times, counts
}

// TestAStepTakesEveryDistantLampEveryTime says that at each of many
// points a step through a medium asks every sun, once, at a count of one,
// however many lamps stand outside the tree. A sun picked by worth is
// asked some steps and not others, which is noise in the light that is
// most of the haze's.
func TestAStepTakesEveryDistantLampEveryTime(t *testing.T) {
	scene := treed(twoSuns(t))
	if len(scene.others) != 2 {
		t.Fatalf("%d lamps outside the tree, want the two suns",
			len(scene.others))
	}
	for each := range 256 {
		point := maths.V(10+float64(each)*0.37, 1, 5)
		times, counts := distantTaken(scene, scene.marched(nil, point,
			maths.Vec{}, byLight))
		for _, sun := range scene.others {
			if times[sun] != 1 || counts[sun] != 1 {
				t.Fatalf("point %v: sun %d asked %d times, counted %v",
					point, sun, times[sun], counts[sun])
			}
		}
	}
}

// sunsByWorth is a faulty pick made on purpose: of a scene's lamps
// outside its tree, one picked by its light, as a step is not to pick
// its suns. It is made here so that the control below does not lean on
// what any pick of the tracer's own does with them.
func sunsByWorth(scene *staged, point maths.Vec) []lit {
	var suns []lit
	var worths []float64
	total := 0.0
	for _, index := range scene.others {
		toward, far, light := scene.lamps[index].Light(point, maths.Vec{})
		suns = append(suns, lit{index: index, toward: toward, far: far,
			light: light, counts: 1})
		worths = append(worths, light.Luminance())
		total += light.Luminance()
	}
	return picked(nil, suns, worths, total, placeShare(point), lampsMarched)
}

// TestAPickOfTheSunsByWorthLosesASunAtEveryStep is the control of the test
// above: the check tells a step that does not take every sun. A pick of
// one of the two suns by its light takes one at a point and never both.
func TestAPickOfTheSunsByWorthLosesASunAtEveryStep(t *testing.T) {
	scene := treed(twoSuns(t))
	for each := range 256 {
		point := maths.V(10+float64(each)*0.37, 1, 5)
		times, _ := distantTaken(scene, sunsByWorth(scene, point))
		if len(times) != 1 {
			t.Fatalf("point %v: %d suns asked, want one of two", point,
				len(times))
		}
	}
}

// TestAStepAsksNoMoreLampsThanItsBound says that a step asks at most the
// distant lamps and lampSplitMost lamps of the tree, at any point of the
// street and well off it: the hard bound on a step's shadow rays.
func TestAStepAsksNoMoreLampsThanItsBound(t *testing.T) {
	scene := treed(twoSuns(t))
	most := len(scene.others) + lampSplitMost
	for each := range 512 {
		point := maths.V(float64(each%37)*1.7-5, float64(each%11)*1.3,
			float64(each%7)*40-100)
		if got := len(scene.marched(nil, point, maths.Vec{},
			byLight)); got > most {
			t.Fatalf("point %v: %d lamps asked, bound %d", point, got, most)
		}
	}
}
