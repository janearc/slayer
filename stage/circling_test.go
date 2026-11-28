package stage

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/generate"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/thing"
)

// capricaOfTheDrop is a locked planet like the Earth going round a tenth
// of a solar mass at a distance given, 930,000 km, for the orbit's tests;
// its sun's light is of the drop's order and is not worked from the drop
// (see TestAPlanetGivenItsLightStandsWhereTheDropGivesIt for that).
func capricaOfTheDrop(locked bool) generate.Planet {
	return generate.Planet{Name: "caprica",
		Mass:   measure.Kilograms(generate.EarthMass),
		Makeup: generate.Makeup{Metal: 0.32, Rock: 0.6798, Ice: 0.0002},
		Locked: locked,
		Sun: generate.Sun{Mass: measure.SolarMasses(0.1),
			Luminosity: 3.84e-5},
		Orbit: generate.Orbit{Distance: measure.Metres(930000e3)},
		Age:   4.5e9, Seed: 1}
}

// earthLight is the sunlight at the Earth, in watts a square metre: the
// solar constant, 1361 (Kopp and Lean 2011, from memory, to shelve).
const earthLight = 1361.0

// A planet given the light it gets and no distance stands where the
// drop's own light, at the drop's own radius from the cold relation and
// its 3000 K, comes to the Earth's, within a few per cent, and the circle
// it goes round on is that far across. The anti-test: a planet given a
// distance as well as a light is refused, so the two are never said twice.
func TestAPlanetGivenItsLightStandsWhereTheDropGivesIt(t *testing.T) {
	drop := generate.Drop{Name: "the lava drop",
		Mass: measure.SolarMasses(0.1), Temperature: measure.Kelvin(3000),
		Age: 1e9, Distance: measure.Metres(48 * generate.AstronomicalUnit)}
	shine, err := drop.Shine()
	if err != nil {
		t.Fatal(err)
	}
	planet := capricaOfTheDrop(true)
	planet.Sun = generate.Sun{Mass: shine.Mass,
		Luminosity: shine.Luminosity.InSolarLuminosities()}
	planet.Orbit = generate.Orbit{Flux: 1}
	stood, err := stoodAt(planet)
	if err != nil {
		t.Fatal(err)
	}
	apart := stood.Orbit.Distance.InMetres()
	flux := shine.Luminosity.InWatts() / (4 * math.Pi * apart * apart)
	if got := flux / earthLight; got < 0.97 || got > 1.03 {
		t.Errorf("%v times the Earth's sunlight at %v m, want 1", got, apart)
	}
	body, err := generate.Make(planet)
	if err != nil {
		t.Fatal(err)
	}
	orbit := goingRound(body, stood, maths.V(0, 0, 0), 0, 1)
	if math.Abs(orbit.Radius-apart) > 1 {
		t.Errorf("a circle of %v m, want %v", orbit.Radius, apart)
	}
	planet.Orbit.Distance = measure.Metres(930000e3)
	if _, err := stoodAt(planet); err == nil {
		t.Errorf("a planet given a distance and a light was placed")
	}
}

// A body that goes round the drop keeps one face to it: the direction of
// the drop, in the body's own space, is the same at every time of its
// year. The anti-test: the same body let to spin by its own day has a
// direction that moves, and the check can tell the two apart.
func TestACircledBodyKeepsOneFaceToTheDrop(t *testing.T) {
	planet := capricaOfTheDrop(true)
	planet.Day = 0
	body, err := generate.Make(planet)
	if err != nil {
		t.Fatal(err)
	}
	drop := maths.V(-4e9, 7, 1.5e9)
	orbit := goingRound(body, planet, drop, 40, 1)
	first := facing(orbit, drop, 0)
	for _, hours := range []float64{1, 3.7, 6.8, 11} {
		elapsed := time.Duration(hours * float64(time.Hour))
		if later := facing(orbit, drop, elapsed); later.Sub(first).Len() >
			1e-9 {
			t.Errorf("after %v hours the drop is toward %v, not %v", hours,
				later, first)
		}
	}
	unlocked := orbit
	unlocked.Locked = false
	if moved := facing(unlocked, drop, 3*time.Hour); moved.Sub(first).Len() <
		0.1 {
		t.Errorf("an unlocked body keeps its face to the drop too: %v",
			moved)
	}
}

// facing is the way to the drop in the body's own space at a time.
func facing(orbit thing.Orbit, drop maths.Vec, at time.Duration) maths.Vec {
	space := orbit.At(at)
	return space.Orientation.In(drop.Sub(space.Origin)).Unit()
}

// The orbit's year is the planet's own, 13.6 hours, and its circle is the
// distance given, about the drop.
func TestTheCircleIsTheOrbitAtItsDistance(t *testing.T) {
	planet := capricaOfTheDrop(true)
	body, _ := generate.Make(planet)
	drop := maths.V(1, 2, 3)
	orbit := goingRound(body, planet, drop, 0, 1)
	if got := orbit.Period.Hours(); got < 13.5 || got > 13.7 {
		t.Errorf("a circuit of %v hours, want 13.6", got)
	}
	if got := orbit.At(2 * time.Hour).Origin.Sub(drop).Len(); math.Abs(got-
		930000e3) > 1 {
		t.Errorf("%v m from the drop, want 930000000", got)
	}
}

// A locked body's clouds are laid about its sun, in the world the stage
// builds. The anti-test: the clouds of a body not locked are not.
func TestACircledBodysCloudsAreLockedAndOthersAreNot(t *testing.T) {
	planet := capricaOfTheDrop(true)
	made, err := generate.Make(planet)
	if err != nil {
		t.Fatal(err)
	}
	for _, locked := range []bool{true, false} {
		body := made
		if locked {
			body = made.Locking(maths.V(0, 0, -1))
		}
		world := thing.NewWorld()
		still := thing.Still(maths.Subspace{})
		if err := body.PlaceOn(world, still, 1e-3, 3); err != nil {
			t.Fatal(err)
		}
		found := false
		for _, volume := range world.Volumes {
			layer, is := volume.Medium.(medium.Clouds)
			if !is {
				continue
			}
			found = true
			if layer.Locked != locked ||
				locked && layer.Toward != maths.V(0, 0, -1) {
				t.Errorf("locked %v: clouds say %v toward %v", locked,
					layer.Locked, layer.Toward)
			}
		}
		if !found {
			t.Errorf("locked %v: a planet like the Earth has no clouds",
				locked)
		}
	}
}
