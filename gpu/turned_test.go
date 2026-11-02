//go:build darwin

package gpu

import (
	"errors"
	"testing"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/metal"
	"github.com/janearc/slayer/scene/thing"
	slayer "github.com/janearc/slayer/trace"
)

// cube is slayer's mesh of a cube of side one about its middle, each
// face turning counterclockwise seen from outside.
func cube(t *testing.T) *mesh.Mesh {
	t.Helper()
	corners := make([]maths.Vec, 8)
	for i := range corners {
		corners[i] = maths.V(float64(i&1)-0.5, float64(i>>1&1)-0.5,
			float64(i>>2&1)-0.5)
	}
	made, err := mesh.New(corners, [][3]int{{0, 4, 6}, {0, 6, 2},
		{1, 3, 7}, {1, 7, 5}, {0, 1, 5}, {0, 5, 4}, {2, 6, 7}, {2, 7, 3},
		{0, 2, 3}, {0, 3, 1}, {4, 5, 7}, {4, 7, 6}}, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// turnedCubes is one cube standing twice, each turned about an axis of
// its own that is none of x, y or z, and each scaled, over a floor,
// under a lamp: a row of a standing carried into the cube's space the
// wrong way, or a shift the wrong way, moves a cube or turns it wrong,
// and the picture shows it.
func turnedCubes(t *testing.T) (*thing.World, maths.Subspace) {
	world := thing.NewWorld()
	solid := cube(t)
	red := &thing.Material{Name: "red", Colour: maths.RGB(0.7, 0.2, 0.2)}
	blue := &thing.Material{Name: "blue", Colour: maths.RGB(0.2, 0.3, 0.7)}
	world.Add(thing.Thing{Shape: solid, Material: red,
		Motion: thing.Still(maths.Subspace{Origin: maths.V(-1.2, 0.3, 0),
			Orientation: maths.About(maths.V(1, 2, 0.5).Unit(), 0.7),
			Scale:       1.6})})
	world.Add(thing.Thing{Shape: solid, Material: blue,
		Motion: thing.Still(maths.Subspace{Origin: maths.V(1.3, 0.1, -0.4),
			Orientation: maths.About(maths.V(-0.3, 0.4, 1).Unit(), 2.1),
			Scale:       0.8})})
	grey := &thing.Material{Name: "floor", Colour: maths.Grey(0.5)}
	world.Add(thing.Thing{Shape: floor(t, 6), Material: grey,
		Motion: at(0, -1.2, 0)})
	world.Lamp(emit.Lit{At: maths.V(2, 6, 5), Strength: maths.RGB(40, 38,
		35)})
	from := maths.V(1, 3, 7)
	return world, maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, maths.Vec{})}
}

// A floor standing a hundred thousand times its own size, seen half a
// degree below level, is drawn where slayer draws it: in its own space
// the ray's way is a hundred thousandth long, and a face is not thrown
// away as edge on for the shortness of the way.
func TestALargeFloorSeenGrazingIsKept(t *testing.T) {
	world := thing.NewWorld()
	grey := &thing.Material{Name: "floor", Colour: maths.Grey(0.5)}
	world.Add(thing.Thing{Shape: floor(t, 1), Material: grey,
		Motion: thing.Still(maths.Subspace{Scale: 1e5})})
	world.Lamp(emit.Lit{At: maths.V(0, 5000, 0), Strength: maths.RGB(1e8,
		1e8, 1e8)})
	from := maths.V(0, 300, 0)
	toward := from.Add(maths.V(0, -0.0087, -1))
	eye := maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, toward)}
	lens := film.Pinhole{Wide: 0.5 * maths.Degree}
	drawn, err := Sampled(world, eye, lens, 0, 32, 32, 1)
	if errors.Is(err, metal.ErrNoDevice) {
		t.Skip("no gpu here")
	}
	if err != nil {
		t.Fatal(err)
	}
	truth, err := slayer.Sampled(world, eye, lens, 0, 32, 32, 1)
	if err != nil {
		t.Fatal(err)
	}
	if found := compare(truth, drawn, 32, 32); found.cover > 0 {
		t.Errorf("%d pixels of the floor are covered on one side only",
			found.cover)
	}
}

// The gpu draws meshes standing turned about any axis and scaled as
// slayer's own tracer does, to the exact tolerance: the same cover and
// depth in every pixel, and the light as near as float allows.
func TestTheGPUDrawsTurnedScaledMeshesAsSlayerDoes(t *testing.T) {
	world, eye := turnedCubes(t)
	lens := film.Pinhole{Wide: 50 * maths.Degree}
	width, height, side := 96, 64, 2
	drawn, err := Sampled(world, eye, lens, 0, width, height, side)
	if errors.Is(err, metal.ErrNoDevice) {
		t.Skip("no gpu here")
	}
	if err != nil {
		t.Fatal(err)
	}
	truth, err := slayer.Sampled(world, eye, lens, 0, width, height, side)
	if err != nil {
		t.Fatal(err)
	}
	found := compare(truth, drawn, width, height)
	t.Logf("apart: %+v", found)
	if found.mean > exact.mean || found.largest > exact.largest {
		t.Errorf("the gpu's light is %.6f off slayer's on average, and "+
			"%.4f at most", found.mean, found.largest)
	}
	if found.cover > 0 || found.depth > 0 {
		t.Errorf("%d pixels' cover and %d pixels' depth differ",
			found.cover, found.depth)
	}
}
