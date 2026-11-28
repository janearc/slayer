package workspace

import (
	"context"
	"errors"
	"image"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gogpu/ui/core/dropdown"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/janearc/slayer/ferruccio/internal/draft"
	"github.com/janearc/slayer/session/placement"
)

const surveyedSceneFixture = "../../../../sets/futian-night-authoring.yaml"

func TestAuthoredMapMovesGroundPlacementWithoutStaging(t *testing.T) {
	work := playWorkspace(t)
	defer work.Close()
	var calls atomic.Int32
	work.working.stageOverride = func(context.Context, string, string, string) (*stagedView, error) {
		calls.Add(1)
		return nil, errors.New("unexpected geometry stage")
	}
	scene, err := filepath.Abs(surveyedSceneFixture)
	if err != nil {
		t.Fatal(err)
	}
	work.openPath(scene)
	work.viewer.SetBounds(geometry.NewRect(0, 0, 720, 720))
	ready := func() {
		t.Helper()
		work.updateWorking()
		waitStaged(t, work)
		work.updateWorking()
		if live := &work.working; !work.WorkingReady() || live.plan == nil || !live.plan.editable || len(live.scene.Objects) < 100 {
			t.Fatalf("authored plan unavailable: %s / %s", work.caption, work.diagnostic)
		}
	}
	ready()
	live := &work.working
	if terrain := live.plan.terrain; terrain == nil || terrain.East-terrain.West != 4000 || terrain.North-terrain.South != 3000 {
		t.Fatalf("Futian terrain outline = %+v", terrain)
	}
	if err := work.Navigate(0, 0, 1.5); err != nil {
		t.Fatal(err)
	}
	work.updateWorking()
	framing := live.plan.span
	first := live.scene.Objects[0]
	x, y := live.plan.point(first)
	if !work.viewer.selectObject(x, y) || work.play.selected != first.ID {
		t.Fatalf("selection = %q, want %q", work.play.selected, first.ID)
	}
	before := work.state.document.Text()
	if err := work.changePose(-1e6, 0, 0); err == nil || work.state.document.Text() != before {
		t.Fatal("out-of-window move changed the draft")
	}
	work.viewer.dragObject = true
	work.beginWorking()
	if live.gesture == nil || work.state.document.Text() != before {
		t.Fatal("starting the map drag modified the source")
	}
	work.viewer.changeObject(x, y, x+10/live.plan.span, y-5/live.plan.span)
	work.endWorking(true)
	after := work.state.document.Text()
	if after == before {
		t.Fatal("releasing the map drag did not edit the draft")
	}
	ready()
	moved := live.scene.Objects[0].At.Origin
	if math.Abs(moved.X-first.At.Origin.X-10) > 1e-7 || math.Abs(moved.Z-first.At.Origin.Z+5) > 1e-7 {
		t.Fatalf("marker = %v after a 10 m east and 5 m north move", moved)
	}
	work.undoCamera(work.ui.Window().Context(), false)
	if work.state.document.Text() != before {
		t.Fatal("undo did not restore the exact source")
	}
	ready()
	if live.scene.Objects[0].At.Origin != first.At.Origin {
		t.Fatal("undo did not restore the marker")
	}
	work.undoCamera(work.ui.Window().Context(), true)
	if work.state.document.Text() != after {
		t.Fatal("redo did not restore the edited source")
	}
	ready()
	if live.scene.Objects[0].At.Origin != moved || live.plan.span != framing || calls.Load() != 0 || work.cancel != nil {
		t.Fatalf("redo marker or worker state is wrong: %v / %d", live.scene.Objects[0].At.Origin, calls.Load())
	}
}

