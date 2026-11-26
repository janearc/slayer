package workspace

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/ferruccio/internal/draft"
	"github.com/janearc/slayer/ferruccio/native/internal/stagewire"
	"github.com/janearc/slayer/session/placement"
)

func TestStagedWorkspaceAcceptsOnlyCurrentResult(t *testing.T) {
	work := playWorkspace(t)
	started := make(chan struct{})
	release := make(chan struct{})
	work.requestStaged(func(_ context.Context, _, _, _ string) (*stagedView, error) {
		close(started)
		<-release
		return &stagedView{scene: &placement.DisplayScene{Width: 640, Height: 400}, notice: "aggregate only"}, nil
	})
	<-started
	if work.working.scene != nil || !strings.Contains(work.caption, "staging") {
		t.Fatal("loading was presented as a current scene")
	}
	work.state.document.Replace(work.state.document.Text() + "\n# new draft\n")
	close(release)
	waitStaged(t, work)
	if work.working.scene != nil {
		t.Fatal("accepted an old draft's staged result")
	}
	work.closeStaged()
}

func TestWorkingFallbackWithBuiltHelper(t *testing.T) {
	helper := os.Getenv("FERRUCCIO_STAGE_HELPER")
	if helper == "" {
		t.Skip("set FERRUCCIO_STAGE_HELPER to check the built helper")
	}
	const source = `scene:
  film: {width: 32, height: 24, rays: 1, wide: 45, exposure: 1}
  cameras:
    view: {location: {at: [0, -5, 2]}, orientation: {look_at: [1, 0, 0]}}
  world:
    materials:
      matte: {colour: {srgb_linear: [0.3, 0.4, 0.5]}, source: eye}
    boxes:
      first: {material: matte, middle: [10, 20, 3], size: [6, 4, 6], source: eye}
`
	work := playWorkspace(t)
	path := filepath.Join(t.TempDir(), "unsaved.yaml")
	document, err := draft.Open(path, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	work.state.document, work.state.path = document, path
	cache := t.TempDir()
	work.working.stageOverride = func(ctx context.Context, path, source, kind string) (*stagedView, error) {
		snapshot, err := stagewire.Run(ctx, helper, path, source, kind, cache, "", "", 2)
		if err != nil {
			return nil, err
		}
		return &stagedView{scene: &snapshot.Scene, notice: snapshot.Notice}, nil
	}
	work.updateWorking()
	waitStaged(t, work)
	defer work.closeStaged()
	if work.working.scene == nil || len(work.working.scene.Batches) != 1 || !strings.Contains(work.diagnostic, "generated members and proxies are read-only") {
		t.Fatalf("staged workspace = %v, %q", work.working.scene, work.diagnostic)
	}
	batch := work.working.scene.Batches[0]
	low, high, ok := batch.Mesh.Bounds()
	low, high = batch.Places[0].Out(low), batch.Places[0].Out(high)
	if !ok || low.X != 7 || high.X != 13 || work.pending.document != nil || work.Running() {
		t.Fatal("working view did not use unsaved staged geometry without a render")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("working view wrote the unsaved source")
	}
}

func TestStagedWorkspaceShowsNoticeAndFailure(t *testing.T) {
	work := playWorkspace(t)
	work.requestStaged(func(_ context.Context, _, _, _ string) (*stagedView, error) {
		return &stagedView{scene: &placement.DisplayScene{Width: 640, Height: 400}, notice: "aggregate only"}, nil
	})
	waitStaged(t, work)
	if work.working.scene == nil || !strings.Contains(work.diagnostic, "aggregate only") {
		t.Fatalf("staged scene or notice missing: %q", work.diagnostic)
	}
	work.diagnostic = ""
	if !strings.Contains(work.workingCaption(), "aggregate only") || !strings.Contains(work.playReadout(), "aggregate only") {
		t.Fatal("staged notice vanished after a later validation")
	}
	work.requestStaged(func(_ context.Context, _, _, _ string) (*stagedView, error) {
		return nil, errors.New("a missing shelf")
	})
	waitStaged(t, work)
	if work.working.scene != nil || !strings.Contains(work.diagnostic, "a missing shelf") || !strings.Contains(work.caption, "unavailable") {
		t.Fatalf("helper failure hidden: %q %q", work.diagnostic, work.caption)
	}
	work.closeStaged()
}

func TestWorkingFallbackStagesAndCancelsWhenDraftChanges(t *testing.T) {
	work := playWorkspace(t)
	source := strings.Replace(checkingSource, "visitor: {kind:", "visitor: {alight: true, kind:", 1)
	work.state.document.Replace(source)
	started := make(chan struct{})
	cancelled := make(chan struct{})
	work.working.stageOverride = func(ctx context.Context, _, _, _ string) (*stagedView, error) {
		close(started)
		<-ctx.Done()
		close(cancelled)
		return nil, ctx.Err()
	}
	work.updateWorking()
	<-started
	if !work.working.staging || work.workingCurrent() {
		t.Fatal("fallback was presented as ready while staging")
	}
	work.state.document.Replace(checkingSource)
	work.updateWorking()
	select {
	case <-cancelled:
	case <-time.After(3 * time.Second):
		t.Fatal("changing to a supported set did not cancel staging")
	}
	work.closeStaged()
}

func TestStagingWaitsForInputToSettle(t *testing.T) {
	work := playWorkspace(t)
	started := make(chan struct{}, 1)
	build := func(_ context.Context, _, _, _ string) (*stagedView, error) {
		started <- struct{}{}
		return &stagedView{scene: &placement.DisplayScene{}, notice: "aggregate"}, nil
	}
	work.requestStaged(build)
	work.requestStaged(build)
	select {
	case <-started:
		t.Fatal("staging started before input settled")
	case <-time.After(50 * time.Millisecond):
	}
	waitStaged(t, work)
	select {
	case <-started:
	default:
		t.Fatal("settled input did not start staging")
	}
	select {
	case <-started:
		t.Fatal("superseded input also started staging")
	default:
	}
	work.closeStaged()
}

func waitStaged(t *testing.T, work *Workspace) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if work.drainStaged() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	work.closeStaged()
	t.Fatal("staging result did not arrive")
}
