package workspace

import "github.com/janearc/slayer/ferruccio/internal/preview"

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
	if work.queueObserved && update.State == work.queueState && update.Held == work.queueHeld {
		return
	}
	work.queueObserved = true
	work.queueSequence = update.Sequence
	work.queueState, work.queueHeld = update.State, update.Held
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
	return label
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
	reason := "preview queue is not configured"
	if work.options.Tools.Transport == "slayer" {
		reason = "shared renderer is not configured"
	}
	if work.options.Tools.Azazel != "" && work.options.Tools.Queue != "" && work.options.Evidence == "" {
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
		switch result.observation.State {
		case "cancelled":
			work.status = "Preview cancellation confirmed"
		case "succeeded":
			work.status = "Preview completed before cancellation"
		case "failed":
			work.status = "Preview job failed"
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
