package set_test

import (
	"strings"
	"testing"

	"github.com/janearc/slayer/set"
)

// withoutSchema is the set text with its schema line removed, so that only
// the domain reader judges it.
func withoutSchema(t *testing.T, raw string) string {
	t.Helper()
	plain := strings.Replace(raw, "$schema: "+set.EcosystemSchema+"\n", "", 1)
	if strings.Contains(plain, "$schema") {
		t.Fatal("the text still names a schema")
	}
	return plain
}

// legacyLaser is the laser fixture without its schema line, with the
// before text replaced by the after text for each pair in edits.
func legacyLaser(t *testing.T, edits ...string) (set.Set, error) {
	t.Helper()
	text := withoutSchema(t, string(laserFixture))
	for i := 0; i+1 < len(edits); i += 2 {
		if !strings.Contains(text, edits[i]) {
			t.Fatalf("the fixture has no %q to change", edits[i])
		}
		text = strings.Replace(text, edits[i], edits[i+1], 1)
	}
	return set.Read("legacy-laser.yaml", []byte(text))
}

// mustHold checks that an error holds the wanted text.
func mustHold(t *testing.T, name string, err error, want string) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("%s: error %v, want one holding %q", name, err, want)
	}
}

// TestSetReadsItsGroundSkyAndFilmAsWritten checks the accessors of terrain,
// moon, sun and film against values written in the text.
func TestSetReadsItsGroundSkyAndFilmAsWritten(t *testing.T) {
	made, err := legacyLaser(t,
		"    step: 4\n", "    step: 4\n    smooth: 6\n",
		"  sun:\n    azimuth: 260\n", "  moon: {azimuth: 45, altitude: 10, face: maria, disc: false}\n  sun:\n    tint: [0.9, 0.8, 0.7]\n    azimuth: 260\n")
	if err != nil {
		t.Fatal(err)
	}
	ground, has := made.Ground()
	west, north, across, down, step := ground.Window()
	if !has || ground.From() != "usgs-kilauea-2019-dem" || ground.File() != "unpacked/kil_2019dem/kil_2019dem.tif" ||
		ground.Smooth() != 6 || west != 263000 || north != 2149000 || across != 1800 || down != 1300 || step != 4 {
		t.Errorf("terrain %+v %v", ground, has)
	}
	moon, has := made.Moonlit()
	azimuth, altitude := moon.Where()
	if !has || azimuth != 45 || altitude != 10 || moon.Disc() || moon.Face() != "maria" {
		t.Errorf("moon %+v %v", moon, has)
	}
	sun, has := made.Sunlit()
	if red, green, blue := sun.Tint(); !has || red != 0.9 || green != 0.8 || blue != 0.7 {
		t.Errorf("tint %v %v %v", red, green, blue)
	}
	film, has := made.Film()
	if _, rigged := film.Rig(); !has || rigged {
		t.Errorf("film has a rig it did not say: %v %v", has, rigged)
	}
}

// TestSetWithoutGroundSkyOrFilmSaysSo checks what each accessor gives for a
// part the set leaves out, including the white tint of an untinted sun.
func TestSetWithoutGroundSkyOrFilmSaysSo(t *testing.T) {
	made, err := set.Read("shoebox.yaml", []byte(shoebox(t)))
	if err != nil {
		t.Fatal(err)
	}
	if _, has := made.Ground(); has {
		t.Error("a set with no terrain has ground")
	}
	if _, has := made.Moonlit(); has {
		t.Error("a set with no moon is moonlit")
	}
	plain, err := legacyLaser(t)
	if err != nil {
		t.Fatal(err)
	}
	sun, _ := plain.Sunlit()
	if red, green, blue := sun.Tint(); red != 1 || green != 1 || blue != 1 {
		t.Errorf("untinted sun %v %v %v", red, green, blue)
	}
}

// TestSunTintRefusesAChannelOutOfRange checks the refusal of a tint that
// would brighten a channel.
func TestSunTintRefusesAChannelOutOfRange(t *testing.T) {
	_, err := legacyLaser(t, "  sun:\n    azimuth: 260\n", "  sun:\n    tint: [0.9, 0.8, 1.5]\n    azimuth: 260\n")
	mustHold(t, "tint above one", err, "tint takes a number")
}

// TestStandsMistAndWeatherKeepWhatTheSetWrote checks the stand and weather
// accessors against the laser fixture.
func TestStandsMistAndWeatherKeepWhatTheSetWrote(t *testing.T) {
	made, err := legacyLaser(t)
	if err != nil {
		t.Fatal(err)
	}
	stand := made.Stands()[0]
	if above, below := stand.Band(); above != 1150 || below < 1e300 {
		t.Errorf("band %v to %v", above, below)
	}
	if stand.Recipe() != "redwoods" {
		t.Errorf("recipe %q", stand.Recipe())
	}
	if weather, fade := stand.Atmosphere(); weather != "grove mist" || !fade {
		t.Errorf("stand atmosphere %q %v", weather, fade)
	}
	if name, clearance := stand.Exclude(); name != "the observatory" || clearance != 40 {
		t.Errorf("exclusion %q %v", name, clearance)
	}
	mist, _ := made.Mist("grove mist")
	if mist.Coverage() != 0.4 || mist.Seed() != 23 {
		t.Errorf("mist coverage %v seed %d", mist.Coverage(), mist.Seed())
	}
	if from, speed := made.Wind(); from != 270 || speed != 2 {
		t.Errorf("wind %v %v", from, speed)
	}
	if thing := made.Things()[0]; thing.Recipe() != "ridge telescope" || thing.Complexity() != "demo" {
		t.Errorf("thing %q %q", thing.Recipe(), thing.Complexity())
	}
}

