package workspace

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"github.com/janearc/slayer/ferruccio/native/internal/settings"
	"github.com/janearc/slayer/ferruccio/native/internal/stagewire"
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
	return func(ctx context.Context, path, source, kind string) (*stagedView, error) {
		snapshot, err := stagewire.Run(ctx, helper, path, source, kind, tools.Cache, tools.Shelf, tools.Data, processors)
		if err != nil {
			return nil, err
		}
		return &stagedView{scene: &snapshot.Scene, notice: snapshot.Notice}, nil
	}
}

// requestStaged starts a captured build without keeping an old scene current.
func (work *Workspace) requestStaged(build stageBuild) {
	live := &work.working
	if live.stager == nil {
		live.stager = newStageLoader(nil)
		live.stager.notify = work.host.RequestRedraw
	}
	live.source, live.path = work.state.document.Text(), work.state.path
	live.scene, live.failure, live.staging, live.displayed = nil, "", true, false
	live.notice = ""
	work.caption = "Interactive geometry · staging current scene…"
	work.diagnostic = "Staging current scene without rendering"
	live.stager.requestBuild(live.path, live.source, work.state.document.Kind(), func(ctx context.Context, path, source, kind string) (*stagedView, error) {
		timer := time.NewTimer(250 * time.Millisecond)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timer.C:
			return build(ctx, path, source, kind)
		}
	})
}

// drainStaged accepts only a detached result for the current path and text.
func (work *Workspace) drainStaged() bool {
	live := &work.working
	if live.stager == nil {
		return false
	}
	result, ok := live.stager.take()
	if !ok {
		return false
	}
	live.staging = false
	if result.path != work.state.path || result.source != work.state.document.Text() {
		_ = result.view.close()
		return true
	}
	live.rollback = nil
	if result.err == nil && (result.view == nil || result.view.scene == nil || result.view.owner != nil) {
		result.err = fmt.Errorf("staging helper returned no detached scene")
	}
	if result.err != nil {
		_ = result.view.close()
		live.scene, live.failure = nil, result.err.Error()
		work.diagnostic = "Interactive geometry unavailable: " + live.failure
		work.caption = "Interactive geometry unavailable · see the reason below"
	} else {
		live.scene, live.failure = result.view.scene, ""
		live.notice = result.view.notice
		live.dirty, live.waiting = true, work.play.renderedText != result.source
		work.diagnostic = result.view.notice
		work.caption = work.workingCaption()
	}
	work.invalidate()
	return true
}

// closeStaged cancels and joins the staging worker before window shutdown.
func (work *Workspace) closeStaged() {
	work.working.rollback = nil
	if work.working.stager == nil {
		return
	}
	work.working.stager.close()
	work.working.stager.wait()
	work.working.stager = nil
	work.working.staging = false
}
