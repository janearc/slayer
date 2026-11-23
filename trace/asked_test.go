package trace

import (
	"math"
	"math/rand/v2"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// suns is a world with a floor, a ball resting over it, and so many
// suns of one colour from all over the sky above, each a different
// strength.
func suns(count int) *thing.World {
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Plane{},
		Material: &thing.Material{Colour: maths.Grey(0.5)}})
	sphereAt := resting(maths.V(0, 1.5, 0))
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1},
		Material: &thing.Material{Colour: maths.Grey(0.5)},
		Motion:   sphereAt})
	random := rand.New(rand.NewPCG(7, 11))
	for range count {
		way := maths.V(random.Float64()*2-1, 0.2+random.Float64(),
			random.Float64()*2-1).Unit()
		light := maths.Grey(0.1 + random.Float64())
		world.Lamp(thing.Sun(way, light))
	}
	return world
}

// levelWorth is what a lamp is worth to a level floor: its light by how
// squarely it lands.
func levelWorth(toward maths.Vec, light maths.Light) float64 {
	return light.Luminance() * math.Max(0, maths.Up.Dot(toward))
}

// evenSteps is how many times each of some lamps is taken by so many
// picks spaced evenly along their worths laid end to end, the first
// pick a share start of a step in: lamp by lamp, the picks that fall
// inside its own stretch of the worth, as a count by its place.
func evenSteps(worths []float64, start float64, asks int) map[int]int {
	total := 0.0
	for _, each := range worths {
		total += each
	}
	taken := map[int]int{}
	for pick := range asks {
		target, laid := (start+float64(pick))*total/float64(asks), 0.0
		for place, each := range worths {
			laid += each
			if target < laid {
				taken[place]++
				break
			}
		}
	}
	return taken
}

// sameTaking is whether the lamps asked are the lamps taken, each
// counted its picks times the worth of a step over its own worth.
func sameTaking(asked []lit, taken map[int]int, worths []float64,
	step float64) bool {
	if len(asked) != len(taken) {
		return false
	}
	for _, one := range asked {
		want := float64(taken[one.index]) * step / worths[one.index]
		if !within(one.counts, want, 1e-9*want) {
			return false
		}
	}
	return true
}

// twoHundred is a scene of two hundred suns, what each is worth to a
// level floor at a point of it, the worth of one of lampsAsked steps
// along them, and the point.
func twoHundred() (*staged, []float64, float64, maths.Vec) {
	scene, point := stage(suns(200), 0), maths.V(40, 0, 40)
	worths, total := make([]float64, len(scene.lamps)), 0.0
	for index, lamp := range scene.lamps {
		toward, _, light := lamp.Light(point, maths.Up)
		worths[index] = levelWorth(toward, light)
		total += worths[index]
	}
	return scene, worths, total / lampsAsked, point
}

// Past lampsEvery lamps a point asks lampsAsked of them: the lamps that
// even steps along the lamps' worth fall in, from a start the place
// gives, each counted its picks times a step over its own worth. Which
// lamps those are is worked here pick by pick and held to what the
// point asks, lamp for lamp.
//
// The sum of count times worth is the lamps' whole worth for any choice
// of lamps, since every count is a step over the worth: it is not what
// this holds.
func TestTwoHundredLampsAreAskedWhereEvenStepsFallAlongTheirWorth(
	t *testing.T) {
	scene, worths, step, point := twoHundred()
	taken := evenSteps(worths, placeShare(point), lampsAsked)
	asked := scene.asked(nil, point, maths.Up, levelWorth)
	if len(taken) < 2 || len(taken) > lampsAsked {
		t.Fatalf("the test's own steps fall in %d lamps", len(taken))
	}
	if !sameTaking(asked, taken, worths, step) {
		t.Errorf("the point asks %+v, and even steps take %v", asked, taken)
	}
}

// Anti: the same lamps picked from another start are told. Steps begun
// half a step on from the place's own fall in other lamps, and the
// check says the point did not ask those.
func TestLampsTakenFromAnotherStartAreNotTheLampsAsked(t *testing.T) {
	scene, worths, step, point := twoHundred()
	other := math.Mod(placeShare(point)+0.5, 1)
	asked := scene.asked(nil, point, maths.Up, levelWorth)
	if sameTaking(asked, evenSteps(worths, other, lampsAsked), worths,
		step) {
		t.Error("lamps taken from another start passed for the lamps " +
			"the point asks")
	}
}

