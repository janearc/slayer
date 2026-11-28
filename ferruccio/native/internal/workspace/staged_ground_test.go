//go:build cgo

package workspace

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/ferruccio/internal/draft"
	"github.com/janearc/slayer/ferruccio/native/internal/settings"
	"github.com/janearc/slayer/ferruccio/native/internal/stagewire"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/session/placement"
	"github.com/janearc/slayer/set"
)

func TestRetrySceneDetailAfterChoosingSourceShelf(t *testing.T) {
	root := t.TempDir()
	missingShelf, readyShelf := t.TempDir(), t.TempDir()
	supersededShelf, latestMissingShelf := t.TempDir(), t.TempDir()
	t.Setenv("SLAYER_SHELF", "")
	t.Setenv("SLAYER_CACHE", t.TempDir())
	raw, err := os.ReadFile("../../../../terrain/testdata/fixture-plain.tif")
	if err != nil {
		t.Fatal(err)
	}
	manifest := fmt.Sprintf("sources:\n  - key: fixture\n    from: https://x.test/fixture.tif\n    licence: test\n    sha256: %x\n", sha256.Sum256(raw))
	for path, content := range map[string][]byte{
		filepath.Join(root, "sources.yaml"):            []byte(manifest),
		filepath.Join(root, "scene.yaml"):              []byte(groundEditFixture),
		filepath.Join(readyShelf, "fixture.tif"):       raw,
		filepath.Join(readyShelf, "fixture-plain.tif"): raw,
	} {
		if err := os.WriteFile(path, content, 0600); err != nil {
			t.Fatal(err)
		}
	}
	path := filepath.Join(root, "scene.yaml")
	document, err := draft.Open(path, []byte(groundEditFixture))
	if err != nil {
		t.Fatal(err)
	}
	unsaved := strings.Replace(groundEditFixture, "east: 1050", "east: 1052", 1)
	document.Replace(unsaved)
	work := playWorkspace(t)
	defer work.closeStaged()
	work.state.document, work.state.path = document, path
	work.editor.SetText(unsaved)
	work.settings = SettingsSource{Path: filepath.Join(root, "config"), Flag: nothing, Environment: nothing}
	if err := settings.Save(work.settings.Path, settings.Shelf, missingShelf); err != nil {
		t.Fatal(err)
	}
	const stone = "scene.set.things/stone"
	previous := &placement.DisplayScene{Width: 40, Height: 40}
	work.working.scene, work.working.source, work.working.path = previous, unsaved, path
	work.working.mode = "solid"
	superseded := make(chan struct{}, 1)
	work.working.stageOverride = func(ctx context.Context, path, source, kind string) (*stagedView, error) {
		if source != unsaved {
			return nil, fmt.Errorf("staging received the wrong draft")
		}
		shelf := work.renderTools().Shelf
		if shelf == supersededShelf {
			superseded <- struct{}{}
			<-ctx.Done()
			return nil, ctx.Err()
		}
		if err := os.Setenv("SLAYER_SHELF", shelf); err != nil {
			return nil, err
		}
		snapshot, err := StageDetailSnapshot(ctx, path, source, kind, stagewire.Camera{}, stone)
		if err != nil {
			return nil, err
		}
		return &stagedView{scene: &snapshot.Scene}, nil
	}
	work.stageSceneDetail(stone, true)
	waitStaged(t, work)
	if work.working.scene != previous || !work.canRetryStaging() ||
		!strings.Contains(work.diagnostic, filepath.Join(missingShelf, "fixture-plain.tif")) {
		t.Fatalf("missing pinned data lost the map or retry: %q", work.diagnostic)
	}
	work.ui.Window().HandleResize(int(smallestWindow.Width), int(smallestWindow.Height))
	work.showWorkingDetails()
	_, retry := detailsPage(t, work, "", "Retry")
	clickLibraryPoint(work, retry)
	if !work.working.staging {
		t.Fatal("the Retry button did not start staging")
	}
	waitStaged(t, work)
	if work.working.scene != previous || !work.canRetryStaging() {
		t.Fatal("retry with the shelf still empty lost the map or Retry action")
	}
	work.play.selected = ""
	var cancelled bool
	work.cancel = func() { cancelled = true }
	work.pending.document = document
	work.saveSetting(settings.Shelf, supersededShelf)
	select {
	case <-superseded:
	case <-time.After(3 * time.Second):
		t.Fatal("detail retry did not start")
	}
	work.saveSetting(settings.Shelf, latestMissingShelf)
	waitStaged(t, work)
	if work.working.scene != previous || !work.canRetryStaging() ||
		!strings.Contains(work.diagnostic, filepath.Join(latestMissingShelf, "fixture-plain.tif")) {
		t.Fatalf("superseded retry lost the map or latest shelf: %q", work.diagnostic)
	}
	work.settingsOpen = true
	work.rebuild()
	sheet := work.root
	work.refreshLayout = false
	work.wantSettings.Store(false)
	work.saveSetting(settings.Shelf, readyShelf)
	if work.root != sheet || !work.wantSettings.Load() {
		t.Fatal("accepting the source shelf replaced the Settings sheet during the input callback")
	}
	if !work.refreshLayout {
		t.Fatal("the detail retry did not defer the layout refresh")
	}
	waitStaged(t, work)
	if work.working.scene == nil || work.working.scene == previous || work.working.detail != stone ||
		work.canRetryStaging() || !document.Matches(unsaved) {
		t.Fatalf("working view did not recover the same unsaved detail: %q", work.diagnostic)
	}
	origin, ok := work.working.scene.Origins[stone]
	const terrainWest = 1001
	if !ok || origin.X+terrainWest != 1052 {
		t.Fatalf("recovered stone easting = %g (found: %t), want 1052", origin.X+terrainWest, ok)
	}
	if work.state.document != document || work.editor.Text() != unsaved || !document.Dirty() {
		t.Fatal("retry lost the live document or its unsaved editor text")
	}
	if cancelled || work.pending.document != document {
		t.Fatal("detail retry interrupted rendering or cleared queued navigation")
	}
	saved, err := os.ReadFile(path)
	if err != nil || string(saved) != groundEditFixture {
		t.Fatalf("retry changed the saved document: %v", err)
	}
	work.Tick()
	if work.root == sheet || !work.settingsOpen {
		t.Fatal("the next tick did not refresh the existing Settings sheet")
	}
}

