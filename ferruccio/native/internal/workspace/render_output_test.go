package workspace

import (
	"errors"
	"image/color"
	"path/filepath"
	"slices"
	"testing"

	"github.com/janearc/slayer/ferruccio/internal/preview"
)

func TestRenderFinderControlTracksAcceptedImage(t *testing.T) {
	work := libraryEventWorkspace(t)
	work.options.Starter = t.TempDir()
	work.output = filepath.Join(work.options.Starter, "preview.png")
	work.rebuild()
	if libraryWidget(work.root, "Show in Finder") != nil {
		t.Fatal("the bundled starter was offered as a render")
	}
	work.options.Evidence = t.TempDir()
	plan, err := preview.Prepare(preview.Request{
		Path: work.state.path, Text: work.state.document.Text(), OutputRoot: work.options.Evidence,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { plan.Close() })
	work.results <- renderResult{plan: plan, generation: work.generation, err: errors.New("render failed")}
	work.Tick()
	if libraryWidget(work.root, "Show in Finder") != nil {
		t.Fatal("a failed render replaced the starter with output")
	}
	writeProofPNG(t, plan.Output, color.RGBA{R: 47, G: 93, B: 121, A: 255})
	work.results <- renderResult{plan: plan, generation: work.generation}
	work.Tick()
	if libraryWidget(work.root, "Show in Finder") == nil {
		t.Fatalf("accepted image offered no Finder action: %s", work.diagnostic)
	}
	work.options.Evidence = t.TempDir()
	command := finderRevealCommand(work.renderOutputPath())
	if want := []string{"/usr/bin/open", "-R", plan.Output}; !slices.Equal(want, command.Args) {
		t.Fatalf("Finder arguments = %q, want %q", command.Args, want)
	}
	work.openPath(filepath.Join(work.examplesFolder(), "ferruccio-playground.yaml"))
	if libraryWidget(work.root, "Show in Finder") != nil {
		t.Fatal("a different document still offered the previous image")
	}
}
