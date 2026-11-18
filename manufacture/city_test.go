package manufacture

import (
	"math"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
)

// A square kilometre count cannot substitute for ten thousand actual lots.
func TestCityPlacesTenThousandBuildingsWithStreetAndCircleClearance(t *testing.T) {
	started := time.Now()
	grid := DefaultCityGrid()
	layout, err := PlanCity(grid, Sow(31))
	if err != nil {
		t.Fatal(err)
	}
	buildings := layout.Buildings()
	if len(buildings) != 10_000 {
		t.Fatalf("buildings: %d", len(buildings))
	}
	if len(layout.Roundabouts()) != 16 {
		t.Fatalf("roundabouts: %d", len(layout.Roundabouts()))
	}
	seen := map[uint64]bool{}
	families := map[string]int{}
	cells := map[[2]int][]maths.Box{}
	roadHalf, setback := grid.RoadWidth.InMetres()/2, grid.Setback.InMetres()
	pitch := grid.BlockLength.InMetres()
	origin := -25 * pitch / 2
	for _, b := range buildings {
		if seen[b.Seed] {
			t.Fatalf("repeated seed: %d", b.Seed)
		}
		seen[b.Seed] = true
		families[b.Kind]++
		if b.Bounds.Min.X < b.Lot.Min.X || b.Bounds.Max.X > b.Lot.Max.X ||
			b.Bounds.Min.Z < b.Lot.Min.Z || b.Bounds.Max.Z > b.Lot.Max.Z {
			t.Fatalf("building escapes its lot: %+v", b)
		}
		for axis := range 2 {
			low, high := b.Bounds.Min.X, b.Bounds.Max.X
			if axis == 1 {
				low, high = b.Bounds.Min.Z, b.Bounds.Max.Z
			}
			for line := range 26 {
				street := origin + float64(line)*pitch
				if low < street+roadHalf+setback && high > street-roadHalf-setback {
					t.Fatalf("building %d intrudes on street %d", b.Seed, line)
				}
			}
		}
		for axis := range 2 {
			low, high := b.Bounds.Min.X, b.Bounds.Max.X
			if axis == 1 {
				low, high = b.Bounds.Min.Z, b.Bounds.Max.Z
			}
			block := math.Floor(((low+high)/2 - origin) / pitch)
			alley := origin + (block+.5)*pitch
			if low < alley+grid.AlleyWidth.InMetres()/2+setback && high > alley-grid.AlleyWidth.InMetres()/2-setback {
				t.Fatalf("building %d intrudes on access alley", b.Seed)
			}
		}
		for _, circle := range layout.Roundabouts() {
			dx := math.Max(b.Bounds.Min.X-circle.At.X, math.Max(0, circle.At.X-b.Bounds.Max.X))
			dz := math.Max(b.Bounds.Min.Z-circle.At.Z, math.Max(0, circle.At.Z-b.Bounds.Max.Z))
			if math.Hypot(dx, dz) < circle.Radius.InMetres()+setback {
				t.Fatalf("building %d intrudes on roundabout", b.Seed)
			}
		}
		cell := [2]int{int(math.Floor(b.Bounds.Min.X / 32)), int(math.Floor(b.Bounds.Min.Z / 32))}
		for x := cell[0] - 1; x <= cell[0]+1; x++ {
			for z := cell[1] - 1; z <= cell[1]+1; z++ {
				for _, other := range cells[[2]int{x, z}] {
					if b.Bounds.Min.X < other.Max.X && b.Bounds.Max.X > other.Min.X &&
						b.Bounds.Min.Z < other.Max.Z && b.Bounds.Max.Z > other.Min.Z {
						t.Fatal("buildings overlap")
					}
				}
			}
		}
		cells[cell] = append(cells[cell], b.Bounds)
	}
	if len(families) != 3 {
		t.Fatalf("family diversity: %v", families)
	}
	t.Logf("placed %d buildings, families %v, in %s", len(buildings), families,
		time.Since(started))
}

