package workspace

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/ferruccio/internal/draft"
)

func TestBorrowedCameraAllowsWorkingInspectionWithoutChangingRender(t *testing.T) {
	work := travelWorkspace(t)
	work.working.cameraName, work.working.cameraWide = "bay", 42
	original := work.working.scene.Eye
	source := work.state.document.Text()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	work.cancel = cancel
	for _, step := range []func() error{
		work.ProofTrackballPress, work.ProofTrackballMove, work.ProofTrackballRelease,
	} {
		if err := step(); err != nil {
			t.Fatal(err)
		}
	}
	if work.working.scene.Eye == original || !strings.Contains(work.workingCaption(), "authored camera") {
		t.Fatal("borrowed camera inspection did not remain visible")
	}
	if ctx.Err() != nil {
		t.Fatal("inspection cancelled the already running preview")
	}
	profile := work.fullRenderProfile(nil)
	if profile.Camera != "bay" || profile.CameraWide != 42 || work.state.document.Text() != source {
		t.Fatal("inspection changed borrowed camera or authored source")
	}
	work.viewer.dragObject = true
	work.beginWorking()
	if ctx.Err() == nil {
		t.Fatal("inspection changed normal object gesture cancellation")
	}
	work.rememberWorking(work.working.gesture)
	if !work.restoreWorking(source) || work.working.scene.Eye != original ||
		work.working.inspection != nil {
		t.Fatal("placement history retained the temporary inspection camera")
	}
}

func TestJobWorkingCameraMovesWithoutEditingOrQueuing(t *testing.T) {
	work := travelWorkspace(t)
	const source = "$schema: https://github.com/janearc/slayer/schemas/job-v1.json\nscene: {set: bay.yaml}\ncamera: {name: bay}\n"
	document, err := draft.Open("bay-job.yaml", []byte(source))
	if err != nil || document.Kind() != "job" {
		t.Fatalf("invalid job fixture: %v", err)
	}
	work.state.document = document
	work.editor.SetText(source)
	work.working.source = source
	original := work.working.scene.Eye
	for _, input := range []struct {
		name                 string
		press, move, release func() error
	}{
		{"Orbit", work.ProofTrackballPress, work.ProofTrackballMove, work.ProofTrackballRelease},
		{"Travel", work.ProofTravelPress, work.ProofTravelMove, work.ProofTravelRelease},
	} {
		before := work.working.scene.Eye
		for _, step := range []func() error{input.press, input.move, input.release} {
			if err := step(); err != nil {
				t.Fatalf("%s: %v", input.name, err)
			}
		}
		if work.working.scene.Eye == before {
			t.Fatalf("%s release discarded the inspection view", input.name)
		}
	}
	request := work.previewRequest(time.Time{})
	if document.Dirty() || request.Text != source || request.Profile.Camera != "" ||
		work.pending.document != nil || work.Running() {
		t.Fatal("working inspection changed authored render inputs or queued a render")
	}
	work.ui.Window().HandleResize(int(smallestWindow.Width), int(smallestWindow.Height))
	work.rebuild()
	drawMinimumFilmWindow(t, work)
	clickLibraryPoint(work, filmControlCentre(t, work, "Reset view"))
	if work.working.scene.Eye != original {
		t.Fatal("Reset view did not restore the original camera")
	}
}

func TestSceneDetailCameraMovesWithoutEditingOrQueuing(t *testing.T) {
	work := travelWorkspace(t)
	work.working.detail = "scene.set.things/stone"
	source := work.state.document.Text()
	for _, input := range []struct {
		name string
		move func() error
	}{
		{"Orbit", func() error {
			for _, step := range []func() error{work.ProofTrackballPress, work.ProofTrackballMove, work.ProofTrackballRelease} {
				if err := step(); err != nil {
					return err
				}
			}
			return nil
		}},
		{"Travel", func() error {
			for _, step := range []func() error{work.ProofTravelPress, work.ProofTravelMove, work.ProofTravelRelease} {
				if err := step(); err != nil {
					return err
				}
			}
			return nil
		}},
		{"Zoom", func() error { return work.Navigate(0, 0, 1.1) }},
	} {
		before := work.working.scene.Eye
		if err := input.move(); err != nil {
			t.Fatalf("%s: %v", input.name, err)
		}
		if work.working.scene.Eye == before {
			t.Fatalf("%s did not move the detail camera", input.name)
		}
	}
	if work.state.document.Text() != source || work.pending.document != nil || work.Running() {
		t.Fatal("detail navigation changed the authored camera or queued a render")
	}
}
