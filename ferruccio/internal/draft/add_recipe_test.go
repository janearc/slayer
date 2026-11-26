package draft

import (
	"math"
	"strings"
	"testing"
)

const addRecipeSource = `$schema: https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: Recipe additions # Keep this note.
  setting: space
  seed: 19
  light: none
  recipes:
    low: {kind: tower, irregularity: 0, pins: {plan: rectangle, skin: concrete, storeys: 6, width: {value: 20, unit: m}, taper: false, twist: false, setbacks: false, spire: false}}
  set:
    things:
      first: {recipe: low, seed: 3, complexity: quick, east: 30, north: 0, height: {value: 0, unit: m}}
  film: {width: 640, height: 400, wide: 40, rays: 1, exposure: 1}
  cameras:
    view:
      location: {at: [0, -60, 20]}
      orientation: {look_at: [30, 0, 10]}
`

func TestAddRecipeThingRetainsSourceAndCanUndo(t *testing.T) {
	document := poseDraft(t, addRecipeSource)
	names, err := document.RecipeNames()
	if err != nil || len(names) != 1 || names[0] != "low" {
		t.Fatalf("recipe names = %v, %v", names, err)
	}
	if err := document.AddRecipeThing("second", "low", 8, [3]float64{35, -2, 4}); err != nil {
		t.Fatal(err)
	}
	objects, err := document.Objects()
	if err != nil || len(objects) != 2 || objects[1].ID() != "scene.set.things/second" || objects[1].Recipe() != "low" || objects[1].Position() != [3]float64{35, -2, 4} {
		t.Fatalf("objects = %+v, %v", objects, err)
	}
	if !strings.Contains(document.Text(), "Keep this note.") || !strings.Contains(document.Text(), "first:") {
		t.Fatal("unrelated source was lost")
	}
	document.Replace(addRecipeSource)
	if document.Text() != addRecipeSource {
		t.Fatal("undo did not restore exact text")
	}
}

func TestAddRecipeThingRefusesWithoutChangingText(t *testing.T) {
	for _, test := range []struct {
		name, recipe string
		at           [3]float64
	}{
		{"first", "low", [3]float64{}},
		{"new", "absent", [3]float64{}},
		{"", "low", [3]float64{}},
		{"new", "low", [3]float64{math.Inf(1), 0, 0}},
	} {
		document := poseDraft(t, addRecipeSource)
		if err := document.AddRecipeThing(test.name, test.recipe, 4, test.at); err == nil || document.Text() != addRecipeSource {
			t.Fatalf("accepted or changed source for %+v: %v", test, err)
		}
	}
}

func TestAddRecipeThingCreatesMissingThingCollection(t *testing.T) {
	source := strings.Replace(addRecipeSource, "  set:\n    things:\n      first: {recipe: low, seed: 3, complexity: quick, east: 30, north: 0, height: {value: 0, unit: m}}\n", "", 1)
	document := poseDraft(t, source)
	if err := document.AddRecipeThing("first", "low", 3, [3]float64{}); err != nil {
		t.Fatal(err)
	}
	objects, err := document.Objects()
	if err != nil || len(objects) != 1 || objects[0].ID() != "scene.set.things/first" {
		t.Fatalf("created collection = %+v, %v", objects, err)
	}
}

func TestAddRecipeThingGroundWritesExplicitHeading(t *testing.T) {
	source := strings.Replace(addRecipeSource, "setting: space", "setting: ground", 1)
	document, err := Open("ground.yaml", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	if err := document.AddRecipeThing("invalid", "low", 4, [3]float64{10, 20, -1}); err == nil || document.Text() != source {
		t.Fatal("negative ground height changed the source")
	}
	if err := document.AddRecipeThing("second", "low", 4, [3]float64{10, 20, 2}); err != nil {
		t.Fatal(err)
	}
	objects, err := document.Objects()
	if err != nil || len(objects) != 2 || !objects[1].Grounded() {
		t.Fatalf("objects %+v: %v", objects, err)
	}
	heading, said := objects[1].Heading()
	if !said || heading != 0 || objects[1].Position() != [3]float64{10, 20, 2} {
		t.Fatalf("ground heading %v %v position %v", heading, said, objects[1].Position())
	}
}

func TestAddKindThingWritesAnEditablePlacement(t *testing.T) {
	document := poseDraft(t, addRecipeSource)
	if err := document.AddKindThing("tower-two", "tower", 7, [3]float64{20, 4, 1}); err != nil {
		t.Fatal(err)
	}
	objects, err := document.Objects()
	if err != nil || len(objects) != 2 || objects[1].Kind() != "tower" || objects[1].Recipe() != "" || objects[1].Position() != [3]float64{20, 4, 1} {
		t.Fatalf("added constructor = %+v, %v", objects, err)
	}
	if !strings.Contains(document.Text(), "Keep this note.") {
		t.Fatal("lost unrelated source")
	}
}
