package gpu

import (
	"errors"
	"math"
	"testing"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/metal"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/thing"
)

// hills is ground of rolling hills, coarse, side squares a side, each
// corner's normal the hills' own there: shaded smooth it is the curved
// ground, shaded flat its facets.
func hills(t *testing.T, side int, smooth bool) *mesh.Mesh {
	t.Helper()
	height := func(x, z float64) float64 {
		return 0.6 * math.Sin(x*0.9) * math.Cos(z*0.7)
	}
	var corners, normals []maths.Vec
	var faces [][3]int
	middle := float64(side) / 2
	for row := range side + 1 {
		for column := range side + 1 {
			x, z := float64(column)-middle, float64(row)-middle
			corners = append(corners, maths.V(x, height(x, z), z))
			slopeX := 0.6 * 0.9 * math.Cos(x*0.9) * math.Cos(z*0.7)
			slopeZ := -0.6 * 0.7 * math.Sin(x*0.9) * math.Sin(z*0.7)
			normals = append(normals, maths.V(-slopeX, 1, -slopeZ))
		}
	}
	for row := range side {
		for column := range side {
			near := row*(side+1) + column
			far := near + side + 1
			faces = append(faces, [3]int{near, far, near + 1},
				[3]int{near + 1, far, far + 1})
		}
	}
	made, err := mesh.New(corners, faces, normals, smooth)
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// lowLit is the hills twice, shaded smooth and shaded flat side by
// side, under a lamp low over them, so the light grazes the slopes and
// the shadow's edge crosses faces, where the lift is asked
func lowLit(t *testing.T) (*thing.World, maths.Subspace) {
	world := thing.NewWorld()
	ground := &thing.Material{Name: "ground", Colour: maths.Grey(0.5)}
	world.Add(thing.Thing{Shape: hills(t, 10, true), Material: ground,
		Motion: at(-5.5, -1, 0)})
	world.Add(thing.Thing{Shape: hills(t, 10, false), Material: ground,
		Motion: at(5.5, -1, 0)})
	world.Lamp(emit.Lit{At: maths.V(-14, 2.5, 3),
		Strength: maths.RGB(400, 380, 350)})
	from := maths.V(0, 9, 13)
	return world, maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, maths.V(0, -1, 0))}
}

// drawn is a world on the gpu at 96 by 64, four rays per pixel, through the
// lens the comparisons use
func drawn(t *testing.T, world *thing.World, eye maths.Subspace) *frame.Frame {
	t.Helper()
	lens := film.Pinhole{Wide: 50 * maths.Degree}
	picture, err := Sampled(world, eye, lens, 0, 96, 64, 2)
	if errors.Is(err, metal.ErrNoDevice) {
		t.Skip("no gpu here")
	}
	if err != nil {
		t.Fatal(err)
	}
	return picture
}

// the gpu draws a mesh shaded smooth as slayer's tracer does: its
// corners' normals blended across each face, and the lights asked from
// the curved surface they draw
func TestTheGPUDrawsASmoothMeshAsSlayerDoes(t *testing.T) {
	world, eye := lowLit(t)
	against(t, world, eye)
}

// the smooth hills and the flat ones are not drawn alike, so the test
// above is asking the kernel to blend, and not passing by facets
func TestSmoothHillsAreNotTheirFacets(t *testing.T) {
	world, eye := lowLit(t)
	flat := thing.NewWorld()
	ground := &thing.Material{Name: "ground", Colour: maths.Grey(0.5)}
	flat.Add(thing.Thing{Shape: hills(t, 10, false), Material: ground,
		Motion: at(-5.5, -1, 0)})
	flat.Add(thing.Thing{Shape: hills(t, 10, false), Material: ground,
		Motion: at(5.5, -1, 0)})
	flat.Lamps = world.Lamps
	smooth := drawn(t, world, eye)
	faceted := drawn(t, flat, eye)
	found := compare(smooth, faceted, 96, 64)
	if found.mean < 1e-2 {
		t.Errorf("smooth and flat hills are %.5f apart: the blend is "+
			"not drawn", found.mean)
	}
}
