//go:build cgo

package workspace

import (
	"bytes"
	"context"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/janearc/slayer/ferruccio/internal/draft"
	"github.com/janearc/slayer/ferruccio/internal/preview"
	"github.com/janearc/slayer/ferruccio/native/internal/stagewire"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/smoke"
	"github.com/janearc/slayer/session/placement"
	"github.com/janearc/slayer/set"
)

const stagedTestSet = `scene:
  film: {width: 32, height: 24, rays: 1, wide: 45, exposure: 1}
  cameras:
    view:
      location: {at: [0, -5, 2]}
      orientation: {look_at: [1, 0, 0]}
  world:
    materials:
      matte: {colour: {srgb_linear: [0.3, 0.4, 0.5]}, source: eye}
    boxes:
      first: {material: matte, middle: [10, 20, 3], size: [2, 4, 6], source: eye}
`

func TestStageWorkingUsesUnsavedSetAndOwnsSession(t *testing.T) {
	t.Setenv("SLAYER_CACHE", t.TempDir())
	path := filepath.Join(t.TempDir(), "scene.yaml")
	if err := os.WriteFile(path, []byte(stagedTestSet), 0600); err != nil {
		t.Fatal(err)
	}
	changed := strings.Replace(stagedTestSet, "[2, 4, 6]", "[6, 4, 6]", 1)
	view, err := stageWorking(context.Background(), path, changed, "legacy-set")
	if err != nil {
		t.Fatal(err)
	}
	defer view.close()
	if view.owner == nil || len(view.scene.Objects) != 1 || view.scene.Objects[0].ID != "scene.world" {
		t.Fatal("staged world has unsafe identity or no owner")
	}
	low, high, has := view.scene.Objects[0].Mesh.Bounds()
	if !has || low.X != 7 || high.X != 13 {
		t.Fatalf("bounds %v %v %v", low, high, has)
	}
	if !strings.Contains(view.notice, "not individually editable") {
		t.Fatalf("missing editability notice: %q", view.notice)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != stagedTestSet {
		t.Fatalf("saved set changed: %v", err)
	}
	if err := view.close(); err != nil || view.owner != nil {
		t.Fatalf("close failed: %v", err)
	}
}

func TestAddedProfileIsConsumedByStagedSavedScene(t *testing.T) {
	for _, test := range []struct{ name, field, family string }{{"pacific-bluefin-tuna", "animal", "fish"}, {"fern", "plant", "fern"}} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("SLAYER_CACHE", t.TempDir())
			work := libraryEventWorkspace(t)
			path := filepath.Join(t.TempDir(), "scene.yaml")
			document, err := draft.Open(path, []byte(addRecipeFixture))
			if err != nil {
				t.Fatal(err)
			}
			work.state.document, work.state.path = document, path
			work.editor.SetText(addRecipeFixture)
			choices, err := addChoices(document, test.name)
			if err != nil {
				t.Fatal(err)
			}
			var chosen addChoice
			for _, candidate := range choices {
				if candidate.name == test.name && candidate.field == test.field {
					chosen = candidate
					break
				}
			}
			if chosen.profile == nil {
				t.Fatal("catalogue profile missing")
			}
			var raw []byte
			if test.field == "animal" {
				_, raw, err = manufacture.EmbeddedAnimalBook(test.family)
			} else {
				_, raw, err = manufacture.EmbeddedPlantBook(test.family)
			}
			if err != nil {
				t.Fatal(err)
			}
			book := filepath.Join(t.TempDir(), "book.yaml")
			if err := os.WriteFile(book, raw, 0600); err != nil {
				t.Fatal(err)
			}
			before, err := stageWorking(context.Background(), path, addRecipeFixture, "set")
			if err != nil {
				t.Fatal(err)
			}
			base := len(before.scene.Objects[0].Mesh.Vertices)
			if err := before.close(); err != nil {
				t.Fatal(err)
			}
			if err := work.addProfile("new-profile", chosen, book, "5", [3]string{"10", "2", "1"}); err != nil {
				t.Fatal(err)
			}
			if err := document.Save(path); err != nil {
				t.Fatal(err)
			}
			saved, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			view, err := stageWorking(context.Background(), path, string(saved), "set")
			if err != nil {
				t.Fatal(err)
			}
			defer view.close()
			if len(view.scene.Objects[0].Mesh.Vertices) <= base {
				t.Fatal("saved profile did not add staged geometry")
			}
		})
	}
}

