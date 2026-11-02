package trace

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/gas"
	"github.com/janearc/slayer/scene/thing"
)

// cut is a medium of thickness k all through the part of a cube that lies
// below a height, and clear above it. a ray sampled ten times across the
// cube finds the cut between two of its samples, and the thickness by how
// many fell below it.
type cut struct {
	block
	below float64
}

func (c cut) Density(p maths.Vec) float64 {
	if p.Y < c.below {
		return c.block.Density(p)
	}
	return 0
}

// volumes are the volumes of a world of them, placed.
func volumes(vs ...thing.Volume) []volume {
	w := thing.NewWorld()
	for _, v := range vs {
		w.AddVolume(v)
	}
	return stage(w, 0).vs
}

// paintedBy is a volume given a white material, so that it is painted.
func paintedBy(m thing.Volume) thing.Volume {
	m.Material = &thing.Material{Colour: maths.Grey(1)}
	return m
}

// how much of a lamp's light gets through the volumes between a point
// and the lamp is beer's law over the way through each: exp(-k d), where
// d is how far the way runs through the volume in its own units.
//
// it is found by ten samples along the way, started part way along the
// first step by a smooth noise, so for a medium of one thickness all
// through, or a cut at a whole number of samples, the answer is exact
// whatever the jitter.
//
// worked by hand: a block of half-width 1 and k of 0.5 seen from 5 below
// has 2 of way, so exp(-1); from inside, 1 of way, exp(-0.5); the block
// doubled in scale, seen from 10 below, is still exp(-1).
//
// a ball of radius 0.5 and k of 1 seen from its middle has 0.5 of way,
// exp(-0.5). a cut at the middle of a block of k of 1.2, 2 across, has
// five samples of ten below it, so exp(-1.2).
//
// a medium thick past its box is cut off at the box's far side by the
// ten samples, no more: exp(-0.5) for k of 0.25 over 2.
func TestClearIsBeersLawAlongTheWayToALamp(t *testing.T) {
	up := maths.V(0, 1, 0)
	for _, c := range []struct {
		name  string
		v     thing.Volume
		from  maths.Vec
		taken float64
	}{
		{"from below a block", mediumOf(block{k: 0.5, half: 1}), maths.V(0, -5, 0), 1},
		{"from inside a block", mediumOf(block{k: 0.5, half: 1}), maths.V(0, 0, 0), 0.5},
		{"from the middle of a ball", mediumOf(ball{k: 1, radius: 0.5}), maths.V(0, 0, 0), 0.5},
		{"from below a doubled block", thing.Volume{Medium: block{k: 0.5, half: 1},
			Motion: at(maths.Vec{}, 2)}, maths.V(0, -10, 0), 1},
		{"from below a block cut at its middle", mediumOf(cut{block{k: 1.2, half: 1}, 0}),
			maths.V(0, -5, 0), 1.2},
		{"from below a block thick past its box", mediumOf(leaky{block{k: 0.25, half: 1}}),
			maths.V(0, -5, 0), 0.5},
		{"from a block that is told it was not crossed",
			mediumOf(told{block{k: 1, half: 100}, 1, 9, false}), maths.V(0, 0, 0), 0},
	} {
		got := clear(volumes(paintedBy(c.v)), c.from, up)
		if want := math.Exp(-c.taken); !near(got, want, 1e-12) {
			t.Errorf("%s: %v, want %v", c.name, got, want)
		}
	}
}

