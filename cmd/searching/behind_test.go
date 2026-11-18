package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/beats"
	"github.com/janearc/slayer/improv/searching"
)

// TestBehindUsesIrregularGrid makes a fixed-tempo or excerpt-relative
// metronome disagree with the actual picture's song clock.
func TestBehindUsesIrregularGrid(t *testing.T) {
	edit := &searching.Edit{Grid: &beats.Grid{Score: beats.Score{Beats: beats.Pulse{
		Times: []float64{120, 120.5, 121.2, 122, 123.01, 124}, Measured: []int{1, 1, 0, 1, 1, 0}}}},
		Plan: searching.Plan{FlowStudy: &searching.FlowPlan{}}, FlowFold: 123.01, FlowAnswer: 124}
	before := explainFrame(edit, 123)
	after := explainFrame(edit, 123.02)
	if before.Bar != 1 || before.Beat != 4 || after.Bar != 2 || after.Beat != 1 || !after.Onset {
		t.Fatalf("grid shifted: before %+v, after %+v", before, after)
	}
	if strings.Contains(before.Says, "inner strands") || !strings.Contains(after.Says, "inner strands") {
		t.Fatal("explanation did not follow the actual fold cue")
	}
	if explainFrame(edit, 121.5).Onset {
		t.Fatal("count-through presented as detected onset")
	}
	if !strings.Contains(explainFrame(edit, 124).Says, "back toward") {
		t.Fatal("answer cue lost")
	}
}

// TestBehindRefusesChangedRecipe proves a companion cannot silently attach
// current authoring to a movie made with another recipe, before decoding it.
func TestBehindRefusesChangedRecipe(t *testing.T) {
	into := t.TempDir()
	os.WriteFile(filepath.Join(into, "render.json"), []byte(`{"frames":1,"fps":6,"plan_sha256":"wrong","score_sha256":"wrong"}`), 0600)
	os.WriteFile(filepath.Join(into, "shoot.yaml"), []byte("changed"), 0600)
	os.WriteFile(filepath.Join(into, "score.yaml"), []byte("changed"), 0600)
	if err := publishBehind(into, false); err == nil || !strings.Contains(err.Error(), "changed since rendering") {
		t.Fatalf("changed recipe: %v", err)
	}
	if _, err := os.Stat(filepath.Join(into, "behind-the-scenes.html")); !os.IsNotExist(err) {
		t.Fatal("published a mismatched companion")
	}
}