func TestScaledProfilesSurviveSaveAndReachStagedGeometry(t *testing.T) {
	for _, test := range []struct{ name, field, family string }{{"pacific-bluefin-tuna", "animal", "fish"}, {"fern", "plant", "fern"}} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("SLAYER_CACHE", t.TempDir())
			source := strings.Replace(addRecipeFixture, "  set:\n    things:\n      first: {recipe: low, seed: 3, complexity: quick, east: 30, north: 0, height: {value: 0, unit: m}}\n", "", 1)
			work := libraryEventWorkspace(t)
			path := filepath.Join(t.TempDir(), "scene.yaml")
			document, err := draft.Open(path, []byte(source))
			if err != nil {
				t.Fatal(err)
			}
			work.state.document, work.state.path = document, path
			work.editor.SetText(source)
			choices, err := addChoices(document, test.name)
			if err != nil {
				t.Fatal(err)
			}
			var chosen addChoice
			for _, candidate := range choices {
				if candidate.name == test.name && candidate.field == test.field {
					chosen = candidate
					break
				}
			}
			var raw []byte
			if test.field == "animal" {
				_, raw, err = manufacture.EmbeddedAnimalBook(test.family)
			} else {
				_, raw, err = manufacture.EmbeddedPlantBook(test.family)
			}
			if err != nil {
				t.Fatal(err)
			}
			book := filepath.Join(t.TempDir(), "book.yaml")
			if err := os.WriteFile(book, raw, 0600); err != nil {
				t.Fatal(err)
			}
			if err := work.addProfile("specimen", chosen, book, "5", [3]string{"10", "2", "1"}); err != nil {
				t.Fatal(err)
			}
			before, err := stageWorking(context.Background(), path, document.Text(), "set")
			if err != nil {
				t.Fatal(err)
			}
			low, high, ok := before.scene.Objects[0].Mesh.Bounds()
			if !ok {
				t.Fatal("unscaled profile has no bounds")
			}
			if err := before.close(); err != nil {
				t.Fatal(err)
			}
			work.updateWorking()
			if err := work.changeThingScale(2); err != nil {
				t.Fatal(err)
			}
			if err := document.Save(path); err != nil {
				t.Fatal(err)
			}
			saved, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			after, err := stageWorking(context.Background(), path, string(saved), "set")
			if err != nil {
				t.Fatal(err)
			}
			defer after.close()
			newLow, newHigh, ok := after.scene.Objects[0].Mesh.Bounds()
			origin := maths.V(10, 1, -2)
			if !ok || newLow.Sub(origin).Sub(low.Sub(origin).Scale(2)).Len() > 1e-7 || newHigh.Sub(origin).Sub(high.Sub(origin).Scale(2)).Len() > 1e-7 {
				t.Fatalf("saved scaled bounds = %v %v; base %v %v", newLow, newHigh, low, high)
			}
		})
	}
}

func TestAddedConstructorIsConsumedByStagedSavedScene(t *testing.T) {
	t.Setenv("SLAYER_CACHE", t.TempDir())
	path := filepath.Join(t.TempDir(), "scene.yaml")
	document, err := draft.Open(path, []byte(addRecipeFixture))
	if err != nil {
		t.Fatal(err)
	}
	before, err := stageWorking(context.Background(), path, addRecipeFixture, "set")
	if err != nil {
		t.Fatal(err)
	}
	baseVertices := len(before.scene.Objects[0].Mesh.Vertices)
	if err := before.close(); err != nil {
		t.Fatal(err)
	}
	if err := document.AddKindThing("house", "city-rowhouse", 7, [3]float64{20, 4, 1}); err != nil {
		t.Fatal(err)
	}
	if err := document.Save(path); err != nil {
		t.Fatal(err)
	}
	saved, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	view, err := stageWorking(context.Background(), path, string(saved), "set")
	if err != nil {
		t.Fatal(err)
	}
	defer view.close()
	if len(view.scene.Objects) != 1 || len(view.scene.Objects[0].Mesh.Vertices) <= baseVertices {
		t.Fatal("saved construction did not add geometry to the staged world")
	}
}

func TestStageSnapshotDetachesCurrentBuffer(t *testing.T) {
	t.Setenv("SLAYER_CACHE", t.TempDir())
	path := filepath.Join(t.TempDir(), "scene.yaml")
	changed := strings.Replace(stagedTestSet, "[2, 4, 6]", "[6, 4, 6]", 1)
	snapshot, err := StageSnapshot(context.Background(), path, changed, "legacy-set")
	if err != nil {
		t.Fatal(err)
	}
	var encoded bytes.Buffer
	if err := stagewire.Write(&encoded, snapshot); err != nil {
		t.Fatal(err)
	}
	decoded, err := stagewire.Read(&encoded)
	if err != nil {
		t.Fatal(err)
	}
	batch := decoded.Scene.Batches[0]
	low, high, ok := batch.Mesh.Bounds()
	low, high = batch.Places[0].Out(low), batch.Places[0].Out(high)
	if !ok || low.X != 7 || high.X != 13 || !strings.Contains(decoded.Notice, "generated members and proxies are read-only") {
		t.Fatalf("detached view = %v %v %q", low, high, decoded.Notice)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("staging wrote the unsaved buffer")
	}
}

