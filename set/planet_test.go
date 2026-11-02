package set_test

import (
	"strings"
	"testing"

	"github.com/janearc/slayer/set"
)

// giant is a set in space with a planet: two columns and three zones.
const giant = `scene:
  name: a giant
  light: sun
  sun: {azimuth: 135, altitude: 20}
  film: {width: 48, height: 32, rays: 1, exposure: 1}
  cameras:
    afar:
      orientation: {point_to: the giant, alpha: 0}
      location:
        loc_relative: {object: the giant, standoff_x: 0,
          standoff_y: -1.0e9, standoff_z: 0}
  planet:
    name: the giant
    made_up: true
    radius: 6.0e7
    gravity: 24.8
    deck: 0.8
    from_star: 1.2
    columns:
      clear:
        - {top: 0, bottom: 150, gases: {neon: 0.5, methane: 0.5}}
      hazed:
        - {top: 0, bottom: 0.5, gases: {xenon: 1},
           haze: {depth: 3, albedo: 0.97, lean: 0.6}}
        - {top: 0.5, bottom: 150, gases: {krypton: 0.25, methane: 0.75}}
    zones:
      - {south: -90, north: -10, column: clear}
      - {south: -10, north: 10, column: hazed}
      - {south: 10, north: 90, column: clear}
`

// a planet is read as written: its size, its distance from its star,
// its zones and each column's layers, gases and haze.
func TestAPlanetIsReadAsWritten(t *testing.T) {
	made, err := set.Read("giant.yaml", []byte(giant))
	if err != nil {
		t.Fatal(err)
	}
	world, has := made.Planet()
	radius, gravity, deck := world.Size()
	if !has || world.Name() != "the giant" || radius != 6e7 ||
		gravity != 24.8 || deck != 0.8 || world.FromStar() != 1.2 {
		t.Errorf("planet %+v", world)
	}
	zones := world.Zones()
	south, north, column := zones[1].Latitudes()
	if len(zones) != 3 || south != -10 || north != 10 || column != "hazed" {
		t.Errorf("zones %+v", zones)
	}
	layers := world.Column("hazed")
	top, bottom := layers[1].Pressures()
	depth, albedo, lean := layers[0].Haze()
	gases := layers[1].Gases()
	if len(layers) != 2 || top != 0.5 || bottom != 150 || depth != 3 ||
		albedo != 0.97 || lean != 0.6 || gases["methane"] != 0.75 ||
		gases["krypton"] != 0.25 {
		t.Errorf("hazed column %+v", layers)
	}
}

// a planet that is not said to be made up, or is said not to be, is
// refused, and so is each mistake in its air or its zones, where it is
// written and saying why.
func TestAPlanetsMistakesAreRefused(t *testing.T) {
	for _, mistake := range []struct{ was, now, said string }{
		{"    made_up: true\n", "", "made_up: true"},
		{"made_up: true", "made_up: false", "made_up: true"},
		{"neon: 0.5, methane", "sodium: 0.5, methane", "d lines"},
		{"neon: 0.5, methane", "radon: 0.5, methane", "decay"},
		{"neon: 0.5, methane", "argon: 0.5, methane", "not a gas"},
		{"methane: 0.75", "methane: 0.7", "sum to"},
		{"north: -10, column: clear", "north: -12, column: clear",
			"break at"},
		{"north: 90, column: clear", "north: 80, column: clear",
			"stop at"},
		{"north: 10, column: hazed", "north: 10, column: cloudy",
			"cloudy"},
		{"{top: 0.5, bottom: 150", "{top: 0.4, bottom: 150",
			"starts above"},
		{"{top: 0, bottom: 150, gases: {neon", "{top: 0, bottom: 0, " +
			"gases: {neon", "below its top"},
		{"    from_star: 1.2\n", "", "from_star"},
	} {
		written := strings.Replace(giant, mistake.was, mistake.now, 1)
		_, err := set.Read("giant.yaml", []byte(written))
		if err == nil || !strings.Contains(err.Error(), mistake.said) {
			t.Errorf("%q: refused with %v, wanted %q", mistake.now, err,
				mistake.said)
		}
	}
}

// a camera finds the planet by its name.
func TestACameraFindsThePlanet(t *testing.T) {
	written := strings.Replace(giant, "point_to: the giant",
		"point_to: the moon", 1)
	if _, err := set.Read("giant.yaml", []byte(written)); err == nil {
		t.Error("a camera pointed at nothing was taken")
	}
}

