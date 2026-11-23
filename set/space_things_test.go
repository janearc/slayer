package set

import (
	"strings"
	"testing"
)

// one is a block of one thing of a kind, standing at the origin, with the
// keys the test adds under it.
func one(name, kind, keys string) string {
	return "  setting: space\n  set:\n    things:\n      " + name +
		":\n        kind: " + kind + "\n        seed: 1\n" +
		"        place: {at: origin}\n" + keys
}

// A star, a point, a lantern, a culture orbital and a nebula are read,
// each key into a typed field of the thing.
func TestSpaceKindsAreRead(t *testing.T) {
	made, err := readV2("  setting: space\n  set:\n    things:\n" +
		"      sun:\n        kind: star\n        seed: 3\n" +
		"        place: {at: origin}\n        star: sun\n        limb: darkened\n" +
		"        corona: {reach: 4, streamers: 7, seed: 3}\n" +
		"      mark:\n        kind: point\n        seed: 1\n        around: sun\n" +
		"      lamp:\n        kind: lantern\n        seed: 1\n        around: sun\n" +
		"        panels: baseline\n        turning: true\n" +
		"      ring:\n        kind: culture-orbital\n        seed: 2\n        around: lamp\n" +
		"        radius: {value: 1840000, unit: km}\n" +
		"        width: {value: 3, unit: radii}\n        walls: {value: 1000, unit: m}\n" +
		"        lean: {value: -45, unit: deg}\n        sense: -1\n" +
		"        lights: {kelvin: {from: 1900, to: 3200}}\n" +
		"      gas:\n        kind: nebula\n        seed: 4\n        around: lamp\n" +
		"        thickness: 4.5e-5\n        neon: 0\n        argon: 0.5\n        glow: 0.2\n")
	if err != nil {
		t.Fatal(err)
	}
	list := made.Things()
	sun, lamp, ring, gas := list[0].space, list[2].space, list[3].space, list[4].space
	if sun.star != "sun" || sun.limb != "darkened" || sun.corona == nil ||
		*sun.corona != (corona{reach: 4, streamers: 7, seed: 3}) {
		t.Errorf("the sun is read as %+v", sun)
	}
	if lamp.panels != "baseline" || lamp.turning == nil || !*lamp.turning {
		t.Errorf("the lantern is read as %+v", lamp)
	}
	if ring.radius == nil || *ring.radius != (length{1840000, "km"}) ||
		*ring.width != (length{3, "radii"}) || *ring.walls != (length{1000, "m"}) ||
		ring.lean == nil || *ring.lean != -45 || ring.sense != -1 ||
		ring.kelvin == nil || *ring.kelvin != (kelvinRange{1900, 3200}) {
		t.Errorf("the orbital is read as %+v", ring)
	}
	if gas.thickness == nil || *gas.thickness != 4.5e-5 || *gas.neon != 0 ||
		*gas.argon != 0.5 || *gas.glow != 0.2 {
		t.Errorf("the nebula is read as %+v", gas)
	}
	if list[1].kind != "point" {
		t.Errorf("the point is a %q", list[1].kind)
	}
}

// A key is the key of a kind: each of these keys is refused on a kind that
// is not its own, and taken on its own.
func TestSpaceKeysBelongToTheirKinds(t *testing.T) {
	for _, each := range []struct{ key, own, other string }{
		{"star: sun", "star", "lantern"},
		{"limb: uniform", "star", "point"},
		{"corona: {reach: 2, streamers: 1, seed: 1}", "star", "nebula"},
		{"panels: baseline", "lantern", "star"},
		{"turning: false", "lantern", "culture-orbital"},
		{"radius: {value: 1, unit: km}", "culture-orbital", "lantern"},
		{"width: {value: 1, unit: km}", "culture-orbital", "nebula"},
		{"walls: {value: 1, unit: km}", "culture-orbital", "star"},
		{"lean: {value: 5, unit: deg}", "culture-orbital", "point"},
		{"sense: 1", "culture-orbital", "lantern"},
		{"lights: {kelvin: {from: 1, to: 2}}", "culture-orbital", "nebula"},
		{"thickness: 1", "nebula", "star"},
		{"neon: 0.1", "nebula", "culture-orbital"},
		{"argon: 0.1", "nebula", "lantern"},
		{"glow: 0.1", "nebula", "point"},
	} {
		if _, err := readV2(one("a", each.own, "        "+each.key+"\n")); err != nil {
			t.Errorf("%s on a %s is refused: %v", each.key, each.own, err)
		}
		_, err := readV2(one("a", each.other, "        "+each.key+"\n"))
		if err == nil || !strings.Contains(err.Error(), each.other) {
			t.Errorf("%s on a %s: read says %v, want a refusal naming the kind",
				each.key, each.other, err)
		}
	}
}

