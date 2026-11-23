package placement

import (
	"strings"
	"testing"
)

// recipeSource is a space set with a plain kind and tower, street and
// lashed street recipes. Its two towers are one kind with two pins.
const recipeSource = `$schema: https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: Display recipes
  setting: space
  seed: 19
  light: none
  recipes:
    low: {kind: tower, irregularity: 0, pins: {plan: rectangle, skin: concrete, storeys: 6, width: {value: 20, unit: m}, taper: false, twist: false, setbacks: false, spire: false}}
    high: {kind: tower, irregularity: 0, pins: {plan: rectangle, skin: concrete, storeys: 12, width: {value: 20, unit: m}, taper: false, twist: false, setbacks: false, spire: false}}
    wet: {kind: street, wetness: 0.7, length: {value: 50, unit: m}}
    lanterns: {kind: lashed-street, style: lantern-rows, street: {value: 12, unit: m}, length: {value: 48, unit: m}}
  set:
    things:
      visitor: {kind: human-scale-2m, seed: 7, complexity: quick, east: 0, north: 0, height: {value: 0, unit: m}}
      first: {recipe: low, seed: 3, complexity: quick, east: 30, north: 0, height: {value: 0, unit: m}}
      second: {recipe: high, seed: 3, complexity: quick, east: 60, north: 0, height: {value: 0, unit: m}}
      road: {recipe: wet, seed: 5, complexity: quick, east: 0, north: 40, height: {value: 0, unit: m}}
      lashed: {recipe: lanterns, seed: 7, complexity: quick, east: 0, north: 90, height: {value: 0, unit: m}}
  film: {width: 640, height: 400, wide: 40, rays: 1, exposure: 1}
  cameras:
    view:
      location: {at: [0, -60, 20]}
      orientation: {look_at: [30, 0, 10]}
`

// objectNamed is the display object of a placement, or fails the test.
func objectNamed(t *testing.T, scene *DisplayScene, name string) DisplayObject {
	t.Helper()
	for _, each := range scene.Objects {
		if each.Name == name {
			return each
		}
	}
	t.Fatalf("no display object named %q", name)
	return DisplayObject{}
}

func TestSceneMakesAThingFromARecipe(t *testing.T) {
	scene, err := NewDisplayCache().Scene(readSelection(t, recipeSource))
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"first", "second", "road", "lashed"} {
		mesh := objectNamed(t, scene, name).Mesh
		if mesh == nil || len(mesh.Vertices) == 0 || len(mesh.Indices) == 0 {
			t.Errorf("%s: a recipe thing has no mesh", name)
		}
	}
}

func TestSceneGivesTwoRecipesOfOneKindTwoMeshes(t *testing.T) {
	scene, err := NewDisplayCache().Scene(readSelection(t, recipeSource))
	if err != nil {
		t.Fatal(err)
	}
	low, high := objectNamed(t, scene, "first").Mesh, objectNamed(t, scene, "second").Mesh
	if low == high {
		t.Fatal("two recipes of one kind, one seed, share a mesh")
	}
	// a quick tower is drawn with the same count of corners whatever its
	// storeys, so the two are told apart by how high they stand.
	_, lowTop, lowHas := low.Bounds()
	_, highTop, highHas := high.Bounds()
	if !lowHas || !highHas {
		t.Fatal("a tower's mesh has no bounds")
	}
	if !(highTop.Y > lowTop.Y) {
		t.Fatalf("a twelve storey tower stands %v m and a six storey one %v m",
			highTop.Y, lowTop.Y)
	}
}

func TestSceneSharesOneRecipeAndSeedAcrossPlacements(t *testing.T) {
	twin := strings.Replace(recipeSource, "      second: {recipe: high,", "      second: {recipe: low,", 1)
	scene, err := NewDisplayCache().Scene(readSelection(t, twin))
	if err != nil {
		t.Fatal(err)
	}
	if objectNamed(t, scene, "first").Mesh != objectNamed(t, scene, "second").Mesh {
		t.Fatal("one recipe and one seed, made twice")
	}
}

func TestSceneRemakesAChangedRecipe(t *testing.T) {
	cache := NewDisplayCache()
	before, err := cache.Scene(readSelection(t, recipeSource))
	if err != nil {
		t.Fatal(err)
	}
	edited := strings.Replace(recipeSource, "storeys: 6,", "storeys: 9,", 1)
	after, err := cache.Scene(readSelection(t, edited))
	if err != nil {
		t.Fatal(err)
	}
	if objectNamed(t, before, "first").Mesh == objectNamed(t, after, "first").Mesh {
		t.Fatal("an edited recipe gave back its old mesh")
	}
	if objectNamed(t, before, "second").Mesh != objectNamed(t, after, "second").Mesh {
		t.Fatal("a recipe that was not edited was made again")
	}
}

func TestSceneKeepsAMeshWhenTheSceneIsReadAgain(t *testing.T) {
	cache := NewDisplayCache()
	before, err := cache.Scene(readSelection(t, recipeSource))
	if err != nil {
		t.Fatal(err)
	}
	after, err := cache.Scene(readSelection(t, recipeSource))
	if err != nil {
		t.Fatal(err)
	}
	for _, each := range before.Objects {
		if each.ID != "scene.world" && each.Mesh != objectNamed(t, after, each.Name).Mesh {
			t.Errorf("%s: the same document was made again", each.Name)
		}
	}
}

func TestSceneKeepsAPlainKindAndARecipeApart(t *testing.T) {
	scene, err := NewDisplayCache().Scene(readSelection(t, recipeSource))
	if err != nil {
		t.Fatal(err)
	}
	if objectNamed(t, scene, "visitor").Mesh == objectNamed(t, scene, "first").Mesh {
		t.Fatal("a plain kind and a recipe share a mesh")
	}
}

func TestSceneStillRefusesABurningPlacement(t *testing.T) {
	burning := strings.Replace(recipeSource, "      visitor: {kind: human-scale-2m,", "      visitor: {alight: true, kind: human-scale-2m,", 1)
	_, err := NewDisplayCache().Scene(readSelection(t, burning))
	if err == nil || !strings.Contains(err.Error(), "burning placements") {
		t.Fatalf("a burning placement was not refused: %v", err)
	}
}

func TestSceneRefusesWithoutASeed(t *testing.T) {
	bare := strings.Replace(recipeSource, "  seed: 19\n", "", 1)
	if _, err := NewDisplayCache().Scene(readSelection(t, bare)); err == nil {
		t.Fatal("a scene with no seed was shown")
	}
}
