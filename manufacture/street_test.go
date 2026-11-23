package manufacture

import (
	"math"
	"slices"
	"testing"

	"github.com/janearc/slayer/maths"
)

// floodTolerance is how far the flood's level may be from streetFloodUp:
// it is the gutter's height (under a quarter of a metre) and the rise to
// it, lerped at a rise of one, three roundings of at most 2.8e-17 each
// at that size, so under 1e-16; picked as 1e-15, ten times that.
const floodTolerance = 1e-15

// streetOf is a street of this wetness and length, made from a seed.
func streetOf(t *testing.T, wetness, length float64, number uint64) good {
	t.Helper()
	made, err := streetRow{spec: StreetSpec{Wetness: wetness,
		Length: length}}.made(sow(number))
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// restingDish is a street of one dish, which no other dish overlaps, so
// that its water has nothing to find but its own rim.
func restingDish(wet float64) (streetGround, streetDish) {
	dish := streetDish{x: 1.5, z: 0, across: 0.8, along: 1.5, deep: 0.03}
	return streetGround{length: 36, wet: wet,
		dishes: []streetDish{dish}}, dish
}

// a street no rain has fallen on has no water standing on it and no
// film: its asphalt mirrors nothing and has no wrinkles to smear it.
func TestADryStreetHasNoWaterAndNoGloss(t *testing.T) {
	made := streetOf(t, 0, 0, 1)
	if slices.Contains(made.slots(), streetWater) {
		t.Error("a dry street has water")
	}
	road := made.dressed[streetAsphalt]
	if road.Gloss != 0 || road.Crinkle != nil {
		t.Errorf("dry asphalt: gloss %g, crinkle %v", road.Gloss, road.Crinkle)
	}
}

// the road mirrors more as it wets, never less, and never more than the
// film it carries allows.
func TestAStreetMirrorsMoreAsItGetsWet(t *testing.T) {
	last := -1.0
	for _, wetness := range []float64{0, 0.1, 0.3, 0.5, 0.7, 1} {
		gloss := streetOf(t, wetness, 0, 1).dressed[streetAsphalt].Gloss
		if gloss < last {
			t.Errorf("at %g the road mirrors %g, less than %g", wetness,
				gloss, last)
		}
		if gloss > streetAsphaltGloss {
			t.Errorf("at %g the road mirrors %g, past its film", wetness,
				gloss)
		}
		last = gloss
	}
	if last != streetAsphaltGloss {
		t.Errorf("a soaked road mirrors %g, not %g", last, streetAsphaltGloss)
	}
}

// the water in a dish is flat, faces up, stands below the lowest of the
// dish's rim and touches the road, which is at or above the water, all the
// way round.
func TestAPuddleIsFlatAndMeetsTheRoadAtItsEdge(t *testing.T) {
	ground, dish := restingDish(0.8)
	water := ground.puddle(dish)
	if len(water) != streetPuddleSides {
		t.Fatalf("%d faces of water, want %d", len(water), streetPuddleSides)
	}
	level := ground.level(dish)
	if middle := ground.height(dish.x, dish.z); middle >= level {
		t.Fatalf("the dish's bottom %g is not under its water %g", middle,
			level)
	}
	for _, one := range water {
		if one.normal().Y <= 0 {
			t.Errorf("a face of water looks down: %v", one.normal())
		}
		for _, corner := range one {
			if corner.Y != level {
				t.Errorf("water at %g, not the level %g", corner.Y, level)
			}
		}
		for _, rim := range one[1:] {
			if ground.height(rim.X, rim.Z) < level-1e-9 {
				t.Errorf("the road at %v is under the water", rim)
			}
		}
	}
	if spill, _ := camber(dish.x + dish.across); level >= spill {
		t.Errorf("the water %g is at the rim %g", level, spill)
	}
}

// a drier street has less water in its dishes, and none before it rains.
func TestAPuddleDeepensAsTheStreetGetsWetter(t *testing.T) {
	_, dish := restingDish(0)
	last := math.Inf(-1)
	for _, wet := range []float64{0.4, 0.6, 0.8, 0.95} {
		ground, _ := restingDish(wet)
		level := ground.level(dish)
		if level < last {
			t.Errorf("at %g the water is at %g, lower than %g", wet, level,
				last)
		}
		last = level
	}
	for _, wet := range []float64{0, 0.2, 0.35} {
		ground, _ := restingDish(wet)
		if len(ground.puddles()) != 0 {
			t.Errorf("at %g there is water in the dishes", wet)
		}
	}
}

// the gutters rise only past streetFloodFirst, to a sheet over the whole
// carriageway.
func TestOnlyAFloodedStreetHasASheetOfWater(t *testing.T) {
	ground, _ := restingDish(0.9)
	if len(ground.flood()) != 0 {
		t.Error("a street at the first of the flood has a sheet")
	}
	ground.wet = 1
	sheet := ground.flood()
	if len(sheet) == 0 {
		t.Fatal("a flooded street has no sheet")
	}
	for _, one := range sheet {
		for _, corner := range one {
			if math.Abs(corner.Y-streetFloodUp) > floodTolerance {
				t.Errorf("the flood stands at %g, not %g", corner.Y,
					streetFloodUp)
			}
		}
		if one.normal().Y <= 0 {
			t.Error("the flood looks down")
		}
	}
}

// a wetness or a length a street cannot keep is refused by its number, and
// a seed nobody sowed makes no street.
func TestAStreetThatCannotBeKeptIsRefused(t *testing.T) {
	for name, spec := range map[string]StreetSpec{
		"below dry":    {Wetness: -0.1},
		"past flooded": {Wetness: 1.1},
		"not a number": {Wetness: math.NaN()},
		"too short":    {Length: 2},
		"too long":     {Length: 400},
	} {
		if _, err := Street(spec); err == nil {
			t.Errorf("%s: the street was accepted", name)
		}
	}
	if _, err := (streetRow{}).made(seed{}); err == nil {
		t.Error("a street was made from a seed nobody sowed")
	}
}

// the street is a kind a set can name, and a plain one is dry.
func TestStreetIsAKindAndAPlainOneIsDry(t *testing.T) {
	row, err := KindNamed(streetKind)
	if err != nil {
		t.Fatal(err)
	}
	made, err := makeOf(row, sow(1))
	if err != nil {
		t.Fatal(err)
	}
	if made.dressed[streetAsphalt].Gloss != 0 {
		t.Error("a plain street is wet")
	}
	if !slices.Contains(KindNames(), streetKind) {
		t.Error("the street is not among the kinds")
	}
}

// every street a set can ask for is a mesh the scene holds: wound true,
// under the face limit, one thing a matter that has parts.
func TestEveryStreetIsAMeshTheSceneHolds(t *testing.T) {
	for _, spec := range []StreetSpec{{}, {Wetness: 0.8},
		{Wetness: 1, Length: streetLeastLength},
		{Wetness: 0.6, Length: streetMostLength}} {
		made := streetOf(t, spec.Wetness, spec.Length, 5)
		things, err := made.things(maths.Local())
		if err != nil {
			t.Fatalf("%+v: %v", spec, err)
		}
		if len(things) != len(made.slots()) {
			matters := len(made.slots())
			t.Errorf("%+v: %d things for %d matters", spec, len(things),
				matters)
		}
	}
}

// standing water is flat and mirrors all it can: it has no wrinkles to
// smear what it mirrors, which the film on the road has.
func TestStandingWaterIsFlatAndTheFilmIsWrinkled(t *testing.T) {
	made := streetOf(t, 0.9, 0, 1)
	water := made.dressed[streetWater]
	if water.Crinkle != nil || water.Gloss != 1 {
		t.Errorf("water: crinkle %v, gloss %g", water.Crinkle, water.Gloss)
	}
	if made.dressed[streetAsphalt].Crinkle == nil {
		t.Error("the film on a wet road is flat")
	}
}

// the same seed is the same street, and another seed another.
func TestASeedMakesTheSameStreetEachTime(t *testing.T) {
	one, again := newStreetGround(36, 0.8, sow(4)),
		newStreetGround(36, 0.8, sow(4))
	if !slices.Equal(one.dishes, again.dishes) {
		t.Error("one seed made two streets")
	}
	other := newStreetGround(36, 0.8, sow(5))
	if slices.Equal(one.dishes, other.dishes) {
		t.Error("two seeds made one street")
	}
}

// no dish, so no puddle, reaches the kerb or the street's ends: a puddle
// that did would stand on nothing.
func TestNoDishLeavesTheCarriageway(t *testing.T) {
	for number := range uint64(30) {
		for _, length := range []float64{streetLeastLength, 36, 100} {
			ground := newStreetGround(length, 1, sow(number))
			for _, dish := range ground.dishes {
				if math.Abs(dish.x)+dish.across > streetRoadHalf ||
					math.Abs(dish.z)+dish.along > length/2 {
					t.Errorf("seed %d, length %g: dish %+v leaves the road",
						number, length, dish)
				}
			}
		}
	}
}

// the road's own slope is the slope of its height: a step along it changes
// the height by about the slope says.
func TestTheRoadsSlopeIsTheSlopeOfItsHeight(t *testing.T) {
	ground := newStreetGround(36, 0.5, sow(9))
	const step = 1e-6
	for _, at := range [][2]float64{{0.5, 1}, {-2.2, 7}, {3, -12}, {1.7, 3.3}} {
		across, along := ground.slope(at[0], at[1])
		wantAcross := (ground.height(at[0]+step, at[1]) -
			ground.height(at[0]-step, at[1])) / (2 * step)
		wantAlong := (ground.height(at[0], at[1]+step) -
			ground.height(at[0], at[1]-step)) / (2 * step)
		if math.Abs(across-wantAcross) > 1e-6 ||
			math.Abs(along-wantAlong) > 1e-6 {
			t.Errorf("at %v slope %g, %g; height says %g, %g", at, across,
				along, wantAcross, wantAlong)
		}
	}
}
