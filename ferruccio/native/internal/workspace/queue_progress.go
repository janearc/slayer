package workspace

import (
	"fmt"

	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/janearc/slayer/ferruccio/internal/preview"
	"github.com/janearc/slayer/ferruccio/native/internal/appearance"
)

// renderProgress is an immutable observation of one durable own-job identity.
type renderProgress struct {
	update     preview.Update
	generation uint64
}

// drainProgress applies observations only on the UI thread.
func (work *Workspace) drainProgress() {
	for {
		select {
		case progress := <-work.progress:
			work.applyProgress(progress)
		default:
			return
		}
	}
}

// applyProgress never changes source, picking geometry or accepted image pixels.
func (work *Workspace) applyProgress(progress renderProgress) {
	if progress.generation != work.generation || work.cancel == nil {
		return
	}
	update := progress.update
	if update.Sequence == 0 || work.queueObserved && update.Sequence != work.queueSequence {
		return
	}
	if queueLabel(update.State) == "" {
		return
	}
	if work.queueObserved && update.State == work.queueState && update.Held == work.queueHeld && update.Progress == work.queueProgress {
		return
	}
	work.queueObserved = true
	work.queueSequence = update.Sequence
	work.queueState, work.queueHeld = update.State, update.Held
	work.queueProgress = update.Progress
	work.refreshLayout = true
	label := work.observedQueueText()
	work.status = label
	work.caption = work.retainedImageCaption(label)
	work.invalidate()
}

// queueLabel names owner observations without treating a claimed job as execution.
func queueLabel(state string) string {
	switch state {
	case "queued":
		return "Queued"
	case "waiting":
		return "Waiting for render admission"
	case "unavailable":
		return "Checking queue"
	case "running":
		return "Running"
	case "succeeded":
		return "Job succeeded · collecting preview"
	case "failed":
		return "Job failed"
	case "cancelled":
		return "Job cancelled"
	case "interrupted":
		return "Job interrupted"
	}
	return ""
}

// observedQueueText combines the current own-job state with its admission reason.
func (work *Workspace) observedQueueText() string {
	label := queueLabel(work.queueState)
	if work.queueHeld != "" {
		label += " · " + work.queueHeld
	}
	if work.queueState == "running" && work.queueProgress.Phase != "" {
		progress := work.queueProgress
		label += " · " + progress.Phase
		if progress.Total > 0 {
			label += fmt.Sprintf(" · %d%% (%d/%d)", int(float64(progress.Done)/float64(progress.Total)*100), progress.Done, progress.Total)
			if progress.Phase == "Rendering" && progress.Done == progress.Total {
				label += " · finishing"
			}
		}
	}
	return label
}

// renderProgressBar shows the observed phase, never inferred elapsed time.
func (work *Workspace) renderProgressBar() widget.Widget {
	progress := work.queueProgress
	if work.cancel == nil || work.queueState != "running" {
		return nil
	}
	if progress.Total <= 0 {
		return activeRenderMarker()
	}
	const width float32 = 160
	filled := width * float32(progress.Done) / float32(progress.Total)
	bar := primitives.HBox().Width(width).Height(10).Background(appearance.Raised)
	if filled > 0 {
		bar = primitives.HBox(primitives.Box().Width(filled).Height(10).Background(appearance.Cyan)).Width(width).Height(10).Background(appearance.Raised)
	}
	bar.Label(fmt.Sprintf("%s: %d of %d", progress.Phase, progress.Done, progress.Total))
	bar.SetID("Render progress")
	return bar
}

// activeRenderMarker reports work without redrawing at an animation cadence.
func activeRenderMarker() *primitives.BoxWidget {
	marker := primitives.Box().Width(10).Height(10).Rounded(5).Background(appearance.Cyan)
	marker.Label("Render active · progress total unavailable")
	marker.SetID("Render progress")
	return marker
}

// retainedImageCaption distinguishes queue metadata from newly accepted pixels.
func (work *Workspace) retainedImageCaption(state string) string {
	if work.output != "" {
		return "Previous render · " + state
	}
	if work.viewer != nil && work.viewer.source != nil {
		return "Previous image · " + state
	}
	return "No preview yet · " + state
}

// queueUnavailable keeps editing available when transport has not been configured.
func (work *Workspace) queueUnavailable() {
	reason := "choose a render queue in Settings"
	if work.options.Tools.Transport == "slayer" {
		reason = "shared renderer is not configured"
	}
	if work.renderTools().Azazel != "" && work.renderTools().Queue != "" && work.evidenceFolder() == "" {
		reason = "shared render destination is not configured"
	}
	work.status = "Ready to edit · " + reason
	work.caption = work.retainedImageCaption(reason)
	work.invalidate()
}

// startAutomaticPreview avoids a modal configuration error during first opening.
func (work *Workspace) startAutomaticPreview() {
	if !work.rendererConfigured() {
		work.queueUnavailable()
		return
	}
	work.problem(work.StartPreview())
}

// previewProgressText reports only observed queue state or local pending work.
func (work *Workspace) previewProgressText() string {
	if work.cancel != nil {
		if work.cancelRequested {
			if work.fullRunning {
				return work.retainedImageCaption("cancelling full render request")
			}
			return work.retainedImageCaption("cancelling previous preview request")
		}
		if work.queueObserved {
			return work.observedQueueText() + " · " + work.retainedImageCaption("waiting for completed pixels")
		}
		if work.options.Tools.Transport == "slayer" {
			return work.retainedImageCaption("preparing preview")
		}
		return work.retainedImageCaption("submitting preview request")
	}
	return "Change pending · waiting for input to settle"
}

func cancellationRenderName(full bool) string {
	if full {
		return "Full render"
	}
	return "Preview"
}

// finishCancellation labels only the client's verified own-process outcome.
func (work *Workspace) finishCancellation(result renderResult) {
	if result.err != nil && work.log != nil {
		_, _ = work.log.Write([]byte(result.err.Error() + "\n"))
	}
	if !result.observation.Terminal {
		work.status = "Cancellation requested · render outcome unconfirmed"
		work.diagnostic = work.status
		if result.err != nil {
			work.diagnostic += "\n" + result.err.Error()
		}
		if work.log != nil {
			work.diagnostic += "\n" + work.log.read()
		}
	} else {
		name := cancellationRenderName(result.full)
		switch result.observation.State {
		case "cancelled":
			work.status = name + " cancellation confirmed"
			if result.full {
				work.fullCancellationNote = work.status
			}
		case "succeeded":
			work.status = name + " completed before cancellation"
		case "failed":
			work.status = name + " job failed"
		default:
			work.status = queueLabel(result.observation.State)
		}
		if result.observation.State == "failed" && result.err != nil {
			work.diagnostic = result.err.Error()
			if work.log != nil {
				work.diagnostic += "\n" + work.log.read()
			}
		}
	}
	work.caption = work.retainedImageCaption(work.status)
}
