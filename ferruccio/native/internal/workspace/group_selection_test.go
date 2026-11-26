package workspace

import (
	"strings"
	"testing"

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
