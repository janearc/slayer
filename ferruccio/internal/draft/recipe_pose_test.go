package draft

import (
	"strings"
	"testing"
)

func TestRecipeThingHasEditableAuthoredPose(t *testing.T) {
	source := strings.Replace(poseSource, "        kind: mcrn-larson\n", "        recipe: low\n", 1)
	document := poseDraft(t, source)
	objects, err := document.Objects()
	if err != nil || len(objects) != 3 {
		t.Fatalf("objects = %+v, %v", objects, err)
	}
	if objects[0].ID() != "scene.set.things/craft" || objects[0].Kind() != "recipe" || objects[0].Recipe() != "low" {
		t.Fatalf("recipe identity = %+v", objects[0])
	}
	change := PoseChange{ID: objects[0].ID(), East: 2, North: -3, Heading: 5, LengthUnit: "m", AngleUnit: "deg"}
	if err := document.ChangePose(change); err != nil {
		t.Fatal(err)
	}
	again, err := document.Objects()
	if err != nil || again[0].Position() != [3]float64{32, 37, 50} || again[0].Recipe() != "low" {
		t.Fatalf("moved recipe = %+v %v", again, err)
	}
	if !strings.Contains(document.Text(), "Keep the east note") {
		t.Fatal("placement comment was lost")
	}
	document.Replace(source)
	if document.Text() != source {
		t.Fatal("undo did not restore source")
	}
}

func TestRecipeThingStillRefusesRelativePlacement(t *testing.T) {
	source := strings.Replace(poseSource, "        kind: mcrn-larson\n", "        recipe: low\n", 1)
	source = strings.Replace(source, "        east: 30 # Keep the east note.\n        north: 40", "        place: {at: frame}", 1)
	document := poseDraft(t, source)
	objects, err := document.Objects()
	if err != nil {
		t.Fatal(err)
	}
	for _, object := range objects {
		if object.Name() == "craft" && object.Kind() == "recipe" {
			t.Fatal("relative recipe was offered for direct pose")
		}
	}
}
