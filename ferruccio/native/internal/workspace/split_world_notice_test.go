package workspace

import (
	"strings"
	"testing"

	"github.com/janearc/slayer/session/placement"
)

func TestWorkingNoticeReportsOnlyDisplayedWorldPrecisionFallback(t *testing.T) {
	const base = "Sky is not shown"
	work := Workspace{working: workingState{
		scene:              &placement.DisplayScene{SplitWorldVertices: true},
		notice:             base,
		displayed:          true,
		splitWorldFallback: true,
	}}
	if got := work.workingNotice(); !strings.Contains(got, splitWorldFallbackNotice) || !strings.HasPrefix(got, base) {
		t.Fatalf("working details lost a notice: %q", got)
	}
	work.working.displayed = false
	if got := work.workingNotice(); got != base {
		t.Fatalf("stale world precision fallback remained visible: %q", got)
	}
	work.working.displayed = true
	work.working.scene = &placement.DisplayScene{}
	if got := work.workingNotice(); got != base {
		t.Fatalf("fallback from a previous world remained visible: %q", got)
	}
}

func TestSceneDetailRetryKeepsDisplayedWorldPrecisionNotice(t *testing.T) {
	work := libraryEventWorkspace(t)
	live := &work.working
	live.scene = &placement.DisplayScene{SplitWorldVertices: true}
	live.path, live.source = work.state.path, work.state.document.Text()
	live.displayed, live.splitWorldFallback = true, true
	live.detailRetry = &failedDetail{id: "scene.set.things/visitor", message: "Detail staging failed"}
	live.stager = newStageLoader(nil)
	defer work.closeStaged()
	work.showWorkingDetails()

	first, next := detailsPage(t, work, "Scene detail unavailable · 1/2", "Next")
	detailsPage(t, work, "Scene detail unavailable · 1/2", "Retry")
	clickLibraryPoint(work, next)
	second, _ := detailsPage(t, work, "Scene detail unavailable · 2/2", "Retry")
	if !strings.Contains(first, live.detailRetry.message) || !strings.Contains(second, splitWorldFallbackNotice) {
		t.Fatalf("retry dialog lost its failure or precision notice: %q / %q", first, second)
	}
}
