package workspace

import (
	"context"
	"errors"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/janearc/slayer/ferruccio/internal/draft"
	"github.com/janearc/slayer/ferruccio/native/internal/stagewire"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/session/placement"
	"github.com/janearc/slayer/set"
)

func TestStagedWorkspaceAcceptsOnlyCurrentResult(t *testing.T) {
	work := playWorkspace(t)
	started := make(chan struct{})
	release := make(chan struct{})
	work.requestStaged(func(_ context.Context, _, _, _ string) (*stagedView, error) {
		close(started)
		<-release
		return &stagedView{scene: &placement.DisplayScene{Width: 640, Height: 400}, notice: "aggregate only"}, nil
	}, false)
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

func TestWorkingOpenPreparesLightweightSceneOffThread(t *testing.T) {
	work := playWorkspace(t)
	work.state.document.Replace(checkingSource)
	started := make(chan struct{}, 1)
	work.working.stageOverride = func(context.Context, string, string, string) (*stagedView, error) {
		started <- struct{}{}
		return nil, errors.New("lightweight scene reached the helper")
	}
	defer work.closeStaged()

	work.updateWorking()
	if !work.working.staging || work.working.parsed != nil || work.working.scene != nil {
		t.Fatal("working view kickoff constructed the scene on the caller's thread")
	}
	waitStaged(t, work)
	if work.working.parsed == nil || work.working.scene == nil ||
		len(work.working.scene.Objects) == 0 || work.working.notice != "" {
		t.Fatalf("working view did not receive lightweight geometry: %q", work.working.failure)
	}
	if work.Running() {
		t.Fatal("opening started a photograph")
	}
	select {
	case <-started:
		t.Fatal("lightweight scene invoked the staging helper")
	default:
	}
}

func TestWorkingOpenNamesLightLimitsOnFastPath(t *testing.T) {
	work := playWorkspace(t)
	source := strings.Replace(checkingSource, "light: none", "light: sun\n  sun: {azimuth: 225, altitude: 28}", 1)
	work.state.document.Replace(source + `  world:
    lamps:
      porch: {at: [0, 0, 1], temperature: 2700, strength: 0.35, radius: 0.0625, source: eye}
`)
	work.working.stageOverride = func(context.Context, string, string, string) (*stagedView, error) {
		return nil, errors.New("lightweight scene reached the helper")
	}
	defer work.closeStaged()
	work.updateWorking()
	waitStaged(t, work)

	scene, notice := work.working.scene, work.working.notice
	if scene == nil || len(scene.Lights) != 2 || scene.Lights[0].Author.Name != "sun" || scene.Lights[1].Author.Name != "porch" {
		t.Fatalf("cached scene did not retain the scene sun and world lamp: %q", work.working.failure)
	}
	for _, expected := range []string{"read-only", "lighting and shadows are omitted", "Local markers cover the scene sun and supported world lights"} {
		if !strings.Contains(notice, expected) {
			t.Fatalf("missing %q in %q", expected, notice)
		}
	}
	if !strings.Contains(work.workingCaption(), "View details") {
		t.Fatal("working caption did not lead to the light limitations")
	}
}

func TestWorkingOpenKeepsInvalidDraftOutOfStagingHelper(t *testing.T) {
	work := playWorkspace(t)
	work.state.document.Replace("scene: {")
	validation := work.state.document.Check()
	if validation == nil {
		t.Fatal("expected a schema refusal for the test draft")
	}
	started := make(chan struct{}, 1)
	work.working.stageOverride = func(context.Context, string, string, string) (*stagedView, error) {
		started <- struct{}{}
		return nil, errors.New("invalid draft reached the helper")
	}
	defer work.closeStaged()

	work.updateWorking()
	if !work.working.staging {
		t.Fatal("working view kickoff inspected the invalid draft on the caller's thread")
	}
	waitStaged(t, work)
	if work.working.parsed != nil || work.working.scene != nil ||
		work.working.failure != validation.Error() {
		t.Fatalf("working view lost the schema refusal: %q", work.working.failure)
	}
	select {
	case <-started:
		t.Fatal("invalid draft invoked the staging helper")
	default:
	}
}

func TestStagedPreviewGoesStraightToItsReplacement(t *testing.T) {
	work := playWorkspace(t)
	source := work.state.document.Text()
	preview := &placement.DisplayScene{Width: 640, Height: 400, Batches: []placement.StagedBatch{{}}}
	work.working.source, work.working.path = source+"\n# previous\n", work.state.path
	work.working.preview = &workingPreview{document: work.state.document, scene: preview, source: source, path: work.state.path}
	started := make(chan struct{})
	work.working.stageOverride = func(ctx context.Context, _, _, _ string) (*stagedView, error) {
		close(started)
		<-ctx.Done()
		return nil, ctx.Err()
	}
	defer work.closeStaged()
	work.updateWorking()
	if work.working.parsed != nil {
		t.Fatal("viewport kickoff parsed the provisional source synchronously")
	}
	<-started
	if work.drainStaged() {
		t.Fatal("metadata delivery reported completed scene staging")
	}
	if !work.working.staging || !work.working.provisional || work.working.scene != preview {
		t.Fatal("staged edit did not retain its preview while building the replacement")
	}
	if work.working.parsed == nil {
		t.Fatal("staged edit lost the current set needed for terrain checks")
	}
}

func TestRapidRestageKeepsOnlyCurrentAuthoringData(t *testing.T) {
	work := playWorkspace(t)
	started := make(chan string, 2)
	work.working.stageOverride = func(ctx context.Context, _, source, _ string) (*stagedView, error) {
		started <- source
		<-ctx.Done()
		return nil, ctx.Err()
	}
	defer work.closeStaged()
	scene := &placement.DisplayScene{Width: 640, Height: 400}
	restage := func(source string) {
		work.state.document.Replace(source)
		work.working.preview = &workingPreview{
			document: work.state.document, scene: scene, source: source, path: work.state.path,
		}
		work.updateWorking()
		waitForStageSource(t, started, source)
	}
	first := groundEditFixture
	second := strings.Replace(first, "west: 1001", "west: 1003", 1)
	restage(first)
	work.state.document.Replace(second)
	work.drainStaged()
	if work.working.parsed != nil {
		t.Fatal("obsolete authoring data was published after a newer edit")
	}
	restage(second)
	work.drainStaged()
	if work.working.parsed == nil {
		t.Fatal("latest authoring data was not published")
	}
	ground, ok := work.working.parsed.Ground()
	if west, _, _, _, _ := ground.Window(); !ok || west != 1003 {
		t.Fatalf("authoring controls received the wrong terrain window: %v", west)
	}
}

func TestInvalidRestageRefusesAuthoringDataAndHelper(t *testing.T) {
	work := playWorkspace(t)
	built := make(chan struct{}, 1)
	work.working.stageOverride = func(context.Context, string, string, string) (*stagedView, error) {
		built <- struct{}{}
		return nil, errors.New("invalid source reached the helper")
	}
	defer work.closeStaged()
	invalid := "scene: {"
	work.state.document.Replace(invalid)
	work.working.preview = &workingPreview{
		document: work.state.document, scene: &placement.DisplayScene{},
		source: invalid, path: work.state.path,
	}
	work.updateWorking()
	if work.working.staging {
		waitStaged(t, work)
	}
	if work.working.parsed != nil || work.working.scene != nil || work.working.failure == "" {
		t.Fatal("invalid scene retained provisional geometry or authoring data")
	}
	select {
	case <-built:
		t.Fatal("invalid draft started the staging helper")
	default:
	}
}

func TestClosingWhilePreparingDefersOnlyTheRead(t *testing.T) {
	loader := newStageLoader(nil)
	work := playWorkspace(t)
	work.working.stager = loader
	reading, release := make(chan struct{}), make(chan struct{})
	var released sync.Once
	defer released.Do(func() { close(release) })
	built := make(chan struct{}, 1)
	loader.requestPreparedBuild("scene.yaml", "", "set",
		func(context.Context, string, string, string) (*stagedView, error) {
			close(reading)
			<-release
			return nil, nil
		},
		func(context.Context, string, string, string) (*stagedView, error) {
			built <- struct{}{}
			return nil, nil
		})
	<-reading

	closed := make(chan struct{})
	go func() {
		work.QuitDesktop()
		close(closed)
	}()
	select {
	case <-closed:
	case <-time.After(3 * time.Second):
		t.Fatal("window close waited for the source read")
	}
	finished := make(chan struct{})
	go func() {
		work.Close()
		close(finished)
	}()
	select {
	case <-finished:
		t.Fatal("app shutdown abandoned the pending source read")
	case <-time.After(30 * time.Millisecond):
	}
	released.Do(func() { close(release) })
	select {
	case <-finished:
	case <-time.After(3 * time.Second):
		t.Fatal("app shutdown did not join the canceled read")
	}
	select {
	case <-built:
		t.Fatal("canceled source read started the helper")
	default:
	}
}

func TestRepeatedRestageCloseKeepsOnlyLatestReadWaiting(t *testing.T) {
	work := playWorkspace(t)
	const attempts = 8
	started := make(chan string, attempts)
	work.working.stageOverride = func(ctx context.Context, _, source, _ string) (*stagedView, error) {
		started <- source
		<-ctx.Done()
		return nil, ctx.Err()
	}
	slot := make(chan struct{}, 1)
	slot <- struct{}{}
	work.working.stageReadSlot = slot
	held := true
	defer func() {
		if held {
			<-slot
		}
		work.Close()
	}()
	var latest string
	for attempt := range attempts {
		latest = fmt.Sprintf("%s\n# edit %d\n", groundEditFixture, attempt)
		work.state.document.Replace(latest)
		work.working.preview = &workingPreview{
			document: work.state.document, scene: &placement.DisplayScene{},
			source: latest, path: work.state.path,
		}
		work.updateWorking()
		if attempt < attempts-1 {
			work.closeStaged()
		}
	}
	select {
	case <-started:
		t.Fatal("a reopened stage bypassed the shared read slot")
	case <-time.After(300 * time.Millisecond):
	}
	<-slot
	held = false
	waitForStageSource(t, started, latest)
	work.drainStaged()
	if work.working.parsed == nil {
		t.Fatal("the latest reopened stage did not publish authoring data")
	}
	select {
	case <-started:
		t.Fatal("a canceled reopen started a helper")
	default:
	}
}

func waitForStageSource(t *testing.T, started <-chan string, source string) {
	t.Helper()
	select {
	case received := <-started:
		if received != source {
			t.Fatal("staging worker opened an obsolete source")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("staging worker did not start")
	}
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
	work.working.rollback = &workingRollback{scene: &placement.DisplayScene{Width: 1}}
	work.requestStaged(func(_ context.Context, _, _, _ string) (*stagedView, error) {
		return &stagedView{scene: &placement.DisplayScene{Width: 640, Height: 400}, notice: "aggregate only"}, nil
	}, false)
	waitStaged(t, work)
	if work.working.scene == nil || work.working.rollback != nil || !strings.Contains(work.diagnostic, "aggregate only") {
		t.Fatalf("staged scene or notice missing: %q", work.diagnostic)
	}
	if !strings.Contains(work.workingCaption(), "View details") || work.working.notice != "aggregate only" {
		t.Fatal("staged notice vanished after a later validation")
	}
	if strings.Contains(work.playReadout(), "aggregate only") {
		t.Fatal("staged notice obscures the editor readout")
	}
	work.diagnostic = ""
	work.requestStaged(func(_ context.Context, _, _, _ string) (*stagedView, error) {
		return nil, errors.New("a missing shelf")
	}, false)
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

func TestUndoWhileStagingRestoresThePreviousExactScene(t *testing.T) {
	work := playWorkspace(t)
	source := strings.Replace(checkingSource, "visitor: {kind:", "visitor: {alight: true, kind:", 1)
	work.state.document.Replace(source)
	work.editor.SetText(source)
	work.play.selected = "scene.set.things/visitor"
	original := &placement.DisplayScene{Width: 640, Height: 400}
	work.working.scene, work.working.source, work.working.path = original, source, work.state.path
	work.working.notice = "original staged scene"
	started, cancelled := make(chan struct{}), make(chan struct{})
	work.working.stageOverride = func(ctx context.Context, _, _, _ string) (*stagedView, error) {
		close(started)
		<-ctx.Done()
		close(cancelled)
		return nil, ctx.Err()
	}
	defer work.closeStaged()
	if err := work.changePose(1, 0, 0); err != nil {
		t.Fatal(err)
	}
	work.updateWorking()
	<-started
	if !work.working.staging || work.working.scene != nil {
		t.Fatal("changed scene was not staging")
	}
	work.undoCamera(work.ui.Window().Context(), false)
	if work.state.document.Text() != source || work.working.scene != original || work.working.source != source || work.working.staging {
		t.Fatal("undo did not immediately restore the exact previous scene")
	}
	if work.working.notice != "original staged scene" || work.working.rollback != nil {
		t.Fatal("restored scene lost its notice or retained the backup")
	}
	select {
	case <-cancelled:
	case <-time.After(3 * time.Second):
		t.Fatal("undo did not cancel the obsolete stage")
	}
}

func TestCommittedGestureKeepsItsUnchangedBaseForUndo(t *testing.T) {
	work := playWorkspace(t)
	source := work.state.document.Text()
	base := &placement.DisplayScene{Width: 640, Height: 400, Objects: []placement.DisplayObject{{ID: "scene.set.things/larson"}}}
	work.working.scene, work.working.source, work.working.path, work.working.displayed = base, source, work.state.path, true
	work.viewer.dragObject = true
	work.beginWorking()
	if work.working.gesture == nil {
		t.Fatal("gesture did not start")
	}
	work.working.scene = &placement.DisplayScene{Width: 111, Height: 222}
	if err := work.changePose(1, 0, 0); err != nil {
		t.Fatal(err)
	}
	work.endWorking(true)
	work.undoCamera(work.ui.Window().Context(), false)
	if work.working.scene == nil || work.working.scene.Width != 640 || work.working.scene.Height != 400 || work.state.document.Text() != source {
		t.Fatal("undo restored the transient dragged scene instead of the unchanged base")
	}
}

func TestCommittedDragShowsOnlyAProvisionalViewUntilRestaged(t *testing.T) {
	work := playWorkspace(t)
	source := strings.Replace(checkingSource, "visitor: {kind:", "visitor: {alight: true, kind:", 1)
	work.state.document.Replace(source)
	work.editor.SetText(source)
	work.play.selected = "scene.set.things/visitor"
	base := &placement.DisplayScene{Width: 640, Height: 400, Batches: []placement.StagedBatch{{}}}
	work.working.scene, work.working.source, work.working.path, work.working.displayed = base, source, work.state.path, true
	work.working.notice = "original limitations"
	work.viewer.dragObject = true
	work.viewer.SetNavigable(true)
	work.beginWorking()
	approximate := &placement.DisplayScene{Width: 640, Height: 400}
	work.working.scene = approximate
	if err := work.changePose(1, 0, 0); err != nil {
		t.Fatal(err)
	}
	work.endWorking(true)
	started, release := make(chan struct{}), make(chan struct{})
	exact := &placement.DisplayScene{Width: 800, Height: 500}
	work.working.stageOverride = func(ctx context.Context, _, _, _ string) (*stagedView, error) {
		close(started)
		select {
		case <-release:
			return &stagedView{scene: exact, notice: "new limitations"}, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	defer work.closeStaged()
	work.updateWorking()
	<-started
	if work.working.scene != approximate || !work.working.provisional || work.workingCurrent() || work.WorkingReady() || work.viewer.navigable {
		t.Fatal("provisional drag was treated as current, selectable geometry")
	}
	work.play.renderedText = work.state.document.Text()
	if work.viewer.selectionCurrent() || work.viewer.selectObject(.5, .5) || work.workingPick(.5, .5) != "" {
		t.Fatal("provisional view allowed selection from an older render")
	}
	if !strings.Contains(work.caption, "Provisional") || work.working.notice != "original limitations" {
		t.Fatal("provisional view or its limitations were not labelled")
	}
	close(release)
	waitStaged(t, work)
	if work.working.scene != exact || work.working.provisional || work.working.notice != "new limitations" {
		t.Fatal("finished staging did not replace the provisional view")
	}
}

func TestUndoCancelsProvisionalDragAndRestoresExactView(t *testing.T) {
	work := playWorkspace(t)
	source := strings.Replace(checkingSource, "visitor: {kind:", "visitor: {alight: true, kind:", 1)
	work.state.document.Replace(source)
	work.editor.SetText(source)
	work.play.selected = "scene.set.things/visitor"
	base := &placement.DisplayScene{Width: 640, Height: 400, Batches: []placement.StagedBatch{{}}}
	work.working.scene, work.working.source, work.working.path, work.working.displayed = base, source, work.state.path, true
	work.ui.Window().Context().RequestFocus(work.viewer)
	if !work.viewer.IsFocused() {
		t.Fatal("window did not focus the viewport")
	}
	work.viewer.dragObject = true
	work.beginWorking()
	work.working.scene = &placement.DisplayScene{Width: 640, Height: 400}
	if err := work.changePose(1, 0, 0); err != nil {
		t.Fatal(err)
	}
	work.endWorking(true)
	started, cancelled := make(chan struct{}), make(chan struct{})
	work.working.stageOverride = func(ctx context.Context, _, _, _ string) (*stagedView, error) {
		close(started)
		<-ctx.Done()
		close(cancelled)
		return nil, ctx.Err()
	}
	defer work.closeStaged()
	work.updateWorking()
	<-started
	changed := work.state.document.Text()
	if work.viewer.navigable {
		t.Fatal("staging did not suspend navigation")
	}
	work.ui.HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyZ, 'z', event.ModNone))
	work.ui.HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyRight, 0, event.ModNone))
	if work.state.document.Text() != changed || !work.working.staging {
		t.Fatal("unmodified key changed source while staging")
	}
	work.ui.HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyZ, 'z', event.ModSuper))
	if work.state.document.Text() != source || work.working.scene == nil || len(work.working.scene.Batches) != 1 || work.working.provisional || work.working.staging {
		t.Fatal("undo did not restore the exact pre-drag scene")
	}
	select {
	case <-cancelled:
	case <-time.After(3 * time.Second):
		t.Fatal("undo did not cancel the obsolete staging work")
	}

	work.ui.HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyZ, 'Z', event.ModSuper|event.ModShift))
	if work.state.document.Text() != changed {
		t.Fatal("Command-Shift-Z did not restore the authored change")
	}
}

