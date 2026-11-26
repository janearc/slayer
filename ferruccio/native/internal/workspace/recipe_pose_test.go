package workspace

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/janearc/slayer/ferruccio/internal/draft"
	"github.com/janearc/slayer/session/placement"
	"github.com/janearc/slayer/set"
)

func TestRecipePoseReachesConstructedDisplay(t *testing.T) {
	const source = `$schema: https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: Recipe pose
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
	document, err := draft.Open("recipe.yaml", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	objects, err := document.Objects()
	if err != nil || len(objects) != 1 {
		t.Fatalf("objects = %v, %v", objects, err)
	}
	if err := document.ChangePose(draft.PoseChange{ID: objects[0].ID(), East: 5, LengthUnit: "m", AngleUnit: "deg"}); err != nil {
		t.Fatal(err)
	}
	read, err := set.Read("recipe.yaml", []byte(document.Text()))
	if err != nil {
		t.Fatal(err)
	}
	scene, err := placement.NewDisplayCache().Scene(read)
	if err != nil {
		t.Fatal(err)
	}
	for _, object := range scene.Objects {
		if object.ID == objects[0].ID() {
			if object.Mesh == nil || len(object.Mesh.Vertices) == 0 || object.At.Origin.X != 35 {
				t.Fatalf("constructed recipe = %+v", object)
			}
			return
		}
	}
	t.Fatal("edited recipe was absent from the display")
}

func TestAddedRecipeReachesConstructedDisplay(t *testing.T) {
	const source = `$schema: https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: Recipe addition
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
	document, err := draft.Open("recipe.yaml", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	if err := document.AddRecipeThing("second", "low", 8, [3]float64{35, -2, 4}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "recipe.yaml")
	if err := document.Save(path); err != nil {
		t.Fatal(err)
	}
	saved, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	read, err := set.Read(path, saved)
	if err != nil {
		t.Fatal(err)
	}
	scene, err := placement.NewDisplayCache().Scene(read)
	if err != nil {
		t.Fatal(err)
	}
	for _, object := range scene.Objects {
		if object.ID == "scene.set.things/second" {
			if object.Mesh == nil || len(object.Mesh.Vertices) == 0 || object.At.Origin.X != 35 || object.At.Origin.Y != 4 || object.At.Origin.Z != 2 {
				t.Fatalf("added recipe = %+v", object)
			}
			return
		}
	}
	t.Fatal("added recipe was absent from the display")
}

func TestAddedConstructorSurvivesSaveAndReachesDisplay(t *testing.T) {
	document, err := draft.Open("scene.yaml", []byte(addRecipeFixture))
	if err != nil {
		t.Fatal(err)
	}
	if err := document.AddKindThing("house", "city-rowhouse", 7, [3]float64{20, 4, 1}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "scene.yaml")
	if err := document.Save(path); err != nil {
		t.Fatal(err)
	}
	saved, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	read, err := set.Read(path, saved)
	if err != nil {
		t.Fatal(err)
	}
	scene, err := placement.NewDisplayCache().Scene(read)
	if err != nil {
		t.Fatal(err)
	}
	for _, object := range scene.Objects {
		if object.ID == "scene.set.things/house" {
			if object.Mesh == nil || len(object.Mesh.Vertices) == 0 || object.At.Origin.X != 20 || object.At.Origin.Y != 1 || object.At.Origin.Z != -4 {
				t.Fatalf("constructed house = %+v", object)
			}
			return
		}
	}
	t.Fatal("saved constructor placement was not displayed")
}
