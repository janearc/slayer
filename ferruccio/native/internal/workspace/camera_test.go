package workspace

import (
	"context"
	"fmt"
	"github.com/gogpu/gogpu"
	"github.com/gogpu/ui/app"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/janearc/slayer/ferruccio/internal/preview"
	"image"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestCameraEditChangesDraftAndCancelsOldRender exercises the real workspace.
func TestCameraEditChangesDraftAndCancelsOldRender(t *testing.T) {
	work := cameraWorkspace(t)
	original := work.state.document.Text()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	work.cancel = cancel
	if err := work.Navigate(30, 15, 0.8); err != nil {
		t.Fatal(err)
	}
	if ctx.Err() != context.Canceled {
		t.Fatal("obsolete render was not canceled")
	}
	if work.cancel == nil {
		t.Fatal("canceled renderer was treated as finished before its result")
	}
	if work.editor.Text() != work.state.document.Text() || work.editor.Text() == original {
		t.Fatal("camera edit did not reach both document and visible editor")
	}
	for key, want := range map[string]string{"yaw": "30", "pitch": "15"} {
		got, err := work.state.document.Value("orbit", key)
		if err != nil || got != want {
			t.Fatalf("orbit %s = %q, %v", key, got, err)
		}
	}
	if wide, _ := work.state.document.Value("wide"); wide != "40" {
		t.Fatalf("wide = %s", wide)
	}
	if !work.pending.take(work.state, work.specimen, time.Now().Add(time.Second), false) {
		t.Fatal("latest camera request was not queued")
	}
	raw, err := os.ReadFile(work.state.path)
	if err != nil || string(raw) != original {
		t.Fatal("navigation overwrote the saved plan", err)
	}
}

// TestCameraRequiresRenderedNavigableImage refuses edits on references and jobs.
func TestCameraRequiresRenderedNavigableImage(t *testing.T) {
	work := cameraWorkspace(t)
	work.viewer.SetNavigable(false)
	original := work.state.document.Text()
	if err := work.Navigate(30, 0, 0); err == nil {
		t.Fatal("reference image allowed camera editing")
	}
	if work.state.document.Text() != original || work.pending.document != nil {
		t.Fatal("refused navigation changed the draft or queued a render")
	}
}

// TestCameraNoOpDoesNotCancelOrQueue avoids renders from clicks and clamped views.
func TestCameraNoOpDoesNotCancelOrQueue(t *testing.T) {
	work := cameraWorkspace(t)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	work.cancel = cancel
	original := work.state.document.Text()
	if err := work.Navigate(0, 0, 0); err != nil {
		t.Fatal(err)
	}
	if ctx.Err() != nil || work.pending.document != nil || work.state.document.Text() != original {
		t.Fatal("no-op altered the current view")
	}
}

// TestFocusedPreviewUndoQueuesTheRestoredCamera proves undo redraws the authored view.
func TestFocusedPreviewUndoQueuesTheRestoredCamera(t *testing.T) {
	work := cameraWorkspace(t)
	before := work.state.document.Text()
	work.viewer.SetBounds(geometry.Rect{Min: geometry.Pt(0, 0), Max: geometry.Pt(200, 120)})
	work.ui.SetRoot(work.viewer)
	point := geometry.Pt(40, 40)
	work.ui.HandleEvent(event.NewMouseEvent(event.MousePress, event.ButtonLeft, event.ButtonStateLeft, point, point, event.ModNone))
	work.ui.HandleEvent(event.NewMouseEvent(event.MouseRelease, event.ButtonLeft, 0, point, point, event.ModNone))
	if err := work.Navigate(30, 15, 0.8); err != nil {
		t.Fatal(err)
	}
	changed := work.state.document.Text()
	work.ui.HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyZ, 'z', event.ModSuper))
	if got := work.state.document.Text(); got != before {
		t.Fatalf("focused preview undo restored draft = %q, want original", got)
	}
	if work.pending.document == nil || work.pending.text != before {
		t.Fatal("focused preview undo did not queue the restored camera draft")
	}
	work.ui.HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyZ, 'Z', event.ModSuper|event.ModShift))
	if got := work.state.document.Text(); got != changed {
		t.Fatal("focused preview redo did not restore the edited camera draft")
	}
	if work.pending.document == nil || work.pending.text != changed {
		t.Fatal("focused preview redo did not queue the restored camera draft")
	}
}

// TestStopRejectsLateResultsEvenWhenDocumentDidNotChange closes the cancel race.
func TestStopRejectsLateResultsEvenWhenDocumentDidNotChange(t *testing.T) {
	work := cameraWorkspace(t)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	work.cancel = cancel
	work.pending.remember(work.state, work.specimen, time.Now())
	previous := work.generation
	work.stopPreview()
	if ctx.Err() != context.Canceled || work.pending.document != nil || work.generation == previous {
		t.Fatal("Stop failed to cancel and invalidate queued/in-flight views")
	}
	work.finishPreview(renderResult{generation: previous, observation: preview.Update{State: "cancelled", Sequence: 1, Terminal: true}, err: fmt.Errorf("late canceled result")})
	if work.diagnostic != "" {
		t.Fatal("late result became a visible error", work.diagnostic)
	}
	if work.cancel != nil {
		t.Fatal("completed worker still appears running")
	}
}

// cameraWorkspace opens a real saved plan with a rendered-surface capability.
func cameraWorkspace(t *testing.T) *Workspace {
	t.Helper()
	raw, err := os.ReadFile("../../../examples/subaru-settings.yaml")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "settings.yaml")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	work, err := New(gogpu.NewApp(gogpu.DefaultConfig()), app.New(), Options{File: path})
	if err != nil {
		t.Fatal(err)
	}
	work.specimen = "subaru"
	work.viewer.set(image.NewRGBA(image.Rect(0, 0, 640, 360)))
	work.viewer.SetNavigable(true)
	t.Cleanup(func() {
		if work.navigationTimer != nil {
			work.navigationTimer.Stop()
		}
	})
	return work
}
