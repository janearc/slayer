package searching

import (
	"reflect"
	"testing"

	"github.com/janearc/slayer/scene/thing"
)

// sharingBenchmarkPlan gives both cases the same small authored population.
// Its repeated sail geometry makes the storage tradeoff visible without
// pretending to represent the historical, independently shaped finale.
func sharingBenchmarkPlan(shared bool) *stagePlan {
	original := examplePopulation()
	population := make([]authoredBeingPlan, 16)
	for i := range population {
		population[i] = original[i%len(original)]
		population[i].Identity = i + 1
		population[i].At = [3]float64{float64(i%8) * 6, float64(i/8) * 5, 0}
	}
	plan := populationStagePlan(population)
	plan.ShareAnatomyPrototypes = &shared
	return plan
}

func TestAnatomySharingBenchmarkUsesTheSameValidPopulation(t *testing.T) {
	shared, separate := sharingBenchmarkPlan(true), sharingBenchmarkPlan(false)
	for _, plan := range []*stagePlan{shared, separate} {
		if err := plan.check(); err != nil {
			t.Fatal(err)
		}
	}
	if !*shared.ShareAnatomyPrototypes || *separate.ShareAnatomyPrototypes {
		t.Fatal("benchmark cases did not select both sharing settings")
	}
	shared.ShareAnatomyPrototypes, separate.ShareAnatomyPrototypes = nil, nil
	if !reflect.DeepEqual(shared, separate) {
		t.Fatal("benchmark changed more than the sharing setting")
	}
}

// BenchmarkAuthoredAnatomySharingConstruction compares the same authored
// population with its prototype cache enabled and disabled. It is deliberately
// opt-in and CPU-only: it draws no frames and uses no soundtrack. A single
// iteration limits each case to one constructed stage; storage accounting is
// outside the timed construction interval.
func BenchmarkAuthoredAnatomySharingConstruction(b *testing.B) {
	for _, shared := range []bool{true, false} {
		name := "separate"
		if shared {
			name = "shared"
		}
		b.Run(name, func(b *testing.B) {
			plan := sharingBenchmarkPlan(shared)
			if err := plan.check(); err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			b.ResetTimer()
			var last *heardStage
			for range b.N {
				var err error
				last, err = newHeardStage(plan, "")
				if err != nil {
					b.Fatal(err)
				}
			}
			b.StopTimer()
			parts := make([]thing.Thing, len(last.world.Things))
			for i, part := range last.world.Things {
				parts[i] = *part
			}
			stored := thing.Storage(parts)
			if len(stored.Unknown) != 0 {
				b.Fatalf("unknown geometry: %v", stored.Unknown)
			}
			b.ReportMetric(float64(stored.Things), "things")
			b.ReportMetric(float64(stored.UniqueMeshes), "unique-meshes")
			b.ReportMetric(float64(stored.ReferencedFaces), "referenced-faces")
			b.ReportMetric(float64(stored.StoredFaces), "stored-faces")
			b.ReportMetric(float64(stored.OwnedGeometryBytes), "owned-geometry-bytes")
		})
	}
}