func TestAuthoredMapOutlinesFollowMapRotationAndZoom(t *testing.T) {
	terrain := &draft.MapRectangle{West: 100, South: 150, East: 300, North: 250}
	view := makeAuthoredMap(nil, false, terrain)
	plan := view.plan
	if plan.east != 200 || plan.north != 200 || !strings.Contains(plan.caption(), "224 m across · north up") {
		t.Fatalf("terrain framing = %+v / %s", plan, plan.caption())
	}
	plan.span, plan.angle = 200, math.Pi/2
	pixels := image.NewRGBA(image.Rect(0, 0, 100, 100))
	red := color.RGBA{R: 255, A: 255}
	plan.drawExtent(pixels, *terrain, red)
	for _, point := range []image.Point{{X: 25, Y: 50}, {X: 75, Y: 50}} {
		if got := pixels.RGBAAt(point.X, point.Y); got != red {
			t.Fatalf("rotated outline at %v = %v", point, got)
		}
	}
	if got := pixels.RGBAAt(50, 50); got != (color.RGBA{}) {
		t.Fatalf("outline fills unknown terrain interior: %v", got)
	}
	if !strings.Contains(plan.caption(), "200 m across · rotated") {
		t.Fatalf("rotated and zoomed caption = %s", plan.caption())
	}
}

func TestAuthoredMapResolvesJobWithoutChangingItsSources(t *testing.T) {
	scene, err := os.ReadFile(surveyedSceneFixture)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	scenePath, jobPath := filepath.Join(dir, "scene.yaml"), filepath.Join(dir, "job.yaml")
	if err := os.WriteFile(scenePath, scene, 0600); err != nil {
		t.Fatal(err)
	}
	job := "$schema: https://github.com/janearc/slayer/schemas/job-v1.json\nscene: {set: scene.yaml}\ncamera: {name: above}\n"
	if err := os.WriteFile(jobPath, []byte(job), 0600); err != nil {
		t.Fatal(err)
	}
	work := playWorkspace(t)
	defer work.Close()
	work.working.stageOverride = func(context.Context, string, string, string) (*stagedView, error) {
		return nil, errors.New("unexpected staging of referenced scene")
	}
	work.openPath(jobPath)
	work.viewer.SetBounds(geometry.NewRect(0, 0, 720, 720))
	work.updateWorking()
	waitStaged(t, work)
	work.updateWorking()
	live := &work.working
	if !work.WorkingReady() || live.plan == nil || !live.plan.readOnly || len(live.scene.Objects) < 100 {
		t.Fatalf("job plan unavailable: %s / %s", work.caption, work.diagnostic)
	}
	first := live.scene.Objects[0]
	x, y := live.plan.point(first)
	if !work.viewer.selectObject(x, y) || work.play.selected != first.ID || !strings.Contains(work.caption, "open scene to edit") {
		t.Fatalf("referenced scene does not appear read-only: %s", work.caption)
	}
	if !work.canShowSceneDetail() {
		t.Fatal("referenced scene selection cannot open detail")
	}
	work.showSceneDetail()
	waitStaged(t, work)
	if live.plan == nil || !live.plan.readOnly || live.detail != "" || !strings.Contains(work.diagnostic, "Scene detail unavailable") {
		t.Fatal("failed job detail did not restore its read-only map")
	}
	const group = "scene.set.stands/near sampans at anchor"
	var groupMark placement.DisplayObject
	for _, mark := range live.scene.Objects {
		if mark.ID == group {
			groupMark = mark
			break
		}
	}
	if !live.plan.groups[group] || groupMark.At.Origin.X != 1350 || groupMark.At.Origin.Z != -1500 {
		t.Fatalf("group's source anchor = %v", groupMark.At.Origin)
	}
	x, y = live.plan.point(groupMark)
	if !work.viewer.selectObject(x+3/float64(live.width), y) || work.play.selected != group {
		t.Fatalf("picked %q from the stand outline", work.play.selected)
	}
	work.toggleDisclosure("find placement")
	submitPlacementSearch(t, work, "near sampans under way")
	if work.play.selected != "scene.set.stands/near sampans under way" {
		t.Fatalf("name picker selected %q for coincident group", work.play.selected)
	}
	work.viewer.dragObject = true
	work.beginWorking()
	if live.gesture != nil {
		t.Fatal("job selection started a placement drag")
	}
	if err := work.changePose(1, 1, 0); err == nil || work.state.document.Text() != job || work.state.path != jobPath {
		t.Fatal("a job placement edit changed the active document")
	}
	document, _, err := ReadDocument(scenePath)
	if err != nil || document.Text() != string(scene) {
		t.Fatal("showing the job plan changed its referenced scene")
	}
	// A retained plan can outlive the active scene during a view request.
	live.scene = nil
	if _, stalePicker := work.objectPicker().(*dropdown.Widget); stalePicker {
		t.Fatal("staging still offers the old job map picker")
	}
	if work.playLayout() == nil || work.playReadout() == "" {
		t.Fatal("job controls vanished during staging")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	doc, err := draft.Open(jobPath, []byte(job))
	if err != nil {
		t.Fatal(err)
	}
	if view, err := prepareAuthoredMap(ctx, doc, jobPath); view != nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled preparation = %v, %v", view, err)
	}
}

