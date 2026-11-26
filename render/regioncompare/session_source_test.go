package regioncompare

import (
	"context"
	"testing"

	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/render/regionset"
)

func TestSessionSourceRefusesAnUnassertedSnapshot(t *testing.T) {
	source := &SessionSource{Context: context.Background()}
	if _, err := source.Draw("job.yaml", regionset.Region{Name: "water", Width: 1, Height: 1}, Identity{1}); err == nil {
		t.Fatal("a source with no snapshot evidence was accepted")
	}
}

func TestFailedDrawForgetsAnEarlierTone(t *testing.T) {
	source := &SessionSource{tone: map[string]colour.Tone{"water": {Exposure: 1}}}
	_, _ = source.Draw("job.yaml", regionset.Region{Name: "water", Width: 1, Height: 1}, Identity{1})
	if _, exists := source.tone["water"]; exists {
		t.Fatal("a failed draw retained the previous tone")
	}
}

func TestSessionSourceRefusesAnUnpinnedReference(t *testing.T) {
	source := &SessionSource{Context: context.Background()}
	if _, err := source.Reference("job.yaml", regionset.Region{Name: "water", Width: 1, Height: 1}, Identity{1}); err == nil {
		t.Fatal("a source with no pinned reference was accepted")
	}
}
