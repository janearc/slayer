//go:build draw && darwin

package gpu

import (
	"errors"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/metal"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/skyglow"
)

// glowOptions is a coarse grid and a short series: enough for the glow
// to vary across the air, and quick to solve.
var glowOptions = skyglow.Options{Nodes: [3]int{12, 6, 9}, Orders: 3,
	Tolerance: 0.01, Workers: 2}

// glowLamp is the one source the test glow is solved from: a warm lamp
// low over one side of the air, not one the tracer scatters itself, so
// the glow carries all its light.
var glowLamp = skyglow.Source{At: maths.V(60, 15, -40),
	Power: maths.RGB(4e5, 3e5, 2e5)}

// glowEye stands in the air near the ground at one corner, looking
// across it toward the lamp and up.
func glowEye() maths.Subspace {
	from := maths.V(-170, 2, 120)
	return maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, maths.V(60, 60, -40))}
}

// glowWorld is the test air with the glow of the lamp solved in it, and
// no lamp: all the light there is is the glow. The glow is of the air in
// the world, or, where it lights none, of another layer of the same haze
// that is not in the world.
func glowWorld(t *testing.T, lightsTheAir bool) *thing.World {
	t.Helper()
	world := thing.NewWorld()
	air := world.AddVolume(flatAir(t, testAirBox))
	field, err := skyglow.Solve(skyglow.Air{Extinction: testHaze,
		Depth: testHazeDepth, Albedo: testHazeAlbedo},
		skyglow.Region{Low: testAirBox.Min, High: testAirBox.Max},
		[]skyglow.Source{glowLamp}, glowOptions)
	if err != nil {
		t.Fatal(err)
	}
	field.Medium = air.Medium
	if !lightsTheAir {
		field.Medium = flatAir(t, testAirBox).Medium
	}
	world.Glow = field
	return world
}

// the air the city's glow is of, seen from inside it, draws on the gpu
// as on slayer's tracer: the haze is a grid of cells on the gpu, so the
// tolerance is the grid's.
func TestTheGPUDrawsTheGlowInTheAirAsSlayerDoes(t *testing.T) {
	within(t, glowWorld(t, true), glowEye(), 96, 64, 2, gridded)
}

// a glow of other air lights none of this air: the gpu draws it dark,
// as slayer's tracer does, and does not take it for the air's.
func TestAGlowOfOtherAirLightsNothing(t *testing.T) {
	const width, height = 48, 32
	drawn, err := Sampled(glowWorld(t, false), glowEye(),
		film.Pinhole{Wide: 50 * maths.Degree}, 0, width, height, 1)
	if errors.Is(err, metal.ErrNoDevice) {
		t.Skip("no gpu here")
	}
	if err != nil {
		t.Fatal(err)
	}
	for row := range height {
		for column := range width {
			if seen := drawn.At(column, row); seen.R != 0 || seen.G != 0 ||
				seen.B != 0 {
				t.Fatalf("pixel %d %d holds %v, the air has no light",
					column, row, seen)
			}
		}
	}
}
