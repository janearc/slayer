package set_test

import (
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/set"
)

// likeEarthSaid is the generate block that stands for the built world's
// saturn template: a planet like the Earth, in a set's units.
const likeEarthSaid = `generate:
          mass: 1
          makeup: {metal: 0.32, rock: 0.6798, ice: 0.0002, gas: 0}
          day: 23.93
          tilt: 23.44
          distance: 1
          sun: {mass: 1, luminosity: 1}
          age: 4.5e9
          seed: 3`

// generatedBuilt is the built world with its body generated: the
// template's line changed to what replace says.
func generatedBuilt(t *testing.T, replace string) string {
	t.Helper()
	source := built(t)
	changed := strings.Replace(source, "template: saturn", replace, 1)
	if changed == source {
		t.Fatal("the built world no longer says template: saturn")
	}
	return changed
}

// a generated body's numbers come back in the units a planet is made
// in: the Earth's mass in kilograms, a day in hours, a distance in
// astronomical units.
func TestAGeneratedBodyIsItsNumbers(t *testing.T) {
	made, err := set.Read("generated.yaml",
		[]byte(generatedBuilt(t, likeEarthSaid)))
	if err != nil {
		t.Fatal(err)
	}
	world, _ := made.World()
	planet, generated := world.Bodies()[0].Generated()
	if !generated || planet.Name != "saturn" || planet.Seed != 3 ||
		math.Abs(planet.Mass.InKilograms()-5.9722e24) > 1e18 ||
		math.Abs(planet.Day.Hours()-23.93) > 1e-9 ||
		planet.Sun.Luminosity != 1 ||
		math.Abs(planet.Orbit.Distance.InMetres()-1.495978707e11) > 1 {
		t.Errorf("planet %+v, generated %v", planet, generated)
	}
	if _, templated := builtRead(t).Bodies()[0].Generated(); templated {
		t.Error("a body from a template says it is generated")
	}
}

// the anti-test: a body from a template and generated both is refused,
// as is a generate block that leaves out a number.
func TestABodyMadeTwoWaysIsRefused(t *testing.T) {
	both := generatedBuilt(t, "template: saturn\n        "+likeEarthSaid)
	if _, err := set.Read("both.yaml", []byte(both)); err == nil ||
		!strings.Contains(err.Error(), "not both") {
		t.Errorf("a body both ways was read: %v", err)
	}
	short := generatedBuilt(t, strings.Replace(likeEarthSaid,
		"          seed: 3", "", 1))
	if _, err := set.Read("short.yaml", []byte(short)); err == nil {
		t.Error("a generated planet with no seed was read")
	}
}
