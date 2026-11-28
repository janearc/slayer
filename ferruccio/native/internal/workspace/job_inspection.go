package workspace

import (
	"github.com/janearc/slayer/ferruccio/internal/draft"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/session/placement"
)

// inspectionCamera is the original pose of a locally moved working view.
type inspectionCamera struct {
	eye   maths.Subspace
	focus maths.Vec
}

func (work *Workspace) cameraInspectionOnly() bool {
	return work.working.detail != "" || work.state.document.Kind() == "job" ||
		work.working.cameraName != "" || work.working.cameraPath != ""
}

// withStagedCamera keeps placement history independent of temporary inspection.
func (work *Workspace) withStagedCamera(scene *placement.DisplayScene) *placement.DisplayScene {
	camera := work.working.inspection
	if scene == nil || camera == nil {
		return scene
	}
	staged := *scene
	staged.Eye, staged.Focus = camera.eye, camera.focus
	return &staged
}

func (work *Workspace) finishCameraInspection(keep bool) {
	live := &work.working
	base := live.gesture
	if base == nil {
		return
	}
	live.gesture, live.dirty = nil, true
	live.timing = workingTiming{}
	if !keep || live.scene == nil || live.path != work.state.path ||
		!work.state.document.Matches(live.source) ||
		(live.scene.Eye == base.Eye && live.scene.Focus == base.Focus) {
		live.scene = base
	} else if live.inspection == nil {
		live.inspection = &inspectionCamera{eye: base.Eye, focus: base.Focus}
		work.refreshLayout = true
	}
	work.invalidate()
}

func (work *Workspace) resetCameraInspection() {
	live := &work.working
	if live.inspection == nil || live.scene == nil {
		return
	}
	work.cancelViewportGesture()
	work.fullCancellationNote = ""
	live.scene = work.withStagedCamera(live.scene)
	live.inspection, live.dirty = nil, true
	work.refreshLayout = true
	work.invalidate()
}

// navigateDetail moves only the local detail camera.
func (work *Workspace) navigateDetail(yaw, pitch, zoom float64) error {
	live := &work.working
	if !work.viewer.navigable || work.photographVisible() || !work.workingCurrent() {
		return nil
	}
	scene := *live.scene
	if live.gesture != nil {
		scene = *live.gesture
	}
	eye, focus := scene.Eye.Origin, scene.Focus
	at, err := draft.PreviewCamera([3]float64{eye.X, -eye.Z, eye.Y}, [3]float64{focus.X, -focus.Z, focus.Y}, yaw, pitch)
	if err != nil {
		return err
	}
	eye = maths.V(at[0], at[2], -at[1])
	if zoom != 0 {
		eye = focus.Add(eye.Sub(focus).Scale(zoom))
	}
	view, err := maths.View(eye, focus, maths.YAxis)
	if err != nil {
		return err
	}
	if live.inspection == nil && live.gesture == nil {
		live.inspection = &inspectionCamera{eye: scene.Eye, focus: focus}
	}
	scene.Eye = *view
	live.scene, live.dirty = &scene, true
	work.refreshLayout = true
	work.invalidate()
	return nil
}