func TestStreamedStagedThingCanBeSelectedAndMoved(t *testing.T) {
	t.Setenv("SLAYER_CACHE", t.TempDir())
	work := playWorkspace(t)
	source, path := work.state.document.Text(), work.state.path
	snapshot, err := StageSnapshot(context.Background(), path, source, "set")
	if err != nil {
		t.Fatal(err)
	}
	var wire bytes.Buffer
	if err := stagewire.Write(&wire, snapshot); err != nil {
		t.Fatal(err)
	}
	loaded, err := stagewire.Read(&wire)
	if err != nil {
		t.Fatal(err)
	}
	work.working.scene, work.working.source, work.working.path, work.working.displayed = &loaded.Scene, source, path, true
	work.play.selected = ""
	if !work.viewer.selectObject(.5, .5) || work.play.selected != "scene.set.things/larson" {
		t.Fatalf("staged selection = %q", work.play.selected)
	}
	if err := work.changePose(1, 0, 0); err != nil {
		t.Fatal(err)
	}
	object, ok := work.selectedObject()
	if !ok || object.Position()[0] != 1 || work.state.document.Text() == source {
		t.Fatal("staged selection did not edit the authored placement")
	}
	if err := work.state.document.Save(path); err != nil {
		t.Fatal(err)
	}
	saved, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	updated, err := StageSnapshot(context.Background(), path, string(saved), "set")
	if err != nil {
		t.Fatal(err)
	}
	find := func(scene placement.DisplayScene) maths.Vec {
		for _, batch := range scene.Batches {
			if batch.Authored && batch.Author.Collection == "scene.set.things" && batch.Author.Name == "larson" {
				return batch.Places[0].Origin
			}
		}
		t.Fatal("saved Larson geometry was not staged")
		return maths.Vec{}
	}
	if delta := find(updated.Scene).Sub(find(loaded.Scene)); delta.Sub(maths.V(1, 0, 0)).Len() > 1e-9 {
		t.Fatalf("saved staged movement = %v", delta)
	}
}

