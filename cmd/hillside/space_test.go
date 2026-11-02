package main

import (
	"crypto/sha256"
	"encoding/hex"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/planet"
)

// spaceScene is a set in space: a planet, its star and a camera south
// of it, a million kilometres off.
const spaceScene = `scene:
  name: a giant
  light: sun
  sun: {azimuth: 180, altitude: 0}
  film: {width: 24, height: 16, rays: 1, exposure: 1}
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
    zones:
      - {south: -90, north: 90, column: clear}
`

// inSpace is a set file in space in a fresh directory, beside a shelf
// holding a methane table and a sources.yaml pinning it, the scene
// changed as asked; and the set file's path.
func inSpace(t *testing.T, was, now string) string {
	t.Helper()
	root := t.TempDir()
	shelf := filepath.Join(root, "shelf")
	os.MkdirAll(shelf, 0o755)
	table := []byte("300.4 300.31 .0000 .5\n619.2 619.01 .7132 .1\n")
	os.WriteFile(filepath.Join(shelf, "karkoschka-1994-methane.txt"),
		table, 0o644)
	sum := sha256.Sum256(table)
	os.WriteFile(filepath.Join(root, "sources.yaml"), []byte(
		"sources:\n  - key: karkoschka-1994-methane\n"+
			"    from: https://x.test/karkoschka-1994-methane.txt\n"+
			"    licence: a test's own\n    sha256: "+
			hex.EncodeToString(sum[:])+"\n"), 0o644)
	os.MkdirAll(filepath.Join(root, "sets"), 0o755)
	path := filepath.Join(root, "sets", "space.yaml")
	os.WriteFile(path, []byte(strings.Replace(spaceScene, was, now, 1)),
		0o644)
	t.Setenv("SLAYER_SHELF", shelf)
	return path
}

// a set in space is its planet alone, at the middle, its face worked
// from its air, against darkness: lit on the side facing the star and
// the camera, and dark on the far side.
func TestASetInSpaceIsItsPlanet(t *testing.T) {
	given := mustParse(t)
	given.set = inSpace(t, "", "")
	world, made, given, err := staged(given)
	if err != nil {
		t.Fatal(err)
	}
	if len(world.Things) != 1 || made.things != 1 || given.eye == nil {
		t.Fatalf("%d things, tallied %d, eye %v", len(world.Things),
			made.things, given.eye)
	}
	face, isFace := world.Things[0].Material.Shine.(*planet.Face)
	if !isFace || world.Things[0].Material.Name != "the giant" {
		t.Fatalf("material %+v", world.Things[0].Material)
	}
	if _, dark := world.Sky.(darkness); !dark {
		t.Errorf("the sky is %T", world.Sky)
	}
	near := face.At(maths.V(0, 0, 6e7))
	far := face.At(maths.V(0, 0, -6e7))
	if !(near.G > 0) || far != (maths.Light{}) {
		t.Errorf("near side %+v, far side %+v", near, far)
	}
}

// a planet with no sun to light it, and a planet over terrain, are
// refused.
func TestASetInSpaceIsHeld(t *testing.T) {
	given := mustParse(t)
	given.set = inSpace(t, "  sun: {azimuth: 180, altitude: 0}\n", "")
	if _, _, _, err := staged(given); err == nil ||
		!strings.Contains(err.Error(), "needs a sun") {
		t.Errorf("no sun: %v", err)
	}
	given.set = onFixture(t, "  terrain:", "  planet:\n    name: p\n"+
		"    made_up: true\n    radius: 1.0e7\n    gravity: 10\n"+
		"    from_star: 1\n    columns:\n      a:\n        - {top: 0, "+
		"bottom: 1, gases: {neon: 1}}\n    zones:\n      - {south: -90, "+
		"north: 90, column: a}\n  terrain:")
	if _, _, _, err := staged(given); err == nil ||
		!strings.Contains(err.Error(), "over terrain") {
		t.Errorf("a planet over terrain: %v", err)
	}
}

// a star's light falls as its distance squared, in every band.
func TestStarlightFallsAsTheSquare(t *testing.T) {
	near, far := starlight(1), starlight(2)
	first, last := near.Span()
	for band := first; band <= last; band++ {
		if want := near.Power(band) / 4; far.Power(band) != want &&
			!(far.Power(band)-want < want*1e-12) {
			t.Errorf("band %d: %g, want %g", band, far.Power(band), want)
		}
	}
	if !(near.Power((first+last)/2) > 0) {
		t.Error("the star gives no light")
	}
}

// a planet's rings are each a volume tipped to its pole, and its star is
// a lamp from the set's sun's way, so the rings are lit and the planet
// shadows them.
func TestAPlanetsRingsAreVolumesUnderItsStar(t *testing.T) {
	given := mustParse(t)
	given.set = inSpace(t, "    zones:\n", "    pole: {azimuth: 180, "+
		"altitude: 40}\n    rings:\n      - {inner: 7.8e7, outer: 8.7e7, "+
		"depth: 0.1, albedo: 0.6, lean: -0.3, made_of: {ice: 1}}\n"+
		"      - {inner: 9.0e7, outer: 1.0e8, depth: 1, albedo: 0.8, "+
		"lean: -0.3, made_of: {ice: 1}}\n    zones:\n")
	world, made, _, err := staged(given)
	if err != nil {
		t.Fatal(err)
	}
	if len(world.Volumes) != 2 || len(world.Lamps) != 1 || made.lamps != 1 {
		t.Fatalf("%d volumes, %d lamps", len(world.Volumes),
			len(world.Lamps))
	}
	pole := skyward(180, 40)
	up := world.Volumes[1].At(0).Orientation.Out(maths.V(0, 1, 0))
	if math.Abs(up.Dot(pole)-1) > 1e-12 {
		t.Errorf("the rings' up is %v, the pole %v", up, pole)
	}
	if slab, isSlab := world.Volumes[1].Medium.(planet.RingSlab); !isSlab ||
		slab.Density(maths.V(9.5e7, 0, 0)) != 1 {
		t.Errorf("the second ring's medium %T", world.Volumes[1].Medium)
	}
}
