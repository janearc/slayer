package cities

import (
	"slices"
	"testing"
)

// The urban village: its houses carry the parts the culture says, no two
// houses alike, packed on a parcel with gaps between them; and what it
// refuses. Written 2026-11-20 beside the code and not yet run.

// villageSeeds are the seeds the house tests draw: eight, as the study's
// row of eight.
var villageSeeds = []Seed{1, 2, 3, 5, 8, 13, 21, 34}

// drawnHouse is the house of a seed drawn at a level, or the test's end.
func drawnHouse(t *testing.T, seed Seed, level Detail) []Rendered {
	t.Helper()
	design, err := VillageHouseAt(seed, nil, level)
	if err != nil {
		t.Fatal(err)
	}
	drawn, err := design.Rendered(level)
	if err != nil {
		t.Fatal(err)
	}
	return drawn
}

// partNames are the names of the meshes drawn, each once.
func partNames(drawn []Rendered) []string {
	var names []string
	for _, one := range drawn {
		if !slices.Contains(names, one.Mesh.Name) {
			names = append(names, one.Mesh.Name)
		}
	}
	return names
}

// TestVillageHousesCarryTheirParts checks that the eight houses between
// them, at the demo level, have every part the brief names as a real
// part: cage bars, air conditioners with drip pipes, clothes poles, water
// tanks, and a lamp over a stair door.
func TestVillageHousesCarryTheirParts(t *testing.T) {
	var names []string
	for _, seed := range villageSeeds {
		house := drawnHouse(t, seed, demoLevel)
		names = append(names, partNames(house)...)
	}
	for _, part := range []string{"cage bar", "air conditioner",
		"drip pipe", "clothes pole", "water tank", "door lamp", "grille"} {
		if !slices.Contains(names, part) {
			t.Errorf("no house of the eight has a %s", part)
		}
	}
}

// TestVillageHousesDiffer checks that the eight houses are not one house
// eight times: their heights are at least four different heights.
func TestVillageHousesDiffer(t *testing.T) {
	var heights []float64
	for _, seed := range villageSeeds {
		drawing, err := villageDrawer(seed, quickLevel)
		if err != nil {
			t.Fatal(err)
		}
		house, err := drawing.villageHouse(0, rect{0, 0, 12, 12},
			[sidesAround]bool{frontSide: true}, nil, seed)
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Contains(heights, house.Height) {
			heights = append(heights, house.Height)
		}
	}
	const leastDifferent = 4
	if len(heights) < leastDifferent {
		t.Errorf("eight houses have %d heights, %v", len(heights), heights)
	}
}

// TestVillageHouseSharesMeshes checks that a house's cages and bars are
// a few meshes placed many times: far fewer distinct sets of triangles
// than parts.
func TestVillageHouseSharesMeshes(t *testing.T) {
	drawn := drawnHouse(t, 3, quickLevel)
	var distinct [][]Triangle
	for _, one := range drawn {
		if !slices.ContainsFunc(distinct, func(known []Triangle) bool {
			return slices.Equal(known, one.Mesh.Triangles)
		}) {
			distinct = append(distinct, one.Mesh.Triangles)
		}
	}
	const sharing = 4
	if len(distinct)*sharing > len(drawn) {
		t.Errorf("%d parts are %d meshes, fewer than %d places a mesh",
			len(drawn), len(distinct), sharing)
	}
}

// TestVillageHullIsCoarser checks that a house drawn at the hull has no
// cage, no sign and fewer parts than at the quick level.
func TestVillageHullIsCoarser(t *testing.T) {
	hull, near := drawnHouse(t, 5, hullLevel), drawnHouse(t, 5, quickLevel)
	if len(hull) >= len(near) {
		t.Errorf("the hull has %d parts, the quick level %d", len(hull),
			len(near))
	}
	if slices.Contains(partNames(hull), "cage bar") {
		t.Error("a house at the hull has cage bars")
	}
}

