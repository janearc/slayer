package manufacture

import (
	"math"
	"testing"

	"github.com/janearc/slayer/scene/shape"
)

func TestParkEnrichmentKeepsBaselinePlantingAndAddsWalkableDiverseGrove(t *testing.T) {
	spec := greenClusteredCity(false)
	baseline, err := PlanClusteredCity(spec, Sow(31))
	if err != nil {
		t.Fatal(err)
	}
	spec.Trees.SizeScale = 1.0 / 250
	spec.Trees.ParkEnrichment = true
	spec.ParkRelief = true
	grown, err := PlanClusteredCity(spec, Sow(31))
	if err != nil {
		t.Fatal(err)
	}
	if len(grown.Buildings()) != len(baseline.Buildings()) || len(grown.Trees()) <= len(baseline.Trees()) {
		t.Fatalf("lost city or failed to add grove: buildings %d/%d trees %d/%d", len(grown.Buildings()), len(baseline.Buildings()), len(grown.Trees()), len(baseline.Trees()))
	}
	old := make(map[uint64]CityTree, len(baseline.trees))
	for _, tree := range baseline.trees {
		old[tree.Seed] = tree
	}
	seen := map[uint64]bool{}
	for i, tree := range grown.trees[:len(baseline.trees)] {
		was, ok := old[tree.Seed]
		if !ok || seen[tree.Seed] {
			t.Fatalf("baseline tree %d missing or duplicated", i)
		}
		seen[tree.Seed] = true
		if tree.Kind != was.Kind || tree.PrototypeSeed != was.PrototypeSeed || tree.Place.Origin.X != was.Place.Origin.X || tree.Place.Origin.Z != was.Place.Origin.Z {
			t.Fatalf("baseline location or identity changed: %+v / %+v", was, tree)
		}
		if math.Abs(tree.Place.Scale-was.Place.Scale/250) > 1e-10 {
			t.Fatalf("wrong physical tree scale: %g / %g", tree.Place.Scale, was.Place.Scale)
		}
		for _, park := range grown.parks {
			if math.Abs(tree.Place.Origin.X-park.At.X) < park.Width.InMetres()/2 && math.Abs(tree.Place.Origin.Z-park.At.Z) < park.Depth.InMetres()/2 {
				if height := parkTerrainHeight(park, tree.Place.Origin); math.Abs(tree.Place.Origin.Y-height) > 1e-12 {
					t.Fatalf("established tree is detached from park terrain: %v / %g", tree.Place.Origin, height)
				}
			}
		}
	}
	park := grown.parks[0]
	families := map[string]int{}
	extra := grown.trees[len(baseline.trees):]
	if len(extra) < 200 {
		t.Fatalf("grove too sparse: %d added", len(extra))
	}
	for _, tree := range extra {
		families[tree.Kind]++
		if math.Abs(tree.Place.Origin.X-park.At.X) > 5 || math.Abs(tree.Place.Origin.Z-park.At.Z) > 5 {
			t.Fatalf("grove escaped local park: %v", tree.Place.Origin)
		}
		clear := 25.0 / 250
		endX := park.At.X + park.Width.InMetres()*.15*math.Sin(2*math.Pi*.5003)
		endZ := park.At.Z + park.Depth.InMetres()*.84*.0003
		if tree.Bounds.Min.X < math.Max(park.At.X, endX)+clear && tree.Bounds.Max.X > math.Min(park.At.X, endX)-clear && tree.Bounds.Min.Z < math.Max(park.At.Z, endZ)+clear && tree.Bounds.Max.Z > math.Min(park.At.Z, endZ)-clear {
			t.Fatalf("tree bounds block camera corridor: %+v", tree.Bounds)
		}
		if height := parkTerrainHeight(park, tree.Place.Origin); math.Abs(tree.Place.Origin.Y-height) > 1e-12 {
			t.Fatalf("tree floats above or sinks below park terrain: %v, terrain %g", tree.Place.Origin, height)
		}
		tall := (tree.Bounds.Max.Y - tree.Bounds.Min.Y) * 250
		if tall < 15 || tall > 35 {
			t.Fatalf("additional tree outside physical size range: %g m", tall)
		}
	}
	for _, family := range []string{"alder", "poplar", "joshua-tree"} {
		if families[family] == 0 {
			t.Fatalf("missing %s among %d new trees", family, len(extra))
		}
	}
	t.Logf("kept %d established trees and added %d physically scaled trees: %v", len(baseline.trees), len(extra), families)
}

