//go:build draw

package gpu

import (
	"testing"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// One clear thing inside another, drawn both ways. trace's own tests
// hold what a look does at each surface; these hold the kernel to the
// pictures that makes.

// seenThrough adds what the clear things of these tests are seen
// against and by: a glowing wall, a red ball and a blue ball glowing in
// front of it, either side of the middle, and a lamp. A bend moves the
// two balls in the picture, so a bend made at the wrong surface shows.
func seenThrough(world *thing.World) maths.Subspace {
	wall := &thing.Material{Name: "wall", Glow: maths.RGB(0.6, 0.6, 0.7)}
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 8}, Material: wall,
		Motion: at(0, 0, -14)})
	red := &thing.Material{Name: "red", Glow: maths.RGB(1.2, 0.1, 0.1)}
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 0.8}, Material: red,
		Motion: at(-1.5, 0.5, -5)})
	blue := &thing.Material{Name: "blue", Glow: maths.RGB(0.1, 0.2, 1.2)}
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 0.8}, Material: blue,
		Motion: at(1.2, -0.4, -6)})
	world.Lamp(emit.Lit{At: maths.V(3, 10, 4), Strength: maths.Grey(60)})
	from := maths.V(0.4, 0.3, 4)
	return maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, maths.Vec{})}
}

// glassOfWater is a hollow glass ball, a unit in radius outside and half
// that inside, and a ball of water in it nine tenths in radius: the
// water runs well into the glass's wall. rank is the glass's; the
// water's is nought. Where the glass outranks the water, the water's
// face inside the wall is no boundary and the look bends at the wall's
// inner face; where it does not, the look bends at the water's face,
// four tenths further out.
func glassOfWater(rank int) (*thing.World, maths.Subspace) {
	world := thing.NewWorld()
	glass := &thing.Material{Name: "glass", Gloss: 1, Index: 1.5,
		Through: 1, Rank: rank}
	water := &thing.Material{Name: "water", Gloss: 1, Index: 1.33,
		Through: 1}
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1}, Material: glass})
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 0.5}, Material: glass})
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 0.9}, Material: water})
	return world, seenThrough(world)
}

// beadUnderWater is a ball of water with a glass bead in it, off its
// middle: leaving the bead leaves the look in the water it was in, so
// the bead's faces are glass against water and the water's far face is
// water against air.
func beadUnderWater() (*thing.World, maths.Subspace) {
	world := thing.NewWorld()
	water := &thing.Material{Name: "water", Gloss: 1, Index: 1.33,
		Through: 1}
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1}, Material: water})
	bead := &thing.Material{Name: "bead", Gloss: 1, Index: 1.5, Through: 1}
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 0.35}, Material: bead,
		Motion: at(0.2, 0.1, 0.2)})
	return world, seenThrough(world)
}

// the gpu draws a glass of water as slayer's tracer does, where the
// glass outranks the water and the water's face in the wall is no
// boundary: the look slips through it and bends at the wall's inner
// face, glass against water
func TestTheGPUDrawsAGlassOfWaterAsSlayerDoes(t *testing.T) {
	world, eye := glassOfWater(1)
	againstAt(t, world, eye, 96, 64, 2)
}

// the anti-test of the one above, in two parts. with no rank between
// them the water's face in the wall is a boundary, and the gpu draws
// that as slayer's tracer does too; and the two pictures are not one
// picture, so the rank reaches the kernel and is not left at nought
func TestWithoutARankTheGPUBendsAtTheWatersFace(t *testing.T) {
	level, eye := glassOfWater(0)
	againstAt(t, level, eye, 96, 64, 2)
	ranked, _ := glassOfWater(1)
	const few = 200 // picked: of 6,144 pixels, far under what a bend four tenths out moves
	if apart := apartBy(drawn(t, ranked, eye), drawn(t, level, eye)); apart < few {
		t.Errorf("the glass of water with a rank and without differ in "+
			"%d pixels on the gpu, want %d or more: the rank is not "+
			"reaching the kernel", apart, few)
	}
}

// the gpu draws a glass bead under water as slayer's tracer does:
// leaving the bead leaves the look in the water, and does not forget
// everything it was inside
func TestTheGPUDrawsAGlassBeadUnderWaterAsSlayerDoes(t *testing.T) {
	world, eye := beadUnderWater()
	againstAt(t, world, eye, 96, 64, 2)
}

// the anti-test of the one above: the bead is there to be seen. the
// same water with no bead in it is another picture on the gpu
func TestTheBeadUnderWaterShowsOnTheGPU(t *testing.T) {
	with, eye := beadUnderWater()
	bare := thing.NewWorld()
	bare.Add(thing.Thing{Shape: shape.Sphere{Radius: 1},
		Material: &thing.Material{Name: "water", Gloss: 1, Index: 1.33,
			Through: 1}})
	seenThrough(bare)
	const few = 50 // picked: the bead covers about a hundred and fifty pixels
	if apart := apartBy(drawn(t, with, eye), drawn(t, bare, eye)); apart < few {
		t.Errorf("the water with a bead and without differ in %d pixels "+
			"on the gpu, want %d or more", apart, few)
	}
}