func TestStagedGeneratedMemberSelectsItsReadOnlyStand(t *testing.T) {
	t.Setenv("SLAYER_CACHE", t.TempDir())
	source := `$schema: https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: Group selection
  setting: space
  light: none
  film: {width: 320, height: 200, rays: 1, wide: 45, exposure: 1}
  cameras:
    view: {orientation: {look_at: [0, 0, 0]}, location: {at: [0, -90, 2]}}
  set:
    things:
      ger: {kind: yurt, seed: 1, complexity: quick, east: 0, north: 0, height: {value: 0, unit: m}}
    stands:
      district: {kind: yurt, count: 20, seed: 2, complexity: quick, within: {of: ger, from: 0, to: 360, near: 8, far: 50}}
`
	path := filepath.Join(t.TempDir(), "groups.yaml")
	document, err := draft.Open(path, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := StageSnapshot(context.Background(), path, source, "set")
	if err != nil {
		t.Fatal(err)
	}
	var centre maths.Vec
	found := false
	for _, batch := range snapshot.Scene.Batches {
		if batch.Authored && batch.Author.Collection == "scene.set.stands" && batch.Author.Name == "district" {
			low, high, ok := batch.Mesh.Bounds()
			if ok {
				centre = batch.Places[0].Out(low.Add(high).Scale(.5))
				found = true
				break
			}
		}
	}
	if !found {
		t.Fatal("real stand has no shared authored batch")
	}
	eye, err := maths.View(centre.Add(maths.V(0, 3, 10)), centre, maths.YAxis)
	if err != nil {
		t.Fatal(err)
	}
	snapshot.Scene.Eye = *eye
	work := playWorkspace(t)
	work.state.document, work.state.path = document, path
	work.working.scene, work.working.source, work.working.path, work.working.displayed = &snapshot.Scene, source, path, true
	if !work.viewer.selectObject(.5, .5) || work.play.selected != "scene.set.stands/district" {
		t.Fatalf("real stand member selected %q", work.play.selected)
	}
	if err := work.changePose(1, 0, 0); err == nil || document.Text() != source {
		t.Fatal("real generated stand was treated as an editable individual")
	}
}

func TestStreamedStagedPointerPreviewsAndCommitsMove(t *testing.T) {
	t.Setenv("SLAYER_CACHE", t.TempDir())
	work := wholePlayRoot(t)
	work.setPlayMode("move")
	drawWholePlayRoot(t, work)
	source, path := work.state.document.Text(), work.state.path
	snapshot, err := StageSnapshot(context.Background(), path, source, "set")
	if err != nil {
		t.Fatal(err)
	}
	work.working.scene, work.working.source, work.working.path, work.working.displayed = &snapshot.Scene, source, path, true
	index := -1
	for i, batch := range snapshot.Scene.Batches {
		if batch.Authored && batch.Author.Collection == "scene.set.things" && batch.Author.Name == "larson" {
			index = i
			break
		}
	}
	if index < 0 {
		t.Fatal("staged Larson has no authored batch")
	}
	before := snapshot.Scene.Batches[index].Places[0].Origin
	beforeAnchor := snapshot.Scene.Origins["scene.set.things/larson"]
	bounds := work.viewer.ScreenBounds()
	start := geometry.Pt((bounds.Min.X+bounds.Max.X)/2, (bounds.Min.Y+bounds.Max.Y)/2)
	end := geometry.Pt(start.X+100, start.Y+40)
	work.ui.HandleEvent(event.NewMouseEvent(event.MousePress, event.ButtonLeft, event.ButtonStateLeft, start, start, event.ModNone))
	if !work.viewer.dragging || !work.viewer.dragObject {
		t.Fatal("pointer did not select the staged authored object")
	}
	work.ui.HandleEvent(event.NewMouseEvent(event.MouseMove, event.ButtonNone, event.ButtonStateLeft, end, end, event.ModNone))
	if work.state.document.Text() != source || work.working.scene.Batches[index].Places[0].Origin == before {
		t.Fatal("drag did not preview the staged batch without editing source")
	}
	shift := work.working.scene.Batches[index].Places[0].Origin.Sub(before)
	if work.working.scene.Origins["scene.set.things/larson"].Sub(beforeAnchor.Add(shift)).Len() > 1e-8 || snapshot.Scene.Origins["scene.set.things/larson"] != beforeAnchor {
		t.Fatal("drag did not update its provisional anchor independently")
	}
	work.ui.HandleEvent(event.NewMouseEvent(event.MouseRelease, event.ButtonLeft, 0, end, end, event.ModNone))
	if work.state.document.Text() == source {
		t.Fatal("release did not commit the staged object's move")
	}
	changed := work.state.document.Text()
	started := make(chan struct{})
	work.requestStaged(func(ctx context.Context, _, _, _ string) (*stagedView, error) {
		close(started)
		<-ctx.Done()
		return nil, ctx.Err()
	}, false)
	defer work.closeStaged()
	<-started
	if !work.working.provisional || work.working.scene == nil || work.workingCurrent() || work.viewer.navigable {
		t.Fatal("the committed pointer move was not retained as a non-selectable provisional view")
	}
	if work.working.scene.Batches[index].Places[0].Origin == before {
		t.Fatal("the provisional view lost the pointer move")
	}
	work.undoCamera(work.ui.Window().Context(), false)
	if work.state.document.Text() != source {
		t.Fatal("one undo did not restore the staged move")
	}
	work.undoCamera(work.ui.Window().Context(), true)
	if work.state.document.Text() != changed {
		t.Fatal("redo did not restore the staged move")
	}
}

func TestStreamedTurnPreviewMatchesRestagedGeometry(t *testing.T) {
	t.Setenv("SLAYER_CACHE", t.TempDir())
	work := wholePlayRoot(t)
	work.setPlayMode("turn")
	drawWholePlayRoot(t, work)
	source, path := work.state.document.Text(), work.state.path
	snapshot, err := StageSnapshot(context.Background(), path, source, "set")
	if err != nil {
		t.Fatal(err)
	}
	work.working.scene, work.working.source, work.working.path, work.working.displayed = &snapshot.Scene, source, path, true
	work.play.selected = "scene.set.things/larson"
	index := -1
	for i, batch := range snapshot.Scene.Batches {
		if batch.Authored && batch.Author.Collection == "scene.set.things" && batch.Author.Name == "larson" {
			index = i
			break
		}
	}
	if index < 0 {
		t.Fatal("staged Larson has no authored batch")
	}
	work.beginWorking()
	work.viewer.dragObject = true
	work.viewer.dragImage = geometry.Pt(.5, .5)
	work.viewer.dragStart = geometry.Pt(0, 0)
	work.viewer.dragCurrent = geometry.Pt(work.viewer.imageBounds().Width()*.15, 0)
	work.moveWorking()
	held := work.working.scene.Batches[index].Places[0]
	// The held and released events use the same pixel displacement.
	x1 := .5 + float64(work.viewer.dragCurrent.X)/float64(work.viewer.imageBounds().Width())
	work.viewer.changeObject(.5, .5, x1, .5)
	if work.state.document.Text() == source {
		t.Fatal("turn did not change the source")
	}
	work.endWorking(true)
	updated, err := StageSnapshot(context.Background(), path, work.state.document.Text(), "set")
	if err != nil {
		t.Fatal(err)
	}
	got := updated.Scene.Batches[index].Places[0]
	if got.Origin.Sub(held.Origin).Len() > 1e-8 || got.Orientation.X.Sub(held.Orientation.X).Len() > 1e-8 || got.Orientation.Y.Sub(held.Orientation.Y).Len() > 1e-8 || got.Orientation.Z.Sub(held.Orientation.Z).Len() > 1e-8 {
		t.Fatalf("turn preview differs from restaged geometry: held=%+v restaged=%+v", held, got)
	}
}

func TestFishPlantAndCityAuthoringReachesSharedStaging(t *testing.T) {
	if os.Getenv("FERRUCCIO_CITY_INTEGRATION") != "1" {
		t.Skip("set FERRUCCIO_CITY_INTEGRATION=1 for the full city staging test")
	}
	t.Setenv("SLAYER_CACHE", t.TempDir())
	dir := os.Getenv("FERRUCCIO_ACCEPTANCE_DIR")
	if dir == "" {
		dir = t.TempDir()
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "scene.yaml")
	work := libraryEventWorkspace(t)
	source := strings.Replace(addRecipeFixture, "  set:\n    things:\n      first: {recipe: low, seed: 3, complexity: quick, east: 30, north: 0, height: {value: 0, unit: m}}\n", "", 1) + `    specimens:
      location: {at: [17, -16, 9]}
      orientation: {look_at: [17, 4, 3]}
    city-overview:
      location: {at: [10600, -3300, 2800]}
      orientation: {look_at: [8002, 1, 0]}
  world:
    unit: 1
    sky:
      horizon: {srgb_linear: [0.35, 0.43, 0.55]}
      zenith: {srgb_linear: [0.12, 0.22, 0.4]}
      ground: {srgb_linear: [0.08, 0.09, 0.07]}
      source: eye
    suns:
      sun:
        toward: [-4, -5, 7]
        colour: {srgb_linear: [3, 2.8, 2.4]}
        source: eye
`
	document, err := draft.Open(path, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	work.state.document, work.state.path = document, path
	work.editor.SetText(source)
	for _, item := range []struct {
		name, profile, field, family string
		at                           [3]string
	}{{"tuna", "pacific-bluefin-tuna", "animal", "fish", [3]string{"10", "2", "3"}}, {"fern", "fern", "plant", "fern", [3]string{"20", "4", "3"}}} {
		choices, err := addChoices(document, item.profile)
		if err != nil {
			t.Fatal(err)
		}
		var selected addChoice
		for _, choice := range choices {
			if choice.name == item.profile && choice.field == item.field {
				selected = choice
				break
			}
		}
		var book []byte
		if item.field == "animal" {
			_, book, err = manufacture.EmbeddedAnimalBook(item.family)
		} else {
			_, book, err = manufacture.EmbeddedPlantBook(item.family)
		}
		if err != nil {
			t.Fatal(err)
		}
		bookPath := filepath.Join(dir, item.family+"-book.yaml")
		if err := os.WriteFile(bookPath, book, 0600); err != nil {
			t.Fatal(err)
		}
		if err := work.addProfile(item.name, selected, bookPath, "5", item.at); err != nil {
			t.Fatal(err)
		}
	}
	if err := work.addChosen("city", addChoice{field: "kind", name: "city-grid-10000"}, "31", [3]string{"8000", "0", "0"}); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"tuna", "fern", "city"} {
		work.play.selected = "scene.set.things/" + name
		beforeMove := document.Text()
		if err := work.changePose(2, 1, 0); err != nil {
			t.Fatal(err)
		}
		afterMove := document.Text()
		read, err := set.Read(path, []byte(document.Text()))
		if err != nil {
			t.Fatal(err)
		}
		work.working.parsed, work.working.source, work.working.path = &read, document.Text(), path
		if err := work.changeThingScale(2); err != nil {
			t.Fatal(err)
		}
		afterScale := document.Text()
		work.undoCamera(work.ui.Window().Context(), false)
		if document.Text() != afterMove {
			t.Fatalf("%s scale did not undo", name)
		}
		work.undoCamera(work.ui.Window().Context(), false)
		if document.Text() != beforeMove {
			t.Fatalf("%s move did not undo", name)
		}
		work.undoCamera(work.ui.Window().Context(), true)
		work.undoCamera(work.ui.Window().Context(), true)
		if document.Text() != afterScale {
			t.Fatalf("%s move and scale did not redo", name)
		}
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
	wanted := map[string]bool{"tuna": false, "fern": false, "city": false}
	for _, one := range read.Things() {
		if _, found := wanted[one.Name()]; found {
			scale, has := one.Scale()
			if !has || scale != 2 {
				t.Fatalf("%s scale = %g %v", one.Name(), scale, has)
			}
			east, north := one.At()
			wantEast, wantNorth := 12.0, 3.0
			switch one.Name() {
			case "fern":
				wantEast, wantNorth = 22, 5
			case "city":
				wantEast, wantNorth = 8002, 1
			}
			if east != wantEast || north != wantNorth {
				t.Fatalf("%s position = %g, %g", one.Name(), east, north)
			}
			wanted[one.Name()] = true
		}
	}
	snapshot, err := StageSnapshot(context.Background(), path, string(saved), "set")
	if err != nil {
		t.Fatal(err)
	}
	for _, batch := range snapshot.Scene.Batches {
		if batch.Authored && batch.Author.Collection == "scene.set.things" {
			delete(wanted, batch.Author.Name)
		}
	}
	if len(wanted) != 0 {
		t.Fatalf("saved authored objects did not reach staging: %v", wanted)
	}
	t.Logf("saved acceptance scene %s", path)
}

func TestStageWorkingRejectsCancelledAndInvalidRequests(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := stageWorking(ctx, "scene.yaml", stagedTestSet, "set"); err != context.Canceled {
		t.Fatalf("cancelled: %v", err)
	}
	if _, err := stageWorking(context.Background(), "scene.yaml", "scene: [", "set"); err == nil {
		t.Fatal("invalid source accepted")
	}
	if _, err := stageWorking(context.Background(), "scene.yaml", stagedTestSet, "catalogue"); err == nil {
		t.Fatal("catalogue accepted")
	}
}

func TestStageWorkingReadsUnsavedJobWithRelativeSet(t *testing.T) {
	t.Setenv("SLAYER_CACHE", t.TempDir())
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "scene.yaml"), []byte(stagedTestSet), 0600); err != nil {
		t.Fatal(err)
	}
	source := `"$schema": https://github.com/janearc/slayer/schemas/job-v1.json
scene: {set: scene.yaml}
film: {width: 48, height: 24, rays: 1}
renderer: cpu
`
	path := filepath.Join(dir, "job.yaml")
	view, err := stageWorking(context.Background(), path, source, "job")
	if err != nil {
		t.Fatal(err)
	}
	defer view.close()
	if view.scene.Width != 48 || view.scene.Height != 24 {
		t.Fatalf("buffer's film not staged: %dx%d", view.scene.Width, view.scene.Height)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("buffer was written to disk: %v", err)
	}
}

