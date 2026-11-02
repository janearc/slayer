package set_test

import (
	"os"
	"strings"
	"testing"

	"github.com/janearc/slayer/set"
)

// kilaueaIki is the kīlauea iki set file, as written.
func kilaueaIki(t *testing.T) string {
	t.Helper()
	source, err := os.ReadFile("testdata/kilauea-iki.yaml")
	if err != nil {
		t.Fatal(err)
	}
	return string(source)
}

// a set says its ground, its film, its things by name and its stands,
// each as written, and a camera finds a thing by its name.
func TestASetSaysItsWorld(t *testing.T) {
	made, err := set.Read("kilauea-iki.yaml", []byte(kilaueaIki(t)))
	if err != nil {
		t.Fatal(err)
	}
	ground, has := made.Ground()
	west, north, across, down, step := ground.Window()
	if !has || ground.From() != "usgs-kilauea-2019-dem" || west != 264500 ||
		north != 2148100 || across != 600 || down != 600 || step != 4 {
		t.Errorf("ground %+v", ground)
	}
	film, has := made.Film()
	width, height, rays := film.Size()
	if !has || width != 960 || height != 600 || rays != 2 ||
		film.Exposure() != 100000 || made.Light() != "moon" {
		t.Errorf("film %d %d %d, light %s", width, height, rays,
			made.Light())
	}
	things := made.Things()
	east, northing := things[0].At()
	if len(things) != 1 || things[0].Name() != "the first tree" ||
		things[0].Kind() != "spruce" || things[0].Seed() != 3 ||
		east != 264800 || northing != 2147900 || !things[0].Alight() ||
		!strings.Contains(things[0].StandsIn(), "ʻōhiʻa") {
		t.Errorf("things %+v", things)
	}
	stands := made.Stands()
	count, seed := stands[1].Count()
	if len(stands) != 2 || stands[1].Kind() != "spruce" || count != 30 ||
		seed != 11 || stands[1].Alight() != 4 {
		t.Errorf("stands %+v", stands)
	}
}

// a world's mistakes are refused where they are written: a kind the
// manufactory has not, a stand with more alight than it holds, a count
// that is not whole, and alight that is not true or false.
func TestAWorldsMistakesAreRefused(t *testing.T) {
	for _, mistake := range []struct{ was, now, said string }{
		{"kind: fern", "kind: ohia", "kind takes"},
		{"alight: 4", "alight: 40", "has 40 alight of 30"},
		{"count: 120", "count: 12.5", "count takes a whole number"},
		{"alight: true", "alight: yes please", "alight takes true or false"},
		{"point_to: the first tree", "point_to: the second tree",
			`wants "the second tree"`},
	} {
		written := strings.Replace(kilaueaIki(t), mistake.was, mistake.now, 1)
		_, err := set.Read("kilauea-iki.yaml", []byte(written))
		if err == nil || !strings.Contains(err.Error(), mistake.said) {
			t.Errorf("%q: refused with %v, wanted %q", mistake.now, err,
				mistake.said)
		}
	}
}

// a set's sun is where it says, below the horizon too, and a sun past
// straight down is refused.
func TestASetsSunIsWhereItSays(t *testing.T) {
	sun := "  light: moon\n  sun:\n    azimuth: 289.65\n    altitude: -1.5\n"
	written := strings.Replace(kilaueaIki(t), "  light: moon\n", sun, 1)
	made, err := set.Read("kilauea-iki.yaml", []byte(written))
	if err != nil {
		t.Fatal(err)
	}
	read, has := made.Sunlit()
	azimuth, altitude := read.Where()
	if !has || azimuth != 289.65 || altitude != -1.5 {
		t.Errorf("the sun %v at %g, %g", has, azimuth, altitude)
	}
	written = strings.Replace(written, "altitude: -1.5", "altitude: -95", 1)
	if _, err := set.Read("kilauea-iki.yaml", []byte(written)); err == nil {
		t.Error("a sun 95 degrees down was read")
	}
	if _, has := kilaueaIkiRead(t).Sunlit(); has {
		t.Error("a set that says no sun has one")
	}
}

// kilaueaIkiRead is the kīlauea iki set file, read.
func kilaueaIkiRead(t *testing.T) set.Set {
	t.Helper()
	made, err := set.Read("kilauea-iki.yaml", []byte(kilaueaIki(t)))
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// rigged is the kīlauea iki set file with a rig in its film in place of
// the height and the exposure.
func rigged(t *testing.T) string {
	t.Helper()
	return strings.Replace(kilaueaIki(t), "    height: 600\n    rays: 2\n"+
		"    exposure: 100000\n", "    rays: 2\n    frame:\n"+
		"      width: 70.41\n      height: 51.61\n    lens:\n"+
		"      focal: 672\n      stop: 8\n      focus: the first tree\n"+
		"    shutter: 1.5\n    glimpses: 4\n", 1)
}

// a set's rig is what it says: its frame, its lens and its shutter.
func TestASetsRigIsWhatItSays(t *testing.T) {
	made, err := set.Read("kilauea-iki.yaml", []byte(rigged(t)))
	if err != nil {
		t.Fatal(err)
	}
	film, _ := made.Film()
	rig, has := film.Rig()
	width, height := rig.Frame()
	focal, stop, focus := rig.Lens()
	seconds, glimpses := rig.Shutter()
	if !has || width != 70.41 || height != 51.61 || focal != 672 ||
		stop != 8 || focus != "the first tree" || seconds != 1.5 ||
		glimpses != 4 {
		t.Errorf("rig %v: %+v", has, rig)
	}
}

// a rig that leaves out any one part, a film that says a height, an
// exposure or a width of view beside a rig, and a lens focused on
// nothing on the set, are refused.
func TestARigsMistakesAreRefused(t *testing.T) {
	const all, decides = "all of them", "decides the height"
	for _, mistake := range []struct{ was, now, said string }{
		{"    glimpses: 4\n", "", all},
		{"    shutter: 1.5\n", "", all},
		{"      width: 70.41\n", "", all},
		{"      height: 51.61\n", "", all},
		{"      focal: 672\n", "", all},
		{"      stop: 8\n", "", all},
		{"      focus: the first tree\n", "", all},
		{"    rays: 2\n", "    rays: 2\n    height: 600\n", decides},
		{"    rays: 2\n", "    rays: 2\n    exposure: 9\n", decides},
		{"    rays: 2\n", "    rays: 2\n    wide: 6\n", decides},
		{"focus: the first tree", "focus: the moon", "focused on"},
	} {
		written := strings.Replace(rigged(t), mistake.was, mistake.now, 1)
		_, err := set.Read("kilauea-iki.yaml", []byte(written))
		if err == nil || !strings.Contains(err.Error(), mistake.said) {
			t.Errorf("%q: refused with %v, wanted %q", mistake.now, err,
				mistake.said)
		}
	}
}
