package workspace

import (
	"errors"
	"fmt"
	"github.com/gogpu/ui/core/dropdown"
	"github.com/gogpu/ui/widget"
	"github.com/janearc/slayer/ferruccio/internal/draft"
	"github.com/janearc/slayer/ferruccio/native/internal/appearance"
	"github.com/janearc/slayer/ferruccio/native/internal/stagewire"
	"github.com/janearc/slayer/ferruccio/native/internal/workingview"
	"github.com/janearc/slayer/internal/observability"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/session/placement"
	"github.com/janearc/slayer/set"
	"image"
	"strings"
	"sync"
	"time"
)

// workingState keeps display geometry resident; source changes only on release.
type workingState struct {
	cache                       *placement.DisplayCache
	parsed                      *set.Set
	scene, gesture              *placement.DisplayScene
	gpu                         *workingview.View
	photograph                  image.Image
	source, path, mode, failure string
	notice, cameraName          string
	detail                      string
	detailFallback              *detailFallback
	detailRetry                 *failedDetail
	cameraPath                  string
	cameraPathAt                time.Duration
	cameraWide                  float64
	frame                       *stagewire.FilmFrame
	requestedFrame              int
	dirty, waiting, displayed   bool
	splitWorldFallback          bool
	width, height               int
	stager                      *stageLoader
	stageAuthoring              *stageAuthoring
	stageReadSlot               chan struct{}
	retiredPreparation          sync.WaitGroup
	stageCache                  *stagewire.Session
	staging                     bool
	stageOverride               stageBuild
	rollback                    *workingRollback
	preview                     *workingPreview
	poseHistory                 []workingPreview
	provisional                 bool
	plan, planFraming           *authoredPlan
	fullScene                   bool
	timing                      workingTiming
	inspection                  *inspectionCamera
	// late is how many readbacks the next GPU view is told to miss.
	late int
}

// workingRollback retains the last exact scene only while its replacement is staging.
type workingRollback struct {
	document             *draft.Document
	scene                *placement.DisplayScene
	parsed               *set.Set
	source, path, notice string
}

// workingPreview is a temporary transformed scene for one exact draft.
type workingPreview struct {
	document             *draft.Document
	scene                *placement.DisplayScene
	source, path, notice string
}

func (work *Workspace) rememberWorking(scene *placement.DisplayScene) {
	live := &work.working
	if scene == nil || live.plan != nil || live.provisional || live.source == "" || live.path != work.state.path {
		return
	}
	if live.rollback != nil && live.rollback.document == work.state.document && live.rollback.source == live.source && live.rollback.path == live.path {
		return
	}
	live.rollback = &workingRollback{document: work.state.document, scene: work.withStagedCamera(scene), parsed: live.parsed, source: live.source, path: live.path, notice: live.notice}
}

// poseBase retains the current approximation across consecutive pose edits;
// it never makes a provisional scene eligible for picking.
func (work *Workspace) poseBase() (*placement.DisplayScene, bool) {
	live := &work.working
	if work.workingCurrent() || (live.provisional && live.scene != nil && live.source == work.state.document.Text() && live.path == work.state.path) {
		return live.scene, true
	}
	if preview := live.preview; preview != nil && preview.document == work.state.document && preview.source == work.state.document.Text() && preview.path == work.state.path {
		return preview.scene, true
	}
	return nil, false
}

// rememberProvisional keeps a few recent approximations for history changes.
// Scenes share the unchanged immutable batches, and each changed batch owns
// its placement slice.
func (work *Workspace) rememberProvisional() {
	live := &work.working
	var saved *workingPreview
	if live.provisional && live.scene != nil && live.source == work.state.document.Text() && live.path == work.state.path {
		saved = &workingPreview{document: work.state.document, scene: work.withStagedCamera(live.scene), source: live.source, path: live.path, notice: live.notice}
	} else if preview := live.preview; preview != nil && preview.document == work.state.document && preview.source == work.state.document.Text() && preview.path == work.state.path {
		saved = preview
	}
	if saved == nil {
		return
	}
	for i := range live.poseHistory {
		if live.poseHistory[i].document == saved.document && live.poseHistory[i].path == saved.path && live.poseHistory[i].source == saved.source {
			live.poseHistory[i] = *saved
			return
		}
	}
	const retainedPoses = 4
	if len(live.poseHistory) == retainedPoses {
		copy(live.poseHistory, live.poseHistory[1:])
		live.poseHistory = live.poseHistory[:retainedPoses-1]
	}
	live.poseHistory = append(live.poseHistory, *saved)
}