func TestTypedPosePreviewMovesAndTurnsOnlySelectedBatch(t *testing.T) {
	work := playWorkspace(t)
	work.state.document.Replace(checkingSource)
	work.editor.SetText(checkingSource)
	work.play.selected = "scene.set.things/visitor"
	base := &placement.DisplayScene{
		Width: 640, Height: 400,
		Origins: map[string]maths.Vec{"scene.set.things/visitor": maths.V(0, 0, 0)},
		Batches: []placement.StagedBatch{
			{Authored: true, Author: thing.AuthorRef{Collection: "scene.set.things", Name: "visitor"}, Places: []maths.Subspace{{Origin: maths.V(1, 0, 0)}}},
			{Authored: true, Author: thing.AuthorRef{Collection: "scene.set.things", Name: "other"}, Places: []maths.Subspace{{Origin: maths.V(4, 0, 0)}}},
		},
	}
	work.working.scene, work.working.source, work.working.path, work.working.displayed = base, checkingSource, work.state.path, true
	work.play.poseText = [4]string{"2", "3", "0", "90"}
	if err := work.submitCoordinates(true); err != nil {
		t.Fatal(err)
	}
	if work.working.preview == nil {
		t.Fatal("typed pose produced no provisional scene")
	}
	moved := work.working.preview.scene.Batches[0].Places[0].Origin
	if moved.Sub(maths.V(2, 0, -2)).Len() > 1e-9 || base.Batches[0].Places[0].Origin != maths.V(1, 0, 0) || work.working.preview.scene.Batches[1].Places[0].Origin != maths.V(4, 0, 0) {
		t.Fatalf("typed pose moved the wrong geometry: %v", moved)
	}
}

