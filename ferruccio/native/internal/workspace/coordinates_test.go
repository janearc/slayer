package workspace

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/gogpu/ui/core/textfield"
	"github.com/gogpu/ui/geometry"
	"github.com/janearc/slayer/ferruccio/internal/draft"
	"github.com/janearc/slayer/set"
)

func TestCoordinatesPanelChangesThreeAxesAndHeadingInOneUndo(t *testing.T) {
	work := libraryEventWorkspace(t)
	work.ui.Window().HandleResize(1280, 800)
	document, err := draft.Open("scene.yaml", []byte(addRecipeFixture))
	if err != nil {
		t.Fatal(err)
	}
	work.state.document = document
	work.editor.SetText(addRecipeFixture)
	work.play.selected = "scene.set.things/first"
	work.rebuild()
	drawLibraryWindow(t, work)
	for _, id := range []string{"Add…", "Made from", "Describe · D"} {
		item := libraryWidget(work.root, id)
		if item == nil {
			t.Fatalf("missing %s", id)
		}
		b := item.(interface{ ScreenBounds() geometry.Rect }).ScreenBounds()
		if b.Min.X < 0 || b.Max.X > 1280 || b.Max.Y > 800 {
			t.Fatalf("%s outside window: %v", id, b)
		}
	}
	control := libraryWidget(work.root, "Coordinates…")
	if control == nil {
		t.Fatal("missing coordinates control")
	}
	bounds := control.(interface{ ScreenBounds() geometry.Rect }).ScreenBounds()
	if bounds.Min.X < 0 || bounds.Max.X > 1280 || bounds.Max.Y > 800 {
		t.Fatalf("coordinates control outside window: %v", bounds)
	}
	clickLibraryPoint(work, geometry.Pt((bounds.Min.X+bounds.Max.X)/2, (bounds.Min.Y+bounds.Max.Y)/2))
	drawLibraryWindow(t, work)
	for label, value := range map[string]string{"East (m)": "40", "North (m)": "12", "Up (m)": "7", "Heading (degrees)": "45"} {
		field, ok := libraryWidget(work.root, label).(*textfield.Widget)
		if !ok {
			t.Fatalf("missing %s", label)
		}
		field.SetText(value)
	}
	apply := libraryWidget(work.root, "Set coordinates")
	if apply == nil {
		t.Fatal("missing coordinate submit")
	}
	bounds = apply.(interface{ ScreenBounds() geometry.Rect }).ScreenBounds()
	if bounds.Max.X > 1280 || bounds.Max.Y > 800 {
		t.Fatalf("coordinate submit outside window: %v", bounds)
	}
	clickLibraryPoint(work, geometry.Pt((bounds.Min.X+bounds.Max.X)/2, (bounds.Min.Y+bounds.Max.Y)/2))
	objects, err := document.Objects()
	if err != nil || len(objects) != 1 {
		t.Fatal(err)
	}
	heading, said := objects[0].Heading()
	if objects[0].Position() != [3]float64{40, 12, 7} || !said || heading != 45 || work.pending.document != nil || work.Running() {
		t.Fatalf("position %v heading %v %v", objects[0].Position(), heading, said)
	}
	work.undoCamera(work.ui.Window().Context(), false)
	if document.Text() != addRecipeFixture {
		t.Fatal("coordinate edit did not undo as one change")
	}
	work.rebuild()
	field, ok := libraryWidget(work.root, "East (m)").(*textfield.Widget)
	if !ok || field.Text() != "30" {
		t.Fatal("undo left stale coordinates in the panel")
	}
}

