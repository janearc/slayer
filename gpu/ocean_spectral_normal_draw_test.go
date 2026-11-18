//go:build draw && darwin

package gpu

import (
	"math"
	"testing"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

func spectralRipples(t *testing.T) (*thing.World, maths.Subspace) {
	t.Helper()
	n, extent := 65, 40.0
	heights := make([]float64, n*n)
	xSlopes, zSlopes := make([]float64, n*n), make([]float64, n*n)
	for row := range n {
		for column := range n {
			x := float64(column)/float64(n-1)*extent - extent/2
			z := float64(row)/float64(n-1)*extent - extent/2
			at := row*n + column
			heights[at] = 0.08 * math.Sin(x*0.8) * math.Cos(z*0.7)
			xSlopes[at] = 0.08 * 0.8 * math.Cos(x*0.8) * math.Cos(z*0.7)
			zSlopes[at] = -0.08 * 0.7 * math.Sin(x*0.8) * math.Sin(z*0.7)
		}
	}
	field, err := shape.NewHeightField(heights, n, extent)
	if err != nil {
		t.Fatal(err)
	}
	field, err = field.WithApproximateSpectralNormals(xSlopes, zSlopes)
	if err != nil {
		t.Fatal(err)
	}
	world := thing.NewWorld()
	water := &thing.Material{Name: "water", Gloss: 1, Index: 1.33, Through: 1}
	world.Add(thing.Thing{Shape: field, Material: water})
	floor := &thing.Material{Name: "floor", Glow: maths.RGB(0.6, 0.55, 0.4)}
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 25}, Material: floor,
		Motion: at(0, -27, 0)})
	world.Lamp(emit.Lit{At: maths.V(-14, 2.5, 3),
		Strength: maths.RGB(400, 380, 350)})
	from := maths.V(1.5, 12, 2)
	eye := maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, maths.V(0, -0.5, 0))}
	return world, eye
}

func TestMetalInterpolatesSpectralHeightFieldNormalsLikeCPU(t *testing.T) {
	world, eye := spectralRipples(t)
	within(t, world, eye, 96, 64, 2,
		tolerance{mean: 2e-4, largest: 5e-2})
}

// Keep a separate appearance-only parity check so changes to optical normals
// cannot mask a regression in the already packed spectral shading normals.
func TestMetalOpaqueSpectralHeightFieldShadingMatchesCPU(t *testing.T) {
	world, eye := spectralRipples(t)
	world.Things[0].Material.Through = 0
	world.Things[0].Material.Colour = maths.Grey(0.5)
	within(t, world, eye, 96, 64, 2,
		// Opaque shading still uses the packed spectral normals. Its cell-hit
		// edge pixels carry the same finite-field march tolerance as other HF
		// Metal comparisons.
		tolerance{mean: 2e-3, largest: 8e-2})
}
