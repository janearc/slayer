package biome_test

import (
	"testing"

	"github.com/janearc/slayer/biome"
	"github.com/janearc/slayer/maths/hexes"
)

// made is the test world under a sea, and the fields it was made from.
func made(test testing.TB, sea float64) (*biome.World, biome.Fields) {
	test.Helper()
	from := fields(test, sea)
	return biome.Make(from), from
}

// risen is whether a tile the fields paint as land the world paints as
// water: a river's tile, or a lake's.
func risen(world *biome.World, from biome.Fields, tile hexes.Tile) bool {
	return !water(from.Paint(tile)) && water(world.Paint(tile))
}

// a river runs the steepest way down: every tile of land the world paints
// as water and that has a lower neighbour flows into its lowest, which is
// water too, a river's, a lake's or the sea's.
func TestRiversRunDownhillToTheSeaOrALake(t *testing.T) {
	world, from := made(t, nought)
	if world.Rivers == nought {
		t.Fatal("the test world has no river")
	}
	flowing := nought
	for _, tile := range tiles(world) {
		next, lower := lowest(from, tile)
		if !risen(world, from, tile) || !lower {
			continue
		}
		flowing++
		if !water(world.Paint(next)) {
			t.Errorf("a river at %v runs into %v, painted %v", tile, next,
				world.Paint(next))
		}
	}
	if flowing < world.Rivers {
		t.Errorf("%d rivers run through %d tiles", world.Rivers, flowing)
	}
}

// anti-tests: a world all under the sea has no river, lake, volcano or
// city on it, and is the deep sea everywhere.
func TestADrownedWorldHasNothingOnIt(t *testing.T) {
	world, _ := made(t, picked(t, "drowned"))
	if world.Rivers+world.Lakes+world.Volcanoes+world.Cities != nought {
		t.Errorf("a drowned world has %d rivers, %d lakes, %d volcanoes "+
			"and %d cities", world.Rivers, world.Lakes, world.Volcanoes,
			world.Cities)
	}
	for _, tile := range tiles(world) {
		if world.Paint(tile) != terran(deepSea) {
			t.Errorf("drowned, %v is painted %v", tile, world.Paint(tile))
		}
	}
}

// a lake is where a river can go no lower: every tile of land the world
// paints as water and that has no lower neighbour, one for each lake.
func TestALakeIsWhereARiverCanGoNoLower(t *testing.T) {
	world, from := made(t, nought)
	basins := nought
	for _, tile := range tiles(world) {
		if _, lower := lowest(from, tile); risen(world, from, tile) && !lower {
			basins++
		}
	}
	if world.Lakes == nought || basins != world.Lakes {
		t.Errorf("the world says %d lakes and holds %d", world.Lakes, basins)
	}
}

// anti-tests: a river that can go lower is no lake: the test world's
// rivers run through tiles with lower ground beside them, and those
// tiles are not counted among its lakes.
func TestNotEveryRiverIsALake(t *testing.T) {
	world, from := made(t, nought)
	for _, tile := range tiles(world) {
		if _, lower := lowest(from, tile); risen(world, from, tile) && lower {
			return
		}
	}
	t.Errorf("every one of %d rivers is a lake", world.Rivers)
}

// a volcano is a peak, higher than everything round it and at least as
// high as a volcano needs, with lava at the top and ash or a river round
// it, and the world counts each.
func TestVolcanoesStandOnPeaks(t *testing.T) {
	world, from := made(t, nought)
	least, peaks := picked(t, "volcano-height"), nought
	for _, tile := range tiles(world) {
		if world.Paint(tile) != terran(lava) {
			continue
		}
		peaks++
		if above(from, tile) < least {
			t.Errorf("a volcano at %v stands only %v high", tile,
				above(from, tile))
		}
		for _, next := range from.Tiling.Neighbours(tile) {
			if above(from, next) >= above(from, tile) {
				t.Errorf("a volcano at %v has %v as high round it", tile, next)
			}
			if paint := world.Paint(next); paint != terran(ash) && !water(paint) {
				t.Errorf("round a volcano at %v, %v is painted %v", tile,
					next, paint)
			}
		}
	}
	if world.Volcanoes == nought || peaks != world.Volcanoes {
		t.Errorf("the world says %d volcanoes and holds %d",
			world.Volcanoes, peaks)
	}
}