// each number of a planet outside what it can be, each part it must
// say left out, and each list written as something else, is refused.
func TestAPlanetsNumbersAndShapesAreHeld(t *testing.T) {
	for _, mistake := range []struct{ was, now, said string }{
		{"radius: 6.0e7", "radius: 0.5", "radius takes"},
		{"radius: 6.0e7", "radius: 2.0e9", "radius takes"},
		{"gravity: 24.8", "gravity: 0.001", "gravity takes"},
		{"gravity: 24.8", "gravity: 2.0e4", "gravity takes"},
		{"deck: 0.8", "deck: 1.1", "deck takes"},
		{"deck: 0.8", "deck: -0.1", "deck takes"},
		{"from_star: 1.2", "from_star: 0.0001", "from_star takes"},
		{"from_star: 1.2", "from_star: 2.0e5", "from_star takes"},
		{"    radius: 6.0e7\n", "", "radius"},
		{"    gravity: 24.8\n", "", "gravity"},
		{"    name: the giant\n", "", "name"},
		{"{top: 0, bottom: 150, gases: {neon", "{top: -1, bottom: 150, " +
			"gases: {neon", "top takes"},
		{"{top: 0.5, bottom: 150", "{top: 0.5, bottom: 2.0e6",
			"bottom takes"},
		{"albedo: 0.97", "albedo: 1.5", "albedo takes"},
		{"lean: 0.6", "lean: 1", "lean takes"},
		{"depth: 3,", "depth: -1,", "depth takes"},
		{"neon: 0.5, methane: 0.5", "neon: 1.5, methane: 0.5",
			"takes a number"},
		{"neon: 0.5, methane: 0.5", "neon: lots, methane: 0.5",
			"takes a number"},
		{"{south: -90, north: -10, column: clear}",
			"{south: -91, north: -10, column: clear}", "south takes"},
		{"{south: 10, north: 90, column: clear}",
			"{south: 10, north: 91, column: clear}", "north takes"},
		{"{south: 10, north: 90, column: clear}",
			"{south: 10, north: 90, column: clear, wind: 5}", "wind"},
		{"{south: -10, north: 10, column: hazed}",
			"{south: -10, north: -10, column: hazed}", "break at"},
		{"{top: 0, bottom: 150, gases: {neon: 0.5, methane: 0.5}}",
			"{top: 0, bottom: 150, gases: {neon: 0.5, methane: 0.5}, " +
				"rain: lead}", "rain"},
	} {
		written := strings.Replace(giant, mistake.was, mistake.now, 1)
		if written == giant {
			t.Fatalf("%q is not in the giant", mistake.was)
		}
		_, err := set.Read("giant.yaml", []byte(written))
		if err == nil || !strings.Contains(err.Error(), mistake.said) {
			t.Errorf("%q: refused with %v, wanted %q", mistake.now, err,
				mistake.said)
		}
	}
}

// a column written as no layers, or as keys, and zones written as keys,
// are refused.
func TestAPlanetsListsAreLists(t *testing.T) {
	clear := "      clear:\n        - {top: 0, bottom: 150, gases: {neon: " +
		"0.5, methane: 0.5}}\n"
	for _, mistake := range []struct{ now, said string }{
		{"      clear: []\n", "a column is a list"},
		{"      clear: {top: 0}\n", "a column is a list"},
	} {
		written := strings.Replace(giant, clear, mistake.now, 1)
		if _, err := set.Read("giant.yaml", []byte(written)); err == nil ||
			!strings.Contains(err.Error(), mistake.said) {
			t.Errorf("%q: refused with %v", mistake.now, err)
		}
	}
	zones := giant[strings.Index(giant, "    zones:"):]
	written := strings.Replace(giant, zones, "    zones: {a: b}\n", 1)
	if _, err := set.Read("giant.yaml", []byte(written)); err == nil ||
		!strings.Contains(err.Error(), "zones is a list") {
		t.Errorf("zones as keys: %v", err)
	}
}

// a set with no planet says so.
func TestASetWithNoPlanetHasNone(t *testing.T) {
	if _, has := kilaueaIkiRead(t).Planet(); has {
		t.Error("kilauea iki has a planet")
	}
}

