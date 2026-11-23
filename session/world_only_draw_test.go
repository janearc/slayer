//go:build draw

package session

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/moonlight"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/set"
	"github.com/janearc/slayer/stage"
	"github.com/janearc/slayer/trace"
)

// worldSets are the set files that say a world, a film and a camera and
// stand on no ground: sets that the world half alone draws.
var worldSets = []string{"shoebox", "widow", "widow-smoke", "widow-tail",
	"fizz"}

// readable is the mode a test's file is written with: the owner writes it
// and everyone reads it.
const readable = 0o644 // defined: the owner writes, everyone reads

// put writes a file for a test, and fails the test where it cannot.
func put(t *testing.T, path, written string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(written), readable); err != nil {
		t.Fatal(err)
	}
}

// readSet is a set file, read.
func readSet(t *testing.T, path string) set.Set {
	t.Helper()
	written, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	read, err := set.Read(path, written)
	if err != nil {
		t.Fatal(err)
	}
	return read
}

// once is how many places of a text a test changes.
const once = 1 // defined: one place

// replaced is a text with a piece of it changed, where it is there.
func replaced(t *testing.T, text, was, now string) string {
	t.Helper()
	if !strings.Contains(text, was) {
		t.Fatalf("%q is not in the text", was)
	}
	return strings.Replace(text, was, now, once)
}

// widowChanged is the widow set with a piece of its text changed, written
// to a fresh directory; and its path.
func widowChanged(t *testing.T, was, now string) string {
	t.Helper()
	written, err := os.ReadFile("../sets/widow.yaml")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "widow.yaml")
	changed := replaced(t, string(written), was, now)
	put(t, path, changed)
	return path
}

// drawnEye is where a set's first camera stands and faces, as slayer's
// set verb takes it: the first camera named, through the shot the set
// makes of it.
func drawnEye(t *testing.T, read set.Set) *maths.Subspace {
	t.Helper()
	for _, name := range read.Cameras() {
		shot, err := read.Shot(name, nowhere)
		if err != nil {
			t.Fatal(err)
		}
		eye := shot.Eye()
		return &eye
	}
	t.Fatal("the set has no camera")
	return nil
}

// samePieces says where two worlds differ, list by list, as exact as
// DeepEqual is.
func samePieces(t *testing.T, name string, got, want *thing.World) {
	t.Helper()
	if len(got.Things) != len(want.Things) ||
		len(got.Volumes) != len(want.Volumes) ||
		len(got.Lamps) != len(want.Lamps) ||
		len(got.Fields) != len(want.Fields) {
		t.Fatalf("%s: %d things, %d volumes, %d lamps, %d fields; wanted "+
			"%d, %d, %d, %d", name, len(got.Things), len(got.Volumes),
			len(got.Lamps), len(got.Fields), len(want.Things),
			len(want.Volumes), len(want.Lamps), len(want.Fields))
	}
	for i := range want.Things {
		if !reflect.DeepEqual(got.Things[i], want.Things[i]) {
			t.Errorf("%s: thing %d differs", name, i)
		}
	}
	for i := range want.Volumes {
		if !reflect.DeepEqual(got.Volumes[i], want.Volumes[i]) {
			t.Errorf("%s: volume %d differs", name, i)
		}
	}
	for i := range want.Lamps {
		if !reflect.DeepEqual(got.Lamps[i], want.Lamps[i]) {
			t.Errorf("%s: lamp %d differs", name, i)
		}
	}
	if !reflect.DeepEqual(got.Sky, want.Sky) ||
		got.VolumeIllumination != want.VolumeIllumination {
		t.Errorf("%s: the sky or the volume lighting differs", name)
	}
}