// anti-tests: a world whose land is lower than a volcano needs has none.
func TestALowWorldHasNoVolcano(t *testing.T) {
	world, _ := made(t, picked(t, "lowland"))
	if world.Volcanoes != nought {
		t.Errorf("a low world has %d volcanoes", world.Volcanoes)
	}
	for _, tile := range tiles(world) {
		if world.Paint(tile) == terran(lava) {
			t.Errorf("a low world has lava at %v", tile)
		}
	}
}

// a city stands on dry land beside water, the sea's or a river's, never
// beside a volcano, and the world counts each.
func TestCitiesStandBesideWater(t *testing.T) {
	world, from := made(t, nought)
	stone, cities := glow(t, "stone"), nought
	for _, tile := range tiles(world) {
		if world.Paint(tile) != stone {
			continue
		}
		cities++
		if paint := from.Paint(tile); water(paint) || paint == terran(rock) ||
			paint == terran(snow) {
			t.Errorf("a city at %v stands where the fields paint %v", tile,
				paint)
		}
		beside := false
		for _, next := range from.Tiling.Neighbours(tile) {
			beside = beside || water(world.Paint(next))
			if world.Paint(next) == terran(lava) {
				t.Errorf("a city at %v stands beside a volcano", tile)
			}
		}
		if !beside {
			t.Errorf("a city at %v stands beside no water", tile)
		}
	}
	if world.Cities == nought || cities != world.Cities {
		t.Errorf("the world says %d cities and holds %d", world.Cities,
			cities)
	}
}

// anti-tests: a world that is snow everywhere has no city on it.
func TestASnowWorldHasNoCity(t *testing.T) {
	world, _ := made(t, picked(t, "snowed"))
	if world.Cities != nought {
		t.Errorf("a world of snow has %d cities", world.Cities)
	}
	for _, tile := range tiles(world) {
		if world.Paint(tile) == glow(t, "stone") {
			t.Errorf("a world of snow has a city at %v", tile)
		}
	}
}

// where there is no lava, river, lake, city or ash, the world paints
// what its fields do.
func TestTheWorldPaintsAsItsFieldsElsewhere(t *testing.T) {
	world, from := made(t, nought)
	stone := glow(t, "stone")
	for _, tile := range tiles(world) {
		paint := world.Paint(tile)
		if paint == terran(lava) || paint == terran(ash) || paint == stone ||
			risen(world, from, tile) {
			continue
		}
		if paint != from.Paint(tile) {
			t.Errorf("%v is painted %v, where its fields paint %v", tile,
				paint, from.Paint(tile))
		}
	}
}

// anti-tests: a world is not its fields everywhere: the test world has
// tiles the fields do not paint as it does.
func TestAWorldIsMoreThanItsFields(t *testing.T) {
	world, from := made(t, nought)
	for _, tile := range tiles(world) {
		if world.Paint(tile) != from.Paint(tile) {
			return
		}
	}
	t.Error("the test world paints every tile as its fields do")
}

// a world made twice is the same world, tile for tile, in its paint and
// its light, and in its counts.
func TestAWorldIsTheSameEveryTime(t *testing.T) {
	from := fields(t, nought)
	from.Sprawl = picked(t, "sprawl")
	one, other := biome.Make(from), biome.Make(from)
	if one.Rivers != other.Rivers || one.Lakes != other.Lakes ||
		one.Volcanoes != other.Volcanoes || one.Cities != other.Cities {
		t.Error("a world made twice counts two ways")
	}
	for _, tile := range tiles(one) {
		if one.Paint(tile) != other.Paint(tile) ||
			one.Shine().Paint(tile) != other.Shine().Paint(tile) {
			t.Errorf("a world made twice is two worlds at %v", tile)
		}
	}
}

// anti-tests: a world of another seed is another world.
func TestAnotherSeedIsAnotherWorld(t *testing.T) {
	from := fields(t, nought)
	other := from
	other.Seed = uint64(picked(t, "other-seed"))
	one, two := biome.Make(from), biome.Make(other)
	for _, tile := range tiles(one) {
		if one.Paint(tile) != two.Paint(tile) {
			return
		}
	}
	t.Error("two seeds make one world")
}
