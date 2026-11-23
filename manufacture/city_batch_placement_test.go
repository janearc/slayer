package manufacture

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
)

func TestCityBatchAddMatchesNestedPlacementExactly(test *testing.T) {
	rows := cityBatchPlacementFixture()
	legacy := newCityBatch()
	legacy.limit = 180
	for _, row := range rows {
		legacyCityBatchAdd(legacy, row.parts, row.place)
	}
	want, err := legacy.finish()
	if err != nil {
		test.Fatal(err)
	}
	for _, mode := range []CityBatchPlacementMode{"", CityBatchFused, CityBatchNested} {
		current := newCityBatch()
		current.placement = mode
		current.limit = 180
		for _, row := range rows {
			current.add(row.parts, row.place)
		}
		got, err := current.finish()
		if err != nil {
			test.Fatal(err)
		}
		assertCityBatchEqual(test, got, want)
	}
}

func assertCityBatchEqual(test *testing.T, got, want good) {
	test.Helper()
	if len(got.groups) != len(want.groups) || len(got.groupParts) != len(want.groupParts) {
		test.Fatalf("material groups/counts = %d/%d, want %d/%d", len(got.groups), len(got.groupParts), len(want.groups), len(want.groupParts))
	}
	for role, expected := range want.groups {
		actual := got.groups[role]
		if actual == nil {
			test.Fatalf("missing material group %d", role)
		}
		if got.groupParts[role] != want.groupParts[role] {
			test.Fatalf("material %d part count = %d, want %d", role, got.groupParts[role], want.groupParts[role])
		}
		wantMembers, gotMembers := expected.Members(), actual.Members()
		if len(gotMembers) != len(wantMembers) {
			test.Fatalf("material %d chunk count = %d, want %d", role, len(gotMembers), len(wantMembers))
		}
		for index := range wantMembers {
			wantMesh, wantOK := wantMembers[index].Mesh.(*mesh.Mesh)
			gotMesh, gotOK := gotMembers[index].Mesh.(*mesh.Mesh)
			if !wantOK || !gotOK {
				test.Fatalf("material %d member %d mesh types = %T/%T", role, index, wantMembers[index].Mesh, gotMembers[index].Mesh)
			}
			_, wantSmooth := wantMesh.CornerNormals(0)
			_, gotSmooth := gotMesh.CornerNormals(0)
			wantPoints, gotPoints := wantMesh.Points(), gotMesh.Points()
			wantFaces, gotFaces := wantMesh.Faces(), gotMesh.Faces()
			if wantSmooth != gotSmooth || len(wantFaces) != len(gotFaces) || len(wantPoints) != len(gotPoints) {
				test.Fatalf("material %d member %d mesh structure differs", role, index)
			}
			for corner := range wantPoints {
				if gotPoints[corner] != wantPoints[corner] {
					test.Fatalf("material %d member %d corner %d changed: got %+v want %+v", role, index, corner, gotPoints[corner], wantPoints[corner])
				}
			}
			for face := range wantFaces {
				if gotFaces[face] != wantFaces[face] {
					test.Fatalf("material %d member %d face %d changed", role, index, face)
				}
			}
			if gotMembers[index].Space != wantMembers[index].Space {
				test.Fatalf("material %d member %d placement changed", role, index)
			}
		}
	}
}

func TestCityBatchAddRetainsMeshFailure(test *testing.T) {
	bad := part{shape: cityBatchFailureForm{}, space: maths.NewSubspace(
		maths.V(2, 3, 4), maths.About(maths.YAxis, .37), 1.2), of: cityPavement}
	place := maths.NewSubspace(maths.V(-5, 7, 11), maths.About(maths.ZAxis, -.21), .8)
	legacy := newCityBatch()
	legacy.limit = 1
	legacyCityBatchAdd(legacy, []part{bad}, place)
	_, wantErr := legacy.finish()
	for _, mode := range []CityBatchPlacementMode{"", CityBatchFused, CityBatchNested} {
		current := newCityBatch()
		current.placement, current.limit = mode, 1
		current.add([]part{bad}, place)
		_, gotErr := current.finish()
		if wantErr == nil || gotErr == nil || gotErr.Error() != wantErr.Error() {
			test.Fatalf("mode %q mesh failures differ: got %v, want %v", mode, gotErr, wantErr)
		}
	}
}

func TestCityAndClusteredCityHonorBatchPlacement(test *testing.T) {
	grid := DefaultCityGrid()
	grid.BlocksX, grid.BlocksZ, grid.RoundaboutEvery = 1, 1, 0
	seed := Sow(73)
	var cityOutputs []good
	for _, mode := range []CityBatchPlacementMode{"", CityBatchFused, CityBatchNested} {
		grid.BatchPlacement = mode
		recipe, err := City(grid)
		if err != nil {
			test.Fatal(err)
		}
		made, err := recipe.how.(cityMaker).made(seed)
		if err != nil {
			test.Fatal(err)
		}
		cityOutputs = append(cityOutputs, made)
	}
	assertCityBatchEqual(test, cityOutputs[0], cityOutputs[1])
	assertCityBatchEqual(test, cityOutputs[0], cityOutputs[2])

	spec := DefaultClusteredCity()
	spec.Grid.BlocksX, spec.Grid.BlocksZ, spec.Grid.RoundaboutEvery = 1, 1, 0
	spec.Count = 1
	spec.Process.Width = spec.Grid.BlockLength
	spec.Process.Depth = spec.Grid.BlockLength
	spec.Parks, spec.Highways = nil, nil
	var districtOutputs []good
	for _, mode := range []CityBatchPlacementMode{"", CityBatchFused, CityBatchNested} {
		spec.Grid.BatchPlacement = mode
		recipe, err := ClusteredCity(spec)
		if err != nil {
			test.Fatal(err)
		}
		made, err := recipe.how.(clusteredCityMaker).made(seed)
		if err != nil {
			test.Fatal(err)
		}
		districtOutputs = append(districtOutputs, made)
	}
	assertCityBatchEqual(test, districtOutputs[0], districtOutputs[1])
	assertCityBatchEqual(test, districtOutputs[0], districtOutputs[2])
}

