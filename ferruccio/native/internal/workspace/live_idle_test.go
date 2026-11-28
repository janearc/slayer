package workspace

import (
	"strings"
	"testing"

	"github.com/janearc/slayer/ferruccio/internal/draft"
)

func TestUnchangedWorkingViewTickDoesNotCopySource(t *testing.T) {
	source := "# " + strings.Repeat("x", 1<<20) + "\n" + checkingSource
	document, err := draft.Open("scene.yaml", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	work := playWorkspace(t)
	work.state.document = document
	work.working.source, work.working.path = source, work.state.path
	if got := testing.AllocsPerRun(20, work.updateWorking); got != 0 {
		t.Fatalf("unchanged working view tick allocated %g times", got)
	}
}
