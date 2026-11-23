package cities

import "testing"

// addedParts are the parts of what people add to a facade, by the names
// they are drawn with.
var addedParts = []string{"planter", "foliage", "bloom", "unit", "balcony",
	"handrail", "upright", "awning", "arm", "dish", "pipe", "cable", "pole",
	"line", "washing", "sign"}

func TestAtTheDemoLevelPeopleHaveAddedThingsToTheFacades(t *testing.T) {
	found := map[string]int{}
	for _, culture := range []string{"portofino", "groningen", "midwest"} {
		for seed := range Seed(detailSeeds) {
			for _, one := range terraceAt(t, culture, seed, "demo") {
				found[one.Mesh.Name]++
				if !closed(one.Mesh) {
					t.Fatalf("%s seed %d: %s is not closed", culture, seed,
						one.Mesh.Name)
				}
			}
		}
	}
	for _, part := range addedParts {
		if found[part] == 0 {
			t.Errorf("no %s over three cultures and %d seeds each", part,
				detailSeeds)
		}
	}
}

func TestBelowTheDemoLevelNothingHasBeenAdded(t *testing.T) {
	for _, level := range []Detail{"hull", "quick"} {
		drawn := terraceAt(t, "portofino", 3, level)
		for _, part := range addedParts {
			if got := len(named(drawn, part)); got != 0 {
				t.Errorf("%d %s at %s", got, part, level)
			}
		}
	}
}

func TestAWallLampIsALightBesideADoorAndABalconyRailIsIron(t *testing.T) {
	lamps, rails := 0, 0
	for seed := range Seed(detailSeeds) {
		for _, one := range terraceAt(t, "portofino", seed, "demo") {
			if one.Lamp != nil && one.Material.Name == "cities wall lamp" {
				lamps++
				if !(one.Lamp.Luminance > 0 && one.Lamp.Heat == wallLampHeat) {
					t.Fatalf("a wall lamp %+v", one.Lamp)
				}
			}
			if one.Mesh.Name == "handrail" {
				rails++
				if one.Material.Name != "cities iron iron" {
					t.Fatalf("a rail of %q", one.Material.Name)
				}
			}
		}
	}
	if lamps == 0 || rails == 0 {
		t.Errorf("%d wall lamps and %d rails", lamps, rails)
	}
}