// restoreProvisional queues an approximation for this exact history text;
// updateWorking still stages and validates the full scene before picking.
func (work *Workspace) restoreProvisional(text string) bool {
	for i := len(work.working.poseHistory) - 1; i >= 0; i-- {
		saved := work.working.poseHistory[i]
		if saved.document == work.state.document && saved.path == work.state.path && saved.source == text {
			work.rememberWorking(work.working.scene)
			work.working.preview = &saved
			work.working.dirty = true
			return true
		}
	}
	return false
}

func (work *Workspace) restoreWorking(text string) bool {
	live, saved := &work.working, work.working.rollback
	if saved == nil || saved.document != work.state.document || saved.source != text || saved.path != work.state.path {
		return false
	}
	if live.stager != nil {
		live.stager.discard()
	}
	live.stageAuthoring = nil
	var inverse *workingRollback
	if !live.provisional && live.scene == saved.scene && live.path == saved.path && live.source != saved.source {
		inverse = &workingRollback{document: saved.document, scene: live.scene, parsed: live.parsed, source: live.source, path: live.path, notice: live.notice}
	}
	live.scene, live.parsed = saved.scene, saved.parsed
	live.source, live.path, live.notice = saved.source, saved.path, saved.notice
	live.failure, live.staging, live.displayed = "", false, false
	live.preview, live.provisional, live.inspection = nil, false, nil
	live.dirty, live.waiting, live.rollback = true, work.play.renderedText != text, inverse
	work.diagnostic = saved.notice
	work.caption = work.workingCaption()
	work.refreshLayout = true
	return true
}

// gestureOrigin resolves a source placement to the renderer's actual anchor.
// Older space snapshots can still use their authored world coordinates.
func gestureOrigin(scene *placement.DisplayScene, object draft.Object) (maths.Vec, error) {
	if scene != nil {
		if origin, ok := scene.Origins[object.ID()]; ok {
			return origin, nil
		}
	}
	if object.Grounded() {
		return maths.Vec{}, fmt.Errorf("the staged terrain anchor is unavailable; wait for the working view to reload")
	}
	position := object.Position()
	return maths.V(position[0], position[2], -position[1]), nil
}

// dragPlacement shares the captured projection between preview and release.
func (work *Workspace) dragPlacement(scene *placement.DisplayScene, x0, y0, x1, y1 float64) (float64, float64, error) {
	if !work.state.document.Matches(work.working.source) || work.working.path != work.state.path {
		return 0, 0, fmt.Errorf("the source changed during the drag")
	}
	if work.working.plan != nil {
		east, north := work.mapDrag(x0, y0, x1, y1)
		return east, north, nil
	}
	if object, ok := work.selectedObject(); ok && (object.AbsolutePosition() || len(scene.Batches) > 0) {
		origin, err := gestureOrigin(scene, object)
		if err != nil {
			return 0, 0, err
		}
		if object.AbsolutePosition() {
			return scene.DragViewDepth(origin, x0, y0, x1, y1)
		}
		return scene.DragHeight(origin.Y, x0, y0, x1, y1)
	}
	return scene.Drag(work.play.selected, x0, y0, x1, y1)
}

// ForceLateReadbacks makes the working view's next count readbacks late on
// purpose, so a proof can show it recover rather than stop.
func (work *Workspace) ForceLateReadbacks(count int) {
	work.working.late = count
	if work.working.gpu != nil {
		work.working.gpu.ForceLateReadbacks(count)
	}
}

// SetWorkingMode selects a display without changing scene source or requesting a render.
func (work *Workspace) SetWorkingMode(mode string) error {
	switch mode {
	case "rendered", "solid", "edges", "wire":
	default:
		return fmt.Errorf("display must be rendered, solid, edges or wire")
	}
	work.fullCancellationNote = ""
	work.working.mode = mode
	work.refreshLayout = true
	work.working.dirty = true
	work.invalidate()
	return nil
}

