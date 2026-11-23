package cities

import "testing"

// tierShape is a building shape the tier tests draw: a front, a depth and
// the storeys of a tall building.
func tierShape(storeys int) massShape {
	return massShape{width: 28, depth: 11.5, storeys: storeys, from: 6}
}

func TestTheTiersOfABuildingAreItsStoreysAndEachIsShallower(t *testing.T) {
	stepped := 0
	for seed := range Seed(seeds) {
		drawing := homeDrawer(t, seed)
		for bay := range 4 {
			at := place{Bay: bay}
			shape := tierShape(8 + bay)
			tiers := drawing.tiers(at, shape)
			total := 0
			for index, each := range tiers {
				total += each.storeys
				if index > 0 && !(each.depth < tiers[index-1].depth) {
					t.Errorf("seed %d bay %d: tier %d is not shallower than "+
						"the one under it", seed, bay, index)
				}
				if each.depth < drawing.fixed("setback-least-depth")-
					tolerance {
					t.Errorf("seed %d bay %d: a tier %v deep", seed, bay,
						each.depth)
				}
			}
			if total != shape.storeys {
				t.Errorf("seed %d bay %d: tiers hold %d of %d storeys", seed,
					bay, total, shape.storeys)
			}
			if len(tiers) > 1 {
				stepped++
			}
		}
	}
	if stepped == 0 {
		t.Error("no building steps back")
	}
}

func TestABuildingThatDoesNotStepBackIsOneTierOfItsShape(t *testing.T) {
	drawing := drawerOf(t, condominiumCulture, 1)
	shape := tierShape(5)
	tiers := drawing.tiers(place{}, shape)
	if len(tiers) != 1 || tiers[0].storeys != 5 || tiers[0].depth != 11.5 {
		t.Errorf("at the quick level the tiers are %v", tiers)
	}
}

func TestEverythingOnADeckStandsInsideTheDeck(t *testing.T) {
	area := roofArea{x0: 0.2, z0: 8.5, x1: 27.8, z1: 12, y: 15}
	carried := 0
	for seed := range Seed(seeds) {
		drawing := homeDrawer(t, seed)
		for level := 1; level <= 3; level++ {
			for _, part := range drawing.deck(place{Bay: level % 4}, level,
				area, 0) {
				low, high := reachOf(t, part)
				carried++
				if !insideArea(low, high, area) {
					t.Errorf("seed %d level %d: a deck part at %v to %v is "+
						"off the deck", seed, level, low, high)
				}
			}
		}
	}
	if carried == 0 {
		t.Error("no deck carries anything")
	}
}

func TestADeckHasARailingPlantingAndSeating(t *testing.T) {
	area := roofArea{x0: 0.2, z0: 8.5, x1: 27.8, z1: 12, y: 15}
	seen := map[string]int{}
	for _, part := range homeDrawer(t, 1).deck(place{}, 1, area, 0) {
		seen[part.Element.(solid).Name]++
	}
	for _, name := range []string{"decking", "deck rail", "deck post",
		"deck planter", "deck table", "deck chair"} {
		if seen[name] == 0 {
			t.Errorf("a deck with no %s", name)
		}
	}
}

func TestSomeDecksHaveAPergolaAndSomeHaveNone(t *testing.T) {
	area := roofArea{x0: 0.2, z0: 8.5, x1: 27.8, z1: 12, y: 15}
	with, without := 0, 0
	for seed := range Seed(seeds) {
		pergola := false
		for _, part := range homeDrawer(t, seed).deck(place{}, 1, area, 0) {
			pergola = pergola || part.Element.(solid).Name == "pergola beam"
		}
		if pergola {
			with++
		} else {
			without++
		}
	}
	if with == 0 || without == 0 {
		t.Errorf("%d decks with a pergola and %d without, of %d", with,
			without, seeds)
	}
}

func TestDecksStandOnlyOnTheRoofsASetbackLeavesBare(t *testing.T) {
	district := Seed(drawerOf(t, condominiumCulture, 1).district)
	for index := range Seed(homeSeeds) {
		seed := index * district
		drawing := homeDrawer(t, seed)
		shared := court{width: drawing.number("court-width", place{}),
			depth: drawing.number("court-depth", place{})}
		want := 0
		for bay := range drawing.count("buildings", place{}) {
			length := shared.width
			if bay >= 2 {
				length = shared.depth
			}
			shape := shared.shapeOf(drawing, place{Bay: bay}, length)
			want += len(drawing.tiers(place{Bay: bay}, shape)) - 1
		}
		design, err := CondominiumAt(seed, nil, demoLevel)
		if err != nil {
			t.Fatal(err)
		}
		drawn, err := design.Rendered(demoLevel)
		if err != nil {
			t.Fatal(err)
		}
		if got := len(named(drawn, "decking")); got != want {
			t.Errorf("seed %d: %d decks for %d setbacks", seed, got, want)
		}
	}
}

func TestSomeCondominiumsHaveATallBuildingThatStepsBack(t *testing.T) {
	district := Seed(drawerOf(t, condominiumCulture, 1).district)
	tall := 0
	for index := range Seed(seeds) {
		seed := index * district
		drawing := homeDrawer(t, seed)
		shared := court{width: 30, depth: 26}
		for bay := range drawing.count("buildings", place{}) {
			shape := shared.shapeOf(drawing, place{Bay: bay}, shared.width)
			if tiers := drawing.tiers(place{Bay: bay}, shape); len(tiers) > 2 {
				tall++
				t.Logf("seed %d bay %d: %d storeys in %v", seed, bay,
					shape.storeys, tiers)
			}
		}
	}
	if tall == 0 {
		t.Error("no building of three tiers or more in the seeds")
	}
}

func TestAStairCoreReachesBackToTheHighestTier(t *testing.T) {
	drawing := homeDrawer(t, 1)
	shared := court{width: 30, depth: 26, mounts: new([]Mount)}
	shape := tierShape(10)
	shape.reach = 8
	parts := shared.stair(drawing, place{}, shape, wall(
		option{Name: "cream", Colour: []float64{1, 1, 1}},
		option{Name: renderCladding, Colour: []float64{1, 1, 1}}))
	front, back := 0.0, 1e9
	for _, part := range parts {
		if solid := part.Element.(solid); solid.Name != "wall" {
			continue
		}
		low, high := reachOf(t, part)
		front, back = max(front, high.Z), min(back, low.Z)
	}
	if front-back < shape.reach {
		t.Errorf("a core %v deep for a setback of %v", front-back, shape.reach)
	}
}
