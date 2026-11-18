package manufacture

import (
	"math"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
)

func testInfillCity() ClusteredCityLayout {
	spec := DefaultClusteredCity()
	p := spec.Parks[0]
	treeAt := p.At.Add(maths.V(92, 0, 306))
	return ClusteredCityLayout{base: CityLayout{grid: spec.Grid, buildings: []CityBuilding{{Seed: 42, Kind: "city-tower"}}},
		parks: []CityPark{p}, parkRelief: true,
		trees: []CityTree{{Bounds: maths.Box{Min: treeAt.Add(maths.V(-5, 0, -5)), Max: treeAt.Add(maths.V(5, 20, 5))}}}}
}

func TestParkInfillIsDenseSeededAndDoesNotAlterCity(t *testing.T) {
	city := testInfillCity()
	original := city.base.buildings[0]
	infill := city.parkInfillPlan(sow(31))
	if len(infill) < 300 {
		t.Fatalf("got %d added buildings; expected a substantial district", len(infill))
	}
	if len(city.base.buildings) != 1 || city.base.buildings[0] != original {
		t.Fatal("original city changed")
	}
	kinds := map[string]bool{}
	seeds := map[uint64]bool{}
	sizes := map[float64]bool{}
	for i, one := range infill {
		b := one.building
		kinds[b.Kind] = true
		sizes[b.Place.Scale] = true
		if seeds[b.Seed] {
			t.Fatalf("seed repeated: %d", b.Seed)
		}
		seeds[b.Seed] = true
		for j := 0; j < i; j++ {
			if boxesOverlapXZ(b.Bounds, infill[j].building.Bounds) {
				t.Fatalf("infill overlaps %d and %d", i, j)
			}
		}
	}
	if len(kinds) < 3 || len(sizes) < 20 {
		t.Fatalf("infill lacks variety: %d kinds, %d sizes", len(kinds), len(sizes))
	}
	t.Logf("added %d original factory buildings, %d families, %d sizes", len(infill), len(kinds), len(sizes))
}

func TestParkInfillAvoidsTreesPathsRoadsAndGrove(t *testing.T) {
	city := testInfillCity()
	p := city.parks[0]
	paths := parkPaths(p)
	grove := maths.Box{Min: p.At.Add(maths.V(-4.2, 0, -3.2)), Max: p.At.Add(maths.V(4.2, 0, 3.2))}
	for _, one := range city.parkInfillPlan(sow(31)) {
		b := one.building
		if boxesOverlapXZ(b.Bounds, city.trees[0].Bounds) || boxesOverlapXZ(b.Bounds, grove) {
			t.Fatalf("infill reached tree or grove: %v", b.Bounds)
		}
		if city.parkInfillMeetsRoad(b.Bounds) || city.parkInfillMeetsCircle(b.Bounds) {
			t.Fatalf("infill blocks a city road: %v", b.Bounds)
		}
		radius := .5 * math.Hypot(b.Bounds.Max.X-b.Bounds.Min.X, b.Bounds.Max.Z-b.Bounds.Min.Z)
		if parkTerrainPathDistance(paths, b.Place.Origin.X, b.Place.Origin.Z) <= radius+.05 {
			t.Fatalf("infill covers path: %v", b.Bounds)
		}
		for _, at := range []maths.Vec{b.Bounds.Min, b.Bounds.Max, b.Place.Origin} {
			if h := CityParkTerrainHeight(p, at); h > one.base+1e-9 {
				t.Fatalf("terrain %g buries foundation base %g", h, one.base)
			}
		}
	}
}

func TestParkInfillPreservesAvenueAndMaterialDetails(t *testing.T) {
	city := testInfillCity()
	b := newCityBatch()
	city.addParkInfill(b, sow(31), "quick", true)
	// Both walkable avenues are present, alongside full-detail night windows.
	if len(b.pending[cityPavement]) < 4 && len(b.members[cityPavement]) == 0 {
		t.Fatal("pedestrian avenues disappeared")
	}
	if b.counts[cityWarmWindow] == 0 || b.counts[cityCoolWindow] == 0 {
		t.Fatal("infill lost occupied night windows")
	}
}