// WorkingReady reports displayed pixels for the current working scene.
func (work *Workspace) WorkingReady() bool {
	return (work.working.gpu != nil || work.working.plan != nil) && work.workingCurrent() && work.working.displayed && !work.working.dirty && work.working.failure == ""
}

// WorkingStats exposes measured work; no inferred FPS or fabricated progress.
func (work *Workspace) WorkingStats() any {
	if work.working.gpu == nil {
		return nil
	}
	return work.working.gpu.Stats()
}

// workingPicker returns the dropdown that chooses the working display mode.
func (work *Workspace) workingPicker() widget.Widget {
	labels := []string{"Rendered", "Solid", "Edges", "Wireframe"}
	modes := []string{"rendered", "solid", "edges", "wire"}
	if work.working.plan != nil {
		labels = []string{"Rendered", "Authored map", "Load full 3D (slow)"}
		modes = []string{"rendered", "edges", "full-scene"}
	} else if work.working.fullScene {
		labels, modes = append(labels, "Authored map"), append(modes, "authored-map")
	}
	selected := 0
	if work.working.plan != nil && !work.photographVisible() {
		selected = 1
	}
	for i, mode := range modes {
		if work.working.plan == nil && work.working.mode == mode {
			selected = i
		}
	}
	choice := dropdown.New(dropdown.Items(labels...), dropdown.Selected(selected), dropdown.PainterOpt(appearance.DropdownPainter{Theme: work.theme, Size: work.size}), dropdown.OnChange(func(i int, _ string) {
		if i >= 0 && i < len(modes) {
			work.problem(work.chooseWorkingDisplay(modes[i]))
		}
	}))
	choice.SetID("Viewport display")
	return choice
}

// workingCurrent reports whether the working scene matches the document text
// and path.
func (work *Workspace) workingCurrent() bool {
	live := &work.working
	return !live.provisional && live.scene != nil && work.state.document.Matches(live.source) && live.path == work.state.path && (live.displayed || (work.acceptedPath == live.path && work.play.renderedText == live.source))
}

const chooseWorkingView = "Choose Solid, Edges or Wire to select objects in the working view"

// photographVisible distinguishes renderer pixels from the native working view.
func (work *Workspace) photographVisible() bool {
	return work.working.photograph != nil && !work.working.displayed
}

// workingPick uses only the projection of the native working view.
func (work *Workspace) workingPick(x, y float64) string {
	if work.photographVisible() || !work.workingCurrent() {
		return ""
	}
	if work.working.plan != nil {
		return work.pickAuthoredMap(x, y).ID
	}
	return work.working.scene.Pick(x, y)
}

// copyDisplay returns a copy of scene with its own object list, or nil when
// scene is nil.
func copyDisplay(scene *placement.DisplayScene) *placement.DisplayScene {
	if scene == nil {
		return nil
	}
	copied := *scene
	copied.Objects = append([]placement.DisplayObject(nil), scene.Objects...)
	copied.Batches = append([]placement.StagedBatch(nil), scene.Batches...)
	return &copied
}

// previewPose gives a checked authored placement an explicitly provisional
// transform while the stage recalculates ground and dependent geometry.
func (work *Workspace) previewPose(base *placement.DisplayScene, object draft.Object, change draft.PoseChange) {
	if base == nil || len(base.Batches) == 0 {
		return
	}
	pivot, err := gestureOrigin(base, object)
	if err != nil {
		return
	}
	scene := copyDisplay(base)
	shift := maths.V(change.East, change.Up, -change.North)
	turn := maths.Rotation(maths.YAxis, -change.Heading*maths.Degree)
	found := false
	for i := range scene.Batches {
		batch := &scene.Batches[i]
		if !batch.Authored || batch.Author.Collection+"/"+batch.Author.Name != object.ID() || len(batch.Places) == 0 {
			continue
		}
		found = true
		batch.Places = append([]maths.Subspace(nil), base.Batches[i].Places...)
		batch.HasBounds = false
		for j := range batch.Places {
			place := &batch.Places[j]
			place.Origin = pivot.Add(turn.Out(place.Origin.Sub(pivot))).Add(shift)
			place.Orientation = turn.Then(place.Orientation)
		}
	}
	if found {
		if _, ok := scene.Origins[object.ID()]; ok {
			origins := make(map[string]maths.Vec, len(scene.Origins))
			for id, origin := range scene.Origins {
				origins[id] = origin
			}
			origins[object.ID()] = pivot.Add(shift)
			scene.Origins = origins
		}
		work.working.preview = &workingPreview{document: work.state.document, scene: work.withStagedCamera(scene), source: work.state.document.Text(), path: work.state.path, notice: work.working.notice}
	}
}