// TestVolumeLightingAndIlluminationAreRead checks the cache selection and
// the sky sample count, and each refusal of the illumination block.
func TestVolumeLightingAndIlluminationAreRead(t *testing.T) {
	made, err := legacyLaser(t, "  renderer: gpu\n",
		"  renderer: gpu-native\n  volume-lighting: cached\n  volume-illumination: {model: isotropic-single, sky-samples: 12}\n")
	if err != nil {
		t.Fatal(err)
	}
	if made.VolumeLighting() != "cached" || made.VolumeIllumination().Samples() != 12 {
		t.Errorf("lighting %q, samples %d", made.VolumeLighting(), made.VolumeIllumination().Samples())
	}
	for name, change := range map[string]struct{ block, want string }{
		"no samples":   {"{model: isotropic-single}", "needs model and sky-samples"},
		"no model":     {"{sky-samples: 12}", "needs model and sky-samples"},
		"odd samples":  {"{model: isotropic-single, sky-samples: 13}", "even"},
		"other model":  {"{model: anisotropic, sky-samples: 12}", "model"},
		"too few":      {"{model: isotropic-single, sky-samples: 1}", "sky-samples"},
		"not a number": {"{model: isotropic-single, sky-samples: many}", "sky-samples"},
	} {
		_, err := legacyLaser(t, "  renderer: gpu\n", "  volume-illumination: "+change.block+"\n")
		mustHold(t, name, err, change.want)
	}
}

// TestFlagsAndFuelAndLaserRefusalsNameTheFault checks refusals that only
// the domain reader makes, because the schema would stop the same text
// first.
func TestFlagsAndFuelAndLaserRefusalsNameTheFault(t *testing.T) {
	plain := withoutSchema(t, fireRecipeFixture(t))
	_, err := set.Read("flag.yaml", []byte(strings.Replace(plain, "  light: none\n", "  vibes: maybe\n  light: none\n", 1)))
	mustHold(t, "flag", err, "vibes takes")
	bed := withoutSchema(t, bedFixture(t))
	raw := strings.Replace(bed, "      fuel: 1", heatVelocity+"      fuel: 1", 1)
	_, err = set.Read("bed.yaml", []byte(strings.Replace(raw, "        south: {value: -0.25, unit: m/s}\n", "", 1)))
	mustHold(t, "heat transport", err, "south")
	_, err = set.Read("bed.yaml", []byte(strings.Replace(raw, "value: 0.5, unit: m/s", "value: 0.5, unit: km/s", 1)))
	mustHold(t, "heat transport unit", err, "unit")
	_, err = legacyLaser(t, "      display-strength: 8\n", "")
	mustHold(t, "laser without strength", err, "display-strength")
	_, err = legacyLaser(t, "        south: {value: 5, unit: m}\n", "")
	mustHold(t, "origin without south", err, "south")
	_, err = legacyLaser(t, "east: {value: 3, unit: m}", "east: {value: 3, unit: km}")
	mustHold(t, "origin unit", err, "unit")
	_, err = legacyLaser(t, "recipe: ridge telescope\n", "recipe: ridge telescope\n        heading: {value: 400, unit: deg}\n")
	mustHold(t, "heading", err, "takes a number from 0 to 360")
	_, err = legacyLaser(t, "  weather:\n", "  laser-pointings: {}\n  weather:\n")
	mustHold(t, "no pointings", err, "laser-pointings needs a direction")
}

// TestPlanetBlueTintsAreOptionalAndRead checks the ring and haze blue
// albedos when said, and that unsaid they are absent, not zero.
func TestPlanetBlueTintsAreOptionalAndRead(t *testing.T) {
	source := strings.Replace(ringed(), "depth: 0.1, albedo: 0.6, lean: -0.3,", "depth: 0.1, albedo: 0.6, lean: -0.3, blue: 0.25,", 1)
	source = strings.Replace(source, "haze: {depth: 3, albedo: 0.97, lean: 0.6}", "haze: {depth: 3, albedo: 0.97, lean: 0.6, blue: 0.4}", 1)
	made, err := set.Read("blue.yaml", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	world, _ := made.Planet()
	rings := world.Rings()
	if blue, has := rings[0].Blue(); !has || blue != 0.25 {
		t.Errorf("first ring blue %v %v", blue, has)
	}
	if _, has := rings[1].Blue(); has {
		t.Error("a ring that says no blue has one")
	}
	hazed := world.Column("hazed")[0]
	if blue, has := hazed.HazeBlue(); !has || blue != 0.4 {
		t.Errorf("haze blue %v %v", blue, has)
	}
	if _, has := world.Column("clear")[0].HazeBlue(); has {
		t.Error("a layer with no haze has a haze blue")
	}
}

// TestPlanetWeatherKeepsItsFilmPace checks the pace and turning a film
// gives the planet's weather.
func TestPlanetWeatherKeepsItsFilmPace(t *testing.T) {
	extra := "    weather: {cells: 32, day: 1000, drag: 100, relax: 200, storms: {rate: .001, wide: 2, strength: .1}, seed: 7, moment: 300, every: 60, pace: 5, turning: -15}\n"
	made, err := set.Read("pace.yaml", []byte(strings.Replace(giant, "    from_star:", extra+"    from_star:", 1)))
	if err != nil {
		t.Fatal(err)
	}
	world, _ := made.Planet()
	weather, has := world.Weather()
	if pace, turning := weather.Film(); !has || pace != 5 || turning != -15 {
		t.Errorf("pace %v, turning %v", pace, turning)
	}
}
