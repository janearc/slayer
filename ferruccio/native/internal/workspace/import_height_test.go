package workspace

import (
	"testing"

	"github.com/gogpu/ui/core/textfield"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/janearc/slayer/ferruccio/internal/draft"
)

const importHeightFixture = `"$schema": https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: size test
  film: {width: 32, height: 24, rays: 1, wide: 45, exposure: 1}
  cameras:
    view: {location: {at: [0, -5, 2]}, orientation: {look_at: [0, 0, 1]}}
  world:
    imports:
      visitor:
        file: model.gltf
        height: {value: 2, unit: m}
        at: [0, 0, 0]
        materials: {slot: matte}
        source: eye
        license: CC0-1.0
`

func TestChangeImportHeightIsUndoableAndDoesNotQueueRender(t *testing.T) {
	work := libraryEventWorkspace(t)
	source := importHeightFixture
	doc, err := draft.Open("test.yaml", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	if err := doc.Check(); err != nil {
		t.Fatal(err)
	}
	work.state.document = doc
	work.editor.SetText(source)
	work.play.selected = "scene.world.imports/visitor"
	if err := work.changeImportHeight(3); err != nil {
		t.Fatal(err)
	}
	if got, _ := doc.Value("scene", "world", "imports", "visitor", "height", "value"); got != "3" {
		t.Fatalf("height = %q", got)
	}
	if work.pending.document != nil || work.Running() {
		t.Fatal("size edit started or queued a render")
	}
	work.undoCamera(work.ui.Window().Context(), false)
	if doc.Text() != source {
		t.Fatal("undo did not restore exact source")
	}
	if err := work.changeImportHeight(-1); err == nil || doc.Text() != source {
		t.Fatal("invalid edit changed draft")
	}
}

func TestImportHeightPanelSubmitsFromKeyboard(t *testing.T) {
	work := libraryEventWorkspace(t)
	doc, err := draft.Open("test.yaml", []byte(importHeightFixture))
	if err != nil {
		t.Fatal(err)
	}
	work.state.document = doc
	work.editor.SetText(importHeightFixture)
	work.play.selected = "scene.world.imports/visitor"
	work.rebuild()
	drawLibraryWindow(t, work)
	control := libraryWidget(work.root, "Size…")
	if control == nil {
		t.Fatal("selected import has no size control")
	}
	bounds := control.(interface{ ScreenBounds() geometry.Rect }).ScreenBounds()
	clickLibraryPoint(work, geometry.Pt((bounds.Min.X+bounds.Max.X)/2, (bounds.Min.Y+bounds.Max.Y)/2))
	drawLibraryWindow(t, work)
	field, ok := libraryWidget(work.root, "Model height in metres").(*textfield.Widget)
	if !ok || field.Text() != "2" {
		t.Fatalf("height field = %v", field)
	}
	field.SetText("3")
	work.ui.Window().Context().RequestFocus(field)
	work.ui.HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyEnter, 0, event.ModNone))
	if got, _ := doc.Value("scene", "world", "imports", "visitor", "height", "value"); got != "3" {
		t.Fatalf("keyboard submission left height %q", got)
	}
	if work.pending.document != nil || work.Running() {
		t.Fatal("size submission requested an offline render")
	}
	work.undoCamera(work.ui.Window().Context(), false)
	if doc.Text() != importHeightFixture || work.pending.document != nil {
		t.Fatal("undo did not restore the source without requesting a render")
	}
	work.rebuild()
	restored, ok := libraryWidget(work.root, "Model height in metres").(*textfield.Widget)
	if !ok || restored.Text() != "2" {
		t.Fatalf("undo left a stale height field: %v", restored)
	}
}

func TestThingSizeControlRequiresCurrentCheckedScene(t *testing.T) {
	work := libraryEventWorkspace(t)
	objects, err := work.state.document.Objects()
	if err != nil || len(objects) == 0 {
		t.Fatalf("starter objects: %v", err)
	}
	work.play.selected = objects[0].ID()
	work.rebuild()
	if libraryWidget(work.root, "Size…") != nil {
		t.Fatal("size control was offered before the constructor was checked")
	}
}

func TestHeightPanelRejectsInvalidTextWithoutChangingDraft(t *testing.T) {
	work := libraryEventWorkspace(t)
	doc, err := draft.Open("test.yaml", []byte(importHeightFixture))
	if err != nil {
		t.Fatal(err)
	}
	work.state.document = doc
	work.editor.SetText(importHeightFixture)
	work.play.selected = "scene.world.imports/visitor"
	for _, value := range []string{"", "words", "NaN", "0", "-2"} {
		if err := work.submitImportHeight(value); err == nil || doc.Text() != importHeightFixture {
			t.Fatalf("invalid input %q changed the source: %v", value, err)
		}
	}
}