func TestParkInfillCitywideCoveragePreservesOriginalBuildings(t *testing.T) {
	spec := DefaultClusteredCity()
	spec.ParkRelief = true
	city, err := PlanClusteredCity(spec, sow(31))
	if err != nil {
		t.Fatal(err)
	}
	before := city.Buildings()
	if len(before) != 10000 {
		t.Fatalf("original buildings %d", len(before))
	}
	extra, candidates, err := city.vacantInfillPlan(sow(31), spec.Process.Spread.InMetres())
	if err != nil {
		t.Fatal(err)
	}
	if candidates == 0 || len(extra) < candidates/2 {
		t.Fatalf("vacant: candidates %d, added %d", candidates, len(extra))
	}
	seen := map[uint64]bool{}
	area := 0.0
	for _, b := range before {
		seen[b.Seed] = true
		area += (b.Bounds.Max.X - b.Bounds.Min.X) * (b.Bounds.Max.Z - b.Bounds.Min.Z)
	}
	baseArea := area
	for _, b := range extra {
		if seen[b.Seed] {
			t.Fatalf("duplicate or existing building seed %d", b.Seed)
		}
		seen[b.Seed] = true
		if b.Bounds.Min.X < b.Lot.Min.X || b.Bounds.Max.X > b.Lot.Max.X || b.Bounds.Min.Z < b.Lot.Min.Z || b.Bounds.Max.Z > b.Lot.Max.Z {
			t.Fatal("added building escaped lot")
		}
		area += (b.Bounds.Max.X - b.Bounds.Min.X) * (b.Bounds.Max.Z - b.Bounds.Min.Z)
	}
	terraces := city.urbanTerracePlan(extra)
	service, err := city.urbanServicePlan(sow(31), extra)
	if err != nil {
		t.Fatal(err)
	}
	serviceArea := 0.0
	for _, box := range service {
		serviceArea += (box.Max.X - box.Min.X) * (box.Max.Z - box.Min.Z)
	}
	terraceArea := 0.0
	for _, part := range terraces {
		terraceArea += (part.footprint.Max.X - part.footprint.Min.X) * (part.footprint.Max.Z - part.footprint.Min.Z)
	}
	parks := city.parkInfillPlan(sow(31))
	parkArea := 0.0
	for _, one := range parks {
		b := one.building.Bounds
		parkArea += (b.Max.X - b.Min.X) * (b.Max.Z - b.Min.Z)
	}
	grid := spec.Grid
	pitch := grid.BlockLength.InMetres()
	wx, wz := float64(grid.BlocksX)*pitch, float64(grid.BlocksZ)*pitch
	total := wx * wz
	road, alley := grid.RoadWidth.InMetres(), grid.AlleyWidth.InMetres()
	nominal := float64(grid.BlocksX+1)*road*wz + float64(grid.BlocksZ+1)*road*wx - float64((grid.BlocksX+1)*(grid.BlocksZ+1))*road*road + float64(grid.BlocksX*grid.BlocksZ)*(2*alley*(pitch-road)-alley*alley)
	if terraceArea <= 0 || (area+terraceArea) <= area {
		t.Fatal("district fabric absent")
	}
	t.Logf("seed 31, no trees: original=%d, vacant candidates=%d, added vacant=%d, park added=%d, terrace pieces=%d", len(before), candidates, len(extra), len(parks), len(terraces))
	t.Logf("axis-aligned areas / city area %.0f m²: original buildings %.2f%%; with vacant %.2f%%; plus disjoint lot terraces %.2f%%; park infill %.2f%%; nominal roads+alleys %.2f%% (not combined)", total, 100*baseArea/total, 100*area/total, 100*(area+terraceArea)/total, 100*parkArea/total, 100*nominal/total)
	t.Logf("unoccupied vacant-lot service pavement: %d tiles, %.2f%% of city area (includes pre-existing block pavement, excludes trees)", len(service), 100*serviceArea/total)
}

func TestParkInfillServiceTilesAvoidTreesAndBuiltLots(t *testing.T) {
	spec := DefaultClusteredCity()
	city, err := PlanClusteredCity(spec, sow(31))
	if err != nil {
		t.Fatal(err)
	}
	extra, _, err := city.vacantInfillPlan(sow(31), spec.Process.Spread.InMetres())
	if err != nil {
		t.Fatal(err)
	}
	lots, err := city.urbanVacantLots(sow(31))
	if err != nil {
		t.Fatal(err)
	}
	selected := map[uint64]bool{}
	for _, b := range extra {
		selected[b.Seed] = true
	}
	for _, lot := range lots {
		if !selected[lot.seed] {
			city.trees = []CityTree{{Bounds: maths.Box{Min: lot.lot.Min.Add(maths.V(.2, 0, .2)), Max: lot.lot.Min.Add(maths.V(1.2, 2, 1.2))}}}
			break
		}
	}
	if len(city.trees) == 0 {
		t.Fatal("missing unoccupied lot")
	}
	service, err := city.urbanServicePlan(sow(31), extra)
	if err != nil || len(service) == 0 {
		t.Fatalf("service tiles: %v, %d", err, len(service))
	}
	for _, box := range service {
		if boxesOverlapXZ(box, city.trees[0].Bounds) {
			t.Fatal("service tile covered a tree")
		}
	}
}

func TestParkInfillTerracesAvoidBuildingsAndTreeHulls(t *testing.T) {
	lot := maths.Box{Min: maths.V(0, 0, 0), Max: maths.V(20, 20, 20)}
	building := CityBuilding{Seed: 32, Lot: lot, Bounds: maths.Box{Min: maths.V(6, 0, 6), Max: maths.V(14, 10, 14)}}
	tree := CityTree{Bounds: maths.Box{Min: maths.V(1, 0, 1), Max: maths.V(2, 4, 2)}}
	city := ClusteredCityLayout{base: CityLayout{buildings: []CityBuilding{building}}, trees: []CityTree{tree}}
	terraces := city.urbanTerracePlan(nil)
	if len(terraces) == 0 || len(terraces) >= 4 {
		t.Fatalf("expected only tree-clear strips; got %d", len(terraces))
	}
	for i, a := range terraces {
		if boxesOverlapXZ(a.footprint, building.Bounds) || boxesOverlapXZ(a.footprint, tree.Bounds) {
			t.Fatal("terrace covers a building or tree")
		}
		for _, b := range terraces[:i] {
			if boxesOverlapXZ(a.footprint, b.footprint) {
				t.Fatal("terraces overlap")
			}
		}
	}
}