// anti-test of TestCityPlacesTenThousandBuildingsWithStreetAndCircleClearance
func TestCityRefusesImpossibleLotsAndUncheckedInputs(t *testing.T) {
	cases := []func(*CityGrid){
		func(g *CityGrid) { g.BlocksX = 0 },
		func(g *CityGrid) { g.BlocksZ = -1 },
		func(g *CityGrid) { g.BlocksX = 100_000 },
		func(g *CityGrid) { g.LotsAcross = 3 },
		func(g *CityGrid) { g.BlockLength = measure.Metres(50) },
		func(g *CityGrid) { g.RoadWidth = measure.Metres(0) },
		func(g *CityGrid) { g.AlleyWidth = measure.Metres(-1) },
		func(g *CityGrid) { g.Setback = measure.Metres(math.NaN()) },
		func(g *CityGrid) { g.RoundaboutRadius = measure.Metres(8) },
		func(g *CityGrid) { g.RoundaboutEvery = -1 },
		func(g *CityGrid) { g.BlockLength = measure.Metres(1e308) },
	}
	for i, change := range cases {
		grid := DefaultCityGrid()
		change(&grid)
		if _, err := City(grid); err == nil {
			t.Fatalf("accepted invalid city %d", i)
		}
		if _, err := PlanCity(grid, Sow(1)); err == nil {
			t.Fatalf("planned invalid city %d", i)
		}
	}
	if _, err := PlanCity(DefaultCityGrid(), seed{}); err == nil {
		t.Fatal("accepted unsown seed")
	}
}

func TestCityVariationIsDeterministicAndIndependentOfGridExtent(t *testing.T) {
	grid := DefaultCityGrid()
	first, _ := PlanCity(grid, Sow(21))
	same, _ := PlanCity(grid, Sow(21))
	other, _ := PlanCity(grid, Sow(22))
	if !reflect.DeepEqual(first.Buildings(), same.Buildings()) {
		t.Fatal("same seed changed")
	}
	if reflect.DeepEqual(first.Buildings(), other.Buildings()) {
		t.Fatal("different seeds agree")
	}
	grid.BlocksZ++
	extended, _ := PlanCity(grid, Sow(21))
	extendedBuildings := extended.Buildings()
	for i, b := range first.Buildings() {
		next := extendedBuildings[i]
		if next.Seed != b.Seed || next.Kind != b.Kind || next.Bounds.Max.Y != b.Bounds.Max.Y {
			t.Fatalf("extension changed existing lot %d", i)
		}
	}
	leaked := first.Buildings()
	leaked[0].Kind = "broken"
	if first.Buildings()[0].Kind == "broken" {
		t.Fatal("layout exposed its storage")
	}
}

func TestCityBakesTenThousandBuildingsIntoMaterialBatches(t *testing.T) {
	started := time.Now()
	row, err := KindNamed("city-grid-10000")
	if err != nil {
		t.Fatal(err)
	}
	made, err := Make(row, Sow(31))
	if err != nil {
		t.Fatal(err)
	}
	things, err := made.Things(maths.Local())
	if err != nil {
		t.Fatal(err)
	}
	if len(things) > 10 {
		t.Fatalf("unbatched materials: %d", len(things))
	}
	meshes, faces := 0, 0
	for _, part := range things {
		group, ok := part.Shape.(*shape.MeshGroup)
		if !ok {
			t.Fatalf("city material is not batched: %T", part.Shape)
		}
		meshes += len(group.Members())
		faces += group.StoredFaces()
	}
	if meshes >= 100 {
		t.Fatalf("city made %d acceleration meshes", meshes)
	}
	if faces < 120_000 {
		t.Fatalf("missing building surfaces: %d", faces)
	}
	t.Logf("10,000 buildings: %d materials, %d mesh batches, %d faces, %s", len(things),
		meshes, faces, time.Since(started))
}

func TestCityBuildingFamiliesHaveValidOutwardGeometryAndSeededVariation(t *testing.T) {
	for _, name := range []string{"city-rowhouse", "city-apartment", "city-tower"} {
		row, err := KindNamed(name)
		if err != nil {
			t.Fatal(err)
		}
		var previous maths.Box
		for number := uint64(0); number < 12; number++ {
			made, err := Make(row, Sow(number))
			if err != nil {
				t.Fatal(name, err)
			}
			if _, err := made.Things(maths.Local()); err != nil {
				t.Fatal(name, err)
			}
			bounds := made.Bounds()
			if bounds.Min.Y != 0 || bounds.Max.Y <= 6 || bounds.Max.X-bounds.Min.X > 20 ||
				bounds.Max.Z-bounds.Min.Z > 20 {
				t.Fatalf("%s bounds: %+v", name, bounds)
			}
			if number > 0 && bounds == previous {
				t.Fatal("family has no seed variation")
			}
			previous = bounds
		}
	}
}

