package workspace

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gogpu/gogpu"
	"github.com/gogpu/ui/app"
)

// TestPlayDoesNotRenderOnStartup checks both interactive and proof startup.
func TestPlayDoesNotRenderOnStartup(t *testing.T) {
	for _, suppress := range []bool{false, true} {
		work, err := New(gogpu.NewApp(gogpu.DefaultConfig()), app.New(), Options{
			Examples: "../../../examples", Composition: "play", SuppressAutomaticPreview: suppress,
		})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { work.clearNavigation(); os.RemoveAll(filepath.Dir(work.state.path)) })
		if work.autoPreview || work.Running() {
			t.Fatal("startup scheduled a preview")
		}
		work.Tick()
		if work.autoPreview || work.Running() {
			t.Fatal("startup tick started a preview")
		}
	}
}

func TestSettledPlayEditRefreshesWorkingViewWithoutRendering(t *testing.T) {
	work := playWorkspace(t)
	work.working.dirty = false
	work.pending.remember(work.state, work.specimen, time.Now().Add(-time.Second))
	work.advanceNavigation()
	if work.Running() || work.pending.document != nil || !work.working.dirty {
		t.Fatal("settled play edit started a render or failed to refresh the working view")
	}
}