func TestConsecutiveTypedPosesKeepTheProvisionalViewAndExactUndo(t *testing.T) {
	work := playWorkspace(t)
	work.state.document.Replace(checkingSource)
	work.editor.SetText(checkingSource)
	work.play.selected = "scene.set.things/visitor"
	base := &placement.DisplayScene{
		Width: 640, Height: 400,
		Origins: map[string]maths.Vec{"scene.set.things/visitor": maths.V(0, 0, 0)},
		Batches: []placement.StagedBatch{{Authored: true, Author: thing.AuthorRef{Collection: "scene.set.things", Name: "visitor"}, Places: []maths.Subspace{{Origin: maths.V(1, 0, 0)}}}},
	}
	work.working.scene, work.working.source, work.working.path, work.working.displayed = base, checkingSource, work.state.path, true
	started := make(chan struct{}, 8)
	work.working.stageOverride = func(ctx context.Context, _, _, _ string) (*stagedView, error) {
		started <- struct{}{}
		<-ctx.Done()
		return nil, ctx.Err()
	}
	defer work.closeStaged()
	work.play.poseText = [4]string{"2", "3", "0", "90"}
	if err := work.submitCoordinates(true); err != nil {
		t.Fatal(err)
	}
	work.updateWorking()
	<-started
	first := work.working.scene
	firstPoint := first.Batches[0].Places[0].Origin
	firstAnchor := first.Origins[work.play.selected]
	if firstAnchor != maths.V(2, 0, -3) {
		t.Fatalf("first provisional anchor = %v", firstAnchor)
	}
	work.play.poseText = [4]string{"4", "3", "0", "180"}
	if err := work.submitCoordinates(true); err != nil {
		t.Fatal(err)
	}
	if work.working.preview == nil {
		t.Fatal("second edit dropped the provisional scene")
	}
	second := work.working.preview.scene
	want := firstAnchor.Add(maths.Rotation(maths.YAxis, -90*maths.Degree).Out(firstPoint.Sub(firstAnchor))).Add(maths.V(2, 0, 0))
	if second.Batches[0].Places[0].Origin.Sub(want).Len() > 1e-9 || second.Origins[work.play.selected] != maths.V(4, 0, -3) || first.Origins[work.play.selected] != firstAnchor {
		t.Fatal("second pose did not use an independent updated anchor")
	}
	work.updateWorking()
	<-started
	if work.working.scene != second || !work.working.provisional || work.working.rollback == nil || work.working.rollback.scene != base {
		t.Fatal("replacement stage lost the second preview or exact undo base")
	}
	work.undoCamera(work.ui.Window().Context(), false)
	if work.working.preview == nil || work.working.preview.scene != first {
		t.Fatal("first undo lost the previous provisional view")
	}
	work.updateWorking()
	if work.working.scene != first || !work.working.provisional || work.workingCurrent() {
		t.Fatal("first undo did not restore a non-selectable provisional view")
	}
	work.undoCamera(work.ui.Window().Context(), true)
	if work.working.preview == nil || work.working.preview.scene != second {
		t.Fatal("redo lost the later provisional view")
	}
	work.updateWorking()
	work.undoCamera(work.ui.Window().Context(), false)
	work.updateWorking()
	work.undoCamera(work.ui.Window().Context(), false)
	if work.state.document.Text() != checkingSource || work.working.scene != base || work.working.provisional {
		t.Fatal("undo did not restore the original exact scene")
	}
	work.undoCamera(work.ui.Window().Context(), true)
	if work.working.preview == nil || work.working.preview.scene != first {
		t.Fatal("redo from the exact original lost the provisional pose")
	}
	work.updateWorking()
	if work.working.scene != first || !work.working.provisional || work.workingCurrent() {
		t.Fatal("redo from the exact original did not stage the provisional pose")
	}
	work.undoCamera(work.ui.Window().Context(), false)
	if work.working.scene != base || work.working.provisional {
		t.Fatal("undo after redo did not restore the exact original")
	}
}