// previewScale shows an approximate uniform size while staging recomputes
// any geometry and dependencies affected by the edit.
func (work *Workspace) previewScale(base *placement.DisplayScene, object draft.Object, ratio float64) {
	if base == nil || len(base.Batches) == 0 {
		return
	}
	pivot, err := gestureOrigin(base, object)
	if err != nil {
		return
	}
	scene := copyDisplay(base)
	found := false
	for i := range scene.Batches {
		batch := &scene.Batches[i]
		if !batch.Authored || batch.Author.Collection+"/"+batch.Author.Name != object.ID() || len(batch.Places) == 0 {
			continue
		}
		found = true
		batch.Places = append([]maths.Subspace(nil), base.Batches[i].Places...)
		batch.HasBounds = false
		for j := range batch.Places {
			place := &batch.Places[j]
			place.Origin = pivot.Add(place.Origin.Sub(pivot).Scale(ratio))
			if place.Scale == 0 {
				place.Scale = 1
			}
			place.Scale *= ratio
		}
	}
	if found {
		work.working.preview = &workingPreview{document: work.state.document, scene: work.withStagedCamera(scene), source: work.state.document.Text(), path: work.state.path, notice: work.working.notice}
	}
}

// beginWorking starts a gesture by keeping a copy of the working scene as
// the base for moves.
func (work *Workspace) beginWorking() {
	work.beginWorkingGesture(false)
}

func (work *Workspace) beginWorkingGesture(cameraBall bool) {
	if work.photographVisible() || !work.workingCurrent() {
		return
	}
	if work.refuseWhileFull("changing the scene") {
		return
	}
	if live := &work.working; live.plan != nil && (cameraBall || !work.viewer.dragObject || !live.plan.editable) {
		return
	}
	if !cameraBall && work.viewer.dragObject && work.refuseObjectGesture() {
		return
	}
	inspecting := (cameraBall || (!work.viewer.dragObject && work.working.detail != "")) && work.cameraInspectionOnly()
	if work.Running() && !inspecting {
		work.stopPreview()
	}
	work.fullCancellationNote = ""
	work.working.gesture = copyDisplay(work.working.scene)
	work.working.dirty = true
}

func (work *Workspace) refuseObjectGesture() bool {
	if group, ok := work.selectedGroup(); ok {
		work.status = group.Name + " · generated members cannot be dragged separately"
		if _, editable := work.state.document.GroupArea(group.ID); editable {
			work.status = group.Name + " · use Region to edit this group area"
		}
		work.invalidate()
		return true
	}
	if object, ok := work.selectedObject(); ok && object.Grounded() {
		if work.working.plan == nil {
			if _, err := gestureOrigin(work.working.scene, object); err != nil {
				work.problem(err)
				return true
			}
		}
		if work.play.mode == "turn" {
			if _, said := object.Heading(); !said {
				work.problem(fmt.Errorf("this terrain object has a generated heading; set an explicit heading in its source before turning it"))
				return true
			}
		}
	}
	return false
}