// a gas takes each channel of a lamp's light by its own extinction, so the
// same ways through the same shapes as above, with the tinted gas's 0.9,
// 0.5 and 0.2 per metre and 2 metres a unit, come to exp(-e u k d) in each
// channel.
//
// a gas that says by its box where it is crossed has its box asked, and a
// ball its sphere: from the middle of a ball of radius 1 along a diagonal
// the sphere is left at 1, and the box not until root 2.
func TestGasClearIsBeersLawInEveryChannel(t *testing.T) {
	g := tinted(redOptics, greenOptics, blueOptics)
	up := maths.V(0, 1, 0)
	diagonal := maths.V(1, 1, 0).Unit()
	of := func(m gas.Of) thing.Volume { return thing.Volume{Medium: m} }
	for _, c := range []struct {
		name  string
		v     thing.Volume
		from  maths.Vec
		along maths.Vec
		taken float64 // k times the way through, in the volume's own units
	}{
		{"from below a block", of(gas.Of{Shape: block{k: 0.5, half: 1}, In: g, Unit: 2}),
			maths.V(0, -5, 0), up, 0.5 * 2},
		{"from below a block that is not told how it is crossed",
			thing.Volume{Medium: gasMedium{block{k: 0.5, half: 1}, g.Per(2)}},
			maths.V(0, -5, 0), up, 0.5 * 2},
		{"from inside a small ball", of(gas.Of{Shape: ball{k: 1, radius: 0.5}, In: g, Unit: 2}),
			maths.V(0, 0, 0), up, 0.5},
		{"from the middle of a ball, along a diagonal", of(gas.Of{Shape: ball{k: 1, radius: 1}, In: g, Unit: 2}),
			maths.V(0, 0, 0), diagonal, 1},
		{"from below a doubled block", thing.Volume{Medium: gas.Of{Shape: block{k: 0.5, half: 1}, In: g, Unit: 2},
			Motion: at(maths.Vec{}, 2)}, maths.V(0, -10, 0), up, 0.5 * 2},
		{"from below a block cut at its middle", of(gas.Of{Shape: cut{block{k: 1.2, half: 1}, 0}, In: g, Unit: 2}),
			maths.V(0, -5, 0), up, 1.2},
		{"from below a block thick past its box", of(gas.Of{Shape: leaky{block{k: 0.25, half: 1}}, In: g, Unit: 2}),
			maths.V(0, -5, 0), up, 0.25 * 2},
		{"from a block that is told it was not crossed", of(gas.Of{Shape: told{block{k: 1, half: 100}, 1, 9, false}, In: g, Unit: 2}),
			maths.V(0, 0, 0), up, 0},
		{"past a block it misses", of(gas.Of{Shape: block{k: 1, half: 1}, In: g, Unit: 2}),
			maths.V(5, 0, 0), up, 0},
	} {
		got := gasClear(volumes(c.v), c.from, c.along)
		want := maths.RGB(math.Exp(-extinct(redOptics)*2*c.taken),
			math.Exp(-extinct(greenOptics)*2*c.taken), math.Exp(-extinct(blueOptics)*2*c.taken))
		if !near(got.R, want.R, 1e-12) || !near(got.G, want.G, 1e-12) ||
			!near(got.B, want.B, 1e-12) {
			t.Errorf("%s: %v, want %v", c.name, got, want)
		}
	}
}

// the jitter that starts the ten samples part way along the first step
// is smooth, so a shadow does not lay its steps down as grain: points
// side by side have the same samples, and the shadow between them is
// the same.
//
// a block 500 across, ten samples of 50 each, is cut half a step above
// the 5th sample's start, so that how many samples fall below the cut
// says where in its step the jitter has put them.
//
// along a line of 2000 points a unit long, in each of the three
// directions, that count changes at most twice. the points are taken
// from below, so the way through the block is the same for each.
func TestTheJitterOfAShadowIsSmoothInSpace(t *testing.T) {
	g := tinted(redOptics, greenOptics, blueOptics)
	// 5 steps of 50 and a half, from the block's foot at -250
	thin := cut{block{k: 0.002, half: 250}, -250 + 5.5*50}
	painted := volumes(paintedBy(mediumOf(thin)))
	gassy := volumes(thing.Volume{Medium: gas.Of{Shape: thin, In: g, Unit: 1}})
	up := maths.V(0, 1, 0)
	for _, axis := range []maths.Vec{maths.XAxis, maths.YAxis, maths.ZAxis} {
		for _, kind := range []string{"clear", "gasClear"} {
			flips, last := 0, math.NaN()
			for i := 0; i < 2000; i++ {
				p := maths.V(0.3, -400, 0.7).Add(axis.Scale(float64(i) / 2000))
				var seen float64
				if kind == "clear" {
					seen = clear(painted, p, up)
				} else {
					seen = gasClear(gassy, p, up).R
				}
				if !math.IsNaN(last) && !near(seen, last, 1e-9) {
					flips++
				}
				last = seen
			}
			if flips > 2 {
				t.Errorf("%s along %v: the shadow changes %d times over a unit", kind, axis, flips)
			}
		}
	}
}

// each of the ten samples of a shadow lies in its own step, however the
// jitter moves it. a cut at three whole steps of a block 2 across, 0.6
// from its foot, has three samples below it at every point, so the
// shadow is exp(-0.6), and exp(-0.9 0.6) in the red of a gas.
func TestEverySampleOfAShadowLiesInItsOwnStep(t *testing.T) {
	g := tinted(redOptics, greenOptics, blueOptics)
	for i := 0; i < 40; i++ {
		p := maths.V(float64(i%17)*0.05-0.4, -5, float64(i%13)*0.06-0.35)
		got := clear(volumes(paintedBy(mediumOf(cut{block{k: 1, half: 1}, -1 + 0.6}))), p, maths.V(0, 1, 0))
		if want := math.Exp(-0.6); !near(got, want, 1e-12) {
			t.Fatalf("at %v the cut block lets %v through, want %v", p, got, want)
		}
		gotGas := gasClear(volumes(thing.Volume{Medium: gas.Of{Shape: cut{block{k: 1, half: 1}, -1 + 0.6},
			In: g, Unit: 1}}), p, maths.V(0, 1, 0))
		if want := math.Exp(-0.9 * 0.6); !near(gotGas.R, want, 1e-12) {
			t.Fatalf("at %v the cut gas lets %v through, want %v", p, gotGas.R, want)
		}
	}
}
