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

	"github.com/janearc/slayer/ferruccio/internal/draft"
	"github.com/janearc/slayer/ferruccio/native/internal/stagewire"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/set"
)

const groundEditFixture = `$schema: https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: Ground editing
  setting: ground
  light: moon
  timeofday: 2200
  terrain: {from: fixture, file: fixture-plain.tif, west: 1001, north: 8999, across: 150, down: 100, step: 2}
  film: {width: 24, height: 16, rays: 1, exposure: 100000}
  cameras:
    view:
      orientation: {point_to: stone, alpha: -25}
      location: {loc_relative: {object: stone, standoff_x: 20, standoff_y: 0, standoff_z: 10}}
  set:
    things:
      stone: {kind: yurt, seed: 4, east: 1050, north: 8990, heading: {value: 30, unit: deg}}
`

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
