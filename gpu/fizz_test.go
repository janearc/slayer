//go:build draw

package gpu

import (
	"testing"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// fizzing is a tablet's worth of fizz: a glass ball of water, the mist
// of microbubbles rising inside it, two bubbles big enough to see, and
// a glowing backdrop, so the light read the mist through the glass is
// the whole picture.
func fizzing() (*thing.World, maths.Subspace) {
	world := thing.NewWorld()
	water := &thing.Material{Name: "glass of water", Gloss: 1,
		Index: 1.33, Through: 1}
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1},
		Material: water})
	bubble := &thing.Material{Name: "bubble", Gloss: 1, Index: 0.75,
		Through: 1}
	for _, place := range []maths.Vec{maths.V(0.1, -0.2, 0.3),
		maths.V(-0.2, 0.3, 0.1)} {
		world.Add(thing.Thing{Shape: shape.Sphere{Radius: 0.06},
			Material: bubble, Motion: at(place.X, place.Y, place.Z)})
	}
	mist := &thing.Material{Name: "mist", Colour: maths.Grey(0.9)}
	world.AddVolume(thing.Volume{
		Medium: medium.Fizz{Radius: 0.25, Height: 1.2, Thick: 0.8,
			Rise: 0.05, Churn: 0.6, Seed: 11},
		Material: mist,
		Motion:   at(0, -0.6, 0)})
	wall := &thing.Material{Name: "wall", Glow: maths.RGB(0.6, 0.6, 0.7)}
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 8},
		Material: wall, Motion: at(0, 0, -14)})
	world.Lamp(emit.Lit{At: maths.V(3, 10, 4), Strength: maths.Grey(60)})
	from := maths.V(0.4, 0.3, 4)
	return world, maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, maths.Vec{})}
}

// the gpu draws a fizzing glass as slayer's tracer does, the mist read
// from a baked grid where the tracer asks it exactly, inside the baked
// tolerance the hall is held to
func TestTheGPUDrawsAFizzingGlassAsSlayerDoes(t *testing.T) {
	world, eye := fizzing()
	within(t, world, eye, 96, 64, 2,
		tolerance{mean: 3e-4, largest: 1e-3})
}