// Each key of the new kinds refuses a value outside its range or its
// words.
func TestSpaceKeysRefuseWhatIsNotTheirs(t *testing.T) {
	for _, each := range []struct{ kind, key string }{
		{"star", "star: moon"},
		{"star", "limb: soft"},
		{"star", "corona: {reach: 0, streamers: 1, seed: 1}"},
		{"star", "corona: {reach: 2, streamers: -1, seed: 1}"},
		{"star", "corona: {reach: 2, streamers: 1.5, seed: 1}"},
		{"star", "corona: {reach: 2, streamers: 1, seed: -1}"},
		{"star", "corona: {reach: 2, streamers: 1}"},
		{"lantern", "panels: dense"},
		{"lantern", "turning: sometimes"},
		{"culture-orbital", "radius: {value: 1, unit: miles}"},
		{"culture-orbital", "radius: {value: 1}"},
		{"culture-orbital", "radius: 7"},
		{"culture-orbital", "lean: {value: 91, unit: deg}"},
		{"culture-orbital", "lean: {value: 5, unit: rad}"},
		{"culture-orbital", "sense: 0"},
		{"culture-orbital", "sense: 2"},
		{"culture-orbital", "lights: {kelvin: {from: 0, to: 3200}}"},
		{"culture-orbital", "lights: {kelvin: {from: 1900}}"},
		{"nebula", "thickness: 0"},
		{"nebula", "neon: 1.5"},
		{"nebula", "argon: -0.5"},
		{"nebula", "glow: -1"},
	} {
		if _, err := readV2(one("a", each.kind, "        "+each.key+"\n")); err == nil {
			t.Errorf("%s on a %s was taken", each.key, each.kind)
		}
	}
	if _, err := readV2(one("a", "star", "")); err != nil {
		t.Errorf("a star with no more keys is refused: %v", err)
	}
}

// A length in radii is the radii of a star, so a set with no star in it
// refuses one, and a set with one takes it.
func TestALengthInRadiiNeedsAStar(t *testing.T) {
	ring := "      ring:\n        kind: culture-orbital\n        seed: 2\n" +
		"        place: {at: origin}\n        radius: {value: 3, unit: radii}\n"
	star := "      sun:\n        kind: star\n        seed: 3\n        place: {at: origin}\n"
	if _, err := readV2("  setting: space\n  set:\n    things:\n" + ring); err == nil {
		t.Error("a length in radii was taken in a set with no star")
	}
	if _, err := readV2("  setting: space\n  set:\n    things:\n" + star + ring); err != nil {
		t.Errorf("a length in radii is refused beside a star: %v", err)
	}
}

// A kind of version 2 is refused in a file that names version 1.
func TestKindsOfVersionTwoAreRefusedOutsideIt(t *testing.T) {
	source := v2Text(defaultFilm, one("a", "star", ""))
	text := strings.Replace(source,
		ecosystemV2Schema, EcosystemSchema, 1)
	if _, err := Read("t.yaml", []byte(text)); err == nil {
		t.Error("a version 1 file with a star was taken")
	}
	legacySource := v2Text(defaultFilm, one("a", "star", ""))
	legacy := strings.Replace(legacySource,
		`"$schema": `+ecosystemV2Schema+"\n", "", 1)
	if _, err := Read("t.yaml", []byte(legacy)); err == nil {
		t.Error("a legacy file with a star was taken")
	}
}
