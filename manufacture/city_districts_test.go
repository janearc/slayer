package manufacture

import (
	"math"
	"reflect"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
)

func TestNeymanScottUsesPoissonParentsAndCorrectOffspringDispersions(t *testing.T) {
	spec := DefaultClusteredCity().Process
	spec.Width, spec.Depth = measure.Metres(4000), measure.Metres(4000)
	spec.ParentSpacing, spec.Spread = measure.Metres(500), measure.Metres(100)
	spec.MeanOffspring = 120
	for _, pattern := range []ClusterPattern{ThomasGaussian, NeymanScottDisk} {
		spec.Pattern = pattern
		sample, err := SampleNeymanScott(spec, Sow(17))
		if err != nil {
			t.Fatal(err)
		}
		parents := map[uint64]maths.Vec{}
		for _, parent := range sample.Parents() {
			parents[parent.Seed] = parent.At
		}
		radial2 := 0.0
		seen := map[uint64]bool{}
		for _, child := range sample.Offspring() {
			if seen[child.Seed] {
				t.Fatal("duplicate child seed")
			}
			seen[child.Seed] = true
			distance := child.At.Sub(parents[child.Parent]).Len()
			radial2 += distance * distance
			if pattern == NeymanScottDisk && distance > 100.0000001 {
				t.Fatal("disk offspring outside radius")
			}
		}
		radial2 /= float64(len(sample.offspring))
		expected := 20000.0
		if pattern == NeymanScottDisk {
			expected = 5000
		}
		if math.Abs(radial2/expected-1) > .08 {
			t.Fatalf("%s radial second moment %v, expected %v", pattern, radial2, expected)
		}
		same, _ := SampleNeymanScott(spec, Sow(17))
		if !reflect.DeepEqual(sample, same) {
			t.Fatal("unstable process")
		}
	}
	parentCounts := map[int]bool{}
	for seed := uint64(0); seed < 8; seed++ {
		sample, _ := SampleNeymanScott(spec, Sow(seed))
		parentCounts[len(sample.parents)] = true
	}
	if len(parentCounts) < 3 {
		t.Fatal("parent counts are fixed, not Poisson")
	}
}

func TestClusteredCityKeepsExactlyTenThousandBuildingsClearOfReservedLand(t *testing.T) {
	spec := DefaultClusteredCity()
	layout, err := PlanClusteredCity(spec, Sow(31))
	if err != nil {
		t.Fatal(err)
	}
	if len(layout.Buildings()) != 10000 || len(layout.parks) != 2 || len(layout.ramps) != 4 {
		t.Fatal("population or reserved infrastructure")
	}
	seen := map[uint64]bool{}
	for _, building := range layout.Buildings() {
		if seen[building.Seed] {
			t.Fatal("duplicate building seed")
		}
		seen[building.Seed] = true
		if layout.reserved(building.Bounds, 0) {
			t.Fatal("building intrudes on park, highway or ramp")
		}
	}
	same, _ := PlanClusteredCity(spec, Sow(31))
	if !reflect.DeepEqual(layout.Buildings(), same.Buildings()) {
		t.Fatal("city changed for identical inputs")
	}
	buildings := len(layout.Buildings())
	t.Logf("%d buildings, %d Poisson district centres, %d parks, %d ground ramps", buildings, len(layout.parents), len(layout.parks), len(layout.ramps))
	row, err := ClusteredCity(spec)
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
	batches, faces := 0, 0
	for _, part := range things {
		group := part.Shape.(*shape.MeshGroup)
		batches += len(group.Members())
		faces += group.StoredFaces()
	}
	if batches > 100 {
		t.Fatal("unbatched district geometry", batches)
	}
	t.Logf("%d material meshes, %d native mesh batches, %d faces", len(things), batches, faces)
}

// anti-test of TestClusteredCityKeepsExactlyTenThousandBuildingsClearOfReservedLand
func TestClusteredCityRefusesAnImpossibleReservedPopulation(t *testing.T) {
	spec := DefaultClusteredCity()
	spec.Parks = []CityPark{{At: maths.Vec{}, Width: measure.Metres(10000), Depth: measure.Metres(10000)}}
	if _, err := PlanClusteredCity(spec, Sow(31)); err == nil {
		t.Fatal("buildings populated an all-park city")
	}
	spec = DefaultClusteredCity()
	spec.Process.Spread = measure.Metres(0)
	if _, err := ClusteredCity(spec); err == nil {
		t.Fatal("zero cluster dispersion")
	}
}

func TestClusteredHighwayRampsJoinDecksAndActualGroundStreets(t *testing.T) {
	layout, err := PlanClusteredCity(DefaultClusteredCity(), Sow(31))
	if err != nil {
		t.Fatal(err)
	}
	for _, ramp := range layout.ramps {
		if ramp.at(0) != ramp.Points[0] || ramp.at(1) != ramp.Points[3] || ramp.at(0).Y != 8 || ramp.at(1).Y != .04 {
			t.Fatal("disconnected ramp elevations")
		}
		if layout.streetClear(ramp.at(1)) {
			t.Fatal("ramp ends off the local road network")
		}
		if _, err := (cityRampDeck{ramp}).skin(cut{}).meshed(); err != nil {
			t.Fatal("invalid connected ramp geometry", err)
		}
	}
}

func TestClusteredNativeTreesAndNightWindowsStayBatchedAndVaried(t *testing.T) {
	spec := greenClusteredCity(true)
	layout, err := PlanClusteredCity(spec, Sow(31))
	if err != nil {
		t.Fatal(err)
	}
	trees := layout.Trees()
	if len(trees) < 1000 {
		t.Fatal("missing street rows and park groves", len(trees))
	}
	seeds, prototypes := map[uint64]bool{}, map[uint64]bool{}
	buildings := layout.Buildings()
	for i, tree := range trees {
		if seeds[tree.Seed] {
			t.Fatal("duplicate planting identity")
		}
		seeds[tree.Seed] = true
		prototypes[tree.PrototypeSeed] = true
		if i%17 == 0 {
			for _, building := range buildings {
				a, b := tree.Bounds, building.Bounds
				if a.Min.X < b.Max.X && a.Max.X > b.Min.X && a.Min.Z < b.Max.Z && a.Max.Z > b.Min.Z {
					t.Fatal("tree crown enters a building")
				}
			}
		}
	}
	if len(prototypes) > 4 {
		t.Fatal("unbounded tree prototypes")
	}
	row, err := ClusteredCity(spec)
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
	unique := map[shape.Shape]bool{}
	for _, part := range things {
		for _, member := range part.Shape.(*shape.MeshGroup).Members() {
			unique[member.Mesh] = true
		}
	}
	if len(unique) > 150 {
		t.Fatal("trees became independent native acceleration structures", len(unique))
	}
	if len(made.Descriptions()) != 6 {
		described := len(made.Descriptions())
		t.Fatal("night needs six sparse actual fixtures", described)
	}
	if made.groups[cityWarmWindow] == nil || made.groups[cityCoolWindow] == nil || made.groups[cityGlass] == nil {
		t.Fatal("night facade lacks occupied and unlit variation")
	}
	lamps := len(made.Descriptions())
	t.Logf("10000 buildings, %d native trees, %d tree prototypes, %d unique native meshes, %d stored faces, %d actual lamps", len(trees), len(prototypes), len(unique), made.faces(), lamps)
}
