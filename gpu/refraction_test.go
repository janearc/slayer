//go:build draw

package gpu

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// pond is a wide clear ball of water with a glowing floor sunk inside
// it, seen from above and a little aside, so the bends run across the
// picture. Nothing is painted and nothing glossy but the water, so
// every difference between the backends is the bend itself.
func pond() (*thing.World, maths.Subspace) {
	world := thing.NewWorld()
	water := &thing.Material{Name: "water", Gloss: 1, Index: 1.33,
		Through: 1}
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 20},
		Material: water, Motion: at(0, -20.5, 0)})
	floor := &thing.Material{Name: "floor", Glow: maths.RGB(0.7, 0.6, 0.4)}
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 15},
		Material: floor, Motion: at(0, -18, 0)})
	from := maths.V(2, 4, 9)
	return world, maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, maths.V(0, -2, 0))}
}

// marble is a clear glass ball before a glowing backdrop: a ray bends
// in, bends out, and lands, two legs of the chain on both backends.
func marble() (*thing.World, maths.Subspace) {
	world := thing.NewWorld()
	glass := &thing.Material{Name: "glass", Gloss: 1, Through: 1}
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1},
		Material: glass})
	wall := &thing.Material{Name: "wall", Glow: maths.RGB(0.5, 0.7, 0.9)}
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 8},
		Material: wall, Motion: at(0, 0, -14)})
	from := maths.V(0.3, 0.2, 4)
	return world, maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, maths.Vec{})}
}

// the gpu bends into water as slayer's tracer does
func TestTheGPUBendsIntoWaterAsSlayerDoes(t *testing.T) {
	world, eye := pond()
	against(t, world, eye)
}

// the gpu reads through a glass ball as slayer's tracer does: in one
// side, out the other, fresnel's toll at each
func TestTheGPUReadsThroughAGlassBallAsSlayerDoes(t *testing.T) {
	world, eye := marble()
	against(t, world, eye)
}

// an opaque glossy scene with Through nowhere renders as it always
// has: the chain adds nothing when no material is clear
func TestAnOpaqueSceneIsUntouchedByTheChain(t *testing.T) {
	world, eye := orrery()
	if encoded, err := encode(world, eye, nil, 0, 4, 3); err != nil {
		t.Fatal(err)
	} else if encoded.cleared {
		t.Fatal("an opaque scene reads as cleared")
	}
	againstAt(t, world, eye, 97, 65, 1)
}

// ripples is a gently waved square of water over a glowing floor: the
// height-field kind, marched and bent on both backends. The waves are
// long and low, so the march is honest work and no pixel grazes.
func ripples() (*thing.World, maths.Subspace) {
	n, extent := 65, 40.0
	heights := make([]float64, n*n)
	for row := range n {
		for column := range n {
			x := float64(column)/float64(n-1)*extent - extent/2
			z := float64(row)/float64(n-1)*extent - extent/2
			heights[row*n+column] = 0.08 * math.Sin(x*0.8) *
				math.Cos(z*0.7)
		}
	}
	field, err := shape.NewHeightField(heights, n, extent)
	if err != nil {
		panic(err)
	}
	world := thing.NewWorld()
	water := &thing.Material{Name: "water", Gloss: 1, Index: 1.33,
		Through: 1}
	world.Add(thing.Thing{Shape: field, Material: water})
	floor := &thing.Material{Name: "floor", Glow: maths.RGB(0.6, 0.55, 0.4)}
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 25},
		Material: floor, Motion: at(0, -27, 0)})
	from := maths.V(1.5, 12, 2)
	return world, maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, maths.V(0, -0.5, 0))}
}

// the gpu marches a waved height field of water as slayer's tracer
// does. Not to the exact tolerance: the field's normal turns at every
// cell line, and the two marches stop within touch on either side of
// a line at a handful of pixels, so the largest allowed difference is
// a few hundredths at those pixels while the mean stays at tens of
// millionths. Measured 7.5e-5 mean and 1.6e-2 largest on 2026-11-10,
// held to about double. Grazing views diverge further - the march
// runs out of steps at different pixels in 32-bit and 64-bit float -
// and wait on the cell-walking intersector DEBT.md records.
func TestTheGPUMarchesAWavedPoolAsSlayerDoes(t *testing.T) {
	world, eye := ripples()
	within(t, world, eye, 96, 64, 2, tolerance{mean: 2e-4, largest: 5e-2})
}
