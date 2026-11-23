//go:build draw

package session

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

// Eight simultaneous flame groups need eight slots and their shared smoke
// needs one more. Every pre-existing volume is a real accepted plume.
func TestFrontPreparationBudgetsSharedSmokeAtTheGPUVolumeLimit(t *testing.T) {
	given := frontOnFixture(t)
	raw, err := os.ReadFile(given.set)
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Replace(string(raw), "group-width: {value: 100, unit: m}",
		"group-width: {value: 0.01, unit: m}", 1)
	text = strings.Replace(text, "kind: fern, count: 4", "kind: fern, count: 8", 1)
	if text == string(raw) {
		t.Fatal("fixture lost grouping width")
	}
	if err := os.WriteFile(given.set, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
	read := readBoundarySet(t, given.set)
	row, has := read.Recipe("wood fire")
	if !has {
		t.Fatal("fixture lost wood fire")
	}
	spec, has := row.Fire()
	if !has || spec.Smoke == nil {
		t.Fatal("fixture lost smoke")
	}
	column, err := plume.New(*spec.Smoke)
	if err != nil {
		t.Fatal(err)
	}
	given.exposure = 1
	// the front asks eight group slots and one shared smoke; existing
	// volumes fill the rest of the limit, so the totals straddle it
	limit := gpu.VolumeLimit()
	for _, sample := range []struct {
		existing, total int
		okay            bool
	}{
		{limit - 10, limit - 1, true},
		{limit - 9, limit, true},
		{limit - 8, limit + 1, false},
	} {
		t.Run(fmt.Sprintf("%d total", sample.total), func(t *testing.T) {
			world := thing.NewWorld()
			for i := 0; i < sample.existing; i++ {
				world.AddVolume(thing.Volume{Medium: column,
					Material: &thing.Material{Name: "existing smoke", Colour: maths.Grey(.3)},
					Motion:   thing.Still(maths.Subspace{Origin: maths.V(float64(i), 0, 0)})})
			}
			var trees []groveTree
			for z := 5; z < 13; z++ {
				trees = append(trees, groveTree{at: maths.V(10, 0, float64(z))})
			}
			stage := staging{given: given, recipes: read, world: world,
				places: map[string]maths.Vec{"the tree": maths.V(10, 0, 5)},
				fuels:  map[string][]groveTree{"front fuel": trees}}
			err := stage.prepareFronts(read)
			if sample.okay {
				if err != nil {
					t.Fatalf("%d volumes should fit: %v", sample.total, err)
				}
				if len(stage.fronts) != 1 || len(stage.fronts[0].plan.Groups()) != 8 {
					t.Fatal("fixture did not retain eight separate flame groups")
				}
			} else {
				refusal := fmt.Sprintf(
					"needs up to %d volumes; GPU limit is %d",
					limit+1, limit)
				if err == nil || !strings.Contains(err.Error(), refusal) {
					t.Fatalf("%d volumes should exceed the GPU limit: %v",
						sample.total, err)
				}
				if len(stage.fronts) != 0 {
					t.Fatal("refused front was retained")
				}
			}
		})
	}
}
