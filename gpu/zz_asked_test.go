package gpu

import (
	"math"
	"testing"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// many is forty point lamps in a ring over the world, each of its own
// strength, all one colour when grey, so a point asks eight picks among
// them by worth
func many(world *thing.World, grey bool) {
	for k := range 40 {
		turn := 2 * math.Pi * float64(k) / 40
		strength := 0.4 + 0.6*float64(k%7)/6
		light := maths.RGB(strength, strength, strength)
		if !grey {
			light = maths.RGB(strength, 0.3+0.7*float64(k%3)/2,
				0.2+0.8*float64(k%5)/4)
		}
		world.Lamp(emit.Lit{At: maths.V(7*math.Cos(turn), 3+float64(k%4),
			7*math.Sin(turn)), Strength: light})
	}
}

// with nothing in the way and lamps of one colour, whichever lamps a
// point picks bring it the light of every lamp exactly, so the gpu's
// picks and the tracer's, made from different bits of the same point,
// draw the same floor
func TestPicksAmongLampsOfOneColourAreExact(t *testing.T) {
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: floor(t, 12), Material: &thing.Material{
		Name: "floor", Colour: maths.Grey(0.6)}, Motion: at(0, -1.5, 0)})
	many(world, true)
	from := maths.V(0, 4, 10)
	eye := maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, maths.V(0, -1, 0))}
	against(t, world, eye)
}

// with things in the way, the picks decide which shadows a point asks, and the
// gpu's and the tracer's pick differently, so the two pictures agree on average
// and not pixel for pixel.
//
// measured on 2026-10-18 at 96 by 64: 1.1 hundredths apart on average at four
// rays per pixel and 0.55 at sixteen, halving as noise does and a bias would
// not. held to about twice that, at a quarter of the size.
func TestPicksAmongManyLampsAgreeOnAverage(t *testing.T) {
	world, eye := hall(t)
	world.Lamps = nil
	many(world, false)
	noisy := tolerance{mean: 2.5e-2, largest: 1e-2}
	within(t, world, eye, 48, 32, 2, noisy)
}
