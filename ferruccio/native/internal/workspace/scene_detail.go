package workspace

import (
	"fmt"
	"strings"
)

type detailFallback struct {
	view               stagedView
	inspection         *inspectionCamera
	detail, mode       string
	displayed, waiting bool
}

type failedDetail struct {
	id, message string
}

func (work *Workspace) canShowSceneDetail() bool {
	id := work.play.selected
	collection, name, ok := strings.Cut(id, "/")
	if !ok || name == "" || id == work.working.detail ||
		(collection != "scene.set.things" && collection != "scene.set.stands") {
		return false
	}
	if work.working.plan != nil {
		return true
	}
	read := work.working.parsed
	if read == nil || read.InSpace() {
		return false
	}
	_, grounded := read.Ground()
	return grounded
}

func (work *Workspace) showSceneDetail() {
	if !work.canShowSceneDetail() {
		return
	}
	work.stageSceneDetail(work.play.selected, true)
}

func (work *Workspace) stageSceneDetail(id string, navigating bool) {
	live := &work.working
	previous := live.detail
	live.detail = id
	build := work.stagedBuilder()
	if build == nil {
		live.detail = previous
		work.problem(fmt.Errorf("scene detail requires the staging helper"))
		return
	}
	if navigating {
		work.stopPreview()
	}
	work.cancelViewportGesture()
	if live.detailFallback == nil && live.scene != nil &&
		live.path == work.state.path && work.state.document.Matches(live.source) {
		live.detailFallback = &detailFallback{
			view:       stagedView{scene: live.scene, notice: live.notice, plan: live.plan},
			inspection: live.inspection, detail: previous, mode: live.mode,
			displayed: live.displayed, waiting: live.waiting,
		}
	}
	live.preview, live.inspection = nil, nil
	if live.mode == "" || live.mode == "rendered" {
		_ = work.SetWorkingMode("solid")
	}
	work.requestStaged(build, false)
	// Settings and Retry can start staging from controls in the current root.
	work.refreshLayout = true
}

func (work *Workspace) restoreDetailFallback(err error) bool {
	live := &work.working
	fallback := live.detailFallback
	if fallback == nil {
		return false
	}
	live.detailFallback = nil
	live.detailRetry = &failedDetail{live.detail, err.Error()}
	live.scene, live.notice = fallback.view.scene, fallback.view.notice
	live.plan = fallback.view.plan
	live.detail, live.mode, live.inspection = fallback.detail, fallback.mode, fallback.inspection
	live.displayed, live.waiting, live.dirty = fallback.displayed, fallback.waiting, true
	work.viewer.SetNavigable(fallback.displayed)
	work.diagnostic = "Scene detail unavailable: " + err.Error()
	work.caption = work.workingCaption()
	work.refreshLayout = true
	work.invalidate()
	return true
}

func (work *Workspace) showSceneMap() {
	work.stopPreview()
	work.cancelViewportGesture()
	live := &work.working
	live.detail, live.detailFallback = "", nil
	live.detailRetry = nil
	live.fullScene = false
	live.source, live.path = "", ""
	live.preview, live.inspection = nil, nil
	work.invalidate()
	work.rebuild()
}
