package set

import (
	"strings"
	"testing"
)

// satellite is a thing of a spacecraft's kind with more keys after its
// kind and seed.
func satellite(name, more string) string {
	return "      " + name + ":\n        kind: hubble\n        seed: 7\n" + more
}

// base is a thing at the origin for others to go around or look at.
const base = "      base:\n        kind: point\n        seed: 1\n        place: {at: origin}\n"

// A scene's anchor is one place that everything is measured from, and
// is read into a typed value.
func TestSceneAnchorIsRead(t *testing.T) {
	made, err := readV2("  anchor:\n    lat: 19.4165\n    lon: -155.2485\n" +
		"    elevation: {value: lidar, unit: m}\n    datum: wgs84\n")
	if err != nil {
		t.Fatal(err)
	}
	got := made.anchor
	if got == nil || got.lat != 19.4165 || got.lon != -155.2485 ||
		!got.lidar || got.datum != "wgs84" {
		t.Errorf("the anchor is %+v", got)
	}
	made, err = readV2("  anchor:\n    lat: 1\n    lon: 2\n" +
		"    elevation: {value: 1200.5, unit: m}\n    datum: wgs84\n")
	if err != nil || made.anchor.lidar || made.anchor.elevation != 1200.5 {
		t.Errorf("a metres elevation is read as %+v, %v", made.anchor, err)
	}
}

// A scene's globe is the world, with the elevation near the anchor and
// beyond it, and how its detail is spent.
func TestSceneGlobeIsRead(t *testing.T) {
	made, err := readV2("  globe:\n    planet: earth\n    terrain:\n" +
		"      near: {from: usgs-kilauea-2019-dem, file: kil.tif}\n" +
		"      beyond: {from: global-dem}\n    detail: by-size\n")
	if err != nil {
		t.Fatal(err)
	}
	want := globeOf{planet: "earth", near: "usgs-kilauea-2019-dem",
		file: "kil.tif", beyond: "global-dem", detail: "by-size"}
	if made.globe == nil || *made.globe != want {
		t.Errorf("the globe is %+v, want %+v", made.globe, want)
	}
}

// A scene's sky is drawn from a catalogue, toward a direction, so far
// round it, down to a magnitude.
func TestSceneSkyIsRead(t *testing.T) {
	made, err := readV2("  sky:\n    stars:\n      from: gaia\n" +
		"      toward: {ra: 110.8, dec: -73.45}\n" +
		"      radius: {value: 10, unit: deg}\n      faintest: 12\n")
	if err != nil {
		t.Fatal(err)
	}
	want := starSky{from: "gaia", toward: skyPoint{110.8, -73.45}, radius: 10,
		faintest: 12}
	if made.sky == nil || *made.sky != want {
		t.Errorf("the sky is %+v, want %+v", made.sky, want)
	}
}

// A thing in orbit, pointing, with an eye of an instrument, is read.
func TestSpacecraftOrbitAttitudeAndEyeAreRead(t *testing.T) {
	made, err := readV2("  setting: space\n  frame: {at: sun-earth-l2}\n  set:\n    things:\n" + base +
		satellite("scope", "        orbit:\n          height: {value: 500, unit: km}\n"+
			"          inclination: 97.4\n          over: base\n"+
			"        attitude: {point: nadir, shield: sun}\n"+
			"        eye:\n          pupil: {value: 2.4, unit: m}\n"+
			"          focal: {value: 57.6, unit: m}\n"+
			"          models: diffraction-limited pupil\n") +
		satellite("other", "        place: {at: frame}\n"+
			"        attitude: {point: {sky: {ra: 110.8, dec: -73.45}}}\n") +
		satellite("third", "        around: base\n"+
			"        attitude: {point: {object: base}}\n"))
	if err != nil {
		t.Fatal(err)
	}
	list := made.Things()
	scope := list[1].space
	if scope.orbit == nil || scope.orbit.height != (length{500, "km"}) ||
		scope.orbit.inclination != 97.4 || scope.orbit.over != "base" {
		t.Errorf("the orbit is %+v", scope.orbit)
	}
	if scope.attitude == nil || scope.attitude.point != "nadir" ||
		scope.attitude.shield != "sun" {
		t.Errorf("the attitude is %+v", scope.attitude)
	}
	if scope.eye == nil || scope.eye.pupil != (length{2.4, "m"}) ||
		scope.eye.focal != (length{57.6, "m"}) ||
		scope.eye.models != "diffraction-limited pupil" {
		t.Errorf("the eye is %+v", scope.eye)
	}
	if got := list[2].space.attitude; got == nil || got.sky == nil ||
		*got.sky != (skyPoint{110.8, -73.45}) {
		t.Errorf("the second attitude is %+v", got)
	}
	if got := list[3].space.attitude; got == nil || got.object != "base" {
		t.Errorf("the third attitude is %+v", got)
	}
}