// moveWorking changes cached transforms only; the YAML history stays untouched.
func (work *Workspace) moveWorking() {
	live, view := &work.working, work.viewer
	base := live.gesture
	if base == nil {
		return
	}
	started := workingClock()
	scene := copyDisplay(base)
	dx, dy := float64(view.dragCurrent.X-view.dragStart.X), float64(view.dragCurrent.Y-view.dragStart.Y)
	bounds := view.imageBounds()
	if bounds.IsEmpty() {
		return
	}
	if view.dragObject {
		x0, y0 := float64(view.dragImage.X), float64(view.dragImage.Y)
		x1, y1 := x0+dx/float64(bounds.Width()), y0+dy/float64(bounds.Height())
		east, north, err := 0.0, 0.0, error(nil)
		if view.mode != "turn" {
			east, north, err = work.dragPlacement(base, x0, y0, x1, y1)
		}
		if err != nil {
			return
		}
		for i := range scene.Objects {
			if scene.Objects[i].ID != work.play.selected {
				continue
			}
			scene.Objects[i].At.Origin = scene.Objects[i].At.Origin.Add(maths.V(east, 0, -north))
			if view.mode == "turn" {
				scene.Objects[i].At.Orientation = maths.Rotation(maths.YAxis, -(x1-x0)*180*maths.Degree).Then(base.Objects[i].At.Orientation)
			}
		}
		if object, ok := work.selectedObject(); ok && live.plan == nil {
			pivot, err := gestureOrigin(base, object)
			if err != nil {
				return
			}
			if _, ok := base.Origins[object.ID()]; ok && view.mode != "turn" {
				origins := make(map[string]maths.Vec, len(base.Origins))
				for id, origin := range base.Origins {
					origins[id] = origin
				}
				origins[object.ID()] = pivot.Add(maths.V(east, 0, -north))
				scene.Origins = origins
			}
			if object.Grounded() && !object.AbsolutePosition() && view.mode != "turn" {
				work.caption = "Moving · terrain height updates on release"
			}
			turn := maths.Rotation(maths.YAxis, -(x1-x0)*180*maths.Degree)
			for i := range scene.Batches {
				batch := &scene.Batches[i]
				if !batch.Authored || batch.Author.Collection+"/"+batch.Author.Name != work.play.selected {
					continue
				}
				batch.Places = append([]maths.Subspace(nil), base.Batches[i].Places...)
				batch.HasBounds = false
				for j := range batch.Places {
					place := &batch.Places[j]
					if view.mode == "turn" {
						place.Origin = pivot.Add(turn.Out(place.Origin.Sub(pivot)))
						place.Orientation = turn.Then(place.Orientation)
					} else {
						place.Origin = place.Origin.Add(maths.V(east, 0, -north))
					}
				}
			}
		}
	} else {
		at, target := base.Eye.Origin, base.Focus
		moved, err := draft.PreviewCamera([3]float64{at.X, -at.Z, at.Y}, [3]float64{target.X, -target.Z, target.Y}, -dx*.35, dy*.35)
		if err != nil {
			return
		}
		eye, err := maths.View(maths.V(moved[0], moved[2], -moved[1]), target, maths.YAxis)
		if err != nil {
			return
		}
		scene.Eye = *eye
	}
	live.scene, live.dirty = scene, true
	live.timing.moved(started)
	work.invalidate()
}

// cancelViewportGesture retires a drag before replacing its projection.
func (work *Workspace) cancelViewportGesture() {
	live := &work.working
	live.timing = workingTiming{}
	if !work.gesturing() && live.gesture == nil {
		return
	}
	if live.gesture != nil {
		live.scene, live.gesture, live.dirty = live.gesture, nil, true
	}
	work.viewer.dragging, work.viewer.dragObject = false, false
	if work.viewer.ball != nil {
		work.viewer.ball.finishMap(false)
		work.viewer.ball.dragging = false
	}
	if work.viewer.travel != nil {
		work.viewer.travel.finishMap(false)
		work.viewer.travel.dragging = false
	}
	if work.ui != nil {
		work.ui.Window().Context().ReleasePointer(work.viewer)
	}
}

// endWorking ends the gesture. The edit is kept only when committed is set
// and the document text has changed.
func (work *Workspace) endWorking(committed bool) {
	live := &work.working
	if live.detail != "" && !work.viewer.dragObject {
		work.finishCameraInspection(committed)
		return
	}
	live.timing = workingTiming{}
	if live.gesture == nil {
		return
	}
	committed = committed && live.source != work.state.document.Text()
	if committed {
		work.rememberWorking(live.gesture)
		if work.viewer.dragObject && live.scene != nil && live.scene != live.gesture && len(live.gesture.Batches) > 0 {
			live.preview = &workingPreview{document: work.state.document, scene: work.withStagedCamera(live.scene), source: work.state.document.Text(), path: work.state.path, notice: live.notice}
		}
	}
	if !committed {
		live.scene = live.gesture
	}
	live.gesture = nil
	live.dirty = true
	if committed {
		live.waiting = true
	} else if live.waiting {
		work.pending.remember(work.state, work.specimen, time.Now())
		work.wakeNavigation(navigationDelay)
	}
	work.invalidate()
}