// a set of a world, a film and a camera, with no ground, is staged as the
// stage stages its world: the same pieces, the set's own lamps and sky and
// none that hillside adds; its eye is its first camera's, and its lens
// the film's, as slayer's set verb has them. a film that says no size is
// drawn at the size the flags say, and one that does is drawn at its own.
func TestAWorldOnlySetIsStagedAsTheStageStagesIt(t *testing.T) {
	for _, name := range worldSets {
		path := filepath.Join("..", "sets", name+".yaml")
		read := readSet(t, path)
		said, _ := read.World()
		want, err := stage.World(said)
		if err != nil {
			t.Fatal(err)
		}
		flagged := mustParse(t)
		given := flagged
		given.said = sizing{}
		given.set = path
		world, made, got, err := staged(given)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		samePieces(t, name, world, want)
		eye, filmed := drawnEye(t, read), filmOf(read)
		if got.eye == nil || *got.eye != *eye {
			t.Errorf("%s: the eye is %v, not %v", name, got.eye, eye)
		}
		if lensFor(got) != stage.Lens(filmed) {
			t.Errorf("%s: the lens is %v, not %v", name, lensFor(got),
				stage.Lens(filmed))
		}
		wide, tall, rays := filmed.Size()
		if wide == nothing {
			wide, tall, rays = flagged.width, flagged.height, flagged.samples
		}
		if got.width != wide || got.height != tall || got.samples != rays {
			t.Errorf("%s: drawn %dx%d at %d, not %dx%d at %d", name,
				got.width, got.height, got.samples, wide, tall, rays)
		}
		if made.lamps != len(want.Lamps) || made.things != len(want.Things) {
			t.Errorf("%s: tallied %d things and %d lamps of %d and %d",
				name, made.things, made.lamps, len(want.Things),
				len(want.Lamps))
		}
	}
}

func TestWorldOnlyUnderwaterSceneStagesModeledMoonAndSkyTransport(t *testing.T) {
	const authored = `"$schema": https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: water under full moon
  setting: space
  light: moon
  moon: {azimuth: 230, altitude: 22, disc: false}
  film: {width: 1, height: 1, rays: 1, exposure: 100000, wide: 40}
  world:
    unit: 1
    materials:
      seawater:
        colour: {srgb_linear: [0.01, 0.04, 0.06]}
        clear:
          through: 1
          index: 1.333
          gloss: 1
          rank: 0
          extinction-per-metre: [0.03, 0.008, 0.004]
          scatter:
            coefficient-per-metre: [0.005, 0.003, 0.002]
            phase-g: 0
            steps: 8
            max-distance: {value: 80, unit: m}
        source: "synthetic test medium"
    ocean:
      material: seawater
      middle: [0, 0, 0]
      extent: {value: 16, unit: m}
      samples: 9
      sea:
        wind: {value: 1, unit: m/s}
        fetch: {value: 1000, unit: m}
        depth: {value: 0, unit: m}
        toward: {value: 0, unit: rad}
      seed: 1
      time: {value: 0, unit: s}
      diffuse-sky-samples: 8
      flat-mean:
        mean-level: {value: 0, unit: m}
        collar-width: {value: 4, unit: m}
        extends-to-infinity: true
      source: "test continuation"
  cameras:
    underwater-control:
      location: {at: [0, -10, -10]}
      orientation: {look_at: [0, 0, 0]}
      initial-media: [seawater]
`
	read, err := set.Read("world-only-moon-ocean.yaml", []byte(authored))
	if err != nil {
		t.Fatal(err)
	}
	drawn := stagedWorld(read, asked{set: "world-only-moon-ocean.yaml"})
	if drawn.err != nil {
		t.Fatal(drawn.err)
	}
	if len(drawn.world.Lamps) != 1 {
		t.Fatalf("staged moon lamps=%d, want one", len(drawn.world.Lamps))
	}
	if _, ok := drawn.world.Lamps[0].(emit.Distant); !ok {
		t.Fatalf("moon lamp is %T, want the modeled distant source", drawn.world.Lamps[0])
	}
	if _, ok := drawn.world.Sky.(moonlight.Skies); !ok {
		t.Fatalf("moon sky is %T, want the computed moonlit atmosphere sky", drawn.world.Sky)
	}
	if err := trace.ValidateClearState(drawn.world, drawn.given.initialMedia); err != nil {
		t.Fatalf("world-only sky-scatter preflight: %v", err)
	}
}

