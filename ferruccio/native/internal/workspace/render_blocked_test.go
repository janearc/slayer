package workspace

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gogpu/gogpu"
	"github.com/gogpu/ui/app"
	"github.com/janearc/slayer/ferruccio/internal/preview"
)

// configuredWorkspace returns a workspace with a valid draft and every tool
// the queue transport needs named, so only the draft can block Render preview.
func configuredWorkspace(t *testing.T) *Workspace {
	t.Helper()
	work, err := New(gogpu.NewApp(gogpu.DefaultConfig()), app.New(), Options{
		Examples: "../../../examples", Composition: "observe",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { work.releaseExample(); os.RemoveAll(filepath.Dir(work.state.path)) })
	work.options.Tools = preview.Tools{Azazel: "/tools/azazel", Queue: "/queue", Slayer: "/tools/slayer"}
	work.options.Evidence = "/evidence"
	work.validate()
	return work
}

// TestRenderButtonStateIsComputedOnChangeNotOnPaint checks that building the
// layout does not recheck the source, and that checking the source does.
func TestRenderButtonStateIsComputedOnChangeNotOnPaint(t *testing.T) {
	work := configuredWorkspace(t)
	if work.renderBlocked {
		t.Fatal("a valid, configured draft is blocked")
	}
	work.state.document.Replace("scene: [unclosed")
	work.layout()
	if work.renderBlocked {
		t.Fatal("building the layout rechecked the source")
	}
	work.validate()
	if !work.renderBlocked {
		t.Fatal("an invalid draft left Render preview enabled after it was checked")
	}
}

// TestRenderButtonStateClearsForAValidDraft is the anti-test: once the draft
// is valid again and checked, the block is lifted.
func TestRenderButtonStateClearsForAValidDraft(t *testing.T) {
	work := configuredWorkspace(t)
	valid := work.state.document.Text()
	work.state.document.Replace("scene: [unclosed")
	work.validate()
	work.state.document.Replace(valid)
	work.validate()
	if work.renderBlocked {
		t.Fatal("a valid draft stayed blocked after it was checked")
	}
}