// A rig's working view is a clearly labelled central-ray approximation;
// changing the physical frame changes its actual field of view and aspect.
func TestPhysicalRigStagesPinholeWorkingView(t *testing.T) {
	t.Setenv("SLAYER_CACHE", t.TempDir())
	old := "film: {width: 640, height: 400, wide: 40, rays: 1, exposure: 1}"
	rig := "film: {width: 640, rays: 1, frame: {width: 36, height: 24}, lens: {focal: 50, stop: 8, focus: first}, shutter: 0.5, glimpses: 2}"
	source := strings.Replace(addRecipeFixture, old, rig, 1)
	path := filepath.Join(t.TempDir(), "rig.yaml")
	first, err := StageSnapshot(context.Background(), path, source, "set")
	if err != nil {
		t.Fatal(err)
	}
	profile, err := preview.InspectFull(path, source, preview.Profile{})
	if err != nil || profile.Width != first.Scene.Width || profile.Height != first.Scene.Height {
		t.Fatalf("render film %+v does not match producer staging %d×%d: %v", profile, first.Scene.Width, first.Scene.Height, err)
	}
	if first.Scene.Width != 640 || first.Scene.Height != 427 || !strings.Contains(first.Notice, "depth of field, motion blur and shutter exposure are not shown") {
		t.Fatalf("rig view %dx%d, notice %q", first.Scene.Width, first.Scene.Height, first.Notice)
	}
	changed := strings.Replace(source, "width: 36, height: 24", "width: 72, height: 24", 1)
	second, err := StageSnapshot(context.Background(), path, changed, "set")
	if err != nil {
		t.Fatal(err)
	}
	if second.Scene.Height != 213 {
		t.Fatalf("changed physical frame height %d", second.Scene.Height)
	}
	ratio := math.Tan(second.Scene.Lens.Wide/2) / math.Tan(first.Scene.Lens.Wide/2)
	if math.Abs(ratio-2) > 1e-6 {
		t.Fatalf("frame width did not change central field of view: ratio %v", ratio)
	}
}