// A layer of the atmosphere reaches the terrain or the world.
func TestAtmosphereLayerExtentCanBeTheWorld(t *testing.T) {
	globe := "  globe:\n    planet: earth\n    terrain:\n      near: {from: a}\n" +
		"      beyond: {from: b}\n    detail: by-size\n"
	terrain := "  terrain:\n    from: usgs-kilauea-2019-dem\n" +
		"    file: unpacked/kil_2019dem/kil_2019dem.tif\n    west: 264500\n" +
		"    north: 2148100\n    across: 600\n    down: 600\n    step: 4\n"
	layer := func(extent string) string {
		return "  weather:\n    low:\n      kind: prescribed-mist\n" +
			"      extinction: {value: 0.018, unit: m^-1}\n" +
			"      depth: {value: 130, unit: m}\n      coverage: 0.7\n      seed: 23\n" +
			"  atmosphere:\n    layers:\n      deck:\n        weather: low\n" +
			"        extent: " + extent + "\n        albedo: 0.95\n"
	}
	made, err := readV2(globe + layer("world"))
	if err != nil {
		t.Fatal(err)
	}
	if got := made.atmosphereLayers; len(got) != 1 || !got[0].world {
		t.Errorf("a world layer is read as %+v", got)
	}
	made, err = readV2(terrain + layer("terrain"))
	if err != nil || made.atmosphereLayers[0].world {
		t.Errorf("a terrain layer is read as %+v, %v", made.atmosphereLayers, err)
	}
	if _, err := readV2(globe + layer("sky")); err == nil {
		t.Error("a layer reaching the sky was taken")
	}
	made, err = readV2(layer("world"))
	if err != nil || len(made.FlatAir()) != 1 {
		t.Errorf("a world layer with no globe or terrain is %+v, %v",
			made.FlatAir(), err)
	}
}

// The anchor, globe, sky, orbit, attitude and eye refuse what is not
// theirs; each is a scene that is good but for those lines.
func TestSpaceWorldKeysRefuseWhatIsNotTheirs(t *testing.T) {
	anchor := func(rest string) string {
		return "  anchor:\n    lat: 1\n    lon: 2\n    elevation: {value: 3, unit: m}\n" + rest
	}
	globe := func(rest string) string {
		return "  globe:\n    planet: earth\n    terrain:\n" + rest + "    detail: by-size\n"
	}
	ship := func(more string) string {
		return "  set:\n    things:\n" + base + satellite("scope", more)
	}
	for why, add := range map[string]string{
		"an anchor with no datum":               anchor(""),
		"an anchor at a latitude past the pole": "  anchor:\n    lat: 91\n    lon: 2\n    elevation: {value: 3, unit: m}\n    datum: wgs84\n",
		"an anchor of another datum":            anchor("    datum: nad83\n"),
		"an elevation in kilometres":            "  anchor:\n    lat: 1\n    lon: 2\n    elevation: {value: 3, unit: km}\n    datum: wgs84\n",
		"a globe of another planet":             "  globe:\n    planet: mars\n    terrain:\n      near: {from: a}\n      beyond: {from: b}\n    detail: by-size\n",
		"a globe with no beyond":                globe("      near: {from: a}\n"),
		"a globe with no source":                globe("      near: {file: a.tif}\n      beyond: {from: b}\n"),
		"a sky from another catalogue":          "  sky:\n    stars:\n      from: hipparcos\n      toward: {ra: 1, dec: 1}\n      radius: {value: 1, unit: deg}\n      faintest: 1\n",
		"a sky toward past the pole":            "  sky:\n    stars:\n      from: gaia\n      toward: {ra: 1, dec: 100}\n      radius: {value: 1, unit: deg}\n      faintest: 1\n",
		"a sky with no magnitude":               "  sky:\n    stars:\n      from: gaia\n      toward: {ra: 1, dec: 1}\n      radius: {value: 1, unit: deg}\n",
		"an orbit over nothing":                 ship("        orbit:\n          height: {value: 1, unit: km}\n          inclination: 1\n          over: nothing\n"),
		"an orbit at a bad angle":               ship("        orbit:\n          height: {value: 1, unit: km}\n          inclination: 181\n          over: base\n"),
		"an orbit over itself":                  ship("        orbit:\n          height: {value: 1, unit: km}\n          inclination: 1\n          over: scope\n"),
		"an attitude with no point":             ship("        around: base\n        attitude: {shield: sun}\n"),
		"an attitude pointing nowhere":          ship("        around: base\n        attitude: {point: zenith}\n"),
		"an attitude at a missing thing":        ship("        around: base\n        attitude: {point: {object: nothing}}\n"),
		"an eye of another model":               ship("        around: base\n        eye:\n          pupil: {value: 1, unit: m}\n          focal: {value: 1, unit: m}\n          models: ray tracing\n"),
		"an eye with no focal length":           ship("        around: base\n        eye:\n          pupil: {value: 1, unit: m}\n          models: diffraction-limited pupil\n"),
		"a place at the anchor of no anchor":    "  set:\n    things:\n      a:\n        kind: point\n        seed: 1\n        place: {at: anchor}\n",
		"a place at the frame of no frame":      "  set:\n    things:\n      a:\n        kind: point\n        seed: 1\n        place: {at: frame}\n",
		"a place and an orbit":                  ship("        place: {at: origin}\n        orbit:\n          height: {value: 1, unit: km}\n          inclination: 1\n          over: base\n"),
	} {
		if _, err := readV2("  setting: space\n" + add); err == nil {
			t.Errorf("%s was taken", why)
		}
	}
}