func TestConsecutiveSizeChangesKeepProvisionalGeometry(t *testing.T) {
	work := playWorkspace(t)
	work.state.document.Replace(checkingSource)
	work.editor.SetText(checkingSource)
	work.play.selected = "scene.set.things/visitor"
	read, err := set.Read(work.state.path, []byte(checkingSource))
	if err != nil {
		t.Fatal(err)
	}
	base := &placement.DisplayScene{
		Origins: map[string]maths.Vec{work.play.selected: maths.V(1, 0, 0)},
		Batches: []placement.StagedBatch{{Authored: true, Author: thing.AuthorRef{Collection: "scene.set.things", Name: "visitor"}, Places: []maths.Subspace{{Origin: maths.V(2, 0, 0)}}}},
	}
	work.working.scene, work.working.parsed, work.working.source, work.working.path, work.working.displayed = base, &read, checkingSource, work.state.path, true
	work.working.stageOverride = func(ctx context.Context, _, _, _ string) (*stagedView, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	defer work.closeStaged()
	if err := work.changeThingScale(2); err != nil {
		t.Fatal(err)
	}
	work.updateWorking()
	first := work.working.scene
	if !work.working.provisional || first.Batches[0].Places[0].Origin != maths.V(3, 0, 0) || first.Batches[0].Places[0].Scale != 2 {
		t.Fatal("first size change did not preview around the authored origin")
	}
	if err := work.changeThingScale(3); err != nil {
		t.Fatal(err)
	}
	work.updateWorking()
	second := work.working.scene
	if !work.working.provisional || second.Batches[0].Places[0].Origin != maths.V(4, 0, 0) || second.Batches[0].Places[0].Scale != 3 || first.Batches[0].Places[0].Scale != 2 {
		t.Fatal("second size change did not preserve and extend the first preview")
	}
	work.undoCamera(work.ui.Window().Context(), false)
	work.updateWorking()
	if work.working.scene != first || !work.working.provisional {
		t.Fatal("undo lost the preceding size preview")
	}
	work.undoCamera(work.ui.Window().Context(), false)
	if work.working.scene != base || work.working.provisional {
		t.Fatal("undo did not restore the exact original")
	}
}

func TestRenderedImageDoesNotWaitForWorkingViewGPU(t *testing.T) {
	work := playWorkspace(t)
	picture := image.NewRGBA(image.Rect(0, 0, 2, 2))
	working := image.NewRGBA(image.Rect(0, 0, 2, 2))
	work.working.scene = &placement.DisplayScene{}
	work.working.source = work.state.document.Text()
	work.working.path = work.state.path
	work.working.photograph = picture
	work.working.mode = "rendered"
	work.working.displayed = true
	work.working.dirty = true
	work.working.waiting = false
	work.viewer.set(working)
	work.updateWorking()
	if work.viewer.source != picture || work.working.displayed || !work.working.dirty {
		t.Fatal("rendered image waited for a working-view GPU draw")
	}
}

func TestStagingWaitsForInputToSettle(t *testing.T) {
	work := playWorkspace(t)
	started := make(chan struct{}, 1)
	build := func(_ context.Context, _, _, _ string) (*stagedView, error) {
		started <- struct{}{}
		return &stagedView{scene: &placement.DisplayScene{}, notice: "aggregate"}, nil
	}
	work.requestStaged(build, false)
	work.requestStaged(build, false)
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

func TestPhotographDoesNotPickWithStagedProjection(t *testing.T) {
	work := playWorkspace(t)
	source := work.state.document.Text()
	mesh := &placement.DisplayMesh{
		Vertices: []placement.DisplayVertex{
			{Position: maths.V(-1, -1, 0)}, {Position: maths.V(1, -1, 0)}, {Position: maths.V(0, 1, 0)},
		},
		Indices: []uint32{0, 1, 2},
	}
	stage := &placement.DisplayScene{
		Width: 640, Height: 400, Lens: film.Pinhole{},
		Batches: []placement.StagedBatch{{
			Mesh: mesh, Places: []maths.Subspace{{Origin: maths.V(0, 0, -5)}},
			Author: thing.AuthorRef{Collection: "scene.set.things", Name: "larson"}, Authored: true,
		}},
	}
	work.working.scene, work.working.source, work.working.path = stage, source, work.state.path
	work.play.renderedText, work.acceptedPath = source, work.state.path
	picture := image.NewRGBA(image.Rect(0, 0, 800, 300))
	work.working.photograph = picture
	work.viewer.set(picture)
	work.play.selected = ""
	if !work.workingCurrent() {
		t.Fatal("stage fixture was not current")
	}
	if work.viewer.selectionCurrent() || work.viewer.selectObject(.5, .5) || work.workingPick(.5, .5) != "" {
		t.Fatal("photograph used the staged projection")
	}
	if !strings.Contains(work.caption, "working view") {
		t.Fatalf("selection did not explain how to resume editing: %q", work.caption)
	}
	picture = image.NewRGBA(image.Rect(0, 0, 640, 400))
	work.working.photograph = picture
	work.viewer.set(picture)
	work.acceptedPath = work.state.path
	if work.viewer.selectionCurrent() || work.workingPick(.5, .5) != "" {
		t.Fatal("matching photograph dimensions authorized picking")
	}
	work.viewer.dragObject = true
	work.beginWorking()
	if work.working.gesture != nil {
		t.Fatal("photo drag used the staging projection")
	}
	work.state.path = filepath.Join(t.TempDir(), "saved-as.yaml")
	work.working.path = work.state.path
	if work.workingCurrent() {
		t.Fatal("render accepted at the old path made the new projection current")
	}
	work.working.displayed = true
	if !work.viewer.selectObject(.5, .5) || work.play.selected != "scene.set.things/larson" {
		t.Fatal("working view no longer picked its projected object")
	}
}

func TestReplacingPhotoCancelsCapturedObjectDrag(t *testing.T) {
	work := playWorkspace(t)
	source := work.state.document.Text()
	view := work.viewer
	view.SetBounds(geometry.Rect{Max: geometry.Pt(640, 400)})
	base := &placement.DisplayScene{Width: 640, Height: 400}
	work.working.scene, work.working.source = base, source
	work.working.path, work.working.displayed = work.state.path, true
	view.selectObject = func(float64, float64) bool { return true }
	commits := 0
	view.changeObject = func(float64, float64, float64, float64) { commits++ }
	ctx := work.ui.Window().Context()
	start := geometry.Pt(320, 100)
	if !view.Event(ctx, event.NewMouseEvent(event.MousePress, event.ButtonLeft, event.ButtonStateLeft, start, start, event.ModNone)) ||
		!view.dragging || !view.dragObject || work.working.gesture == nil {
		t.Fatal("object drag did not capture the working projection")
	}
	path := filepath.Join(t.TempDir(), "replacement.png")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	encoded := png.Encode(file, image.NewRGBA(image.Rect(0, 0, 480, 180)))
	if err := errors.Join(encoded, file.Close()); err != nil {
		t.Fatal(err)
	}
	if err := work.showImage(path); err != nil {
		t.Fatal(err)
	}
	view.SetNavigable(true)
	end := geometry.Pt(370, 150)
	view.Event(ctx, event.NewMouseEvent(event.MouseRelease, event.ButtonLeft, 0, end, end, event.ModNone))
	if commits != 0 || work.gesturing() || work.working.gesture != nil || work.state.document.Text() != source {
		t.Fatal("release committed a drag across photograph replacement")
	}
	aspect := float64(base.Width) / float64(base.Height)
	workingBounds := fitImageBounds(view.Bounds(), aspect)
	photoBounds := view.imageBounds()
	if workingBounds.Width() != 640 || workingBounds.Height() != 400 ||
		photoBounds.Width() != 640 || photoBounds.Height() != 240 {
		t.Fatalf("working view inherited photo framing: working=%v photo=%v", workingBounds, photoBounds)
	}
	view.setWorking(image.NewRGBA(image.Rect(0, 0, 639, 400)), aspect)
	if got := view.imageBounds(); got != workingBounds {
		t.Fatalf("working pixels were not displayed at their staged aspect: %v", got)
	}
}
