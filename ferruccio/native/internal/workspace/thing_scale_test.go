package workspace

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gogpu/ui/core/textfield"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/janearc/slayer/ferruccio/internal/draft"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/set"
)

func TestThingSizePanelChangesAndUndoesFactorWithoutRendering(t *testing.T) {
	work := libraryEventWorkspace(t)
	document, err := draft.Open("test.yaml", []byte(checkingSource))
	if err != nil {
		t.Fatal(err)
	}
	work.state.document = document
	work.editor.SetText(checkingSource)
	work.play.selected = "scene.set.things/visitor"
	work.updateWorking()
	work.rebuild()
	if libraryWidget(work.root, "Size…") == nil {
		t.Fatal("supported object has no size control")
	}
	work.toggleDisclosure("size")
	drawLibraryWindow(t, work)
	field, ok := libraryWidget(work.root, "Uniform size factor").(*textfield.Widget)
	if !ok || field.Text() != "1" {
		t.Fatalf("size field = %v", field)
	}
	field.SetText("2")
	work.ui.Window().Context().RequestFocus(field)
	work.ui.HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyEnter, 0, event.ModNone))
	if got, _ := document.Value("scene", "set", "things", "visitor", "scale"); got != "2" || work.pending.document != nil || work.Running() {
		t.Fatalf("size was not edited without rendering: %q", got)
	}
	work.undoCamera(work.ui.Window().Context(), false)
	if document.Text() != checkingSource || work.pending.document != nil {
		t.Fatal("size undo did not restore exact source")
	}
}

func TestThingSizeFieldReplacesItsValueWithCommandA(t *testing.T) {
	work := libraryEventWorkspace(t)
	document, err := draft.Open("test.yaml", []byte(checkingSource))
	if err != nil {
		t.Fatal(err)
	}
	work.state.document = document
	work.editor.SetText(checkingSource)
	work.play.selected = "scene.set.things/visitor"
	work.updateWorking()
	work.rebuild()
	work.toggleDisclosure("size")
	drawLibraryWindow(t, work)
	field := libraryWidget(work.root, "Uniform size factor").(*textfield.Widget)
	work.ui.Window().Context().RequestFocus(field)
	work.ui.HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyA, 0, event.ModSuper))
	for _, key := range []struct {
		code event.Key
		text rune
	}{{event.Key1, '1'}, {event.KeyPeriod, '.'}, {event.Key5, '5'}} {
		work.ui.HandleEvent(event.NewKeyEvent(event.KeyPress, key.code, key.text, event.ModNone))
	}
	if got := field.Text(); got != "1.5" {
		t.Fatalf("typed size = %q, want 1.5", got)
	}
	work.ui.HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyEnter, 0, event.ModNone))
	if got, _ := document.Value("scene", "set", "things", "visitor", "scale"); got != "1.5" {
		t.Fatalf("saved size = %q, want 1.5", got)
	}
}

func TestDeformingProfileIsNotOfferedUniformScale(t *testing.T) {
	work := libraryEventWorkspace(t)
	_, raw, err := manufacture.EmbeddedAnimalBook("whale")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "whale.yaml"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	source := strings.Replace(checkingSource, "  set:\n", "  recipes:\n    orca: {animal: {family: whale, file: whale.yaml, species: orca, pose: swim}}\n  set:\n", 1)
	source = strings.Replace(source, "kind: human-scale-2m", "recipe: orca", 1)
	path := filepath.Join(dir, "scene.yaml")
	document, err := draft.Open(path, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	read, err := set.Read(path, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	work.state.document, work.state.path = document, path
	work.working.source, work.working.path, work.working.parsed = source, path, &read
	work.play.selected = "scene.set.things/visitor"
	if work.canScaleThing() {
		t.Fatal("deforming profile was offered a size control")
	}
	if err := work.changeThingScale(2); err == nil || document.Text() != source {
		t.Fatal("deforming profile accepted a scale edit")
	}
}

func TestRepeatedSizeStepsDoNotWaitForRestaging(t *testing.T) {
	work := libraryEventWorkspace(t)
	document, err := draft.Open("test.yaml", []byte(checkingSource))
	if err != nil {
		t.Fatal(err)
	}
	read, err := set.Read("test.yaml", []byte(checkingSource))
	if err != nil {
		t.Fatal(err)
	}
	work.state.document = document
	work.editor.SetText(checkingSource)
	work.play.selected = "scene.set.things/visitor"
	work.working.parsed, work.working.source, work.working.path = &read, checkingSource, work.state.path
	if err := work.stepThingScale(1.1); err != nil {
		t.Fatal(err)
	}
	first := document.Text()
	path := work.state.path
	work.state.path = filepath.Join(t.TempDir(), "different.yaml")
	if work.canScaleThing() {
		t.Fatal("new document path reused old scale eligibility")
	}
	work.state.path = path
	if err := work.stepThingScale(1.1); err != nil {
		t.Fatalf("second step waited for staging: %v", err)
	}
	object, ok := work.selectedObject()
	if !ok {
		t.Fatal("selection lost")
	}
	factor, _ := object.SizeScale()
	if math.Abs(factor-1.21) > 1e-12 {
		t.Fatalf("factor %v", factor)
	}
	work.undoCamera(work.ui.Window().Context(), false)
	if document.Text() != first {
		t.Fatal("one undo did not restore the previous step")
	}
	changed := strings.Replace(document.Text(), "kind: human-scale-2m", "kind: yurt", 1)
	document.Replace(changed)
	if work.canScaleThing() {
		t.Fatal("changed constructor reused stale scale eligibility")
	}
}

func TestSizePanelStepsAtSmallestWindow(t *testing.T) {
	work := libraryEventWorkspace(t)
	work.ui.Window().HandleResize(1280, 800)
	document, err := draft.Open("test.yaml", []byte(checkingSource))
	if err != nil {
		t.Fatal(err)
	}
	read, err := set.Read("test.yaml", []byte(checkingSource))
	if err != nil {
		t.Fatal(err)
	}
	work.state.document = document
	work.editor.SetText(checkingSource)
	work.play.selected = "scene.set.things/visitor"
	work.working.parsed, work.working.source, work.working.path = &read, checkingSource, work.state.path
	work.toggleDisclosure("size")
	drawLibraryWindow(t, work)
	control := libraryWidget(work.root, "+ 10%")
	if control == nil {
		t.Fatal("larger control missing")
	}
	bounds := control.(interface{ ScreenBounds() geometry.Rect }).ScreenBounds()
	if bounds.Min.X < 0 || bounds.Max.X > 1280 || bounds.Max.Y > 800 {
		t.Fatalf("size control outside window: %v", bounds)
	}
	clickLibraryPoint(work, geometry.Pt((bounds.Min.X+bounds.Max.X)/2, (bounds.Min.Y+bounds.Max.Y)/2))
	object, ok := work.selectedObject()
	if !ok {
		t.Fatal("selection lost")
	}
	factor, _ := object.SizeScale()
	if math.Abs(factor-1.1) > 1e-12 {
		t.Fatalf("factor %v", factor)
	}
}