// updateWorking runs on the app thread after its GPU device has become available.
func (work *Workspace) updateWorking() {
	if work.composition != "play" || work.closing {
		return
	}
	live := &work.working
	if live.rollback != nil && (live.rollback.document != work.state.document || live.rollback.path != work.state.path) {
		live.rollback = nil
	}
	if live.gesture == nil && (live.path != work.state.path || !work.state.document.Matches(live.source)) {
		work.resetWorkingFrame()
		live.detailFallback = nil
		live.detailRetry = nil
		text := work.state.document.Text()
		if live.inspection != nil {
			live.inspection = nil
			work.refreshLayout = true
		}
		if live.preview != nil && (live.preview.document != work.state.document || live.preview.source != text || live.preview.path != work.state.path) {
			live.preview = nil
		}
		live.provisional = false
		live.plan, live.fullScene = nil, false
		live.stageAuthoring = nil
		if live.stager != nil {
			live.stager.discard()
			live.staging = false
		}
		live.source, live.path = text, work.state.path
		live.failure, live.notice = "", ""
		live.parsed = nil
		kind := work.state.document.Kind()
		if kind == "set" || kind == "legacy-set" || kind == "job" {
			if build := work.stagedBuilder(); build != nil {
				work.requestStaged(build, live.preview == nil)
				return
			}
		}
		if work.openCheck == work.state.document {
			work.validate()
		}
		if kind == "job" || live.cameraName != "" || live.cameraPath != "" {
			live.scene = nil
			live.rollback = nil
			live.preview = nil
			live.failure = "working view does not yet display job documents"
			if live.cameraName != "" {
				live.failure = "the staging helper is required to open camera " + live.cameraName
			}
			if live.cameraPath != "" {
				live.failure = "the staging helper is required to open a job camera path"
			}
			if err := work.state.document.Check(); err != nil {
				live.failure = err.Error()
			}
			work.reportWorkingFailure()
			return
		}
		if live.cache == nil {
			live.cache = placement.NewDisplayCache()
		}
		read, err := set.Read(work.state.path, []byte(text))
		live.scene = nil
		if err == nil {
			live.parsed = &read
			work.refreshLayout = true
			live.scene, err = live.cache.Scene(read)
			if err == nil {
				live.notice = localWorkingNotice(read, live.scene)
			}
		}
		if err != nil {
			live.failure = err.Error()
			live.rollback = nil
			live.preview = nil
			work.reportWorkingFailure()
			return
		}
		live.rollback = nil
		live.preview = nil
		live.dirty = true
		live.waiting = work.play.renderedText != text
	}
	work.drainStaged()
	if live.scene == nil || live.failure != "" {
		return
	}
	active := live.photograph == nil || work.gesturing() || live.waiting || (live.mode != "" && live.mode != "rendered")
	if !active {
		if live.displayed {
			work.viewer.set(live.photograph)
			live.displayed = false
			work.caption = chooseWorkingView
		}
		return
	}
	if live.plan != nil {
		work.updateAuthoredMap()
		return
	}
	if live.gpu == nil {
		provider := work.host.GPUContextProvider()
		if provider == nil || provider.Device().IsNil() {
			return
		}
		var err error
		live.gpu, err = workingview.New(provider)
		if err != nil {
			live.failure = err.Error()
			work.diagnostic = live.failure
			return
		}
		work.host.TrackResource(live.gpu)
		live.gpu.ForceLateReadbacks(live.late)
		live.dirty = true
	}
	aspect := float64(live.scene.Width) / float64(live.scene.Height)
	bounds := fitImageBounds(work.viewer.Bounds(), aspect)
	if bounds.IsEmpty() {
		bounds = work.viewer.Bounds()
	}
	w, h := max(1, int(bounds.Width())), max(1, int(bounds.Height()))
	if w <= 1 || h <= 1 {
		return
	}
	if w != live.width || h != live.height {
		live.width, live.height, live.dirty = w, h, true
	}
	if live.dirty {
		live.scene.SetViewport(w, h)
		mode := live.mode
		if mode == "" || mode == "rendered" {
			mode = "edges"
		}
		_ = live.gpu.SetMode(mode)
		bg, edge := appearance.Panel, appearance.Cyan
		live.gpu.SetPalette([4]float32{bg.R, bg.G, bg.B, bg.A}, [4]float32{edge.R, edge.G, edge.B, edge.A})
		var picture image.Image
		var err error
		drawStarted := workingClock()
		if len(live.scene.Batches) > 0 {
			picture, err = live.gpu.DrawStable(live.scene, w, h)
		} else {
			picture, err = live.gpu.Draw(live.scene, w, h)
		}
		if !drawStarted.IsZero() {
			observability.Observe("ferruccio_working_draw_attempt_seconds", time.Since(drawStarted).Seconds())
		}
		if errors.Is(err, workingview.ErrReadbackLate) {
			if !drawStarted.IsZero() {
				observability.Add("ferruccio_working_readback_late_total", 1)
			}
			// The frame stays dirty and the next tick draws it again.
			work.host.RequestRedraw()
			return
		}
		if err != nil {
			live.timing = workingTiming{}
			live.failure = err.Error()
			work.diagnostic = live.failure
			return
		}
		live.dirty = false
		if active {
			fallback := live.gpu.Stats().SplitWorldFallback
			if fallback != live.splitWorldFallback {
				live.splitWorldFallback = fallback
				work.refreshLayout = true
			}
			work.viewer.setWorking(picture, aspect)
			work.viewer.volumeBounds = visibleVolumeBounds(live.scene)
			work.viewer.lightMarkers = visibleLightMarkers(live.scene)
			if !drawStarted.IsZero() {
				observability.Observe("ferruccio_working_frame_seconds", live.gpu.Stats().Elapsed.Seconds())
				live.timing.pixelsReady(time.Now())
			}
			live.displayed = true
			work.viewer.SetNavigable(!live.provisional)
		}
	}
	if live.displayed {
		work.caption = work.workingCaption()
		work.viewer.SetNavigable(!live.provisional)
	}
}

