package main

import (
	"testing"

	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

func TestNamedOpticalRecipesReachActualGeometryAndKeepSeparateStock(t *testing.T) {
	recipes := `  recipes:
    compact:
      kind: optical-observatory
      enclosure:
        kind: dome
        diameter: {value: 12, unit: m}
        opening: {value: 30, unit: deg}
      telescope:
        aperture: {value: 6, unit: m}
        mount: alt-azimuth
        primary: segmented
        pointing: {azimuth: 0, altitude: 45}
    wide:
      kind: optical-observatory
      enclosure:
        kind: dome
        diameter: {value: 32, unit: m}
        opening: {value: 40, unit: deg}
      telescope:
        aperture: {value: 14, unit: m}
        mount: alt-azimuth
        primary: segmented
        pointing: {azimuth: 70, altitude: 60}
  terrain:`
	given := mustParse(t)
	given.set = onFixture(t, "  terrain:", recipes)
	read := readBoundarySet(t, given.set)
	stage := staging{given: given, recipes: read, have: stock{},
		world: thing.NewWorld(), radii: map[string]float64{}}
	var previous maths.Box
	for i, name := range []string{"compact", "wide"} {
		row, has := read.Recipe(name)
		if !has {
			t.Fatal("fixture lost its checked optical recipe", name)
		}
		kind, err := manufacture.Observatory(row.Optical())
		if err != nil {
			t.Fatal(err)
		}
		object, err := manufacture.MakeWithComplexity(kind,
			manufacture.Sow(stir(given.seed, 7)), "quick")
		if err != nil {
			t.Fatal(err)
		}
		parts, err := object.Things(maths.Subspace{})
		if err != nil || len(parts) == 0 {
			t.Fatalf("public recipe reference has no actual parts: %v", err)
		}
		before := len(stage.world.Things)
		at := maths.V(float64(i)*50, 0, 10)
		bounds, err := stage.putRecipe("optical-observatory", name, "quick", 7, uint64(i), at, false, name)
		if err != nil || bounds != object.Bounds() {
			t.Fatalf("recipe %s lost its authored physical extent: %v, %v, want %v", name, err, bounds, object.Bounds())
		}
		if i > 0 && bounds == previous {
			t.Fatal("two authored recipe extents no longer distinguish stock identities")
		}
		previous = bounds
		if len(stage.world.Things)-before != len(parts) {
			t.Fatal("actual recipe drew the wrong number of material parts")
		}
		for j, expected := range parts {
			actual := stage.world.Things[before+j]
			if actual.Shape.Bounds() != expected.Shape.Bounds() ||
				actual.Material == nil || expected.Material == nil ||
				actual.Material.Colour != expected.Material.Colour {
				t.Fatalf("actual %s part%d differs from its public recipe", name, j)
			}
			space := actual.At(0)
			for _, axis := range []maths.Vec{{}, maths.XAxis, maths.YAxis, maths.ZAxis} {
				if gap := space.Out(axis).Sub(at.Add(axis)).Len(); !(gap <= 1e-12) {
					t.Fatalf("pointing already baked into %s was rotated again by %g m", name, gap)
				}
			}
		}
	}
}
