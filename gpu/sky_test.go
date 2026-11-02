package gpu

import (
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// lopsided is a sky that is different every way: red to the east, green
// overhead, blue to the north, so a table read mirrored, turned or
// transposed lights a ball differently.
type lopsided struct{}

// Light is the sky's light along a way.
func (lopsided) Light(toward maths.Vec) maths.Light {
	toward = toward.Unit()
	return maths.RGB(0.5+0.5*toward.X, 0.5+0.5*toward.Y,
		0.5-0.5*toward.Z)
}

// a ball under a lopsided sky, lit by the sky alone, is drawn by the
// kernel as the tracer draws it: its east side red, its top green, its
// north side blue, from the same table of the sky, read the same way.
func TestTheKernelReadsTheSkyAsTheTracerDoes(t *testing.T) {
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1},
		Material: &thing.Material{Name: "ball", Colour: maths.Grey(0.8)}})
	world.Sky = lopsided{}
	from := maths.V(2, 1.5, 4)
	eye := maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, maths.Vec{})}
	againstAt(t, world, eye, 33, 33, 1)
}

// a flat floor's first look of the sky goes straight east, with a z of zero,
// which fast math's atan2 read as half a turn round, west: every floor under a
// lopsided sky was 2.2 thousandths off the tracer, and under stars 0.13. the
// floor is lit by the sky alone.
func TestAFloorIsLitByTheSkyAsTheTracerDoes(t *testing.T) {
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: floor(t, 12), Material: &thing.Material{
		Name: "floor", Colour: maths.Grey(0.5)}, Motion: at(0, -1.5, 0)})
	world.Sky = lopsided{}
	from := maths.V(0, 2.5, 9)
	eye := maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, maths.V(0, -0.3, 0))}
	against(t, world, eye)
}