func TestCityConstructorsRejectUnknownBatchPlacement(test *testing.T) {
	grid := DefaultCityGrid()
	grid.BatchPlacement = "almost-fused"
	if _, err := City(grid); err == nil {
		test.Fatal("City accepted an unknown batch placement")
	}
	spec := DefaultClusteredCity()
	spec.Grid.BatchPlacement = "almost-fused"
	if _, err := ClusteredCity(spec); err == nil {
		test.Fatal("ClusteredCity accepted an unknown batch placement")
	}
}

func TestWithCityBatchPlacementCopiesOnlyNamedCityKinds(t *testing.T) {
	for _, name := range []string{"city-grid-10000", "city-clustered-10000",
		"city-clustered-green-10000", "city-clustered-night-10000"} {
		original, err := KindNamed(name)
		if err != nil {
			t.Fatal(err)
		}
		configured, err := WithCityBatchPlacement(original, CityBatchNested)
		if err != nil {
			t.Fatal(err)
		}
		if configured.name != original.name {
			t.Fatalf("kind name changed from %q to %q", original.name, configured.name)
		}
		switch how := configured.how.(type) {
		case cityMaker:
			if how.grid.BatchPlacement != CityBatchNested {
				t.Fatalf("%s placement mode = %q", name, how.grid.BatchPlacement)
			}
		case clusteredCityMaker:
			if how.spec.Grid.BatchPlacement != CityBatchNested {
				t.Fatalf("%s placement mode = %q", name, how.spec.Grid.BatchPlacement)
			}
		default:
			t.Fatalf("%s returned unsupported maker %T", name, configured.how)
		}
		switch how := original.how.(type) {
		case cityMaker:
			if how.grid.BatchPlacement != "" {
				t.Fatalf("configuring %s mutated the catalogue kind", name)
			}
		case clusteredCityMaker:
			if how.spec.Grid.BatchPlacement != "" {
				t.Fatalf("configuring %s mutated the catalogue kind", name)
			}
		}
	}
	if _, err := WithCityBatchPlacement(Kind{}, CityBatchFused); err == nil {
		t.Fatal("accepted an empty kind")
	}
	boulder, err := KindNamed("boulder")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := WithCityBatchPlacement(boulder, CityBatchFused); err == nil {
		t.Fatal("accepted a non-city kind")
	}
	city, err := KindNamed("city-grid-10000")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := WithCityBatchPlacement(city, "sideways"); err == nil {
		t.Fatal("accepted an unknown mode")
	}
}

// BenchmarkCityBatchPlacementAllocations compares the old nested-surface path
// with the fused append on the same bounded 192-building fixture. It measures
// transient construction allocations; retained mesh geometry is equivalent.
func BenchmarkCityBatchPlacementAllocations(benchmark *testing.B) {
	rows := cityBatchPlacementFixture()
	for _, run := range []struct {
		name string
		add  func(*cityBatch, []part, maths.Subspace)
		mode CityBatchPlacementMode
	}{
		{"legacy_nested_surfaces", legacyCityBatchAdd, ""},
		{"api_nested", func(batch *cityBatch, parts []part, place maths.Subspace) { batch.add(parts, place) }, CityBatchNested},
		{"api_fused", func(batch *cityBatch, parts []part, place maths.Subspace) { batch.add(parts, place) }, CityBatchFused},
	} {
		benchmark.Run(run.name, func(benchmark *testing.B) {
			benchmark.ReportAllocs()
			for iteration := 0; iteration < benchmark.N; iteration++ {
				batch := newCityBatch()
				batch.placement = run.mode
				batch.limit = 180
				for _, row := range rows {
					run.add(batch, row.parts, row.place)
				}
				if _, err := batch.finish(); err != nil {
					benchmark.Fatal(err)
				}
			}
		})
	}
}

type cityBatchPlacementRow struct {
	parts []part
	place maths.Subspace
}

func cityBatchPlacementFixture() []cityBatchPlacementRow {
	rows := make([]cityBatchPlacementRow, 0, 192)
	for index := range 192 {
		family := []string{"city-rowhouse", "city-apartment", "city-tower"}[index%3]
		seed := citySeed(uint64(index + 1))
		design := cityBuildingDesign(family, sow(seed))
		angle := float64(index%17) * .031
		corner := maths.V(float64(index%16)*37, 0, float64(index/16)*41)
		place := maths.NewSubspace(corner,
			maths.About(maths.YAxis, angle), .9+float64(index%5)*.05)
		rows = append(rows, cityBatchPlacementRow{parts: design.parts("quick"), place: place})
	}
	return rows
}

func legacyCityBatchAdd(batch *cityBatch, parts []part, place maths.Subspace) {
	for _, piece := range parts {
		batch.skin(piece.of, piece.skin().placed(place))
	}
}

type cityBatchFailureForm struct{}

func (cityBatchFailureForm) skin(cut) surface {
	return surface{{maths.V(math.NaN(), 0, 0), maths.V(1, 0, 0), maths.V(0, 1, 0)}}
}
