package workspace

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/janearc/slayer/ferruccio/internal/draft"
	"github.com/janearc/slayer/ferruccio/native/internal/settings"
	"github.com/janearc/slayer/ferruccio/native/internal/stagewire"
	"github.com/janearc/slayer/session/placement"
	"github.com/janearc/slayer/set"
)

// stagedBuilder captures the helper and settings without accessing UI state
// from its worker. A missing helper leaves the existing display refusal intact.
func (work *Workspace) stagedBuilder() stageBuild {
	if work.working.stageOverride != nil {
		return work.working.stageOverride
	}
	executable, err := os.Executable()
	if err != nil {
		return nil
	}
	helper := filepath.Join(filepath.Dir(executable), "ferruccio-stage")
	info, err := os.Stat(helper)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&0111 == 0 {
		return nil
	}
	tools := work.renderTools()
	processorSetting := work.resolved().Get(settings.Processors).Effective()
	processors, err := strconv.Atoi(processorSetting)
	if processorSetting == settings.ProcessorAll {
		processors, err = min(runtime.NumCPU(), 8), nil
	}
	if err != nil || processors < 1 || processors > 8 {
		processors = 2
	}
	if work.working.stageCache == nil {
		work.working.stageCache = &stagewire.Session{}
	}
	cache := work.working.stageCache
	camera := work.workingStageCamera()
	detail := work.working.detail
	return func(ctx context.Context, path, source, kind string) (*stagedView, error) {
		snapshot, err := cache.RunDetail(ctx, helper, path, source, kind, tools.Cache, tools.Shelf, tools.Data, processors, camera, detail)
		if err != nil {
			return nil, err
		}
		return &stagedView{scene: &snapshot.Scene, notice: snapshot.Notice, frame: snapshot.Frame}, nil
	}
}

func (work *Workspace) canRetryStaging() bool {
	live := &work.working
	return live.stager != nil && (live.failure != "" || live.detailRetry != nil) && !live.staging &&
		live.path == work.state.path && work.state.document.Matches(live.source)
}

// retryStaging reopens the producer's paths without modifying the draft.
func (work *Workspace) retryStaging() {
	if !work.canRetryStaging() {
		return
	}
	if retry := work.working.detailRetry; retry != nil {
		work.stageSceneDetail(retry.id, false)
		return
	}
	if build := work.stagedBuilder(); build != nil {
		work.requestStaged(build, false)
	}
}

// refreshStagedDetail keeps the original fallback across setting changes
// that supersede an active or previously failed detail request.
func (work *Workspace) refreshStagedDetail() bool {
	live := &work.working
	if live.path != work.state.path || !work.state.document.Matches(live.source) {
		return false
	}
	if live.staging && live.detail != "" {
		work.stageSceneDetail(live.detail, false)
		return true
	}
	if live.detailRetry != nil && work.canRetryStaging() {
		work.retryStaging()
		return true
	}
	return false
}

// requestStaged starts a captured build without keeping an old scene current.
// An initial request can accept the lightweight display before starting a helper.
func (work *Workspace) requestStaged(build stageBuild, initial bool) {
	live := &work.working
	if live.path != work.state.path || !work.state.document.Matches(live.source) {
		work.resetWorkingFrame()
	}
	live.detailRetry = nil
	if live.stager == nil {
		live.stager = newStageLoader(nil)
		live.stager.notify = work.host.RequestRedraw
	}
	live.source, live.path = work.state.document.Text(), work.state.path
	live.scene, live.failure, live.staging, live.displayed = nil, "", true, false
	live.notice = ""
	live.provisional, live.plan = false, nil
	if preview := live.preview; preview != nil && preview.document == work.state.document && preview.path == live.path && preview.source == live.source {
		live.scene, live.notice, live.provisional = preview.scene, preview.notice, true
		live.dirty, live.waiting = true, true
	}
	work.viewer.SetNavigable(false)
	live.preview = nil
	prepare := work.stagedAuthoring(initial)
	subject := "current scene"
	if live.requestedFrame > 0 {
		subject = fmt.Sprintf("film frame %d", live.requestedFrame)
	} else if _, name, ok := strings.Cut(live.detail, "/"); ok {
		subject = "detail for " + name
	} else if live.fullScene {
		subject = "full 3D scene"
	}
	view := "Interactive geometry"
	if work.viewer.source != nil {
		view = "Previous view"
	}
	work.caption = view + " · staging " + subject + "…"
	if live.provisional {
		work.caption = work.workingCaption()
	}
	work.diagnostic = "Staging " + subject + " without rendering"
	readyAt := time.Now().Add(250 * time.Millisecond)
	live.stager.requestPreparedBuild(live.path, live.source, work.state.document.Kind(), prepare, func(ctx context.Context, path, source, kind string) (*stagedView, error) {
		timer := time.NewTimer(time.Until(readyAt))
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timer.C:
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		return build(ctx, path, source, kind)
	})
	work.invalidate()
}

