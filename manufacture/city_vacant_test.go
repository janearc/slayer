package manufacture

import (
	"reflect"
	"testing"
)

func TestClusteredCityVacantLandscapeMatchesFullGridScan(t *testing.T) {
	seed := Sow(31)
	defaultLayout, err := PlanClusteredCity(greenClusteredCity(false), seed)
	if err != nil {
		t.Fatal(err)
	}
	onSpec := greenClusteredCity(false)
	onSpec.VacantLotIndex = CityVacantLotIndexOn
	onLayout, err := PlanClusteredCity(onSpec, seed)
	if err != nil {
		t.Fatal(err)
	}
	offSpec := greenClusteredCity(false)
	offSpec.VacantLotIndex = CityVacantLotIndexOff
	offLayout, err := PlanClusteredCity(offSpec, seed)
	if err != nil {
		t.Fatal(err)
	}
	plain, err := PlanClusteredCity(DefaultClusteredCity(), seed)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(defaultLayout.Buildings(), plain.Buildings()) ||
		!reflect.DeepEqual(defaultLayout.Buildings(), onLayout.Buildings()) ||
		!reflect.DeepEqual(defaultLayout.Buildings(), offLayout.Buildings()) {
		t.Fatal("vacant-lot caching changed selected buildings")
	}
	all, err := PlanCity(defaultLayout.base.grid, seed)
	if err != nil {
		t.Fatal(err)
	}
	occupied := make(map[uint64]bool, len(defaultLayout.base.buildings))
	for _, building := range defaultLayout.base.buildings {
		occupied[building.Seed] = true
	}

	want := newCityBatch()
	var vacant []cityVacantLot
	marginOnly := 0
	for _, lot := range all.buildings {
		if occupied[lot.Seed] || defaultLayout.reserved(lot.Lot, 0) {
			continue
		}
		if defaultLayout.reserved(lot.Lot, defaultLayout.base.grid.Setback.InMetres()) {
			marginOnly++
		}
		vacant = append(vacant, cityVacantLot{seed: lot.Seed, lot: lot.Lot})
		role := cityVacantSoil
		if sow(lot.Seed).draw(141, 0, 0) < .65 {
			role = cityVacantMeadow
		}
		want.rectangle(role, lot.Lot.Min.X, lot.Lot.Min.Z,
			lot.Lot.Max.X-lot.Lot.Min.X, lot.Lot.Max.Z-lot.Lot.Min.Z, -.005)
	}
	if marginOnly == 0 {
		t.Fatal("fixture has no vacant lot inside a reserved clearance margin")
	}
	if !reflect.DeepEqual(defaultLayout.vacant, vacant) || !reflect.DeepEqual(onLayout.vacant, vacant) {
		t.Fatal("cached vacant lots differ from the full-grid scan")
	}
	if len(offLayout.vacant) != 0 {
		t.Fatal("rescan mode retained vacant-lot descriptors")
	}
	t.Logf("cached %d vacant lot records at capacity %d", len(defaultLayout.vacant), cap(defaultLayout.vacant))

	for _, current := range []struct {
		name string
		city ClusteredCityLayout
	}{{"default", defaultLayout}, {"on", onLayout}, {"off", offLayout}} {
		got := newCityBatch()
		if err := current.city.vacantLandscape(got, seed); err != nil {
			t.Fatalf("%s landscape: %v", current.name, err)
		}
		for _, role := range []slot{cityVacantSoil, cityVacantMeadow} {
			if !reflect.DeepEqual(got.pending[role], want.pending[role]) {
				t.Fatalf("%s vacant material %d changed lot order or geometry", current.name, role)
			}
		}
	}
}

func TestCityVacantLotIndexRejectsUnsupportedModesAndKinds(t *testing.T) {
	green, err := ClusteredCity(greenClusteredCity(false))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := WithCityVacantLotIndex(green, "sideways"); err == nil {
		t.Fatal("unknown mode accepted")
	}
	plain, err := ClusteredCity(DefaultClusteredCity())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := WithCityVacantLotIndex(plain, CityVacantLotIndexOff); err == nil {
		t.Fatal("tree-free clustered city accepted the setting")
	}
	grid, err := City(DefaultCityGrid())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := WithCityVacantLotIndex(grid, CityVacantLotIndexOff); err == nil {
		t.Fatal("grid city accepted the setting")
	}
	spec := DefaultClusteredCity()
	spec.VacantLotIndex = CityVacantLotIndexOn
	if _, err := PlanClusteredCity(spec, Sow(31)); err == nil {
		t.Fatal("explicit mode without trees accepted")
	}
	spec.VacantLotIndex = "sideways"
	spec.Trees = &CityTreeSpec{}
	if _, err := PlanClusteredCity(spec, Sow(31)); err == nil {
		t.Fatal("unknown spec mode accepted")
	}
}

func BenchmarkClusteredCityPlanAndVacantLandscape(b *testing.B) {
	seed := Sow(31)
	for _, mode := range []CityVacantLotIndexMode{CityVacantLotIndexOn, CityVacantLotIndexOff} {
		b.Run(string(mode), func(b *testing.B) {
			green := greenClusteredCity(false)
			green.VacantLotIndex = mode
			b.ReportAllocs()
			for range b.N {
				layout, err := PlanClusteredCity(green, seed)
				if err != nil {
					b.Fatal(err)
				}
				if err := layout.vacantLandscape(newCityBatch(), seed); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkClusteredCityPlanWithoutTrees(b *testing.B) {
	seed := Sow(31)
	plain := DefaultClusteredCity()
	b.ReportAllocs()
	for range b.N {
		if _, err := PlanClusteredCity(plain, seed); err != nil {
			b.Fatal(err)
		}
	}
}
