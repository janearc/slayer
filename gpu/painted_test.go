package gpu

import (
	"math"
	"testing"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/globe"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/texture"
	"github.com/janearc/slayer/scene/thing"
)

// banded is a shine in bands of light by height in its own space, so a
// turned thing's bands turn with it
type banded struct{}

func (banded) At(at maths.Vec) maths.Light {
	part := 0.5 + 0.5*math.Sin(at.Y*4)
	return maths.RGB(1.5*part, 0.8*part, 0.3+0.2*part)
}

// fading is a paint that greys as the pixel asking grows, so a footprint
// the gpu hands back wrong paints wrong
type fading struct{}

func (fading) At(maths.Vec) maths.Light { return maths.Grey(0.9) }

func (fading) Within(at maths.Vec, footprint float64) maths.Light {
	return maths.RGB(0.9/(1+footprint*40), 0.5, 0.9*math.Abs(at.X))
}

// slab is a flat floor of slayer's mesh, side squares a side
func slab(t *testing.T, side int) *mesh.Mesh {
	t.Helper()
	flat := floor(t, side)
	made, err := mesh.New(flat.Vertices, flat.Faces, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// gallery is every way a surface is painted or shines: a checkered ball
// turned and grown, a globe shining in bands and tipped, a glossy floor
// of slayer's mesh painted by its footprint, mirroring both, and a puff
// in front dimming part of each, under a lamp and a sky.
func gallery(t *testing.T) (*thing.World, maths.Subspace) {
	world := thing.NewWorld()
	// the sky that lights them is lopsided, as the sky test's: a starry
	// sky lights a dim surface 1.3e-4 apart on the two, which is older
	// than painting and has a test of its own to come
	world.Sky = lopsided{}
	checks := &thing.Material{Name: "checks",
		Texture: texture.Checker(0.3, maths.RGB(0.9, 0.2, 0.1),
			maths.RGB(0.1, 0.4, 0.9))}
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1}, Material: checks,
		Motion: thing.Still(maths.Subspace{Origin: maths.V(-1.6, 0, 0),
			Scale:       1.5,
			Orientation: maths.About(maths.V(1, 1, 0).Unit(), 0.7)})})
	ball, err := globe.New(1.1)
	if err != nil {
		t.Fatal(err)
	}
	glowing := &thing.Material{Name: "bands", Shine: banded{}}
	world.Add(thing.Thing{Shape: ball, Material: glowing,
		Motion: thing.Still(maths.Subspace{Origin: maths.V(1.7, 0.2, 0),
			Orientation: maths.About(maths.V(0, 0, 1), 0.5)})})
	ground := &thing.Material{Name: "ground", Texture: fading{},
		Gloss: 0.6, Index: 1.8}
	world.Add(thing.Thing{Shape: slab(t, 12), Material: ground,
		Motion: at(0, -1.5, 0)})
	world.Lamp(emit.Lit{At: maths.V(2, 6, 5),
		Strength: maths.RGB(60, 55, 50)})
	haze := &thing.Material{Name: "haze", Colour: maths.RGB(0.7, 0.7, 0.8)}
	world.AddVolume(thing.Volume{Medium: medium.Puff{Radius: 1.4,
		Thick: 0.5}, Material: haze, Motion: at(0.4, 0.3, 2.5)})
	from := maths.V(0, 2.5, 9)
	return world, maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, maths.V(0, -0.3, 0))}
}

// the gpu draws painted and shining surfaces as slayer's tracer does,
// each painted on the cpu where the kernel says the ray met it. measured
// on 2026-10-18, 1.3 millionths apart on average and 3.1 thousandths at
// most: one pixel of 6,144, where one of its rays, mirrored off the
// floor, lands within a float's rounding of a checker's edge and takes
// the other check. held to about eight times and one and a half.
func TestTheGPUPaintsAsSlayerDoes(t *testing.T) {
	world, eye := gallery(t)
	within(t, world, eye, 96, 64, 2, tolerance{mean: 1e-5, largest: 5e-3})
}

// what a material does outside the visible is not asked by a visible
// picture, on the gpu as on the cpu
func TestARadiantMaterialIsDrawnInTheVisible(t *testing.T) {
	world, eye := orrery()
	world.Things[1].Material = &thing.Material{Name: "radiant",
		Colour: maths.RGB(0.6, 0.6, 0.6), Radiant: &thing.Spectrum{}}
	againstAt(t, world, eye, 97, 65, 1)
}