// small numbers a planet may have are taken: a radius of a metre and a
// half, gravity of half a metre a second squared, half an astronomical
// unit from its star, and a haze half an optical depth thick.
func TestAPlanetsSmallNumbersAreTaken(t *testing.T) {
	for _, fine := range []struct{ was, now string }{
		{"radius: 6.0e7", "radius: 1.5"},
		{"gravity: 24.8", "gravity: 0.5"},
		{"from_star: 1.2", "from_star: 0.5"},
		{"depth: 3,", "depth: 0.5,"},
	} {
		written := strings.Replace(giant, fine.was, fine.now, 1)
		if _, err := set.Read("giant.yaml", []byte(written)); err != nil {
			t.Errorf("%q refused: %v", fine.now, err)
		}
	}
}

// numbers just past a planet's edges are refused, and so are a planet
// with no zones and a zone of no width at the pole.
func TestAPlanetsEdgesAreHeld(t *testing.T) {
	for _, mistake := range []struct{ was, now, said string }{
		{"depth: 3,", "depth: 1.5e6,", "depth takes"},
		{"{top: 0.5, bottom: 150", "{top: 1.5e6, bottom: 150", "top takes"},
		{"lean: 0.6", "lean: -1.5", "lean takes"},
		{"{south: -90, north: -10, column: clear}",
			"{south: -90, north: -90.5, column: clear}", "north takes"},
		{"{south: 10, north: 90, column: clear}",
			"{south: 90.5, north: 90, column: clear}", "south takes"},
		{"{south: 10, north: 90, column: clear}",
			"{south: 10, north: 90, column: clear}\n" +
				"      - {south: 90, north: 90, column: clear}", "break at"},
	} {
		written := strings.Replace(giant, mistake.was, mistake.now, 1)
		_, err := set.Read("giant.yaml", []byte(written))
		if err == nil || !strings.Contains(err.Error(), mistake.said) {
			t.Errorf("%q: refused with %v, wanted %q", mistake.now, err,
				mistake.said)
		}
	}
	zones := giant[strings.Index(giant, "    zones:"):]
	written := strings.Replace(giant, zones, "", 1)
	if _, err := set.Read("giant.yaml", []byte(written)); err == nil ||
		!strings.Contains(err.Error(), "and zones") {
		t.Errorf("a planet with no zones: %v", err)
	}
}

// ringed is the giant with a pole and two rings.
func ringed() string {
	return strings.Replace(giant, "    zones:\n", "    pole: {azimuth: 180, "+
		"altitude: 40}\n    rings:\n      - {inner: 7.8e7, outer: 8.7e7, "+
		"depth: 0.1, albedo: 0.6, lean: -0.3,\n         made_of: {ice: 0.6, "+
		"zns-ag: 0.2, csi-tl: 0.2}}\n      - {inner: 8.82e7, outer: 1.08e8, "+
		"depth: 0.4, albedo: 0.7, lean: -0.3,\n         made_of: {ice: 1}}\n"+
		"    zones:\n", 1)
}

// a planet's pole and rings are read as written, and a pole unsaid is
// straight up.
func TestAPlanetsPoleAndRingsAreRead(t *testing.T) {
	made, err := set.Read("giant.yaml", []byte(ringed()))
	if err != nil {
		t.Fatal(err)
	}
	world, _ := made.Planet()
	azimuth, altitude := world.Pole()
	rings := world.Rings()
	inner, outer := rings[1].Reach()
	depth, albedo, lean := rings[0].Optics()
	stuff := rings[0].MadeOf()
	if azimuth != 180 || altitude != 40 || len(rings) != 2 ||
		inner != 8.82e7 || outer != 1.08e8 || depth != 0.1 ||
		albedo != 0.6 || lean != -0.3 || stuff["zns-ag"] != 0.2 ||
		stuff["csi-tl"] != 0.2 || stuff["ice"] != 0.6 {
		t.Errorf("pole %g %g, rings %+v", azimuth, altitude, rings)
	}
	plain, _ := set.Read("giant.yaml", []byte(giant))
	world, _ = plain.Planet()
	if azimuth, altitude := world.Pole(); azimuth != 0 || altitude != 90 {
		t.Errorf("an unsaid pole at %g, %g", azimuth, altitude)
	}
}

