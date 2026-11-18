package searching

import (
	"testing"

	"github.com/janearc/slayer/beats"
)

// TestCutBoundaries checks exact handover, rather than sampling both scenes
// at a cut or leaving a missing frame between them.
func TestCutBoundaries(t *testing.T) {
	edit := &Edit{ShotSpans: [][2]float64{{10, 20}, {20, 30}}, Plan: Plan{Shots: []ShotPlan{{Scene: "orbital"}, {Scene: "swarm"}}}}
	for _, example := range []struct {
		at   float64
		want int
	}{{9, -1}, {10, 0}, {19.999, 0}, {20, 1}, {30, -1}, {12, 0}} {
		if got := edit.shot(example.at); got != example.want {
			t.Fatalf("at %g: %d", example.at, got)
		}
	}
	if edit.ShotAt(20).Scene != "swarm" {
		t.Fatal("companion chose a different shot")
	}
}

// TestCutsRefuseOverlap rejects an edit that hides one shot behind another.
func TestCutsRefuseOverlap(t *testing.T) {
	times := make([]float64, 32)
	for index := range times {
		times[index] = float64(index)
	}
	edit := &Edit{Grid: &beats.Grid{Score: beats.Score{Duration: 32, Beats: beats.Pulse{Times: times}}},
		Plan: Plan{FullCut: true, Shots: []ShotPlan{
			{From: "1.1", Until: "3.1", Scene: "slate", Caption: "first"},
			{From: "2.1", Until: "4.1", Scene: "slate", Caption: "overlaps"}}}}
	if edit.checkShots() == nil {
		t.Fatal("overlap accepted")
	}
	edit.Plan.Shots[1].From = "3.1"
	if err := edit.checkShots(); err != nil {
		t.Fatal(err)
	}
	if len(edit.ShotSpans) != 2 {
		t.Fatal("a second validation accumulated cuts")
	}
}