// reportWorkingFailure keeps the error visible on the status line and in details.
func (work *Workspace) reportWorkingFailure() {
	work.diagnostic = "Staging failed: " + work.working.failure
	work.caption = "Interactive geometry unavailable · see View details"
}

// workingCaption keeps staged limitations visible after drawing and validation.
func (work *Workspace) workingCaption() string {
	if work.fullCancellationNote != "" && work.status == work.fullCancellationNote {
		return work.fullCancellationNote
	}
	if plan := work.working.plan; plan != nil {
		return plan.caption()
	}
	if work.working.provisional {
		return "Provisional · terrain, camera and dependencies updating"
	}
	if work.working.inspection != nil {
		return "Working camera moved · renders keep the authored camera"
	}
	if _, name, ok := strings.Cut(work.working.detail, "/"); ok {
		return "Scene detail · " + name
	}
	if work.working.cameraPath != "" {
		return "Staged working view · job camera path · see View details"
	}
	if work.working.cameraName != "" {
		return fmt.Sprintf("Staged working view · job camera %q · see View details", work.working.cameraName)
	}
	if work.workingNotice() != "" {
		return "Staged working view · see View details"
	}
	return "Working view · opaque native geometry · Slayer lighting refines after release"
}

// WorkingDiagnostic exposes the actual geometry/device failure to native regression.
func (work *Workspace) WorkingDiagnostic() string { return work.working.failure }

// LayoutReady reports that the native view has received its first real layout.
func (work *Workspace) LayoutReady() bool { return !work.viewer.Bounds().IsEmpty() }

const splitWorldFallbackNotice = "High-precision world drawing is unavailable · Geometry at large scene coordinates may lose detail"

func (work *Workspace) workingPrecisionNotice() string {
	live := &work.working
	if live.displayed && live.plan == nil && live.scene != nil && live.scene.SplitWorldVertices && live.splitWorldFallback {
		return splitWorldFallbackNotice
	}
	return ""
}

func (work *Workspace) workingNotice() string {
	notice, precision := work.working.notice, work.workingPrecisionNotice()
	if precision == "" {
		return notice
	}
	if notice == "" {
		return precision
	}
	return notice + " · " + precision
}