// TestVillagePacksHousesApart checks that a village's houses stand on
// lots that do not meet, each a gap from every other.
func TestVillagePacksHousesApart(t *testing.T) {
	village, err := NewVillage(VillageAsk{}, 7, nil)
	if err != nil {
		t.Fatal(err)
	}
	const leastHouses = 12
	if len(village.Houses) < leastHouses {
		t.Fatalf("a parcel of the culture's size has %d houses",
			len(village.Houses))
	}
	for index, one := range village.Houses {
		for _, other := range village.Houses[index+1:] {
			if one.Lot.X0 < other.Lot.X1 && other.Lot.X0 < one.Lot.X1 &&
				one.Lot.Z0 < other.Lot.Z1 && other.Lot.Z0 < one.Lot.Z1 {
				t.Errorf("lots %v and %v meet", one.Lot, other.Lot)
			}
		}
	}
}

// TestVillageEdgesAreLanes checks that every house at the parcel's edge
// has that wall on a lane, where its shops and signs are.
func TestVillageEdgesAreLanes(t *testing.T) {
	village, err := NewVillage(VillageAsk{Width: 60, Depth: 40}, 9, nil)
	if err != nil {
		t.Fatal(err)
	}
	least := village.Houses[0].Lot.X0
	for _, house := range village.Houses {
		least = min(least, house.Lot.X0)
	}
	const slack = 1
	for _, house := range village.Houses {
		if house.Lot.X0 < least+slack && !house.Lanes[westSide] {
			t.Errorf("the house on %v at the west edge is not on a lane",
				house.Lot)
		}
	}
}

// TestVillageOnSlopeStandsEachHouse checks that on a slope each house
// stands on its own footing, with a plinth under it.
func TestVillageOnSlopeStandsEachHouse(t *testing.T) {
	const fall = 0.1
	slope := func(x, z float64) float64 { return fall * x }
	village, err := NewVillage(VillageAsk{Width: 40, Depth: 30}, 4, slope)
	if err != nil {
		t.Fatal(err)
	}
	for _, house := range village.Houses {
		if !(house.Footing.drop() > 0) {
			t.Errorf("the house on %v has no drop on a slope", house.Lot)
		}
	}
	drawn, err := village.Design().Rendered(quickLevel)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(partNames(drawn), "plinth") {
		t.Error("no house on the slope stands on a plinth")
	}
}

// TestVillageHangsCables checks that a village hangs cables over its
// lanes and its alleys.
func TestVillageHangsCables(t *testing.T) {
	village, err := NewVillage(VillageAsk{}, 11, nil)
	if err != nil {
		t.Fatal(err)
	}
	if village.Cables == 0 {
		t.Error("a village of the culture's size hangs no cable")
	}
}

// TestVillageRefusesNoParcel is an anti-test: a parcel below nought, one
// that holds no plot, and a level a building is not drawn at are each
// refused, and not drawn as nothing.
func TestVillageRefusesNoParcel(t *testing.T) {
	for _, ask := range []VillageAsk{{Width: -1}, {Width: 3, Depth: 3},
		{Level: "clumps"}} {
		if _, err := NewVillage(ask, 1, nil); err == nil {
			t.Errorf("a village asked %+v is not refused", ask)
		}
	}
	if _, err := VillageHouseAt(1, nil, "clumps"); err == nil {
		t.Error("a house at the clumps level is not refused")
	}
}

// TestCultureRefusesMeasuredKind is an anti-test: a number a culture
// says was measured is refused, for nothing in a culture is.
func TestCultureRefusesMeasuredKind(t *testing.T) {
	one := element{Name: "x", Says: "x", Unit: "m", By: "x",
		Kind: "measured", Scope: scopeDistrict, Draw: drawRange}
	if one.check() == nil {
		t.Error("an element of kind measured is not refused")
	}
	one.Kind = recalledKind
	if err := one.check(); err != nil {
		t.Errorf("an element of kind recalled is refused: %v", err)
	}
}

// TestAlleyStartsFromLashings checks that the alley's lashings are the
// lashings with their own gaps: the lantern colours are the lashings',
// and the least gap is the alley's.
func TestAlleyStartsFromLashings(t *testing.T) {
	alley := drawerOf(t, alleyCulture, 1)
	lashings := drawerOf(t, lashingsCulture, 1)
	if len(alley.options("lantern-colour")) !=
		len(lashings.options("lantern-colour")) {
		t.Error("the alley does not keep the lashings' lanterns")
	}
	if alley.fixed("span-least") >= lashings.fixed("span-least") {
		t.Error("the alley strings no narrower gap than the lanes")
	}
}
