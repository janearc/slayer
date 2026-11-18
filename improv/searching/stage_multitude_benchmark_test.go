package searching

import (
	"testing"

	"github.com/janearc/slayer/scene/thing"
)

// BenchmarkHistoricalMultitudeConstruction inventories the existing v9
// finale constructor without rendering it. Run this explicitly with
// `game test -- -run '^$' -bench '^BenchmarkHistoricalMultitudeConstruction$' -benchtime=1x -count=1`;
// the single-iteration setting bounds peak work to one constructed scene.
func BenchmarkHistoricalMultitudeConstruction(b *testing.B) {
	var inventory thing.GeometryStorage
	b.ReportAllocs()
	for iteration := 0; iteration < b.N; iteration++ {
		stage := &heardStage{world: thing.NewWorld()}
		if err := stage.makeMultitude(); err != nil {
			b.Fatal(err)
		}
		if got, want := len(stage.crowd.beings), 3067; got != want {
			b.Fatalf("historical multitude has %d beings, want %d", got, want)
		}
		b.StopTimer()
		parts := make([]thing.Thing, len(stage.world.Things))
		for index, part := range stage.world.Things {
			parts[index] = *part
		}
		inventory = thing.Storage(parts)
		b.StartTimer()
	}
	for _, metric := range []struct {
		name  string
		value float64
	}{
		{"things", float64(inventory.Things)},
		{"unique-meshes", float64(inventory.UniqueMeshes)},
		{"referenced-faces", float64(inventory.ReferencedFaces)},
		{"stored-faces", float64(inventory.StoredFaces)},
		{"owned-geometry-bytes", float64(inventory.OwnedGeometryBytes)},
		{"unknown-shapes", float64(len(inventory.Unknown))},
	} {
		b.ReportMetric(metric.value, metric.name)
	}
	if len(inventory.Unknown) != 0 {
		b.Fatalf("geometry inventory missed shape accounting: %v", inventory.Unknown)
	}
	b.Logf("geometry inventory: %+v", inventory)
}
