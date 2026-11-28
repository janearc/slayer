package workspace

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gogpu/gogpu"
	"github.com/gogpu/ui/app"
	"github.com/janearc/slayer/ferruccio/internal/preview"
)

// TestAcceptedSetPreviewProtectsItsCamera checks the viewport join.
func TestAcceptedSetPreviewProtectsItsCamera(t *testing.T) {
	path, err := filepath.Abs("../../../examples/oakland-larson/set.yaml")
	if err != nil {
		t.Fatal(err)
	}
	work, err := New(gogpu.NewApp(gogpu.DefaultConfig()), app.New(), Options{File: path})
	if err != nil {
		t.Fatal(err)
	}
	before := work.state.document.Text()
	work.editor.ApplyText(before + "# unsaved camera edit\n")
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
	accepted := work.state.document.Text()
	if err := work.Navigate(10, 0, 0); err == nil || !strings.Contains(err.Error(), "working camera") {
		t.Fatalf("accepted photograph did not protect its camera: %v", err)
	}
	if work.editor.Text() != accepted || work.state.document.Text() != accepted {
		t.Fatal("navigating a photograph edited the source")
	}
	work.undoCamera(work.ui.Window().Context(), false)
	if work.state.document.Text() != before {
		t.Fatal("photograph disabled source undo")
	}
}
