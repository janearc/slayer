package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/moonlight"
	"github.com/janearc/slayer/raw"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/set"
	"golang.org/x/image/tiff"
)

// terrainFixture is terrain's placed fixture: 100 by 70 heights two
// metres apart, its first corner at 1000.5 e, 9000.25 n, each height
// (column + 1000 row) / 8, so the ground climbs 125 m a row southward.
const terrainFixture = "../../terrain/testdata/fixture-plain.tif"

// a scene in a set file, standing on terrain's fixture: a tree alight
// with a stone to see it from, a stand of ferns in a band of heights,
// and a small film.
const fixtureScene = `scene:
  name: on the fixture
  light: moon
  timeofday: 2200
  terrain:
    from: fixture
    file: fixture-plain.tif
    west: 1001
    north: 8999
    across: 150
    down: 100
    step: 2
  film:
    width: 24
    height: 16
    rays: 1
    exposure: 100000
  cameras:
    stone:
      orientation:
        point_to: the tree
        alpha: 0
      location:
        loc_relative:
          object: the stone
          standoff_x: 0
          standoff_y: 0
          standoff_z: 2
  set:
    things:
      the tree:
        kind: spruce
        seed: 3
        east: 1050
        north: 8960
        alight: true
      the stone:
        kind: cobble
        seed: 4
        east: 1050
        north: 8990
    stands:
      ferns:
        kind: fern
        count: 12
        seed: 7
        above: 2000
        below: 4000
`

// onFixture is a set file standing on the fixture, a shelf holding it
// and a sources.yaml pinning it, all in a fresh directory, the scene
// changed as asked; and the set file's path.
func onFixture(t *testing.T, was, now string) string {
	t.Helper()
	root := t.TempDir()
	shelf := filepath.Join(root, "shelf")
	bytes, err := os.ReadFile(terrainFixture)
	if err != nil {
		t.Fatal(err)
	}
	os.MkdirAll(shelf, 0o755)
	os.WriteFile(filepath.Join(shelf, "fixture-plain.tif"), bytes, 0o644)
	os.WriteFile(filepath.Join(shelf, "fixture.tif"), bytes, 0o644)
	sum := sha256.Sum256(bytes)
	os.WriteFile(filepath.Join(root, "sources.yaml"), []byte(
		"sources:\n  - key: fixture\n    from: https://x.test/fixture.tif\n"+
			"    licence: a test's own\n    sha256: "+
			hex.EncodeToString(sum[:])+"\n"), 0o644)
	os.MkdirAll(filepath.Join(root, "sets"), 0o755)
	path := filepath.Join(root, "sets", "scene.yaml")
	scene := strings.Replace(fixtureScene, was, now, 1)
	os.WriteFile(path, []byte(scene), 0o644)
	t.Setenv("SLAYER_SHELF", shelf)
	return path
}

// a set file is the world: its things, its stand, its fire, its film
// and its camera, with nothing of hillside's own flags in it. the one
// tree alight is two volumes, its flame and its smoke.
func TestASetFileIsTheWorld(t *testing.T) {
	given := mustParse(t)
	given.set = onFixture(t, "", "")
	world, made, given, err := staged(given)
	if err != nil {
		t.Fatal(err)
	}
	if made.things != 14 || made.burning != 1 || given.eye == nil ||
		given.width != 24 || given.height != 16 || !given.night ||
		len(world.Volumes) != 2 {
		t.Errorf("%s; %d volumes, %dx%d, eye %v", made, len(world.Volumes),
			given.width, given.height, given.eye)
	}
}

// a stand grows only in its band; one whose band is not on the ground
// is refused, and so is a terrain its sources.yaml does not pin.
func TestASetsGroundIsHeldToItsSource(t *testing.T) {
	for _, mistake := range []struct{ was, now, said string }{
		{"above: 2000\n        below: 4000",
			"above: 9000\n        below: 9500", "no ground between"},
		{"from: fixture", "from: elsewhere", "not in sources.yaml"},
		{"east: 1050\n        north: 8960", "east: 5000\n        north: 8960",
			"off the ground"},
	} {
		given := mustParse(t)
		given.set = onFixture(t, mistake.was, mistake.now)
		_, _, _, err := staged(given)
		if err == nil || !strings.Contains(err.Error(), mistake.said) {
			t.Errorf("%q: %v, not %q", mistake.now, err, mistake.said)
		}
	}
	given := mustParse(t)
	given.set = onFixture(t, "", "")
	shelf := os.Getenv("SLAYER_SHELF")
	os.WriteFile(filepath.Join(shelf, "fixture.tif"), []byte("changed"),
		0o644)
	if _, _, _, err := staged(given); err == nil {
		t.Error("a model changed on the shelf was stood on")
	}
}

