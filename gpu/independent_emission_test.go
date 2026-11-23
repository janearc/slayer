package gpu

import (
	"strings"
	"testing"

	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

func TestIndependentSurfaceEmissionIsRefused(t *testing.T) {
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Plane{}, Material: &thing.Material{IndependentEmission: true}})
	if err := newScene().things(world, 0); err == nil || !strings.Contains(err.Error(), "independent surface emission is CPU-only") {
		t.Fatalf("GPU accepted independent emission or returned unclear error: %v", err)
	}
}
