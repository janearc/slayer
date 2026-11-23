package workspace

import (
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/ferruccio/internal/draft"
	"github.com/janearc/slayer/session/placement"
	"github.com/janearc/slayer/set"
)

// describingWorkspace is a workspace with a draft, and a working scene for
// it when built is set. It has no window: nothing here may draw.
func describingWorkspace(t *testing.T, built bool) *Workspace {
	t.Helper()
	document, err := draft.Open("set.yaml", []byte(checkingSource))
	if err != nil {
		t.Fatal(err)
	}
	work := &Workspace{viewer: newImageView(), composition: "play"}
	work.state = documentState{document: document, path: "/sets/set.yaml"}
	if built {
		read, err := set.Read(work.state.path, []byte(checkingSource))
		if err != nil {
			t.Fatal(err)
		}
		scene, err := placement.NewDisplayCache().Scene(read)
		if err != nil {
			t.Fatal(err)
		}
		work.working = workingState{scene: scene, source: checkingSource, path: work.state.path, displayed: true}
	}
	return work
}

func TestLookSceneIsTheWorkingSceneOnlyWhenItIsForThisDraft(t *testing.T) {
	built := describingWorkspace(t, true)
	if scene, why := built.lookScene(); scene == nil || why != "" {
		t.Fatalf("a current scene was not used: %q", why)
	}
	built.working.source = "an older draft"
	if scene, why := built.lookScene(); scene != nil || !strings.Contains(why, "not current") {
		t.Fatalf("a scene for another draft was used: %v %q", scene, why)
	}
	unbuilt := describingWorkspace(t, false)
	if scene, why := unbuilt.lookScene(); scene != nil || !strings.Contains(why, "not been built") {
		t.Fatalf("no scene was invented: %v %q", scene, why)
	}
	refused := describingWorkspace(t, true)
	refused.working.failure = "working display needs an authored film size"
	if scene, why := refused.lookScene(); scene != nil || !strings.Contains(why, "authored film size") {
		t.Fatalf("a refusal was not passed on: %v %q", scene, why)
	}
}

func TestLookPointerIsUnknownUntilThePointerHasBeenOverThePicture(t *testing.T) {
	work := describingWorkspace(t, true)
	if at, why := work.lookPointer(); at != nil || !strings.Contains(why, "not been over") {
		t.Fatalf("a pointer that was never over the picture had a place: %v %q", at, why)
	}
}

func TestLookInputHoldsTheDraftTheDisplayAndTheSelection(t *testing.T) {
	work := describingWorkspace(t, true)
	objects, err := work.state.document.Objects()
	if err != nil || len(objects) == 0 {
		t.Fatal("the draft has no placements", err)
	}
	work.play.selected = objects[0].ID()
	input := work.lookInput(time.Unix(0, 0))
	if input.Text != checkingSource || input.File != "/sets/set.yaml" || input.Composition != "play" {
		t.Fatalf("the draft was not carried: %+v", input)
	}
	if input.Display != "rendered" || input.SelectedID != objects[0].ID() || input.Scene == nil || len(input.Things) == 0 {
		t.Fatalf("the window's state was not carried: %+v", input)
	}
	work.working.mode = "wire"
	if got := work.lookInput(time.Unix(0, 0)).Display; got != "wire" {
		t.Fatalf("the display was %q", got)
	}
}

func TestWriteLookRefusesADraftWithNoFile(t *testing.T) {
	work := describingWorkspace(t, true)
	work.state.path = ""
	if _, err := work.WriteLook(); err == nil {
		t.Fatal("a look was written for a draft with no file")
	}
}
