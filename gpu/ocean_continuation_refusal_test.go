package gpu

import (
	"strings"
	"testing"

	"github.com/janearc/slayer/arid"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

func TestMetalEncoderRefusesCPUOnlyOceanContinuation(t *testing.T) {
	core, err := shape.NewHeightField(make([]float64, 9), 3, 2)
	if err != nil {
		t.Fatal(err)
	}
	continued, err := arid.NewFlatMeanContinuation(core, 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: continued,
		Material: &thing.Material{Name: "clear", Index: 1.33, Through: 1, Gloss: 1}})
	if _, err := encode(world, maths.Local(), film.Pinhole{}, 0, 1, 1); err == nil ||
		!strings.Contains(err.Error(), "shape of kind *arid.FlatMeanContinuation") {
		t.Fatalf("Metal encoding error = %v, want explicit CPU-only shape refusal", err)
	}
}

func TestMetalEncoderExplicitlyRefusesInfiniteMeanPlane(t *testing.T) {
	world := thing.NewWorld()
	world.OceanSurface = &thing.OceanSurface{InfiniteMeanPlane: true}
	_, err := encode(world, maths.Local(), film.Pinhole{}, 0, 1, 1)
	if err == nil || !strings.Contains(err.Error(),
		"CPU-only infinite mean-plane ocean continuation") {
		t.Fatalf("Metal encoding error = %v, want explicit continuation refusal", err)
	}
}