func TestCityTreeSizeScaleRejectsInvalidValues(t *testing.T) {
	spec := DefaultCityTrees()
	for _, value := range []float64{-1, 2, math.NaN(), math.Inf(1)} {
		spec.SizeScale = value
		if err := spec.check(); err == nil {
			t.Fatalf("accepted invalid tree scale %g", value)
		}
	}
}

func TestEnrichedParkUsesSharedDetailedNativeFoliageAndDistinctLooks(t *testing.T) {
	spec := greenClusteredCity(false)
	spec.Trees.SizeScale = 1.0 / 250
	spec.Trees.ParkEnrichment = true
	city, err := PlanClusteredCity(spec, Sow(31))
	if err != nil {
		t.Fatal(err)
	}
	// A small sample tests real quick foliage without building a whole extra
	// city acceleration hierarchy while production rendering is active.
	var sample []CityTree
	families := map[string]bool{}
	for _, tree := range city.trees {
		if !families[tree.Kind] {
			sample = append(sample, tree)
			families[tree.Kind] = true
		}
	}
	if len(families) != 4 {
		t.Fatalf("missing prototype families: %v", families)
	}
	city.trees = sample
	made, err := city.treeGroups(good{dressed: looks{}, groups: map[slot]*shape.MeshGroup{}, groupParts: map[slot]int{}}, *spec.Trees, Sow(31))
	if err != nil {
		t.Fatal(err)
	}
	if len(made.groups) < 3 {
		t.Fatalf("plant families lost distinct material roles: %d", len(made.groups))
	}
	var members, stored, logical int
	for _, group := range made.groups {
		members += group.MemberCount()
		stored += group.StoredFaces()
		logical += group.FaceCount()
	}
	if members < 100 || stored < 1000 || logical < stored {
		t.Fatalf("native foliage detail/sharing lost: members=%d stored=%d logical=%d", members, stored, logical)
	}
	t.Logf("four species: %d foliage parts, %d stored faces, %d logical faces, %d material groups", members, stored, logical, len(made.groups))
}

func TestUrbanRowsPreserveGroveAndReserveWholeTreeBounds(t *testing.T) {
	spec := greenClusteredCity(false)
	spec.ParkRelief = true
	spec.Trees.SizeScale = 1.0 / 250
	spec.Trees.ParkEnrichment = true
	before, err := PlanClusteredCity(spec, Sow(31))
	if err != nil {
		t.Fatal(err)
	}
	spec.Trees.UrbanRows = true
	after, err := PlanClusteredCity(spec, Sow(31))
	if err != nil {
		t.Fatal(err)
	}
	if len(after.trees) <= len(before.trees) {
		t.Fatalf("no new street trees: %d / %d", len(after.trees), len(before.trees))
	}
	for i, tree := range before.trees {
		if tree != after.trees[i] {
			t.Fatalf("existing tree %d moved or changed", i)
		}
	}
	park := after.parks[0]
	for _, tree := range after.trees {
		for _, side := range []float64{-1, 1} {
			x := park.At.X + side*4.18
			if tree.Bounds.Min.X < x+.008 && tree.Bounds.Max.X > x-.008 && tree.Bounds.Min.Z < park.At.Z+3 && tree.Bounds.Max.Z > park.At.Z-3 {
				t.Fatalf("tree overlaps avenue: %s %+v", tree.Kind, tree.Bounds)
			}
		}
	}
	for _, row := range after.trees[len(before.trees):] {
		if math.Abs(math.Abs(row.Place.Origin.X-park.At.X)-4.25) > 1e-9 {
			t.Fatalf("row escaped expected avenue: %v", row.Place.Origin)
		}
		if math.Abs(row.Place.Origin.Y-parkTerrainHeight(park, row.Place.Origin)) > 1e-12 {
			t.Fatal("row tree is detached from terrain")
		}
		for _, existing := range before.trees {
			a, b := row.Bounds, existing.Bounds
			if a.Min.X < b.Max.X && a.Max.X > b.Min.X && a.Min.Z < b.Max.Z && a.Max.Z > b.Min.Z {
				t.Fatalf("new row overlaps existing tree: %+v / %+v", a, b)
			}
		}
	}
	t.Logf("kept %d trees and added %d street-row trees", len(before.trees), len(after.trees)-len(before.trees))
	spec.Trees.ParkEnrichment = false
	if err := spec.Trees.check(); err == nil {
		t.Fatal("urban rows without required enrichment accepted")
	}
}