// TestGroundMoveStagesSavedTerrainHeight exercises source editing and the
// detached snapshot without drawing an offline render.
func TestGroundMoveStagesSavedTerrainHeight(t *testing.T) {
	root := t.TempDir()
	shelf := filepath.Join(root, "shelf")
	if err := os.MkdirAll(shelf, 0755); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile("../../../../terrain/testdata/fixture-plain.tif")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(shelf, "fixture-plain.tif"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(shelf, "fixture.tif"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(raw)
	manifest := fmt.Sprintf("sources:\n  - key: fixture\n    from: https://x.test/fixture.tif\n    licence: test\n    sha256: %x\n", sum)
	if err := os.WriteFile(filepath.Join(root, "sources.yaml"), []byte(manifest), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SLAYER_SHELF", shelf)
	t.Setenv("SLAYER_CACHE", t.TempDir())
	path := filepath.Join(root, "scene.yaml")
	if err := os.WriteFile(path, []byte(groundEditFixture), 0600); err != nil {
		t.Fatal(err)
	}
	document, err := draft.Open("scene.yaml", []byte(groundEditFixture))
	if err != nil {
		t.Fatal(err)
	}
	work := playWorkspace(t)
	work.state.document, work.state.path = document, path
	work.editor.SetText(document.Text())
	first, err := StageSnapshot(context.Background(), path, document.Text(), "set")
	if err != nil {
		t.Fatal(err)
	}
	var wire bytes.Buffer
	if err := stagewire.Write(&wire, first); err != nil {
		t.Fatal(err)
	}
	detached, err := stagewire.Read(&wire)
	if err != nil {
		t.Fatal(err)
	}
	work.working.scene, work.working.source, work.working.path, work.working.displayed = &detached.Scene, document.Text(), path, true
	work.play.selected = ""
	var hitX, hitY float64
	for y := .1; y <= .9 && work.play.selected == ""; y += .05 {
		for x := .1; x <= .9 && work.play.selected == ""; x += .05 {
			if work.viewer.selectObject(x, y) {
				hitX, hitY = x, y
			}
		}
	}
	if work.play.selected != "scene.set.things/stone" {
		centre, hit := detached.Scene.PickAt(.5, .5)
		for _, batch := range detached.Scene.Batches {
			if batch.Authored && batch.Author.Name == "stone" {
				low, high, _ := batch.Mesh.Bounds()
				t.Logf("stone geometry %v %v at %v", low, high, batch.Places[0].Origin)
			}
		}
		t.Fatalf("ground selection = %q; eye %v forward %v origin %v centre hit %+v %v", work.play.selected, detached.Scene.Eye.Origin, detached.Scene.Eye.Forward(), detached.Scene.Origins["scene.set.things/stone"], centre, hit)
	}
	object, ok := work.selectedObject()
	if !ok || !object.Grounded() {
		t.Fatal("authored ground object is not selectable")
	}
	before, err := gestureOrigin(&detached.Scene, object)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := set.Read(path, []byte(document.Text()))
	if err != nil {
		t.Fatal(err)
	}
	work.working.parsed = &parsed
	work.play.poseText = [4]string{"1052", "8988", "0", "30"}
	if err := work.submitCoordinates(true); err != nil {
		t.Fatal(err)
	}
	preview := work.working.preview
	if preview == nil {
		t.Fatal("typed terrain move did not retain a provisional scene")
	}
	matched := false
	for i, batch := range detached.Scene.Batches {
		if batch.Authored && batch.Author.Collection == "scene.set.things" && batch.Author.Name == "stone" && len(batch.Places) > 0 {
			moved := preview.scene.Batches[i].Places[0].Origin.Sub(batch.Places[0].Origin)
			if math.Abs(moved.X-2) > 1e-9 || math.Abs(moved.Z-2) > 1e-9 || math.Abs(moved.Y) > 1e-9 {
				t.Fatalf("provisional terrain move = %v", moved)
			}
			matched = true
			break
		}
	}
	if !matched {
		t.Fatal("staged terrain object has no authored geometry")
	}
	started := make(chan struct{}, 2)
	work.working.stageOverride = func(ctx context.Context, _, _, _ string) (*stagedView, error) {
		started <- struct{}{}
		<-ctx.Done()
		return nil, ctx.Err()
	}
	defer work.closeStaged()
	work.updateWorking()
	<-started
	work.play.poseText = [4]string{"1054", "8988", "0", "30"}
	if err := work.submitCoordinates(true); err != nil || work.working.preview == nil {
		t.Fatalf("second terrain edit while staging failed: %v", err)
	}
	work.updateWorking()
	<-started
	work.undoCamera(work.ui.Window().Context(), false)
	work.undoCamera(work.ui.Window().Context(), false)
	work.working.displayed = true
	work.beginWorking()
	if work.working.gesture == nil {
		t.Fatal("ground move did not begin")
	}
	east, north, err := detached.Scene.DragHeight(before.Y, hitX, hitY, hitX+.03, hitY+.03)
	if err != nil {
		t.Fatal(err)
	}
	work.viewer.changeObject(hitX, hitY, hitX+.03, hitY+.03)
	changed, ok := work.selectedObject()
	if !ok || math.Abs(changed.Position()[0]-(1050+east)) > 1e-8 || math.Abs(changed.Position()[1]-(8990+north)) > 1e-8 {
		t.Fatalf("ground pointer changed source by wrong amount: %+v", changed.Position())
	}
	work.endWorking(true)
	document.Replace(groundEditFixture)
	work.editor.SetText(document.Text())
	work.clearNavigation()
	work.working.scene, work.working.source, work.working.path, work.working.displayed = &detached.Scene, document.Text(), path, true
	work.working.waiting = false
	if err := work.changePose(2, -2, 0); err != nil {
		t.Fatal(err)
	}
	changedText := document.Text()
	work.undoCamera(work.ui.Window().Context(), false)
	if document.Text() != groundEditFixture {
		t.Fatal("undo did not restore the ground placement")
	}
	work.undoCamera(work.ui.Window().Context(), true)
	if document.Text() != changedText {
		t.Fatal("redo did not restore the ground move")
	}
	if err := document.Save(path); err != nil {
		t.Fatal(err)
	}
	updated, err := StageSnapshot(context.Background(), path, document.Text(), "set")
	if err != nil {
		t.Fatal(err)
	}
	after := updated.Scene.Origins[work.play.selected]
	delta := after.Sub(before)
	if math.Abs(delta.X-2) > 1e-9 || math.Abs(delta.Z-2) > 1e-9 || delta.Y == 0 {
		t.Fatalf("saved ground move = %v; terrain height must also change", delta)
	}
	findGeometry := func(snapshot stagewire.Snapshot) (maths.Subspace, bool) {
		for _, batch := range snapshot.Scene.Batches {
			if batch.Authored && batch.Author.Collection == "scene.set.things" && batch.Author.Name == "stone" {
				return batch.Places[0], true
			}
		}
		return maths.Subspace{}, false
	}
	oldPlace, oldOK := findGeometry(detached)
	newPlace, newOK := findGeometry(updated)
	if !oldOK || !newOK || math.Abs((newPlace.Origin.Y-oldPlace.Origin.Y)-delta.Y) > 1e-9 {
		t.Fatalf("staged ground geometry did not follow anchor height: %v → %v", oldPlace.Origin.Y, newPlace.Origin.Y)
	}
	if !strings.Contains(document.Text(), "heading:") {
		t.Fatal("move lost authored heading")
	}
	if err := work.changePose(0, 0, 15); err != nil {
		t.Fatal(err)
	}
	if err := document.Save(path); err != nil {
		t.Fatal(err)
	}
	turned, err := StageSnapshot(context.Background(), path, document.Text(), "set")
	if err != nil {
		t.Fatal(err)
	}
	turnedPlace, ok := findGeometry(turned)
	wantTurn := maths.Rotation(maths.YAxis, -15*maths.Degree).Then(newPlace.Orientation)
	if !ok || turnedPlace.Orientation.X.Sub(wantTurn.X).Len() > 1e-8 || turnedPlace.Orientation.Y.Sub(wantTurn.Y).Len() > 1e-8 || turnedPlace.Orientation.Z.Sub(wantTurn.Z).Len() > 1e-8 {
		t.Fatal("saved ground turn did not reach staged geometry")
	}
	generated := strings.Replace(groundEditFixture, ", heading: {value: 30, unit: deg}", "", 1)
	chosen, err := draft.Open(path, []byte(generated))
	if err != nil {
		t.Fatal(err)
	}
	if err := chosen.SetGeneratedHeading("scene.set.things/stone", 30); err != nil {
		t.Fatal(err)
	}
	chosenSnapshot, err := StageSnapshot(context.Background(), path, chosen.Text(), "set")
	if err != nil {
		t.Fatal(err)
	}
	chosenPlace, ok := findGeometry(chosenSnapshot)
	if !ok || chosenPlace.Orientation.X.Sub(oldPlace.Orientation.X).Len() > 1e-8 || chosenPlace.Orientation.Z.Sub(oldPlace.Orientation.Z).Len() > 1e-8 {
		t.Fatal("chosen heading did not reach the producer's staged geometry")
	}
}

func TestGroundGestureNeedsStagedAnchorAndExplicitTurn(t *testing.T) {
	source := strings.Replace(groundEditFixture, ", heading: {value: 30, unit: deg}", "", 1)
	document, err := draft.Open("ground.yaml", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	work := playWorkspace(t)
	work.state.document = document
	work.play.selected = "scene.set.things/stone"
	object, ok := work.selectedObject()
	if !ok || !object.Grounded() {
		t.Fatal("ground placement is unavailable")
	}
	if _, err := gestureOrigin(nil, object); err == nil {
		t.Fatal("missing terrain anchor was guessed")
	}
	if err := work.changePose(0, 0, 5); err == nil || !strings.Contains(err.Error(), "generated heading") {
		t.Fatalf("unheaded terrain turn = %v", err)
	}
	if document.Text() != source {
		t.Fatal("refused turn changed the document")
	}
}

// The region editor changes the producer's stand, with source history and
// saved geometry checked without an offline render.
func TestGroupRegionEditStagesAndUndoRedo(t *testing.T) {
	root := t.TempDir()
	shelf := filepath.Join(root, "shelf")
	if err := os.MkdirAll(shelf, 0755); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile("../../../../terrain/testdata/fixture-plain.tif")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"fixture-plain.tif", "fixture.tif"} {
		if err := os.WriteFile(filepath.Join(shelf, name), raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	sum := sha256.Sum256(raw)
	if err := os.WriteFile(filepath.Join(root, "sources.yaml"), []byte(fmt.Sprintf("sources:\n  - key: fixture\n    from: https://x.test/fixture.tif\n    licence: test\n    sha256: %x\n", sum)), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SLAYER_SHELF", shelf)
	t.Setenv("SLAYER_CACHE", t.TempDir())
	source := groundEditFixture + "    stands:\n      grove: {kind: yurt, count: 4, seed: 2, complexity: quick, habit: scatter, area: {west: 20, south: 20, east: 80, north: 80}}\n"
	path := filepath.Join(root, "scene.yaml")
	document, err := draft.Open(path, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	work := playWorkspace(t)
	work.state.document, work.state.path = document, path
	work.editor.SetText(source)
	work.play.selected = "scene.set.stands/grove"
	first, err := StageSnapshot(context.Background(), path, source, "set")
	if err != nil {
		t.Fatal(err)
	}
	position := func(snapshot stagewire.Snapshot) maths.Vec {
		for _, batch := range snapshot.Scene.Batches {
			if batch.Authored && batch.Author.Name == "grove" && len(batch.Places) > 0 {
				return batch.Places[0].Origin
			}
		}
		t.Fatal("no generated member")
		return maths.Vec{}
	}
	before := position(first)
	work.play.groupAreaText = [4]string{"100", "20", "160", "80"}
	if err := work.submitGroupArea(); err != nil {
		t.Fatal(err)
	}
	changed := document.Text()
	work.undoCamera(work.ui.Window().Context(), false)
	if document.Text() != source {
		t.Fatal("region did not undo")
	}
	work.undoCamera(work.ui.Window().Context(), true)
	if document.Text() != changed {
		t.Fatal("region did not redo")
	}
	if err := document.Save(path); err != nil {
		t.Fatal(err)
	}
	saved, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	after, err := StageSnapshot(context.Background(), path, string(saved), "set")
	if err != nil {
		t.Fatal(err)
	}
	moved := position(after)
	if math.Abs((moved.X-before.X)-80) > .001 {
		t.Fatalf("saved region did not move generated member east by 80m: %v -> %v", before, moved)
	}
}

func TestGroundThingScaleSavesAndStagesAtTerrainAnchor(t *testing.T) {
	root := t.TempDir()
	shelf := filepath.Join(root, "shelf")
	if err := os.MkdirAll(shelf, 0755); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile("../../../../terrain/testdata/fixture-plain.tif")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"fixture-plain.tif", "fixture.tif"} {
		if err := os.WriteFile(filepath.Join(shelf, name), raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	sum := sha256.Sum256(raw)
	manifest := fmt.Sprintf("sources:\n  - key: fixture\n    from: https://x.test/fixture.tif\n    licence: test\n    sha256: %x\n", sum)
	if err := os.WriteFile(filepath.Join(root, "sources.yaml"), []byte(manifest), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SLAYER_SHELF", shelf)
	t.Setenv("SLAYER_CACHE", t.TempDir())
	path := filepath.Join(root, "scene.yaml")
	document, err := draft.Open(path, []byte(groundEditFixture))
	if err != nil {
		t.Fatal(err)
	}
	work := playWorkspace(t)
	work.state.document, work.state.path = document, path
	work.editor.SetText(document.Text())
	work.play.selected = "scene.set.things/stone"
	first, err := StageSnapshot(context.Background(), path, document.Text(), "set")
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := set.Read(path, []byte(document.Text()))
	if err != nil {
		t.Fatal(err)
	}
	work.working.parsed, work.working.source, work.working.path = &parsed, document.Text(), path
	if !work.canScaleThing() {
		t.Fatal("ground yurt was not offered scale")
	}
	beforeText := document.Text()
	if err := work.changeThingScale(2); err != nil {
		t.Fatal(err)
	}
	afterText := document.Text()
	work.undoCamera(work.ui.Window().Context(), false)
	if document.Text() != beforeText {
		t.Fatal("scale undo failed")
	}
	work.undoCamera(work.ui.Window().Context(), true)
	if document.Text() != afterText {
		t.Fatal("scale redo failed")
	}
	if err := document.Save(path); err != nil {
		t.Fatal(err)
	}
	saved, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	second, err := StageSnapshot(context.Background(), path, string(saved), "set")
	if err != nil {
		t.Fatal(err)
	}
	get := func(snapshot stagewire.Snapshot) (maths.Vec, maths.Vec) {
		anchor := snapshot.Scene.Origins["scene.set.things/stone"]
		for _, batch := range snapshot.Scene.Batches {
			if batch.Authored && batch.Author.Name == "stone" && len(batch.Places) > 0 && len(batch.Mesh.Vertices) > 0 {
				return anchor, batch.Places[0].Out(batch.Mesh.Vertices[0].Position)
			}
		}
		t.Fatal("missing stone geometry")
		return maths.Vec{}, maths.Vec{}
	}
	anchor, a := get(first)
	newAnchor, b := get(second)
	if anchor != newAnchor || b.Sub(anchor).Sub(a.Sub(anchor).Scale(2)).Len() > 1e-6 {
		t.Fatalf("terrain scale anchor %v -> %v, vertex %v -> %v", anchor, newAnchor, a, b)
	}
	for _, one := range []struct{ name, replacement string }{{"burning", "kind: yurt, alight: true, scale: 2"}, {"terrain-founded village", "kind: village, scale: 2"}} {
		bad := strings.Replace(groundEditFixture, "kind: yurt", one.replacement, 1)
		if _, err := StageSnapshot(context.Background(), path, bad, "set"); err == nil || !strings.Contains(err.Error(), "scale is not supported") {
			t.Fatalf("%s scaling was not explicitly refused: %v", one.name, err)
		}
	}
}

func TestAddGroundThingSavesAndStagesOnTerrain(t *testing.T) {
	root := t.TempDir()
	shelf := filepath.Join(root, "shelf")
	if err := os.MkdirAll(shelf, 0755); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile("../../../../terrain/testdata/fixture-plain.tif")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"fixture-plain.tif", "fixture.tif"} {
		if err := os.WriteFile(filepath.Join(shelf, name), raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	sum := sha256.Sum256(raw)
	manifest := fmt.Sprintf("sources:\n  - key: fixture\n    from: https://x.test/fixture.tif\n    licence: test\n    sha256: %x\n", sum)
	if err := os.WriteFile(filepath.Join(root, "sources.yaml"), []byte(manifest), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SLAYER_SHELF", shelf)
	t.Setenv("SLAYER_CACHE", t.TempDir())
	path := filepath.Join(root, "scene.yaml")
	implicit := strings.Replace(groundEditFixture, "  setting: ground\n", "", 1)
	document, err := draft.Open(path, []byte(implicit))
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := set.Read(path, []byte(document.Text()))
	if err != nil {
		t.Fatal(err)
	}
	work := playWorkspace(t)
	work.state.document, work.state.path = document, path
	work.editor.SetText(document.Text())
	work.play.selected = ""
	work.working.parsed, work.working.source, work.working.path = &parsed, document.Text(), path
	work.play.selected = "scene.set.things/stone"
	work.defaultTerrainAddition()
	if work.play.addAt != [3]string{"1050", "8990", "0"} {
		t.Fatalf("selected-object default %v", work.play.addAt)
	}
	work.play.selected = ""
	work.toggleDisclosure("add")
	if work.play.addAt != [3]string{"1076", "8949", "0"} {
		t.Fatalf("map-centre default %v", work.play.addAt)
	}
	before := document.Text()
	if err := work.addChosen("new-yurt", addChoice{field: "kind", name: "yurt"}, "5", [3]string{"5000", "5000", "0"}); err == nil || document.Text() != before {
		t.Fatal("out-of-terrain addition accepted")
	}
	if err := work.addChosen("new-yurt", addChoice{field: "kind", name: "yurt"}, "5", [3]string{"1076", "8949", "-1"}); err == nil || document.Text() != before {
		t.Fatal("negative ground height accepted")
	}
	if err := work.addChosen("new-yurt", addChoice{field: "kind", name: "yurt"}, "5", [3]string{"1076", "8949", "2"}); err != nil {
		t.Fatal(err)
	}
	changed := document.Text()
	work.undoCamera(work.ui.Window().Context(), false)
	if document.Text() != before {
		t.Fatal("addition undo failed")
	}
	work.undoCamera(work.ui.Window().Context(), true)
	if document.Text() != changed {
		t.Fatal("addition redo failed")
	}
	if err := document.Save(path); err != nil {
		t.Fatal(err)
	}
	saved, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	read, err := set.Read(path, saved)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, one := range read.Things() {
		if one.Name() == "new-yurt" {
			heading, said := one.Heading()
			if !said || heading != 0 || one.Height() != 2 {
				t.Fatal("ground placement lost heading or height")
			}
			found = true
		}
	}
	if !found {
		t.Fatal("saved set lost addition")
	}
	snapshot, err := StageSnapshot(context.Background(), path, string(saved), "set")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := snapshot.Scene.Origins["scene.set.things/new-yurt"]; !ok {
		t.Fatal("new object did not stage on terrain")
	}
	choices, err := addChoices(document, "fern")
	if err != nil {
		t.Fatal(err)
	}
	var chosen addChoice
	for _, choice := range choices {
		if choice.name == "fern" && choice.field == "plant" {
			chosen = choice
			break
		}
	}
	if chosen.profile == nil {
		t.Fatal("fern profile missing")
	}
	_, book, err := manufacture.EmbeddedPlantBook("fern")
	if err != nil {
		t.Fatal(err)
	}
	bookPath := filepath.Join(root, "fern.yaml")
	if err := os.WriteFile(bookPath, book, 0600); err != nil {
		t.Fatal(err)
	}
	work.working.parsed, work.working.source, work.working.path = &read, document.Text(), path
	if err := work.addProfile("new-fern", chosen, bookPath, "6", [3]string{"1078", "8949", "0"}); err != nil {
		t.Fatal(err)
	}
	if err := document.Save(path); err != nil {
		t.Fatal(err)
	}
	saved, err = os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err = StageSnapshot(context.Background(), path, string(saved), "set")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := snapshot.Scene.Origins["scene.set.things/new-fern"]; !ok {
		t.Fatal("new fern did not stage on terrain")
	}
	read, err = set.Read(path, saved)
	if err != nil {
		t.Fatal(err)
	}
	work.working.parsed, work.working.source, work.working.path = &read, document.Text(), path
	work.play.selected = "scene.set.things/new-yurt"
	beforeCoordinates := document.Text()
	work.play.poseText = [4]string{"5000", "8948", "3", "45"}
	if err := work.submitCoordinates(true); err == nil || document.Text() != beforeCoordinates {
		t.Fatal("out-of-terrain coordinates changed source")
	}
	work.play.poseText = [4]string{"1078", "8948", "3", "45"}
	if err := work.submitCoordinates(true); err != nil {
		t.Fatal(err)
	}
	afterCoordinates := document.Text()
	work.undoCamera(work.ui.Window().Context(), false)
	if document.Text() != beforeCoordinates {
		t.Fatal("coordinates undo failed")
	}
	work.undoCamera(work.ui.Window().Context(), true)
	if document.Text() != afterCoordinates {
		t.Fatal("coordinates redo failed")
	}
	if err := document.Save(path); err != nil {
		t.Fatal(err)
	}
	saved, err = os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	read, err = set.Read(path, saved)
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range read.Things() {
		if one.Name() == "new-yurt" {
			east, north := one.At()
			heading, said := one.Heading()
			if east != 1078 || north != 8948 || one.Height() != 3 || !said || heading != 45 {
				t.Fatalf("saved coordinates %v %v %v %v", east, north, one.Height(), heading)
			}
		}
	}
	if _, err := StageSnapshot(context.Background(), path, string(saved), "set"); err != nil {
		t.Fatal(err)
	}
}

func TestTypedGroundPlacementsSaveAndRestageAbsoluteUp(t *testing.T) {
	for _, route := range []struct {
		name, light string
		origin      maths.Vec
	}{
		{"terrain", "  light: moon\n  timeofday: 2200\n  terrain: {from: fixture, file: fixture-plain.tif, west: 1001, north: 8999, across: 150, down: 100, step: 2}", maths.V(49, 0, 9)},
		{"groundless Earth", "  light: moon\n  moon: {azimuth: 56, altitude: 10}\n  air: earth\n  groundless-air: {coordinate-origin: earth-mean-radius, reference-altitude: {value: 12, unit: m}, boundary: earth-standard-land}", maths.V(1050, 12, -8990)},
	} {
		t.Run(route.name, func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv("SLAYER_CACHE", t.TempDir())
			t.Setenv("SLAYER_SHELF", dir)
			if route.name == "terrain" {
				raw, err := os.ReadFile("../../../../terrain/testdata/fixture-plain.tif")
				if err != nil {
					t.Fatal(err)
				}
				manifest := fmt.Sprintf("sources:\n  - key: fixture\n    from: https://x.test/fixture.tif\n    licence: test\n    sha256: %x\n", sha256.Sum256(raw))
				for name, contents := range map[string][]byte{"sources.yaml": []byte(manifest), "fixture.tif": raw, "fixture-plain.tif": raw} {
					if err := os.WriteFile(filepath.Join(dir, name), contents, 0600); err != nil {
						t.Fatal(err)
					}
				}
			}
			source := strings.Replace(groundEditFixture,
				"  light: moon\n  timeofday: 2200\n  terrain: {from: fixture, file: fixture-plain.tif, west: 1001, north: 8999, across: 150, down: 100, step: 2}",
				route.light, 1)
			source = strings.Replace(source, "east: 1050, north: 8990",
				"position: {east: {value: 1050, unit: m}, north: {value: 8990, unit: m}, up: {value: 12, unit: m}}", 1)
			path := filepath.Join(dir, "scene.yaml")
			if err := os.WriteFile(path, []byte(source), 0600); err != nil {
				t.Fatal(err)
			}
			document, err := draft.Open(path, []byte(source))
			if err != nil {
				t.Fatal(err)
			}
			before, err := StageSnapshot(context.Background(), path, source, "set")
			if err != nil {
				t.Fatal(err)
			}
			const id = "scene.set.things/stone"
			anchor, ok := before.Scene.Origins[id]
			if !ok || anchor.X != route.origin.X || anchor.Z != route.origin.Z || (route.name != "terrain" && anchor.Y != route.origin.Y) {
				t.Fatalf("unexpected producer anchor %v", anchor)
			}
			work := playWorkspace(t)
			work.state.document, work.state.path = document, path
			work.editor.SetText(source)
			work.working.scene, work.working.source, work.working.path, work.working.displayed = &before.Scene, source, path, true
			work.selectWorkingObject(id)
			object, ok := work.selectedObject()
			if !ok || !object.Grounded() || !object.AbsolutePosition() || object.ScaleEditable() {
				t.Fatalf("unexpected absolute ground selection %+v", object)
			}
			if readout := work.playReadout(); !strings.Contains(readout, "Up 12.00 m") || strings.Contains(readout, "above ground") {
				t.Fatalf("readout uses the wrong height reference: %q", readout)
			}
			if route.name == "terrain" {
				plan, shown, err := document.AuthoredMap()
				if err != nil || !shown || len(plan.Marks) != 1 || plan.Marks[0].ID != id || plan.Marks[0].East != 1050 || plan.Marks[0].North != 8990 {
					t.Fatalf("typed ground placement missing from authored map: %v, %t, %v", plan.Marks, shown, err)
				}
			}
			wantEast, wantNorth, err := before.Scene.DragViewDepth(anchor, .5, .5, .55, .52)
			if err != nil {
				t.Fatal(err)
			}
			east, north, err := work.dragPlacement(&before.Scene, .5, .5, .55, .52)
			if err != nil || east != wantEast || north != wantNorth {
				t.Fatalf("absolute pointer delta %g east, %g north (%v), want %g east, %g north", east, north, err, wantEast, wantNorth)
			}
			work.play.poseText = [4]string{"1052", "8988", "-3", "55"}
			if err := work.submitCoordinates(true); err != nil {
				t.Fatal(err)
			}
			wantShift := maths.V(2, -15, 2)
			if preview := work.working.preview; preview == nil || preview.scene.Origins[id] != anchor.Add(wantShift) {
				t.Fatal("preview lost the staged absolute anchor")
			}
			if err := document.Save(path); err != nil {
				t.Fatal(err)
			}
			saved, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			parsed, err := set.Read(path, saved)
			if err != nil {
				t.Fatal(err)
			}
			stone := parsed.Things()[0]
			position, absolute := stone.Position()
			heading, turned := stone.Heading()
			if !absolute || position.East() != 1052 || position.North() != 8988 || position.Up() != -3 || !turned || heading != 55 {
				t.Fatalf("saved position %v, heading %v", position, heading)
			}
			after, err := StageSnapshot(context.Background(), path, string(saved), "set")
			if err != nil {
				t.Fatal(err)
			}
			if got := after.Scene.Origins[id].Sub(anchor); got.Sub(wantShift).Len() > 1e-9 {
				t.Fatalf("producer staged shift %v, want %v", got, wantShift)
			}
			findPlace := func(scene placement.DisplayScene) maths.Subspace {
				for _, batch := range scene.Batches {
					if batch.Authored && batch.Author.Collection == "scene.set.things" && batch.Author.Name == "stone" && len(batch.Places) > 0 {
						return batch.Places[0]
					}
				}
				t.Fatal("absolute ground placement lost its authored geometry")
				return maths.Subspace{}
			}
			original, turnedPlace := findPlace(before.Scene), findPlace(after.Scene)
			wantTurn := maths.Rotation(maths.YAxis, -25*maths.Degree).Then(original.Orientation)
			if wantTurn.X.Sub(turnedPlace.Orientation.X).Len() > 1e-9 || wantTurn.Z.Sub(turnedPlace.Orientation.Z).Len() > 1e-9 {
				t.Fatal("saved heading did not reach producer geometry")
			}
			work.undoCamera(work.ui.Window().Context(), false)
			if document.Text() != source || work.editor.Text() != source {
				t.Fatal("undo did not restore the exact absolute ground source")
			}
		})
	}
}
