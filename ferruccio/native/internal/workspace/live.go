package workspace

import (
	"errors"
	"fmt"
	"github.com/gogpu/ui/core/dropdown"
	"github.com/gogpu/ui/widget"
	"github.com/janearc/slayer/ferruccio/internal/draft"
	"github.com/janearc/slayer/ferruccio/native/internal/appearance"
	"github.com/janearc/slayer/ferruccio/native/internal/workingview"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/session/placement"
	"github.com/janearc/slayer/set"
	"image"
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
	notice                      string
	dirty, waiting, displayed   bool
	width, height               int
	stager                      *stageLoader
	staging                     bool
	stageOverride               stageBuild
	rollback                    *workingRollback
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

func (work *Workspace) rememberWorking(scene *placement.DisplayScene) {
	live := &work.working
	if scene == nil || live.source == "" || live.path != work.state.path {
		return
	}
	if live.rollback != nil && live.rollback.document == work.state.document && live.rollback.source == live.source && live.rollback.path == live.path {
		return
	}
	live.rollback = &workingRollback{document: work.state.document, scene: scene, parsed: live.parsed, source: live.source, path: live.path, notice: live.notice}
}

func (work *Workspace) restoreWorking(text string) bool {
	live, saved := &work.working, work.working.rollback
	if saved == nil || saved.document != work.state.document || saved.source != text || saved.path != work.state.path {
		return false
	}
	if live.stager != nil {
		live.stager.discard()
	}
	var inverse *workingRollback
	if live.scene == saved.scene && live.path == saved.path && live.source != saved.source {
		inverse = &workingRollback{document: saved.document, scene: live.scene, parsed: live.parsed, source: live.source, path: live.path, notice: live.notice}
	}
	live.scene, live.parsed = saved.scene, saved.parsed
	live.source, live.path, live.notice = saved.source, saved.path, saved.notice
	live.failure, live.staging, live.displayed = "", false, false
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
	work.working.mode = mode
	work.refreshLayout = true
	work.working.dirty = true
	work.invalidate()
	return nil
}

// WorkingReady reports an actual GPU frame for the current working scene.
func (work *Workspace) WorkingReady() bool {
	return work.working.gpu != nil && work.working.displayed && !work.working.dirty && work.working.failure == ""
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
	selected := 0
	for i, mode := range modes {
		if work.working.mode == mode {
			selected = i
		}
	}
	choice := dropdown.New(dropdown.Items(labels...), dropdown.Selected(selected), dropdown.PainterOpt(appearance.DropdownPainter{Theme: work.theme, Size: work.size}), dropdown.OnChange(func(i int, _ string) {
		if i >= 0 && i < len(modes) {
			work.problem(work.SetWorkingMode(modes[i]))
		}
	}))
	choice.SetID("Viewport display")
	return choice
}

// workingCurrent reports whether the working scene matches the document text
// and path.
func (work *Workspace) workingCurrent() bool {
	live := &work.working
	return live.scene != nil && live.source == work.state.document.Text() && live.path == work.state.path && (live.displayed || work.play.renderedText == live.source)
}

// workingPick returns the object under x and y in the working scene, or in
// the last rendered scene. It returns the empty string when nothing is there.
func (work *Workspace) workingPick(x, y float64) string {
	if work.workingCurrent() {
		return work.working.scene.Pick(x, y)
	}
	if work.play.geometry != nil && work.play.renderedText == work.state.document.Text() {
		return work.play.geometry.Pick(x, y)
	}
	return ""
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

// beginWorking starts a gesture by keeping a copy of the working scene as
// the base for moves.
func (work *Workspace) beginWorking() {
	if !work.workingCurrent() {
		return
	}
	if work.refuseWhileFull("changing the scene") {
		return
	}
	if group, ok := work.selectedGroup(); ok && work.viewer.dragObject {
		work.status = group.Name + " · generated members cannot be dragged separately"
		if _, editable := work.state.document.GroupArea(group.ID); editable {
			work.status = group.Name + " · use Region to edit this group area"
		}
		work.invalidate()
		return
	}
	if object, ok := work.selectedObject(); ok && object.Grounded() && work.viewer.dragObject {
		if _, err := gestureOrigin(work.working.scene, object); err != nil {
			work.problem(err)
			return
		}
		if work.play.mode == "turn" {
			if _, said := object.Heading(); !said {
				work.problem(fmt.Errorf("this terrain object has a generated heading; set an explicit heading in its source before turning it"))
				return
			}
		}
	}
	if work.Running() {
		work.stopPreview()
	}
	work.working.gesture = copyDisplay(work.working.scene)
	work.working.dirty = true
}

// moveWorking changes cached transforms only; the YAML history stays untouched.
func (work *Workspace) moveWorking() {
	live, view := &work.working, work.viewer
	base := live.gesture
	if base == nil {
		return
	}
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
			if object, ok := work.selectedObject(); ok && len(base.Batches) > 0 {
				origin, originErr := gestureOrigin(base, object)
				if originErr != nil {
					return
				}
				east, north, err = base.DragHeight(origin.Y, x0, y0, x1, y1)
			} else {
				east, north, err = base.Drag(work.play.selected, x0, y0, x1, y1)
			}
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
		if object, ok := work.selectedObject(); ok {
			pivot, err := gestureOrigin(base, object)
			if err != nil {
				return
			}
			if object.Grounded() && view.mode != "turn" {
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
	work.invalidate()
}

// endWorking ends the gesture. The edit is kept only when committed is set
// and the document text has changed.
func (work *Workspace) endWorking(committed bool) {
	live := &work.working
	if live.gesture == nil {
		return
	}
	committed = committed && live.source != work.state.document.Text()
	if committed {
		work.rememberWorking(live.gesture)
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
	text := work.state.document.Text()
	if live.rollback != nil && (live.rollback.document != work.state.document || live.rollback.path != work.state.path) {
		live.rollback = nil
	}
	if live.gesture == nil && (live.source != text || live.path != work.state.path) {
		if live.stager != nil {
			live.stager.discard()
			live.staging = false
		}
		live.source, live.path = text, work.state.path
		live.failure, live.notice = "", ""
		live.parsed = nil
		if work.state.document.Kind() == "job" {
			if err := work.state.document.Check(); err == nil {
				if build := work.stagedBuilder(); build != nil {
					work.requestStaged(build)
					return
				}
			}
			live.scene = nil
			live.rollback = nil
			live.failure = "working view does not yet display job documents"
			if err := work.state.document.Check(); err != nil {
				live.failure = err.Error()
			}
			work.diagnostic = "Interactive geometry unavailable: " + live.failure
			work.caption = "Interactive geometry unavailable · see the reason below"
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
		}
		if err != nil {
			if work.state.document.Check() == nil {
				if build := work.stagedBuilder(); build != nil {
					work.requestStaged(build)
					return
				}
			}
			live.failure = err.Error()
			live.rollback = nil
			work.diagnostic = "Interactive geometry unavailable: " + err.Error()
			work.caption = "Interactive geometry unavailable · see the reason below"
			return
		}
		live.rollback = nil
		live.dirty = true
		live.waiting = work.play.renderedText != text
	}
	work.drainStaged()
	if live.scene == nil || live.failure != "" {
		return
	}
	active := live.photograph == nil || work.gesturing() || live.waiting || (live.mode != "" && live.mode != "rendered")
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
	if work.viewer.aspect <= 0 {
		work.viewer.aspect = float64(live.scene.Width) / float64(live.scene.Height)
	}
	bounds := work.viewer.imageBounds()
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
		picture, err := live.gpu.Draw(live.scene, w, h)
		if errors.Is(err, workingview.ErrReadbackLate) {
			// The frame stays dirty and the next tick draws it again.
			work.host.RequestRedraw()
			return
		}
		if err != nil {
			live.failure = err.Error()
			work.diagnostic = live.failure
			return
		}
		live.dirty = false
		if active {
			work.viewer.setWorking(picture, work.viewer.aspect)
			live.displayed = true
			work.viewer.SetNavigable(true)
		}
	}
	if !active && live.displayed && live.photograph != nil {
		work.viewer.set(live.photograph)
		live.displayed = false
		work.caption = "Grab something · drag to change it · scroll to approach · ⌘Z undo"
	}
	if live.displayed {
		work.caption = work.workingCaption()
		work.viewer.SetNavigable(true)
	}
}

// workingCaption keeps staged limitations visible after drawing and validation.
func (work *Workspace) workingCaption() string {
	if work.working.notice != "" {
		return "Staged working view · see View limitations"
	}
	return "Working view · opaque native geometry · Slayer lighting refines after release"
}

// WorkingDiagnostic exposes the actual geometry/device failure to native regression.
func (work *Workspace) WorkingDiagnostic() string { return work.working.failure }

// LayoutReady reports that the native view has received its first real layout.
func (work *Workspace) LayoutReady() bool { return !work.viewer.Bounds().IsEmpty() }
