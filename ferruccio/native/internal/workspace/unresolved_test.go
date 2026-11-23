package workspace

import (
	"errors"
	"strings"
	"testing"

	"github.com/janearc/slayer/session/placement"
)

// TestRefusalLiftsWhenTheQueueSaysTheJobEnded checks that an ended answer
// lifts the refusal and says so, and that submitting is allowed again.
func TestRefusalLiftsWhenTheQueueSaysTheJobEnded(t *testing.T) {
	work := configuredWorkspace(t)
	work.unresolvedJob = 41
	work.resolving = true
	work.resolutions <- resolution{sequence: 41, ended: true}
	work.drainResolution()
	if work.unresolvedJob != 0 || work.resolving || !strings.Contains(work.status, "job 41 has ended") {
		t.Fatalf("job=%d resolving=%t status=%q; want the refusal lifted and said", work.unresolvedJob, work.resolving, work.status)
	}
	if err := work.refuseWhileUnresolved(); err != nil {
		t.Fatalf("a lifted refusal still refuses: %v", err)
	}
}

// TestRefusalStandsWhileTheJobIsLiveOrUnreadable is the anti-test: a live job
// and an unreadable reply each keep the refusal and say in words what is true.
func TestRefusalStandsWhileTheJobIsLiveOrUnreadable(t *testing.T) {
	for name, answer := range map[string]resolution{
		"live":       {sequence: 41},
		"unreadable": {sequence: 41, err: errors.New("status unavailable")},
	} {
		t.Run(name, func(t *testing.T) {
			work := configuredWorkspace(t)
			work.unresolvedJob = 41
			work.resolving = true
			work.resolutions <- answer
			work.drainResolution()
			if work.unresolvedJob != 41 || work.resolving || !strings.Contains(work.status, "41") || !strings.Contains(work.status, "Press Preview") {
				t.Fatalf("job=%d status=%q; want the refusal kept with what to do", work.unresolvedJob, work.status)
			}
		})
	}
}

// TestNothingIsSubmittedWhileAnEarlierJobIsUnresolved checks that a render
// request is refused in words and starts nothing.
func TestNothingIsSubmittedWhileAnEarlierJobIsUnresolved(t *testing.T) {
	work := configuredWorkspace(t)
	work.unresolvedJob = 41
	err := work.StartPreview()
	if err == nil || !strings.Contains(err.Error(), "job 41") || !strings.Contains(err.Error(), "no second job") {
		t.Fatalf("StartPreview = %v; want a refusal naming job 41", err)
	}
	if work.cancel != nil {
		t.Fatal("a refused request started work")
	}
}

// TestStaleRecheckAnswerIsIgnored is the anti-test for a late answer: one
// about another sequence changes nothing.
func TestStaleRecheckAnswerIsIgnored(t *testing.T) {
	work := configuredWorkspace(t)
	work.unresolvedJob = 42
	work.resolving = true
	work.resolutions <- resolution{sequence: 41, ended: true}
	work.drainResolution()
	if work.unresolvedJob != 42 {
		t.Fatal("an answer about an older job lifted the refusal for a newer one")
	}
}

// TestWorkingDragIsRefusedAtItsStartDuringAFullRender checks that the drag
// does not begin, and says why, so no work is done and then lost.
func TestWorkingDragIsRefusedAtItsStartDuringAFullRender(t *testing.T) {
	work := configuredWorkspace(t)
	work.working.scene = &placement.DisplayScene{}
	work.working.source, work.working.path = work.state.document.Text(), work.state.path
	work.working.displayed = true
	work.fullRunning = true
	work.beginWorking()
	if work.working.gesture != nil || !strings.Contains(work.status, "full render is running") {
		t.Fatalf("gesture=%v status=%q; want the drag refused", work.working.gesture, work.status)
	}
}

// TestWorkingDragBeginsWhenNoFullRenderRuns is the anti-test: without a full
// render the drag begins.
func TestWorkingDragBeginsWhenNoFullRenderRuns(t *testing.T) {
	work := configuredWorkspace(t)
	work.working.scene = &placement.DisplayScene{}
	work.working.source, work.working.path = work.state.document.Text(), work.state.path
	work.working.displayed = true
	work.beginWorking()
	if work.working.gesture == nil {
		t.Fatal("a drag with no full render running did not begin")
	}
}
