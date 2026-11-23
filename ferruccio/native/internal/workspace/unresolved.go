package workspace

import (
	"context"
	"fmt"
	"time"
)

// unresolvedRecheckInterval is how often, while a refusal stands, the window
// asks the queue again about the earlier job. It was chosen: often enough that
// the refusal lifts soon after the job ends, rarely enough to be quiet.
const unresolvedRecheckInterval = 30 * time.Second

// unresolvedCheckWindow bounds one status read and cancel for an earlier job.
const unresolvedCheckWindow = 15 * time.Second

// resolution is the answer to one recheck of the earlier job.
type resolution struct {
	sequence uint64
	ended    bool
	err      error
}

// refuseWhileUnresolved refuses a new submission while a job this window
// started is live and unobserved. Each refusal starts a recheck, so asking to
// render again is also how the user asks for the job to be looked at.
func (work *Workspace) refuseWhileUnresolved() error {
	if work.unresolvedJob == 0 {
		return nil
	}
	work.checkUnresolved()
	work.status = fmt.Sprintf("Azazel job %d from this window may still be running; checking it again", work.unresolvedJob)
	return fmt.Errorf("Azazel job %d, started by this window, has no confirmed end, so no second job is submitted. The window is asking the queue about it now; press Preview again for the answer", work.unresolvedJob)
}

// checkUnresolved reads the earlier job's status and, if it is still live,
// asks for its cancellation again, off the UI thread. One check runs at a time.
func (work *Workspace) checkUnresolved() {
	if work.unresolvedJob == 0 || work.unresolvedPlan == nil || work.resolving {
		return
	}
	work.resolving, work.lastResolve = true, time.Now()
	plan, tools, sequence := work.unresolvedPlan, work.renderTools(), work.unresolvedJob
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), unresolvedCheckWindow)
		defer cancel()
		_, ended, err := plan.Resolve(ctx, tools, sequence)
		work.resolutions <- resolution{sequence: sequence, ended: ended, err: err}
		work.host.RequestRedraw()
	}()
}

// recheckUnresolved repeats the check on the named interval while the
// refusal stands. The window's tick calls it.
func (work *Workspace) recheckUnresolved() {
	if work.unresolvedJob != 0 && !work.resolving && time.Since(work.lastResolve) >= unresolvedRecheckInterval {
		work.checkUnresolved()
	}
}

// drainResolution applies a finished recheck: the refusal lifts when the queue
// says the job ended, and otherwise the status says what is true and what to do.
func (work *Workspace) drainResolution() {
	select {
	case answer := <-work.resolutions:
		work.resolving = false
		if answer.sequence != work.unresolvedJob {
			return
		}
		switch {
		case answer.err != nil:
			work.status = fmt.Sprintf("Azazel job %d could not be read just now and may still be running: %v. Press Preview to check again, or ask the queue owner to cancel it", answer.sequence, answer.err)
		case answer.ended:
			work.unresolvedJob, work.unresolvedPlan = 0, nil
			work.status = fmt.Sprintf("Azazel job %d has ended; you can render again", answer.sequence)
		default:
			work.status = fmt.Sprintf("Azazel job %d is still running under that sequence number; its cancellation was asked for again. Press Preview to check again, or ask the queue owner to cancel it", answer.sequence)
		}
		work.invalidate()
	default:
	}
}