// drainStaged accepts only a detached result for the current path and text.
func (work *Workspace) drainStaged() bool {
	live := &work.working
	if live.stager == nil {
		return false
	}
	result, ok := live.stager.take()
	// A completed stage cannot precede its preparation check. Poll the
	// handoff after taking the result so a just-finished check is not lost.
	work.drainStagedAuthoring()
	if !ok {
		return false
	}
	live.staging = false
	live.provisional = false
	live.plan = nil
	live.stageAuthoring = nil
	if result.path != work.state.path || result.source != work.state.document.Text() {
		live.detailFallback = nil
		_ = result.view.close()
		live.scene, live.displayed = nil, false
		return true
	}
	work.cancelViewportGesture()
	live.rollback = nil
	live.poseHistory = nil
	if result.err == nil && (result.view == nil || result.view.scene == nil || result.view.owner != nil) {
		result.err = fmt.Errorf("staging helper returned no detached scene")
	}
	if result.err == nil && live.requestedFrame != 0 &&
		(result.view.frame == nil || result.view.frame.Number != live.requestedFrame) {
		result.err = fmt.Errorf("staging helper returned the wrong film frame")
	}
	if result.err != nil {
		_ = result.view.close()
		if work.restoreDetailFallback(result.err) {
			return true
		}
		live.scene, live.failure = nil, result.err.Error()
		work.reportWorkingFailure()
	} else {
		live.scene, live.failure = result.view.scene, ""
		live.frame = result.view.frame
		live.plan = result.view.plan
		if live.plan != nil {
			live.plan.retainFraming(live.planFraming, live.scene.Objects, work.play.selected)
			live.planFraming = live.plan
		}
		work.refreshLayout = true
		live.notice = result.view.notice
		live.dirty, live.waiting = true, work.play.renderedText != result.source
		work.diagnostic = result.view.notice
		work.caption = work.workingCaption()
	}
	live.detailFallback = nil
	work.invalidate()
	return true
}

// closeStaged joins helper work before releasing its cache. A canceled
// source read owns no helper resources and can finish off the window thread.
func (work *Workspace) closeStaged() {
	work.resetWorkingFrame()
	work.working.detailFallback = nil
	work.working.detailRetry = nil
	work.working.rollback = nil
	work.working.preview = nil
	work.working.poseHistory = nil
	work.working.stageAuthoring = nil
	if loader := work.working.stager; loader != nil {
		if loader.close() {
			work.working.retiredPreparation.Go(loader.wait)
		} else {
			loader.wait()
		}
		work.working.stager = nil
		work.working.staging = false
	}
	if work.working.stageCache != nil {
		if err := work.working.stageCache.Close(); err != nil {
			work.diagnostic = err.Error()
		}
		work.working.stageCache = nil
	}
}

// QuitDesktop stops staging before a programmatic desktop exit.
func (work *Workspace) QuitDesktop() {
	work.closeStaged()
	work.host.Quit()
}

