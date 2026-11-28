package generate

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/scene/bodies"
)

// The targets below are the published values for Earth and Jupiter
// (NASA's planetary fact sheets, to be shelved with the relations Derive
// cites). A planet given their numbers should come out near them; it is
// a planet like them, so each target has a tolerance, not an equality.

const au = 1.495978707e11 // metres

// sunLike is a star like the Sun.
var sunLike = Sun{Mass: measure.Kilograms(1.989e30), Luminosity: 1}

// likeEarth is Earth's numbers.
func likeEarth() Planet {
	return Planet{
		Name:   "like earth",
		Mass:   measure.Kilograms(5.972e24),
		Makeup: Makeup{Metal: 0.32, Rock: 0.6798, Ice: 0.0002},
		Day:    time.Duration(23.93 * float64(time.Hour)),
		Tilt:   measure.Degrees(23.44),
		Sun:    sunLike,
		Orbit:  Orbit{Distance: measure.Metres(au), Eccentricity: 0.0167},
		Age:    4.5e9,
		Seed:   5,
	}
}

// likeJupiter is Jupiter's numbers.
func likeJupiter() Planet {
	return Planet{
		Name:   "like jupiter",
		Mass:   measure.Kilograms(1.898e27),
		Makeup: Makeup{Metal: 0.02, Rock: 0.03, Ice: 0.05, Gas: 0.90},
		Day:    time.Duration(9.93 * float64(time.Hour)),
		Tilt:   measure.Degrees(3.13),
		Sun:    sunLike,
		Orbit:  Orbit{Distance: measure.Metres(5.2 * au), Eccentricity: 0.0489},
		Age:    4.5e9,
		Seed:   9,
	}
}

// near says whether got is within a share of want.
func near(got, want, share float64) bool {
	return math.Abs(got-want) <= share*math.Abs(want)
}

func TestLikeEarthComesOutNearEarth(t *testing.T) {
	derived, err := Derive(likeEarth())
	if err != nil {
		t.Fatal(err)
	}
	if derived.Kind != bodies.Rock {
		t.Errorf("kind %v, want rock", derived.Kind)
	}
	if radius := derived.Radius.InKilometres(); !near(radius, 6371, 0.05) {
		t.Errorf("radius %.0f km, want 6371 within 5%%", radius)
	}
	if !near(derived.Gravity, 9.81, 0.05) {
		t.Errorf("gravity %.2f, want 9.81 within 5%%", derived.Gravity)
	}
	// the equilibrium temperature, with Earth's albedo of about 0.3
	if !near(derived.Temperature, 255, 0.04) {
		t.Errorf("temperature %.0f K, want 255 within 4%%", derived.Temperature)
	}
	if !derived.Air || derived.Water != Ocean {
		t.Errorf("air %v, water %v; want air and oceans", derived.Air, derived.Water)
	}
}

func TestLikeJupiterComesOutNearJupiter(t *testing.T) {
	derived, err := Derive(likeJupiter())
	if err != nil {
		t.Fatal(err)
	}
	if derived.Kind != bodies.Giant {
		t.Errorf("kind %v, want giant", derived.Kind)
	}
	if radius := derived.Radius.InKilometres(); !near(radius, 69911, 0.10) {
		t.Errorf("radius %.0f km, want 69911 within 10%%", radius)
	}
	if !near(derived.Temperature, 110, 0.10) {
		t.Errorf("temperature %.0f K, want 110 within 10%%", derived.Temperature)
	}
}

func TestLikeEarthIsMadeWithGround(t *testing.T) {
	body, err := Make(likeEarth())
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := body.Ground(); !ok {
		t.Error("a rock with oceans was made with no ground")
	}
}

// The anti-tests: what a generator must never make.

func TestAGasWorldHasNoGround(t *testing.T) {
	body, err := Make(likeJupiter())
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := body.Ground(); ok {
		t.Error("a giant was given ground to stand on")
	}
}

func TestAWorldTooHotForOceansHasNone(t *testing.T) {
	hot := likeEarth()
	hot.Orbit.Distance = measure.Metres(0.3 * au)
	derived, err := Derive(hot)
	if err != nil {
		t.Fatal(err)
	}
	if derived.Water == Ocean {
		t.Errorf("at %.0f K a world kept its oceans", derived.Temperature)
	}
}

func TestARogueIsNotWarmedBySunlight(t *testing.T) {
	rogue := likeEarth()
	rogue.Sun = Sun{}
	rogue.Orbit = Orbit{}
	derived, err := Derive(rogue)
	if err != nil {
		t.Fatal(err)
	}
	if derived.Temperature > 60 {
		t.Errorf("a rogue with no sun came out at %.0f K", derived.Temperature)
	}
}

func TestAMakeupThatDoesNotAddUpIsRefused(t *testing.T) {
	wrong := likeEarth()
	wrong.Makeup = Makeup{Metal: 0.5, Rock: 0.6}
	if _, err := Derive(wrong); err == nil {
		t.Error("shares adding to 1.1 were accepted")
	}
}

func TestAnOrbitWhoseDistanceAndYearDisagreeIsRefused(t *testing.T) {
	wrong := likeEarth()
	wrong.Orbit.Year = 2
	if _, err := Derive(wrong); err == nil {
		t.Error("an orbit of 1 AU and two years was accepted")
	}
}
