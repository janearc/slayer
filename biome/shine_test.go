package biome_test

import (
	"math"
	"testing"

	"github.com/janearc/slayer/biome"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/hexes"
)

// lights is the test world's three city lights, dim to bright.
func lights(test testing.TB) []maths.Light {
	test.Helper()
	return []maths.Light{glow(test, "amber"), glow(test, "white"),
		glow(test, "blue")}
}

// with no sprawl and no lights given, lava glows as lava does, a city
// as sodium does, and nothing else gives off anything.
func TestLavaAndCitiesGlowAndNothingElse(t *testing.T) {
	world, _ := made(t, nought)
	shine, stone := world.Shine(), glow(t, "stone")
	for _, tile := range tiles(world) {
		want := dark()
		switch world.Paint(tile) {
		case terran(lava):
			want = glow(t, "lava")
		case stone:
			want = glow(t, "sodium")
		}
		if got := shine.Paint(tile); got != want {
			t.Errorf("%v, painted %v, gives off %v, not %v", tile,
				world.Paint(tile), got, want)
		}
	}
}

// anti-tests: a world all under the sea gives off nothing anywhere.
func TestTheSeaGivesOffNothing(t *testing.T) {
	world, _ := made(t, picked(t, "drowned"))
	for _, tile := range tiles(world) {
		if got := world.Shine().Paint(tile); got != dark() {
			t.Errorf("drowned, %v gives off %v", tile, got)
		}
	}
}

// with lights given and no sprawl, every city glows one of them: the one
// when one is given, and when three are, not all the same one.
func TestACityGlowsOneOfItsLights(t *testing.T) {
	world, _ := made(t, nought)
	stone := glow(t, "stone")
	for _, colours := range [][]maths.Light{lights(t), {glow(t, "amber")}} {
		world.Lights = colours
		used := map[maths.Light]bool{}
		for _, tile := range tiles(world) {
			if world.Paint(tile) != stone {
				continue
			}
			got := world.Shine().Paint(tile)
			used[got] = true
			if !given(got, colours) {
				t.Errorf("a city at %v glows %v, which it was not given",
					tile, got)
			}
		}
		one, many := len(colours) == unity, len(colours) > unity
		if one && len(used) != unity || many && len(used) <= unity {
			t.Errorf("%d cities glow %d of %d lights", world.Cities,
				len(used), len(colours))
		}
	}
}

// given is whether a light is one of some lights.
func given(light maths.Light, lights []maths.Light) bool {
	for _, each := range lights {
		if each == light {
			return true
		}
	}
	return false
}

// anti-tests: with lights given, no city glows sodium.
func TestACityGivenLightsIsNotSodium(t *testing.T) {
	world, _ := made(t, nought)
	world.Lights = lights(t)
	stone, sodium := glow(t, "stone"), glow(t, "sodium")
	for _, tile := range tiles(world) {
		if world.Paint(tile) == stone && world.Shine().Paint(tile) == sodium {
			t.Errorf("a city at %v given lights glows sodium", tile)
		}
	}
}

// ring is the test ring's fields, the inside of a ring world, as the
// orbital scene paints, its cities sprawling so far.
func ring(test reporter, sprawl float64) biome.Fields {
	test.Helper()
	//ninja:vow stranger hexes makes no tiling; a Band is a plain value
	tiling := hexes.Band{Radius: picked(test, "band-radius"),
		Width: picked(test, "band-width"), Around: int(picked(test,
			"band-around"))}
	return biome.Fields{Tiling: tiling,
		Seed:  uint64(picked(test, "band-seed")),
		Scale: picked(test, "scale"), Sprawl: sprawl}
}

// sprawled is the test ring with its cities' lights spreading, in some
// colours.
func sprawled(test testing.TB, colours []maths.Light) *biome.World {
	test.Helper()
	spread := ring(test, picked(test, "sprawl"))
	world := biome.Make(spread)
	world.Lights = colours
	return world
}

// darkCities is how many of a world's cities give off nothing, and how
// many give off something.
func darkCities(test reporter, world *biome.World) (dim, lit int) {
	stone := glow(test, "stone")
	for _, tile := range tiles(world) {
		switch {
		case world.Paint(tile) != stone:
		case world.Shine().Paint(tile) == dark():
			dim++
		default:
			lit++
		}
	}
	return dim, lit
}

// a sprawl under one still sprawls: one city in thirty is a heart and
// lit, and the rest, out of a heart's small reach, are dark.
func TestASmallSprawlStillSprawls(t *testing.T) {
	small := ring(t, picked(t, "small-sprawl"))
	world := biome.Make(small)
	if dim, lit := darkCities(t, world); dim == nought || lit == nought {
		t.Errorf("a small sprawl leaves %d cities dark and %d lit", dim, lit)
	}
}

// anti-tests: with no sprawl, no city is dark: each glows on its own.
func TestNoSprawlLeavesNoCityDark(t *testing.T) {
	world := biome.Make(ring(t, nought))
	if dim, lit := darkCities(t, world); dim != nought || lit == nought {
		t.Errorf("no sprawl leaves %d cities dark and %d lit", dim, lit)
	}
}

// brightestOf is the most any channel of some lights is.
func brightestOf(colours []maths.Light) float64 {
	var most float64
	for _, colour := range colours {
		for _, channel := range []float64{colour.R, colour.G, colour.B} {
			most = math.Max(most, channel)
		}
	}
	return most
}

// a sprawl lights the land round its cities' hearts, past the cities
// themselves, never the sea, and no brighter than its brightest light
// times the most a light may be.
func TestASprawlLightsTheLandAndNeverTheSea(t *testing.T) {
	world := sprawled(t, lights(t))
	stone := glow(t, "stone")
	most := brightestOf(world.Lights) * picked(t, "brightest")
	beyond := nought
	for _, tile := range tiles(world) {
		paint, got := world.Paint(tile), world.Shine().Paint(tile)
		if paint == terran(lava) {
			continue
		}
		lit := got != dark()
		if lit && water(paint) {
			t.Errorf("%v, painted as water, gives off %v", tile, got)
		}
		if lit && paint != stone {
			beyond++
		}
		if got.R > most || got.G > most || got.B > most {
			t.Errorf("%v gives off %v, past %v", tile, got, most)
		}
	}
	if beyond == nought {
		t.Error("no city's light reaches past its own tiles")
	}
}

// anti-tests: a sprawl given no lights is sodium's colour wherever it
// lights, only dimmer or brighter.
func TestASprawlGivenNoLightsIsSodium(t *testing.T) {
	world := sprawled(t, nil)
	sodium, within := glow(t, "sodium"), picked(t, "within")
	lit := nought
	for _, tile := range tiles(world) {
		got := world.Shine().Paint(tile)
		if world.Paint(tile) == terran(lava) || got == dark() {
			continue
		}
		lit++
		share := got.R / sodium.R
		if math.Abs(got.G-share*sodium.G) > within ||
			math.Abs(got.B-share*sodium.B) > within {
			t.Errorf("%v gives off %v, not sodium's colour", tile, got)
		}
	}
	if lit == nought {
		t.Error("a sprawl given no lights lights nothing")
	}
}
