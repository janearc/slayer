package draft

import (
	"math"
	"os"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestThingScaleChangesOnlySelectedPlacementAndCanUndo(t *testing.T) {
	document := poseDraft(t, poseSource)
	objects, err := document.Objects()
	if err != nil || len(objects) != 3 {
		t.Fatal(err)
	}
	value, has := objects[0].SizeScale()
	if value != 1 || has || !objects[0].ScaleEditable() || objects[2].ScaleEditable() {
		t.Fatalf("scale metadata = %v %v %+v", value, has, objects)
	}
	if err := document.SetThingScale(objects[0].ID(), 1); err != nil || document.Text() != poseSource {
		t.Fatal("default scale no-op changed the source")
	}
	if err := document.SetThingScale(objects[0].ID(), 2); err != nil {
		t.Fatal(err)
	}
	again, err := document.Objects()
	if err != nil {
		t.Fatal(err)
	}
	value, has = again[0].SizeScale()
	if value != 2 || !has || again[1] != objects[1] || !strings.Contains(document.Text(), "Keep the east note.") {
		t.Fatal("scale edit lost metadata or changed another object")
	}
	document.Replace(poseSource)
	if document.Text() != poseSource {
		t.Fatal("undo did not restore source")
	}
}

func TestThingScaleRefusesInvalidValuesAndUnsupportedObjects(t *testing.T) {
	document := poseDraft(t, poseSource)
	for _, value := range []float64{0, -1, .009, 101, math.NaN(), math.Inf(1)} {
		if err := document.SetThingScale("scene.set.things/craft", value); err == nil || document.Text() != poseSource {
			t.Fatalf("accepted invalid scale %v", value)
		}
	}
	if err := document.SetThingScale("scene.world.imports/craft", 2); err == nil || document.Text() != poseSource {
		t.Fatal("treated an import as a constructor")
	}
	spinning := strings.Replace(poseSource, "        seed: 7\n", "        seed: 7\n        part-spin: {material: blade, pivot-metres: [0, 0, 0], axis: [0, 0, 1], rpm: 2}\n", 1)
	spin := poseDraft(t, spinning)
	objects, _ := spin.Objects()
	if objects[0].ScaleEditable() || spin.SetThingScale(objects[0].ID(), 2) == nil || spin.Text() != spinning {
		t.Fatal("accepted scale with part-spin")
	}
}

func TestTypedFireScaleUsesAuthoredKindEvenWithRecipe(t *testing.T) {
	source := strings.Replace(poseSource, "kind: mcrn-larson", "kind: fire\n        recipe: bonfire", 1)
	source = strings.Replace(source, "        east: 30 # Keep the east note.\n        north: 40\n        height: {value: 50, unit: m}\n",
		"        position:\n          east: {value: 30, unit: m}\n          north: {value: 40, unit: m}\n          up: {value: 50, unit: m}\n", 1)
	var tree yaml.Node
	if err := yaml.Unmarshal([]byte(source), &tree); err != nil {
		t.Fatal(err)
	}
	document := poseDraft(t, poseSource)
	_, poses, err := document.posesIn(&tree)
	if err != nil {
		t.Fatal(err)
	}
	if len(poses) == 0 || poses[0].object.ID() != "scene.set.things/craft" || poses[0].object.ScaleEditable() {
		t.Fatal("typed fire offered a size factor when it also named a recipe")
	}
	document.Replace(source)
	if err := document.SetThingScale("scene.set.things/craft", 2); err == nil || document.Text() != source {
		t.Fatal("typed fire accepted a size edit")
	}
}

func TestTypedFireRecipeRefusesDraftScale(t *testing.T) {
	raw, err := os.ReadFile("../../../doc/fire-study.yaml")
	if err != nil {
		t.Fatal(err)
	}
	source := strings.Replace(string(raw), "ecosystem-v1.json", "ecosystem-v2.json", 1)
	start, end := strings.Index(source, "  terrain:"), strings.Index(source, "  recipes:")
	source = source[:start] + "  setting: space\n" + source[end:]
	source = strings.Replace(source, "        east: 263812\n        north: 2148748\n",
		"        position:\n          east: {value: 0, unit: m}\n          north: {value: 0, unit: m}\n          up: {value: 0, unit: m}\n", 1)
	document := poseDraft(t, source)
	objects, err := document.Objects()
	if err != nil {
		t.Fatal(err)
	}
	if len(objects) != 1 || objects[0].ScaleEditable() {
		t.Fatal("typed fire recipe offered a size factor")
	}
	if err := document.SetThingScale(objects[0].ID(), 2); err == nil || document.Text() != source {
		t.Fatal("typed fire recipe accepted a size edit")
	}
}