func TestSelectedRigControlsFitSmallestWindow(t *testing.T) {
	old := "film: {width: 640, height: 400, wide: 40, rays: 1, exposure: 1}"
	rig := "film: {width: 640, rays: 1, frame: {width: 36, height: 24}, lens: {focal: 50, stop: 8, focus: first}, shutter: 0.5, glimpses: 2}"
	source := strings.Replace(addRecipeFixture, old, rig, 1)
	work := libraryEventWorkspace(t)
	work.ui.Window().HandleResize(1280, 800)
	document, err := draft.Open("rig.yaml", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	read, err := set.Read("rig.yaml", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	work.state.document = document
	work.editor.SetText(source)
	work.play.selected = "scene.set.things/first"
	work.working.parsed, work.working.source, work.working.path = &read, source, work.state.path
	work.rebuild()
	drawLibraryWindow(t, work)
	for _, id := range []string{"Coordinates…", "Size…", "Physical film…", "Add…", "Made from", "Describe · D"} {
		control := libraryWidget(work.root, id)
		if control == nil {
			t.Fatalf("missing %s", id)
		}
		bounds := control.(interface{ ScreenBounds() geometry.Rect }).ScreenBounds()
		if bounds.Min.X < 0 || bounds.Max.X > 1280 || bounds.Min.Y < 0 || bounds.Max.Y > 800 {
			t.Fatalf("%s outside window: %v", id, bounds)
		}
	}
}

func TestGroundCoordinatesKeepGeneratedHeading(t *testing.T) {
	source := strings.Replace(groundEditFixture, ", heading: {value: 30, unit: deg}", "", 1)
	path := filepath.Join(t.TempDir(), "ground.yaml")
	document, err := draft.Open(path, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	read, err := set.Read(path, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	work := libraryEventWorkspace(t)
	work.state.document, work.state.path = document, path
	work.editor.SetText(source)
	work.play.selected = "scene.set.things/stone"
	work.working.parsed, work.working.source, work.working.path = &read, source, path
	work.toggleDisclosure("coordinates")
	drawLibraryWindow(t, work)
	if libraryWidget(work.root, "Heading (degrees)") != nil {
		t.Fatal("generated heading was presented as authored")
	}
	work.play.poseText = [4]string{"1052", "8988", "2", "45"}
	if err := work.submitCoordinates(false); err != nil {
		t.Fatal(err)
	}
	objects, err := document.Objects()
	if err != nil {
		t.Fatal(err)
	}
	heading, said := objects[0].Heading()
	if said || heading != 0 || objects[0].Position() != [3]float64{1052, 8988, 2} {
		t.Fatalf("ground coordinates %v, heading %v %v", objects[0].Position(), heading, said)
	}
}

func TestGroundGeneratedHeadingCanBeExplicitlySetAndUndone(t *testing.T) {
	source := strings.Replace(groundEditFixture, ", heading: {value: 30, unit: deg}", "", 1)
	document, err := draft.Open("ground.yaml", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	work := libraryEventWorkspace(t)
	work.state.document = document
	work.editor.SetText(source)
	work.play.selected = "scene.set.things/stone"
	if err := work.submitGeneratedHeading("not an angle"); err == nil || document.Text() != source {
		t.Fatal("invalid heading changed source", err)
	}
	work.toggleDisclosure("coordinates")
	drawLibraryWindow(t, work)
	field, ok := libraryWidget(work.root, "New heading (degrees)").(*textfield.Widget)
	if !ok || field.Text() != "" {
		t.Fatal("generated heading should have an empty choice field")
	}
	field.SetText("0")
	button := libraryWidget(work.root, "Replace generated heading")
	if button == nil {
		t.Fatal("missing explicit heading control")
	}
	bounds := button.(interface{ ScreenBounds() geometry.Rect }).ScreenBounds()
	clickLibraryPoint(work, geometry.Pt((bounds.Min.X+bounds.Max.X)/2, (bounds.Min.Y+bounds.Max.Y)/2))
	objects, err := document.Objects()
	if err != nil {
		t.Fatal(err)
	}
	heading, said := objects[0].Heading()
	if !said || heading != 0 || work.Running() {
		t.Fatalf("heading %v, authored %v", heading, said)
	}
	work.undoCamera(work.ui.Window().Context(), false)
	if document.Text() != source {
		t.Fatal("undo did not restore generated heading")
	}
}
