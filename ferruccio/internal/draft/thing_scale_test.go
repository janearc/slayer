package draft

import (
	"math"
	"strings"
	"testing"
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