// a set with a world and no film is refused as every set with no film is,
// and one with no camera is refused by name.
func TestAWorldOnlySetNeedsAFilmAndACamera(t *testing.T) {
	for _, mistake := range []struct{ was, now, said string }{
		{"  film:\n    wide: 40\n    print: {knee: 0.7, top: 8, " +
			"source: eye}\n", "", "says no film"},
		{"  cameras:\n    eye:\n      location:\n        at: [0, -230, 45]\n" +
			"      orientation:\n        look_at: [0, 0, 0]\n", "",
			"no camera"},
	} {
		given := mustParse(t)
		given.set = widowChanged(t, mistake.was, mistake.now)
		if _, _, _, err := staged(given); err == nil ||
			!strings.Contains(err.Error(), mistake.said) {
			t.Errorf("%q: %v, not %q", mistake.now, err, mistake.said)
		}
	}
}

// the print a world-only set's film says is how its picture is developed:
// its exposure, knee and top, written to the picture as the tone says; a
// film that says none is developed as hillside's own tone does.
func TestAWorldOnlySetIsDevelopedThroughItsFilmsPrint(t *testing.T) {
	given := mustParse(t)
	given.set = widowChanged(t, "    print: {knee: 0.7, top: 8, source: eye}\n",
		"    exposure: 40\n    print: {knee: 0.5, top: 4, source: eye}\n")
	world, _, got, err := staged(given)
	if err != nil {
		t.Fatal(err)
	}
	filmed := filmOf(readSet(t, given.set))
	knee, top, _ := filmed.Print()
	tone := got.tone()
	if tone.Exposure != filmed.Exposure() || tone.Knee != knee ||
		tone.Top != top || tone == developing {
		t.Fatalf("the tone is %+v, not the film's exposure %v, knee %v "+
			"and top %v", tone, filmed.Exposure(), knee, top)
	}
	drawn, err := draw(world, got, nothing)
	if err != nil {
		t.Fatal(err)
	}
	got.out = filepath.Join(t.TempDir(), "tone.png")
	if err := save(drawn, got, thing.NewWorld(), nothing); err != nil {
		t.Fatal(err)
	}
	written, err := os.ReadFile(got.out)
	if err != nil {
		t.Fatal(err)
	}
	reference, plain := tone.Develop(drawn), developing.Develop(drawn)
	if bytes.Equal(reference.Pix, plain.Pix) {
		t.Fatal("the film's print and hillside's own develop alike here")
	}
	actual, err := png.Decode(bytes.NewReader(written))
	if err != nil {
		t.Fatal(err)
	}
	if !samePicture(actual, reference) {
		t.Error("the saved picture is not the film's print of the light")
	}
}

// sameColour is whether two colours are alike, as the screen has them.
func sameColour(got, want color.Color) bool {
	gr, gg, gb, ga := got.RGBA()
	wr, wg, wb, wa := want.RGBA()
	return gr == wr && gg == wg && gb == wb && ga == wa
}

// samePicture is whether two pictures are alike over the bounds of the
// second.
func samePicture(got, want image.Image) bool {
	bounds := want.Bounds()
	if got.Bounds() != bounds {
		return false
	}
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if !sameColour(got.At(x, y), want.At(x, y)) {
				return false
			}
		}
	}
	return true
}

// filmOf is a set's film, whether or not it says a size.
func filmOf(read set.Set) set.Film {
	film, _ := read.Film()
	return film
}

// a film that says no print is developed as hillside's own tone does.
func TestAFilmWithNoPrintIsDevelopedAsHillsideDoes(t *testing.T) {
	given := mustParse(t)
	given.set = widowChanged(t, "    print: {knee: 0.7, top: 8, source: eye}\n",
		"")
	_, _, got, err := staged(given)
	if err != nil || got.tone() != developing {
		t.Errorf("developed as %+v: %v", got.tone(), err)
	}
}

// a world-only set whose film says no width is seen through the pinhole's
// own, not through the degrees of hillside's own lens.
func TestAWorldOnlySetWithNoWidthIsSeenThroughThePinholesOwnLens(t *testing.T) {
	given := mustParse(t)
	given.set = widowChanged(t, "    wide: 40\n", "")
	_, _, got, err := staged(given)
	if err != nil {
		t.Fatal(err)
	}
	filmed := filmOf(readSet(t, given.set))
	if lensFor(got) != stage.Lens(filmed) || stage.Lens(filmed).Wide != nothing {
		t.Errorf("the lens is %v, not the pinhole's own", lensFor(got))
	}
}