// with nothing in the way, the lamps a point asks are worth, counted as
// asked, exactly what every lamp is worth; with lampsEvery or fewer
// every lamp is asked, once. Past lampsEvery that sum says nothing of
// which lamps were asked: see the test above.
func TestTheLampsAskedAreWorthEveryLamp(t *testing.T) {
	up := maths.V(0, 1, 0)
	worth := func(toward maths.Vec, light maths.Light) float64 {
		return light.Luminance() * math.Max(0, up.Dot(toward))
	}
	for _, count := range []int{3, lampsAsked, 12, lampsEvery} {
		scene := stage(suns(count), 0)
		point := maths.V(40, 0, 40)
		all, got := 0.0, 0.0
		for _, lamp := range scene.lamps {
			toward, _, light := lamp.Light(point, up)
			all += worth(toward, light)
		}
		asked := scene.asked(nil, point, up, worth)
		for _, one := range asked {
			got += one.counts * worth(one.toward, one.light)
			if count <= lampsEvery && one.counts != 1 {
				t.Errorf("%d lamps: lamp %d counts %g", count, one.index,
					one.counts)
			}
		}
		if count <= lampsEvery && len(asked) != count {
			t.Errorf("%d lamps, %d asked", count, len(asked))
		}
		if !(math.Abs(got-all) <= 1e-9*all) {
			t.Errorf("%d lamps: worth %g asked, %g in all", count, got, all)
		}
	}
}

// lampsLit is the lamps a point asks under so many suns, in the order
// asked, and how many of those count other than once.
func lampsLit(count int) (indices []int, weighted int) {
	up := maths.V(0, 1, 0)
	scene := stage(suns(count), 0)
	for _, one := range scene.asked(nil, maths.V(0.3, 0, -0.7), up,
		byLight) {
		indices = append(indices, one.index)
		if one.counts != 1 {
			weighted++
		}
	}
	return indices, weighted
}

// between lampsAsked and lampsEvery lamps, a point lights every lamp,
// in order and once each, with no picking: twelve lamps are twelve
// asks, and eight are eight as before. The anti-tests: twenty lamps,
// past lampsEvery, are still sampled, lampsAsked picks at most, some
// counting more than once; and so are lampsEvery and one.
func TestTwelveLampsAreEachLitAndTwentyAreSampled(t *testing.T) {
	for _, count := range []int{lampsAsked, 12} {
		indices, weighted := lampsLit(count)
		if len(indices) != count || weighted != 0 {
			t.Errorf("%d lamps: %d asked, %d weighted", count,
				len(indices), weighted)
		}
		for place, index := range indices {
			if index != place {
				t.Errorf("%d lamps: ask %d is lamp %d", count, place,
					index)
			}
		}
	}
	for _, count := range []int{lampsEvery + 1, 20} {
		indices, weighted := lampsLit(count)
		if len(indices) > lampsAsked || weighted == 0 {
			t.Errorf("%d lamps: %d asked, %d weighted, not sampled",
				count, len(indices), weighted)
		}
	}
}

// the same point always asks the same lamps, so a replay meets them.
func TestAPointAsksTheSameLamps(t *testing.T) {
	scene := stage(suns(200), 0)
	point, up := maths.V(0.3, 0, -0.7), maths.V(0, 1, 0)
	first, again := scene.asked(nil, point, up, byLight),
		scene.asked(nil, point, up, byLight)
	if len(first) != len(again) {
		t.Fatalf("%d lamps asked, then %d", len(first), len(again))
	}
	for i := range first {
		if first[i].index != again[i].index ||
			first[i].counts != again[i].counts {
			t.Errorf("ask %d: lamp %d, then lamp %d", i, first[i].index,
				again[i].index)
		}
	}
}

// under a ball, part of the sky's two hundred suns is hidden. a point
// asks only some, so its light is not every lamp's, but over many
// points the lamps asked light the floor as every lamp does, within a
// hundredth.
func TestTheLampsAskedAreRightOnAverage(t *testing.T) {
	scene := stage(suns(200), 0)
	up := maths.V(0, 1, 0)
	asked, every := 0.0, 0.0
	for i := range 40 {
		for k := range 40 {
			point := maths.V(-2+float64(i)*0.1, 0, -2+float64(k)*0.1)
			asked += scene.lamplight(point, up, maths.Grey(1)).Luminance()
			from := point.Add(up.Scale(lift))
			for _, lamp := range scene.lamps {
				toward, far, light := lamp.Light(point, up)
				square := up.Dot(toward)
				if square <= 0 {
					continue
				}
				reached := scene.transmit(from, toward, far)
				every += light.Times(reached).Luminance() * square /
					math.Pi
			}
		}
	}
	if !(math.Abs(asked-every) <= 0.01*every) {
		t.Errorf("the floor under the ball: %g asked, %g of every lamp",
			asked, every)
	}
}