// a ring's mistakes are refused where they are written: rings out of
// order, a ring inside out, a make-up not summing to one, something no
// ring is made of, numbers past their edges, and rings not a list.
func TestARingsMistakesAreRefused(t *testing.T) {
	for _, mistake := range []struct{ was, now, said string }{
		{"inner: 8.82e7", "inner: 8.6e7", "inside the one before"},
		{"outer: 8.7e7", "outer: 7.0e7", "outer edge is inside"},
		{"made_of: {ice: 1}", "made_of: {ice: 0.9}", "sums to"},
		{"made_of: {ice: 1}", "made_of: {lead: 1}", "not something"},
		{"made_of: {ice: 1}", "made_of: {ice: 1.5}", "takes a number"},
		{"depth: 0.1,", "depth: -0.1,", "depth takes"},
		{"depth: 0.1,", "depth: 2000,", "depth takes"},
		{"albedo: 0.6,", "albedo: 1.6,", "albedo takes"},
		{"lean: -0.3,\n         made_of: {ice: 0.6", "lean: -1,\n" +
			"         made_of: {ice: 0.6", "lean takes"},
		{"inner: 7.8e7", "inner: 0.5", "inner takes"},
		{"outer: 1.08e8", "outer: 2.0e12", "outer takes"},
		{"altitude: 40}", "altitude: 95}", "altitude takes"},
		{"{azimuth: 180,", "{azimuth: 400,", "azimuth takes"},
		{"    rings:\n      - {inner: 7.8e7", "    rings: {a: b}\n" +
			"    other:\n      - {inner: 7.8e7", "rings is a list"},
		{"depth: 0.4, albedo", "depth: 0.4, colour: red, albedo",
			"colour"},
	} {
		written := strings.Replace(ringed(), mistake.was, mistake.now, 1)
		if written == ringed() {
			t.Fatalf("%q is not in the ringed giant", mistake.was)
		}
		_, err := set.Read("giant.yaml", []byte(written))
		if err == nil || !strings.Contains(err.Error(), mistake.said) {
			t.Errorf("%q: refused with %v, wanted %q", mistake.now, err,
				mistake.said)
		}
	}
}

// numbers at a ring's edges are taken: touching rings, a clear ring, a
// black one, a white one, a make-up of one stuff.
func TestARingsEdgesAreTaken(t *testing.T) {
	for _, fine := range []struct{ was, now string }{
		{"inner: 8.82e7", "inner: 8.7e7"},
		{"depth: 0.1,", "depth: 0,"},
		{"albedo: 0.6,", "albedo: 0,"},
		{"albedo: 0.7,", "albedo: 1,"},
		{"inner: 7.8e7", "inner: 1"},
	} {
		written := strings.Replace(ringed(), fine.was, fine.now, 1)
		if _, err := set.Read("giant.yaml", []byte(written)); err != nil {
			t.Errorf("%q refused: %v", fine.now, err)
		}
	}
}

// a ring's and a pole's numbers just past their edges are refused, and
// just inside them taken.
func TestARingsAndPolesEdges(t *testing.T) {
	for _, mistake := range []struct{ was, now, said string }{
		{"inner: 7.8e7", "inner: 1.5e12", "inner takes"},
		{"lean: -0.3,\n         made_of: {ice: 0.6", "lean: 1.5,\n" +
			"         made_of: {ice: 0.6", "lean takes"},
		{"altitude: 40}", "altitude: 90.5}", "altitude takes"},
		{"altitude: 40}", "altitude: -90.5}", "altitude takes"},
		{"{azimuth: 180,", "{azimuth: 360.5,", "azimuth takes"},
		{"outer: 8.7e7", "outer: 7.8e7", "outer edge is inside"},
	} {
		written := strings.Replace(ringed(), mistake.was, mistake.now, 1)
		_, err := set.Read("giant.yaml", []byte(written))
		if err == nil || !strings.Contains(err.Error(), mistake.said) {
			t.Errorf("%q: refused with %v, wanted %q", mistake.now, err,
				mistake.said)
		}
	}
	for _, fine := range []struct{ was, now string }{
		{"{azimuth: 180,", "{azimuth: 0.5,"},
		{"inner: 7.8e7, outer: 8.7e7", "inner: 1, outer: 1.5"},
	} {
		written := strings.Replace(ringed(), fine.was, fine.now, 1)
		if _, err := set.Read("giant.yaml", []byte(written)); err != nil {
			t.Errorf("%q refused: %v", fine.now, err)
		}
	}
}