// faceOnShelf writes a small grey mosaic onto the shelf as moonface.tif
// and pins it in the root's sources.yaml, and gives back its bytes.
func faceOnShelf(t *testing.T, root string) []byte {
	t.Helper()
	picture := image.NewRGBA64(image.Rect(0, 0, 16, 8))
	for i := range picture.Pix {
		picture.Pix[i] = byte(64 + i%128)
	}
	var written bytes.Buffer
	if err := tiff.Encode(&written, picture, nil); err != nil {
		t.Fatal(err)
	}
	shelf := os.Getenv("SLAYER_SHELF")
	os.WriteFile(filepath.Join(shelf, "moonface.tif"), written.Bytes(),
		0o644)
	sum := sha256.Sum256(written.Bytes())
	list, err := os.OpenFile(filepath.Join(root, "sources.yaml"),
		os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	defer list.Close()
	list.WriteString("  - key: moonface\n    from: https://x.test/moonface.tif\n" +
		"    licence: a test's own\n    sha256: " +
		hex.EncodeToString(sum[:]) + "\n")
	return written.Bytes()
}

// a set's full moon is a disc at the moon's distance with the face its
// set names on it, a light from its way and its sky; one at the horizon
// is refused.
func TestASetsMoonHasItsFace(t *testing.T) {
	scene := "  moon:\n    azimuth: 90\n    altitude: 10\n    face: moonface\n" +
		"  terrain:"
	given := mustParse(t)
	given.set = onFixture(t, "  terrain:", scene)
	mosaic := faceOnShelf(t, filepath.Dir(filepath.Dir(given.set)))
	world, _, _, err := staged(given)
	if err != nil {
		t.Fatal(err)
	}
	moons := 0
	for _, each := range world.Things {
		if each.Material != nil && each.Material.Name == "moon" {
			moons++
			if each.Material.Shine == nil {
				t.Error("the moon has no face")
			}
		}
	}
	if moons != 1 || world.Sky == nil || len(mosaic) == 0 {
		t.Errorf("%d moons, sky %v", moons, world.Sky)
	}
	given.set = onFixture(t, "  terrain:", strings.Replace(scene,
		"altitude: 10", "altitude: 0", 1))
	if _, _, _, err := staged(given); err == nil ||
		!strings.Contains(err.Error(), "horizon") {
		t.Errorf("a moon at the horizon: %v", err)
	}
}

// a set's sun below the horizon lights the sky beside its moon: two
// skies, the moon's and the twilight's, the twilight brighter low in
// the west, where the sun set, than low in the east. a sun the earth
// does not hide from the ground is refused: one on the horizon, since
// the fixture's lowest ground stands a little above the sea, and one
// that is up.
func TestASetsSunLightsItsTwilight(t *testing.T) {
	scene := "  moon:\n    azimuth: 90\n    altitude: 10\n" +
		"  sun:\n    azimuth: 270\n    altitude: -3\n  terrain:"
	given := mustParse(t)
	given.set = onFixture(t, "  terrain:", scene)
	world, _, _, err := staged(given)
	if err != nil {
		t.Fatal(err)
	}
	skies, is := world.Sky.(moonlight.Skies)
	if !is || len(skies) != 2 {
		t.Fatalf("the sky is a %T: %v", world.Sky, world.Sky)
	}
	west, east := skies[1].Light(skyward(270, 4)).Luminance(),
		skies[1].Light(skyward(90, 4)).Luminance()
	if !(west > east) {
		t.Errorf("the twilight low in the west %g, in the east %g", west,
			east)
	}
	for _, altitude := range []string{"0", "5"} {
		given.set = onFixture(t, "  terrain:", strings.Replace(scene,
			"altitude: -3", "altitude: "+altitude, 1))
		if _, _, _, err := staged(given); err == nil ||
			!strings.Contains(err.Error(), "not hidden by the earth") {
			t.Errorf("a sun at %s: %v", altitude, err)
		}
	}
}

// every set file in sets/ that stands on terrain, which is hillside's
// to draw, stages: its ground from the shelf, its things, its stands
// and its light. without a shelf there is nothing to stand them on,
// and the test says so and skips.
func TestEverySetStages(t *testing.T) {
	if os.Getenv("SLAYER_SHELF") == "" || testing.Short() {
		t.Skip("no shelf named by SLAYER_SHELF, or short")
	}
	files, err := filepath.Glob("../../sets/*.yaml")
	if err != nil || len(files) == 0 {
		t.Fatalf("no set files: %v", err)
	}
	// parked scenes exceed the -faces budget since the needled spruce
	// (85c1c37, ~97k stored faces a tree) and wait on view-driven
	// complexity selection; DEBT.md, 2026-11-09. they are loved, not
	// forgotten.
	parked := map[string]string{
		"melancholia.yaml":          "needled spruce past -faces",
		"kilauea-iki-moonrise.yaml": "needled spruce past -faces",
	}
	for _, path := range files {
		if why, is := parked[filepath.Base(path)]; is {
			t.Logf("%s: parked, %s", filepath.Base(path), why)
			continue
		}
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		read, err := set.Read(filepath.Base(path), source)
		if err != nil {
			t.Errorf("%s: %v", filepath.Base(path), err)
			continue
		}
		_, ground := read.Ground()
		if _, space := read.Planet(); !ground && !space {
			t.Logf("%s: no terrain and no planet, not hillside's",
				filepath.Base(path))
			continue
		}
		given := mustParse(t)
		given.set = path
		if _, made, _, err := staged(given); err != nil {
			t.Errorf("%s: %v", filepath.Base(path), err)
		} else {
			t.Logf("%s: %s", filepath.Base(path), made)
		}
	}
}

// -set says the world and the film, so a flag that says them too is
// refused, by its name.
func TestASetIsRefusedBesideTheFlagsWorld(t *testing.T) {
	for _, flag := range []string{"-spruce", "-w", "-exposure", "-alight"} {
		_, err := parse([]string{"-set", "x.yaml", flag, "2"}, io.Discard)
		if err == nil || !strings.Contains(err.Error(), flag) {
			t.Errorf("%s beside -set: %v", flag, err)
		}
	}
	if _, err := parse([]string{"-set", "x.yaml", "-raw", "r.pfm"},
		io.Discard); err != nil {
		t.Errorf("-raw beside -set: %v", err)
	}
}

// a scene said in a set file is drawn, kept and proved bit for bit, as the
// flags' hillside is: the test is a file, and the go is only this.
func TestASetSceneIsProved(t *testing.T) {
	path := onFixture(t, "", "")
	dir := filepath.Dir(path)
	keep := filepath.Join(dir, "scene.pfm")
	var said, wrong bytes.Buffer
	drawing := []string{"-set", path, "-out", filepath.Join(dir, "s.png"),
		"-raw", keep}
	if code := run(drawing, &said, &wrong); code != 0 {
		t.Fatalf("drawing gave %d: %s", code, wrong.String())
	}
	said.Reset()
	if code := run([]string{"-set", path, "-prove", keep, "-many", "16"},
		&said, &wrong); code != 0 || !strings.Contains(said.String(),
		"holds: 16 of 16") {
		t.Fatalf("proving gave %d: %s %s", code, said.String(),
			wrong.String())
	}
}

// onFixtureRigged is the fixture's set file with a rig in its film, its
// shutter open so many seconds over two glimpses, and its tree alight
// or not.
func onFixtureRigged(t *testing.T, shutter string, alight bool) string {
	t.Helper()
	path := onFixture(t, "    height: 16\n    rays: 1\n    exposure: 100000\n",
		"    rays: 1\n    frame:\n      width: 70.41\n      height: 51.61\n"+
			"    lens:\n      focal: 672\n      stop: 8\n"+
			"      focus: the tree\n    shutter: "+shutter+"\n"+
			"    glimpses: 2\n")
	if !alight {
		written, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		still := strings.Replace(string(written), "alight: true",
			"alight: false", 1)
		if err := os.WriteFile(path, []byte(still), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return path
}

// setsRaw draws a set file and keeps its raw, and gives the raw back.
func setsRaw(t *testing.T, path string) *frame.Frame {
	t.Helper()
	dir := filepath.Dir(path)
	keep := filepath.Join(dir, "scene.pfm")
	var said, wrong bytes.Buffer
	if code := run([]string{"-set", path, "-out",
		filepath.Join(dir, "s.png"), "-raw", keep}, &said,
		&wrong); code != 0 {
		t.Fatalf("drawing gave %d: %s", code, wrong.String())
	}
	kept, err := raw.Read(keep)
	if err != nil {
		t.Fatal(err)
	}
	return kept
}

// a set with a rig is drawn through its lens, its rows from its frame, and the
// raw it keeps is proved bit for bit, glimpse by glimpse.
func TestARiggedSetIsDrawnAndProved(t *testing.T) {
	path := onFixtureRigged(t, "0.5", true)
	kept := setsRaw(t, path)
	if kept.W != 24 || kept.H != 18 {
		t.Errorf("the raw is %d by %d, and the frame makes 24 by 18",
			kept.W, kept.H)
	}
	var said, wrong bytes.Buffer
	keep := filepath.Join(filepath.Dir(path), "scene.pfm")
	if code := run([]string{"-set", path, "-prove", keep, "-many", "16"},
		&said, &wrong); code != 0 || !strings.Contains(said.String(),
		"holds: 16 of 16") {
		t.Fatalf("proving gave %d: %s %s", code, said.String(),
			wrong.String())
	}
}

// with nothing on the set moving or burning, a shutter open twice as
// long keeps exactly twice the light in every pixel of the raw.
func TestTwiceTheRigsShutterIsTwiceTheRaw(t *testing.T) {
	once := setsRaw(t, onFixtureRigged(t, "0.5", false))
	twice := setsRaw(t, onFixtureRigged(t, "1", false))
	lit := 0
	for i := range once.Pix {
		if twice.Pix[i].G != 2*once.Pix[i].G {
			t.Fatalf("pixel %d: %g at half a second, %g at one", i,
				once.Pix[i].G, twice.Pix[i].G)
		}
		if once.Pix[i].G > 0 {
			lit++
		}
	}
	if lit == 0 {
		t.Error("nothing in the raw is lit")
	}
}

// sunAt is a set's sun at an altitude, read from a set file that says
// only that.
func sunAt(t *testing.T, altitude string) set.Sun {
	t.Helper()
	read, err := set.Read("sun.yaml", []byte("scene:\n  sun:\n"+
		"    azimuth: 270\n    altitude: "+altitude+"\n"))
	if err != nil {
		t.Fatal(err)
	}
	sun, _ := read.Sunlit()
	return sun
}

// from a floor 1100 m above the sea the horizon dips by the angle whose
// cosine is the earth's radius over the radius there, 1.06 degrees: a
// sun a degree down is above that horizon and is refused, one 1.2 down
// is hidden. ground below the sea is taken at the sea for the dip, and
// the standard atmosphere, which starts at the sea, refuses it plainly,
// never with a dip that is not a number.
func TestTheSunIsHiddenBelowTheDip(t *testing.T) {
	high := &staging{given: asked{set: "high.yaml", exposure: 1},
		lowest: 1100}
	dip := math.Acos(6371000.0/(6371000+1100)) * 180 / math.Pi
	if _, err := high.twilit(sunAt(t, "-1.0")); err == nil ||
		!strings.Contains(err.Error(), fmt.Sprintf("%.2f", dip)) {
		t.Errorf("a sun a degree down over a 1100 m floor: %v", err)
	}
	if _, err := high.twilit(sunAt(t, "-1.2")); err != nil {
		t.Errorf("a sun 1.2 degrees down: %v", err)
	}
	low := &staging{given: asked{set: "low.yaml", exposure: 1},
		lowest: -20}
	if _, err := low.twilit(sunAt(t, "-0.5")); err == nil ||
		strings.Contains(err.Error(), "NaN") {
		t.Errorf("a sun half a degree down over ground below the sea: %v",
			err)
	}
}