// A place at the anchor or the frame is good when the scene says one.
func TestAPlaceAtTheAnchorOrTheFrameNeedsOne(t *testing.T) {
	things := func(at string) string {
		return "  set:\n    things:\n      a:\n        kind: point\n        seed: 1\n        place: {at: " + at + "}\n"
	}
	anchor := "  anchor:\n    lat: 1\n    lon: 2\n    elevation: {value: lidar, unit: m}\n    datum: wgs84\n"
	frame := "  frame: {at: sun-earth-l2}\n"
	if _, err := readV2("  setting: space\n" + anchor + things("anchor")); err != nil {
		t.Errorf("a place at the anchor is refused beside one: %v", err)
	}
	if _, err := readV2("  setting: space\n" + frame + things("frame")); err != nil {
		t.Errorf("a place at the frame is refused beside one: %v", err)
	}
}

// A camera that sees through an instrument names a thing that has an eye.
func TestCameraInstrumentNeedsAThingWithAnEye(t *testing.T) {
	camera := "  cameras:\n    down:\n      title: d\n      eye: {instrument: scope}\n" +
		"      moves:\n        - look: {object: base}\n          duration: {value: 1, unit: s}\n"
	eye := "        eye:\n          pupil: {value: 1, unit: m}\n          focal: {value: 2, unit: m}\n          models: diffraction-limited pupil\n"
	text := func(more string) []byte {
		source := v2Text(defaultFilm,
			"  setting: space\n  set:\n    things:\n"+base+
				satellite("scope", "        around: base\n"+more))
		return []byte(strings.Replace(source, defaultCamera, camera, 1))
	}
	if _, err := Read("t.yaml", text(eye)); err != nil {
		t.Errorf("an instrument with an eye is refused: %v", err)
	}
	if _, err := Read("t.yaml", text("")); err == nil {
		t.Error("an instrument with no eye was taken")
	}
}

// What version 2 says of the world is refused in a file that names no
// schema, by name.
func TestWorldKeysOfVersionTwoAreRefusedOutsideIt(t *testing.T) {
	legacy := strings.Replace(v2Text(defaultFilm, "  anchor:\n    lat: 1\n"),
		`"$schema": `+ecosystemV2Schema+"\n", "", 1)
	if _, err := Read("t.yaml", []byte(legacy)); err == nil ||
		!strings.Contains(err.Error(), "schema version 2") {
		t.Errorf("a legacy file with an anchor: read says %v", err)
	}
	world := strings.Replace(v2Text(defaultFilm, "  terrain:\n    from: usgs-kilauea-2019-dem\n"+
		"    file: unpacked/kil_2019dem/kil_2019dem.tif\n    west: 264500\n"+
		"    north: 2148100\n    across: 600\n    down: 600\n    step: 4\n"+
		"  weather:\n    low:\n      kind: prescribed-mist\n"+
		"      extinction: {value: 0.018, unit: m^-1}\n"+
		"      depth: {value: 130, unit: m}\n      coverage: 0.7\n      seed: 23\n"+
		"  atmosphere:\n    layers:\n      deck:\n        weather: low\n"+
		"        extent: world\n        albedo: 0.95\n"),
		`"$schema": `+ecosystemV2Schema+"\n", "", 1)
	if _, err := Read("t.yaml", []byte(world)); err == nil ||
		!strings.Contains(err.Error(), "extent takes") {
		t.Errorf("a legacy layer reaching the world: read says %v", err)
	}
}
