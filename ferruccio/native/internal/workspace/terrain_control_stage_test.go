package workspace

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/gogpu/ui/core/textfield"
	"github.com/gogpu/ui/geometry"
	"github.com/janearc/slayer/ferruccio/internal/draft"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/session/placement"
	"github.com/janearc/slayer/set"
)

func TestGroundCoordinateControlAcceptsSecondEditBeforeTheNextTick(t *testing.T) {
	work := libraryEventWorkspace(t)
	work.ui.Window().HandleResize(1280, 800)
	path := filepath.Join(t.TempDir(), "ground.yaml")
	document, err := draft.Open(path, []byte(groundEditFixture))
	if err != nil {
		t.Fatal(err)
	}
	checked, err := set.Read(path, []byte(groundEditFixture))
	if err != nil {
		t.Fatal(err)
	}
	const object = "scene.set.things/stone"
	work.state = documentState{document: document, path: path}
	work.editor.SetText(groundEditFixture)
	work.play.selected = object
	work.working.source, work.working.path = groundEditFixture, path
	work.working.parsed, work.working.displayed = &checked, true
	work.working.scene = &placement.DisplayScene{
		Origins: map[string]maths.Vec{object: maths.V(1050, 0, -8990)},
		Batches: []placement.StagedBatch{{
			Authored: true, Author: thing.AuthorRef{Collection: "scene.set.things", Name: "stone"},
			Places: []maths.Subspace{maths.Local()},
		}},
	}
	started := make(chan string, 1)
	work.working.stageOverride = func(ctx context.Context, _, source, _ string) (*stagedView, error) {
		started <- source
		<-ctx.Done()
		return nil, ctx.Err()
	}
	defer work.Close()
	work.toggleDisclosure("coordinates")
	drawLibraryWindow(t, work)

	east, ok := libraryWidget(work.root, "East (m)").(*textfield.Widget)
	control := libraryWidget(work.root, "Set coordinates")
	if !ok || control == nil {
		t.Fatal("coordinate controls missing")
	}
	bounds := control.(interface{ ScreenBounds() geometry.Rect }).ScreenBounds()
	click := geometry.Pt((bounds.Min.X+bounds.Max.X)/2, (bounds.Min.Y+bounds.Max.Y)/2)
	east.SetText("1052")
	clickLibraryPoint(work, click)
	if selected, _ := work.selectedObject(); selected.Position()[0] != 1052 {
		t.Fatal("first coordinate control did not move the object")
	}
	work.updateWorking()
	waitForStageSource(t, started, document.Text())
	if !work.working.staging || work.working.parsed != nil {
		t.Fatal("test did not leave the new metadata queued before the next tick")
	}

	east.SetText("1054")
	clickLibraryPoint(work, click)
	if selected, _ := work.selectedObject(); selected.Position()[0] != 1054 {
		t.Fatalf("second coordinate control left the object at %g: %s",
			selected.Position()[0], work.diagnostic)
	}
}
