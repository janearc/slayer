package main

import (
	"errors"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
)

// Optical accepts positive apertures smaller than the YAML minimum. At this
// scale the structure's top faces underflow to zero area. Keep the actual
// mesh refusal through the stock adapter, after the recipe itself succeeds.
func TestManufacturedStockKeepsATinyOpticalMeshRefusal(t *testing.T) {
	spec := manufacture.Optical{
		Diameter: measure.Metres(20), Aperture: measure.Metres(4),
		Opening: measure.Degrees(30), Azimuth: measure.Degrees(20),
		Altitude: measure.Degrees(60),
	}
	given := mustParse(t)
	positive, err := manufacturedRecipe(given, "optical-observatory", 0,
		"quick", &spec)
	if err != nil || len(positive.things) != 3 {
		t.Fatalf("ordinary optical stock: %d materials, %v", len(positive.things), err)
	}
	spec.Aperture = measure.Metres(1e-200)
	row, err := manufacture.Observatory(spec)
	if err != nil {
		t.Fatalf("tiny positive aperture refused by the recipe: %v", err)
	}
	made, err := manufacture.MakeWithComplexity(row, manufacture.Sow(0), "quick")
	if err != nil {
		t.Fatalf("tiny recipe could not assemble: %v", err)
	}
	_, cause := made.Things(maths.Subspace{})
	if !errors.Is(cause, mesh.ErrMesh) {
		t.Fatalf("control did not reach mesh validation: %v", cause)
	}
	_, err = manufacturedRecipe(given, "optical-observatory", 0, "quick", &spec)
	if !errors.Is(err, mesh.ErrMesh) || err.Error() != cause.Error() {
		t.Fatalf("stock lost original mesh refusal: got %v, want %v", err, cause)
	}
}