func TestParkInfillProductionTreeLayout(t *testing.T) {
	spec := greenClusteredCity(true)
	spec.ParkRelief = true
	spec.ParkPathWidth = measure.Metres(4.0 / 250)
	spec.Trees.SizeScale = 1.0 / 250
	spec.Trees.ParkEnrichment = true
	spec.Trees.UrbanRows = true
	city, err := PlanClusteredCity(spec, Sow(31))
	if err != nil {
		t.Fatal(err)
	}
	extra, candidates, err := city.vacantInfillPlan(sow(31), spec.Process.Spread.InMetres())
	if err != nil {
		t.Fatal(err)
	}
	park := city.parkInfillPlan(sow(31))
	terraces := city.urbanTerracePlan(extra)
	buckets := parkInfillTreeBuckets(city.trees)
	path := city.parks[0]
	start, end := CityParkLongPathAt(path, .5), CityParkLongPathAt(path, .5003)
	clearance := 25.0 / 250
	corridor := maths.Box{Min: maths.V(math.Min(start.X, end.X)-clearance, 0, math.Min(start.Z, end.Z)-clearance), Max: maths.V(math.Max(start.X, end.X)+clearance, 0, math.Max(start.Z, end.Z)+clearance)}
	for _, tree := range city.trees {
		if boxesOverlapXZ(tree.Bounds, corridor) {
			t.Fatalf("tree blocks camera path or climb: %+v", tree.Bounds)
		}
	}
	for _, building := range extra {
		if boxesOverlapXZ(building.Bounds, corridor) || building.Bounds.Max.Y >= 80000.0/250 {
			t.Fatalf("vacant-lot building blocks camera route: %+v", building.Bounds)
		}
		if parkInfillMeetsTrees(building.Bounds, buckets) {
			t.Fatalf("vacant-lot building overlaps a tree: %+v", building.Bounds)
		}
	}
	for _, building := range park {
		if boxesOverlapXZ(building.building.Bounds, corridor) || building.building.Bounds.Max.Y >= 80000.0/250 {
			t.Fatalf("park building blocks camera route: %+v", building.building.Bounds)
		}
		if parkInfillMeetsTrees(building.building.Bounds, buckets) {
			t.Fatalf("park building overlaps a tree: %+v", building.building.Bounds)
		}
	}
	for _, terrace := range terraces {
		if boxesOverlapXZ(terrace.footprint, corridor) || terrace.height >= 80000.0/250 {
			t.Fatalf("terrace blocks camera route: %+v", terrace.footprint)
		}
		if parkInfillMeetsTrees(terrace.footprint, buckets) {
			t.Fatalf("terrace overlaps a tree: %+v", terrace.footprint)
		}
	}
	service, err := city.urbanServicePlan(sow(31), extra)
	if err != nil {
		t.Fatal(err)
	}
	for _, tile := range service {
		if parkInfillMeetsTrees(tile, buckets) || boxesOverlapXZ(tile, corridor) {
			t.Fatalf("service tile overlaps trees or camera corridor: %+v", tile)
		}
	}
	if len(city.base.buildings) != 10000 || len(city.trees) < 7000 || len(extra) < 5000 || len(park) < 100 || len(terraces) < 10000 {
		t.Fatalf("production layout unexpectedly sparse: buildings=%d trees=%d vacant=%d park=%d terraces=%d", len(city.base.buildings), len(city.trees), len(extra), len(park), len(terraces))
	}
	t.Logf("seed 31, production trees: original=%d trees=%d vacant candidates=%d added vacant=%d park added=%d terraces=%d service tiles=%d", len(city.base.buildings), len(city.trees), candidates, len(extra), len(park), len(terraces), len(service))
}
func TestParkInfillTerracesMakeValidatedNativeMeshes(t *testing.T) {
	spec := DefaultClusteredCity()
	city, err := PlanClusteredCity(spec, Sow(31))
	if err != nil {
		t.Fatal(err)
	}
	infill, _, err := city.vacantInfillPlan(Sow(31), spec.Process.Spread.InMetres())
	if err != nil {
		t.Fatal(err)
	}
	batch := newCityBatch()
	city.addUrbanTerraces(batch, infill)
	if _, err := batch.finish(); err != nil {
		t.Fatal(err)
	}
}

func TestParkInfillFullCityMake(t *testing.T) {
	spec := DefaultClusteredCity()
	spec.ParkRelief = true
	spec.Night = true
	trees := DefaultCityTrees()
	spec.Trees = &trees
	row, err := ClusteredCity(spec)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Make(row, Sow(31)); err != nil {
		t.Fatal(err)
	}
}
