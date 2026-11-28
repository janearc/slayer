package workspace

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gogpu/ui/event"

	"github.com/janearc/slayer/ferruccio/internal/draft"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/session/placement"
)

func TestGeneratedMemberSelectsReadOnlyGroup(t *testing.T) {
	source := strings.Replace(addRecipeFixture, "  film:", "    stands:\n      grove: {kind: cobble, count: 3, seed: 4}\n  film:", 1)
	document, err := draft.Open("scene.yaml", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	work := playWorkspace(t)
	work.state.document = document
	mesh := &placement.DisplayMesh{Vertices: []placement.DisplayVertex{{Position: maths.V(-1, -1, 0)}, {Position: maths.V(1, -1, 0)}, {Position: maths.V(0, 1, 0)}}, Indices: []uint32{0, 1, 2}}
	scene := &placement.DisplayScene{Width: 100, Height: 100, Lens: film.Pinhole{}, Batches: []placement.StagedBatch{{Mesh: mesh, Places: []maths.Subspace{{Origin: maths.V(0, 0, -5)}}, Author: thing.AuthorRef{Collection: "scene.set.stands", Name: "grove"}, Authored: true}}}
	work.working.scene, work.working.source, work.working.path, work.working.displayed = scene, source, work.state.path, true
	if !work.viewer.selectObject(.5, .5) || work.play.selected != "scene.set.stands/grove" {
		t.Fatalf("member selection = %q", work.play.selected)
	}
	if !strings.Contains(work.playReadout(), "generated group · read-only") {
		t.Fatalf("group identity not explained: %q", work.playReadout())
	}
	_, cancel := context.WithCancel(context.Background())
	work.cancel = cancel
	work.queueObserved, work.queueState, work.queueHeld = true, "queued", "No recent queue worker activity · check the queue worker"
	work.working.notice = "generated members are not individually editable"
	work.diagnostic = work.working.notice
	if readout := work.playReadout(); !strings.Contains(readout, "Queued") || !strings.Contains(readout, "check the queue worker") {
		t.Fatalf("selected group hid queued render status: %q", readout)
	}
	cancel()
	work.cancel = nil
	work.diagnostic = ""
	work.viewer.dragObject = true
	work.beginWorking()
	work.viewer.changeObject(.5, .5, .6, .6)
	if work.working.gesture != nil || document.Text() != source {
		t.Fatal("a generated member was moved as an independent object")
	}
	if err := work.changePose(1, 0, 0); err == nil {
		t.Fatal("keyboard movement changed a read-only group")
	}
}

// Move mode shifts the authored rectangle as one edit, including undo and redo.
func TestGroupRegionMovesThroughWindow(t *testing.T) {
	source := strings.Replace(addRecipeFixture, "setting: space", "setting: ground", 1)
	source = strings.Replace(source, "  film:", `    stands:
      grove:
        kind: yurt
        count: 3
        seed: 4
        habit: scatter
        area: {west: 0x10, south: 0o2, east: 3.0e1, north: 0b101000} # Keep region note.
  film:`, 1)
	document, err := draft.Open("scene.yaml", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	work := playWorkspace(t)
	work.state.document = document
	work.editor.SetText(source)
	id := "scene.set.stands/grove"
	work.play.selected = id
	if readout := work.playReadout(); !strings.Contains(readout, "region editable") {
		t.Fatalf("numeric region was not selectable: %q", readout)
	}
	work.setPlayMode("move")
	focusPlayViewer(work)
	work.working.dirty = false
	work.ui.HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyRight, 0, event.ModShift))
	changed := document.Text()
	if got, ok := document.GroupArea(id); !ok || got != [4]float64{17, 2, 31, 40} {
		t.Fatalf("shifted area = %v, %t", got, ok)
	}
	if work.editor.Text() != changed || !work.working.dirty {
		t.Fatal("move did not synchronize source and trigger staging")
	}
	if !strings.Contains(changed, "# Keep region note.") {
		t.Fatal("move lost the region comment")
	}
	work.ui.HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyZ, 'z', event.ModSuper))
	if document.Text() != source {
		t.Fatal("one undo did not restore the entire region")
	}
	work.ui.HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyZ, 'Z', event.ModSuper|event.ModShift))
	if document.Text() != changed {
		t.Fatal("redo did not restore the region move")
	}
	path := filepath.Join(t.TempDir(), "moved.yaml")
	if err := document.Save(path); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	reopened, err := draft.Open(path, raw)
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := reopened.GroupArea(id); !ok || got != [4]float64{17, 2, 31, 40} {
		t.Fatalf("saved area = %v, %t", got, ok)
	}
	work.setPlayMode("turn")
	focusPlayViewer(work)
	work.ui.HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyRight, 0, event.ModNone))
	if document.Text() != changed {
		t.Fatal("turn edited the group region")
	}
}