// Close releases private staging resources after the desktop loop exits.
// The window-close hook also calls this path, so closing twice is harmless.
func (work *Workspace) Close() {
	work.closeStaged()
	work.working.retiredPreparation.Wait()
}

// stageAuthoring is a one-request handoff for controls that need a parsed set
// before the helper returns the full staged geometry.
type stageAuthoring struct {
	document *draft.Document
	ready    <-chan *set.Set
	checked  <-chan error
}

// stagedAuthoring captures source reads and the optional lightweight build.
func (work *Workspace) stagedAuthoring(initial bool) stagePrepare {
	work.working.stageAuthoring = nil
	if !initial && !work.working.provisional {
		return nil
	}
	document := work.state.document
	detail := work.working.detail
	snapshot := *document
	snapshot.Replace(work.working.source)
	useMap := !work.working.fullScene
	var cache *placement.DisplayCache
	if initial && useMap && document.Kind() != "job" && work.working.cameraName == "" && work.working.cameraPath == "" {
		if work.working.cache == nil {
			work.working.cache = placement.NewDisplayCache()
		}
		cache = work.working.cache
	}
	ready := make(chan *set.Set, 1)
	checked := make(chan error, 1)
	work.working.stageAuthoring = &stageAuthoring{document, ready, checked}
	if work.working.stageReadSlot == nil {
		work.working.stageReadSlot = make(chan struct{}, 1)
	}
	slot := work.working.stageReadSlot
	notify := work.working.stager.wakePreparing
	return func(ctx context.Context, path, source, kind string) (*stagedView, error) {
		select {
		case slot <- struct{}{}:
			defer func() { <-slot }()
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		err := snapshot.Check()
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		checked <- err
		notify(ctx)
		if err != nil {
			return nil, err
		}
		if useMap && detail == "" {
			if view, err := prepareAuthoredMap(ctx, &snapshot, path); view != nil || err != nil {
				if view != nil && !view.plan.readOnly {
					imports, err := snapshot.HasImportedGeometry()
					if err == nil && !imports && ctx.Err() == nil {
						if parsed, err := set.Read(path, []byte(source)); err == nil && ctx.Err() == nil {
							ready <- &parsed
							view.plan.editable = true
						}
					}
					if !view.plan.editable {
						view.notice += " Placement edits require a checked terrain window."
					}
				}
				return view, err
			}
		}
		if kind == "job" || ctx.Err() != nil {
			return nil, ctx.Err()
		}
		parsed, err := set.Read(path, []byte(source))
		if err != nil || ctx.Err() != nil {
			return nil, ctx.Err()
		}
		var view *stagedView
		if cache != nil {
			if scene, err := cache.Scene(parsed); err == nil {
				view = &stagedView{scene: scene, notice: localWorkingNotice(parsed, scene)}
			}
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		ready <- &parsed
		notify(ctx)
		return view, nil
	}
}

// authoringSet consumes completed source preparation before a control
// checks its placement. Input events can arrive between window ticks.
func (work *Workspace) authoringSet() *set.Set {
	work.drainStagedAuthoring()
	live := &work.working
	if live.path != work.state.path || !work.state.document.Matches(live.source) {
		return nil
	}
	return live.parsed
}

func (work *Workspace) drainStagedAuthoring() {
	live := &work.working
	authoring := live.stageAuthoring
	if authoring == nil {
		return
	}
	if authoring.document != work.state.document || live.path != work.state.path || !work.state.document.Matches(live.source) {
		live.stageAuthoring = nil
		return
	}
	select {
	case err := <-authoring.checked:
		authoring.checked = nil
		if work.openCheck == authoring.document {
			work.openCheck = nil
			work.refreshRenderBlocked(err == nil)
			if work.status == "Checking YAML…" {
				work.applyOpenCheck(err)
			}
		}
	default:
	}
	if authoring.checked != nil {
		return
	}
	select {
	case live.parsed = <-authoring.ready:
		live.stageAuthoring = nil
		work.refreshLayout = true
		work.invalidate()
	default:
	}
}
