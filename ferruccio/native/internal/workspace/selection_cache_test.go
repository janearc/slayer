package workspace

import (
	"os"
	"strings"
	"testing"

	"github.com/janearc/slayer/ferruccio/internal/draft"
)

func TestSelectionCacheTracksSourceAndDocument(t *testing.T) {
	work := libraryEventWorkspace(t)
	document, err := draft.Open("first.yaml", []byte(addRecipeFixture))
	if err != nil {
		t.Fatal(err)
	}
	work.state.document = document
	work.play.selected = "scene.set.things/first"
	first, ok := work.selectedObject()
	if !ok || first.Position()[0] != 30 {
		t.Fatal("initial selection missing")
	}
	if err := document.ChangePose(draft.PoseChange{ID: first.ID(), East: 5, LengthUnit: "m", AngleUnit: "deg"}); err != nil {
		t.Fatal(err)
	}
	moved, ok := work.selectedObject()
	if !ok || moved.Position()[0] != 35 {
		t.Fatal("source edit left stale selection")
	}
	document.Replace("not: [valid")
	if _, ok := work.selectedObject(); ok {
		t.Fatal("invalid source reused old selection")
	}
	other, err := draft.Open("other.yaml", []byte(addRecipeFixture))
	if err != nil {
		t.Fatal(err)
	}
	work.state.document = other
	restored, ok := work.selectedObject()
	if !ok || restored.Position()[0] != 30 {
		t.Fatal("new document reused old selection")
	}
}

func TestSelectionCacheTracksGroupRegionEligibility(t *testing.T) {
	source := strings.Replace(checkingSource, "setting: space", "setting: ground", 1)
	source = strings.Replace(source, "  film:", "    stands:\n      grove: {kind: yurt, count: 4, seed: 2, habit: scatter, area: {west: 20, south: 20, east: 80, north: 80}}\n  film:", 1)
	doc, err := draft.Open("scene.yaml", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	work := libraryEventWorkspace(t)
	work.state.document = doc
	work.play.selected = "scene.set.stands/grove"
	if _, ok := work.selectedGroup(); !ok || !work.play.selectionRegion {
		t.Fatal("editable region missing")
	}
	doc.Replace(strings.Replace(source, "habit: scatter, area: {west: 20, south: 20, east: 80, north: 80}", "within: {of: visitor, from: 0, to: 90, near: 1, far: 10}", 1))
	if _, ok := work.selectedGroup(); !ok || work.play.selectionRegion {
		t.Fatal("old region eligibility survived source change")
	}
}

func TestFutianDefaultTerrainSelection(t *testing.T) {
	path := os.Getenv("FERRUCCIO_FUTIAN_SOURCE")
	if path == "" {
		t.Skip("set FERRUCCIO_FUTIAN_SOURCE to check the real set")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	document, err := draft.Open(path, raw)
	if err != nil {
		t.Fatal(err)
	}
	work := libraryEventWorkspace(t)
	work.state.document = document
	work.play.selected = "scene.set.things/tower 006"
	object, ok := work.selectedObject()
	if !ok || !object.Grounded() || !object.ScaleEditable() {
		t.Fatalf("Futian tower selection %+v, %v", object, ok)
	}
	if err := document.ChangePose(draft.PoseChange{ID: object.ID(), East: 1, LengthUnit: "m", AngleUnit: "deg"}); err != nil {
		t.Fatal(err)
	}
	moved, ok := work.selectedObject()
	if !ok || moved.Position()[0] != object.Position()[0]+1 {
		t.Fatal("Futian tower edit did not update selected source")
	}
	if err := document.SetThingScale(object.ID(), 2); err != nil {
		t.Fatal(err)
	}
	if output := os.Getenv("FERRUCCIO_FUTIAN_EDITED_COPY"); output != "" {
		if err := os.WriteFile(output, []byte(document.Text()), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if scale, ok := document.GroupArea("scene.set.stands/does-not-exist"); ok {
		t.Fatalf("invented region %v", scale)
	}
}
