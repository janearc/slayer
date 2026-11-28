package set

import (
	"math"
	"os"
	"strings"
	"testing"

	"github.com/janearc/slayer/scene/generate"
)

// capricaSet is the lava drop recipes' set, as a test holds it: a copy in
// testdata, with a line the test may spoil.
func capricaSet(t *testing.T, spoil func(string) string) Set {
	t.Helper()
	source, err := os.ReadFile("testdata/caprica-lava-drop.yaml")
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	if spoil != nil {
		text = spoil(text)
	}
	made, err := Read("caprica-lava-drop.yaml", []byte(text))
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// standing is how far, in metres, the lava drop's place in a set is from
// the distance the planet's flux puts it, and how far the camera named
// "drop" is from the drop's centre, in metres: the planet stands at the
// origin, so the drop's distance from the origin is the planet's.
func standing(t *testing.T, made Set) (off, camera float64) {
	t.Helper()
	world, _ := made.World()
	placed, middle, err := world.Binaries()[0].Placed(world.Unit())
	if err != nil {
		t.Fatal(err)
	}
	drop := middle.Add(placed.Drop.At.Scale(1 / world.Unit()))
	planet, _ := world.Bodies()[0].Generated()
	planet.Sun = generate.Sun{Mass: placed.Drop.Shine.Mass,
		Luminosity: placed.Drop.Shine.Luminosity.InSolarLuminosities()}
	where, err := planet.Distance()
	if err != nil {
		t.Fatal(err)
	}
	shot, err := made.Shot("drop", nil)
	if err != nil {
		t.Fatal(err)
	}
	return math.Abs(drop.Len() - where.InMetres()),
		shot.Eye().Origin.Sub(drop).Len()
}

// The lava drop set puts the drop where Caprica's flux says it is, to a
// kilometre, and the camera "drop" 70,000 km off it, to a half per cent:
// both are worked by hand from the distance the generator gives, and this
// holds them to it. The anti-test: the same set with the drop's place
// moved five per cent is far from it, so a change to the generator's
// distance that was not followed by the set is caught.
func TestTheLavaDropSetStandsWhereItsFluxPutsCaprica(t *testing.T) {
	off, camera := standing(t, capricaSet(t, nil))
	if off > 1e3 {
		t.Errorf("the drop stands %v m from where the flux puts it", off)
	}
	if math.Abs(camera-70000e3) > 0.005*70000e3 {
		t.Errorf("the camera \"drop\" is %v m off the drop, want 70000 km",
			camera)
	}
	off, _ = standing(t, capricaSet(t, func(source string) string {
		return strings.Replace(source, "drop_at: [-374139656",
			"drop_at: [-393000000", 1)
	}))
	if off < 1e6 {
		t.Errorf("a drop moved five per cent is only %v m off", off)
	}
}

// dropSet is a set that says a pair of stars with a lava drop, and a
// planet that goes round it, with a line the test may spoil.
func dropSet(spoil func(string) string) []byte {
	source := `"$schema": https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: a drop
  film: {width: 64, height: 36, rays: 1, exposure: 1}
  cameras:
    view:
      title: view
      orientation: {look_at: [0, 0, 0]}
      location: {at: [0, -5, 2]}
  world:
    unit: 1
    binaries:
      pair:
        generate:
          primary: {mass: 2.45, age: 1.2e7}
          companion: {mass: 0.62, temperature: 25000, radius: 0.0122}
          separation: 9.2
          eccentricity: 0.16
          drop: {mass: 0.1, temperature: 3000, age: 1.0e9, distance: 48, made_up: true}
        drop_at: [-9.3e8, 0, 0]
        time: 4
    bodies:
      caprica:
        generate:
          mass: 1
          makeup: {metal: 0.32, rock: 0.6798, ice: 0.0002, gas: 0}
          tilt: 0
          distance: 0.0062167
          age: 4.5e9
          seed: 1
          locked: true
        circles: pair
        phase: 0
        detail: 5
`
	if spoil != nil {
		source = spoil(source)
	}
	return []byte(source)
}

// A set says a lava drop and the planet that goes round it, and reads
// their numbers; the pair is worked back so the drop stands where the set
// says, to a metre, whatever the time, and the planet is locked.
func TestADropAndItsPlanetReadTheirNumbers(t *testing.T) {
	made, err := Read("drop.yaml", dropSet(nil))
	if err != nil {
		t.Fatal(err)
	}
	world, _ := made.World()
	pair := world.Binaries()[0]
	drop := pair.Generated().Drop
	if drop == nil || drop.Mass.InSolarMasses() < 0.0999 ||
		drop.Temperature.InKelvin() != 3000 {
		t.Fatalf("the pair's drop is %+v", drop)
	}
	placed, middle, err := pair.Placed(world.Unit())
	if err != nil {
		t.Fatal(err)
	}
	at := middle.Add(placed.Drop.At.Scale(1 / world.Unit()))
	if off := at.Sub(fromMap([3]float64{-9.3e8, 0, 0})).Len(); off > 1 {
		t.Errorf("the drop stands %v m from where the set puts it", off)
	}
	body := world.Bodies()[0]
	planet, _ := body.Generated()
	if body.Circles() != "pair" || !planet.Locked {
		t.Errorf("the body circles %q, locked %v", body.Circles(), planet.Locked)
	}
	if math.Abs(planet.Orbit.Distance.InMetres()-930000e3) > 1e6 {
		t.Errorf("a distance of %v m", planet.Orbit.Distance.InMetres())
	}
}

// A planet says the light it gets, in the Earth's, in place of a distance,
// and the set reads it as a flux and no distance. The anti-tests: both
// said, or neither, is refused, and so is a drop that says a Coulomb
// share, which the generator does not take.
func TestAPlanetSaysItsFluxInPlaceOfItsDistance(t *testing.T) {
	made, err := Read("drop.yaml", dropSet(func(source string) string {
		return strings.Replace(source, "distance: 0.0062167", "flux: 1", 1)
	}))
	if err != nil {
		t.Fatal(err)
	}
	world, _ := made.World()
	planet, _ := world.Bodies()[0].Generated()
	if planet.Orbit.Flux != 1 || planet.Orbit.Distance.InMetres() != 0 {
		t.Errorf("the orbit is %+v, want a flux of 1 and no distance",
			planet.Orbit)
	}
	for name, spoil := range map[string]func(string) string{
		"both": func(source string) string {
			return strings.Replace(source, "distance: 0.0062167",
				"distance: 0.0062167\n          flux: 1", 1)
		},
		"neither": func(source string) string {
			return strings.Replace(source, "          distance: 0.0062167\n",
				"", 1)
		},
		"a Coulomb share": func(source string) string {
			return strings.Replace(source, "made_up: true",
				"coulomb: 0.15, made_up: true", 1)
		},
	} {
		if _, err := Read("drop.yaml", dropSet(spoil)); err == nil {
			t.Errorf("a planet with %s was read", name)
		}
	}
}

// The anti-test: a drop whose numbers are made up and does not say
// made_up: true is refused, with the line the generator would say; a
// drop young enough that nothing is made up need not say it.
func TestAMadeUpDropMustSayMadeUp(t *testing.T) {
	_, err := Read("drop.yaml", dropSet(func(source string) string {
		return strings.Replace(source, ", made_up: true", "", 1)
	}))
	if err == nil || !strings.Contains(err.Error(), "made up:") ||
		!strings.Contains(err.Error(), "made_up: true") {
		t.Errorf("a made up drop with no made_up was read: %v", err)
	}
	_, err = Read("drop.yaml", dropSet(func(source string) string {
		source = strings.Replace(source, ", made_up: true", "", 1)
		return strings.Replace(source, "age: 1.0e9", "age: 1.0e6", 1)
	}))
	if err != nil {
		t.Errorf("a young drop was refused: %v", err)
	}
}

// The anti-test: a pair says its middle or where its drop is, one of the
// two; says where its drop is only if it has one; and a body that goes
// round a drop is locked and has no sun or middle of its own.
func TestADropsPlaceAndItsPlanetsAreSaidOneWay(t *testing.T) {
	for name, spoil := range map[string]func(string) string{
		"both middle and drop_at": func(source string) string {
			return strings.Replace(source, "        time: 4\n",
				"        middle: [0, 0, 0]\n        time: 4\n", 1)
		},
		"drop_at with no drop": func(source string) string {
			return strings.Replace(source, `          drop: {mass: 0.1, temperature: 3000, age: 1.0e9, distance: 48, made_up: true}
`, "", 1)
		},
		"circling and not locked": func(source string) string {
			return strings.Replace(source, "          locked: true\n", "", 1)
		},
		"circling with a sun": func(source string) string {
			return strings.Replace(source, "          locked: true\n",
				"          locked: true\n          sun: {mass: 1, luminosity: 1}\n", 1)
		},
		"circling with a middle": func(source string) string {
			return strings.Replace(source, "        circles: pair\n",
				"        circles: pair\n        middle: [0, 0, 0]\n", 1)
		},
	} {
		if _, err := Read("drop.yaml", dropSet(spoil)); err == nil {
			t.Errorf("%s was read", name)
		}
	}
}
