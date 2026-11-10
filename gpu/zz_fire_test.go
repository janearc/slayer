package gpu

import (
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/flame"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/plume"
	"github.com/janearc/slayer/scene/thing"
	slayer "github.com/janearc/slayer/trace"
)

// fire is a flame's body over a floor, its smoke rising out of it, lit
// by the flame's own light cut into cells as hillside cuts it: a boxed
// medium that shines, thicker to blue, and one that is painted grey and
// tinted, both of them timed, and neither a puff the kernel knows.
func fire(t *testing.T) (*thing.World, maths.Subspace) {
	t.Helper()
	world := thing.NewWorld()
	ground := &thing.Material{Name: "ground", Colour: maths.Grey(0.35)}
	world.Add(thing.Thing{Shape: slab(t, 16), Material: ground})
	body, err := flame.NewBody(flame.Fire{Radius: 1.2, Height: 4,
		Absorption: 1, Hottest: measure.Kelvin(1603),
		Edge: measure.Kelvin(900), Seed: 5}, 2e-5)
	if err != nil {
		t.Fatal(err)
	}
	world.AddVolume(thing.Volume{Medium: body,
		Material: &thing.Material{Name: "flame", Shine: body.Shine()}})
	column, err := plume.New(plume.Fire{Heat: 4000, Convective: 0.71,
		Base: 2.4, Yield: 0.0024, Combustion: 21, Extinction: 8,
		Air: 288, Density: 1.2, Top: 12, Seed: 5})
	if err != nil {
		t.Fatal(err)
	}
	world.AddVolume(thing.Volume{Medium: column,
		Material: &thing.Material{Name: "smoke", Colour: maths.Grey(0.3)}})
	for _, source := range body.Tick(0).(*flame.Body).Sources(3) {
		world.Lamp(emit.Lit{At: source.At, Strength: source.Strength,
			Beyond: source.Across / 2})
	}
	from := maths.V(0, 3, 11)
	return world, maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, maths.V(0, 2.5, 0))}
}

// the gpu draws a fire as slayer's tracer does, the flame and its smoke
// read from grids the cpu asked them into, where the tracer asks the
// media themselves at every step: so the two come together as the grid
// is made finer, and a mistake would not.
//
// measured on 2026-10-18 with the smooth flame: 0.91, 0.22 and 0.053
// hundredths apart on average at 48, 96 and 192 cells, four times closer
// for twice the cells. with the flame of six octaves and rags its
// finest eddy, 1.5 cm here, is under 192 cells' 2.5 cm, so it closes
// more slowly: 0.27 and 0.091, three times. held to twice.
func TestTheGPUDrawsAFireAsSlayerDoes(t *testing.T) {
	cells, most := boxCells, mostBoxCells
	defer func() { boxCells, mostBoxCells = cells, most }()
	var apart [2]float64
	for k, side := range []int{96, 192} {
		boxCells, mostBoxCells = side, side
		world, eye := fire(t)
		apart[k] = drawnBoth(t, world, eye).mean
	}
	t.Logf("apart at 96 and 192 cells: %v", apart)
	if apart[1] > 1.5e-3 || apart[1] > apart[0]/2 {
		t.Errorf("at 96 and 192 cells the fire is %v apart: it does "+
			"not come together as the grid grows finer", apart)
	}
}

// drawnBoth is how far apart a world is drawn on the gpu and the cpu at 96 by
// 64, four rays per pixel
func drawnBoth(t *testing.T, world *thing.World,
	eye maths.Subspace) difference {
	t.Helper()
	drawn := drawn(t, world, eye)
	lens := film.Pinhole{Wide: 50 * maths.Degree}
	truth, err := slayer.Sampled(world, eye, lens, 0, 96, 64, 2)
	if err != nil {
		t.Fatal(err)
	}
	return compare(truth, drawn, 96, 64)
}
