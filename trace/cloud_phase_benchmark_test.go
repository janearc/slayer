package trace

import (
	"testing"

	"github.com/janearc/slayer/scene/medium"
)

var cloudPhaseMeasured float64

func TestCloudPhaseSelectionDoesNotAllocate(t *testing.T) {
	cloud, err := medium.NewClearCloudWithHenyeyGreenstein(
		medium.Puff{Radius: 1, Thick: 1}, .8)
	if err != nil {
		t.Fatal(err)
	}
	var field medium.Medium = cloud
	allocations := testing.AllocsPerRun(1000, func() {
		phase := clearCloudPhaseOf(field)
		cloudPhaseMeasured = phase.at(.3, 0)
	})
	if allocations != 0 {
		t.Fatalf("phase selection allocated %g times per cell", allocations)
	}
}

func BenchmarkClearCloudPhaseSelection(b *testing.B) {
	cloud, err := medium.NewClearCloudWithHenyeyGreenstein(
		medium.Puff{Radius: 1, Thick: 1}, .8)
	if err != nil {
		b.Fatal(err)
	}
	var field medium.Medium = cloud
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		phase := clearCloudPhaseOf(field)
		cloudPhaseMeasured = phase.at(.3, 0)
	}
}
