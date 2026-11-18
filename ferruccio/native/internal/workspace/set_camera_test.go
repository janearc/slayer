package workspace

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/gogpu/gogpu"
	"github.com/gogpu/ui/app"
	"github.com/janearc/slayer/ferruccio/internal/preview"
)

// TestAcceptedSetPreviewEnablesItsAuthoredCamera checks the viewport join.
func TestAcceptedSetPreviewEnablesItsAuthoredCamera(t *testing.T) {
	path, err := filepath.Abs("../../../examples/oakland-larson/set.yaml")
	if err != nil {
		t.Fatal(err)
	}
	work, err := New(gogpu.NewApp(gogpu.DefaultConfig()), app.New(), Options{File: path})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := preview.Prepare(preview.Request{Path: path, Text: work.state.document.Text()})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { plan.Close(); os.RemoveAll(plan.Directory) })
	file, err := os.Create(plan.Output)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(file, image.NewRGBA(image.Rect(0, 0, 640, 360))); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	work.finishPreview(renderResult{generation: work.generation, plan: plan})
	if !work.viewer.navigable {
		t.Fatal("accepted set render left its camera controls disabled")
	}
	before := work.state.document.Text()
	if err := work.Navigate(10, 0, 0); err != nil {
		t.Fatal(err)
	}
	work.clearNavigation()
	if work.editor.Text() == before || work.editor.Text() != work.state.document.Text() {
		t.Fatal("set camera drag did not reach the editable source")
	}
}