func TestPhysicalGaugeEditSavesAndStagesThroughProducer(t *testing.T) {
	t.Setenv("SLAYER_CACHE", t.TempDir())
	old := "film: {width: 640, height: 400, wide: 40, rays: 1, exposure: 1}"
	rig := "film: {width: 640, rays: 1, frame: {width: 36, height: 24}, lens: {focal: 50, stop: 8, focus: first}, shutter: 0.5, glimpses: 2}"
	source := strings.Replace(addRecipeFixture, old, rig, 1)
	path := filepath.Join(t.TempDir(), "rig.yaml")
	document, err := draft.Open(path, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	work := playWorkspace(t)
	work.state.document, work.state.path = document, path
	work.editor.SetText(source)
	if err := work.changePhysicalFrame(1); err != nil {
		t.Fatal(err)
	}
	changed := document.Text()
	work.undoCamera(work.ui.Window().Context(), false)
	if document.Text() != source {
		t.Fatal("frame undo failed")
	}
	work.undoCamera(work.ui.Window().Context(), true)
	if document.Text() != changed {
		t.Fatal("frame redo failed")
	}
	if err := document.Save(path); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := StageSnapshot(context.Background(), path, string(raw), "set")
	if err != nil {
		t.Fatal(err)
	}
	profile, err := preview.InspectFull(path, string(raw), preview.Profile{})
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Scene.Width != 640 || snapshot.Scene.Height != 443 || profile.Height != 443 || !strings.Contains(snapshot.Notice, "Physical lens shown as a pinhole") {
		t.Fatalf("staged %dx%d profile %+v notice %q", snapshot.Scene.Width, snapshot.Scene.Height, profile, snapshot.Notice)
	}
}

func TestPhysicalLensEditSavesAndStagesThroughProducer(t *testing.T) {
	t.Setenv("SLAYER_CACHE", t.TempDir())
	old := "film: {width: 640, height: 400, wide: 40, rays: 1, exposure: 1}"
	rig := "film: {width: 640, rays: 1, frame: {width: 36, height: 24}, lens: {focal: 50, stop: 8, focus: first}, shutter: 0.5, glimpses: 2}"
	source := strings.Replace(addRecipeFixture, old, rig, 1)
	path := filepath.Join(t.TempDir(), "rig.yaml")
	before, err := StageSnapshot(context.Background(), path, source, "set")
	if err != nil {
		t.Fatal(err)
	}
	document, err := draft.Open(path, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	if err := document.SetPhysicalLens(draft.PhysicalLens{FocalMM: 100, FStop: 2.8}); err != nil {
		t.Fatal(err)
	}
	if err := document.Save(path); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := set.Read(path, raw)
	if err != nil {
		t.Fatal(err)
	}
	film, _ := parsed.Film()
	authored, ok := film.Rig()
	focal, stop, focus := authored.Lens()
	if !ok || focal != 100 || stop != 2.8 || focus != "first" {
		t.Fatalf("producer lens focal %v, f-stop %v, focus %q, rig %t", focal, stop, focus, ok)
	}
	after, err := StageSnapshot(context.Background(), path, string(raw), "set")
	if err != nil {
		t.Fatal(err)
	}
	// The eye and focus object are 70 metres apart. The inverse image
	// distance for a thin lens is 1/focal - 1/subject distance.
	const focusDistanceMM = 70_000
	want := (1.0/100 - 1.0/focusDistanceMM) / (1.0/50 - 1.0/focusDistanceMM)
	ratio := math.Tan(after.Scene.Lens.Wide/2) / math.Tan(before.Scene.Lens.Wide/2)
	if math.Abs(ratio-want) > 1e-6 || after.Scene.Width != before.Scene.Width || after.Scene.Height != before.Scene.Height {
		t.Fatalf("edited lens field-of-view ratio %v, want %v, frame %dx%d", ratio, want, after.Scene.Width, after.Scene.Height)
	}
}

// TestFutianEditedTowerStages is an opt-in, non-rendering real-scene check.
func TestFutianEditedTowerStages(t *testing.T) {
	path := os.Getenv("FERRUCCIO_FUTIAN_SOURCE")
	if path == "" {
		t.Skip("set FERRUCCIO_FUTIAN_SOURCE for real Futian staging")
	}
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	document, err := draft.Open(path, source)
	if err != nil {
		t.Fatal(err)
	}
	objects, err := document.Objects()
	if err != nil {
		t.Fatal(err)
	}
	var target draft.Object
	found := false
	for _, object := range objects {
		if object.ID() == "scene.set.things/tower 006" {
			target = object
			found = true
			break
		}
	}
	if !found || !target.Grounded() || !target.ScaleEditable() {
		t.Fatal("Futian tower is not an editable ground placement")
	}
	if err := document.ChangePose(draft.PoseChange{ID: target.ID(), East: 1, LengthUnit: "m", AngleUnit: "deg"}); err != nil {
		t.Fatal(err)
	}
	if err := document.SetThingScale(target.ID(), 2); err != nil {
		t.Fatal(err)
	}
	parsed, err := set.Read(path, []byte(document.Text()))
	if err != nil {
		t.Fatal(err)
	}
	ground, ok := parsed.Ground()
	if !ok {
		t.Fatal("Futian terrain missing")
	}
	west, north, _, _, _ := ground.Window()
	snapshot, err := StageSnapshot(context.Background(), path, document.Text(), "set")
	if err != nil {
		t.Fatal(err)
	}
	anchor, ok := snapshot.Scene.Origins[target.ID()]
	if !ok {
		t.Fatal("edited tower has no staged anchor")
	}
	position := target.Position()
	if math.Abs(anchor.X-(position[0]+1-west)) > 1e-6 || math.Abs(anchor.Z-(north-position[1])) > 1e-6 {
		t.Fatalf("edited tower anchor %v", anchor)
	}
	found = false
	for _, batch := range snapshot.Scene.Batches {
		if batch.Authored && batch.Author.Collection == "scene.set.things" && batch.Author.Name == "tower 006" {
			for _, place := range batch.Places {
				if place.Scale == 2 {
					found = true
					break
				}
			}
			if found {
				break
			}
		}
	}
	if !found {
		t.Fatal("edited tower's scale did not reach its staged instance")
	}
	t.Logf("Futian edited tower at %v with scale 2; %d shared batches", anchor, len(snapshot.Scene.Batches))
}

func TestStageSnapshotKeepsSkyOnlySceneWithVisibleLimitations(t *testing.T) {
	t.Setenv("SLAYER_CACHE", t.TempDir())
	path := filepath.Join(t.TempDir(), "sky.yaml")
	source, _, _ := strings.Cut(stagedTestSet, "    materials:")
	source += `    sky:
      stars: {seed: 29, share: 0.08, size: 0.5, bright: 0.6}
      source: eye
`
	snapshot, err := StageSnapshot(context.Background(), path, source, "legacy-set")
	if err != nil {
		t.Fatal(err)
	}
	var encoded bytes.Buffer
	if err := stagewire.Write(&encoded, snapshot); err != nil {
		t.Fatal(err)
	}
	decoded, err := stagewire.Read(&encoded)
	if err != nil {
		t.Fatal(err)
	}
	if len(decoded.Scene.Batches) != 0 || decoded.Scene.Pick(.5, .5) != "" ||
		decoded.Scene.Eye.Origin == (maths.Vec{}) || !strings.Contains(decoded.Notice, "Sky is not shown") ||
		!strings.Contains(decoded.Notice, "No drawable surfaces in the working view") {
		t.Fatalf("sky-only snapshot = %+v", decoded)
	}
}

func TestStageSnapshotAndCacheKeepTheSameSkyGradient(t *testing.T) {
	t.Setenv("SLAYER_CACHE", t.TempDir())
	path := filepath.Join(t.TempDir(), "sky.yaml")
	source, _, _ := strings.Cut(stagedTestSet, "    materials:")
	source = strings.Replace(source, "scene:\n", `"$schema": https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: Working gradient
  setting: space
  seed: 17
  light: none
`, 1)
	source += `    sky:
      horizon: {srgb_linear: [0.2, 0.3, 0.4]}
      zenith: {srgb_linear: [0.1, 0.2, 0.7]}
      ground: {srgb_linear: [0.5, 0.2, 0.1]}
      source: eye
`
	snapshot, err := StageSnapshot(context.Background(), path, source, "set")
	if err != nil {
		t.Fatal(err)
	}
	var encoded bytes.Buffer
	if err := stagewire.Write(&encoded, snapshot); err != nil {
		t.Fatal(err)
	}
	decoded, err := stagewire.Read(&encoded)
	if err != nil {
		t.Fatal(err)
	}
	read, err := set.Read(path, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	cached, err := placement.NewDisplayCache().Scene(read)
	if err != nil {
		t.Fatal(err)
	}
	if cached.SkyGradient == nil || decoded.Scene.SkyGradient == nil || *cached.SkyGradient != *decoded.Scene.SkyGradient ||
		len(decoded.Scene.Batches) != 0 || decoded.Scene.Pick(.5, .5) != "" ||
		!strings.Contains(decoded.Notice, workingSkyNotice) || !strings.Contains(localWorkingNotice(read, cached), workingSkyNotice) {
		t.Fatalf("sky-only gradient cache and stage diverged: %v, %v, %q", cached.SkyGradient, decoded.Scene.SkyGradient, decoded.Notice)
	}
}

func TestStageSnapshotShowsAuthoredOpenOceanAsCameraProxy(t *testing.T) {
	t.Setenv("SLAYER_CACHE", t.TempDir())
	path := filepath.Join(t.TempDir(), "open-sea.yaml")
	const source = `"$schema": https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: open sea
  film: {width: 80, height: 60, rays: 1, wide: 45, exposure: 1}
  cameras:
    view:
      location: {at: [400, -300, 2]}
      orientation: {look_at: [400, -285, 0]}
  world:
    unit: 1
    materials:
      water:
        clear: {through: 1, index: 1.33, gloss: 1, rank: 0}
        source: eye
    ocean:
      material: water
      unbounded: true
      middle: [0, 0, 0]
      sea:
        wind: {value: 0, unit: m/s}
        fetch: {value: 0, unit: m}
        depth: {value: 0, unit: m}
        toward: {value: 0, unit: rad}
      seed: 7
      time: {value: 0, unit: s}
      source: eye
`
	snapshot, err := StageSnapshot(context.Background(), path, source, "set")
	if err != nil {
		t.Fatal(err)
	}
	var encoded bytes.Buffer
	if err := stagewire.Write(&encoded, snapshot); err != nil {
		t.Fatal(err)
	}
	decoded, err := stagewire.Read(&encoded)
	if err != nil {
		t.Fatal(err)
	}
	if len(decoded.Scene.Batches) != 1 ||
		!strings.Contains(decoded.Notice, "finite mean-level proxy around the camera; waves and coastline are not shown") {
		t.Fatalf("authored open ocean snapshot = %+v", decoded)
	}
	if hit, ok := decoded.Scene.PickAt(.5, .9); !ok || hit.ID != "scene.world" || decoded.Scene.Pick(.5, .9) != "" {
		t.Fatalf("water should be visible without a selectable identity: %+v", hit)
	}
	batch := decoded.Scene.Batches[0]
	at := batch.Places[0]
	eye := at.In(decoded.Scene.Eye.Origin)
	low, high, ok := batch.Mesh.Bounds()
	if !ok || !batch.HasBounds || math.Abs(eye.Y-2) > 1e-9 ||
		math.Abs(eye.X-(low.X+high.X)/2) > 1e-9 || math.Abs(eye.Z-(low.Z+high.Z)/2) > 1e-9 ||
		low.Y != 0 || high.Y != 0 {
		t.Fatalf("sea proxy bounds %v %v do not enclose the authored eye projection from %v", low, high, eye)
	}
}

func TestWorkingSnapshotShowsProducerSmokeGroundAndBall(t *testing.T) {
	t.Setenv("SLAYER_CACHE", t.TempDir())
	path := "../../../examples/first-picture.yaml"
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := StageSnapshot(context.Background(), path, string(source), "job")
	if err != nil {
		t.Fatal(err)
	}
	fixture, ok := smoke.Get(1)
	if !ok {
		t.Fatal("smoke one is unavailable")
	}
	_, eye := fixture.Scene()
	var encoded bytes.Buffer
	if err := stagewire.Write(&encoded, snapshot); err != nil {
		t.Fatal(err)
	}
	decoded, err := stagewire.Read(&encoded)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Scene.Eye != eye || decoded.Scene.Lens.Wide != fixture.Wide || decoded.Scene.Width != 160 || decoded.Scene.Height != 90 {
		t.Fatalf("starter framing differs from the smoke eye and job film: %+v", decoded.Scene)
	}
	batches := decoded.Scene.Batches
	if len(batches) != 2 || batches[0].Mesh.Notice != "" || batches[1].Mesh.Notice == "" ||
		!strings.Contains(decoded.Notice, "Infinite planes shown as finite working-view patches") {
		t.Fatalf("smoke ground and ball snapshot = %+v", decoded)
	}
	ball, ok := decoded.Scene.PickAt(.5, .5)
	if !ok || ball.Point.Y <= 0.5 || ball.ID != "scene.world" {
		t.Fatalf("smoke ball did not remain in front of the ground: %+v", ball)
	}
	ground, ok := decoded.Scene.PickAt(.15, .9)
	if !ok || math.Abs(ground.Point.Y) > 1e-9 || ground.ID != "scene.world" || decoded.Scene.Pick(.15, .9) != "" {
		t.Fatalf("smoke ground should be visible and read-only: %+v", ground)
	}
	if focus := sharedFocus(batches[:1], decoded.Scene.Eye); focus != decoded.Scene.Focus {
		t.Fatalf("ground proxy moved the working focus: %v, want %v", decoded.Scene.Focus, focus)
	}
}
