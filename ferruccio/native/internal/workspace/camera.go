package workspace

import (
	"fmt"
	"time"

	"github.com/gogpu/ui/widget"
)

// Navigate edits the current supported camera and queues its next preview.
func (work *Workspace) Navigate(yaw, pitch, zoom float64) error {
	if err := work.fullRenderRefusal("moving the camera"); err != nil {
		return err
	}
	if !work.viewer.navigable || work.viewer.source == nil || !work.state.document.CanNavigate() {
		return fmt.Errorf("render a scene with a supported camera before moving it")
	}
	before := work.state.document.Text()
	if err := work.state.document.Navigate(yaw, pitch, zoom); err != nil {
		return err
	}
	if work.state.document.Text() == before {
		return nil
	}
	work.stopPreview()
	work.editor.ApplyText(work.state.document.Text())
	work.needsCheck = false
	work.pending.remember(work.state, work.specimen, time.Now())
	work.caption = "Previous view · camera changed · preparing preview"
	work.status = "Camera changed · waiting for input to settle"
	work.diagnostic = ""
	work.wakeNavigation(navigationDelay)
	work.invalidate()
	return nil
}

// undoCamera restores editor history and queues a render for a navigable camera draft.
func (work *Workspace) undoCamera(ctx widget.Context, redo bool) {
	if work.editor == nil || work.state.document == nil {
		return
	}
	if work.refuseWhileFull("undoing a camera change") {
		return
	}
	before := work.state.document.Text()
	if redo {
		work.editor.Redo(ctx)
	} else {
		work.editor.Undo(ctx)
	}
	if work.state.document.Text() == before || !work.state.document.CanNavigate() {
		return
	}
	work.needsCheck = false
	work.pending.remember(work.state, work.specimen, time.Now())
	work.caption = "Previous render · history restored · preparing preview"
	work.status = "History restored · drawing your world"
	work.diagnostic = ""
	work.wakeNavigation(navigationDelay)
	work.invalidate()
}

// StopPreview requests cancellation on the UI thread for this window only.
func (work *Workspace) StopPreview() {
	work.cancelRender()
	work.invalidate()
}

// stopPreview makes room for a newer preview: it discards queued navigation
// and cancels this window's preview. A running full render is left alone;
// only cancelRender, which answers the user's own request, stops it.
func (work *Workspace) stopPreview() {
	if work.fullRunning {
		return
	}
	work.cancelRender()
}

// cancelRender discards queued navigation and cancels only this window's
// render, full or preview.
func (work *Workspace) cancelRender() {
	work.clearNavigation()
	work.generation++
	work.cancelRequested = work.cancel != nil
	if work.cancelRequested {
		work.cancel()
		work.status = "Cancelling preview request"
	} else {
		work.status = "Preview stopped"
	}
}

// clearNavigation stops the redraw timer without changing a running job.
func (work *Workspace) clearNavigation() {
	work.pending.clear()
	if work.navigationTimer != nil {
		work.navigationTimer.Stop()
		work.navigationTimer = nil
	}
}

// wakeNavigation wakes the demand-driven window without mutating UI state.
func (work *Workspace) wakeNavigation(delay time.Duration) {
	if work.navigationTimer != nil {
		work.navigationTimer.Stop()
	}
	work.navigationTimer = time.AfterFunc(delay, work.host.RequestRedraw)
}

// advanceNavigation starts at most one settled view after the old worker exits.
func (work *Workspace) advanceNavigation() {
	if work.closing {
		return
	}
	busy := work.cancel != nil || work.gesturing()
	if work.pending.take(work.state, work.specimen, time.Now(), busy) {
		work.startAutomaticPreview()
		return
	}
	if work.pending.document != nil && work.cancel == nil {
		work.wakeNavigation(max(time.Until(work.pending.due), 50*time.Millisecond))
	}
}

// finishPreview accepts a completed render only for the current view generation.
func (work *Workspace) finishPreview(result renderResult) {
	wasCancelling := work.cancelRequested
	work.cancel = nil
	work.cancelRequested = false
	work.fullRunning = false
	work.unresolvedJob, work.unresolvedPlan = 0, nil
	if result.err != nil && work.options.Tools.Transport != "slayer" && result.observation.Sequence != 0 && !result.observation.Terminal {
		work.unresolvedJob, work.unresolvedPlan = result.observation.Sequence, result.plan
	}
	if work.closing {
		work.releaseExample()
		work.host.Quit()
		return
	}
	if result.generation != work.generation {
		if wasCancelling && work.pending.document == nil {
			work.finishCancellation(result)
		}
		return
	}
	if result.err != nil {
		work.status = "Preview failed"
		if work.queueState == "cancelled" {
			work.status = "Preview cancelled"
		}
		if work.queueState == "interrupted" {
			work.status = "Preview interrupted"
		}
		work.diagnostic = failureDiagnostic(result.err, work.log.read())
		return
	}
	if !work.state.accepts(result.plan) || (result.plan.Kind == "catalogue" && result.plan.Specimen != work.specimen) {
		work.status = "Preview finished for an older draft; render again"
		if result.full {
			work.status = "Full render finished for an older draft; its picture is at " + result.plan.Output
		}
		return
	}
	if err := work.showImage(result.plan.Output); err != nil {
		work.status = "Cannot open the preview"
		work.diagnostic = err.Error()
		return
	}
	work.working.waiting = false
	work.working.dirty = true
	work.output = result.plan.Output
	work.acceptedGeneration = result.generation
	work.acceptedText, work.acceptedPath = work.state.document.Text(), work.state.path
	if work.composition == "play" {
		work.play.geometry = result.geometry
		work.play.renderedText = work.state.document.Text()
	}
	work.caption = "Slayer preview · one ray per side"
	if result.full {
		work.caption = fmt.Sprintf("Full Slayer render · %d × %d · %d rays per side · exact", result.plan.Profile.Width, result.plan.Profile.Height, result.plan.Profile.Rays)
	}
	if work.state.document.CanNavigate() {
		work.caption = "Drag to orbit · scroll to zoom · arrows / + / − · ⌘Z to undo"
		if work.composition == "play" {
			work.caption = "Grab something · drag to change it · scroll to approach · ⌘Z undo"
			if work.play.geometry == nil {
				work.caption = "Drag to look around · Source to edit this scene"
			}
		}
		if result.plan.Kind == "catalogue" {
			work.caption = result.plan.Specimen + " · " + work.caption
		}
		work.viewer.SetNavigable(true)
	}
	if result.full {
		_ = work.SetWorkingMode("rendered")
		work.caption = fmt.Sprintf("Full Slayer render · %d × %d · %d rays per side · exact", result.plan.Profile.Width, result.plan.Profile.Height, result.plan.Profile.Rays)
	}
	work.caption += work.reducedNote
	work.status = "Preview ready · " + work.state.document.Name()
	work.diagnostic = ""
}