// Streets join all four approaches without drawing asphalt across islands.
func TestCityRoundaboutsConnectApproachesAndKeepTheCentreRoadFree(t *testing.T) {
	layout, err := PlanCity(DefaultCityGrid(), Sow(31))
	if err != nil {
		t.Fatal(err)
	}
	batch := newCityBatch()
	layout.paving(batch)
	made, err := batch.finish()
	if err != nil {
		t.Fatal(err)
	}
	road, island := made.groups[cityRoad], made.groups[cityIsland]
	at := layout.Roundabouts()[0].At
	hit := func(group *shape.MeshGroup, x, z float64) bool {
		_, found := group.Hit(maths.Ray{At: at.Add(maths.V(x, 10, z)), Dir: maths.V(0, -1,
			0)}, 0, 20)
		return found
	}
	if hit(road, 0, 0) || !hit(island, 0, 0) {
		t.Fatal("traffic island has a through road or no raised surface")
	}
	for _, offset := range [][2]float64{{13, 0}, {-13, 0}, {0, 13}, {0, -13}, {22, 0},
		{-22, 0}, {0, 22}, {0, -22}} {
		if !hit(road, offset[0], offset[1]) {
			t.Fatalf("disconnected approach at %v", offset)
		}
	}
	if hit(road, 17, 8) {
		t.Fatal("circle fills its outside corner")
	}
}

func TestCityRepresentationsRetainEnvelopesAndGableVolume(t *testing.T) {
	for _, name := range []string{"city-rowhouse", "city-apartment", "city-tower"} {
		row, _ := KindNamed(name)
		hull, err := MakeWithComplexity(row, Sow(8), "hull")
		if err != nil {
			t.Fatal(err)
		}
		quick, err := Make(row, Sow(8))
		if err != nil {
			t.Fatal(err)
		}
		if hull.Bounds() != quick.Bounds() {
			t.Fatalf("%s changes physical envelope", name)
		}
		if _, err := MakeWithComplexity(row, Sow(8), "clumps"); err == nil {
			t.Fatal("accepted unsupported representation")
		}
		if _, err := MakeWithComplexity(row, seed{}, "hull"); err == nil {
			t.Fatal("accepted unsown seed")
		}
	}
	roof := cityGable{width: 12, depth: 10, rise: 2}
	volume := 0.0
	for _, face := range roof.skin(cut{}) {
		volume += face[0].Dot(face[1].Cross(face[2])) / 6
	}
	if math.Abs(volume-120) > 1e-10 {
		t.Fatalf("gable volume %v, triangle prism formula gives 120", volume)
	}
}

// anti-test of TestCityBuildingFamiliesHaveValidOutwardGeometryAndSeededVariation
func TestCityRecipeRefusesInvalidDimensionsUnitsCountsAndBudgets(t *testing.T) {
	source := string(cityChoices)
	broken := []string{
		"numbers: [",
		strings.Replace(source, "value: 128", "value: -128", 1),
		strings.Replace(source, "unit: m", "unit: km", 1),
		strings.Replace(source, "value: 25", "value: 2.5", 1),
		strings.Replace(source, "value: 60000", "value: 600000", 1),
		strings.Replace(source, "name: width-min\n  value: 10", "name: width-min\n  value: 19", 1),
		strings.Replace(source, "value: 0.18", "value: 1.8", 1),
	}
	for i, raw := range broken {
		if _, err := readCityChoices([]byte(raw)); err == nil {
			t.Fatalf("accepted invalid recipe %d", i)
		}
	}
}

func TestCityCustomRecipeAndHullKeepBuildingIdentities(t *testing.T) {
	grid := DefaultCityGrid()
	grid.BlocksX, grid.BlocksZ, grid.LotsAcross = 2, 3, 2
	grid.RoundaboutEvery = 0
	row, err := City(grid)
	if err != nil {
		t.Fatal(err)
	}
	layout, err := PlanCity(grid, Sow(42))
	if err != nil {
		t.Fatal(err)
	}
	if len(layout.Buildings()) != 24 || len(layout.Roundabouts()) != 0 {
		t.Fatal("custom population or junctions")
	}
	hull, err := MakeWithComplexity(row, Sow(42), "hull")
	if err != nil {
		t.Fatal(err)
	}
	quick, err := Make(row, Sow(42))
	if err != nil {
		t.Fatal(err)
	}
	if hull.Bounds() != quick.Bounds() || len(quick.Descriptions()) != 0 {
		t.Fatal("custom envelope or lights")
	}
	for _, building := range layout.Buildings() {
		family, _ := KindNamed(building.Kind)
		made, err := Make(family, Sow(building.Seed))
		if err != nil {
			t.Fatal(err)
		}
		if transformed(made.Bounds(), building.Place) != building.Bounds {
			t.Fatal("lot does not recreate its specimen")
		}
	}
	if _, err := MakeWithComplexity(row, Sow(42), "clumps"); err == nil {
		t.Fatal("unsupported city detail")
	}
	if _, err := MakeWithComplexity(row, seed{}, "hull"); err == nil {
		t.Fatal("unsown custom city")
	}
}