func TestAuthoredMapFullSceneRequiresExplicitChoice(t *testing.T) {
	work := playWorkspace(t)
	defer work.Close()
	var calls atomic.Int32
	work.working.stageOverride = func(context.Context, string, string, string) (*stagedView, error) {
		calls.Add(1)
		return &stagedView{scene: &placement.DisplayScene{Width: 640, Height: 360}}, nil
	}
	scene, err := filepath.Abs(surveyedSceneFixture)
	if err != nil {
		t.Fatal(err)
	}
	work.openPath(scene)
	work.updateWorking()
	waitStaged(t, work)
	if work.working.plan == nil || calls.Load() != 0 {
		t.Fatal("opening the file did not use the authored map")
	}
	if err := work.SetWorkingMode("wire"); err != nil {
		t.Fatal(err)
	}
	if err := work.chooseWorkingDisplay("full-scene"); err != nil {
		t.Fatal(err)
	}
	waitStaged(t, work)
	if !work.working.fullScene || work.working.plan != nil || calls.Load() != 1 || work.working.mode != "wire" {
		t.Fatal("explicit full scene choice did not keep the selected display style")
	}
	if err := work.chooseWorkingDisplay("authored-map"); err != nil {
		t.Fatal(err)
	}
	waitStaged(t, work)
	if work.working.fullScene || work.working.plan == nil || calls.Load() != 1 {
		t.Fatal("returning to the map started another full stage")
	}
}

func TestAuthoredMapOpensSelectedDetailAndReturns(t *testing.T) {
	work := playWorkspace(t)
	defer work.Close()
	var calls atomic.Int32
	work.working.stageOverride = func(context.Context, string, string, string) (*stagedView, error) {
		calls.Add(1)
		return &stagedView{scene: &placement.DisplayScene{Width: 640, Height: 360}}, nil
	}
	scene, err := filepath.Abs(surveyedSceneFixture)
	if err != nil {
		t.Fatal(err)
	}
	work.openPath(scene)
	work.viewer.SetBounds(geometry.NewRect(0, 0, 720, 720))
	work.updateWorking()
	waitStaged(t, work)
	work.updateWorking()
	first := work.working.scene.Objects[0]
	x, y := work.working.plan.point(first)
	if !work.viewer.selectObject(x, y) || !work.canShowSceneDetail() {
		t.Fatal("map selection did not offer detail")
	}
	work.showSceneDetail()
	waitStaged(t, work)
	if work.working.detail != first.ID || work.working.plan != nil || calls.Load() != 1 {
		t.Fatalf("selected detail did not stage: %q, calls %d", work.working.detail, calls.Load())
	}
	work.showSceneMap()
	work.updateWorking()
	waitStaged(t, work)
	if work.working.detail != "" || work.working.plan == nil || calls.Load() != 1 {
		t.Fatal("return to map failed or staged full geometry")
	}
}

