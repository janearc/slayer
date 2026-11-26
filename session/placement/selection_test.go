package placement

import (
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/set"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const selectionSource = `$schema: https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: Native selection
  setting: space
  seed: 19
  light: none
  set:
    things:
      visitor:
        kind: human-scale-2m
        seed: 7
        complexity: quick
        east: 0
        north: 0
        height: {value: 0, unit: m}
        heading: {value: 0, unit: deg}
      farther:
        kind: human-scale-2m
        seed: 11
        complexity: quick
        east: 0
        north: 4
        height: {value: 0, unit: m}
  film: {width: 640, height: 400, wide: 40, rays: 1, exposure: 1}
  cameras:
    view:
      location: {at: [0, -6, 1]}
      orientation: {look_at: [0, 0, 1]}
`

func readSelection(t *testing.T, source string) set.Set {
	t.Helper()
	read, err := set.Read("selection.yaml", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	return read
}

func TestSpaceGeometrySelectsNearestActualHumanSurface(t *testing.T) {
	geometry, err := SpaceGeometry(readSelection(t, selectionSource))
	if err != nil {
		t.Fatal(err)
	}
	ray := maths.Ray{At: maths.V(0, 1.3, 6), Dir: maths.V(0, 0, -1)}
	if got := geometry.Hit(ray); got != "visitor" {
		t.Fatalf("hit %q", got)
	}
	ray.At = maths.V(0, 1.3, -1)
	if got := geometry.Hit(ray); got != "farther" {
		t.Fatalf("farther hit %q", got)
	}
}

func TestSpaceGeometryDoesNotSelectHumanBoundingBoxGap(t *testing.T) {
	geometry, err := SpaceGeometry(readSelection(t, selectionSource))
	if err != nil {
		t.Fatal(err)
	}
	for _, ray := range []maths.Ray{
		{At: maths.V(0, .3, 6), Dir: maths.V(0, 0, -1)},
		{At: maths.V(2, 1.3, 6), Dir: maths.V(0, 0, -1)},
		{At: maths.V(0, 1.3, 6), Dir: maths.V(0, 0, 1)},
		{}, {At: maths.V(math.NaN(), 0, 0), Dir: maths.V(0, 0, -1)},
	} {
		if got := geometry.Hit(ray); got != "" {
			t.Fatalf("false hit %q for %+v", got, ray)
		}
	}
}

func TestSpaceGeometryMovesAndTurnsNativeSurfaces(t *testing.T) {
	source := strings.Replace(selectionSource, "east: 0", "east: 3", 1)
	source = strings.Replace(source, "value: 0, unit: deg", "value: 90, unit: deg", 1)
	geometry, err := SpaceGeometry(readSelection(t, source))
	if err != nil {
		t.Fatal(err)
	}
	if got := geometry.Hit(maths.Ray{At: maths.V(3, 1.3, 6), Dir: maths.V(0, 0, -1)}); got != "visitor" {
		t.Fatalf("moved hit %q", got)
	}
	if got := geometry.Hit(maths.Ray{At: maths.V(0, 1.3, 6), Dir: maths.V(0, 0, -1)}); got != "farther" {
		t.Fatalf("stale geometry hit %q", got)
	}
}

func TestSpaceGeometryRefusesUnsupportedSet(t *testing.T) {
	for _, source := range []string{
		strings.Replace(selectionSource, "setting: space", "setting: ground", 1),
		strings.Replace(selectionSource, "        seed: 7", "        seed: 7\n        alight: true", 1),
	} {
		if _, err := SpaceGeometry(readSelection(t, source)); err == nil {
			t.Fatal("accepted unsupported set")
		}
	}
}

// These retained values pin the pre-refactor session stir finaliser.
func TestCanonicalConstructionSeedsRetainStagingValues(t *testing.T) {
	for _, row := range []struct {
		scene  uint64
		object int
		want   uint64
	}{
		{19, 7, 10507770595773694144}, {0, 0, 16294208416658607535}, {1, 0, 16490336266968443936},
	} {
		if got := Seed(row.scene, row.object); got != row.want {
			t.Fatalf("seed %d,%d = %d", row.scene, row.object, got)
		}
	}
	if Seed(1, 2) == Seed(2, 1) {
		t.Fatal("scene and object identity collapsed")
	}
}

func TestSpaceAtRetainsEastUpMinusNorthAndHeading(t *testing.T) {
	moved := strings.Replace(selectionSource, "east: 0", "east: 3", 1)
	read := readSelection(t, strings.Replace(moved, "value: 0, unit: deg", "value: 90, unit: deg", 1))
	at := SpaceAt(read.Things()[0])
	if at.Origin != maths.V(3, 0, 0) || at.Forward().Sub(maths.XAxis).Len() > 1e-12 {
		t.Fatal("wrong metre frame", at)
	}
	point := maths.V(1, 2, 3)
	if at.In(at.Out(point)).Sub(point).Len() > 1e-12 {
		t.Fatal("nonreversible transform")
	}
}

func TestSpaceAtKeepsOriginAndAppliesScale(t *testing.T) {
	source := strings.Replace(selectionSource, "        east: 0\n", "        east: 3\n        scale: 2\n", 1)
	read := readSelection(t, source)
	at := SpaceAt(read.Things()[0])
	if at.Origin != maths.V(3, 0, 0) || at.Scale != 2 || at.Out(maths.V(1, 0, 0)).Sub(at.Origin).Len() != 2 {
		t.Fatalf("scaled placement = %+v", at)
	}
}

func TestSpaceGeometryPicksScaledNativeSurface(t *testing.T) {
	baseRead := readSelection(t, selectionSource)
	scaledRead := readSelection(t, strings.Replace(selectionSource, "        east: 0\n", "        east: 0\n        scale: 2\n", 1))
	base, err := SpaceGeometry(baseRead)
	if err != nil {
		t.Fatal(err)
	}
	scaled, err := SpaceGeometry(scaledRead)
	if err != nil {
		t.Fatal(err)
	}
	display, err := NewDisplayCache().Scene(baseRead)
	if err != nil {
		t.Fatal(err)
	}
	low, high, ok := display.Objects[0].Mesh.Bounds()
	if !ok {
		t.Fatal("base object has no bounds")
	}
	right := math.Max(math.Abs(low.X), math.Abs(high.X))
	for ix := 11; ix <= 19; ix += 2 {
		for iy := 1; iy <= 9; iy++ {
			ray := maths.Ray{At: maths.V(right*float64(ix)/10, high.Y*2*float64(iy)/10, 6), Dir: maths.V(0, 0, -1)}
			if scaled.Hit(ray) == "visitor" && base.Hit(ray) != "visitor" {
				return
			}
		}
	}
	t.Fatal("no ray selected the enlarged part of the actual native surface")
}

func TestSpaceGeometrySelectsActualNativeLarson(t *testing.T) {
	source := strings.ReplaceAll(selectionSource, "human-scale-2m", "mcrn-larson-landed")
	read := readSelection(t, source)
	geometry, err := SpaceGeometry(read)
	if err != nil {
		t.Fatal(err)
	}
	row, err := manufacture.KindNamed("mcrn-larson-landed")
	if err != nil {
		t.Fatal(err)
	}
	specimen, err := Make(Construction{Kind: row, SceneSeed: 19, ObjectSeed: 7, Complexity: "quick"})
	if err != nil {
		t.Fatal(err)
	}
	box := specimen.Bounds()
	center := box.Min.Add(box.Max).Scale(.5)
	ray := maths.Ray{At: maths.V(center.X, center.Y, box.Max.Z+1), Dir: maths.V(0, 0, -1)}
	if got := geometry.Hit(ray); got != "visitor" {
		t.Fatalf("Larson actual surface hit = %q", got)
	}
	ray.At.X = box.Max.X + 1
	if got := geometry.Hit(ray); got != "" {
		t.Fatalf("outside Larson false hit %q", got)
	}
}

func TestSpaceGeometryRefusesUnsupportedManufactureRepresentation(t *testing.T) {
	source := strings.ReplaceAll(strings.ReplaceAll(selectionSource, "human-scale-2m", "can"), "complexity: quick", "complexity: hull")
	if _, err := SpaceGeometry(readSelection(t, source)); err == nil {
		t.Fatal("substituted unsupported specimen representation")
	}
}

func TestEmptyAndNilSpaceGeometryMiss(t *testing.T) {
	var absent *spaceGeometry
	ray := maths.Ray{At: maths.V(0, 1, 6), Dir: maths.V(0, 0, -1)}
	if absent.Hit(ray) != "" {
		t.Fatal("nil geometry selected")
	}
	source := selectionSource[:strings.Index(selectionSource, "  set:")] + selectionSource[strings.Index(selectionSource, "  film:"):]
	geometry, err := SpaceGeometry(readSelection(t, source))
	if err != nil || geometry.Hit(ray) != "" {
		t.Fatal("empty scene selected", err)
	}
}

func TestWorldSurfaceOccludesManufacturedSelection(t *testing.T) {
	source := selectionSource + `  world:
    materials:
      wall: {colour: {srgb_linear: [0.2, 0.2, 0.2]}, source: eye}
    boxes:
      blocker:
        material: wall
        middle: [0, -2, 1.3]
        size: [2, 0.1, 2]
        source: eye
`
	geometry, err := SpaceGeometry(readSelection(t, source))
	if err != nil {
		t.Fatal(err)
	}
	ray := maths.Ray{At: maths.V(0, 1.3, 6), Dir: maths.V(0, 0, -1)}
	if got := geometry.Hit(ray); got != "" {
		t.Fatalf("selected %q through native wall", got)
	}
	ray.At.Z = 1
	if got := geometry.Hit(ray); got != "visitor" {
		t.Fatalf("wall behind ray blocked %q", got)
	}
}

func TestSpaceGeometrySelectsFrozenAnimalProfile(t *testing.T) {
	dir := t.TempDir()
	raw, err := os.ReadFile(filepath.Join("..", "..", "manufacture", "animals", "vertebrates", "fishes", "fish.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "fish.yaml")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	source := `$schema: https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: animal selection
  setting: space
  seed: 19
  light: none
  recipes:
    fish: {animal: {family: fish, file: fish.yaml, species: pacific-bluefin-tuna}}
  set:
    things:
      visitor: {recipe: fish, seed: 7, complexity: quick, east: 0, north: 0, height: {value: 0, unit: m}}
  film: {width: 8, height: 8, rays: 1}
  cameras:
    view: {location: {at: [0, -10, 0]}, orientation: {look_at: [0, 0, 0]}}
`
	read, err := set.Read(filepath.Join(dir, "scene.yaml"), []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	geometry, err := SpaceGeometry(read)
	if err != nil {
		t.Fatal(err)
	}
	if len(geometry.surfaces) == 0 {
		t.Fatal("no selectable animal surfaces")
	}
	// The center of the first fish surface's box is inside its body.
	first := geometry.surfaces[0]
	box := first.shape.(interface{ Bounds() maths.Box }).Bounds()
	center := first.at.Out(box.Min.Add(box.Max).Scale(.5))
	ray := maths.Ray{At: center.Add(maths.V(0, 0, 10)), Dir: maths.V(0, 0, -1)}
	if got := geometry.Hit(ray); got != "visitor" {
		t.Fatalf("selected %q", got)
	}
}

func TestSpaceGeometryComparesScaledSurfaceDistances(t *testing.T) {
	near := selectionSurface{name: "near", shape: shape.Sphere{Radius: 1}, at: maths.NewSubspace(maths.V(0, 0, 3), maths.Identity, .5)}
	far := selectionSurface{name: "far", shape: shape.Sphere{Radius: 1}, at: maths.NewSubspace(maths.V(0, 0, 0), maths.Identity, 2)}
	ray := maths.Ray{At: maths.V(0, 0, 6), Dir: maths.V(0, 0, -1)}
	for _, surfaces := range [][]selectionSurface{{near, far}, {far, near}} {
		geometry := spaceGeometry{surfaces: surfaces}
		if got := geometry.Hit(ray); got != "near" {
			t.Fatalf("selected %q, want near", got)
		}
	}
}

func TestSpaceGeometryStillRefusesUnrelatedRecipes(t *testing.T) {
	if _, err := SpaceGeometry(readSelection(t, recipeSource)); err == nil {
		t.Fatal("accepted unsupported non-model recipe")
	}
}
