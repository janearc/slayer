package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/set"
)

// flatReadable is the mode a test's file is written with: the owner
// writes it and everyone reads it.
const flatReadable = 0o644

// flatSet is a set of the sets directory, as its text.
func flatSet(t *testing.T, name string) string {
	t.Helper()
	written, err := os.ReadFile(filepath.Join("..", "sets", name))
	if err != nil {
		t.Fatal(err)
	}
	return string(written)
}

// flatStaged stages a world-only set from its text, as a file in a
// folder of its own, and gives what stagedWorld gives.
func flatStaged(t *testing.T, text string) drawing {
	t.Helper()
	path := filepath.Join(t.TempDir(), "flat.yaml")
	if err := os.WriteFile(path, []byte(text), flatReadable); err != nil {
		t.Fatal(err)
	}
	written, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	read, err := set.Read(path, written)
	if err != nil {
		t.Fatal(err)
	}
	if err := refuseUndrawn(read, path, drawnFeatures); err != nil {
		t.Fatal(err)
	}
	return stagedWorld(read, asked{set: path})
}

// flatChanged is a text with a piece of it changed, where it is there.
func flatChanged(t *testing.T, text, was, now string) string {
	t.Helper()
	if !strings.Contains(text, was) {
		t.Fatalf("%q is not in the text", was)
	}
	return strings.Replace(text, was, now, 1)
}

// A set that says a flat layer of air is staged with the air as a painted
// volume and the glow of the city's own light in it, and the glow lights
// that air and no other.
func TestASetWithFlatAirIsStagedWithTheAirAndItsGlow(t *testing.T) {
	done := flatStaged(t, flatSet(t, "skyglow-air.yaml"))
	if done.err != nil {
		t.Fatal(done.err)
	}
	world := done.world
	if len(world.Volumes) != 1 || world.Glow == nil {
		t.Fatalf("%d volumes, glow %v", len(world.Volumes), world.Glow)
	}
	if !world.Glow.Lights(world.Volumes[0].Medium) {
		t.Error("the glow does not light the air it was solved for")
	}
	if over := world.Glow.Mean(maths.V(0, 60, 0)); over.Luminance() <= 0 {
		t.Errorf("a city of lit signs has no glow over its street: %v", over)
	}
	if len(done.given.cached) != 1 ||
		!strings.Contains(done.given.cached[0].String(), "skyglow:") {
		t.Errorf("the progress line is %v", done.given.cached)
	}
}

// Anti-test: a set that says no flat air is staged as it was: no volume,
// no glow, and nothing said of one.
func TestASetWithNoFlatAirIsStagedAsBefore(t *testing.T) {
	done := flatStaged(t, flatSet(t, "skyglow-clear.yaml"))
	if done.err != nil {
		t.Fatal(done.err)
	}
	if len(done.world.Volumes) != 0 || done.world.Glow != nil ||
		len(done.given.cached) != 0 {
		t.Errorf("%d volumes, glow %v, notes %v", len(done.world.Volumes),
			done.world.Glow, done.given.cached)
	}
}

// flatAirless is a flat set with a layer of air and nothing that shines
// but a dark tower: the air is there, and it has nothing to glow with.
const flatAirless = `"$schema": https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: air with no light
  light: ambient
  volume-illumination: {model: isotropic-single, sky-samples: 2}
  weather:
    haze:
      kind: prescribed-mist
      extinction: {value: 0.0039, unit: m^-1}
      depth: {value: 300, unit: m}
      coverage: 1
      seed: 1
  atmosphere:
    layers:
      air:
        weather: haze
        extent: world
        albedo: 0.9
  film: {rays: 1, wide: 60, exposure: 3}
  cameras:
    street:
      title: a street
      location: {at: [0, -100, 10]}
      orientation: {look_at: [0, 100, 10]}
  world:
    materials:
      concrete:
        colour: {srgb_linear: [0.06, 0.06, 0.065]}
        source: eye
    boxes:
      tower:
        material: concrete
        middle: [0, 0, 20]
        size: [10, 10, 40]
        source: eye
`

// Zero emitters give a dark sky: the air is staged and has no glow, since
// nothing shines to glow it.
func TestAirWithNothingShiningHasNoGlow(t *testing.T) {
	done := flatStaged(t, flatAirless)
	if done.err != nil {
		t.Fatal(done.err)
	}
	if len(done.world.Volumes) != 1 || done.world.Glow != nil {
		t.Errorf("%d volumes, glow %v", len(done.world.Volumes),
			done.world.Glow)
	}
}

// Anti-test: flat air that cannot be solved honestly is refused, by what
// is wrong, and not drawn some other way.
func TestFlatAirThatCannotBeStagedIsRefused(t *testing.T) {
	air := flatSet(t, "skyglow-air.yaml")
	cases := map[string]struct{ text, want string }{
		"no volume-illumination": {flatChanged(t, air,
			"  volume-illumination: {model: isotropic-single, "+
				"sky-samples: 2}\n", ""), "volume-illumination"},
		"patchy haze": {flatChanged(t, air, "coverage: 1\n",
			"coverage: 0.5\n"), "even layer"},
		"no albedo": {flatChanged(t, air, "albedo: 0.9\n", "albedo: 0\n"),
			"even layer"},
		"a world unit of two": {flatChanged(t, air, "  world:\n",
			"  world:\n    unit: 2\n"), "metres"},
		"two layers": {flatChanged(t, air, "        albedo: 0.9\n",
			"        albedo: 0.9\n      other air:\n"+
				"        weather: night haze\n        extent: world\n"+
				"        albedo: 0.5\n"), "one layer"},
	}
	for name, one := range cases {
		done := flatStaged(t, one.text)
		if done.err == nil || !strings.Contains(done.err.Error(), one.want) {
			t.Errorf("%s: staged with error %v, wanted one holding %q",
				name, done.err, one.want)
		}
	}
}