func TestAuthoredMapBallsChangeOnlyMapFraming(t *testing.T) {
	work := playWorkspace(t)
	defer work.Close()
	source := work.state.document.Text()
	mapped := makeAuthoredMap([]draft.MapMark{{ID: "test", East: 10, North: 20}}, false, nil)
	live := &work.working
	live.scene, live.plan = mapped.scene, mapped.plan
	live.source, live.path = source, work.state.path
	live.dirty = true
	work.viewer.SetBounds(geometry.NewRect(0, 0, 720, 720))
	work.updateAuthoredMap()
	ctx := work.ui.Window().Context()
	drag := func(ball *cameraBall, dx, dy float32, mods event.Modifiers, keep bool) {
		t.Helper()
		start, _ := ball.disc(work.viewer.Bounds())
		end := geometry.Pt(start.X+dx, start.Y+dy)
		if !ball.Event(ctx, event.NewMouseEvent(event.MousePress, event.ButtonLeft, event.ButtonStateLeft, start, start, mods)) || !ball.dragging {
			t.Fatal("map ball did not capture the press")
		}
		ball.Event(ctx, event.NewMouseEvent(event.MouseDrag, event.ButtonLeft, event.ButtonStateLeft, end, end, mods))
		if keep {
			ball.Event(ctx, event.NewMouseEvent(event.MouseRelease, event.ButtonLeft, 0, end, end, mods))
		} else {
			ball.Event(ctx, event.NewKeyEvent(event.KeyPress, event.KeyEscape, 0, event.ModNone))
		}
		if ball.dragging {
			t.Fatal("map ball did not release the gesture")
		}
	}
	drag(work.viewer.ball, 32, 0, event.ModNone, true)
	if live.plan.angle <= 0 || strings.Contains(work.workingCaption(), "north up") {
		t.Fatal("Orbit did not rotate the plan and correct the caption")
	}
	beforeX, beforeY := live.plan.point(live.scene.Objects[0])
	drag(work.viewer.travel, 48, -24, event.ModNone, true)
	afterX, afterY := live.plan.point(live.scene.Objects[0])
	if math.Abs(afterX-beforeX-48.0/720) > 1e-7 || math.Abs(afterY-beforeY+24.0/720) > 1e-7 {
		t.Fatalf("Travel panned marker from %v, %v to %v, %v", beforeX, beforeY, afterX, afterY)
	}
	point := geometry.Pt(float32(afterX*720), float32(afterY*720))
	press := event.NewMouseEvent(event.MousePress, event.ButtonLeft, event.ButtonStateLeft, point, point, event.ModNone)
	if !work.viewer.Event(ctx, press) || !work.viewer.dragging || !work.viewer.dragObject {
		t.Fatal("placement press did not capture the map")
	}
	spanDuringDrag := live.plan.span
	scroll := event.NewWheelEvent(geometry.Pt(0, 80), point, point, event.ModNone)
	if !work.viewer.Event(ctx, scroll) || live.plan.span != spanDuringDrag {
		t.Fatal("wheel changed map scale during a captured placement drag")
	}
	work.viewer.Event(ctx, event.NewKeyEvent(event.KeyPress, event.KeyEscape, 0, event.ModNone))
	beforeZoom := live.plan.span
	drag(work.viewer.travel, 0, 32, event.ModShift, true)
	if live.plan.span <= beforeZoom {
		t.Fatal("Shift-Travel did not zoom")
	}
	angle, span := live.plan.angle, live.plan.span
	drag(work.viewer.ball, -32, 0, event.ModNone, false)
	if live.plan.angle != angle || live.plan.span != span {
		t.Fatal("Escape did not restore the framing")
	}
	at := live.scene.Objects[0]
	x, y := live.plan.point(at)
	east, north := work.mapDrag(x, y, x+.1, y-.2)
	at.At.Origin.X += east
	at.At.Origin.Z -= north
	x1, y1 := live.plan.point(at)
	if math.Abs(x1-x-.1) > 1e-7 || math.Abs(y1-y+.2) > 1e-7 {
		t.Fatal("placement drag and rotated map disagree")
	}
	if work.state.document.Text() != source || live.stager != nil || work.Running() {
		t.Fatal("map navigation changed the draft or started background work")
	}
}
