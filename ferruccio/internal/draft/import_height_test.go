package draft

import (
	"math"
	"strings"
	"testing"
)

func TestSetImportHeightRejectsBadInputsAndPreservesNoop(t *testing.T) {
	source := strings.Replace(poseSource, "        file: model.gltf\n", "        file: model.gltf\n        height: {value: 2, unit: m}\n", 1)
	document := poseDraft(t, source)
	if err := document.SetImportHeight("scene.world.imports/craft", 2); err != nil || document.Text() != source {
		t.Fatalf("no-op changed draft: %v", err)
	}
	for _, n := range []float64{0, -1, math.NaN(), math.Inf(1), 1000001} {
		if err := document.SetImportHeight("scene.world.imports/craft", n); err == nil || document.Text() != source {
			t.Fatalf("accepted height %v: %v", n, err)
		}
	}
	if err := document.SetImportHeight("scene.set.things/craft", 3); err == nil || document.Text() != source {
		t.Fatal("scaled a non-import")
	}
	if err := document.SetImportHeight("scene.world.imports/missing", 3); err == nil || document.Text() != source {
		t.Fatal("accepted missing import")
	}
}

func TestSetImportHeightAddsMissingHeight(t *testing.T) {
	document := poseDraft(t, poseSource)
	objects, err := document.Objects()
	if err != nil {
		t.Fatal(err)
	}
	if _, has := objects[2].ImportHeight(); has {
		t.Fatal("an omitted import height was invented")
	}
	if err := document.SetImportHeight("scene.world.imports/craft", 4); err != nil {
		t.Fatal(err)
	}
	got, err := document.Value("scene", "world", "imports", "craft", "height", "value")
	if err != nil || got != "4" {
		t.Fatalf("new height = %q, %v", got, err)
	}
	objects, err = document.Objects()
	if err != nil {
		t.Fatal(err)
	}
	if height, has := objects[2].ImportHeight(); !has || height != 4 {
		t.Fatalf("authored height = %v, %v", height, has)
	}
}
