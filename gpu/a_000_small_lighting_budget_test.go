package gpu

import (
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/plume"
	"github.com/janearc/slayer/scene/thing"
)

// Keep the adversarial arithmetic check before GPU fixtures. Submetre
// bounds make a mistaken multiplication shrink the grid rather than
// reserve gigabytes before a later assertion can notice the wrong size.
func TestSmallSmokeGridRespectsResolutionBeforeAnyGPUPreparation(t *testing.T) {
	column, err := plume.New(plume.Fire{
		Heat: 1, Convective: .71, Base: .1, Yield: .0024,
		Combustion: 21, Extinction: 8, Air: 288, Density: 1.2,
		Top: .5, Seed: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	size := column.Bounds().Max.Sub(column.Bounds().Min)
	longest := max(size.X, size.Y, size.Z)
	if !(longest > 0 && longest < 1) {
		t.Fatalf("allocation-safe fixture needs sub-metre bounds: %v", size)
	}
	world := thing.NewWorld()
	world.AddVolume(thing.Volume{Medium: column,
		Material: &thing.Material{Colour: maths.Grey(.3)}})
	encoded, err := encode(world, maths.Subspace{}, nil, 0, 4, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(encoded.smokes) != 1 {
		t.Fatal("small fixture lost its lighting grid")
	}
	before := len(encoded.boxes.bytes)
	encoded.reserveLighting()
	cells := encoded.smokes[0].grid.cells
	most := 0
	for _, count := range cells {
		if count < 2 || count > 64 {
			t.Fatalf("small-grid dimension exceeds the authored budget: %v", cells)
		}
		most = max(most, count)
	}
	if most != 64 {
		t.Fatalf("small-grid longest side is %d, want 64", most)
	}
	wantBytes := cells[0] * cells[1] * cells[2] * 3 * 4
	if len(encoded.boxes.bytes)-before != wantBytes {
		t.Fatal("lighting reservation does not hold three float32 channels per cell")
	}
}
