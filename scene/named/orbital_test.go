package named_test

import (
	"testing"

	"github.com/janearc/slayer/scene/named"
)

// The orbital is drawn by the cpu and the gpu: cmd/orbital's gpu test
// holds the gpu's pictures of both views to the tracer's.
func TestTheOrbitalDrawsOnBoth(t *testing.T) {
	scene := mustGet(t, "orbital")
	if !scene.Draws(named.CPU) || !scene.Draws(named.GPU) {
		t.Errorf("the orbital draws on %v", scene.Renderers())
	}
}

// Distant proof texture density is explicit and cannot grow beyond the
// existing full-detail default. Orbital geometry is unchanged.
func TestOrbitalProofFloorBudget(t *testing.T) {
	scene := mustGet(t, "orbital")
	for _, count := range []int{16, 512, 7700} {
		if err := scene.Check(map[string]any{"floor-tiles": count}); err != nil {
			t.Fatal(err)
		}
	}
	for _, count := range []int{0, 15, 7701} {
		if err := scene.Check(map[string]any{"floor-tiles": count}); err == nil {
			t.Fatalf("accepted floor budget %d", count)
		}
	}
}
