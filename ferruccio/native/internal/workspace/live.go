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
	scene, gesture              *placement.DisplayScene
	gpu                         *workingview.View
	photograph                  image.Image
	source, path, mode, failure string
	dirty, waiting, displayed   bool
	width, height               int
	// late is how many readbacks the next GPU view is told to miss.
	late int
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
			east, north, err = base.Drag(work.play.selected, x0, y0, x1, y1)
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
	if live.gesture == nil && (live.source != text || live.path != work.state.path) {
		live.source, live.path = text, work.state.path
		live.failure = ""
		if live.cache == nil {
			live.cache = placement.NewDisplayCache()
		}
		read, err := set.Read(work.state.path, []byte(text))
		live.scene = nil
		if err == nil {
			live.scene, err = live.cache.Scene(read)
		}
		if err != nil {
			live.failure = err.Error()
			work.diagnostic = "Interactive geometry unavailable: " + err.Error()
			work.caption = "Slayer can still draw this scene · interactive geometry is unavailable"
			return
		}
		live.dirty = true
		live.waiting = work.play.renderedText != text
	}
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
		work.caption = "Working view · opaque native geometry · Slayer lighting refines after release"
		work.viewer.SetNavigable(true)
	}
}

// WorkingDiagnostic exposes the actual geometry/device failure to native regression.
func (work *Workspace) WorkingDiagnostic() string { return work.working.failure }

// LayoutReady reports that the native view has received its first real layout.
func (work *Workspace) LayoutReady() bool { return !work.viewer.Bounds().IsEmpty() }
