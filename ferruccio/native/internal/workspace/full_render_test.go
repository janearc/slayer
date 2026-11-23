package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gogpu/gogpu"
	"github.com/gogpu/ui/app"
)

// newRunningWorkspace returns a workspace whose render is held by a stub that
// records whether it was cancelled.
func newRunningWorkspace(t *testing.T, full bool) (*Workspace, *bool) {
	t.Helper()
	work, err := New(gogpu.NewApp(gogpu.DefaultConfig()), app.New(), Options{
		Examples: "../../../examples", Composition: "observe",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { work.clearNavigation(); os.RemoveAll(filepath.Dir(work.state.path)) })
	cancelled := false
	work.cancel = func() { cancelled = true }
	work.fullRunning = full
	return work, &cancelled
}

// TestFullRenderIgnoresUnrelatedActions checks that nothing but a request to
// stop cancels a running full render, and that each refusal is in words.
func TestFullRenderIgnoresUnrelatedActions(t *testing.T) {
	work, cancelled := newRunningWorkspace(t, true)
	work.stopPreview()
	if *cancelled {
		t.Fatal("stopPreview cancelled a running full render")
	}
	if err := work.Navigate(0.1, 0.1, 0); err == nil || !strings.Contains(err.Error(), "full render is running") {
		t.Fatalf("camera gesture error = %v; want a full render refusal", err)
	}
	for name, action := range map[string]func(){
		"reference": func() { work.reference("venus") },
		"example":   func() { work.chooseExample("first-picture.yaml") },
		"open":      work.open,
	} {
		work.status = ""
		action()
		if *cancelled || !strings.Contains(work.status, "full render is running") {
			t.Fatalf("%s: cancelled=%t status=%q", name, *cancelled, work.status)
		}
	}
}

// TestFullRenderStopsOnRequest is the anti-test: the user's own request to
// stop still cancels a full render.
func TestFullRenderStopsOnRequest(t *testing.T) {
	work, cancelled := newRunningWorkspace(t, true)
	work.StopPreview()
	if !*cancelled || !work.cancelRequested {
		t.Fatal("StopPreview did not cancel the full render")
	}
}

// TestPreviewStillLatestWins is the anti-test for previews: a later action
// cancels a running preview.
func TestPreviewStillLatestWins(t *testing.T) {
	work, cancelled := newRunningWorkspace(t, false)
	work.stopPreview()
	if !*cancelled {
		t.Fatal("stopPreview did not cancel a running preview")
	}
}
