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
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1},
		Material: &thing.Material{Colour: maths.Grey(0.5)},
		Motion:   resting(maths.V(0, 1.5, 0))})
	random := rand.New(rand.NewPCG(7, 11))
	for range count {
		way := maths.V(random.Float64()*2-1, 0.2+random.Float64(),
			random.Float64()*2-1).Unit()
		world.Lamp(thing.Sun(way, maths.Grey(0.1+random.Float64())))
	}
	return world
}

// with nothing in the way, the lamps a point asks are worth, counted as
// asked, exactly what every lamp is worth: with two hundred lamps as
// with eight; with eight or fewer every lamp is asked once.
func TestTheLampsAskedAreWorthEveryLamp(t *testing.T) {
	up := maths.V(0, 1, 0)
	worth := func(toward maths.Vec, light maths.Light) float64 {
		return light.Luminance() * math.Max(0, up.Dot(toward))
	}
	for _, count := range []int{3, 8, 200} {
		scene := stage(suns(count), 0)
		point := maths.V(40, 0, 40)
		all, got := 0.0, 0.0
		for _, lamp := range scene.lamps {
			toward, _, light := lamp.Light(point, up)
			all += worth(toward, light)
		}
		asked := scene.asked(point, up, worth)
		for _, one := range asked {
			got += one.counts * worth(one.toward, one.light)
			if count <= lampsAsked && one.counts != 1 {
				t.Errorf("%d lamps: lamp %d counts %g", count, one.index,
					one.counts)
			}
		}
		if count <= lampsAsked && len(asked) != count {
			t.Errorf("%d lamps, %d asked", count, len(asked))
		}
		if !(math.Abs(got-all) <= 1e-9*all) {
			t.Errorf("%d lamps: worth %g asked, %g in all", count, got, all)
		}
	}
}

// the same point always asks the same lamps, so a replay meets them.
func TestAPointAsksTheSameLamps(t *testing.T) {
	scene := stage(suns(200), 0)
	point, up := maths.V(0.3, 0, -0.7), maths.V(0, 1, 0)
	first, again := scene.asked(point, up, byLight),
		scene.asked(point, up, byLight)
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
