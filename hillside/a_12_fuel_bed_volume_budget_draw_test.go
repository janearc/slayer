//go:build draw

package hillside

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/janearc/slayer/gpu"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/plume"
	"github.com/janearc/slayer/scene/thing"
)

// A 16-by-8 bed has two actual 64-patch groups. Combined with valid
// existing volumes, the GPU volume limit admits totals at and under it
// and refuses one more.
func TestFuelBedReservationCountsExistingVolumesAndEveryFlameGroup(t *testing.T) {
	given := fuelOnFixture(t)
	raw, err := os.ReadFile(given.set)
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Replace(string(raw), "      width: 8", "      width: 16", 1)
	if err := os.WriteFile(given.set, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
	read := readBoundarySet(t, given.set)
	column, err := plume.New(plume.Fire{
		Heat: 40, Convective: .71, Base: 1, Yield: .0024, Combustion: 21,
		Extinction: 10.6, Air: 288, Density: 1.2, Top: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	given.onGPU = false
	// the bed asks two group slots; existing volumes fill the rest of
	// the limit, so the totals straddle it
	limit := gpu.VolumeLimit()
	for _, sample := range []struct {
		existing, total int
		okay            bool
	}{
		{limit - 3, limit - 1, true},
		{limit - 2, limit, true},
		{limit - 1, limit + 1, false},
	} {
		t.Run(fmt.Sprintf("%d total", sample.total), func(t *testing.T) {
			world := thing.NewWorld()
			for i := 0; i < sample.existing; i++ {
				world.AddVolume(thing.Volume{Medium: column,
					Material: &thing.Material{Name: "existing smoke", Colour: maths.Grey(.3)},
					Motion:   thing.Still(maths.Subspace{Origin: maths.V(float64(i), 0, 0)})})
			}
			stage := staging{given: given, recipes: read, world: world}
			if err := stage.terrain(read); err != nil {
				t.Fatal(err)
			}
			err := stage.prepareBeds(read)
			defer (&liveScene{beds: stage.beds}).closeBeds()
			if sample.okay {
				if err != nil {
					t.Fatalf("%d volumes should fit: %v", sample.total, err)
				}
				if len(stage.beds) != 1 || len(stage.beds[0].groups) != 2 {
					t.Fatal("fixture lost its two real flame groups")
				}
				patches := 0
				for _, group := range stage.beds[0].groups {
					patches += len(group.field.Pieces())
				}
				if patches != 128 {
					t.Fatalf("retained%d patches, want128", patches)
				}
			} else {
				if err == nil || !strings.Contains(err.Error(), "fuel bed exceeds GPU volume budget") {
					t.Fatalf("%d volumes should exceed the GPU limit: %v",
						sample.total, err)
				}
				if len(stage.beds) != 0 {
					t.Fatal("refused bed retained a model")
				}
			}
		})
	}
}
