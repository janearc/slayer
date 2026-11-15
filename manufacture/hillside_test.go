package manufacture

import (
	"testing"

	"github.com/janearc/slayer/forest"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/sky"
	"github.com/janearc/slayer/scene/thing"
)

// the smoke test: everything the manufactory makes so far, stood on a
// ground and drawn by slayer's tracer (trace) on the cpu. it proves
// a picture comes out and that the things are in it. it cannot prove a fern
// looks like a fern: set HILLSIDE_OUT to a path and it writes a larger
// picture there, to be looked at.
//
// the scene, its ground and its light are this test's own, set by
// eye. the things and their looks are the kinds' rows.
var soil = &thing.Material{Name: "soil",
	Colour: maths.RGB(0.2, 0.17, 0.1)}

const (
	// the ground is so many cells of so many metres, and things stand
	// on a grid of places this far apart
	groundCells, groundSize = 8, 160.0
	placesASide, apart      = 6, 2.4
)

// bare is the hillside with nothing on it: ground, sky and sun.
func bare(t *testing.T) *thing.World {
	t.Helper()
	world := thing.NewWorld()
	world.Sky = sky.Gradient{Horizon: maths.RGB(0.7, 0.8, 0.95),
		Zenith: maths.RGB(0.25, 0.45, 0.9),
		Ground: maths.RGB(0.34, 0.29, 0.18)}
	// a sun, and a faint second one from the other side standing in
	// for the light of the sky, which the tracer does not gather
	world.Lamp(thing.Sun(maths.V(-0.5, 0.8, 0.45), maths.Grey(1.5)))
	world.Lamp(thing.Sun(maths.V(0.6, 0.5, -0.3), maths.Grey(0.4)))
	ground, err := forest.Ground(groundCells, groundSize)
	if err != nil {
		t.Fatal(err)
	}
	corner := maths.V(-groundSize/2, 0, -groundSize/2)
	world.Add(thing.Thing{Shape: ground, Material: soil,
		Motion: thing.Still(maths.Subspace{Origin: corner})})
	return world
}
